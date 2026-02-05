package message

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/marstid/goznp/pkg/state"
	"github.com/marstid/goznp/pkg/znp"
)

// Listener continuously listens for incoming Zigbee messages.
// It parses messages and updates the state manager accordingly.
type Listener struct {
	adapter  ZNPClient
	manager  *state.Manager
	eventBus EventBus
	parser   *Parser
	logger   *slog.Logger
	timeout  time.Duration
}

// ZNPClient interface for ZNP operations.
// Extracted to allow mocking in tests.
type ZNPClient interface {
	WaitForIncomingMsg(ctx context.Context, srcAddr uint16, clusterID uint16, timeout time.Duration) (*znp.IncomingMessage, error)
}

// EventBus interface for event publishing.
type EventBus interface {
	Publish(ctx context.Context, event interface{}) error
}

// NewListener creates a new message listener.
func NewListener(adapter ZNPClient, manager *state.Manager, eventBus EventBus, timeout time.Duration, logger *slog.Logger) *Listener {
	return &Listener{
		adapter:  adapter,
		manager:  manager,
		eventBus: eventBus,
		parser:   NewParser(),
		logger:   logger,
		timeout:  timeout,
	}
}

// Start begins listening for incoming messages.
// This blocks until the context is cancelled.
func (l *Listener) Start(ctx context.Context) error {
	l.logger.Info("Starting message listener")
	defer l.logger.Info("Message listener stopped")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Wait for any incoming message (no filters)
			msg, err := l.adapter.WaitForIncomingMsg(ctx, 0, 0, l.timeout)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				if errors.Is(err, znp.ErrTimeout) {
					continue // No message, try again
				}
				l.logger.Warn("Failed to receive message", "error", err)
				continue
			}

			l.handleMessage(ctx, msg)
		}
	}
}

// handleMessage processes an incoming Zigbee message.
func (l *Listener) handleMessage(ctx context.Context, msg *znp.IncomingMessage) {
	// Find device by network address in state manager
	dev, ok := l.manager.GetDeviceByNwkAddr(msg.SrcAddr)
	if !ok {
		l.logger.Debug("Received message from unknown device", "nwkAddr", msg.SrcAddr, "clusterID", msg.ClusterID)
		// Device might not be interviewed yet, skip but don't error
		return
	}

	// Parse ZCL message
	parsed := l.parser.Parse(msg)

	// Update message source info
	parsed.Source = &DeviceIdentifier{
		IEEEAddr: dev.IEEEAddr,
		NwkAddr:  dev.NwkAddr,
	}

	// Handle based on command type
	switch parsed.CommandID {
	case uint8(parsed.CommandID): // ReportAttributes
		l.handleReportAttributes(ctx, parsed, dev, msg)
	default:
		l.logger.Debug("Unhandled ZCL command", "commandID", parsed.CommandID, "nwkAddr", msg.SrcAddr, "clusterID", msg.ClusterID)
	}
}

// handleReportAttributes processes attribute reports from a device.
func (l *Listener) handleReportAttributes(ctx context.Context, parsed *ParsedMessage, dev *state.DeviceState, msg *znp.IncomingMessage) {
	if parsed.Error != nil {
		l.logger.Warn("Failed to parse report attributes",
			"ieeeAddr", state.FormatIEEEAddr(dev.IEEEAddr),
			"nwkAddr", msg.SrcAddr,
			"error", parsed.Error,
			"clusterID", msg.ClusterID)
		return
	}

	if len(parsed.Attributes) == 0 {
		return
	}

	// Update state in manager
	changed := l.manager.UpdateState(dev.IEEEAddr, parsed.ClusterID, parsed.Attributes)
	if changed {
		l.logger.Debug("Device state changed",
			"ieeeAddr", state.FormatIEEEAddr(dev.IEEEAddr),
			"slug", dev.Slug(),
			"clusterID", parsed.ClusterID,
			"attributes", len(parsed.Attributes))
	}

	// Update device last seen
	dev.LastSeen = time.Now()

	// Publish event for state change if this is a report attributes command
	if changed && l.eventBus != nil {
		// Note: A more sophisticated implementation would publish individual attribute changes
	}
}

// handleReadResponse processes attribute read responses.
func (l *Listener) handleReadResponse(ctx context.Context, parsed *ParsedMessage, dev *state.DeviceState, msg *znp.IncomingMessage) {
	if parsed.Error != nil {
		l.logger.Warn("Failed to parse read response",
			"ieeeAddr", state.FormatIEEEAddr(dev.IEEEAddr),
			"nwkAddr", msg.SrcAddr,
			"error", parsed.Error)
		return
	}

	if len(parsed.Attributes) == 0 {
		return
	}

	// Update state in manager
	l.manager.UpdateState(dev.IEEEAddr, parsed.ClusterID, parsed.Attributes)

	// Update device last seen
	dev.LastSeen = time.Now()
}

// handleWriteResponse processes attribute write responses.
func (l *Listener) handleWriteResponse(ctx context.Context, parsed *ParsedMessage, dev *state.DeviceState, msg *znp.IncomingMessage) {
	if parsed.Error != nil {
		l.logger.Warn("Failed to parse write response",
			"ieeeAddr", state.FormatIEEEAddr(dev.IEEEAddr),
			"nwkAddr", msg.SrcAddr,
			"error", parsed.Error)
		return
	}

	// Write responses typically don't have values to update state
	// Just log if needed
	l.logger.Debug("Write response received",
		"ieeeAddr", state.FormatIEEEAddr(dev.IEEEAddr),
		"nwkAddr", msg.SrcAddr,
		"clusterID", msg.ClusterID)
}

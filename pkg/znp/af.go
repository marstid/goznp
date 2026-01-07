package znp

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// MaxClusters is the maximum number of clusters allowed per endpoint.
// This limit prevents buffer overflows when serializing cluster lists.
const MaxClusters = 32

// EndpointConfig defines endpoint registration parameters.
type EndpointConfig struct {
	Endpoint    uint8
	AppProfID   uint16 // Profile ID (0x0104 = Home Automation).
	AppDeviceID uint16 // Device ID.
	AppDevVer   uint8  // Device version.
	LatencyReq  uint8  // Latency requirement (0=none).
	InClusters  []uint16
	OutClusters []uint16
}

// Common profile IDs - defined in commands.go as ApplicationProfile type.
// Use: ProfileHomeAutomation, ProfileSmartEnergy, ProfileGreenPower, ProfileLightLink.

// AfDelete unregisters/deletes an endpoint.
// Returns status (0=success, 0x09=not found).
func (z *ZNP) AfDelete(ctx context.Context, endpoint uint8) (uint8, error) {
	buf := NewBuffaloWriter()
	buf.WriteUint8(endpoint)

	resp, err := z.Request(ctx, unpi.AF, CmdAfDelete, buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("af delete: %w", err)
	}

	if len(resp.Data) < 1 {
		return 0, fmt.Errorf("af delete: response too short")
	}
	return resp.Data[0], nil
}

// AfRegister registers an endpoint on the coordinator.
// Returns status (0=success).
func (z *ZNP) AfRegister(ctx context.Context, config EndpointConfig) (uint8, error) {
	// Validate cluster list lengths to prevent buffer overflow.
	if len(config.InClusters) > MaxClusters {
		return 0, fmt.Errorf("af register: too many input clusters: %d (max %d)",
			len(config.InClusters), MaxClusters)
	}
	if len(config.OutClusters) > MaxClusters {
		return 0, fmt.Errorf("af register: too many output clusters: %d (max %d)",
			len(config.OutClusters), MaxClusters)
	}

	// Build request manually due to variable length cluster lists.
	buf := NewBuffaloWriter()
	buf.WriteUint8(config.Endpoint)
	buf.WriteUint16(config.AppProfID)
	buf.WriteUint16(config.AppDeviceID)
	buf.WriteUint8(config.AppDevVer)
	buf.WriteUint8(config.LatencyReq)
	buf.WriteUint8(uint8(len(config.InClusters)))
	for _, c := range config.InClusters {
		buf.WriteUint16(c)
	}
	buf.WriteUint8(uint8(len(config.OutClusters)))
	for _, c := range config.OutClusters {
		buf.WriteUint16(c)
	}

	resp, err := z.Request(ctx, unpi.AF, CmdAfRegister, buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("af register: %w", err)
	}

	if len(resp.Data) < 1 {
		return 0, fmt.Errorf("af register: response too short")
	}
	return resp.Data[0], nil
}

// DataRequestOptions defines AF data request options.
type DataRequestOptions uint8

const (
	// AfOptionNone is no special options.
	AfOptionNone DataRequestOptions = 0x00
	// AfOptionAckRequest requests acknowledgment.
	AfOptionAckRequest DataRequestOptions = 0x10
	// AfOptionDiscoverRoute enables route discovery.
	AfOptionDiscoverRoute DataRequestOptions = 0x20
	// AfOptionEnableSecurity enables APS security.
	AfOptionEnableSecurity DataRequestOptions = 0x40
	// AfOptionSkipRouting skips routing (direct).
	AfOptionSkipRouting DataRequestOptions = 0x80
)

// DataRequest contains parameters for an AF data request.
type DataRequest struct {
	DstAddr     uint16
	DstEndpoint uint8
	SrcEndpoint uint8
	ClusterID   uint16
	TransID     uint8
	Options     DataRequestOptions
	Radius      uint8 // 0 = use default (30).
	Data        []byte
}

// AfDataRequest sends data to a device endpoint.
// Returns status (0=success).
func (z *ZNP) AfDataRequest(ctx context.Context, req DataRequest) (uint8, error) {
	// Default radius to 30 if 0.
	radius := req.Radius
	if radius == 0 {
		radius = 30
	}

	// Build request.
	buf := NewBuffaloWriter()
	buf.WriteUint16(req.DstAddr)
	buf.WriteUint8(req.DstEndpoint)
	buf.WriteUint8(req.SrcEndpoint)
	buf.WriteUint16(req.ClusterID)
	buf.WriteUint8(req.TransID)
	buf.WriteUint8(uint8(req.Options))
	buf.WriteUint8(radius)
	buf.WriteUint8(uint8(len(req.Data)))
	buf.WriteBytes(req.Data)

	resp, err := z.Request(ctx, unpi.AF, CmdAfDataRequest, buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("af data request: %w", err)
	}

	if len(resp.Data) < 1 {
		return 0, fmt.Errorf("af data request: response too short")
	}
	return resp.Data[0], nil
}

// DataConfirm contains the async confirmation of a data request.
type DataConfirm struct {
	Status   uint8
	Endpoint uint8
	TransID  uint8
}

// WaitForDataConfirm waits for a data confirmation matching the transaction ID.
func (z *ZNP) WaitForDataConfirm(ctx context.Context, transID uint8, timeout time.Duration) (*DataConfirm, error) {
	// Create matcher for data confirm AREQ.
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.AF,
		CommandID: CmdAfDataConfirm.ID,
	}

	// Wait for confirmation with timeout.
	frame, err := z.waiter.WaitFor(ctx, matcher, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait for data confirm: %w", err)
	}

	// Parse response.
	buf := NewBuffalo(frame.Data)

	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	endpoint, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read endpoint: %w", err)
	}

	receivedTransID, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read transId: %w", err)
	}

	// Check if this is the transaction we're waiting for.
	if receivedTransID != transID {
		return nil, fmt.Errorf("received wrong transaction ID: expected %d, got %d", transID, receivedTransID)
	}

	return &DataConfirm{
		Status:   status,
		Endpoint: endpoint,
		TransID:  receivedTransID,
	}, nil
}

// IncomingMessage represents a received AF message.
type IncomingMessage struct {
	GroupID      uint16
	ClusterID    uint16
	SrcAddr      uint16
	SrcEndpoint  uint8
	DstEndpoint  uint8
	WasBroadcast bool
	LinkQuality  uint8
	SecurityUse  bool
	Timestamp    uint32
	TransSeqNum  uint8
	Data         []byte
}

// WaitForIncomingMsg waits for an incoming message.
// If srcAddr is non-zero, only matches messages from that address.
// If clusterID is non-zero, only matches messages from that cluster.
func (z *ZNP) WaitForIncomingMsg(ctx context.Context, srcAddr uint16, clusterID uint16, timeout time.Duration) (*IncomingMessage, error) {
	// Create matcher for incoming message AREQ.
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.AF,
		CommandID: CmdAfIncomingMsg.ID,
	}

	// Track deadline for proper timeout handling.
	deadline := time.Now().Add(timeout)

	// Keep trying until we get a matching message or timeout.
	for {
		// Calculate remaining time to prevent waiter exhaustion.
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("wait for incoming message: %w", ErrTimeout)
		}

		// Use shorter wait duration or remaining time, whichever is smaller.
		waitDuration := 100 * time.Millisecond
		if remaining < waitDuration {
			waitDuration = remaining
		}

		// Wait for any incoming message.
		waitCtx, cancel := context.WithTimeout(ctx, waitDuration)
		frame, err := z.waiter.WaitFor(waitCtx, matcher, waitDuration)
		cancel()

		if err != nil {
			// Check if context was canceled.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// Timeout on this wait, check if overall deadline exceeded.
				if time.Now().After(deadline) {
					return nil, fmt.Errorf("wait for incoming message: %w", ErrTimeout)
				}
				// No match yet, try again.
				continue
			}
		}

		// Parse the message.
		msg, err := parseIncomingMsg(frame.Data)
		if err != nil {
			// Invalid message, skip and continue waiting
			continue
		}

		// Check filters
		if srcAddr != 0 && msg.SrcAddr != srcAddr {
			// Wrong source address, continue waiting
			continue
		}
		if clusterID != 0 && msg.ClusterID != clusterID {
			// Wrong cluster, continue waiting
			continue
		}

		// Message matches filters
		return msg, nil
	}
}

// parseIncomingMsg parses an incoming message frame.
func parseIncomingMsg(data []byte) (*IncomingMessage, error) {
	buf := NewBuffalo(data)

	groupID, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read groupID: %w", err)
	}

	clusterID, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read clusterID: %w", err)
	}

	srcAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read srcAddr: %w", err)
	}

	srcEndpoint, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read srcEndpoint: %w", err)
	}

	dstEndpoint, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read dstEndpoint: %w", err)
	}

	wasBroadcast, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read wasBroadcast: %w", err)
	}

	linkQuality, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read linkQuality: %w", err)
	}

	securityUse, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read securityUse: %w", err)
	}

	timestamp, err := buf.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}

	transSeqNum, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read transSeqNum: %w", err)
	}

	// Read data length
	dataLen, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read data length: %w", err)
	}

	// Read data payload
	msgData, err := buf.ReadBytes(int(dataLen))
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return &IncomingMessage{
		GroupID:      groupID,
		ClusterID:    clusterID,
		SrcAddr:      srcAddr,
		SrcEndpoint:  srcEndpoint,
		DstEndpoint:  dstEndpoint,
		WasBroadcast: wasBroadcast != 0,
		LinkQuality:  linkQuality,
		SecurityUse:  securityUse != 0,
		Timestamp:    timestamp,
		TransSeqNum:  transSeqNum,
		Data:         msgData,
	}, nil
}

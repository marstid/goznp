package znp

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// ExtNetworkInfo contains extended network information.
type ExtNetworkInfo struct {
	ShortAddr     uint16
	DevState      uint8
	PanID         uint16
	ParentAddr    uint16
	ExtendedPanID [8]byte
	ParentExtAddr [8]byte
	Channel       uint8
}

// ExtNwkInfo retrieves extended network information.
func (z *ZNP) ExtNwkInfo(ctx context.Context) (*ExtNetworkInfo, error) {
	// Send CmdZdoExtNwkInfo SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoExtNwkInfo, nil)
	if err != nil {
		return nil, fmt.Errorf("ext nwk info request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)

	// Read short address (uint16)
	shortAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read short address: %w", err)
	}

	// Read device state (uint8)
	devState, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read device state: %w", err)
	}

	// Read PAN ID (uint16)
	panId, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read PAN ID: %w", err)
	}

	// Read parent address (uint16)
	parentAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read parent address: %w", err)
	}

	// Read extended PAN ID (8 bytes)
	extendedPanId, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read extended PAN ID: %w", err)
	}

	// Read parent extended address (8 bytes)
	parentExtAddr, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read parent extended address: %w", err)
	}

	// Read channel (uint8)
	channel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read channel: %w", err)
	}

	return &ExtNetworkInfo{
		ShortAddr:     shortAddr,
		DevState:      devState,
		PanID:         panId,
		ParentAddr:    parentAddr,
		ExtendedPanID: extendedPanId,
		ParentExtAddr: parentExtAddr,
		Channel:       channel,
	}, nil
}

// StartupFromApp starts the ZigBee stack from application.
// The startDelay parameter specifies milliseconds to wait before starting (0 for immediate).
func (z *ZNP) StartupFromApp(ctx context.Context, startDelay uint16) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(startDelay)

	// Send CmdZdoStartupFromApp SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoStartupFromApp, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("startup from app request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	return status, nil
}

// MgmtNwkUpdateReq sends a network update request for channel change.
// dstAddr is the destination address (0xFFFF for broadcast).
// dstAddrMode is the address mode (0x0F for broadcast).
// channelMask is the target channel mask (bit N = channel N, e.g., 0x00008000 for channel 15).
// scanDuration is 0xFE for channel change, 0x00-0x05 for energy scan.
// scanCount is number of scans (ignored for channel change).
// nwkManagerAddr is the network manager address (usually 0x0000).
func (z *ZNP) MgmtNwkUpdateReq(ctx context.Context, dstAddr uint16, dstAddrMode uint8, channelMask uint32, scanDuration uint8, scanCount uint8, nwkManagerAddr uint16) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)
	writer.WriteUint8(dstAddrMode)
	writer.WriteUint32(channelMask)
	writer.WriteUint8(scanDuration)
	writer.WriteUint8(scanCount)
	writer.WriteUint16(nwkManagerAddr)

	// Send CmdZdoMgmtNwkUpdateReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoMgmtNwkUpdateReq, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("mgmt nwk update request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	return status, nil
}

// WaitForStateChange waits for a device state change indication.
// Returns the new device state.
func (z *ZNP) WaitForStateChange(ctx context.Context, timeout time.Duration) (DevState, error) {
	// Create matcher for state change indication (AREQ)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoStateChangeInd.ID,
	}

	// Wait for state change indication
	resp, err := z.waiter.WaitFor(ctx, matcher, timeout)
	if err != nil {
		return 0, fmt.Errorf("waiting for state change failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	state, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read state: %w", err)
	}

	return DevState(state), nil
}

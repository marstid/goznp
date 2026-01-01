package znp

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/unpi"
)

// PingCapabilities contains the adapter capabilities from ping response.
type PingCapabilities struct {
	Capabilities uint16
}

// HasCapability checks if a specific capability bit is set.
func (p *PingCapabilities) HasCapability(bit uint16) bool {
	return p.Capabilities&bit != 0
}

// ZStackVariant identifies the Z-Stack firmware variant.
// Note: Product ID is reported by the firmware, not determined by the filename.
// Koenkk's "Z-Stack_3.x.0" folder contains firmware that reports Product ID 1.
type ZStackVariant uint8

const (
	ZStack12     ZStackVariant = 0 // Z-Stack Home 1.2 (legacy, CC2530/CC2531)
	ZStack3      ZStackVariant = 1 // Z-Stack 3.0 (CC2652/CC1352, modern)
	ZStack3Multi ZStackVariant = 2 // Z-Stack 3 Multi-PAN (rare)
)

// String returns the Z-Stack variant name.
func (v ZStackVariant) String() string {
	switch v {
	case ZStack12:
		return "Z-Stack Home 1.2"
	case ZStack3:
		return "Z-Stack 3.0"
	case ZStack3Multi:
		return "Z-Stack 3 Multi-PAN"
	default:
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

// VersionInfo contains firmware version information.
type VersionInfo struct {
	TransportRev uint8
	Product      uint8 // Maps to ZStackVariant
	MajorRel     uint8
	MinorRel     uint8
	MaintRel     uint8
	Revision     uint32 // Build date as YYYYMMDD
}

// Variant returns the Z-Stack variant based on the product ID.
func (v *VersionInfo) Variant() ZStackVariant {
	return ZStackVariant(v.Product)
}

// String returns a formatted version string.
func (v *VersionInfo) String() string {
	return fmt.Sprintf("%d.%d.%d (rev %d)", v.MajorRel, v.MinorRel, v.MaintRel, v.Revision)
}

// FullString returns a complete version description including the Z-Stack variant.
func (v *VersionInfo) FullString() string {
	return fmt.Sprintf("%s - SDK %d.%d.%d (build %d)", v.Variant(), v.MajorRel, v.MinorRel, v.MaintRel, v.Revision)
}

// ResetIndication contains reset event details.
type ResetIndication struct {
	Reason       uint8
	TransportRev uint8
	Product      uint8
	MajorRel     uint8
	MinorRel     uint8
	MaintRel     uint8
}

// Ping sends a ping request to verify connectivity.
func (z *ZNP) Ping(ctx context.Context) (*PingCapabilities, error) {
	// Send CmdSysPing SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysPing, nil)
	if err != nil {
		return nil, fmt.Errorf("ping request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	capabilities, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ping response: %w", err)
	}

	return &PingCapabilities{
		Capabilities: capabilities,
	}, nil
}

// Version retrieves the firmware version information.
func (z *ZNP) Version(ctx context.Context) (*VersionInfo, error) {
	// Send CmdSysVersion SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysVersion, nil)
	if err != nil {
		return nil, fmt.Errorf("version request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)

	transportRev, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read transportRev: %w", err)
	}

	product, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	majorRel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read majorRel: %w", err)
	}

	minorRel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read minorRel: %w", err)
	}

	maintRel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read maintRel: %w", err)
	}

	revision, err := buf.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("failed to read revision: %w", err)
	}

	return &VersionInfo{
		TransportRev: transportRev,
		Product:      product,
		MajorRel:     majorRel,
		MinorRel:     minorRel,
		MaintRel:     maintRel,
		Revision:     revision,
	}, nil
}

// Reset performs a system reset.
func (z *ZNP) Reset(ctx context.Context, resetType ResetType) (*ResetIndication, error) {
	// Build request data with reset type
	writer := NewBuffaloWriter()
	writer.WriteUint8(uint8(resetType))

	// Create matcher for expected reset indication (AREQ)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.SYS,
		CommandID: CmdSysResetInd.ID,
	}

	// Register waiter before sending the AREQ
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start waiting for reset indication in a separate goroutine
	responseChan := make(chan *unpi.Frame, 1)
	errorChan := make(chan error, 1)

	go func() {
		resp, err := z.waiter.WaitFor(waitCtx, matcher, DefaultResetTimeout)
		if err != nil {
			errorChan <- err
		} else {
			responseChan <- resp
		}
	}()

	// Send CmdSysResetReq AREQ
	if err := z.Send(unpi.SYS, CmdSysResetReq, writer.Bytes()); err != nil {
		cancel()
		return nil, fmt.Errorf("reset request failed: %w", err)
	}

	// Wait for reset indication or error
	var resp *unpi.Frame
	select {
	case resp = <-responseChan:
		// Got the reset indication
	case err := <-errorChan:
		return nil, fmt.Errorf("waiting for reset indication failed: %w", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Parse reset indication using Buffalo
	buf := NewBuffalo(resp.Data)

	reason, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read reason: %w", err)
	}

	transportRev, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read transportRev: %w", err)
	}

	product, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	majorRel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read majorRel: %w", err)
	}

	minorRel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read minorRel: %w", err)
	}

	maintRel, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read maintRel: %w", err)
	}

	return &ResetIndication{
		Reason:       reason,
		TransportRev: transportRev,
		Product:      product,
		MajorRel:     majorRel,
		MinorRel:     minorRel,
		MaintRel:     maintRel,
	}, nil
}

// StackTuneOperation defines stack tune operations.
type StackTuneOperation uint8

const (
	// StackTuneSetTxPower sets the TX power level.
	StackTuneSetTxPower StackTuneOperation = 0
)

// StackTune tunes stack parameters.
// This is the preferred method for setting TX power on Z-Stack 3.0.
// Operation 0 = set TX power, value = power in dBm.
// Returns the actual value set.
func (z *ZNP) StackTune(ctx context.Context, operation StackTuneOperation, value uint8) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint8(uint8(operation))
	writer.WriteUint8(value)

	// Send CmdSysStackTune SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysStackTune, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("stack tune request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	actualValue, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read actual value: %w", err)
	}

	return actualValue, nil
}

// SetTxPower sets the transmit power level using StackTune.
// Returns the actual TX power set (may differ from requested due to hardware limits).
// Power is specified in dBm (typically -20 to +20).
func (z *ZNP) SetTxPower(ctx context.Context, power int8) (int8, error) {
	// Convert signed power to unsigned for StackTune
	// Z-Stack expects unsigned byte where negative values wrap around
	var powerUint uint8
	if power < 0 {
		powerUint = uint8(256 + int(power)) // Two's complement
	} else {
		powerUint = uint8(power)
	}

	actual, err := z.StackTune(ctx, StackTuneSetTxPower, powerUint)
	if err != nil {
		return 0, err
	}

	// Convert back to signed
	var actualSigned int8
	if actual > 127 {
		actualSigned = int8(int(actual) - 256)
	} else {
		actualSigned = int8(actual)
	}

	return actualSigned, nil
}

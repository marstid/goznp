package znp

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/unpi"
)

// BdbStartCommissioning starts the BDB commissioning process.
// mode determines the commissioning action:
//   - BdbCommissioningModeNetworkSteering (0x02): Join existing network
//   - BdbCommissioningModeNetworkFormation (0x04): Form new network (coordinator)
//   - BdbCommissioningModeFindingBinding (0x08): Find and bind
func (z *ZNP) BdbStartCommissioning(ctx context.Context, mode BdbCommissioningMode) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint8(uint8(mode))

	// Send CmdAppCnfBdbStartCommissioning SREQ
	resp, err := z.Request(ctx, unpi.APPCNF, CmdAppCnfBdbStartCommissioning, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("bdb start commissioning request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	return status, nil
}

// BdbSetChannel sets the BDB channel for commissioning.
// isPrimary specifies if this is the primary (true) or secondary (false) channel.
// channel is a channel mask (bit N = channel N).
func (z *ZNP) BdbSetChannel(ctx context.Context, isPrimary bool, channel uint32) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	if isPrimary {
		writer.WriteUint8(1)
	} else {
		writer.WriteUint8(0)
	}
	writer.WriteUint32(channel)

	// Send CmdAppCnfBdbSetChannel SREQ
	resp, err := z.Request(ctx, unpi.APPCNF, CmdAppCnfBdbSetChannel, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("bdb set channel request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	return status, nil
}

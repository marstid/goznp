package adapter

import (
	"context"

	"github.com/marstid/goznp/pkg/zcl"
)

// Identify Cluster (0x0003)

// Identify starts the identify mode on a device for the specified duration.
// The device will perform a visual/audible identification (e.g., flashing lights).
// Duration is in seconds. Use 0 to stop identifying.
func (a *Adapter) Identify(ctx context.Context, nwkAddr uint16, endpoint uint8, durationSecs uint16) error {
	// Identify payload: identifyTime (2 bytes LE)
	payload := []byte{
		byte(durationSecs & 0xFF),
		byte(durationSecs >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIdentify, zcl.CmdIdentify, payload)
}

// TriggerEffect triggers a specific identification effect on a device.
// Effects: Blink (0x00), Breathe (0x01), Okay (0x02), ChannelChange (0x0B),
// FinishEffect (0xFE), StopEffect (0xFF)
func (a *Adapter) TriggerEffect(ctx context.Context, nwkAddr uint16, endpoint uint8, effectID uint8, effectVariant uint8) error {
	// TriggerEffect payload: effectId (1 byte) + effectVariant (1 byte)
	payload := []byte{effectID, effectVariant}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIdentify, zcl.CmdTriggerEffect, payload)
}

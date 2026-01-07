package adapter

import (
	"context"

	"github.com/marstid/goznp/pkg/zcl"
)

// TurnOn sends On command to a device.
func (a *Adapter) TurnOn(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOn, nil)
}

// TurnOff sends Off command to a device.
func (a *Adapter) TurnOff(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOff, nil)
}

// Toggle sends Toggle command to a device.
func (a *Adapter) Toggle(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffToggle, nil)
}

// OffWithEffect turns off a device with a visual effect (typically for lights).
// effectID specifies the effect type:
//   - 0x00: DelayedAllOff - fade to off over 0.8 seconds
//   - 0x01: DyingLight - 50% dim down in 0.8s then fade to off in 12s
//
// effectVariant is an effect-specific variant value (typically 0 for default).
func (a *Adapter) OffWithEffect(ctx context.Context, nwkAddr uint16, endpoint uint8, effectID, effectVariant uint8) error {
	// OffWithEffect payload: effectId (1 byte) + effectVariant (1 byte)
	payload := []byte{effectID, effectVariant}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOffWithEffect, payload)
}

// OnWithTimedOff turns on a device for a specified duration.
// The device will automatically turn off after the specified time.
// onTime and offWaitTime are in 1/10th seconds (e.g., 10 = 1 second).
//
// Parameters:
//   - onTime: Duration the device stays on (1/10th seconds)
//   - offWaitTime: Additional delay before turning off (1/10th seconds)
//
// The onOffControl parameter is typically 0x00 for normal operation.
func (a *Adapter) OnWithTimedOff(ctx context.Context, nwkAddr uint16, endpoint uint8, onTime, offWaitTime uint16) error {
	// OnWithTimedOff payload: onOffControl (1 byte) + onTime (2 bytes LE) + offWaitTime (2 bytes LE)
	payload := []byte{
		0x00, // onOffControl (0x00 = accept command only if device is currently off)
		byte(onTime & 0xFF),
		byte(onTime >> 8),
		byte(offWaitTime & 0xFF),
		byte(offWaitTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOnWithTimedOff, payload)
}

// SetStartupOnOff configures the device behavior when it powers on.
// value specifies the startup behavior:
//   - 0x00: Turn off when powered on
//   - 0x01: Turn on when powered on
//   - 0x02: Toggle state when powered on
//   - 0xFF: Restore previous state when powered on
//
// This setting is persistent across power cycles.
func (a *Adapter) SetStartupOnOff(ctx context.Context, nwkAddr uint16, endpoint uint8, value uint8) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrStartUpOnOff: {
			Type:  zcl.TypeEnum8,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, values)
}

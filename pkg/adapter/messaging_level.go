package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// Level Control Cluster (0x0008)

// SetBrightness sets the brightness level on a dimmable device.
// level is 0-254 (0=off, 254=full brightness).
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
// Uses MoveToLevelWithOnOff command which also handles on/off state.
func (a *Adapter) SetBrightness(ctx context.Context, nwkAddr uint16, endpoint uint8, level uint8, transitionTime uint16) error {
	// MoveToLevelWithOnOff payload: level (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		level,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelMoveToLevelWithOnOff, payload)
}

// GetBrightness reads the current brightness level from a device.
// Returns 0-254 (0=off, 254=full brightness).
func (a *Adapter) GetBrightness(ctx context.Context, nwkAddr uint16, endpoint uint8) (uint8, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.AttrLevelCurrentLevel)
	if err != nil {
		return 0, fmt.Errorf("failed to read brightness level: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read brightness returned status 0x%02X", result.Status)
	}

	if level, ok := result.Value.(uint8); ok {
		return level, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// MoveLevel starts continuously changing the level at the specified rate.
// The level will continue changing until it reaches the limit or StopLevel is called.
//
// Parameters:
//   - moveMode: Direction of movement (zcl.MoveModeUp or zcl.MoveModeDown)
//   - rate: Units per second to change the level (0 = use device's DefaultMoveRate attribute)
//
// Example: Increase brightness at 50 units/second:
//
//	err := adapter.MoveLevel(ctx, nwkAddr, endpoint, uint8(zcl.MoveModeUp), 50)
func (a *Adapter) MoveLevel(ctx context.Context, nwkAddr uint16, endpoint uint8, moveMode, rate uint8) error {
	// Move payload: moveMode (1 byte) + rate (1 byte)
	payload := []byte{moveMode, rate}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelMove, payload)
}

// StepLevel changes the level by a fixed amount over a transition time.
// This is useful for dimming/brightening by a specific increment.
//
// Parameters:
//   - stepMode: Direction of step (zcl.StepModeUp or zcl.StepModeDown)
//   - stepSize: Amount to change the level (0-254)
//   - transitionTime: Time for the transition in tenths of a second (e.g., 10 = 1 second)
//
// Example: Decrease brightness by 20 units over 0.5 seconds:
//
//	err := adapter.StepLevel(ctx, nwkAddr, endpoint, uint8(zcl.StepModeDown), 20, 5)
func (a *Adapter) StepLevel(ctx context.Context, nwkAddr uint16, endpoint uint8, stepMode, stepSize uint8, transitionTime uint16) error {
	// Step payload: stepMode (1 byte) + stepSize (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		stepMode,
		stepSize,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelStep, payload)
}

// StopLevel stops any ongoing level change started by MoveLevel or StepLevel.
// This command immediately halts the level transition and maintains the current level.
//
// Example:
//
//	err := adapter.StopLevel(ctx, nwkAddr, endpoint)
func (a *Adapter) StopLevel(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// Stop command has no payload
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelStop, nil)
}

// SetStartupLevel configures the brightness level the device uses when powered on.
// This persists across power cycles, allowing devices to remember their preferred startup state.
//
// Parameters:
//   - level: The startup level (0-254 for specific level, 255 to restore previous level)
//
// Special values:
//   - 0-254: Set to a specific brightness level on power-up
//   - 255 (0xFF): Restore the level from before power loss
//
// Example: Set device to turn on at 50% brightness:
//
//	err := adapter.SetStartupLevel(ctx, nwkAddr, endpoint, 127)
//
// Example: Set device to restore previous level on power-up:
//
//	err := adapter.SetStartupLevel(ctx, nwkAddr, endpoint, 255)
func (a *Adapter) SetStartupLevel(ctx context.Context, nwkAddr uint16, endpoint uint8, level uint8) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrLevelStartupLevel: {
			Type:  zcl.TypeUint8,
			Value: level,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, values)
}

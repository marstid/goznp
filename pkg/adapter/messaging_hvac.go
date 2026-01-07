package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// Thermostat Cluster (0x0201)

// ThermostatStatus contains current thermostat state.
type ThermostatStatus struct {
	LocalTemperature *float64                  // Current temperature in °C
	CoolingSetpoint  *float64                  // Cooling setpoint in °C
	HeatingSetpoint  *float64                  // Heating setpoint in °C
	SystemMode       *zcl.ThermostatSystemMode // Current operating mode
	RunningState     *uint16                   // Bitmap of running states
	CoolingDemand    *uint8                    // 0-100%
	HeatingDemand    *uint8                    // 0-100%
}

// GetThermostatStatus reads the current thermostat status from a device.
// Returns the current temperature, setpoints, mode, and demand values.
// Temperature values are converted from centidegrees (0.01°C) to degrees Celsius.
func (a *Adapter) GetThermostatStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*ThermostatStatus, error) {
	// Define attributes to read
	attrs := []zcl.AttributeID{
		zcl.AttrThermostatLocalTemp,
		zcl.AttrThermostatOccupiedCoolingSetpoint,
		zcl.AttrThermostatOccupiedHeatingSetpoint,
		zcl.AttrThermostatSystemMode,
		zcl.AttrThermostatRunningState,
		zcl.AttrThermostatPICoolingDemand,
		zcl.AttrThermostatPIHeatingDemand,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read thermostat attributes: %w", err)
	}

	status := &ThermostatStatus{}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrThermostatLocalTemp:
			if raw, ok := toInt16(r.Value); ok {
				// Convert from centidegrees (0.01°C) to degrees
				temp := float64(raw) / 100.0
				status.LocalTemperature = &temp
			}
		case zcl.AttrThermostatOccupiedCoolingSetpoint:
			if raw, ok := toInt16(r.Value); ok {
				// Convert from centidegrees (0.01°C) to degrees
				setpoint := float64(raw) / 100.0
				status.CoolingSetpoint = &setpoint
			}
		case zcl.AttrThermostatOccupiedHeatingSetpoint:
			if raw, ok := toInt16(r.Value); ok {
				// Convert from centidegrees (0.01°C) to degrees
				setpoint := float64(raw) / 100.0
				status.HeatingSetpoint = &setpoint
			}
		case zcl.AttrThermostatSystemMode:
			if mode, ok := r.Value.(uint8); ok {
				m := zcl.ThermostatSystemMode(mode)
				status.SystemMode = &m
			}
		case zcl.AttrThermostatRunningState:
			if state, ok := toUint16(r.Value); ok {
				status.RunningState = &state
			}
		case zcl.AttrThermostatPICoolingDemand:
			if demand, ok := r.Value.(uint8); ok {
				status.CoolingDemand = &demand
			}
		case zcl.AttrThermostatPIHeatingDemand:
			if demand, ok := r.Value.(uint8); ok {
				status.HeatingDemand = &demand
			}
		}
	}

	return status, nil
}

// SetThermostatMode sets the operating mode of a thermostat.
// mode specifies the desired operating mode (Off, Auto, Cool, Heat, etc.).
func (a *Adapter) SetThermostatMode(ctx context.Context, nwkAddr uint16, endpoint uint8, mode zcl.ThermostatSystemMode) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrThermostatSystemMode: {
			Type:  zcl.TypeEnum8,
			Value: uint8(mode),
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, values)
}

// SetThermostatSetpoint sets the heating and cooling setpoints on a thermostat.
// Temperatures are in degrees Celsius and are converted to centidegrees (0.01°C) for the device.
// heating: Desired heating setpoint in °C
// cooling: Desired cooling setpoint in °C
func (a *Adapter) SetThermostatSetpoint(ctx context.Context, nwkAddr uint16, endpoint uint8, heating, cooling float64) error {
	// Convert from degrees to centidegrees (0.01°C)
	heatingCentidegrees := int16(heating * 100)
	coolingCentidegrees := int16(cooling * 100)

	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrThermostatOccupiedHeatingSetpoint: {
			Type:  zcl.TypeInt16,
			Value: heatingCentidegrees,
		},
		zcl.AttrThermostatOccupiedCoolingSetpoint: {
			Type:  zcl.TypeInt16,
			Value: coolingCentidegrees,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, values)
}

// AdjustThermostatSetpoint raises or lowers the thermostat setpoint.
// This uses the SetpointRaiseLower command to adjust the setpoint by a relative amount.
//
// Parameters:
//   - mode: 0=Heat, 1=Cool, 2=Both
//   - amount: Adjustment amount in 0.1°C increments (positive to raise, negative to lower)
//
// Example: Raise heating setpoint by 1°C:
//
//	err := adapter.AdjustThermostatSetpoint(ctx, nwkAddr, endpoint, 0, 10)
func (a *Adapter) AdjustThermostatSetpoint(ctx context.Context, nwkAddr uint16, endpoint uint8, mode uint8, amount int8) error {
	// SetpointRaiseLower payload: mode (1 byte) + amount (1 byte, signed)
	payload := []byte{mode, uint8(amount)}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, zcl.CmdThermostatSetpointRaiseLower, payload)
}

// WindowCovering Cluster (0x0102)

// WindowCoveringStatus contains current covering position.
type WindowCoveringStatus struct {
	Type         *zcl.WindowCoveringType
	LiftPercent  *uint8 // 0=fully open, 100=fully closed
	TiltPercent  *uint8 // 0=fully open, 100=fully closed
	ConfigStatus *uint8
}

// GetWindowCoveringStatus reads the current window covering status from a device.
// Returns the covering type, lift percentage (0=fully open, 100=fully closed),
// tilt percentage, and configuration status.
func (a *Adapter) GetWindowCoveringStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*WindowCoveringStatus, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrWindowCoveringType,
		zcl.AttrWindowCoveringCurrentPositionLiftPercent,
		zcl.AttrWindowCoveringCurrentPositionTiltPercent,
		zcl.AttrWindowCoveringConfigStatus,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read window covering status: %w", err)
	}

	status := &WindowCoveringStatus{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrWindowCoveringType:
			if val, ok := r.Value.(uint8); ok {
				coveringType := zcl.WindowCoveringType(val)
				status.Type = &coveringType
			}
		case zcl.AttrWindowCoveringCurrentPositionLiftPercent:
			if val, ok := r.Value.(uint8); ok {
				status.LiftPercent = &val
			}
		case zcl.AttrWindowCoveringCurrentPositionTiltPercent:
			if val, ok := r.Value.(uint8); ok {
				status.TiltPercent = &val
			}
		case zcl.AttrWindowCoveringConfigStatus:
			if val, ok := r.Value.(uint8); ok {
				status.ConfigStatus = &val
			}
		}
	}

	return status, nil
}

// OpenWindowCovering fully opens (raises) the covering.
// This sends the UpOpen command to the device.
func (a *Adapter) OpenWindowCovering(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringUpOpen, nil)
}

// CloseWindowCovering fully closes (lowers) the covering.
// This sends the DownClose command to the device.
func (a *Adapter) CloseWindowCovering(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringDownClose, nil)
}

// StopWindowCovering stops any ongoing movement.
// This immediately halts the covering's motion and maintains the current position.
func (a *Adapter) StopWindowCovering(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringStop, nil)
}

// SetWindowCoveringPosition sets the lift position as percentage (0=open, 100=closed).
// The covering will move to the specified position.
// percent is 0-100 where 0 is fully open and 100 is fully closed.
func (a *Adapter) SetWindowCoveringPosition(ctx context.Context, nwkAddr uint16, endpoint uint8, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("percent must be 0-100, got %d", percent)
	}
	// GoToLiftPercent payload: percentageLiftValue (1 byte)
	payload := []byte{percent}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringGoToLiftPercent, payload)
}

// SetWindowCoveringTilt sets the tilt angle as percentage (0=open, 100=closed).
// The covering slats will tilt to the specified angle.
// percent is 0-100 where 0 is fully open and 100 is fully closed.
func (a *Adapter) SetWindowCoveringTilt(ctx context.Context, nwkAddr uint16, endpoint uint8, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("percent must be 0-100, got %d", percent)
	}
	// GoToTiltPercent payload: percentageTiltValue (1 byte)
	payload := []byte{percent}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringGoToTiltPercent, payload)
}

// Fan Cluster (0x0202)

// FanStatus contains current fan state.
type FanStatus struct {
	Mode           *zcl.FanMode         // Current fan mode
	ModeSequence   *zcl.FanModeSequence // Supported fan mode sequence
	PercentSetting *uint8               // Fan speed setting as percentage (0-100%)
	PercentCurrent *uint8               // Current fan speed as percentage (0-100%)
	SpeedMax       *uint8               // Maximum fan speed
	SpeedSetting   *uint8               // Fan speed setting
	SpeedCurrent   *uint8               // Current fan speed
}

// GetFanStatus reads the current fan status from a device.
// Returns all available fan attributes including mode, speed, and percentages.
func (a *Adapter) GetFanStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*FanStatus, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrFanMode,
		zcl.AttrFanModeSequence,
		zcl.AttrFanPercentSetting,
		zcl.AttrFanPercentCurrent,
		zcl.AttrFanSpeedMax,
		zcl.AttrFanSpeedSetting,
		zcl.AttrFanSpeedCurrent,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFan, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read fan attributes: %w", err)
	}

	status := &FanStatus{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrFanMode:
			if val, ok := r.Value.(uint8); ok {
				mode := zcl.FanMode(val)
				status.Mode = &mode
			}
		case zcl.AttrFanModeSequence:
			if val, ok := r.Value.(uint8); ok {
				seq := zcl.FanModeSequence(val)
				status.ModeSequence = &seq
			}
		case zcl.AttrFanPercentSetting:
			if val, ok := r.Value.(uint8); ok {
				status.PercentSetting = &val
			}
		case zcl.AttrFanPercentCurrent:
			if val, ok := r.Value.(uint8); ok {
				status.PercentCurrent = &val
			}
		case zcl.AttrFanSpeedMax:
			if val, ok := r.Value.(uint8); ok {
				status.SpeedMax = &val
			}
		case zcl.AttrFanSpeedSetting:
			if val, ok := r.Value.(uint8); ok {
				status.SpeedSetting = &val
			}
		case zcl.AttrFanSpeedCurrent:
			if val, ok := r.Value.(uint8); ok {
				status.SpeedCurrent = &val
			}
		}
	}

	return status, nil
}

// SetFanMode sets the operating mode of a fan.
// mode specifies the desired fan mode (Off, Low, Medium, High, On, Auto, Smart).
func (a *Adapter) SetFanMode(ctx context.Context, nwkAddr uint16, endpoint uint8, mode zcl.FanMode) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrFanMode: {
			Type:  zcl.TypeEnum8,
			Value: uint8(mode),
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFan, values)
}

// SetFanSpeed sets the fan speed as a percentage.
// percent is 0-100 where 0 is off and 100 is maximum speed.
func (a *Adapter) SetFanSpeed(ctx context.Context, nwkAddr uint16, endpoint uint8, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("percent must be 0-100, got %d", percent)
	}
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrFanPercentSetting: {
			Type:  zcl.TypeUint8,
			Value: percent,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFan, values)
}

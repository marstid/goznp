package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// Power Data Types

// PowerData contains power consumption readings from a smart plug.
type PowerData struct {
	// Electrical Measurement cluster (0x0B04) - instantaneous values
	Voltage     *float64 // RMS Voltage in volts
	Current     *float64 // RMS Current in amps
	ActivePower *float64 // Active power in watts
	PowerFactor *int8    // Power factor (-100 to 100%)

	// Simple Metering cluster (0x0702) - cumulative values
	TotalEnergy  *float64 // Total energy in kWh
	InstantPower *float64 // Instantaneous demand in watts (from metering)

	// Debug info
	RawVoltage uint16
	RawCurrent uint16
	RawPower   int16
	RawEnergy  uint64
}

// BatteryInfo contains battery status information.
type BatteryInfo struct {
	Voltage    *float64 // Voltage in V (nil if not available)
	Percentage *uint8   // Percentage remaining 0-100% (nil if not available)
	LowBattery bool     // True if battery alarm is active
}

// MainsInfo contains mains power information.
type MainsInfo struct {
	Voltage   *float64 // Voltage in V (nil if not available)
	Frequency *uint8   // Frequency in Hz (nil if not available)
}

// Electrical Measurement & Metering

// ReadPowerData reads power consumption data from a smart plug.
func (a *Adapter) ReadPowerData(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PowerData, error) {
	data := &PowerData{}

	// Try to read from Electrical Measurement cluster (0x0B04)
	elecAttrs := []zcl.AttributeID{
		zcl.AttrElecRMSVoltage,
		zcl.AttrElecRMSCurrent,
		zcl.AttrElecActivePower,
		zcl.AttrElecPowerFactor,
		zcl.AttrElecACVoltageDivisor,
		zcl.AttrElecACCurrentDivisor,
		zcl.AttrElecACPowerDivisor,
	}

	elecResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterElectricalMeas, elecAttrs...)
	if err == nil {
		// Parse divisors first (defaults for when device doesn't report them)
		// Tuya TS011F typically reports:
		// - Voltage: actual value (divisor 1)
		// - Current: in mA (divisor 1000)
		// - Power: actual value (divisor 1)
		voltageDivisor := uint16(1)
		currentDivisor := uint16(1000) // Most Tuya plugs report current in mA
		powerDivisor := uint16(1)

		for _, r := range elecResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrElecACVoltageDivisor:
				if v, ok := toUint16(r.Value); ok && v > 0 {
					voltageDivisor = v
				}
			case zcl.AttrElecACCurrentDivisor:
				if v, ok := toUint16(r.Value); ok && v > 0 {
					currentDivisor = v
				}
			case zcl.AttrElecACPowerDivisor:
				if v, ok := toUint16(r.Value); ok && v > 0 {
					powerDivisor = v
				}
			}
		}

		// Now parse measurement values
		for _, r := range elecResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrElecRMSVoltage:
				if v, ok := toUint16(r.Value); ok {
					data.RawVoltage = v
					voltage := float64(v) / float64(voltageDivisor)
					data.Voltage = &voltage
				}
			case zcl.AttrElecRMSCurrent:
				if v, ok := toUint16(r.Value); ok {
					data.RawCurrent = v
					current := float64(v) / float64(currentDivisor)
					data.Current = &current
				}
			case zcl.AttrElecActivePower:
				if v, ok := toInt16(r.Value); ok {
					data.RawPower = v
					power := float64(v) / float64(powerDivisor)
					data.ActivePower = &power
				}
			case zcl.AttrElecPowerFactor:
				if v, ok := toInt8(r.Value); ok {
					data.PowerFactor = &v
				}
			}
		}
	}

	// Try to read from Simple Metering cluster (0x0702)
	meterAttrs := []zcl.AttributeID{
		zcl.AttrMeterCurrentSummation,
		zcl.AttrMeterInstantDemand,
		zcl.AttrMeterMultiplier,
		zcl.AttrMeterDivisor,
	}

	meterResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMeteringSimple, meterAttrs...)
	if err == nil {
		// Parse multiplier/divisor first (defaults)
		multiplier := uint32(1)
		divisor := uint32(1)

		for _, r := range meterResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrMeterMultiplier:
				if v, ok := toUint32(r.Value); ok && v > 0 {
					multiplier = v
				}
			case zcl.AttrMeterDivisor:
				if v, ok := toUint32(r.Value); ok && v > 0 {
					divisor = v
				}
			}
		}

		// Now parse measurement values
		for _, r := range meterResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrMeterCurrentSummation:
				if v, ok := toUint48(r.Value); ok {
					data.RawEnergy = v
					// Apply multiplier/divisor to get kWh
					energy := float64(v) * float64(multiplier) / float64(divisor)
					data.TotalEnergy = &energy
				}
			case zcl.AttrMeterInstantDemand:
				if v, ok := toInt32(r.Value); ok {
					// Apply multiplier/divisor to get watts
					power := float64(v) * float64(multiplier) / float64(divisor) * 1000 // Convert kW to W
					data.InstantPower = &power
				}
			}
		}
	}

	return data, nil
}

// PowerConfiguration Cluster (0x0001)

// GetBatteryInfo reads battery status from a device.
// Returns battery voltage, percentage, and alarm status from the PowerConfiguration cluster.
// Voltage is converted from device units (0.1V) to volts.
// Percentage is converted from device range (0-200 = 0-100%) to 0-100%.
func (a *Adapter) GetBatteryInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*BatteryInfo, error) {
	info := &BatteryInfo{}

	// Read battery attributes
	attrs := []zcl.AttributeID{
		zcl.AttrPowerBatteryVoltage,
		zcl.AttrPowerBatteryPercentage,
		zcl.AttrPowerBatteryAlarmState,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read battery attributes: %w", err)
	}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrPowerBatteryVoltage:
			if raw, ok := r.Value.(uint8); ok {
				// Convert from 0.1V units to volts
				voltage := float64(raw) / 10.0
				info.Voltage = &voltage
			}
		case zcl.AttrPowerBatteryPercentage:
			if raw, ok := r.Value.(uint8); ok {
				// Convert from 0-200 (0.5% steps) to 0-100%
				percentage := raw / 2
				info.Percentage = &percentage
			}
		case zcl.AttrPowerBatteryAlarmState:
			// Check if any alarm bit is set
			if alarmState, ok := toUint32(r.Value); ok {
				info.LowBattery = alarmState != 0
			}
		}
	}

	return info, nil
}

// GetMainsInfo reads mains power information from a device.
// Returns mains voltage and frequency from the PowerConfiguration cluster.
// Voltage is converted from device units (0.1V) to volts.
// Frequency is converted from device units (2Hz increments) to Hz.
func (a *Adapter) GetMainsInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*MainsInfo, error) {
	info := &MainsInfo{}

	// Read mains attributes
	attrs := []zcl.AttributeID{
		zcl.AttrPowerMainsVoltage,
		zcl.AttrPowerMainsFrequency,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read mains attributes: %w", err)
	}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrPowerMainsVoltage:
			if raw, ok := toUint16(r.Value); ok {
				// Convert from 0.1V units to volts
				voltage := float64(raw) / 10.0
				info.Voltage = &voltage
			}
		case zcl.AttrPowerMainsFrequency:
			if raw, ok := r.Value.(uint8); ok {
				// Convert from 2Hz increments to Hz
				frequency := raw * 2
				info.Frequency = &frequency
			}
		}
	}

	return info, nil
}

// Energy Management

// ResetEnergy attempts to reset the energy counter on a smart plug.
// This uses manufacturer-specific methods and may not work on all devices.
// For Tuya TS011F plugs, zigbee2mqtt uses Basic cluster resetFactDefault command.
func (a *Adapter) ResetEnergy(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// Method 1: Basic cluster (0x0000) command 0 (resetFactDefault)
	// This is what zigbee2mqtt uses for TS011F energy reset
	err := a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterBasic, 0x00, nil)
	if err == nil {
		return nil
	}

	// Method 2: Tuya cluster 0xE001, attribute 0xD004, value 1
	tuyaValues1 := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTuyaEnergyReset: {
			Type:  zcl.TypeEnum8,
			Value: uint8(1),
		},
	}
	err = a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTuyaSpecific, tuyaValues1)
	if err == nil {
		return nil
	}

	// Method 3: Tuya cluster 0xE000, attribute 0xD004
	tuyaValues2 := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTuyaEnergyReset: {
			Type:  zcl.TypeEnum8,
			Value: uint8(1),
		},
	}
	err = a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterID(0xE000), tuyaValues2)
	if err == nil {
		return nil
	}

	// Method 4: Try writing 0 to the metering summation attribute (standard but rarely works)
	meterValues := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrMeterCurrentSummation: {
			Type:  zcl.TypeUint48,
			Value: uint64(0),
		},
	}
	err = a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMeteringSimple, meterValues)
	if err == nil {
		return nil
	}

	return fmt.Errorf("energy reset failed (tried 4 methods): %w", err)
}

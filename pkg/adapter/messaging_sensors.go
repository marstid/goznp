package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
)

// Sensor Data Types

// SensorData contains temperature and humidity sensor readings.
type SensorData struct {
	Temperature *float32 // Celsius, nil if not available
	Humidity    *float32 // Percent, nil if not available
	Battery     *uint8   // Percent, nil if not available
}

// SensorReport contains a sensor report with device address.
type SensorReport struct {
	NwkAddr     uint16
	ClusterID   uint16 // For debugging
	Endpoint    uint8  // For debugging
	Temperature *float32
	Humidity    *float32
	Battery     *uint8
	// Power consumption data from smart plugs
	Voltage *float64 // RMS Voltage in volts
	Current *float64 // RMS Current in amps
	Power   *float64 // Active power in watts
	Energy  *float64 // Total energy in kWh
	// On/Off state
	OnOff *bool
	// Debug info
	FrameCommandID uint8
	ParseError     string
	Attributes     []ReportedAttribute
}

// ReportedAttribute contains raw attribute data for debugging.
type ReportedAttribute struct {
	ID    uint16
	Value interface{}
}

// PressureData contains atmospheric pressure readings.
type PressureData struct {
	Pressure    *float64 // Pressure in hPa (hectopascals)
	MinPressure *float64 // Minimum measurable pressure in hPa
	MaxPressure *float64 // Maximum measurable pressure in hPa
}

// AirQualityData contains air quality sensor readings.
type AirQualityData struct {
	CO           *float32 // Carbon monoxide in ppm
	CO2          *float32 // Carbon dioxide in ppm
	PM25         *float32 // PM2.5 in µg/m³
	Formaldehyde *float32 // Formaldehyde in ppm
}

// Temperature & Humidity Measurement

// ReadSensorData reads temperature, humidity, and battery from a sensor.
func (a *Adapter) ReadSensorData(ctx context.Context, nwkAddr uint16, endpoint uint8) (*SensorData, error) {
	data := &SensorData{}

	// Read temperature (cluster 0x0402, attr 0x0000)
	// Value is int16 in units of 0.01°C
	tempResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTempMeasurement, zcl.AttrTempMeasuredValue)
	if err == nil && len(tempResults) > 0 && tempResults[0].Status == zcl.StatusSuccess {
		if raw, ok := tempResults[0].Value.(int16); ok {
			temp := float32(raw) / 100.0
			data.Temperature = &temp
		}
	}

	// Read humidity (cluster 0x0405, attr 0x0000)
	// Value is uint16 in units of 0.01%
	humResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterHumidityMeas, zcl.AttrHumidityMeasuredValue)
	if err == nil && len(humResults) > 0 && humResults[0].Status == zcl.StatusSuccess {
		if raw, ok := humResults[0].Value.(uint16); ok {
			hum := float32(raw) / 100.0
			data.Humidity = &hum
		}
	}

	// Read battery percentage (cluster 0x0001, attr 0x0021)
	// Value is uint8, 0-200 representing 0-100% (0.5% steps)
	battResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, zcl.AttrPowerBatteryPercentage)
	if err == nil && len(battResults) > 0 && battResults[0].Status == zcl.StatusSuccess {
		if raw, ok := battResults[0].Value.(uint8); ok {
			batt := raw / 2 // Convert from 0-200 to 0-100%
			data.Battery = &batt
		}
	}

	return data, nil
}

// Pressure Measurement Cluster (0x0403)

// GetPressure reads atmospheric pressure from a device.
// Returns pressure in hPa (hectopascals), also known as millibars.
// The device reports pressure in units of 10 Pa (0.1 hPa), which is converted to hPa.
// Some devices support scaled values with a scale attribute (10^scale multiplier).
func (a *Adapter) GetPressure(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PressureData, error) {
	data := &PressureData{}

	// Try to read regular pressure attributes first
	regularAttrs := []zcl.AttributeID{
		zcl.AttrPressureMeasuredValue,
		zcl.AttrPressureMinMeasuredValue,
		zcl.AttrPressureMaxMeasuredValue,
	}

	regularResults, regularErr := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPressureMeas, regularAttrs...)

	// Try to read scaled pressure attributes and scale
	scaledAttrs := []zcl.AttributeID{
		zcl.AttrPressureScaledValue,
		zcl.AttrPressureMinScaledValue,
		zcl.AttrPressureMaxScaledValue,
		zcl.AttrPressureScale,
	}

	scaledResults, scaledErr := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPressureMeas, scaledAttrs...)

	// If both failed, return error
	if regularErr != nil && scaledErr != nil {
		return nil, fmt.Errorf("failed to read pressure attributes: %w", regularErr)
	}

	// Check if we have a scale attribute for scaled values
	var scale int8
	hasScale := false
	if scaledErr == nil {
		for _, r := range scaledResults {
			if r.AttributeID == zcl.AttrPressureScale && r.Status == zcl.StatusSuccess {
				if s, ok := toInt8(r.Value); ok {
					scale = s
					hasScale = true
				}
			}
		}
	}

	// Calculate scale multiplier: 10^scale
	// For example: scale=-1 means multiply by 0.1, scale=0 means multiply by 1
	scaleMultiplier := 1.0
	if hasScale {
		for i := int8(0); i < scale; i++ {
			scaleMultiplier *= 10.0
		}
		for i := int8(0); i > scale; i-- {
			scaleMultiplier /= 10.0
		}
	}

	// Try scaled values first if scale is available and we got successful reads
	if hasScale && scaledErr == nil {
		for _, r := range scaledResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}

			switch r.AttributeID {
			case zcl.AttrPressureScaledValue:
				if raw, ok := toInt16(r.Value); ok {
					// Apply scale multiplier to get hPa
					pressure := float64(raw) * scaleMultiplier
					data.Pressure = &pressure
				}
			case zcl.AttrPressureMinScaledValue:
				if raw, ok := toInt16(r.Value); ok {
					minPressure := float64(raw) * scaleMultiplier
					data.MinPressure = &minPressure
				}
			case zcl.AttrPressureMaxScaledValue:
				if raw, ok := toInt16(r.Value); ok {
					maxPressure := float64(raw) * scaleMultiplier
					data.MaxPressure = &maxPressure
				}
			}
		}
	}

	// If scaled values didn't work or aren't available, use regular values
	// Regular values are in units of 10 Pa (0.1 hPa), so divide by 10 to get hPa
	if data.Pressure == nil && regularErr == nil {
		for _, r := range regularResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}

			switch r.AttributeID {
			case zcl.AttrPressureMeasuredValue:
				if raw, ok := toInt16(r.Value); ok {
					// Convert from 10 Pa (0.1 hPa) to hPa
					pressure := float64(raw) / 10.0
					data.Pressure = &pressure
				}
			case zcl.AttrPressureMinMeasuredValue:
				if raw, ok := toInt16(r.Value); ok {
					minPressure := float64(raw) / 10.0
					data.MinPressure = &minPressure
				}
			case zcl.AttrPressureMaxMeasuredValue:
				if raw, ok := toInt16(r.Value); ok {
					maxPressure := float64(raw) / 10.0
					data.MaxPressure = &maxPressure
				}
			}
		}
	}

	return data, nil
}

// Air Quality Measurement Clusters

// GetAirQuality reads air quality measurements from a device.
// It will read from whichever clusters the device supports.
// Values are nil if the device doesn't support that measurement type.
func (a *Adapter) GetAirQuality(ctx context.Context, nwkAddr uint16, endpoint uint8) (*AirQualityData, error) {
	data := &AirQualityData{}

	// Try to read CO (cluster 0x040C, attr 0x0000)
	// Value is float32 (single) in ppm
	coResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterCarbonMonoxide, zcl.AttrCOMeasuredValue)
	if err == nil && len(coResults) > 0 && coResults[0].Status == zcl.StatusSuccess {
		if val, ok := coResults[0].Value.(float32); ok {
			data.CO = &val
		}
	}

	// Try to read CO2 (cluster 0x040D, attr 0x0000)
	// Value is float32 (single) in ppm
	co2Results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterCarbonDioxide, zcl.AttrCO2MeasuredValue)
	if err == nil && len(co2Results) > 0 && co2Results[0].Status == zcl.StatusSuccess {
		if val, ok := co2Results[0].Value.(float32); ok {
			data.CO2 = &val
		}
	}

	// Try to read PM2.5 (cluster 0x042A, attr 0x0000)
	// Value is float32 (single) in µg/m³
	pm25Results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPM25Measurement, zcl.AttrPM25MeasuredValue)
	if err == nil && len(pm25Results) > 0 && pm25Results[0].Status == zcl.StatusSuccess {
		if val, ok := pm25Results[0].Value.(float32); ok {
			data.PM25 = &val
		}
	}

	// Try to read Formaldehyde (cluster 0x042B, attr 0x0000)
	// Value is float32 (single) in ppm
	formaldehydeResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFormaldehydeMeasurement, zcl.AttrFormaldehydeMeasuredValue)
	if err == nil && len(formaldehydeResults) > 0 && formaldehydeResults[0].Status == zcl.StatusSuccess {
		if val, ok := formaldehydeResults[0].Value.(float32); ok {
			data.Formaldehyde = &val
		}
	}

	return data, nil
}

// GetCO2Level reads carbon dioxide concentration.
// Returns the CO2 level in ppm, or nil if not available.
func (a *Adapter) GetCO2Level(ctx context.Context, nwkAddr uint16, endpoint uint8) (*float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterCarbonDioxide, zcl.AttrCO2MeasuredValue)
	if err != nil {
		return nil, fmt.Errorf("failed to read CO2 level: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return nil, fmt.Errorf("read CO2 level returned status 0x%02X", result.Status)
	}

	if val, ok := result.Value.(float32); ok {
		return &val, nil
	}
	return nil, fmt.Errorf("unexpected value type: %T", result.Value)
}

// GetPM25Level reads PM2.5 particulate concentration.
// Returns the PM2.5 level in µg/m³, or nil if not available.
func (a *Adapter) GetPM25Level(ctx context.Context, nwkAddr uint16, endpoint uint8) (*float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPM25Measurement, zcl.AttrPM25MeasuredValue)
	if err != nil {
		return nil, fmt.Errorf("failed to read PM2.5 level: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return nil, fmt.Errorf("read PM2.5 level returned status 0x%02X", result.Status)
	}

	if val, ok := result.Value.(float32); ok {
		return &val, nil
	}
	return nil, fmt.Errorf("unexpected value type: %T", result.Value)
}

// Sensor Report Handling
// WaitForSensorReport waits for an incoming sensor report from any device.
// Returns when a temperature, humidity, or battery report is received.
// Duplicate messages (same source, cluster, and sequence number within
// the deduplication window) are automatically filtered out.
func (a *Adapter) WaitForSensorReport(ctx context.Context, timeout time.Duration) (*SensorReport, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	dedupe := a.dedupe
	a.mu.Unlock()

	// Track deadline for proper timeout handling across retries
	deadline := time.Now().Add(timeout)

	for {
		// Check if we've exceeded the overall timeout
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("wait for sensor report: timeout")
		}

		// Wait for any incoming message (srcAddr=0, clusterID=0 matches all)
		msg, err := znpClient.WaitForIncomingMsg(ctx, 0, 0, remaining)
		if err != nil {
			return nil, err
		}

		report := &SensorReport{
			NwkAddr: msg.SrcAddr,
		}

		// Parse ZCL frame
		frame, err := zcl.ParseFrame(msg.Data)
		if err != nil {
			// Store error for debugging
			report.ParseError = fmt.Sprintf("parsing ZCL frame: %v", err)
			return report, nil
		}

		// Store frame info for debugging
		report.FrameCommandID = frame.CommandID

		// Check for duplicates using (srcAddr, clusterID, transSeqNum)
		// This filters out mesh network retransmissions
		if dedupe != nil && dedupe.isDuplicate(msg.SrcAddr, msg.ClusterID, frame.TransSeqNum) {
			// Duplicate message, skip and wait for next
			continue
		}

		// Only handle Report Attributes command
		if frame.CommandID != uint8(zcl.CmdReportAttributes) {
			return report, nil
		}

		results, err := zcl.ParseReportAttributesPayload(frame.Payload)
		if err != nil {
			report.ParseError = fmt.Sprintf("parsing attributes: %v", err)
			return report, nil
		}

		// Store cluster and endpoint for debugging
		report.ClusterID = msg.ClusterID
		report.Endpoint = msg.SrcEndpoint

		// Store raw attributes for debugging
		for _, r := range results {
			report.Attributes = append(report.Attributes, ReportedAttribute{
				ID:    uint16(r.AttributeID),
				Value: r.Value,
			})
		}

		// Check cluster ID and parse accordingly
		switch zcl.ClusterID(msg.ClusterID) {
		case zcl.ClusterOnOff:
			for _, r := range results {
				if r.AttributeID == zcl.AttrOnOff {
					var state bool
					switch v := r.Value.(type) {
					case bool:
						state = v
					case uint8:
						state = v != 0
					}
					report.OnOff = &state
				}
			}

		case zcl.ClusterTempMeasurement:
			for _, r := range results {
				if r.AttributeID == zcl.AttrTempMeasuredValue {
					if raw, ok := r.Value.(int16); ok {
						temp := float32(raw) / 100.0
						report.Temperature = &temp
					}
				}
			}

		case zcl.ClusterHumidityMeas:
			for _, r := range results {
				if r.AttributeID == zcl.AttrHumidityMeasuredValue {
					if raw, ok := r.Value.(uint16); ok {
						hum := float32(raw) / 100.0
						report.Humidity = &hum
					}
				}
			}

		case zcl.ClusterPowerConfig:
			for _, r := range results {
				if r.AttributeID == zcl.AttrPowerBatteryPercentage {
					if raw, ok := r.Value.(uint8); ok {
						batt := raw / 2
						report.Battery = &batt
					}
				}
			}

		case zcl.ClusterElectricalMeas:
			// Parse power consumption attributes
			for _, r := range results {
				switch r.AttributeID {
				case zcl.AttrElecRMSVoltage:
					if raw, ok := toUint16(r.Value); ok {
						v := float64(raw) / 1.0 // Tuya reports actual voltage
						report.Voltage = &v
					}
				case zcl.AttrElecRMSCurrent:
					if raw, ok := toUint16(r.Value); ok {
						c := float64(raw) / 1000.0 // Tuya reports current in mA
						report.Current = &c
					}
				case zcl.AttrElecActivePower:
					if raw, ok := toInt16(r.Value); ok {
						p := float64(raw) / 1.0 // Tuya reports actual power
						report.Power = &p
					}
				}
			}

		case zcl.ClusterMeteringSimple:
			// Parse energy metering attributes
			for _, r := range results {
				if r.AttributeID == zcl.AttrMeterCurrentSummation {
					if raw, ok := toUint48(r.Value); ok {
						// Tuya reports energy in Wh, convert to kWh
						e := float64(raw) / 1000.0
						report.Energy = &e
					}
				}
			}
		}

		return report, nil
	}
}

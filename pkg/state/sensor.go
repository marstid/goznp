package state

import (
	"time"

	"github.com/marstid/goznp/pkg/adapter"
)

// SensorReading represents a single sensor reading with timestamps.
type SensorReading[T any] struct {
	Value      T         `json:"value"`
	Timestamp  time.Time `json:"timestamp"`   // When the reading was taken (device time)
	ReceivedAt time.Time `json:"received_at"` // When we received the reading
}

// DeviceSensors stores all sensor readings for a device.
type DeviceSensors struct {
	IEEEAddr    [8]byte
	LastUpdated time.Time

	// Environmental
	Temperature *SensorReading[float32]
	Humidity    *SensorReading[float32]
	Pressure    *SensorReading[float64]

	// Air Quality
	CO           *SensorReading[float32] // Carbon monoxide (ppm)
	CO2          *SensorReading[float32] // Carbon dioxide (ppm)
	PM25         *SensorReading[float32] // PM2.5 (µg/m³)
	Formaldehyde *SensorReading[float32] // Formaldehyde (ppm)

	// Power (smart plugs, etc.)
	Voltage *SensorReading[float64] // RMS Voltage (V)
	Current *SensorReading[float64] // RMS Current (A)
	Power   *SensorReading[float64] // Active Power (W)
	Energy  *SensorReading[float64] // Total Energy (kWh)

	// Battery
	Battery *SensorReading[uint8] // Battery level (%)

	// Illuminance
	Illuminance *SensorReading[float32] // Lux

	// Occupancy
	Occupied *SensorReading[bool]
}

// SensorCache stores the latest sensor readings for all devices.
type SensorCache struct {
	sensors map[[8]byte]*DeviceSensors
}

// NewSensorCache creates a new sensor cache.
func NewSensorCache() *SensorCache {
	return &SensorCache{
		sensors: make(map[[8]byte]*DeviceSensors),
	}
}

// GetOrCreate gets or creates a DeviceSensors entry.
func (sc *SensorCache) GetOrCreate(ieeeAddr [8]byte) *DeviceSensors {
	if _, ok := sc.sensors[ieeeAddr]; !ok {
		sc.sensors[ieeeAddr] = &DeviceSensors{
			IEEEAddr: ieeeAddr,
		}
	}
	return sc.sensors[ieeeAddr]
}

// GetLatest returns the latest sensor readings for a device.
// Returns nil if device has no sensor data.
func (sc *SensorCache) GetLatest(ieeeAddr [8]byte) *DeviceSensors {
	if devSensors, ok := sc.sensors[ieeeAddr]; ok {
		return devSensors
	}
	return nil
}

// RemoveDevice removes all sensor data for a device.
func (sc *SensorCache) RemoveDevice(ieeeAddr [8]byte) {
	delete(sc.sensors, ieeeAddr)
}

// Update updates sensor data from a sensor report.
func (sc *SensorCache) Update(ieeeAddr [8]byte, report *adapter.SensorReport) error {
	devSensors := sc.GetOrCreate(ieeeAddr)
	defaultTimestamp := time.Now()
	defaultReceivedAt := time.Now()

	if report.Temperature != nil {
		devSensors.Temperature = &SensorReading[float32]{
			Value:      *report.Temperature,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.Humidity != nil {
		devSensors.Humidity = &SensorReading[float32]{
			Value:      *report.Humidity,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.Voltage != nil {
		devSensors.Voltage = &SensorReading[float64]{
			Value:      *report.Voltage,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.Current != nil {
		devSensors.Current = &SensorReading[float64]{
			Value:      *report.Current,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.Power != nil {
		devSensors.Power = &SensorReading[float64]{
			Value:      *report.Power,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.Energy != nil {
		devSensors.Energy = &SensorReading[float64]{
			Value:      *report.Energy,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.Battery != nil {
		devSensors.Battery = &SensorReading[uint8]{
			Value:      *report.Battery,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	if report.OnOff != nil {
		devSensors.Occupied = &SensorReading[bool]{
			Value:      *report.OnOff,
			Timestamp:  defaultTimestamp,
			ReceivedAt: defaultReceivedAt,
		}
	}

	devSensors.LastUpdated = defaultReceivedAt
	return nil
}

// UpdateFromAttributes updates sensor data from raw attribute reports.
// This handles parsing of individual attributes for various sensor clusters.
func (sc *SensorCache) UpdateFromAttributes(ieeeAddr [8]byte, clusterID uint16, attrs []adapter.ReportedAttribute) error {
	devSensors := sc.GetOrCreate(ieeeAddr)
	now := time.Now()

	for _, attr := range attrs {
		value := attr.Value

		// Temperature (0x0402)
		if clusterID == 0x0402 && attr.ID == 0x0000 {
			if temp, ok := toFloat32(value); ok {
				devSensors.Temperature = &SensorReading[float32]{
					Value:      temp,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// Humidity (0x0405)
		if clusterID == 0x0405 && attr.ID == 0x0000 {
			if hum, ok := toFloat32(value); ok {
				devSensors.Humidity = &SensorReading[float32]{
					Value:      hum,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// Pressure (0x0403)
		if clusterID == 0x0403 && attr.ID == 0x0000 {
			if press, ok := toFloat64(value); ok {
				devSensors.Pressure = &SensorReading[float64]{
					Value:      press,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// Illuminance (0x0400)
		if clusterID == 0x0400 && attr.ID == 0x0000 {
			if lux, ok := toFloat32(value); ok {
				devSensors.Illuminance = &SensorReading[float32]{
					Value:      lux,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// Occupancy (0x0406)
		if clusterID == 0x0406 && attr.ID == 0x0000 {
			if occ, ok := toBool(value); ok {
				devSensors.Occupied = &SensorReading[bool]{
					Value:      occ,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// Power Configuration - Battery (0x0001)
		if clusterID == 0x0001 && attr.ID == 0x0021 {
			if batt, ok := toUint8(value); ok {
				devSensors.Battery = &SensorReading[uint8]{
					Value:      batt,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// Electrical Measurement (0x0B04)
		if clusterID == 0x0B04 {
			if attr.ID == 0x0505 { // RMS Voltage
				if volt, ok := toFloat64(value); ok {
					devSensors.Voltage = &SensorReading[float64]{
						Value:      volt,
						Timestamp:  now,
						ReceivedAt: now,
					}
				}
			}
			if attr.ID == 0x0508 { // RMS Current
				if curr, ok := toFloat64(value); ok {
					devSensors.Current = &SensorReading[float64]{
						Value:      curr,
						Timestamp:  now,
						ReceivedAt: now,
					}
				}
			}
			if attr.ID == 0x050B { // Active Power
				if powr, ok := toFloat64(value); ok {
					devSensors.Power = &SensorReading[float64]{
						Value:      powr,
						Timestamp:  now,
						ReceivedAt: now,
					}
				}
			}
		}

		// Simple Metering (0x0702)
		if clusterID == 0x0702 && attr.ID == 0x0000 {
			// Current summation (energy)
			if energy, ok := toFloat64(value); ok {
				devSensors.Energy = &SensorReading[float64]{
					Value:      energy / 1000.0, // Convert to kWh
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}

		// OnOff (0x0006) - used for basic state tracking
		if clusterID == 0x0006 && attr.ID == 0x0000 {
			if onOff, ok := toBool(value); ok {
				devSensors.Occupied = &SensorReading[bool]{
					Value:      onOff,
					Timestamp:  now,
					ReceivedAt: now,
				}
			}
		}
	}

	devSensors.LastUpdated = now
	return nil
}

// Helper type conversion functions

func toFloat32(v interface{}) (float32, bool) {
	switch val := v.(type) {
	case float32:
		return val, true
	case float64:
		return float32(val), true
	case int:
		return float32(val), true
	case int16:
		return float32(val), true
	case uint16:
		return float32(val), true
	default:
		return 0, false
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int16:
		return float64(val), true
	case uint16:
		return float64(val), true
	case int32:
		return float64(val), true
	case uint32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

func toUint8(v interface{}) (uint8, bool) {
	switch val := v.(type) {
	case uint8:
		return val, true
	case uint16:
		if val <= 255 {
			return uint8(val), true
		}
	case uint32:
		if val <= 255 {
			return uint8(val), true
		}
	case int:
		if val >= 0 && val <= 255 {
			return uint8(val), true
		}
	default:
		return 0, false
	}
	return 0, false
}

func toBool(v interface{}) (bool, bool) {
	switch val := v.(type) {
	case bool:
		return val, true
	case uint8:
		return val != 0, true
	case uint16:
		return val != 0, true
	case int:
		return val != 0, true
	default:
		return false, false
	}
}

// ToTimestamped converts a SensorReading to a TimestampedValue.
func (sr *SensorReading[T]) ToTimestamped() *TimestampedValue[T] {
	if sr == nil {
		return nil
	}
	return &TimestampedValue[T]{
		Value:      sr.Value,
		Timestamp:  sr.Timestamp,
		ReceivedAt: sr.ReceivedAt,
	}
}

// TimestampedValue represents a value with timestamps.
type TimestampedValue[T any] struct {
	Value      T         `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	ReceivedAt time.Time `json:"received_at"`
}

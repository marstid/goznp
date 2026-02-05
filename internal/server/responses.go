package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/marstid/goznp/pkg/state"
)

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// DeviceListResponse represents the list devices response.
type DeviceListResponse struct {
	Devices []DeviceSummary `json:"devices"`
	Count   int             `json:"count"`
}

// DeviceSummary represents a summary of a device for list views.
type DeviceSummary struct {
	IEEEAddr     string    `json:"ieee_addr"`
	NwkAddr      string    `json:"nwk_addr"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name,omitempty"`
	Model        string    `json:"model,omitempty"`
	Manufacturer string    `json:"manufacturer,omitempty"`
	LastSeen     time.Time `json:"last_seen"`
	JoinedAt     time.Time `json:"joined_at"`
	IsRouter     bool      `json:"is_router"`
	IsBattery    bool      `json:"is_battery_powered"`
}

// DeviceDetailsResponse represents the device details response.
type DeviceDetailsResponse struct {
	IEEEAddr     string              `json:"ieee_addr"`
	NwkAddr      string              `json:"nwk_addr"`
	Slug         string              `json:"slug"`
	Name         string              `json:"name,omitempty"`
	Manufacturer string              `json:"manufacturer,omitempty"`
	Model        string              `json:"model,omitempty"`
	Capabilities string              `json:"capabilities,omitempty"`
	Endpoints    []uint8             `json:"endpoints"`
	LastSeen     time.Time           `json:"last_seen"`
	JoinedAt     time.Time           `json:"joined_at"`
	State        DeviceStateResponse `json:"state"`
}

// DeviceStateResponse represents the cached device state.
type DeviceStateResponse struct {
	OnOff       *bool     `json:"on_off,omitempty"`
	Brightness  *uint8    `json:"brightness,omitempty"`
	ColorTemp   *uint16   `json:"color_temp,omitempty"`
	ColorHue    *uint8    `json:"color_hue,omitempty"`
	ColorSat    *uint8    `json:"color_saturation,omitempty"`
	Temperature *float32  `json:"temperature,omitempty"`
	Humidity    *float32  `json:"humidity,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// SensorDataResponse represents the sensor data response.
type SensorDataResponse struct {
	IEEEAddr    string        `json:"ieee_addr"`
	Slug        string        `json:"slug,omitempty"`
	LastUpdated time.Time     `json:"last_updated"`
	Reading     SensorReading `json:"reading"`
}

// SensorReading represents sensor readings.
type SensorReading struct {
	Temperature *state.TimestampedValue[float32] `json:"temperature,omitempty"`
	Humidity    *state.TimestampedValue[float32] `json:"humidity,omitempty"`
	Pressure    *state.TimestampedValue[float64] `json:"pressure,omitempty"`
	Voltage     *state.TimestampedValue[float64] `json:"voltage,omitempty"`
	Current     *state.TimestampedValue[float64] `json:"current,omitempty"`
	Power       *state.TimestampedValue[float64] `json:"power,omitempty"`
	Energy      *state.TimestampedValue[float64] `json:"energy,omitempty"`
	Battery     *state.TimestampedValue[uint8]   `json:"battery,omitempty"`
	Illuminance *state.TimestampedValue[float32] `json:"illuminance,omitempty"`
	Occupied    *state.TimestampedValue[bool]    `json:"occupied,omitempty"`
}

// TimestampedValue represents a value with timestamps.
type TimestampedValue[T any] struct {
	Value      T         `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	ReceivedAt time.Time `json:"received_at"`
}

// ControlResponse represents a device control response.
type ControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// DeviceNameResponse represents the device name response.
type DeviceNameResponse struct {
	Success bool   `json:"success"`
	Slug    string `json:"slug,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data == nil {
		w.Write([]byte("{}"))
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes an error response.
func writeErrorResponse(w http.ResponseWriter, code int, err error) {
	response := ErrorResponse{
		Error: http.StatusText(code),
	}
	if err != nil {
		response.Message = err.Error()
	}
	WriteJSON(w, code, response)
}

// deviceStateToResponse converts a device state to API response.
func deviceStateToResponse(dev *state.DeviceState) DeviceStateResponse {
	r := DeviceStateResponse{
		LastUpdated: dev.LastUpdated,
	}

	// OnOff state (cluster 0x0006, attr 0x0000)
	if val := dev.GetOnOffState(); val != nil {
		r.OnOff = val
	}

	// Brightness (cluster 0x0008, attr 0x0000)
	if val := dev.GetBrightness(); val != nil {
		r.Brightness = val
	}

	return r
}

// deviceSummaryFromState converts a device state to summary.
func deviceSummaryFromState(dev *state.DeviceState) DeviceSummary {
	return DeviceSummary{
		IEEEAddr:     state.FormatIEEEAddr(dev.IEEEAddr),
		NwkAddr:      fmt.Sprintf("0x%04X", dev.NwkAddr),
		Slug:         dev.Slug(),
		Name:         dev.Name,
		Model:        dev.Model,
		Manufacturer: dev.Manufacturer,
		LastSeen:     dev.LastSeen,
		JoinedAt:     dev.JoinedAt,
		IsRouter:     dev.IsRouter(),
		IsBattery:    dev.IsBatteryPowered(),
	}
}

// sensorDataToResponse converts sensor readings to API response.
func sensorDataToResponse(sensors *state.DeviceSensors, slug string) SensorDataResponse {
	return SensorDataResponse{
		IEEEAddr:    state.FormatIEEEAddr(sensors.IEEEAddr),
		Slug:        slug,
		LastUpdated: sensors.LastUpdated,
		Reading: SensorReading{
			Temperature: sensors.Temperature.ToTimestamped(),
			Humidity:    sensors.Humidity.ToTimestamped(),
			Pressure:    sensors.Pressure.ToTimestamped(),
			Voltage:     sensors.Voltage.ToTimestamped(),
			Current:     sensors.Current.ToTimestamped(),
			Power:       sensors.Power.ToTimestamped(),
			Energy:      sensors.Energy.ToTimestamped(),
			Battery:     sensors.Battery.ToTimestamped(),
			Illuminance: sensors.Illuminance.ToTimestamped(),
			Occupied:    sensors.Occupied.ToTimestamped(),
		},
	}
}

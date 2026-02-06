package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/state"
	"github.com/marstid/goznp/pkg/zcl"
)

// Handlers contains all HTTP handlers for the daemon API.
type Handlers struct {
	manager *state.Manager
	adapter *adapter.Adapter
	logger  *slog.Logger
}

// NewHandlers creates a new handlers instance.
func NewHandlers(manager *state.Manager, adapter *adapter.Adapter, logger *slog.Logger) *Handlers {
	return &Handlers{
		manager: manager,
		adapter: adapter,
		logger:  logger,
	}
}

// HealthCheck returns the health status of the daemon.
// GET /health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := h.adapter.Ping(ctx)
	if err != nil {
		h.logger.Warn("Health check failed", "error", err)
		writeErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
	}
	WriteJSON(w, http.StatusOK, response)
}

// ListDevices returns all devices.
// GET /api/v1/devices
func (h *Handlers) ListDevices(w http.ResponseWriter, _ *http.Request) {
	devices := h.manager.GetDevices()

	response := DeviceListResponse{
		Devices: make([]DeviceSummary, len(devices)),
		Count:   len(devices),
	}

	for i, dev := range devices {
		response.Devices[i] = deviceSummaryFromState(dev)
	}

	WriteJSON(w, http.StatusOK, response)
}

// GetDevice returns details for a specific device.
// GET /api/v1/devices/:identifier (can be slug or IEEE address)
func (h *Handlers) GetDevice(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")
	if identifier == "" {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("identifier is required"))
		return
	}

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		if err == state.ErrDeviceNotFound {
			writeErrorResponse(w, http.StatusNotFound, err)
		} else {
			h.logger.Error("Failed to get device", "identifier", identifier, "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	response := DeviceDetailsResponse{
		IEEEAddr:     state.FormatIEEEAddr(dev.IEEEAddr),
		NwkAddr:      fmt.Sprintf("0x%04X", dev.NwkAddr),
		Slug:         h.manager.GetSlugForIEEE(dev.IEEEAddr),
		Name:         dev.Name,
		Model:        dev.Model,
		Manufacturer: dev.Manufacturer,
		Endpoints:    dev.Endpoints,
		LastSeen:     dev.LastSeen,
		JoinedAt:     dev.JoinedAt,
		State:        deviceStateToResponse(dev),
	}

	WriteJSON(w, http.StatusOK, response)
}

// GetDeviceState returns the cached state for a specific device.
// GET /api/v1/devices/:identifier/state
func (h *Handlers) GetDeviceState(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		if err == state.ErrDeviceNotFound {
			writeErrorResponse(w, http.StatusNotFound, err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	response := deviceStateToResponse(dev)
	WriteJSON(w, http.StatusOK, response)
}

// GetSensorData returns the latest sensor readings for a device.
// GET /api/v1/devices/:identifier/sensor
func (h *Handlers) GetSensorData(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		if err == state.ErrDeviceNotFound {
			writeErrorResponse(w, http.StatusNotFound, err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	sensors := h.manager.GetSensorData(dev.IEEEAddr)
	if sensors == nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Errorf("no sensor data available"))
		return
	}

	response := sensorDataToResponse(sensors, h.manager.GetSlugForIEEE(dev.IEEEAddr))
	WriteJSON(w, http.StatusOK, response)
}

// SetDeviceName sets a friendly name for a device.
// POST /api/v1/devices/:identifier/name
// Request body: {"name": "Device Name"}
func (h *Handlers) SetDeviceName(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	slug, err := h.manager.SetDeviceNameByIdentifier(identifier, req.Name)
	if err != nil {
		switch err {
		case state.ErrDeviceNotFound:
			writeErrorResponse(w, http.StatusNotFound, err)
		case state.ErrNameInUse:
			writeErrorResponse(w, http.StatusConflict, err)
		default:
			writeErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	response := DeviceNameResponse{
		Success: true,
		Slug:    slug,
	}
	WriteJSON(w, http.StatusOK, response)
}

// TurnOn turns a device on.
// POST /api/v1/devices/:identifier/on
func (h *Handlers) TurnOn(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	endpoint := h.getEndpointForCluster(dev, zcl.ClusterOnOff)
	if endpoint == 0 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("device doesn't support OnOff cluster"))
		return
	}

	if err := h.adapter.TurnOn(ctx, dev.NwkAddr, endpoint); err != nil {
		h.logger.Error("Failed to turn on device",
			"identifier", identifier,
			"nwkAddr", dev.NwkAddr,
			"error", err)
		writeErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := ControlResponse{Success: true}
	WriteJSON(w, http.StatusOK, response)
}

// TurnOff turns a device off.
// POST /api/v1/devices/:identifier/off
func (h *Handlers) TurnOff(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	endpoint := h.getEndpointForCluster(dev, zcl.ClusterOnOff)
	if endpoint == 0 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("device doesn't support OnOff cluster"))
		return
	}

	if err := h.adapter.TurnOff(ctx, dev.NwkAddr, endpoint); err != nil {
		h.logger.Error("Failed to turn off device",
			"identifier", identifier,
			"nwkAddr", dev.NwkAddr,
			"error", err)
		writeErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := ControlResponse{Success: true}
	WriteJSON(w, http.StatusOK, response)
}

// Toggle toggles a device.
// POST /api/v1/devices/:identifier/toggle
func (h *Handlers) Toggle(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	endpoint := h.getEndpointForCluster(dev, zcl.ClusterOnOff)
	if endpoint == 0 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("device doesn't support OnOff cluster"))
		return
	}

	if err := h.adapter.Toggle(ctx, dev.NwkAddr, endpoint); err != nil {
		h.logger.Error("Failed to toggle device",
			"identifier", identifier,
			"nwkAddr", dev.NwkAddr,
			"error", err)
		writeErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := ControlResponse{Success: true}
	WriteJSON(w, http.StatusOK, response)
}

// SetBrightness sets the brightness level of a device.
// POST /api/v1/devices/:identifier/brightness
// Request body: {"level": 128} (0-254)
func (h *Handlers) SetBrightness(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err)
		return
	}

	var req struct {
		Level      uint8  `json:"level"`
		Transition uint16 `json:"transition,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	if req.Level > 254 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("brightness level must be 0-254"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	endpoint := h.getEndpointForCluster(dev, zcl.ClusterLevelControl)
	if endpoint == 0 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("device doesn't support LevelControl cluster"))
		return
	}

	if err := h.adapter.SetBrightness(ctx, dev.NwkAddr, endpoint, req.Level, req.Transition); err != nil {
		h.logger.Error("Failed to set brightness",
			"identifier", identifier,
			"nwkAddr", dev.NwkAddr,
			"level", req.Level,
			"error", err)
		writeErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := ControlResponse{Success: true}
	WriteJSON(w, http.StatusOK, response)
}

// SetColorTemperature sets the color temperature.
// POST /api/v1/devices/:identifier/color-temp
// Request body: {"kelvin": 2700} (2700-6500)
func (h *Handlers) SetColorTemperature(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err)
		return
	}

	var req struct {
		Kelvin uint16 `json:"kelvin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	if req.Kelvin < 2700 || req.Kelvin > 6500 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("color temperature must be 2700-6500 Kelvin"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	endpoint := h.getEndpointForCluster(dev, zcl.ClusterColorControl)
	if endpoint == 0 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("device doesn't support ColorControl cluster"))
		return
	}

	if err := h.adapter.SetColorKelvin(ctx, dev.NwkAddr, endpoint, req.Kelvin, 0); err != nil {
		h.logger.Error("Failed to set color temperature",
			"identifier", identifier,
			"nwkAddr", dev.NwkAddr,
			"kelvin", req.Kelvin,
			"error", err)
		writeErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := ControlResponse{Success: true}
	WriteJSON(w, http.StatusOK, response)
}

// Identify makes a device identify itself for a specific duration.
// POST /api/v1/devices/:identifier/identify
// Request body: {"time": 5} (duration in seconds, default: 5)
func (h *Handlers) Identify(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")

	dev, err := h.manager.GetDeviceByIdentifier(identifier)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err)
		return
	}

	var req struct {
		Time uint16 `json:"time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	duration := req.Time
	if duration == 0 {
		duration = 5 // Default 5 seconds
	}
	if duration > 255 {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("duration must be 0-255 seconds"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	endpoint := h.getEndpointForCluster(dev, zcl.ClusterIdentify)
	if endpoint == 0 {
		endpoint = 1 // Identify cluster is usually on endpoint 1
	}

	if err := h.adapter.Identify(ctx, dev.NwkAddr, endpoint, duration); err != nil {
		h.logger.Error("Failed to identify device",
			"identifier", identifier,
			"nwkAddr", dev.NwkAddr,
			"duration", duration,
			"error", err)
		writeErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := ControlResponse{Success: true}
	WriteJSON(w, http.StatusOK, response)
}

// Helper methods

// getEndpointForCluster returns the first endpoint that supports the given cluster.
func (h *Handlers) getEndpointForCluster(dev *state.DeviceState, _ zcl.ClusterID) uint8 {
	// For now, just return endpoint 1 if it exists
	// A real implementation would check interview data
	for _, ep := range dev.Endpoints {
		if ep == 1 || ep == 11 {
			return ep
		}
	}
	return 0
}

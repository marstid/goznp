package server

import (
	"net/http"

	"log/slog"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/state"
)

// SetupRoutes configures and returns the HTTP router with all handlers registered.
func SetupRoutes(mux *http.ServeMux, manager *state.Manager, adapter *adapter.Adapter, logger *slog.Logger) {
	handlers := NewHandlers(manager, adapter, logger)

	// Health check
	mux.HandleFunc("GET /health", handlers.HealthCheck)

	// Devices
	mux.HandleFunc("GET /api/v1/devices", handlers.ListDevices)
	mux.HandleFunc("GET /api/v1/devices/", GetDevice(handlers))
	mux.HandleFunc("GET /api/v1/devices/{identifier}", handlers.GetDevice)
	mux.HandleFunc("GET /api/v1/devices/{identifier}/state", handlers.GetDeviceState)
	mux.HandleFunc("GET /api/v1/devices/{identifier}/sensor", handlers.GetSensorData)
	mux.HandleFunc("POST /api/v1/devices/{identifier}/name", handlers.SetDeviceName)

	// Device control
	mux.HandleFunc("POST /api/v1/devices/{identifier}/on", handlers.TurnOn)
	mux.HandleFunc("POST /api/v1/devices/{identifier}/off", handlers.TurnOff)
	mux.HandleFunc("POST /api/v1/devices/{identifier}/toggle", handlers.Toggle)
	mux.HandleFunc("POST /api/v1/devices/{identifier}/brightness", handlers.SetBrightness)
	mux.HandleFunc("POST /api/v1/devices/{identifier}/color-temp", handlers.SetColorTemperature)
	mux.HandleFunc("POST /api/v1/devices/{identifier}/identify", handlers.Identify)
}

// GetDevice is a wrapper handler that serves as a redirect from the base path.
// This handles requests like GET /api/v1/devices/ without an identifier.
func GetDevice(handlers *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If the request is just /api/v1/devices/, redirect to list
		if r.URL.Path == "/api/v1/devices/" || r.URL.Path == "/api/v1/devices" {
			http.Redirect(w, r, "/api/v1/devices", http.StatusPermanentRedirect)
			return
		}
		// Otherwise, try to parse as device identifier
		handlers.GetDevice(w, r)
	}
}

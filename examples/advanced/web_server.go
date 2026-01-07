//go:build ignore

// Example: Simple HTTP server for controlling Zigbee devices.
// This demonstrates building a REST API on top of goznp.
// Note: You'll need to import a web framework. This example uses the standard
// library's net/http with JSON responses to keep dependencies minimal.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

var (
	adptr *adapter.Adapter
	ctx   context.Context
)

func main() {
	// Get config from environment
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Create adapter
	adptr = adapter.New(adapter.WithSerialPath(port))
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}

	// Ensure cleanup on exit
	defer func() {
		cancel()
		adptr.Close()
	}()

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/devices", handleDevices)
	mux.HandleFunc("/api/devices/", handleDevice)
	mux.HandleFunc("/api/network", handleNetwork)
	mux.HandleFunc("/api/groups", handleGroups)
	mux.HandleFunc("/api/groups/", handleGroup)

	// Enable CORS (simple version)
	mux.HandleFunc("/", withCORS(mux))

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		fmt.Printf("=== goznp HTTP Server ===\n")
		fmt.Printf("Serial: %s\n", port)
		fmt.Printf("HTTP: http://localhost:%s\n\n", httpPort)
		fmt.Println("API Endpoints:")
		fmt.Println("  GET    /api/info       - Adapter and network info")
		fmt.Println("  GET    /api/devices    - List all devices")
		fmt.Println("  GET    /api/devices/:addr - Device details")
		fmt.Println("  POST   /api/devices/:addr - Control device (body: {action, value})")
		fmt.Println("  GET    /api/network    - Network status")
		fmt.Println("  GET    /api/groups     - List groups")
		fmt.Println("  POST   /api/groups/:id - Control group (body: {action, value})")
		fmt.Println("\nServer running... Press Ctrl+C to stop\n")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nShutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)
	wg.Wait()

	fmt.Println("Server stopped")
}

// withCORS wraps handlers to add CORS headers
func withCORS(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// API Response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, err string) {
	writeJSON(w, status, APIResponse{Success: false, Error: err})
}

func writeSuccess(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// handleInfo returns adapter and network information
func handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	info, err := adptr.GetInfo(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	nwkInfo, err := adptr.GetNetworkInfo(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := struct {
		Adapter struct {
			Transport float64 `json:"transport_rev"`
			Product   int     `json:"product_id"`
			Major     int     `json:"major_rel"`
			Minor     int     `json:"minor_rel"`
			Maint     int     `json:"maint_rel"`
		} `json:"adapter"`
		Network struct {
			ShortAddr  string `json:"short_addr"`
			PanID      uint16 `json:"pan_id"`
			ExtPanID   string `json:"ext_pan_id"`
			Channel    uint8  `json:"channel"`
			DevState   string `json:"device_state"`
			ParentAddr uint16 `json:"parent_addr"`
		} `json:"network"`
	}{
		Adapter: struct {
			Transport float64 `json:"transport_rev"`
			Product   int     `json:"product_id"`
			Major     int     `json:"major_rel"`
			Minor     int     `json:"minor_rel"`
			Maint     int     `json:"maint_rel"`
		}{
			Transport: info.Version.TransportRev,
			Product:   int(info.Version.Product),
			Major:     int(info.Version.MajorRel),
			Minor:     int(info.Version.MinorRel),
			Maint:     int(info.Version.MaintRel),
		},
		Network: struct {
			ShortAddr  string `json:"short_addr"`
			PanID      uint16 `json:"pan_id"`
			ExtPanID   string `json:"ext_pan_id"`
			Channel    uint8  `json:"channel"`
			DevState   string `json:"device_state"`
			ParentAddr uint16 `json:"parent_addr"`
		}{
			ShortAddr:  fmt.Sprintf("0x%04X", nwkInfo.ShortAddr),
			PanID:      nwkInfo.PanID,
			ExtPanID:   fmt.Sprintf("%016X", nwkInfo.ExtendedPanID),
			Channel:    nwkInfo.Channel,
			DevState:   deviceStateString(nwkInfo.DevState),
			ParentAddr: nwkInfo.ParentAddr,
		},
	}

	writeSuccess(w, data)
}

// handleDevices returns list of all devices
func handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var deviceList []map[string]any
	for _, dev := range devices {
		d := map[string]any{
			"ieee_addr":    formatIEEEAddr(dev.IEEEAddr),
			"nwk_addr":     dev.NwkAddr,
			"manufacturer": dev.Manufacturer,
			"model":        dev.Model,
			"power_source": powerSourceString(dev.Capabilities),
			"endpoints":    len(dev.Endpoints),
			"last_seen":    dev.LastSeen.Unix(),
		}

		// Add device name if available
		if name, _ := adptr.GetDeviceName(ctx, dev.IEEEAddr); name != nil {
			d["name"] = name.FriendlyName
		}

		deviceList = append(deviceList, d)
	}

	writeSuccess(w, deviceList)
}

// handleDevice handles device details and control
func handleDevice(w http.ResponseWriter, r *http.Request) {
	// Parse device address from path (e.g., /api/devices/0x1234)
	addr := r.URL.Path[len("/api/devices/"):]
	var nwkAddr uint16
	if len(addr) > 2 && addr[0:2] == "0x" {
		fmt.Sscanf(addr, "0x%04X", &nwkAddr)
	} else {
		fmt.Sscanf(addr, "%d", &nwkAddr)
	}

	if r.Method == http.MethodGet {
		handleDeviceGet(w, nwkAddr)
	} else if r.Method == http.MethodPost {
		handleDevicePost(w, r, nwkAddr)
	} else {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleDeviceGet(w http.ResponseWriter, nwkAddr uint16) {
	devices, _ := adptr.GetDevices(ctx)
	var target *adapter.Device
	for _, d := range devices {
		if d.NwkAddr == nwkAddr {
			target = d
			break
		}
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	data := make(map[string]any)
	data["ieee_addr"] = formatIEEEAddr(target.IEEEAddr)
	data["nwk_addr"] = target.NwkAddr
	data["manufacturer"] = target.Manufacturer
	data["model"] = target.Model
	data["power_source"] = powerSourceString(target.Capabilities)
	data["last_seen"] = target.LastSeen.Unix()

	// Add device name
	if name, _ := adptr.GetDeviceName(ctx, target.IEEEAddr); name != nil {
		data["name"] = name.FriendlyName
		data["description"] = name.Description
	}

	// Add endpoints with clusters
	var endpoints []map[string]any
	for _, ep := range target.Endpoints {
		epInfo := map[string]any{
			"id":       ep.ID,
			"profile":  ep.ProfileID,
			"device":   ep.DeviceID,
			"in_clust": ep.InClusters,
			"out_clust": ep.OutClusters,
		}
		endpoints = append(endpoints, epInfo)
	}
	data["endpoints"] = endpoints

	writeSuccess(w, data)
}

func handleDevicePost(w http.ResponseWriter, r *http.Request, nwkAddr uint16) {
	var req struct {
		Action string `json:"action"`
		Value  any    `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default to endpoint 1
	endpoint := uint8(1)
	if epVal := r.URL.Query().Get("endpoint"); epVal != "" {
		fmt.Sscanf(epVal, "%d", &endpoint)
	}

	switch req.Action {
	case "on":
		if err := adptr.TurnOn(ctx, nwkAddr, endpoint); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
		} else {
			writeSuccess(w, map[string]string{"status": "turned on"})
		}

	case "off":
		if err := adptr.TurnOff(ctx, nwkAddr, endpoint); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
		} else {
			writeSuccess(w, map[string]string{"status": "turned off"})
		}

	case "toggle":
		if err := adptr.Toggle(ctx, nwkAddr, endpoint); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
		} else {
			writeSuccess(w, map[string]string{"status": "toggled"})
		}

	case "brightness":
		level := uint8(254)
		if val, ok := req.Value.(float64); ok {
			level = uint8(val)
		}
		if err := adptr.SetBrightness(ctx, nwkAddr, endpoint, level); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
		} else {
			writeSuccess(w, map[string]uint8{"brightness": level})
		}

	case "identify":
		duration := uint16(5)
		if val, ok := req.Value.(float64); ok {
			duration = uint16(val)
		}
		if err := adptr.Identify(ctx, nwkAddr, endpoint, duration); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
		} else {
			writeSuccess(w, map[string]uint16{"duration": duration})
		}

	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown action: %s", req.Action))
	}
}

// handleNetwork returns network status
func handleNetwork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	info, err := adptr.GetNetworkInfo(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	devices, _ := adptr.GetDevices(ctx)

	data := struct {
		ShortAddr   string `json:"short_addr"`
		PanID       uint16 `json:"pan_id"`
		ExtPanID    string `json:"ext_pan_id"`
		Channel     uint8  `json:"channel"`
		DevState    string `json:"device_state"`
		DeviceCount int    `json:"device_count"`
		Uptime      int64  `json:"uptime_seconds"`
	}{
		ShortAddr:   fmt.Sprintf("0x%04X", info.ShortAddr),
		PanID:       info.PanID,
		ExtPanID:    fmt.Sprintf("%016X", info.ExtendedPanID),
		Channel:     info.Channel,
		DevState:    deviceStateString(info.DevState),
		DeviceCount: len(devices),
		Uptime:      time.Since(time.Now()).Seconds(), // Simplified
	}

	writeSuccess(w, data)
}

// handleGroups returns all groups
func handleGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get coordinator endpoint
	coordEP := adptr.GetEndpointForProfile(0x0104)

	// Get group names (requires reading from Basic cluster)
	groups, err := adptr.GetGroupNames(ctx, coordEP)
	if err != nil {
		// Create empty response if no groups
		writeSuccess(w, []any{})
		return
	}

	var groupList []map[string]any
	for _, g := range groups {
		groupList = append(groupList, map[string]any{
			"id":   g.GroupID,
			"name": g.Name,
		})
	}

	writeSuccess(w, groupList)
}

// handleGroup controls a group
func handleGroup(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/groups/"):]
	var groupID uint16
	fmt.Sscanf(id, "%d", &groupID)

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get coordinator endpoint
	coordEP := adptr.GetEndpointForProfile(0x0104)

	var err error
	switch req.Action {
	case "on":
		err = adptr.GroupOn(ctx, groupID, coordEP)
	case "off":
		err = adptr.GroupOff(ctx, groupID, coordEP)
	case "toggle":
		err = adptr.GroupToggle(ctx, groupID, coordEP)
	case "identify":
		err = adptr.GroupIdentify(ctx, groupID, coordEP, 10)
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown action: %s", req.Action))
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
	} else {
		writeSuccess(w, map[string]string{"status": req.Action + " executed"})
	}
}

func formatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%016X", addr)
}

func deviceStateString(state uint8) string {
	switch state & 0x0F {
	case 9:
		return "connected"
	case 10:
		return "running"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
}

func powerSourceString(caps uint8) string {
	if caps&0x02 != 0 {
		return "mains"
	} else if caps&0x04 != 0 {
		return "battery"
	}
	return "unknown"
}

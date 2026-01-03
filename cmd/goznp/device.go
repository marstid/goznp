package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	devicedb "github.com/marstid/goznp/pkg/devices"
	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
	"github.com/spf13/cobra"
)

var (
	permitDuration   uint8
	pairTimeout      int
	pairNoBind       bool
	deviceAddr       string
	deviceIEEE       string
	deviceEndpoint   uint8
	interviewDevices bool
	deviceForce      bool

	// Brightness/level control flags
	brightnessLevel uint8
	transitionTime  uint16

	// Color temperature flags
	colorTempKelvin uint16
	colorTempMireds uint16

	// Identify flags
	identifyDuration uint16
	identifyEffect   string

	// Group flags
	groupID   uint16
	groupName string

	// Scene flags
	sceneID uint8

	// Device name flags
	deviceName        string
	deviceDescription string
	deviceNameLookup  string // For --name flag
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Device management commands",
	Long:  "Commands for managing paired Zigbee devices",
}

// goznp device permit -p <port> [--duration <seconds>]
var devicePermitCmd = &cobra.Command{
	Use:   "permit",
	Short: "Open network for device pairing",
	Long:  "Open the network for devices to join (default 60 seconds)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDevicePermit(ctx)
	},
}

func runDevicePermit(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	if permitDuration == 0 {
		fmt.Println("Closing network for pairing...")
	} else if permitDuration == 255 {
		fmt.Println("Opening network for pairing (always open)...")
	} else {
		fmt.Printf("Opening network for pairing (%d seconds)...\n", permitDuration)
	}

	if err := a.PermitJoin(ctx, permitDuration); err != nil {
		return fmt.Errorf("failed to permit join: %w", err)
	}

	if permitDuration == 0 {
		fmt.Println("Network closed for pairing")
	} else {
		fmt.Println("Network is open for pairing")
		fmt.Println("Use 'goznp device watch' to monitor for new devices")
	}

	return nil
}

// goznp device pair -p <port> [--timeout <seconds>] [--nobind]
var devicePairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Pair a new device",
	Long: `Open network for pairing, watch for device joins, then close.

This command opens the network for device pairing for the specified timeout
(default 180 seconds), actively monitors for joining devices, and then
automatically closes the network when done.

After pairing, each device is automatically interviewed and its clusters are
bound to the coordinator so it can send reports. Use --nobind to skip binding.

Put your device in pairing mode before running this command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDevicePair(ctx)
	},
}

func runDevicePair(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create timeout context
	timeout := time.Duration(pairTimeout) * time.Second
	pairCtx, cancel := context.WithTimeout(ctx, timeout+60*time.Second) // Extra time for interview/bind
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(pairCtx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Get coordinator IEEE for binding (do this once upfront)
	var coordIEEE [8]byte
	var coordErr error
	if !pairNoBind {
		coordIEEE, coordErr = a.GetCoordinatorIEEE(pairCtx)
		if coordErr != nil {
			fmt.Printf("Warning: Could not get coordinator IEEE address: %v\n", coordErr)
			fmt.Println("Auto-bind will be skipped")
		}
	}

	// Track results
	var resultsMu sync.Mutex
	type pairResult struct {
		device      *adapter.Device
		interviewed bool
		bound       int
		errors      []string
	}
	results := make([]pairResult, 0)

	// Set up device event handler - interview and bind IMMEDIATELY on join
	a.OnDeviceEvent(func(event adapter.DeviceEvent) {
		if event.Type != adapter.DeviceEventJoined || event.Device == nil {
			return
		}

		dev := event.Device
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("\n[%s] Device JOINED: %s (0x%04X)\n",
			timestamp,
			znp.FormatIEEEAddr(dev.IEEEAddr),
			dev.NwkAddr,
		)

		result := pairResult{device: dev}

		// IMMEDIATELY interview while device is still awake
		fmt.Printf("  Interviewing (device must stay awake)...\n")
		interviewCtx, interviewCancel := context.WithTimeout(context.Background(), 15*time.Second)
		interview, err := a.InterviewDeviceWithAddr(interviewCtx, dev.NwkAddr, dev.IEEEAddr)
		interviewCancel()

		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("interview: %v", err))
			fmt.Printf("  Interview FAILED: %v\n", err)
		} else if !interview.Success || len(interview.Endpoints) == 0 {
			result.errors = append(result.errors, "no endpoints discovered")
			fmt.Printf("  Interview incomplete (no endpoints)\n")
		} else {
			result.interviewed = true
			if interview.Manufacturer != "" || interview.Model != "" {
				fmt.Printf("  Device: %s %s (%s)\n", interview.Manufacturer, interview.Model, interview.PowerSource)
			}

			// IMMEDIATELY bind while device is still awake
			if !pairNoBind && coordErr == nil {
				for _, ep := range interview.Endpoints {
					for _, clusterID := range ep.InClusters {
						if !shouldBindCluster(clusterID) {
							continue
						}

						bindCtx, bindCancel := context.WithTimeout(context.Background(), 5*time.Second)
						err := a.Bind(bindCtx, dev.IEEEAddr, dev.NwkAddr, ep.Endpoint, clusterID, coordIEEE, 1)
						bindCancel()

						clusterName := zcl.ClusterID(clusterID).String()
						if err != nil {
							result.errors = append(result.errors, fmt.Sprintf("bind %s: %v", clusterName, err))
							fmt.Printf("  Bind %s: FAILED\n", clusterName)
						} else {
							result.bound++
							fmt.Printf("  Bind %s: OK\n", clusterName)
						}
					}
				}
			}
		}

		resultsMu.Lock()
		results = append(results, result)
		resultsMu.Unlock()

		fmt.Printf("  Done! Waiting for more devices...\n")
	})

	// Open permit join
	fmt.Printf("Opening network for pairing (%d seconds)...\n", pairTimeout)
	fmt.Println("Waiting for devices to join... (Ctrl+C to stop early)")
	if pairNoBind {
		fmt.Println("Auto-bind: DISABLED (--nobind)")
	} else if coordErr != nil {
		fmt.Println("Auto-bind: DISABLED (coordinator error)")
	} else {
		fmt.Println("Auto-bind: ENABLED (interview & bind immediately on join)")
	}
	fmt.Println()

	// Calculate permit duration (max 254 for permit join command)
	permitDur := uint8(254)
	if pairTimeout < 254 {
		permitDur = uint8(pairTimeout)
	}

	if err := a.PermitJoin(pairCtx, permitDur); err != nil {
		return fmt.Errorf("failed to open network: %w", err)
	}

	// Create timer for the pairing window
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Wait for timeout or context cancellation
	select {
	case <-timer.C:
		fmt.Println("\n\nPairing window closed (timeout)")
	case <-ctx.Done():
		fmt.Println("\n\nPairing interrupted by user")
	}

	// Close permit join
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = a.PermitJoin(closeCtx, 0)

	// Summary
	resultsMu.Lock()
	defer resultsMu.Unlock()

	fmt.Println()
	if len(results) == 0 {
		fmt.Println("No devices joined during pairing window")
		return nil
	}

	fmt.Printf("=== Pairing Summary: %d device(s) ===\n", len(results))
	for _, r := range results {
		status := "OK"
		if !r.interviewed {
			status = "FAILED (not interviewed)"
		} else if r.bound == 0 && !pairNoBind && coordErr == nil {
			status = "PARTIAL (no bindings)"
		}
		fmt.Printf("  %s (0x%04X): %s\n",
			znp.FormatIEEEAddr(r.device.IEEEAddr),
			r.device.NwkAddr,
			status,
		)
		if r.bound > 0 {
			fmt.Printf("    Bound %d cluster(s)\n", r.bound)
		}
		for _, e := range r.errors {
			fmt.Printf("    Error: %s\n", e)
		}
	}

	return nil
}

// shouldBindCluster returns true if the cluster should be auto-bound.
// These are clusters that typically send reports we want to receive.
func shouldBindCluster(clusterID uint16) bool {
	switch zcl.ClusterID(clusterID) {
	case zcl.ClusterOnOff, // Switch state
		zcl.ClusterLevelControl,     // Dimmer level
		zcl.ClusterColorControl,     // Color/temperature
		zcl.ClusterTempMeasurement,  // Temperature sensor
		zcl.ClusterHumidityMeas,     // Humidity sensor
		zcl.ClusterPressureMeas,     // Pressure sensor
		zcl.ClusterIlluminanceMeas,  // Light sensor
		zcl.ClusterOccupancySensing, // Motion sensor
		zcl.ClusterElectricalMeas,   // Power monitoring
		zcl.ClusterMeteringSimple,   // Energy metering
		zcl.ClusterPowerConfig,      // Battery level
		zcl.ClusterIASZone:          // Security sensors
		return true
	}
	return false
}

// goznp device list -p <port> [--interview]
var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List paired devices",
	Long: `Display all devices currently paired to the network.

Use --interview to perform full device discovery including manufacturer,
model, power source, and supported clusters for each device.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceList(ctx)
	},
}

func runDeviceList(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 300*time.Second) // Longer timeout for interviews
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("Fetching device list...")

	devices, err := a.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	if len(devices) == 0 {
		fmt.Println("\nNo devices paired")
		return nil
	}

	// If interview flag is set, run full interview on each device
	if interviewDevices {
		return runDeviceListWithInterview(ctx, a, devices)
	}

	// Otherwise, use the quick device type detection
	return runDeviceListQuick(ctx, a, devices)
}

func runDeviceListQuick(ctx context.Context, a *adapter.Adapter, devices []*adapter.Device) error {
	fmt.Printf("\nFound %d device(s):\n\n", len(devices))

	// Fetch custom device names
	deviceNames, _ := a.ListDeviceNames(ctx) // Ignore errors - names are optional
	namesByIEEE := make(map[[8]byte]string)
	for _, dn := range deviceNames {
		namesByIEEE[dn.IEEEAddr] = dn.Name
	}

	// Create a tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Network Addr\tIEEE Address\tName")
	fmt.Fprintln(w, "------------\t-----------------------\t----")

	for _, dev := range devices {
		name := namesByIEEE[dev.IEEEAddr]
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(w, "0x%04X\t%s\t%s\n",
			dev.NwkAddr,
			znp.FormatIEEEAddr(dev.IEEEAddr),
			name,
		)
	}

	w.Flush()

	fmt.Println("\nTip: Use --interview for detailed device information (manufacturer, model, clusters)")

	return nil
}

func runDeviceListWithInterview(ctx context.Context, a *adapter.Adapter, devices []*adapter.Device) error {
	fmt.Printf("Interviewing %d device(s)...\n\n", len(devices))

	// Fetch custom device names
	deviceNames, _ := a.ListDeviceNames(ctx) // Ignore errors - names are optional
	namesByIEEE := make(map[[8]byte]string)
	for _, dn := range deviceNames {
		namesByIEEE[dn.IEEEAddr] = dn.Name
	}

	results := make([]*adapter.InterviewResult, 0, len(devices))

	for i, dev := range devices {
		fmt.Printf("[%d/%d] Interviewing 0x%04X...", i+1, len(devices), dev.NwkAddr)

		// Use InterviewDeviceWithAddr to skip IeeeAddrReq (we already have it from NVRAM)
		result, err := a.InterviewDeviceWithAddr(ctx, dev.NwkAddr, dev.IEEEAddr)
		if err != nil {
			fmt.Printf(" FAILED: %v\n", err)
			// Create a placeholder result for failed interviews
			result = &adapter.InterviewResult{
				IEEEAddr: dev.IEEEAddr,
				NwkAddr:  dev.NwkAddr,
				Success:  false,
				Errors:   []string{err.Error()},
			}
		} else if result.Success {
			fmt.Printf(" OK (%s %s)\n", result.Manufacturer, result.Model)
		} else {
			fmt.Printf(" partial (errors: %d)\n", len(result.Errors))
		}

		results = append(results, result)
	}

	// Display summary table
	fmt.Printf("\n%d device(s):\n\n", len(results))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Addr\tIEEE Address\tName\tType\tManufacturer\tModel\tPower\tEndpoints")
	fmt.Fprintln(w, "------\t--------------------\t----\t----------\t------------\t-----\t------\t---------")

	for _, r := range results {
		deviceType := r.DeviceType.String()
		manufacturer := r.Manufacturer
		model := r.Model
		power := r.PowerSource.String()
		endpoints := fmt.Sprintf("%d", len(r.Endpoints))

		// Handle empty/unknown values
		if manufacturer == "" {
			manufacturer = "-"
		}
		if model == "" {
			model = "-"
		}
		if !r.Success && len(r.Errors) > 0 {
			deviceType = "Unknown"
			power = "-"
			endpoints = "-"
		}

		// Get custom name
		name := namesByIEEE[r.IEEEAddr]
		if name == "" {
			name = "-"
		}

		fmt.Fprintf(w, "0x%04X\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.NwkAddr,
			r.IEEEAddrString(),
			name,
			deviceType,
			manufacturer,
			model,
			power,
			endpoints,
		)
	}

	w.Flush()

	// Show detailed info for each device
	fmt.Println("\nDetailed Device Information:")
	fmt.Println("============================")

	for _, r := range results {
		fmt.Printf("\nDevice 0x%04X (%s):\n", r.NwkAddr, r.IEEEAddrString())

		if !r.Success {
			fmt.Println("  Status: FAILED")
			for _, e := range r.Errors {
				fmt.Printf("    - %s\n", e)
			}
			continue
		}

		// Helper to show - for empty mandatory fields
		strOrNA := func(s string) string {
			if s == "" {
				return "-"
			}
			return s
		}

		// Mandatory fields - always shown
		fmt.Printf("  Type:         %s\n", r.DeviceType)

		// Try to look up friendly device name
		if info := devicedb.Lookup(r.Manufacturer, r.Model); info != nil {
			fmt.Printf("  Vendor:       %s\n", info.Vendor)
			fmt.Printf("  Product:      %s\n", info.Model)
			fmt.Printf("  Description:  %s\n", info.Description)
			fmt.Printf("  Raw ID:       %s / %s\n", r.Manufacturer, r.Model)
		} else {
			fmt.Printf("  Manufacturer: %s\n", strOrNA(r.Manufacturer))
			fmt.Printf("  Model:        %s\n", strOrNA(r.Model))
		}

		fmt.Printf("  Power:        %s\n", r.PowerSource)
		fmt.Printf("  Serial:       %s\n", strOrNA(r.SerialNumber))

		// Optional fields - only shown if present
		if r.SWBuildID != "" {
			fmt.Printf("  SW Build:     %s\n", r.SWBuildID)
		}
		if r.ProductCode != "" {
			fmt.Printf("  Product Code: %s\n", r.ProductCode)
		}
		if r.ProductURL != "" {
			fmt.Printf("  Product URL:  %s\n", r.ProductURL)
		}
		if r.ManufacturerVersionDetails != "" {
			fmt.Printf("  Manuf Ver:    %s\n", r.ManufacturerVersionDetails)
		}
		if r.ProductLabel != "" {
			fmt.Printf("  Label:        %s\n", r.ProductLabel)
		}
		if r.ManufacturerCode != 0 {
			fmt.Printf("  Manuf Code:   0x%04X\n", r.ManufacturerCode)
		}

		if len(r.Endpoints) > 0 {
			fmt.Printf("  Endpoints:    %d\n", len(r.Endpoints))
			for _, ep := range r.Endpoints {
				fmt.Printf("    Endpoint %d (Profile: 0x%04X, Device: 0x%04X):\n",
					ep.Endpoint, ep.ProfileID, ep.DeviceID)
				if len(ep.InClusters) > 0 {
					fmt.Printf("      In:  ")
					for i, c := range ep.InClusters {
						if i > 0 {
							fmt.Printf(", ")
						}
						fmt.Printf("%s", zcl.ClusterID(c))
					}
					fmt.Println()
				}
				if len(ep.OutClusters) > 0 {
					fmt.Printf("      Out: ")
					for i, c := range ep.OutClusters {
						if i > 0 {
							fmt.Printf(", ")
						}
						fmt.Printf("%s", zcl.ClusterID(c))
					}
					fmt.Println()
				}
			}
		}

		if len(r.Errors) > 0 {
			fmt.Printf("  Warnings:\n")
			for _, e := range r.Errors {
				fmt.Printf("    - %s\n", e)
			}
		}
	}

	return nil
}

// goznp device watch -p <port>
var deviceWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for device events",
	Long:  "Monitor device join/leave events in real-time until Ctrl+C",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceWatch(ctx)
	},
}

func runDeviceWatch(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Set up device event handler
	eventChan := make(chan adapter.DeviceEvent, 10)
	a.OnDeviceEvent(func(event adapter.DeviceEvent) {
		select {
		case eventChan <- event:
		default:
			// Channel full, drop event
		}
	})

	// Enable permit join
	fmt.Printf("Opening network for pairing (%d seconds)...\n", permitDuration)
	if err := a.PermitJoin(ctx, permitDuration); err != nil {
		return fmt.Errorf("failed to permit join: %w", err)
	}

	fmt.Println("Watching for device events... (Ctrl+C to exit)")
	fmt.Println()

	// Watch for events until context is cancelled
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down...")
			// Close permit join on exit
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = a.PermitJoin(closeCtx, 0)
			return nil

		case event := <-eventChan:
			timestamp := time.Now().Format("15:04:05")
			switch event.Type {
			case adapter.DeviceEventJoined:
				fmt.Printf("[%s] Device JOINED: %s (0x%04X)\n",
					timestamp,
					znp.FormatIEEEAddr(event.Device.IEEEAddr),
					event.Device.NwkAddr,
				)
			case adapter.DeviceEventLeft:
				fmt.Printf("[%s] Device LEFT: %s\n",
					timestamp,
					znp.FormatIEEEAddr(event.IEEEAddr),
				)
			}
		}
	}
}

// goznp device status -p <port> --addr <nwk-addr>
var deviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query device status",
	Long:  "Read status attributes from a device (manufacturer, model, on/off, battery)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceStatus(ctx)
	},
}

func runDeviceStatus(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceNameLookup, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	// Get device IEEE address to fetch custom name
	var customName string
	devices, err := a.GetDevices(ctx)
	if err == nil {
		for _, dev := range devices {
			if dev.NwkAddr == nwkAddr {
				// Get custom name if set
				nameInfo, _ := a.GetDeviceName(ctx, dev.IEEEAddr)
				if nameInfo != nil && nameInfo.Name != "" {
					customName = nameInfo.Name
				}
				break
			}
		}
	}

	if customName != "" {
		fmt.Printf("Querying device 0x%04X (%s) endpoint %d...\n", nwkAddr, customName, deviceEndpoint)
	} else {
		fmt.Printf("Querying device 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)
	}

	status, err := a.GetDeviceStatus(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("failed to get device status: %w", err)
	}

	// Display status
	fmt.Println("\nDevice Status:")
	if customName != "" {
		fmt.Printf("  Name:             %s\n", customName)
	}
	fmt.Printf("  Network Address:  0x%04X\n", status.NwkAddr)

	if status.Manufacturer != "" {
		fmt.Printf("  Manufacturer:     %s\n", status.Manufacturer)
	}
	if status.Model != "" {
		fmt.Printf("  Model:            %s\n", status.Model)
	}

	fmt.Printf("  Power Source:     %s\n", formatPowerSource(status.PowerSource))

	if status.BatteryPercent != nil {
		fmt.Printf("  Battery:          %d%%\n", *status.BatteryPercent)
	} else {
		fmt.Printf("  Battery:          -\n")
	}

	if status.OnOff != nil {
		onOffStr := "off"
		if *status.OnOff {
			onOffStr = "on"
		}
		fmt.Printf("  On/Off State:     %s\n", onOffStr)
	} else {
		fmt.Printf("  On/Off State:     -\n")
	}

	return nil
}

// formatPowerSource formats power source as human-readable string.
func formatPowerSource(ps zcl.PowerSource) string {
	switch ps {
	case zcl.PowerSourceUnknown:
		return "Unknown"
	case zcl.PowerSourceMains:
		return "Mains"
	case zcl.PowerSourceBattery:
		return "Battery"
	case zcl.PowerSourceDC:
		return "DC"
	case zcl.PowerSourceEmergencyMains:
		return "Emergency Mains"
	case zcl.PowerSourceEmergencyDC:
		return "Emergency DC"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", ps)
	}
}

// goznp device on -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn device on",
	Long:  "Send On command to device (On/Off cluster)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "on")
	},
}

// goznp device off -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn device off",
	Long:  "Send Off command to device (On/Off cluster)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "off")
	},
}

// goznp device toggle -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle device state",
	Long:  "Send Toggle command to device (On/Off cluster)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "toggle")
	},
}

func runDeviceControl(ctx context.Context, action string) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceNameLookup, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Sending %s command to device 0x%04X endpoint %d...\n", action, nwkAddr, deviceEndpoint)

	switch action {
	case "on":
		err = a.TurnOn(ctx, nwkAddr, deviceEndpoint)
	case "off":
		err = a.TurnOff(ctx, nwkAddr, deviceEndpoint)
	case "toggle":
		err = a.Toggle(ctx, nwkAddr, deviceEndpoint)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	fmt.Printf("Command sent successfully\n")

	return nil
}

// goznp device brightness -p <port> --addr <nwk-addr> [--endpoint <ep>] [--level <0-254>] [--transition <ms>]
var deviceBrightnessCmd = &cobra.Command{
	Use:   "brightness",
	Short: "Get or set device brightness",
	Long: `Get or set the brightness level on a dimmable device.

Without --level flag: reads and displays the current brightness.
With --level flag: sets the brightness to the specified level.

Level values:
  0   = Off (will also turn off the device)
  1   = Minimum brightness
  254 = Maximum brightness

Transition time is in milliseconds (default: 0 for instant).
The --transition flag specifies how long the transition should take.

Examples:
  goznp device brightness --addr 0xBE87                      # Read current brightness
  goznp device brightness --addr 0xBE87 --level 128          # Set to 50%
  goznp device brightness --addr 0xBE87 --level 254 --transition 1000  # Fade to max over 1s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceBrightness(ctx)
	},
}

func runDeviceBrightness(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceNameLookup, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	// If no level specified, read current brightness
	if brightnessLevel == 0 && transitionTime == 0 {
		fmt.Printf("Reading brightness from device 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)

		level, err := a.GetBrightness(ctx, nwkAddr, deviceEndpoint)
		if err != nil {
			return fmt.Errorf("failed to read brightness: %w", err)
		}

		percent := int(level) * 100 / 254
		fmt.Printf("\nBrightness: %d (%.0f%%)\n", level, float64(percent))
		return nil
	}

	// Set brightness
	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	fmt.Printf("Setting brightness on device 0x%04X endpoint %d to %d", nwkAddr, deviceEndpoint, brightnessLevel)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetBrightness(ctx, nwkAddr, deviceEndpoint, brightnessLevel, transitionTenths); err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	percent := int(brightnessLevel) * 100 / 254
	fmt.Printf("Brightness set to %d (%.0f%%)\n", brightnessLevel, float64(percent))

	return nil
}

// goznp device color-temp -p <port> --addr <nwk-addr> [--endpoint <ep>] [--kelvin <K>] [--mireds <mireds>] [--transition <ms>]
var deviceColorTempCmd = &cobra.Command{
	Use:   "color-temp",
	Short: "Get or set color temperature",
	Long: `Get or set the color temperature on a tunable white light.

Without --kelvin or --mireds flag: reads and displays the current color temperature.
With --kelvin or --mireds flag: sets the color temperature.

Temperature can be specified in either:
  --kelvin   Color temperature in Kelvin (e.g., 2700 for warm, 6500 for cool)
  --mireds   Color temperature in mireds (1,000,000 / Kelvin)

Common color temperatures:
  2700K (370 mireds) = Warm white (incandescent)
  3000K (333 mireds) = Soft white
  4000K (250 mireds) = Cool white
  5000K (200 mireds) = Daylight
  6500K (154 mireds) = Cool daylight

Transition time is in milliseconds (default: 0 for instant).

Examples:
  goznp device color-temp --addr 0xBE87                         # Read current temp
  goznp device color-temp --addr 0xBE87 --kelvin 2700           # Set to warm white
  goznp device color-temp --addr 0xBE87 --kelvin 6500 --transition 2000  # Cool over 2s
  goznp device color-temp --addr 0xBE87 --mireds 250            # Set using mireds`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceColorTemp(ctx)
	},
}

func runDeviceColorTemp(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// If no temperature specified, read current color temp
	if colorTempKelvin == 0 && colorTempMireds == 0 {
		fmt.Printf("Reading color temperature from device 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)

		info, err := a.GetColorTempInfo(ctx, nwkAddr, deviceEndpoint)
		if err != nil {
			return fmt.Errorf("failed to read color temp: %w", err)
		}

		fmt.Println("\nColor Temperature:")
		if info.CurrentMireds > 0 {
			kelvin := 1000000 / int(info.CurrentMireds)
			fmt.Printf("  Current:  %d mireds (%dK)\n", info.CurrentMireds, kelvin)
		} else {
			fmt.Println("  Current:  -")
		}

		if info.MinMireds > 0 && info.MaxMireds > 0 {
			minK := 1000000 / int(info.MaxMireds) // Max mireds = min Kelvin
			maxK := 1000000 / int(info.MinMireds) // Min mireds = max Kelvin
			fmt.Printf("  Range:    %d-%d mireds (%d-%dK)\n", info.MinMireds, info.MaxMireds, minK, maxK)
		}
		return nil
	}

	// Calculate mireds from Kelvin if specified
	var targetMireds uint16
	if colorTempKelvin > 0 {
		targetMireds = uint16(1000000 / int(colorTempKelvin))
	} else {
		targetMireds = colorTempMireds
	}

	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	kelvin := 1000000 / int(targetMireds)
	fmt.Printf("Setting color temperature on device 0x%04X endpoint %d to %d mireds (%dK)", nwkAddr, deviceEndpoint, targetMireds, kelvin)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetColorTemperature(ctx, nwkAddr, deviceEndpoint, targetMireds, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color temperature: %w", err)
	}

	fmt.Printf("Color temperature set to %d mireds (%dK)\n", targetMireds, kelvin)

	return nil
}

// ============================================================================
// Identify Cluster Commands
// ============================================================================

// goznp device identify -p <port> --addr <nwk-addr> [--endpoint <ep>] [--duration <secs>] [--effect <effect>]
var deviceIdentifyCmd = &cobra.Command{
	Use:   "identify",
	Short: "Make device identify itself",
	Long: `Trigger visual/audible identification on a device.

The device will flash, blink, or perform another visual indication to help
locate it physically. This is useful when you have many devices and need
to find a specific one.

Use --duration to specify how long to identify (default: 5 seconds).
Use --effect to trigger a specific effect (requires ZCL 6+ support).

Effects:
  blink    - Light flashes on/off once
  breathe  - Light fades in and out
  okay     - Brief green flash (color lights)
  channel  - Brief orange flash (color lights)
  stop     - Stop immediately

Examples:
  goznp device identify --addr 0xBE87 --endpoint 11              # Identify for 5s
  goznp device identify --addr 0xBE87 --endpoint 11 --duration 10 # Identify for 10s
  goznp device identify --addr 0xBE87 --endpoint 11 --effect blink # Trigger blink effect`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceIdentify(ctx)
	},
}

func runDeviceIdentify(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceNameLookup, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	// If effect specified, use TriggerEffect
	if identifyEffect != "" {
		effectID, err := parseIdentifyEffect(identifyEffect)
		if err != nil {
			return err
		}

		fmt.Printf("Triggering %s effect on device 0x%04X endpoint %d...\n", identifyEffect, nwkAddr, deviceEndpoint)
		if err := a.TriggerEffect(ctx, nwkAddr, deviceEndpoint, effectID, 0); err != nil {
			return fmt.Errorf("trigger effect failed: %w", err)
		}
		fmt.Println("Effect triggered!")
		return nil
	}

	// Use standard identify command
	duration := identifyDuration
	if duration == 0 {
		duration = 5 // Default 5 seconds
	}

	fmt.Printf("Starting identify on device 0x%04X endpoint %d for %d seconds...\n", nwkAddr, deviceEndpoint, duration)
	if err := a.Identify(ctx, nwkAddr, deviceEndpoint, duration); err != nil {
		return fmt.Errorf("identify failed: %w", err)
	}

	fmt.Printf("Device is identifying for %d seconds\n", duration)
	return nil
}

func parseIdentifyEffect(effect string) (uint8, error) {
	switch strings.ToLower(effect) {
	case "blink":
		return zcl.IdentifyEffectBlink, nil
	case "breathe":
		return zcl.IdentifyEffectBreathe, nil
	case "okay", "ok":
		return zcl.IdentifyEffectOkay, nil
	case "channel":
		return zcl.IdentifyEffectChannelChg, nil
	case "finish":
		return zcl.IdentifyEffectFinishEffect, nil
	case "stop":
		return zcl.IdentifyEffectStopEffect, nil
	default:
		return 0, fmt.Errorf("unknown effect %q (valid: blink, breathe, okay, channel, stop)", effect)
	}
}

// ============================================================================
// Groups Cluster Commands
// ============================================================================

// Parent command for group operations
var deviceGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage device group membership",
	Long:  "Add, remove, or list group memberships for a device",
}

// goznp device group add --addr <nwk-addr> --group <id> [--name <name>]
var deviceGroupAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add device to a group",
	Long: `Add a device endpoint to a Zigbee group.

Groups allow controlling multiple devices with a single command (multicast).
Group IDs are 16-bit values (1-65535). Group 0 is reserved.

Examples:
  goznp device group add --addr 0xBE87 --endpoint 11 --group 1
  goznp device group add --addr 0xBE87 --endpoint 11 --group 1 --name "Living Room"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupAdd(ctx)
	},
}

func runDeviceGroupAdd(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Adding device 0x%04X endpoint %d to group %d", nwkAddr, deviceEndpoint, groupID)
	if groupName != "" {
		fmt.Printf(" (%s)", groupName)
	}
	fmt.Println("...")

	if err := a.AddToGroup(ctx, nwkAddr, deviceEndpoint, groupID, groupName); err != nil {
		return fmt.Errorf("add to group failed: %w", err)
	}

	fmt.Printf("Device added to group %d\n", groupID)
	return nil
}

// goznp device group remove --addr <nwk-addr> --group <id>
var deviceGroupRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove device from a group",
	Long: `Remove a device endpoint from a Zigbee group.

Use --group 0 or --all to remove from all groups.

Examples:
  goznp device group remove --addr 0xBE87 --endpoint 11 --group 1
  goznp device group remove --addr 0xBE87 --endpoint 11 --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupRemove(ctx)
	},
}

var removeAllGroups bool

func runDeviceGroupRemove(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	if removeAllGroups {
		fmt.Printf("Removing device 0x%04X endpoint %d from all groups...\n", nwkAddr, deviceEndpoint)
		if err := a.RemoveFromAllGroups(ctx, nwkAddr, deviceEndpoint); err != nil {
			return fmt.Errorf("remove from all groups failed: %w", err)
		}
		fmt.Println("Device removed from all groups")
		return nil
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (or use --all)")
	}

	fmt.Printf("Removing device 0x%04X endpoint %d from group %d...\n", nwkAddr, deviceEndpoint, groupID)
	if err := a.RemoveFromGroup(ctx, nwkAddr, deviceEndpoint, groupID); err != nil {
		return fmt.Errorf("remove from group failed: %w", err)
	}

	fmt.Printf("Device removed from group %d\n", groupID)
	return nil
}

// goznp device group list --addr <nwk-addr>
var deviceGroupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List device group memberships",
	Long: `Query which groups a device endpoint belongs to.

Example:
  goznp device group list --addr 0xBE87 --endpoint 11`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupList(ctx)
	},
}

func runDeviceGroupList(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Querying group membership for device 0x%04X endpoint %d...\n\n", nwkAddr, deviceEndpoint)

	membership, err := a.GetGroupMembership(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("get group membership failed: %w", err)
	}

	if len(membership.Groups) == 0 {
		fmt.Println("Device is not a member of any groups")
	} else {
		fmt.Printf("Device is a member of %d group(s):\n", len(membership.Groups))
		for _, g := range membership.Groups {
			fmt.Printf("  - Group %d (0x%04X)\n", g, g)
		}
	}

	if membership.Capacity != 0xFF {
		fmt.Printf("\nRemaining capacity: %d groups\n", membership.Capacity)
	}

	return nil
}

// goznp device group on --group <id>
var deviceGroupOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn on all devices in a group",
	Long: `Send On command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.
This is much faster than sending individual commands to each device.

Example:
  goznp device group on --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "on")
	},
}

// goznp device group off --group <id>
var deviceGroupOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn off all devices in a group",
	Long: `Send Off command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.

Example:
  goznp device group off --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "off")
	},
}

// goznp device group toggle --group <id>
var deviceGroupToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle all devices in a group",
	Long: `Send Toggle command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.

Example:
  goznp device group toggle --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "toggle")
	},
}

// goznp device group brightness --group <id> --level <0-254>
var deviceGroupBrightnessCmd = &cobra.Command{
	Use:   "brightness",
	Short: "Set brightness for all devices in a group",
	Long: `Set brightness level on all dimmable devices in a Zigbee group.

Level values:
  0   = Off (will also turn off devices)
  1   = Minimum brightness
  254 = Maximum brightness

Transition time is in milliseconds (default: 0 for instant).

Examples:
  goznp device group brightness --group 1 --level 128                    # Set to 50%
  goznp device group brightness --group 1 --level 254 --transition 1000  # Fade to max over 1s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupBrightness(ctx)
	},
}

// goznp device group scene --group <id> --scene <id>
var deviceGroupSceneCmd = &cobra.Command{
	Use:   "recall-scene",
	Short: "Recall scene on all devices in a group",
	Long: `Recall a scene on all devices in a Zigbee group.

This sends the RecallScene command to all group members simultaneously.
The devices will transition to their saved scene states.

Example:
  goznp device group recall-scene --group 1 --scene 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupScene(ctx)
	},
}

func runDeviceGroupCommand(ctx context.Context, action string) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Sending %s command to group %d...\n", action, groupID)

	switch action {
	case "on":
		err = a.GroupTurnOn(ctx, groupID)
	case "off":
		err = a.GroupTurnOff(ctx, groupID)
	case "toggle":
		err = a.GroupToggle(ctx, groupID)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	fmt.Printf("Command sent to group %d\n", groupID)

	return nil
}

func runDeviceGroupBrightness(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	fmt.Printf("Setting brightness on group %d to %d", groupID, brightnessLevel)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.GroupSetBrightness(ctx, groupID, brightnessLevel, transitionTenths); err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	percent := int(brightnessLevel) * 100 / 254
	fmt.Printf("Brightness set to %d (%.0f%%) on group %d\n", brightnessLevel, float64(percent), groupID)

	return nil
}

func runDeviceGroupScene(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Recalling scene %d on group %d...\n", sceneID, groupID)

	if err := a.GroupRecallScene(ctx, groupID, sceneID); err != nil {
		return fmt.Errorf("recall scene failed: %w", err)
	}

	fmt.Printf("Scene %d recalled on group %d\n", sceneID, groupID)
	return nil
}

// ============================================================================
// Scenes Cluster Commands
// ============================================================================

// Parent command for scene operations
var deviceSceneCmd = &cobra.Command{
	Use:   "scene",
	Short: "Manage device scenes",
	Long: `Store, recall, and manage scenes on a device.

Scenes save device state (brightness, color, etc.) that can be recalled instantly.
Scenes are stored on the device itself, so they work even when the coordinator is offline.`,
}

// goznp device scene store --addr <nwk-addr> --group <id> --scene <id>
var deviceSceneStoreCmd = &cobra.Command{
	Use:   "store",
	Short: "Store current state as a scene",
	Long: `Store the device's current state as a scene.

The device saves its current attribute values (brightness, color, on/off state, etc.)
to the specified scene ID within the group.

Note: The device must be a member of the group before storing a scene.

Examples:
  goznp device scene store --addr 0xBE87 --endpoint 11 --group 1 --scene 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneStore(ctx)
	},
}

func runDeviceSceneStore(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Storing current state as scene %d in group %d on device 0x%04X endpoint %d...\n",
		sceneID, groupID, nwkAddr, deviceEndpoint)

	if err := a.StoreScene(ctx, nwkAddr, deviceEndpoint, groupID, sceneID); err != nil {
		return fmt.Errorf("store scene failed: %w", err)
	}

	fmt.Printf("Scene %d stored successfully\n", sceneID)
	return nil
}

// goznp device scene recall --addr <nwk-addr> --group <id> --scene <id>
var deviceSceneRecallCmd = &cobra.Command{
	Use:   "recall",
	Short: "Recall/activate a scene",
	Long: `Recall a previously stored scene.

The device transitions to the saved attribute values for the specified scene.

Examples:
  goznp device scene recall --addr 0xBE87 --endpoint 11 --group 1 --scene 1
  goznp device scene recall --addr 0xBE87 --endpoint 11 --group 1 --scene 1 --transition 1000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneRecall(ctx)
	},
}

func runDeviceSceneRecall(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Recalling scene %d from group %d on device 0x%04X endpoint %d", sceneID, groupID, nwkAddr, deviceEndpoint)

	var transTime *uint16
	if transitionTime > 0 {
		// Convert ms to tenths of a second
		tenths := transitionTime / 100
		transTime = &tenths
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.RecallScene(ctx, nwkAddr, deviceEndpoint, groupID, sceneID, transTime); err != nil {
		return fmt.Errorf("recall scene failed: %w", err)
	}

	fmt.Printf("Scene %d recalled\n", sceneID)
	return nil
}

// goznp device scene remove --addr <nwk-addr> --group <id> [--scene <id>] [--all]
var deviceSceneRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a scene",
	Long: `Remove a scene from the device.

Use --scene to remove a specific scene, or --all to remove all scenes for the group.

Examples:
  goznp device scene remove --addr 0xBE87 --endpoint 11 --group 1 --scene 1
  goznp device scene remove --addr 0xBE87 --endpoint 11 --group 1 --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneRemove(ctx)
	},
}

var removeAllScenes bool

func runDeviceSceneRemove(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	if removeAllScenes {
		fmt.Printf("Removing all scenes for group %d on device 0x%04X endpoint %d...\n",
			groupID, nwkAddr, deviceEndpoint)
		if err := a.RemoveAllScenes(ctx, nwkAddr, deviceEndpoint, groupID); err != nil {
			return fmt.Errorf("remove all scenes failed: %w", err)
		}
		fmt.Println("All scenes removed")
		return nil
	}

	fmt.Printf("Removing scene %d from group %d on device 0x%04X endpoint %d...\n",
		sceneID, groupID, nwkAddr, deviceEndpoint)

	if err := a.RemoveScene(ctx, nwkAddr, deviceEndpoint, groupID, sceneID); err != nil {
		return fmt.Errorf("remove scene failed: %w", err)
	}

	fmt.Printf("Scene %d removed\n", sceneID)
	return nil
}

// goznp device scene list --addr <nwk-addr> --group <id>
var deviceSceneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scenes for a group",
	Long: `Query which scenes are stored on the device for a group.

Example:
  goznp device scene list --addr 0xBE87 --endpoint 11 --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneList(ctx)
	},
}

func runDeviceSceneList(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Querying scenes for group %d on device 0x%04X endpoint %d...\n\n",
		groupID, nwkAddr, deviceEndpoint)

	membership, err := a.GetSceneMembership(ctx, nwkAddr, deviceEndpoint, groupID)
	if err != nil {
		return fmt.Errorf("get scene membership failed: %w", err)
	}

	if membership.Status != 0 {
		return fmt.Errorf("device returned error status 0x%02X", membership.Status)
	}

	if len(membership.Scenes) == 0 {
		fmt.Printf("No scenes stored for group %d\n", groupID)
	} else {
		fmt.Printf("Group %d has %d scene(s):\n", groupID, len(membership.Scenes))
		for _, s := range membership.Scenes {
			fmt.Printf("  - Scene %d\n", s)
		}
	}

	if membership.Capacity != 0xFF {
		fmt.Printf("\nRemaining capacity: %d scenes\n", membership.Capacity)
	}

	return nil
}

// goznp device info -p <port> --addr <nwk-addr>
var deviceInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Query device capabilities",
	Long:  "Query device endpoints and supported clusters (capabilities)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceInfo(ctx)
	},
}

// goznp device power -p <port> --addr <nwk-addr> [--endpoint <ep>]
var devicePowerCmd = &cobra.Command{
	Use:   "power",
	Short: "Read power consumption",
	Long:  "Read power consumption data from a smart plug (voltage, current, power, energy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDevicePower(ctx)
	},
}

// goznp device reset-energy -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceResetEnergyCmd = &cobra.Command{
	Use:   "reset-energy",
	Short: "Reset energy counter",
	Long:  "Reset the accumulated energy counter on a smart plug (manufacturer-specific)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceResetEnergy(ctx)
	},
}

func runDeviceResetEnergy(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Attempting to reset energy counter on device 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)

	if err := a.ResetEnergy(ctx, nwkAddr, deviceEndpoint); err != nil {
		return fmt.Errorf("failed to reset energy: %w", err)
	}

	fmt.Println("Energy counter reset successfully!")
	fmt.Println("(Run 'device power' to verify the new value)")

	return nil
}

// goznp device remove -p <port> (--addr <nwk-addr> | --ieee <ieee-addr>) [--force]
var deviceRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a device from the network",
	Long: `Request a device to leave the Zigbee network.

This sends a ZDO Management Leave Request to the device, which tells it
to gracefully leave the network. The device will be unpaired and will
need to be re-paired to rejoin.

You can specify the device by either:
  --addr   Network address (hex, e.g., 0x1234)
  --ieee   IEEE address (e.g., 00:11:22:33:44:55:66:77)

Note: Battery-powered devices may be asleep and not respond immediately.
For sleepy devices, you may need to wake them up (e.g., press a button)
or use --force to remove them from the coordinator's device list.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceRemove(ctx)
	},
}

func runDeviceRemove(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Validate that at least one address type is provided
	if deviceAddr == "" && deviceIEEE == "" {
		return fmt.Errorf("either --addr or --ieee must be specified")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Get device list to find the device
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device list: %w", err)
	}

	var targetDevice *adapter.Device

	if deviceIEEE != "" {
		// Find by IEEE address
		ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
		if err != nil {
			return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
		}

		fmt.Printf("Looking up device %s...\n", znp.FormatIEEEAddr(ieeeAddr))

		for _, dev := range devices {
			if dev.IEEEAddr == ieeeAddr {
				targetDevice = dev
				break
			}
		}

		if targetDevice == nil {
			return fmt.Errorf("device %s not found in paired devices", znp.FormatIEEEAddr(ieeeAddr))
		}
	} else {
		// Find by network address
		nwkAddr, err := parseNetworkAddr(deviceAddr)
		if err != nil {
			return err
		}

		fmt.Printf("Looking up device 0x%04X...\n", nwkAddr)

		for _, dev := range devices {
			if dev.NwkAddr == nwkAddr {
				targetDevice = dev
				break
			}
		}

		if targetDevice == nil {
			return fmt.Errorf("device 0x%04X not found in paired devices", nwkAddr)
		}
	}

	fmt.Printf("Found device: %s (0x%04X)\n", znp.FormatIEEEAddr(targetDevice.IEEEAddr), targetDevice.NwkAddr)
	fmt.Printf("Sending leave request...\n")

	// Send leave request (no removeChildren, no rejoin)
	err = a.RemoveDevice(ctx, targetDevice.NwkAddr, targetDevice.IEEEAddr, false, false)
	if err != nil {
		if deviceForce {
			fmt.Printf("Warning: Leave request failed (%v), device may be offline\n", err)
			fmt.Println("Force-removing from coordinator's NVRAM...")

			// Force remove from NVRAM
			if err := a.ForceRemoveDevice(ctx, targetDevice.IEEEAddr); err != nil {
				return fmt.Errorf("force remove failed: %w", err)
			}

			fmt.Println("Device entry removed from coordinator NVRAM")
			fmt.Println("Note: The device still thinks it's joined - factory reset it to re-pair")
			return nil
		}
		return fmt.Errorf("failed to remove device: %w", err)
	}

	fmt.Println("Device removed successfully")
	fmt.Println("The device has been asked to leave the network and can be re-paired if needed")

	return nil
}

func runDevicePower(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Reading power data from device 0x%04X endpoint %d...\n\n", nwkAddr, deviceEndpoint)

	data, err := a.ReadPowerData(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("failed to read power data: %w", err)
	}

	fmt.Println("Power Data:")

	// Electrical Measurement values
	if data.Voltage != nil {
		fmt.Printf("  Voltage:      %.1f V\n", *data.Voltage)
	} else {
		fmt.Println("  Voltage:      -")
	}

	if data.Current != nil {
		fmt.Printf("  Current:      %.3f A\n", *data.Current)
	} else {
		fmt.Println("  Current:      -")
	}

	if data.ActivePower != nil {
		fmt.Printf("  Active Power: %.1f W\n", *data.ActivePower)
	} else {
		fmt.Println("  Active Power: -")
	}

	if data.PowerFactor != nil {
		fmt.Printf("  Power Factor: %d%%\n", *data.PowerFactor)
	} else {
		fmt.Println("  Power Factor: -")
	}

	// Simple Metering values
	fmt.Println()
	if data.TotalEnergy != nil {
		fmt.Printf("  Total Energy: %.3f kWh\n", *data.TotalEnergy)
	} else {
		fmt.Println("  Total Energy: -")
	}

	if data.InstantPower != nil {
		fmt.Printf("  Instant Power: %.1f W (from metering)\n", *data.InstantPower)
	}

	// Always show raw values for debugging different device variants
	fmt.Println("\nRaw Values (for debugging):")
	fmt.Printf("  Raw Voltage: %d\n", data.RawVoltage)
	fmt.Printf("  Raw Current: %d\n", data.RawCurrent)
	fmt.Printf("  Raw Power:   %d\n", data.RawPower)
	fmt.Printf("  Raw Energy:  %d\n", data.RawEnergy)

	return nil
}

func runDeviceInfo(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Querying device 0x%04X capabilities...\n\n", nwkAddr)

	caps, err := a.GetDeviceCapabilities(ctx, nwkAddr)
	if err != nil {
		return fmt.Errorf("failed to get device capabilities: %w", err)
	}

	if len(caps.Endpoints) == 0 {
		fmt.Println("No endpoints found (device may be asleep)")
		return nil
	}

	// Get device IEEE address to fetch custom name
	devices, err := a.GetDevices(ctx)
	if err == nil {
		for _, dev := range devices {
			if dev.NwkAddr == nwkAddr {
				// Get custom name if set
				nameInfo, _ := a.GetDeviceName(ctx, dev.IEEEAddr)
				if nameInfo != nil && nameInfo.Name != "" {
					fmt.Printf("Device 0x%04X (%s) has %d endpoint(s):\n", nwkAddr, nameInfo.Name, len(caps.Endpoints))
				} else {
					fmt.Printf("Device 0x%04X has %d endpoint(s):\n", nwkAddr, len(caps.Endpoints))
				}
				goto endpointLoop
			}
		}
	}
	// Fallback if we couldn't get device list or find the device
	fmt.Printf("Device 0x%04X has %d endpoint(s):\n", nwkAddr, len(caps.Endpoints))
endpointLoop:

	for _, ep := range caps.Endpoints {
		fmt.Printf("\n  Endpoint %d:\n", ep.Endpoint)
		fmt.Printf("    Profile:  0x%04X (%s)\n", ep.ProfileID, formatProfileID(ep.ProfileID))
		fmt.Printf("    Device:   0x%04X\n", ep.DeviceID)

		if len(ep.InClusters) > 0 {
			fmt.Printf("    Input Clusters (device capabilities):\n")
			for _, cluster := range ep.InClusters {
				fmt.Printf("      - 0x%04X %s\n", cluster, zcl.ClusterID(cluster).String())
			}
		}

		if len(ep.OutClusters) > 0 {
			fmt.Printf("    Output Clusters (device can request):\n")
			for _, cluster := range ep.OutClusters {
				fmt.Printf("      - 0x%04X %s\n", cluster, zcl.ClusterID(cluster).String())
			}
		}
	}

	return nil
}

func formatProfileID(profileID uint16) string {
	switch profileID {
	case 0x0104:
		return "Home Automation"
	case 0x0109:
		return "Smart Energy"
	case 0xC05E:
		return "ZigBee Light Link"
	case 0xA1E0:
		return "Green Power"
	default:
		return "Unknown"
	}
}

// goznp device sensor -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceSensorCmd = &cobra.Command{
	Use:   "sensor",
	Short: "Read sensor data",
	Long:  "Read temperature, humidity, and battery from a sensor device",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSensor(ctx)
	},
}

// goznp device listen -p <port>
var deviceListenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for sensor reports",
	Long:  "Listen for incoming sensor reports (temperature, humidity, etc.) from all devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceListen(ctx)
	},
}

func runDeviceListen(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("Listening for sensor reports... (Ctrl+C to stop)")
	fmt.Println("Tip: Press a button on your sensor to trigger a report")
	fmt.Println()
	fmt.Println("Note: Smart plugs typically don't send automatic power reports unless")
	fmt.Println("      configured to do so. Use 'device power' to actively read power data.")
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			report, err := a.WaitForSensorReport(ctx, 5*time.Second)
			if err != nil {
				continue // Timeout, try again
			}

			// Skip if not a Report Attributes command (0x0A)
			// Most messages are device announcements or other non-report commands
			if report.FrameCommandID != 0x0A && report.ParseError == "" {
				continue
			}

			timestamp := time.Now().Format("15:04:05")
			fmt.Printf("[%s] Device 0x%04X (Cluster: 0x%04X, Endpoint: %d, Cmd: 0x%02X):\n",
				timestamp, report.NwkAddr, report.ClusterID, report.Endpoint, report.FrameCommandID)

			// Show parse errors
			if report.ParseError != "" {
				fmt.Printf("  Parse error: %s\n", report.ParseError)
			}

			// Display sensor data
			if report.Temperature != nil {
				fmt.Printf("  Temperature: %.1f°C\n", *report.Temperature)
			}
			if report.Humidity != nil {
				fmt.Printf("  Humidity:    %.1f%%\n", *report.Humidity)
			}
			if report.Battery != nil {
				fmt.Printf("  Battery:     %d%%\n", *report.Battery)
			}

			// Display on/off state
			if report.OnOff != nil {
				state := "OFF"
				if *report.OnOff {
					state = "ON"
				}
				fmt.Printf("  On/Off:      %s\n", state)
			}

			// Display power consumption data
			if report.Voltage != nil {
				fmt.Printf("  Voltage:     %.1f V\n", *report.Voltage)
			}
			if report.Current != nil {
				fmt.Printf("  Current:     %.3f A\n", *report.Current)
			}
			if report.Power != nil {
				fmt.Printf("  Power:       %.1f W\n", *report.Power)
			}
			if report.Energy != nil {
				fmt.Printf("  Energy:      %.3f kWh\n", *report.Energy)
			}

			// Display raw attributes for debugging if nothing else was displayed
			hasData := report.Temperature != nil || report.Humidity != nil || report.Battery != nil ||
				report.Voltage != nil || report.Current != nil || report.Power != nil || report.Energy != nil ||
				report.OnOff != nil
			if !hasData && len(report.Attributes) > 0 {
				fmt.Println("  Raw attributes:")
				for _, attr := range report.Attributes {
					fmt.Printf("    Attr 0x%04X: %v (type: %T)\n", attr.ID, attr.Value, attr.Value)
				}
			}
			fmt.Println()
		}
	}
}

func runDeviceSensor(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Reading sensor data from device 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)
	fmt.Println("(Note: Battery sensors may be asleep and not respond immediately)")
	fmt.Println()

	data, err := a.ReadSensorData(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("failed to read sensor data: %w", err)
	}

	fmt.Println("Sensor Data:")
	if data.Temperature != nil {
		fmt.Printf("  Temperature:  %.1f°C\n", *data.Temperature)
	} else {
		fmt.Println("  Temperature:  - (device may be asleep)")
	}

	if data.Humidity != nil {
		fmt.Printf("  Humidity:     %.1f%%\n", *data.Humidity)
	} else {
		fmt.Println("  Humidity:     -")
	}

	if data.Battery != nil {
		fmt.Printf("  Battery:      %d%%\n", *data.Battery)
	} else {
		fmt.Println("  Battery:      -")
	}

	return nil
}

// parseNetworkAddr parses hex network address (e.g., "0x1234" or "1234").
func parseNetworkAddr(s string) (uint16, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	val, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid network address %q: %w", s, err)
	}
	return uint16(val), nil
}

// resolveDeviceAddr resolves a device by name, IEEE address, or network address.
// Priority: --name > --ieee > --addr
func resolveDeviceAddr(ctx context.Context, a *adapter.Adapter, nameLookup, ieeeLookup, addrLookup string) (uint16, error) {
	// If name is provided, look it up
	if nameLookup != "" {
		names, err := a.ListDeviceNames(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to list device names: %w", err)
		}

		var matchedIEEE [8]byte
		found := false
		for _, n := range names {
			if n.Name == nameLookup {
				matchedIEEE = n.IEEEAddr
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("no device found with name %q", nameLookup)
		}

		devices, err := a.GetDevices(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get devices: %w", err)
		}

		for _, d := range devices {
			if d.IEEEAddr == matchedIEEE {
				return d.NwkAddr, nil
			}
		}
		return 0, fmt.Errorf("device %q not found in network", nameLookup)
	}

	// If IEEE is provided, look it up
	if ieeeLookup != "" {
		ieeeAddr, err := znp.ParseIEEEAddr(ieeeLookup)
		if err != nil {
			return 0, fmt.Errorf("invalid IEEE address: %w", err)
		}

		devices, err := a.GetDevices(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get devices: %w", err)
		}

		for _, d := range devices {
			if d.IEEEAddr == ieeeAddr {
				return d.NwkAddr, nil
			}
		}
		return 0, fmt.Errorf("device with IEEE %s not found in network", ieeeLookup)
	}

	// Otherwise parse the network address
	if addrLookup == "" {
		return 0, fmt.Errorf("one of --addr, --ieee, or --name must be specified")
	}
	return parseNetworkAddr(addrLookup)
}

// parseClusterID parses hex cluster ID (e.g., "0x0006" or "6").
func parseClusterID(s string) (uint16, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	val, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid cluster ID %q: %w", s, err)
	}
	return uint16(val), nil
}

var (
	bindCluster   string
	bindUnbind    bool
	reportCluster string
	reportAttr    string
	reportMin     uint16
	reportMax     uint16
)

// goznp device bind -p <port> --addr <nwk-addr> --cluster <cluster-id> [--unbind]
var deviceBindCmd = &cobra.Command{
	Use:   "bind",
	Short: "Bind device cluster to coordinator",
	Long: `Create or remove a binding from a device cluster to the coordinator.

Binding tells a device where to send attribute reports. Without binding,
devices don't know where to send automatic reports even if reporting is configured.

Common clusters:
  0x0006 - OnOff (switch state)
  0x0702 - SimpleMetering (energy)
  0x0B04 - ElectricalMeasurement (power/voltage/current)
  0x0402 - Temperature
  0x0405 - Humidity

Example:
  goznp device bind -p /dev/ttyUSB0 --addr 0xB120 --cluster 0x0006`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceBind(ctx)
	},
}

func runDeviceBind(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	clusterID, err := parseClusterID(bindCluster)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Get device list to find IEEE address
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device list: %w", err)
	}

	var deviceIEEEAddr [8]byte
	found := false
	for _, dev := range devices {
		if dev.NwkAddr == nwkAddr {
			deviceIEEEAddr = dev.IEEEAddr
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("device 0x%04X not found in paired devices", nwkAddr)
	}

	// Get coordinator IEEE address
	info, err := a.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get adapter info: %w", err)
	}

	// Get coordinator IEEE from network info
	coordIEEE, err := a.GetCoordinatorIEEE(ctx)
	if err != nil {
		return fmt.Errorf("failed to get coordinator IEEE: %w", err)
	}

	action := "Binding"
	if bindUnbind {
		action = "Unbinding"
	}

	fmt.Printf("%s device 0x%04X (%s) cluster 0x%04X to coordinator (%s)...\n",
		action,
		nwkAddr,
		znp.FormatIEEEAddr(deviceIEEEAddr),
		clusterID,
		znp.FormatIEEEAddr(coordIEEE),
	)
	_ = info // Used for debug if needed

	if bindUnbind {
		err = a.Unbind(ctx, deviceIEEEAddr, nwkAddr, deviceEndpoint, clusterID, coordIEEE, 1)
	} else {
		err = a.Bind(ctx, deviceIEEEAddr, nwkAddr, deviceEndpoint, clusterID, coordIEEE, 1)
	}

	if err != nil {
		return fmt.Errorf("%s failed: %w", strings.ToLower(action), err)
	}

	fmt.Printf("%s successful!\n", action)
	return nil
}

// goznp device configure-reporting -p <port> --addr <nwk-addr> --cluster <cluster-id> --attr <attr-id> --min <seconds> --max <seconds>
var deviceConfigureReportingCmd = &cobra.Command{
	Use:   "configure-reporting",
	Short: "Configure attribute reporting",
	Long: `Configure a device to automatically report attribute changes.

This tells the device when to send attribute reports:
  --min: Minimum interval between reports (seconds)
  --max: Maximum interval between reports (seconds, 0xFFFF=no periodic)

Note: You must also bind the cluster to the coordinator for reports to be received.

Common cluster/attribute combinations:
  OnOff (0x0006):
    0x0000 - OnOff state
  ElectricalMeasurement (0x0B04):
    0x0505 - RMS Voltage
    0x0508 - RMS Current
    0x050B - Active Power
  SimpleMetering (0x0702):
    0x0000 - Current Summation (energy)

Example:
  goznp device configure-reporting -p /dev/ttyUSB0 --addr 0xB120 --cluster 0x0006 --attr 0x0000 --min 1 --max 300`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceConfigureReporting(ctx)
	},
}

func runDeviceConfigureReporting(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	clusterID, err := parseClusterID(reportCluster)
	if err != nil {
		return err
	}

	attrID, err := parseClusterID(reportAttr) // Same format as cluster
	if err != nil {
		return fmt.Errorf("invalid attribute ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Determine data type based on known attributes
	dataType := getDataTypeForAttribute(zcl.ClusterID(clusterID), zcl.AttributeID(attrID))

	fmt.Printf("Configuring reporting on device 0x%04X endpoint %d:\n", nwkAddr, deviceEndpoint)
	fmt.Printf("  Cluster:   0x%04X (%s)\n", clusterID, zcl.ClusterID(clusterID))
	fmt.Printf("  Attribute: 0x%04X\n", attrID)
	fmt.Printf("  Data Type: 0x%02X (%s)\n", uint8(dataType), dataType)
	fmt.Printf("  Min:       %d seconds\n", reportMin)
	fmt.Printf("  Max:       %d seconds\n", reportMax)

	err = a.ConfigureReporting(ctx, nwkAddr, deviceEndpoint,
		zcl.ClusterID(clusterID),
		zcl.AttributeID(attrID),
		dataType,
		reportMin, reportMax,
		nil, // No reportable change threshold for now
	)

	if err != nil {
		return fmt.Errorf("configure reporting failed: %w", err)
	}

	fmt.Println("Configure reporting successful!")
	fmt.Println("\nNote: Make sure the cluster is bound to the coordinator:")
	fmt.Printf("  goznp device bind -p %s --addr 0x%04X --cluster 0x%04X\n", portPath, nwkAddr, clusterID)
	return nil
}

// goznp device reporting -p <port> (--addr <nwk-addr> | --ieee <ieee-addr>) --cluster <cluster-id> --attr <attr-id>
var deviceReportingCmd = &cobra.Command{
	Use:   "reporting",
	Short: "Read reporting configuration",
	Long: `Query a device's current reporting configuration for an attribute.

This shows how the device is configured to send attribute reports:
  - Min Interval: Minimum seconds between reports (even if value changes)
  - Max Interval: Maximum seconds between reports (even if value doesn't change)
  - Reportable Change: How much the value must change to trigger a report

Common cluster/attribute combinations:
  OnOff (0x0006):
    0x0000 - OnOff state
  ElectricalMeasurement (0x0B04):
    0x0505 - RMS Voltage
    0x0508 - RMS Current
    0x050B - Active Power
  SimpleMetering (0x0702):
    0x0000 - Current Summation (energy)

Examples:
  goznp device reporting --addr 0xE2CA --cluster 0x0702 --attr 0x0000
  goznp device reporting --ieee 00:11:22:33:44:55:66:77 --cluster 0x0006 --attr 0x0000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceReporting(ctx)
	},
}

func runDeviceReporting(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Validate that at least one address type is provided
	if deviceAddr == "" && deviceIEEE == "" {
		return fmt.Errorf("either --addr or --ieee must be specified")
	}

	clusterID, err := parseClusterID(reportCluster)
	if err != nil {
		return err
	}

	attrID, err := parseClusterID(reportAttr) // Same format as cluster
	if err != nil {
		return fmt.Errorf("invalid attribute ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Resolve network address
	var nwkAddr uint16
	if deviceAddr != "" {
		nwkAddr, err = parseNetworkAddr(deviceAddr)
		if err != nil {
			return err
		}
	} else {
		// Look up by IEEE address from paired devices
		ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
		if err != nil {
			return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
		}

		devices, err := a.GetDevices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get paired devices: %w", err)
		}

		var found bool
		for _, dev := range devices {
			if dev.IEEEAddr == ieeeAddr {
				nwkAddr = dev.NwkAddr
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("device %s not found in paired devices", znp.FormatIEEEAddr(ieeeAddr))
		}
		fmt.Printf("Resolved IEEE %s to network address 0x%04X\n", deviceIEEE, nwkAddr)
	}

	fmt.Printf("Reading reporting config from device 0x%04X endpoint %d:\n", nwkAddr, deviceEndpoint)
	fmt.Printf("  Cluster:   0x%04X (%s)\n", clusterID, zcl.ClusterID(clusterID))
	fmt.Printf("  Attribute: 0x%04X\n", attrID)
	fmt.Println()

	results, err := a.ReadReportingConfig(ctx, nwkAddr, deviceEndpoint, zcl.ClusterID(clusterID), zcl.AttributeID(attrID))
	if err != nil {
		return fmt.Errorf("read reporting config failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No reporting configuration returned")
		return nil
	}

	for _, r := range results {
		fmt.Printf("Attribute 0x%04X:\n", r.AttributeID)
		if r.Status != 0x00 {
			fmt.Printf("  Status: 0x%02X (error)\n", r.Status)
			continue
		}

		if r.Direction == 0x00 {
			// Outgoing reports (device -> coordinator)
			fmt.Printf("  Direction:    Reported (device sends reports)\n")
			fmt.Printf("  Data Type:    0x%02X (%s)\n", uint8(r.DataType), r.DataType)
			fmt.Printf("  Min Interval: %d seconds\n", r.MinInterval)
			if r.MaxInterval == 0xFFFF {
				fmt.Printf("  Max Interval: disabled (no periodic reports)\n")
			} else {
				fmt.Printf("  Max Interval: %d seconds\n", r.MaxInterval)
			}
			if r.ReportableChange != nil {
				fmt.Printf("  Reportable Change: %v\n", r.ReportableChange)
			}
		} else {
			// Incoming reports (coordinator -> device)
			fmt.Printf("  Direction:      Received (device receives reports)\n")
			fmt.Printf("  Timeout Period: %d seconds\n", r.TimeoutPeriod)
		}
	}

	return nil
}

// goznp device bindings -p <port> --addr <nwk-addr>
var deviceBindingsCmd = &cobra.Command{
	Use:   "bindings",
	Short: "List device bindings",
	Long: `Display the binding table from a device.

The binding table shows where the device will send reports for each cluster.
Use this to verify that clusters are properly bound to the coordinator.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceBindings(ctx)
	},
}

func runDeviceBindings(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Querying binding table from device 0x%04X...\n\n", nwkAddr)

	bindings, err := a.GetBindingTable(ctx, nwkAddr)
	if err != nil {
		return fmt.Errorf("failed to get binding table: %w", err)
	}

	if len(bindings) == 0 {
		fmt.Println("No bindings found on device")
		fmt.Println("\nTip: Use 'goznp device bind' to create bindings")
		return nil
	}

	fmt.Printf("Found %d binding(s):\n\n", len(bindings))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Endpoint\tCluster\tDestination\tDst Endpoint")
	fmt.Fprintln(w, "--------\t-------\t-----------\t------------")

	for _, b := range bindings {
		dstStr := znp.FormatIEEEAddr(b.DstAddr)
		if b.DstAddrMode == 0x01 {
			// Group address - first 2 bytes
			groupID := uint16(b.DstAddr[0]) | uint16(b.DstAddr[1])<<8
			dstStr = fmt.Sprintf("Group 0x%04X", groupID)
		}

		fmt.Fprintf(w, "%d\t%s (0x%04X)\t%s\t%d\n",
			b.SrcEndpoint,
			zcl.ClusterID(b.ClusterID),
			b.ClusterID,
			dstStr,
			b.DstEndpoint,
		)
	}
	w.Flush()

	return nil
}

// ============================================================================
// Device Name Commands
// ============================================================================

// Parent command for device name operations
var deviceNameCmd = &cobra.Command{
	Use:   "name",
	Short: "Manage custom device names",
	Long:  "Set, get, list, or delete custom names and descriptions for devices",
}

// goznp device name set --ieee <addr> --name <name> [--description <desc>]
var deviceNameSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set custom name for a device",
	Long: `Set a custom name and optional description for a device.

The custom name is stored on the coordinator and associated with the device's
IEEE address. This name persists across network restarts.

Examples:
  goznp device name set --ieee 00:11:22:33:44:55:66:77 --name "Living Room Light"
  goznp device name set --ieee 00:11:22:33:44:55:66:77 --name "Kitchen Sensor" --description "Temperature and humidity sensor"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameSet(ctx)
	},
}

func runDeviceNameSet(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if deviceIEEE == "" {
		return fmt.Errorf("--ieee is required")
	}

	if deviceName == "" {
		return fmt.Errorf("--name is required")
	}

	ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
	if err != nil {
		return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Setting custom name for device %s...\n", znp.FormatIEEEAddr(ieeeAddr))

	if err := a.SetDeviceName(ctx, ieeeAddr, deviceName, deviceDescription); err != nil {
		return fmt.Errorf("failed to set device name: %w", err)
	}

	fmt.Printf("Device name set to: %s\n", deviceName)
	if deviceDescription != "" {
		fmt.Printf("Description: %s\n", deviceDescription)
	}

	return nil
}

// goznp device name get --ieee <addr>
var deviceNameGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get custom name for a device",
	Long: `Get the custom name and description for a device.

Example:
  goznp device name get --ieee 00:11:22:33:44:55:66:77`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameGet(ctx)
	},
}

func runDeviceNameGet(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if deviceIEEE == "" {
		return fmt.Errorf("--ieee is required")
	}

	ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
	if err != nil {
		return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	info, err := a.GetDeviceName(ctx, ieeeAddr)
	if err != nil {
		return fmt.Errorf("failed to get device name: %w", err)
	}

	if info == nil || info.Name == "" {
		fmt.Println("No custom name set")
		return nil
	}

	fmt.Printf("Device: %s\n", znp.FormatIEEEAddr(ieeeAddr))
	fmt.Printf("Name: %s\n", info.Name)
	if info.Description != "" {
		fmt.Printf("Description: %s\n", info.Description)
	}

	return nil
}

// goznp device name list
var deviceNameListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices with custom names",
	Long: `List all devices that have custom names set.

Example:
  goznp device name list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameList(ctx)
	},
}

func runDeviceNameList(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	names, err := a.ListDeviceNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to list device names: %w", err)
	}

	if len(names) == 0 {
		fmt.Println("No devices with custom names")
		return nil
	}

	fmt.Printf("Found %d device(s) with custom names:\n\n", len(names))

	// Create a tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IEEE Address\tName\tDescription")
	fmt.Fprintln(w, "------------\t----\t-----------")

	for _, device := range names {
		desc := device.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			znp.FormatIEEEAddr(device.IEEEAddr),
			device.Name,
			desc,
		)
	}

	w.Flush()

	return nil
}

// goznp device name delete --ieee <addr>
var deviceNameDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete custom name for a device",
	Long: `Delete the custom name and description for a device.

Example:
  goznp device name delete --ieee 00:11:22:33:44:55:66:77`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameDelete(ctx)
	},
}

func runDeviceNameDelete(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if deviceIEEE == "" {
		return fmt.Errorf("--ieee is required")
	}

	ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
	if err != nil {
		return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Deleting custom name for device %s...\n", znp.FormatIEEEAddr(ieeeAddr))

	if err := a.DeleteDeviceName(ctx, ieeeAddr); err != nil {
		return fmt.Errorf("failed to delete device name: %w", err)
	}

	fmt.Println("Device name deleted")

	return nil
}

// getDataTypeForAttribute returns the ZCL data type for known attributes.
func getDataTypeForAttribute(cluster zcl.ClusterID, attr zcl.AttributeID) zcl.DataType {
	switch cluster {
	case zcl.ClusterOnOff:
		if attr == zcl.AttrOnOff {
			return zcl.TypeBoolean
		}
	case zcl.ClusterElectricalMeas:
		switch attr {
		case zcl.AttrElecRMSVoltage, zcl.AttrElecRMSCurrent:
			return zcl.TypeUint16
		case zcl.AttrElecActivePower:
			return zcl.TypeInt16
		}
	case zcl.ClusterMeteringSimple:
		if attr == zcl.AttrMeterCurrentSummation {
			return zcl.TypeUint48
		}
	case zcl.ClusterTempMeasurement:
		if attr == zcl.AttrTempMeasuredValue {
			return zcl.TypeInt16
		}
	case zcl.ClusterHumidityMeas:
		if attr == zcl.AttrHumidityMeasuredValue {
			return zcl.TypeUint16
		}
	}
	// Default to boolean for unknown
	return zcl.TypeBoolean
}

func init() {
	// Device permit flags
	devicePermitCmd.Flags().Uint8Var(&permitDuration, "duration", 60, "Join window duration in seconds (0=close, 255=always)")

	// Device pair flags
	devicePairCmd.Flags().IntVar(&pairTimeout, "timeout", 180, "Pairing timeout in seconds")
	devicePairCmd.Flags().BoolVar(&pairNoBind, "nobind", false, "Skip automatic binding to coordinator")

	// Device watch flags
	deviceWatchCmd.Flags().Uint8Var(&permitDuration, "duration", 60, "Join window duration in seconds")

	// Device list flags
	deviceListCmd.Flags().BoolVarP(&interviewDevices, "interview", "i", false, "Perform full device interview (slower but more info)")

	// Device status/control flags
	deviceStatusCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceStatusCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceStatusCmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
	deviceStatusCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Add --addr and --endpoint to control commands
	for _, cmd := range []*cobra.Command{deviceOnCmd, deviceOffCmd, deviceToggleCmd} {
		cmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
		cmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
		cmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
		cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	}

	// Device brightness flags
	deviceBrightnessCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceBrightnessCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceBrightnessCmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
	deviceBrightnessCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceBrightnessCmd.Flags().Uint8Var(&brightnessLevel, "level", 0, "Brightness level (0-254, 0=off, 254=max)")
	deviceBrightnessCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	// Device color-temp flags
	deviceColorTempCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceColorTempCmd.MarkFlagRequired("addr")
	deviceColorTempCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceColorTempCmd.Flags().Uint16Var(&colorTempKelvin, "kelvin", 0, "Color temperature in Kelvin (2700-6500)")
	deviceColorTempCmd.Flags().Uint16Var(&colorTempMireds, "mireds", 0, "Color temperature in mireds (154-370)")
	deviceColorTempCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	// Device identify flags
	deviceIdentifyCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceIdentifyCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceIdentifyCmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
	deviceIdentifyCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceIdentifyCmd.Flags().Uint16Var(&identifyDuration, "duration", 0, "Identify duration in seconds (default: 5)")
	deviceIdentifyCmd.Flags().StringVar(&identifyEffect, "effect", "", "Trigger effect (blink, breathe, okay, channel, stop)")

	// Device group flags
	for _, cmd := range []*cobra.Command{deviceGroupAddCmd, deviceGroupRemoveCmd, deviceGroupListCmd} {
		cmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
		cmd.MarkFlagRequired("addr")
		cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	}
	deviceGroupAddCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
	deviceGroupAddCmd.MarkFlagRequired("group")
	deviceGroupAddCmd.Flags().StringVar(&groupName, "name", "", "Optional group name (max 16 chars)")
	deviceGroupRemoveCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID to remove from")
	deviceGroupRemoveCmd.Flags().BoolVar(&removeAllGroups, "all", false, "Remove from all groups")

	// Device group command flags (for on/off/toggle/brightness)
	for _, cmd := range []*cobra.Command{deviceGroupOnCmd, deviceGroupOffCmd, deviceGroupToggleCmd} {
		cmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
		cmd.MarkFlagRequired("group")
	}
	deviceGroupBrightnessCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
	deviceGroupBrightnessCmd.MarkFlagRequired("group")
	deviceGroupBrightnessCmd.Flags().Uint8Var(&brightnessLevel, "level", 0, "Brightness level (0-254)")
	deviceGroupBrightnessCmd.MarkFlagRequired("level")
	deviceGroupBrightnessCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")
	deviceGroupSceneCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
	deviceGroupSceneCmd.MarkFlagRequired("group")
	deviceGroupSceneCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	deviceGroupSceneCmd.MarkFlagRequired("scene")

	// Device scene flags
	for _, cmd := range []*cobra.Command{deviceSceneStoreCmd, deviceSceneRecallCmd, deviceSceneRemoveCmd, deviceSceneListCmd} {
		cmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
		cmd.MarkFlagRequired("addr")
		cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
		cmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (0 for global scenes)")
		cmd.MarkFlagRequired("group")
	}
	deviceSceneStoreCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	deviceSceneStoreCmd.MarkFlagRequired("scene")
	deviceSceneRecallCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	deviceSceneRecallCmd.MarkFlagRequired("scene")
	deviceSceneRecallCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")
	deviceSceneRemoveCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID to remove")
	deviceSceneRemoveCmd.Flags().BoolVar(&removeAllScenes, "all", false, "Remove all scenes for the group")

	// Build group subcommand hierarchy
	deviceGroupCmd.AddCommand(deviceGroupAddCmd)
	deviceGroupCmd.AddCommand(deviceGroupRemoveCmd)
	deviceGroupCmd.AddCommand(deviceGroupListCmd)
	deviceGroupCmd.AddCommand(deviceGroupOnCmd)
	deviceGroupCmd.AddCommand(deviceGroupOffCmd)
	deviceGroupCmd.AddCommand(deviceGroupToggleCmd)
	deviceGroupCmd.AddCommand(deviceGroupBrightnessCmd)
	deviceGroupCmd.AddCommand(deviceGroupSceneCmd)

	// Build scene subcommand hierarchy
	deviceSceneCmd.AddCommand(deviceSceneStoreCmd)
	deviceSceneCmd.AddCommand(deviceSceneRecallCmd)
	deviceSceneCmd.AddCommand(deviceSceneRemoveCmd)
	deviceSceneCmd.AddCommand(deviceSceneListCmd)

	// Device sensor flags
	deviceSensorCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceSensorCmd.MarkFlagRequired("addr")
	deviceSensorCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Device info flags
	deviceInfoCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceInfoCmd.MarkFlagRequired("addr")

	// Device power flags
	devicePowerCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	devicePowerCmd.MarkFlagRequired("addr")
	devicePowerCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Device reset-energy flags
	deviceResetEnergyCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceResetEnergyCmd.MarkFlagRequired("addr")
	deviceResetEnergyCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Device remove flags
	deviceRemoveCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceRemoveCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceRemoveCmd.Flags().BoolVar(&deviceForce, "force", false, "Force remove even if device doesn't respond")

	// Device bind flags
	deviceBindCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceBindCmd.MarkFlagRequired("addr")
	deviceBindCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceBindCmd.Flags().StringVar(&bindCluster, "cluster", "", "Cluster ID (hex, e.g., 0x0006)")
	deviceBindCmd.MarkFlagRequired("cluster")
	deviceBindCmd.Flags().BoolVar(&bindUnbind, "unbind", false, "Remove binding instead of creating")

	// Device configure-reporting flags
	deviceConfigureReportingCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceConfigureReportingCmd.MarkFlagRequired("addr")
	deviceConfigureReportingCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceConfigureReportingCmd.Flags().StringVar(&reportCluster, "cluster", "", "Cluster ID (hex, e.g., 0x0006)")
	deviceConfigureReportingCmd.MarkFlagRequired("cluster")
	deviceConfigureReportingCmd.Flags().StringVar(&reportAttr, "attr", "", "Attribute ID (hex, e.g., 0x0000)")
	deviceConfigureReportingCmd.MarkFlagRequired("attr")
	deviceConfigureReportingCmd.Flags().Uint16Var(&reportMin, "min", 1, "Minimum reporting interval (seconds)")
	deviceConfigureReportingCmd.Flags().Uint16Var(&reportMax, "max", 300, "Maximum reporting interval (seconds)")

	// Device reporting (read) flags
	deviceReportingCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceReportingCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceReportingCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceReportingCmd.Flags().StringVar(&reportCluster, "cluster", "", "Cluster ID (hex, e.g., 0x0006)")
	deviceReportingCmd.MarkFlagRequired("cluster")
	deviceReportingCmd.Flags().StringVar(&reportAttr, "attr", "", "Attribute ID (hex, e.g., 0x0000)")
	deviceReportingCmd.MarkFlagRequired("attr")

	// Device bindings flags
	deviceBindingsCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceBindingsCmd.MarkFlagRequired("addr")

	// Device name flags
	deviceNameSetCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceNameSetCmd.MarkFlagRequired("ieee")
	deviceNameSetCmd.Flags().StringVar(&deviceName, "name", "", "Custom device name")
	deviceNameSetCmd.MarkFlagRequired("name")
	deviceNameSetCmd.Flags().StringVar(&deviceDescription, "description", "", "Optional device description")

	deviceNameGetCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceNameGetCmd.MarkFlagRequired("ieee")

	deviceNameDeleteCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceNameDeleteCmd.MarkFlagRequired("ieee")

	// Build name subcommand hierarchy
	deviceNameCmd.AddCommand(deviceNameSetCmd)
	deviceNameCmd.AddCommand(deviceNameGetCmd)
	deviceNameCmd.AddCommand(deviceNameListCmd)
	deviceNameCmd.AddCommand(deviceNameDeleteCmd)

	// Add port/baud flags to all device commands
	for _, cmd := range []*cobra.Command{
		devicePermitCmd, devicePairCmd, deviceListCmd, deviceWatchCmd, deviceStatusCmd,
		deviceOnCmd, deviceOffCmd, deviceToggleCmd, deviceBrightnessCmd, deviceColorTempCmd,
		deviceIdentifyCmd,
		deviceGroupAddCmd, deviceGroupRemoveCmd, deviceGroupListCmd,
		deviceGroupOnCmd, deviceGroupOffCmd, deviceGroupToggleCmd, deviceGroupBrightnessCmd, deviceGroupSceneCmd,
		deviceSceneStoreCmd, deviceSceneRecallCmd, deviceSceneRemoveCmd, deviceSceneListCmd,
		deviceSensorCmd, deviceListenCmd, deviceInfoCmd, devicePowerCmd, deviceResetEnergyCmd,
		deviceRemoveCmd, deviceBindCmd, deviceConfigureReportingCmd, deviceReportingCmd, deviceBindingsCmd,
		deviceNameSetCmd, deviceNameGetCmd, deviceNameListCmd, deviceNameDeleteCmd,
	} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Build device command hierarchy
	deviceCmd.AddCommand(devicePermitCmd)
	deviceCmd.AddCommand(devicePairCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceWatchCmd)
	deviceCmd.AddCommand(deviceStatusCmd)
	deviceCmd.AddCommand(deviceOnCmd)
	deviceCmd.AddCommand(deviceOffCmd)
	deviceCmd.AddCommand(deviceToggleCmd)
	deviceCmd.AddCommand(deviceBrightnessCmd)
	deviceCmd.AddCommand(deviceColorTempCmd)
	deviceCmd.AddCommand(deviceColorCmd)
	deviceCmd.AddCommand(deviceIdentifyCmd)
	deviceCmd.AddCommand(deviceGroupCmd)
	deviceCmd.AddCommand(deviceSceneCmd)
	deviceCmd.AddCommand(deviceSensorCmd)
	deviceCmd.AddCommand(deviceListenCmd)
	deviceCmd.AddCommand(deviceInfoCmd)
	deviceCmd.AddCommand(devicePowerCmd)
	deviceCmd.AddCommand(deviceResetEnergyCmd)
	deviceCmd.AddCommand(deviceRemoveCmd)
	deviceCmd.AddCommand(deviceBindCmd)
	deviceCmd.AddCommand(deviceConfigureReportingCmd)
	deviceCmd.AddCommand(deviceReportingCmd)
	deviceCmd.AddCommand(deviceBindingsCmd)
	deviceCmd.AddCommand(deviceNameCmd)
}

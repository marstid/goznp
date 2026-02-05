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

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	devicedb "github.com/marstid/goznp/pkg/devices"
	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
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

	// Brightness/level control flags.
	brightnessLevel uint8
	transitionTime  uint16

	// Identify flags.
	identifyDuration uint16
	identifyEffect   string

	// Group flags.
	groupID   uint16
	groupName string

	// Scene flags.
	sceneID uint8

	// Device name flags.
	deviceName       string
	deviceComment    string
	deviceNameLookup string // For --name flag.
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Device management commands",
	Long:  "Commands for managing paired Zigbee devices",
}

// goznp device permit -p <port> [--duration <seconds>].
var devicePermitCmd = &cobra.Command{
	Use:   "permit",
	Short: "Open network for device pairing",
	Long: `Open the network for devices to join.

This opens the network for the specified duration (default: 60 seconds) allowing
devices to join. Put your device in pairing mode (usually by pressing and holding
a button for 3-5 seconds) before running this command.

Special values for --duration:
  0   - Close network for pairing
  255 - Open network permanently (not recommended for security)

For automated pairing with device interviews, use 'device pair' instead.

Examples:
  # Open network for 60 seconds (default)
  goznp device permit

  # Open network for 5 minutes
  goznp device permit --duration 300

  # Open network permanently
  goznp device permit --duration 255

  # Close network for pairing
  goznp device permit --duration 0`,
	RunE: func(_ *cobra.Command, _ []string) error {
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

	switch permitDuration {
	case 0:
		fmt.Println("Closing network for pairing...")
	case 255:
		fmt.Println("Opening network for pairing (always open)...")
	default:
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

// goznp device pair -p <port> [--timeout <seconds>] [--nobind].
var devicePairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Pair a new device",
	Long: `Open network for pairing, watch for device joins, interview, and bind.

This command opens the network for device pairing for the specified timeout
(default: 180 seconds), actively monitors for joining devices, and then
automatically closes the network when done.

What happens during pairing:
  1. Network opens for devices to join
  2. When a device joins, it's automatically interviewed
  3. Device clusters are bound to the coordinator for automatic reporting
  4. Network closes after timeout or when done

Use --nobind to skip automatic binding (useful if you want to manually configure bindings).

Examples:
  # Pair a device with defaults (3 minute timeout, auto-bind)
  goznp device pair

  # Pair with 10 minute timeout
  goznp device pair --timeout 600

  # Pair without automatic binding
  goznp device pair --nobind

  # Pair for 5 minutes without binding
  goznp device pair --timeout 300 --nobind

Note: Put your device in pairing mode (usually hold button for 3-5 seconds) before running.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDevicePair(ctx)
	},
}

func runDevicePair(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create timeout context.
	timeout := time.Duration(pairTimeout) * time.Second
	pairCtx, cancel := context.WithTimeout(ctx, timeout+60*time.Second) // Extra time for interview/bind.
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(pairCtx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Get coordinator IEEE for binding (do this once upfront).
	var coordIEEE [8]byte
	var coordErr error
	if !pairNoBind {
		coordIEEE, coordErr = a.GetCoordinatorIEEE(pairCtx)
		if coordErr != nil {
			fmt.Printf("Warning: Could not get coordinator IEEE address: %v\n", coordErr)
			fmt.Println("Auto-bind will be skipped")
		}
	}

	// Track results.
	var resultsMu sync.Mutex
	type pairResult struct {
		device      *adapter.Device
		interviewed bool
		bound       int
		configured  int
		errors      []string
	}
	results := make([]pairResult, 0)

	// Deduplication: track recently-seen IEEE addresses to avoid processing
	// duplicate join events (devices can send multiple TcDeviceInd messages).
	const deduplicationCooldown = 10 * time.Second
	recentJoins := make(map[[8]byte]time.Time)

	// Set up device event handler - interview and bind IMMEDIATELY on join.
	a.OnDeviceEvent(func(event adapter.DeviceEvent) {
		if event.Type != adapter.DeviceEventJoined || event.Device == nil {
			return
		}

		dev := event.Device
		now := time.Now()

		// Check for duplicate join event (same IEEE address within cooldown period).
		resultsMu.Lock()
		if lastSeen, exists := recentJoins[dev.IEEEAddr]; exists {
			if now.Sub(lastSeen) < deduplicationCooldown {
				resultsMu.Unlock()
				// Silently ignore duplicate - already being processed.
				return
			}
		}
		recentJoins[dev.IEEEAddr] = now
		resultsMu.Unlock()

		timestamp := now.Format("15:04:05")
		fmt.Printf("\n[%s] Device JOINED: %s (0x%04X)\n",
			timestamp,
			znp.FormatIEEEAddr(dev.IEEEAddr),
			dev.NwkAddr,
		)

		result := pairResult{device: dev}

		// Brief delay to let device fully initialize after joining.
		// Some devices send TcDeviceInd before they're ready to respond to ZDO queries.
		time.Sleep(2 * time.Second)

		// Interview while device is still awake.
		fmt.Printf("  Interviewing (device must stay awake)...\n")
		interviewCtx, interviewCancel := context.WithTimeout(context.Background(), 15*time.Second)
		interview, err := a.InterviewDeviceWithAddr(interviewCtx, dev.NwkAddr, dev.IEEEAddr)
		interviewCancel()

		// Retry once if no endpoints discovered (device may need more init time).
		if err == nil && len(interview.Endpoints) == 0 {
			fmt.Printf("  No endpoints yet, retrying in 3s...\n")
			time.Sleep(3 * time.Second)
			interviewCtx, interviewCancel = context.WithTimeout(context.Background(), 15*time.Second)
			interview, err = a.InterviewDeviceWithAddr(interviewCtx, dev.NwkAddr, dev.IEEEAddr)
			interviewCancel()
		}

		//nolint:gocritic // Complex interview logic with multiple conditions best expressed as if-else
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

			// IMMEDIATELY bind while device is still awake.
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

							// Configure reporting immediately while device is awake.
							// Only for clusters that support attribute reporting.
							if shouldConfigureReporting(clusterID) {
								configCtx, configCancel := context.WithTimeout(context.Background(), 5*time.Second)
								configErr := configureSensorReporting(configCtx, a, dev.NwkAddr, ep.Endpoint, clusterID)
								configCancel()
								if configErr != nil {
									result.errors = append(result.errors, fmt.Sprintf("configure %s: %v", clusterName, configErr))
									fmt.Printf("  Configure %s: FAILED (%v)\n", clusterName, configErr)
								} else {
									result.configured++
									fmt.Printf("  Configure %s: OK\n", clusterName)
								}
							}
						}
					}
				}
			}
		}

		resultsMu.Lock()
		// Replace existing entry for same IEEE address (device may rejoin with new nwk addr).
		replaced := false
		for i, r := range results {
			if r.device.IEEEAddr == dev.IEEEAddr {
				results[i] = result
				replaced = true
				break
			}
		}
		if !replaced {
			results = append(results, result)
		}
		resultsMu.Unlock()

		fmt.Printf("  Done! Waiting for more devices...\n")
	})

	// Open permit join.
	fmt.Printf("Opening network for pairing (%d seconds)...\n", pairTimeout)
	fmt.Println("Waiting for devices to join... (Ctrl+C to stop early)")
	switch {
	case pairNoBind:
		fmt.Println("Auto-bind: DISABLED (--nobind)")
	case coordErr != nil:
		fmt.Println("Auto-bind: DISABLED (coordinator error)")
	default:
		fmt.Println("Auto-bind: ENABLED (interview & bind immediately on join)")
	}
	fmt.Println()

	// Calculate permit duration (max 254 for permit join command).
	permitDur := uint8(254)
	if pairTimeout < 254 {
		permitDur = uint8(pairTimeout)
	}

	if err := a.PermitJoin(pairCtx, permitDur); err != nil {
		return fmt.Errorf("failed to open network: %w", err)
	}

	// Create timer for the pairing window.
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Wait for timeout or context cancellation.
	select {
	case <-timer.C:
		fmt.Println("\n\nPairing window closed (timeout)")
	case <-ctx.Done():
		fmt.Println("\n\nPairing interrupted by user")
	}

	// Close permit join.
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	//nolint:errcheck // Best effort cleanup
	_ = a.PermitJoin(closeCtx, 0)

	// Summary.
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
		if r.configured > 0 {
			fmt.Printf("    Configured %d cluster(s) for reporting\n", r.configured)
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
	case zcl.ClusterOnOff, // Switch state.
		zcl.ClusterLevelControl,     // Dimmer level.
		zcl.ClusterColorControl,     // Color/temperature.
		zcl.ClusterTempMeasurement,  // Temperature sensor.
		zcl.ClusterHumidityMeas,     // Humidity sensor.
		zcl.ClusterPressureMeas,     // Pressure sensor.
		zcl.ClusterIlluminanceMeas,  // Light sensor.
		zcl.ClusterOccupancySensing, // Motion sensor.
		zcl.ClusterElectricalMeas,   // Power monitoring.
		zcl.ClusterMeteringSimple,   // Energy metering.
		zcl.ClusterPowerConfig,      // Battery level.
		zcl.ClusterIASZone:          // Security sensors.
		return true
	}
	return false
}

// shouldConfigureReporting returns true if we should configure reporting for this cluster.
func shouldConfigureReporting(clusterID uint16) bool {
	switch zcl.ClusterID(clusterID) {
	case zcl.ClusterTempMeasurement,
		zcl.ClusterHumidityMeas,
		zcl.ClusterPowerConfig,
		zcl.ClusterOnOff,
		zcl.ClusterElectricalMeas:
		return true
	}
	return false
}

// configureSensorReporting configures attribute reporting for common sensor clusters.
// This must be called after binding while the device is still awake.
// Returns nil if the cluster doesn't need reporting configuration.
func configureSensorReporting(ctx context.Context, a *adapter.Adapter, nwkAddr uint16, endpoint uint8, clusterID uint16) error {
	switch zcl.ClusterID(clusterID) {
	case zcl.ClusterTempMeasurement:
		// Temperature: report every 30-3600s or on 0.2°C change.
		return a.ConfigureReporting(ctx, nwkAddr, endpoint, zcl.ClusterID(clusterID),
			zcl.AttrTempMeasuredValue, zcl.TypeInt16, 30, 3600, int16(20))

	case zcl.ClusterHumidityMeas:
		// Humidity: report every 30-3600s or on 1% change.
		return a.ConfigureReporting(ctx, nwkAddr, endpoint, zcl.ClusterID(clusterID),
			zcl.AttrHumidityMeasuredValue, zcl.TypeUint16, 30, 3600, uint16(100))

	case zcl.ClusterPowerConfig:
		// Battery: report every 3600-65000s (battery changes slowly).
		return a.ConfigureReporting(ctx, nwkAddr, endpoint, zcl.ClusterID(clusterID),
			zcl.AttrPowerBatteryPercentage, zcl.TypeUint8, 3600, 65000, uint8(10))

	case zcl.ClusterOnOff:
		// On/Off: report immediately on change, max every hour
		return a.ConfigureReporting(ctx, nwkAddr, endpoint, zcl.ClusterID(clusterID),
			zcl.AttrOnOff, zcl.TypeBoolean, 0, 3600, nil)

	case zcl.ClusterElectricalMeas:
		// Power monitoring: report every 10-300s or on significant change
		return a.ConfigureReporting(ctx, nwkAddr, endpoint, zcl.ClusterID(clusterID),
			zcl.AttrElecActivePower, zcl.TypeInt16, 10, 300, int16(5))

	default:
		// Cluster doesn't need automatic reporting config
		return nil
	}
}

// goznp device list -p <port> [--interview]
var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List paired devices",
	Long: `Display all devices currently paired to the network.

Shows basic information by default (network address, IEEE address, name).
Use --interview to perform full device discovery including manufacturer,
model, power source, and supported clusters for each device (slower but more detailed).

Examples:
  # List all devices (basic info)
  goznp device list

  # List devices with full interview details
  goznp device list --interview`,
	RunE: func(_ *cobra.Command, _ []string) error {
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
	//nolint:errcheck // Names are optional, errors intentionally ignored
	deviceNames, _ := a.ListDeviceNames(ctx)
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

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}

	fmt.Println("\nTip: Use --interview for detailed device information (manufacturer, model, clusters)")

	return nil
}

func runDeviceListWithInterview(ctx context.Context, a *adapter.Adapter, devices []*adapter.Device) error {
	fmt.Printf("Interviewing %d device(s)...\n\n", len(devices))

	// Fetch custom device names
	//nolint:errcheck // Names are optional, errors intentionally ignored
	deviceNames, _ := a.ListDeviceNames(ctx)
	namesByIEEE := make(map[[8]byte]string)
	for _, dn := range deviceNames {
		namesByIEEE[dn.IEEEAddr] = dn.Name
	}

	results := make([]*adapter.InterviewResult, 0, len(devices))

	for i, dev := range devices {
		fmt.Printf("[%d/%d] Interviewing 0x%04X...", i+1, len(devices), dev.NwkAddr)

		// Use InterviewDeviceWithAddr to skip IeeeAddrReq (we already have it from NVRAM)
		result, err := a.InterviewDeviceWithAddr(ctx, dev.NwkAddr, dev.IEEEAddr)
		switch {
		case err != nil:
			fmt.Printf(" FAILED: %v\n", err)
			// Create a placeholder result for failed interviews
			result = &adapter.InterviewResult{
				IEEEAddr: dev.IEEEAddr,
				NwkAddr:  dev.NwkAddr,
				Success:  false,
				Errors:   []string{err.Error()},
			}
		case result.Success:
			fmt.Printf(" OK (%s %s)\n", result.Manufacturer, result.Model)
		default:
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
		// Get custom name for this device.
		name := namesByIEEE[r.IEEEAddr]
		if name != "" {
			fmt.Printf("\nDevice 0x%04X %s (%s)\n", r.NwkAddr, r.IEEEAddrString(), name)
		} else {
			fmt.Printf("\nDevice 0x%04X %s\n", r.NwkAddr, r.IEEEAddrString())
		}

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
		if name != "" {
			fmt.Printf("  Name:         %s\n", name)
		}
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

	return ctx.Err()
}

// goznp device watch -p <port>
var deviceWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for device events",
	Long:  "Monitor device join/leave events in real-time until Ctrl+C",
	RunE: func(_ *cobra.Command, _ []string) error {
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

	// Watch for events until context is canceled.
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down...")
			// Close permit join on exit
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			//nolint:errcheck // Best effort cleanup
			_ = a.PermitJoin(closeCtx, 0)
			cancel()
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
	RunE: func(_ *cobra.Command, _ []string) error {
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
				//nolint:errcheck // Name is optional, errors intentionally ignored
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
var deviceRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a device from the network",
	Long: `Request a device to leave the Zigbee network.

This sends a ZDO Management Leave Request to the device, which tells it
to gracefully leave the network. The device will be unpaired and will
need to be re-paired to rejoin.

You can specify the device by:
  --name   Custom name set on the device
  --ieee   IEEE address (e.g., 00:11:22:33:44:55:66:77)
  --addr   Network address (hex, e.g., 0x1234)

Note: Battery-powered devices may be asleep and not respond immediately.
For sleepy devices, you may need to wake them up (e.g., press a button)
before removing, or use --force to remove them from the coordinator's device list.

Examples:
  # Remove device by name
  goznp device remove --name "Kitchen Light"

  # Remove device by network address
  goznp device remove --addr 0x1234

  # Force remove a sleeping sensor
  goznp device remove --name "Motion Sensor" --force`,
	RunE: func(_ *cobra.Command, _ []string) error {
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

	// Device status flags
	deviceStatusCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceStatusCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceStatusCmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
	deviceStatusCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Device remove flags
	deviceRemoveCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceRemoveCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceRemoveCmd.Flags().BoolVar(&deviceForce, "force", false, "Force remove even if device doesn't respond")

	// Add port/baud flags to all core device commands
	for _, cmd := range []*cobra.Command{
		devicePermitCmd, devicePairCmd, deviceListCmd, deviceWatchCmd,
		deviceStatusCmd, deviceRemoveCmd,
	} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Register core commands only (other commands register via their own init functions)
	deviceCmd.AddCommand(devicePermitCmd)
	deviceCmd.AddCommand(devicePairCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceWatchCmd)
	deviceCmd.AddCommand(deviceStatusCmd)
	deviceCmd.AddCommand(deviceRemoveCmd)
}

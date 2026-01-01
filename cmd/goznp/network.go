package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/znp"
	"github.com/spf13/cobra"
)

var (
	// Network form flags
	networkChannel uint8
	networkPanID   uint16
	networkProfile string

	// Network channel flags
	forceChannel bool

	// Network power flags
	txPower int8

	// Network profile flags
	profileList bool
)

// networkCmd is the parent command for network operations.
var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Network management commands",
	Long:  "Commands for managing the Zigbee network (form, channel, power)",
}

var networkFormCmd = &cobra.Command{
	Use:   "form",
	Short: "Form a new Zigbee network",
	Long:  "Form a new Zigbee network as coordinator. Generates random PAN ID and network key.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runNetworkForm(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runNetworkForm(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create adapter with longer timeout for network formation
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	config := adapter.NetworkFormConfig{
		Channel: networkChannel,
		PanID:   networkPanID,
		Profile: networkProfile,
	}

	// Use default channel if not specified
	channelStr := "default (15)"
	if networkChannel != 0 {
		channelStr = fmt.Sprintf("%d", networkChannel)
	}

	fmt.Printf("Forming network on channel %s...\n", channelStr)

	if err := a.FormNetwork(ctx, config); err != nil {
		return fmt.Errorf("network formation failed: %w", err)
	}

	// Get network info to display
	networkInfo, err := a.GetNetworkInfo(ctx)
	if err != nil {
		fmt.Println("Network formed successfully!")
		return nil
	}

	fmt.Println("Network formed successfully!")
	fmt.Printf("  PAN ID:           0x%04X\n", networkInfo.PanID)
	fmt.Printf("  Extended PAN ID:  %s\n", znp.FormatIEEEAddr(networkInfo.ExtendedPanID))
	fmt.Printf("  Channel:          %d\n", networkInfo.Channel)
	fmt.Println("  Network Key:      (generated)")

	return nil
}

var networkChannelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Show or change network channel",
	Long:  "Show current channel or change to a new channel. Use --set to change, --force for forced (non-seamless) change.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runNetworkChannel(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runNetworkChannel(ctx context.Context) error {
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

	// If no channel specified, just show current
	if networkChannel == 0 {
		channel, err := a.GetCurrentChannel(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current channel: %w", err)
		}
		if channel == 0 {
			fmt.Println("Channel: (network not formed)")
		} else {
			fmt.Printf("Channel: %d\n", channel)
		}
		return nil
	}

	// Change channel
	seamless := !forceChannel
	modeStr := "seamless"
	if forceChannel {
		modeStr = "forced"
		// Confirm forced channel change
		fmt.Println("WARNING: Forced channel change will disrupt existing network!")
		fmt.Print("Proceed? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Printf("Changing channel to %d (%s)...\n", networkChannel, modeStr)

	if err := a.ChangeChannel(ctx, networkChannel, seamless); err != nil {
		return fmt.Errorf("channel change failed: %w", err)
	}

	fmt.Println("Channel changed successfully!")
	fmt.Printf("  New Channel: %d\n", networkChannel)

	return nil
}

var networkPowerCmd = &cobra.Command{
	Use:   "power",
	Short: "Show or set TX power",
	Long:  "Show current TX power or set a new power level in dBm",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		setRequested := cmd.Flags().Changed("set")
		if err := runNetworkPower(ctx, setRequested); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runNetworkPower(ctx context.Context, setRequested bool) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	if !setRequested {
		// Show TX power info
		fmt.Println("TX Power Configuration")
		fmt.Println()

		// Try to read current TX power from NV
		currentPower, err := a.GetTxPower(ctx)
		if err != nil {
			fmt.Println("  Current:  unknown (not configured)")
		} else {
			fmt.Printf("  Current:  %d dBm\n", currentPower)
		}
		fmt.Println()
		fmt.Println("  Hardware Limits:")
		fmt.Println("    CC2652R/RB:     max 5 dBm (default: 5 dBm)")
		fmt.Println("    CC2652P/1352P:  max 20 dBm (default: 9 dBm)")
		fmt.Println()
		fmt.Println("  Usage: goznp network power --set <dBm>")
		return nil
	}

	fmt.Printf("Setting TX power to %d dBm...\n", txPower)

	err = a.SetTxPower(ctx, txPower)
	if err != nil {
		return fmt.Errorf("failed to set TX power: %w", err)
	}

	fmt.Printf("TX power set to %d dBm\n", txPower)

	return nil
}

var networkProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show registered application profiles",
	Long: `Show registered application profiles or list all available profiles.

The coordinator registers multiple endpoints with different Application Profiles
during startup to support communication with various device types (following the
zigbee-herdsman pattern). Profiles cannot be changed dynamically.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runNetworkProfile(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runNetworkProfile(ctx context.Context) error {
	// If --list flag, show available profiles (no port required)
	if profileList {
		fmt.Println("Supported Zigbee Application Profiles:")
		fmt.Println()
		for _, p := range znp.AvailableProfiles() {
			fmt.Printf("  %-4s  %s (0x%04X)\n", p.ShortName(), p, uint16(p))
		}
		fmt.Println()
		fmt.Println("Note: All profiles are automatically registered as endpoints at startup.")
		fmt.Println("      The coordinator can communicate with devices of any profile type.")
		return nil
	}

	// Port is required to show registered profiles
	port, err := getPortPath()
	if err != nil {
		return fmt.Errorf("%w (use --list to see all supported profiles)", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Show registered profiles
	profiles := a.RegisteredProfiles()
	fmt.Println("Registered Endpoint Profiles:")
	fmt.Println()
	for _, p := range profiles {
		ep := adapter.GetEndpointForProfile(p)
		fmt.Printf("  Endpoint %3d: %s (0x%04X)\n", ep, p, uint16(p))
	}
	fmt.Println()
	fmt.Println("Use --list to see all supported profiles")

	return nil
}

var networkTopologyCmd = &cobra.Command{
	Use:   "topology",
	Short: "Show network mesh topology",
	Long: `Show the Zigbee mesh network topology by querying neighbor tables.

This discovers how devices are connected in the mesh:
- Coordinator (0x0000) is the central hub
- Routers (plugs) can relay messages and have children
- End Devices (sensors) connect to a parent (coordinator or router)`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runNetworkTopology(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runNetworkTopology(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("Discovering network topology...")
	fmt.Println()

	// Get all devices from NVRAM
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	fmt.Println("Coordinator (0x0000)")

	if len(devices) == 0 {
		fmt.Println("  └─ No devices paired")
		return nil
	}

	// Collect device info first (querying neighbor tables), then display
	type deviceInfo struct {
		dev        *adapter.Device
		deviceType string
		neighbors  []adapter.NeighborInfo
	}
	deviceInfos := make([]deviceInfo, 0, len(devices))

	fmt.Printf("Querying %d device(s)...", len(devices))

	for _, dev := range devices {
		// Try to get neighbor table with a short timeout
		// Only routers respond to this - end devices are typically asleep
		queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
		neighbors, err := a.GetNeighborTable(queryCtx, dev.NwkAddr)
		queryCancel()

		deviceType := "EndDevice"
		if err == nil {
			deviceType = "Router"
		}

		deviceInfos = append(deviceInfos, deviceInfo{
			dev:        dev,
			deviceType: deviceType,
			neighbors:  neighbors,
		})
		fmt.Print(".")
	}
	fmt.Println(" done")
	fmt.Println()

	// Now display the topology tree
	for i, info := range deviceInfos {
		prefix := "├─"
		childPrefix := "│     "
		if i == len(deviceInfos)-1 {
			prefix = "└─"
			childPrefix = "      "
		}

		fmt.Printf("  %s 0x%04X [%s] %s\n", prefix, info.dev.NwkAddr, info.deviceType, znp.FormatIEEEAddr(info.dev.IEEEAddr))

		// Display neighbors for routers
		if info.deviceType == "Router" && len(info.neighbors) > 0 {
			// Filter out coordinator from display
			displayNeighbors := make([]adapter.NeighborInfo, 0)
			for _, n := range info.neighbors {
				if n.NwkAddr != 0x0000 {
					displayNeighbors = append(displayNeighbors, n)
				}
			}

			for j, n := range displayNeighbors {
				nPrefix := childPrefix + "├─"
				if j == len(displayNeighbors)-1 {
					nPrefix = childPrefix + "└─"
				}
				nType := formatDeviceType(n.DeviceType)
				lqi := int(n.LQI) * 100 / 255
				fmt.Printf("  %s 0x%04X %s LQI: %d%%\n", nPrefix, n.NwkAddr, nType, lqi)
			}
		}
	}

	fmt.Println()
	fmt.Println("Legend:")
	fmt.Println("  LQI = Link Quality Indicator (signal strength, higher = better)")
	fmt.Println("  Router = Mains-powered, can relay messages")
	fmt.Println("  EndDevice = Battery-powered, sleeps most of the time")

	return nil
}

func formatDeviceType(dt uint8) string {
	switch dt {
	case 0:
		return "Coordinator"
	case 1:
		return "Router"
	case 2:
		return "EndDevice"
	default:
		return fmt.Sprintf("Unknown(%d)", dt)
	}
}

func formatRelation(rel uint8) string {
	switch rel {
	case 0:
		return "Parent"
	case 1:
		return "Child"
	case 2:
		return "Sibling"
	case 3:
		return "None"
	case 4:
		return "PreviousChild"
	default:
		return fmt.Sprintf("Unknown(%d)", rel)
	}
}

var resetFactoryCmd = &cobra.Command{
	Use:   "factory",
	Short: "Factory reset the adapter",
	Long:  "Erase all network configuration and return adapter to factory state",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runResetFactory(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runResetFactory(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Confirm factory reset
	fmt.Println("WARNING: This will erase all network configuration!")
	fmt.Print("Proceed? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("Performing factory reset...")

	if err := a.FactoryReset(ctx); err != nil {
		return fmt.Errorf("factory reset failed: %w", err)
	}

	fmt.Println("Factory reset complete. Adapter is in pristine state.")

	return nil
}

func init() {
	// Network form flags
	networkFormCmd.Flags().Uint8Var(&networkChannel, "channel", 0, "Zigbee channel (11-26, default: 15)")
	networkFormCmd.Flags().Uint16Var(&networkPanID, "pan-id", 0, "PAN ID (default: random)")
	networkFormCmd.Flags().StringVar(&networkProfile, "profile", "ha", "Zigbee profile (ha=Home Automation, se=Smart Energy)")

	// Network channel flags
	networkChannelCmd.Flags().Uint8Var(&networkChannel, "set", 0, "Target channel to set (11-26)")
	networkChannelCmd.Flags().BoolVar(&forceChannel, "force", false, "Force channel change (breaks existing network)")

	// Network power flags
	networkPowerCmd.Flags().Int8Var(&txPower, "set", 0, "TX power level in dBm to set")

	// Network profile flags
	networkProfileCmd.Flags().BoolVar(&profileList, "list", false, "List all supported profiles")
	networkProfileCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	networkProfileCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")

	// Add port/baud flags to network commands (except profile which has special handling)
	for _, cmd := range []*cobra.Command{networkFormCmd, networkChannelCmd, networkPowerCmd, resetFactoryCmd, networkTopologyCmd} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Build network command hierarchy
	networkCmd.AddCommand(networkFormCmd)
	networkCmd.AddCommand(networkChannelCmd)
	networkCmd.AddCommand(networkPowerCmd)
	networkCmd.AddCommand(networkProfileCmd)
	networkCmd.AddCommand(networkTopologyCmd)
}

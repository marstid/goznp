package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

// Power and Energy Monitoring Commands

// goznp device info -p <port> --addr <nwk-addr>
var deviceInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Query device capabilities",
	Long:  "Query device endpoints and supported clusters (capabilities)",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceInfo(ctx)
	},
}

// goznp device power -p <port> --addr <nwk-addr> [--endpoint <ep>]
var devicePowerCmd = &cobra.Command{
	Use:   "power",
	Short: "Read power consumption",
	Long:  "Read power consumption data from a smart plug (voltage, current, power, energy)",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDevicePower(ctx)
	},
}

// goznp device reset-energy -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceResetEnergyCmd = &cobra.Command{
	Use:   "reset-energy",
	Short: "Reset energy counter",
	Long:  "Reset the accumulated energy counter on a smart plug (manufacturer-specific)",
	RunE: func(_ *cobra.Command, _ []string) error {
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
				//nolint:errcheck // Name is optional, errors intentionally ignored
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

func init() {
	// Device info flags
	deviceInfoCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	//nolint:errcheck // Required flag in init
	deviceInfoCmd.MarkFlagRequired("addr")
	deviceInfoCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	deviceInfoCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")

	// Device power flags
	devicePowerCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	//nolint:errcheck // Required flag in init
	devicePowerCmd.MarkFlagRequired("addr")
	devicePowerCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	devicePowerCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	devicePowerCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")

	// Device reset-energy flags
	deviceResetEnergyCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	//nolint:errcheck // Required flag in init
	deviceResetEnergyCmd.MarkFlagRequired("addr")
	deviceResetEnergyCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceResetEnergyCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	deviceResetEnergyCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")

	// Register commands with device parent command
	deviceCmd.AddCommand(deviceInfoCmd)
	deviceCmd.AddCommand(devicePowerCmd)
	deviceCmd.AddCommand(deviceResetEnergyCmd)
}

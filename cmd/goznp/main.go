package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/serial"
	"github.com/marstid/goznp/pkg/znp"
	"github.com/spf13/cobra"
)

var (
	// Version information - set by GoReleaser via ldflags
	version   = "dev"
	buildTime = "unknown"

	// Global flags
	portPath string
	baudRate int
)

// getPortPath returns the serial port path from flag or environment variable.
// Priority: -p flag > GOZNP_PORT env var
func getPortPath() (string, error) {
	if portPath != "" {
		return portPath, nil
	}
	if envPort := os.Getenv("GOZNP_PORT"); envPort != "" {
		return envPort, nil
	}
	return "", fmt.Errorf("serial port not specified (use -p flag or set GOZNP_PORT environment variable)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "goznp",
	Short: "Go ZNP - Zigbee Network Processor CLI",
	Long:  "Command-line tool for interacting with Zigbee adapters via ZNP (Z-Stack Network Processor) protocol",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("goznp %s (built %s)\n", version, buildTime)
	},
}

func init() {
	// Add global flags to commands that need them
	for _, cmd := range []*cobra.Command{infoCmd, pingCmd, resetCmd} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Add all commands to root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(pingCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(networkCmd)
	rootCmd.AddCommand(deviceCmd)

	// Add factory-reset as subcommand of reset
	resetCmd.AddCommand(resetFactoryCmd)
}

// setupSignalHandler creates a context that cancels on SIGINT/SIGTERM
func setupSignalHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	return ctx
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available serial ports",
	Long:  "Shows USB ports with VID/PID info and highlights detected CC2652/SONOFF adapters",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runList(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runList() error {
	ports, err := serial.ListUSBPorts()
	if err != nil {
		return fmt.Errorf("failed to list ports: %w", err)
	}

	if len(ports) == 0 {
		fmt.Println("No serial ports found")
		return nil
	}

	fmt.Println("Available serial ports:")
	for _, port := range ports {
		var details []string

		if port.IsUSB {
			usbInfo := fmt.Sprintf("USB: VID=%s PID=%s", port.VID, port.PID)
			if port.Product != "" {
				usbInfo += fmt.Sprintf(" Product=%s", port.Product)
			}
			details = append(details, usbInfo)
		}

		detailStr := ""
		if len(details) > 0 {
			detailStr = fmt.Sprintf(" (%s)", strings.Join(details, ", "))
		}

		tag := ""
		if serial.IsCC2652Adapter(port) {
			tag = " [CC2652 Adapter]"
		}

		fmt.Printf("  %s%s%s\n", port.Name, detailStr, tag)
	}

	return nil
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display adapter information",
	Long:  "Connect to adapter and display firmware version, capabilities, and device info",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runInfo(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runInfo(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create adapter with timeout
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

	info, err := a.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get adapter info: %w", err)
	}

	// Display information
	fmt.Println("Adapter Information:")
	if info.Version != nil {
		fmt.Printf("  Z-Stack Variant:  %s\n", info.Version.Variant())
		fmt.Printf("  SDK Version:      %d.%d.%d\n", info.Version.MajorRel, info.Version.MinorRel, info.Version.MaintRel)
		fmt.Printf("  Build Date:       %d\n", info.Version.Revision)
		fmt.Printf("  Transport Rev:    %d\n", info.Version.TransportRev)
	}
	fmt.Printf("  Capabilities:     0x%04X\n", info.Capabilities)

	// Get network info
	networkInfo, err := a.GetNetworkInfo(ctx)
	if err != nil {
		// Non-fatal, just show what we have
		fmt.Printf("\nNetwork Information: (unavailable: %v)\n", err)
	} else {
		fmt.Println("\nNetwork Information:")
		fmt.Printf("  IEEE Address:     %s\n", znp.FormatIEEEAddr(networkInfo.IEEEAddr))

		// Check if network is formed (Channel 0 or PAN ID 0xFFFE = not formed)
		networkFormed := networkInfo.Channel != 0 && networkInfo.PanID != 0xFFFE

		if networkFormed {
			fmt.Printf("  Network Address:  0x%04X\n", networkInfo.ShortAddr)
			fmt.Printf("  PAN ID:           0x%04X\n", networkInfo.PanID)
			fmt.Printf("  Extended PAN ID:  %s\n", znp.FormatIEEEAddr(networkInfo.ExtendedPanID))
			fmt.Printf("  Channel:          %d\n", networkInfo.Channel)
			fmt.Printf("  Device Type:      %s\n", deviceTypeName(networkInfo.DeviceType))

			// Show registered profiles
			fmt.Printf("  Profiles:         ")
			if len(networkInfo.Profiles) > 0 {
				for i, p := range networkInfo.Profiles {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s", p.ShortName())
				}
				fmt.Println()
			} else {
				fmt.Println("HA")
			}
		} else {
			fmt.Printf("  Network Status:   Not formed\n")
		}

		// Note: TX Power cannot be read without setting it, so we don't show it here
		// Use 'goznp network power --set <value>' to configure TX power

		if networkFormed {
			fmt.Printf("\nAssociated Devices: %d\n", networkInfo.NumAssocDevices)
		}
	}

	return nil
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the adapter",
	Long:  "Ping the adapter and display capabilities with response time",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runPing(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runPing(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create adapter with timeout
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

	// Measure ping time
	start := time.Now()
	caps, err := a.Ping(ctx)
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	fmt.Printf("Ping successful! (%dms)\n", duration.Milliseconds())
	fmt.Printf("  Capabilities: 0x%04X\n", caps.Capabilities)

	return nil
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the adapter",
	Long:  "Perform soft reset and wait for reset indication",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runReset(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runReset(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create adapter with timeout
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("Sending soft reset...")

	if err := a.Reset(ctx); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	// Get version after reset
	version := a.Version()
	if version == nil {
		return fmt.Errorf("failed to retrieve version after reset")
	}

	fmt.Println("Reset complete!")
	fmt.Printf("  Reason: Soft reset\n")
	fmt.Printf("  Version: %s\n", version.String())

	return nil
}

// deviceTypeName returns a human-readable device type name.
// The deviceType is a bit mask:
//   - Bit 0 (0x01): Coordinator capability
//   - Bit 1 (0x02): Router capability
//   - Bit 2 (0x04): End Device capability
func deviceTypeName(t uint8) string {
	// Handle as bit mask
	var capabilities []string

	if t&0x01 != 0 {
		capabilities = append(capabilities, "Coordinator")
	}
	if t&0x02 != 0 {
		capabilities = append(capabilities, "Router")
	}
	if t&0x04 != 0 {
		capabilities = append(capabilities, "End Device")
	}

	if len(capabilities) == 0 {
		return fmt.Sprintf("Unknown (%d)", t)
	}

	return strings.Join(capabilities, " + ")
}

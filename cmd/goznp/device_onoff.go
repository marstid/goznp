package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
)

// goznp device on -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn device on",
	Long:  "Send On command to device (On/Off cluster)",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "on")
	},
}

// goznp device off -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn device off",
	Long:  "Send Off command to device (On/Off cluster)",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "off")
	},
}

// goznp device toggle -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle device state",
	Long:  "Send Toggle command to device (On/Off cluster)",
	RunE: func(_ *cobra.Command, _ []string) error {
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
	RunE: func(_ *cobra.Command, _ []string) error {
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

func init() {
	// Add flags to control commands
	for _, cmd := range []*cobra.Command{deviceOnCmd, deviceOffCmd, deviceToggleCmd} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
		cmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
		cmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
		cmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
		cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	}

	// Device brightness flags
	deviceBrightnessCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	deviceBrightnessCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	deviceBrightnessCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceBrightnessCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceBrightnessCmd.Flags().StringVar(&deviceNameLookup, "name", "", "Device name (alternative to --addr)")
	deviceBrightnessCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceBrightnessCmd.Flags().Uint8Var(&brightnessLevel, "level", 0, "Brightness level (0-254, 0=off, 254=max)")
	deviceBrightnessCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	// Register commands as subcommands of deviceCmd
	deviceCmd.AddCommand(deviceOnCmd)
	deviceCmd.AddCommand(deviceOffCmd)
	deviceCmd.AddCommand(deviceToggleCmd)
	deviceCmd.AddCommand(deviceBrightnessCmd)
}

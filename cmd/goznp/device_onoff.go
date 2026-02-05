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
	Long: `Send On command to a Zigbee device.

The device can be specified by:
  - Custom name: --name "Kitchen Light"
  - IEEE address: --ieee 00:11:22:33:44:55:66:77
  - Network address: --addr 0x1234

Examples:
  goznp devices on --name "Kitchen Light"
  goznp devices on --addr 0x1234`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "on")
	},
}

// goznp device off -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn device off",
	Long: `Send Off command to a Zigbee device.

The device can be specified by:
  - Custom name: --name "Kitchen Light"
  - IEEE address: --ieee 00:11:22:33:44:55:66:77
  - Network address: --addr 0x1234

Examples:
  goznp devices off --name "Kitchen Light"
  goznp devices off --addr 0x1234`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "off")
	},
}

// goznp device toggle -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle device state",
	Long: `Send Toggle command to a Zigbee device.

The device can be specified by:
  - Custom name: --name "Kitchen Light"
  - IEEE address: --ieee 00:11:22:33:44:55:66:77
  - Network address: --addr 0x1234

Examples:
  goznp devices toggle --name "Kitchen Light"
  goznp devices toggle --addr 0x1234`,
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
		return fmt.Errorf("failed to connect to adapter: %w\n\n"+
			"Troubleshooting:\n"+
			"  - Check that the adapter is connected\n"+
			"  - Verify the port path is correct\n"+
			"  - Try running 'goznp scan' to see available ports\n"+
			"  - Check for permission to access the port",
			err)
	}
	defer a.Close()

	// Resolve device address (supports --name, ieee, --addr)
	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Sending %s command to device at 0x%04X endpoint %d...\n", action, nwkAddr, deviceEndpoint)

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
		return fmt.Errorf("device command failed: %w\n\n"+
			"Troubleshooting:\n"+
			"  - Check if device is still connected to the network\n"+
			"  - Try using 'goznp devices status --name \"<name>\"' to verify device status\n"+
			"  - The device may be offline or out of range",
			err)
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
		return fmt.Errorf("failed to connect to adapter: %w", err)
	}
	defer a.Close()

	// Resolve device address (supports --name, ieee, --addr)
	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	// If no level specified, read current brightness
	if brightnessLevel == 0 && transitionTime == 0 {
		fmt.Printf("Reading brightness from device at 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)

		level, err := a.GetBrightness(ctx, nwkAddr, deviceEndpoint)
		if err != nil {
			return fmt.Errorf("failed to read brightness: %w\n\n"+
				"Troubleshooting:\n"+
				"  - Device may not support brightness control (check endpoints)\n"+
				"  - Try endpoint 1 for devices with multiple endpoints\n"+
				"  - Use 'goznp device info' to see device capabilities",
				err)
		}

		percent := int(level) * 100 / 254
		fmt.Printf("\nBrightness: %d (%.0f%%)\n", level, float64(percent))
		return nil
	}

	// Set brightness
	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	fmt.Printf("Setting brightness on device at 0x%04X endpoint %d to %d", nwkAddr, deviceEndpoint, brightnessLevel)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetBrightness(ctx, nwkAddr, deviceEndpoint, brightnessLevel, transitionTenths); err != nil {
		return fmt.Errorf("failed to set brightness: %w\n\n"+
			"Troubleshooting:\n"+
			"  - Brightness must be 0-254 (0=off, 1-254=dimmable range)\n"+
			"  - Device may not support brightness control\n"+
			"  - Check if device has power and is online",
			err)
	}

	percent := int(brightnessLevel) * 100 / 254
	fmt.Printf("Brightness set to %d (%.0f%%)\n", brightnessLevel, float64(percent))

	return nil
}

func init() {
	// Add flags to control commands using centralized helpers
	for _, cmd := range []*cobra.Command{deviceOnCmd, deviceOffCmd, deviceToggleCmd} {
		AddConnectionFlags(cmd)
		AddDeviceFlags(cmd)
	}

	// Device brightness flags
	AddConnectionFlags(deviceBrightnessCmd)
	AddDeviceFlags(deviceBrightnessCmd)
	deviceBrightnessCmd.Flags().Uint8Var(&brightnessLevel, "level", 0, "Brightness level (0-254, 0=off, 254=max)")
	deviceBrightnessCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	// Register commands as subcommands of deviceCmd
	deviceCmd.AddCommand(deviceOnCmd)
	deviceCmd.AddCommand(deviceOffCmd)
	deviceCmd.AddCommand(deviceToggleCmd)
	deviceCmd.AddCommand(deviceBrightnessCmd)
}

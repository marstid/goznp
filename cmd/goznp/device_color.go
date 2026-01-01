package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/spf13/cobra"
)

var (
	// Color control flags
	colorHue        uint8
	colorSaturation uint8
	colorX          string // String to support both float and int formats
	colorY          string
	colorR          uint8
	colorG          uint8
	colorB          uint8
	colorKelvin     uint16
)

// Parent command for color operations
var deviceColorCmd = &cobra.Command{
	Use:   "color",
	Short: "Control device color",
	Long:  "Set and read color on color-capable Zigbee lights using various color models",
}

// goznp device color hue-sat <ieee> <endpoint> <hue> <saturation> [transition]
var deviceColorHueSatCmd = &cobra.Command{
	Use:   "hue-sat",
	Short: "Set color using hue and saturation",
	Long: `Set the color of a light using hue and saturation values.

Hue represents the color on the color wheel:
  0   = Red
  85  = Green
  170 = Blue
  254 = Red (wraps around)

Saturation represents color intensity:
  0   = White (no color)
  254 = Fully saturated (vivid color)

Transition time is optional and defaults to 10 (1 second).
Transition time is in tenths of a second (e.g., 10 = 1 second, 50 = 5 seconds).

Examples:
  goznp device color hue-sat --addr 0xBE87 --endpoint 11 --hue 0 --sat 254           # Red
  goznp device color hue-sat --addr 0xBE87 --endpoint 11 --hue 85 --sat 254          # Green
  goznp device color hue-sat --addr 0xBE87 --endpoint 11 --hue 170 --sat 254         # Blue
  goznp device color hue-sat --addr 0xBE87 --endpoint 11 --hue 42 --sat 200 --transition 50  # Orange with 5s transition`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runColorHueSat(ctx)
	},
}

func runColorHueSat(ctx context.Context) error {
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

	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	fmt.Printf("Setting color on device 0x%04X endpoint %d to hue=%d, saturation=%d",
		nwkAddr, deviceEndpoint, colorHue, colorSaturation)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetHueSaturation(ctx, nwkAddr, deviceEndpoint, colorHue, colorSaturation, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}

	fmt.Printf("Color set to hue=%d, saturation=%d\n", colorHue, colorSaturation)

	return nil
}

// goznp device color xy <ieee> <endpoint> <x> <y> [transition]
var deviceColorXYCmd = &cobra.Command{
	Use:   "xy",
	Short: "Set color using CIE XY coordinates",
	Long: `Set the color of a light using CIE 1931 XY chromaticity coordinates.

X and Y coordinates can be specified as:
  - Integer values (0-65279): Raw ZCL values
  - Decimal values (0.0-1.0): Normalized coordinates (will be converted to 0-65535)

The CIE XY color space is device-independent and provides accurate color representation.

Transition time is optional and defaults to 10 (1 second).
Transition time is in tenths of a second.

Examples:
  goznp device color xy --addr 0xBE87 --endpoint 11 --x 0.675 --y 0.322         # Red
  goznp device color xy --addr 0xBE87 --endpoint 11 --x 0.4091 --y 0.518        # Green
  goznp device color xy --addr 0xBE87 --endpoint 11 --x 0.167 --y 0.04          # Blue
  goznp device color xy --addr 0xBE87 --endpoint 11 --x 44203 --y 21102         # Red (raw values)
  goznp device color xy --addr 0xBE87 --endpoint 11 --x 0.3 --y 0.3 --transition 20  # Warm white with 2s transition`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runColorXY(ctx)
	},
}

func runColorXY(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	// Parse X and Y coordinates
	x, err := parseXYCoordinate(colorX)
	if err != nil {
		return fmt.Errorf("invalid X coordinate: %w", err)
	}

	y, err := parseXYCoordinate(colorY)
	if err != nil {
		return fmt.Errorf("invalid Y coordinate: %w", err)
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

	fmt.Printf("Setting color on device 0x%04X endpoint %d to X=%d, Y=%d",
		nwkAddr, deviceEndpoint, x, y)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetColorXY(ctx, nwkAddr, deviceEndpoint, x, y, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}

	fmt.Printf("Color set to X=%d (%.4f), Y=%d (%.4f)\n",
		x, float64(x)/65535.0,
		y, float64(y)/65535.0)

	return nil
}

// parseXYCoordinate parses an XY coordinate that can be either:
// - A decimal value (0.0-1.0) which gets scaled to 0-65535
// - An integer value (0-65279) which is used directly
func parseXYCoordinate(s string) (uint16, error) {
	// Try to parse as float first
	if strings.Contains(s, ".") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid coordinate %q: %w", s, err)
		}
		if f < 0.0 || f > 1.0 {
			return 0, fmt.Errorf("coordinate %.4f out of range (0.0-1.0)", f)
		}
		// Scale to 0-65535
		return uint16(f * 65535.0), nil
	}

	// Parse as integer
	val, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid coordinate %q: %w", s, err)
	}
	if val > 65279 {
		return 0, fmt.Errorf("coordinate %d out of range (0-65279)", val)
	}
	return uint16(val), nil
}

// goznp device color rgb <ieee> <endpoint> <r> <g> <b> [transition]
var deviceColorRGBCmd = &cobra.Command{
	Use:   "rgb",
	Short: "Set color using RGB values",
	Long: `Set the color of a light using RGB (Red, Green, Blue) values.

RGB values range from 0-255 for each color channel.
The RGB values are automatically converted to CIE XY color space for the device.

Note: Due to color space conversion and device gamut limitations, the actual
color may differ slightly from the requested RGB value.

Transition time is optional and defaults to 10 (1 second).
Transition time is in tenths of a second.

Examples:
  goznp device color rgb --addr 0xBE87 --endpoint 11 --r 255 --g 0 --b 0       # Red
  goznp device color rgb --addr 0xBE87 --endpoint 11 --r 0 --g 255 --b 0       # Green
  goznp device color rgb --addr 0xBE87 --endpoint 11 --r 0 --g 0 --b 255       # Blue
  goznp device color rgb --addr 0xBE87 --endpoint 11 --r 255 --g 128 --b 0     # Orange
  goznp device color rgb --addr 0xBE87 --endpoint 11 --r 128 --g 0 --b 128 --transition 30  # Purple with 3s transition`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runColorRGB(ctx)
	},
}

func runColorRGB(ctx context.Context) error {
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

	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	fmt.Printf("Setting color on device 0x%04X endpoint %d to RGB(%d, %d, %d)",
		nwkAddr, deviceEndpoint, colorR, colorG, colorB)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetColorRGB(ctx, nwkAddr, deviceEndpoint, colorR, colorG, colorB, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}

	fmt.Printf("Color set to RGB(%d, %d, %d)\n", colorR, colorG, colorB)

	return nil
}

// goznp device color kelvin <ieee> <endpoint> <kelvin> [transition]
var deviceColorKelvinCmd = &cobra.Command{
	Use:   "kelvin",
	Short: "Set color temperature in Kelvin",
	Long: `Set the color temperature of a tunable white or color light using Kelvin values.

Color temperature represents the warmth or coolness of white light:
  2000K-2700K = Warm white (incandescent, candlelight)
  3000K       = Soft white
  4000K       = Cool white
  5000K       = Daylight
  6500K+      = Cool daylight

Valid range is typically 2000-6500K for most lights, but 1500-10000K is supported.

Transition time is optional and defaults to 10 (1 second).
Transition time is in tenths of a second.

Examples:
  goznp device color kelvin --addr 0xBE87 --endpoint 11 --kelvin 2700            # Warm white
  goznp device color kelvin --addr 0xBE87 --endpoint 11 --kelvin 4000            # Cool white
  goznp device color kelvin --addr 0xBE87 --endpoint 11 --kelvin 6500            # Daylight
  goznp device color kelvin --addr 0xBE87 --endpoint 11 --kelvin 2700 --transition 50  # Warm with 5s transition`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runColorKelvin(ctx)
	},
}

func runColorKelvin(ctx context.Context) error {
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

	// Convert milliseconds to tenths of second for ZCL
	transitionTenths := transitionTime / 100

	fmt.Printf("Setting color temperature on device 0x%04X endpoint %d to %dK",
		nwkAddr, deviceEndpoint, colorKelvin)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetColorKelvin(ctx, nwkAddr, deviceEndpoint, colorKelvin, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color temperature: %w", err)
	}

	fmt.Printf("Color temperature set to %dK\n", colorKelvin)

	return nil
}

// goznp device color get <ieee> <endpoint>
var deviceColorGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Read current color state",
	Long: `Read and display the current color state from a device.

This displays:
  - Current hue and saturation
  - Current X and Y chromaticity coordinates
  - Active color mode (Hue/Saturation, XY, or Color Temperature)

Examples:
  goznp device color get --addr 0xBE87 --endpoint 11`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runColorGet(ctx)
	},
}

func runColorGet(ctx context.Context) error {
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

	fmt.Printf("Reading color state from device 0x%04X endpoint %d...\n\n", nwkAddr, deviceEndpoint)

	colorState, err := a.GetColor(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("failed to read color state: %w", err)
	}

	fmt.Println("Color State:")
	fmt.Printf("  Hue:         %d (%.1f°)\n", colorState.Hue, float64(colorState.Hue)*360.0/254.0)
	fmt.Printf("  Saturation:  %d (%.1f%%)\n", colorState.Saturation, float64(colorState.Saturation)*100.0/254.0)
	fmt.Printf("  X:           %d (%.4f)\n", colorState.X, float64(colorState.X)/65535.0)
	fmt.Printf("  Y:           %d (%.4f)\n", colorState.Y, float64(colorState.Y)/65535.0)
	fmt.Printf("  Color Mode:  %s\n", formatColorMode(colorState.ColorMode))

	return nil
}

// formatColorMode returns a human-readable color mode string.
func formatColorMode(mode uint8) string {
	switch mode {
	case 0:
		return "Hue/Saturation"
	case 1:
		return "XY Chromaticity"
	case 2:
		return "Color Temperature"
	default:
		return fmt.Sprintf("Unknown (%d)", mode)
	}
}

func init() {
	// Color hue-sat flags
	deviceColorHueSatCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceColorHueSatCmd.MarkFlagRequired("addr")
	deviceColorHueSatCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceColorHueSatCmd.Flags().Uint8Var(&colorHue, "hue", 0, "Hue value (0-254)")
	deviceColorHueSatCmd.MarkFlagRequired("hue")
	deviceColorHueSatCmd.Flags().Uint8Var(&colorSaturation, "sat", 254, "Saturation value (0-254)")
	deviceColorHueSatCmd.Flags().Uint16Var(&transitionTime, "transition", 10, "Transition time in milliseconds")

	// Color XY flags
	deviceColorXYCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceColorXYCmd.MarkFlagRequired("addr")
	deviceColorXYCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceColorXYCmd.Flags().StringVar(&colorX, "x", "", "X coordinate (0.0-1.0 or 0-65279)")
	deviceColorXYCmd.MarkFlagRequired("x")
	deviceColorXYCmd.Flags().StringVar(&colorY, "y", "", "Y coordinate (0.0-1.0 or 0-65279)")
	deviceColorXYCmd.MarkFlagRequired("y")
	deviceColorXYCmd.Flags().Uint16Var(&transitionTime, "transition", 10, "Transition time in milliseconds")

	// Color RGB flags
	deviceColorRGBCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceColorRGBCmd.MarkFlagRequired("addr")
	deviceColorRGBCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceColorRGBCmd.Flags().Uint8Var(&colorR, "r", 0, "Red value (0-255)")
	deviceColorRGBCmd.MarkFlagRequired("r")
	deviceColorRGBCmd.Flags().Uint8Var(&colorG, "g", 0, "Green value (0-255)")
	deviceColorRGBCmd.MarkFlagRequired("g")
	deviceColorRGBCmd.Flags().Uint8Var(&colorB, "b", 0, "Blue value (0-255)")
	deviceColorRGBCmd.MarkFlagRequired("b")
	deviceColorRGBCmd.Flags().Uint16Var(&transitionTime, "transition", 10, "Transition time in milliseconds")

	// Color kelvin flags
	deviceColorKelvinCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceColorKelvinCmd.MarkFlagRequired("addr")
	deviceColorKelvinCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	deviceColorKelvinCmd.Flags().Uint16Var(&colorKelvin, "kelvin", 0, "Color temperature in Kelvin (2000-6500)")
	deviceColorKelvinCmd.MarkFlagRequired("kelvin")
	deviceColorKelvinCmd.Flags().Uint16Var(&transitionTime, "transition", 10, "Transition time in milliseconds")

	// Color get flags
	deviceColorGetCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceColorGetCmd.MarkFlagRequired("addr")
	deviceColorGetCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Add port/baud flags to all color commands
	for _, cmd := range []*cobra.Command{
		deviceColorHueSatCmd, deviceColorXYCmd, deviceColorRGBCmd,
		deviceColorKelvinCmd, deviceColorGetCmd,
	} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Build color command hierarchy
	deviceColorCmd.AddCommand(deviceColorHueSatCmd)
	deviceColorCmd.AddCommand(deviceColorXYCmd)
	deviceColorCmd.AddCommand(deviceColorRGBCmd)
	deviceColorCmd.AddCommand(deviceColorKelvinCmd)
	deviceColorCmd.AddCommand(deviceColorGetCmd)
}

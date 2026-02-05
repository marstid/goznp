package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

var (
	colorHue        uint8
	colorSaturation uint8
	colorX          string
	colorY          string
	colorR          uint8
	colorG          uint8
	colorB          uint8
	colorKelvin     uint16
)

var colorCmd = &cobra.Command{
	Use:   "color",
	Short: "Control device color",
	Long: `Set and read color on color-capable Zigbee lights.

Supports multiple color models:
  - Hue/Saturation: Color wheel (0-254 hue, 0-254 saturation)
  - XY: CIE 1931 color space (0-65535 X, 0-65535 Y or 0.0000-1.0000 fractional)
  - RGB: Standard Red/Green/Blue (0-255 each)
  - Temperature: Color temperature in Kelvin (2700K-6500K)

Examples:
  goznp color hue-sat --name "Living Room" --hue 85 --sat 254
  goznp color temperature --name "Kitchen" --kelvin 3000
  goznp color rgb --name "Bedroom" --red 255 --green 0 --blue 0`,
}

var colorHueSatCmd = &cobra.Command{
	Use:   "hue-sat",
	Short: "Set color using hue and saturation",
	Long: `Set the color using the HSV color model.

Hue represents the color on the color wheel (0-254):
  0   = Red
  85  = Green
  170 = Blue
  254 = Red (wraps around)

Saturation represents color intensity (0-254):
  0   = White (no color)
  254 = Fully saturated (vivid color)

Examples:
  goznp color hue-sat --name "Living Room" --hue 0 --sat 254      # Red
  goznp color hue-sat --name "Living Room" --hue 85 --sat 254     # Green
  goznp color hue-sat --name "Living Room" --hue 170 --sat 254    # Blue`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runColorHueSat(ctx)
	},
}

func runColorHueSat(ctx context.Context) error {
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

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

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

	fmt.Println("Color set successfully")
	return nil
}

var colorXYCmd = &cobra.Command{
	Use:   "xy",
	Short: "Set color using XY coordinates",
	Long: `Set the color using CIE 1931 XY color space coordinates.

XY values can be provided as:
  - Scaled integers (0-65535)
  - Fractional values (0.0-1.0, e.g., 0.7006)

Common colors (fractional):
  Red:    X=0.7006, Y=0.2993
  Green:  X=0.1724, Y=0.7478
  Blue:   X=0.1355, Y=0.0399
  White:  X=0.3227, Y=0.3290

Examples:
  goznp color xy --name "Living Room" --x 0.7006 --y 0.2993
  goznp color xy --name "Living Room" --x 45860 --y 19615`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runColorXY(ctx)
	},
}

func runColorXY(ctx context.Context) error {
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

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	x, y, err := parseXYCoordinates(colorX, colorY)
	if err != nil {
		return err
	}

	transitionTenths := transitionTime / 100

	fmt.Printf("Setting color on device 0x%04X endpoint %d to XY=(%d,%d)",
		nwkAddr, deviceEndpoint, x, y)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetColorXY(ctx, nwkAddr, deviceEndpoint, x, y, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}

	fmt.Println("Color set successfully")
	return nil
}

var colorRGBCmd = &cobra.Command{
	Use:   "rgb",
	Short: "Set color using RGB values",
	Long: `Set the color using standard RGB values.

RGB values range from 0 to 255 for each component:
  R: Red component (0-255)
  G: Green component (0-255)
  B: Blue component (0-255)

Examples:
  goznp color rgb --name "Living Room" --red 255 --green 0 --blue 0      # Red
  goznp color rgb --name "Living Room" --red 0 --green 255 --blue 0      # Green
  goznp color rgb --name "Living Room" --red 0 --green 0 --blue 255      # Blue
  goznp color rgb --name "Living Room" --red 255 --green 165 --blue 0    # Orange`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runColorRGB(ctx)
	},
}

func runColorRGB(ctx context.Context) error {
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

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	transitionTenths := transitionTime / 100

	fmt.Printf("Setting color on device 0x%04X endpoint %d to RGB=(%d,%d,%d)",
		nwkAddr, deviceEndpoint, colorR, colorG, colorB)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.SetColorRGB(ctx, nwkAddr, deviceEndpoint, colorR, colorG, colorB, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}

	fmt.Println("Color set successfully")
	return nil
}

var colorTemperatureCmd = &cobra.Command{
	Use:   "temperature",
	Short: "Set color temperature",
	Long: `Set the color temperature in Kelvin.

Color temperature represents the warmth or coolness of white light:
  2700K = Warm white (incandescent-like)
  3000K = Soft white
  4000K = Neutral white
  5000K = Cool white
  6500K = Daylight

Not all lights support the full range.

Examples:
  goznp color temperature --name "Living Room" --kelvin 3000
  goznp color temperature --name "Kitchen" --kelvin 5000`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runColorTemperature(ctx)
	},
}

func runColorTemperature(ctx context.Context) error {
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

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Setting color temperature on device 0x%04X endpoint %d to %dK", nwkAddr, deviceEndpoint, colorKelvin)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	transitionTenths := transitionTime / 100
	if err := a.SetColorKelvin(ctx, nwkAddr, deviceEndpoint, colorKelvin, transitionTenths); err != nil {
		return fmt.Errorf("failed to set color temperature: %w", err)
	}

	mireds := zcl.KelvinToMireds(colorKelvin)
	fmt.Printf("Color temperature set to %dK (%d mireds)\n", colorKelvin, mireds)
	return nil
}

var colorGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Read current color and temperature",
	Long: `Query the current color and color temperature from a light.

Example:
  goznp color get --name "Living Room"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runColorGet(ctx)
	},
}

func runColorGet(ctx context.Context) error {
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

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Reading color from device 0x%04X endpoint %d...\n\n", nwkAddr, deviceEndpoint)

	colorState, err := a.GetColor(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("failed to read color: %w", err)
	}

	fmt.Println("Current Color State:")
	fmt.Printf("  Mode: ")
	switch colorState.ColorMode {
	case 0:
		fmt.Println("Hue/Saturation")
		fmt.Printf("  Hue: %d\n", colorState.Hue)
		fmt.Printf("  Saturation: %d\n", colorState.Saturation)
	case 1:
		fmt.Println("XY Color Space")
		fmt.Printf("  X: %d\n", colorState.X)
		fmt.Printf("  Y: %d\n", colorState.Y)
	case 2:
		fmt.Println("Color Temperature")
		if temp, err := a.GetColorTemperature(ctx, nwkAddr, deviceEndpoint); err == nil {
			kelvin := zcl.MiredsToKelvin(temp)
			fmt.Printf("  Temperature: %dK (%d mireds)\n", kelvin, temp)
		}
	default:
		fmt.Printf("Unknown (%d)\n", colorState.ColorMode)
	}

	return nil
}

func parseXYCoordinates(xStr, yStr string) (uint16, uint16, error) {
	x, err := parseXYCoordinate(xStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid X coordinate: %w", err)
	}

	y, err := parseXYCoordinate(yStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid Y coordinate: %w", err)
	}

	return x, y, nil
}

func parseXYCoordinate(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("coordinate is required")
	}

	if strings.Contains(s, ".") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid coordinate %q: %w", s, err)
		}
		if f < 0.0 || f > 1.0 {
			return 0, fmt.Errorf("coordinate %.4f out of range (0.0-1.0)", f)
		}
		return uint16(f * 65535), nil
	}

	i, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid coordinate %q: %w", s, err)
	}
	if i > 65535 {
		return 0, fmt.Errorf("coordinate %d out of range (0-65535)", i)
	}
	return uint16(i), nil
}

func init() {
	AddConnectionFlags(colorHueSatCmd)
	AddConnectionFlags(colorXYCmd)
	AddConnectionFlags(colorRGBCmd)
	AddConnectionFlags(colorTemperatureCmd)
	AddConnectionFlags(colorGetCmd)

	AddDeviceFlags(colorHueSatCmd)
	AddDeviceFlags(colorXYCmd)
	AddDeviceFlags(colorRGBCmd)
	AddDeviceFlags(colorTemperatureCmd)
	AddDeviceFlags(colorGetCmd)

	colorHueSatCmd.Flags().Uint8Var(&colorHue, "hue", 0, "Hue value (0-254)")
	colorHueSatCmd.Flags().Uint8Var(&colorSaturation, "sat", 0, "Saturation value (0-254)")
	colorHueSatCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	colorXYCmd.Flags().StringVar(&colorX, "x", "", "X coordinate (0-65535 or 0.0-1.0)")
	colorXYCmd.Flags().StringVar(&colorY, "y", "", "Y coordinate (0-65535 or 0.0-1.0)")
	colorXYCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	colorRGBCmd.Flags().Uint8Var(&colorR, "red", 0, "Red value (0-255)")
	colorRGBCmd.Flags().Uint8Var(&colorG, "green", 0, "Green value (0-255)")
	colorRGBCmd.Flags().Uint8Var(&colorB, "blue", 0, "Blue value (0-255)")
	colorRGBCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	colorTemperatureCmd.Flags().Uint16Var(&colorKelvin, "kelvin", 0, "Color temperature in Kelvin (2700-6500)")
	colorTemperatureCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")

	colorCmd.AddCommand(colorHueSatCmd)
	colorCmd.AddCommand(colorXYCmd)
	colorCmd.AddCommand(colorRGBCmd)
	colorCmd.AddCommand(colorTemperatureCmd)
	colorCmd.AddCommand(colorGetCmd)
}

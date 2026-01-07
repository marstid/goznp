//go:build ignore

// Example: Control smart bulbs using the On/Off, Level Control, and Color Control clusters.
// This demonstrates basic and advanced light control operations.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

func main() {
	// Get serial port from device address from environment
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
		fmt.Printf("GOZNP_PORT not set, using default: %s\n", port)
	}

	// Device to control (network address and endpoint)
	nwkAddrArg := os.Getenv("DEVICE_ADDR")
	if nwkAddrArg == "" {
		fmt.Println("Usage: DEVICE_ADDR=0x1234 ENDPOINT=1 [COMMAND] go run light_control.go")
		fmt.Println("Commands: on, off, toggle, brightness LEVEL, color RGB, temp KELVIN")
		fmt.Println("\nExamples:")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 go run light_control.go         # Turn on")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 go run light_control.go off     # Turn off")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 go run light_control.go toggle  # Toggle state")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 COMMAND=brightness VALUE=128 go run light_control.go")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 COMMAND=color RGB=255,0,0 go run light_control.go")
		return
	}

	var nwkAddr uint16
	if _, err := fmt.Sscanf(nwkAddrArg, "0x%04X", &nwkAddr); err != nil {
		fmt.Sscanf(nwkAddrArg, "%d", &nwkAddr)
	}

	// Get endpoint (default to 1)
	endpointArg := os.Getenv("ENDPOINT")
	endpoint := uint8(1)
	if endpointArg != "" {
		fmt.Sscanf(endpointArg, "%d", &endpoint)
	}

	// Get command
	command := os.Getenv("COMMAND")
	if command == "" && len(os.Args) > 1 {
		command = os.Args[1]
	}

	// Create and open adapter connection
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	// Execute the requested command
	switch command {
	case "off":
		turnOff(ctx, adptr, nwkAddr, endpoint)

	case "toggle":
		toggle(ctx, adptr, nwkAddr, endpoint)

	case "brightness":
		value := os.Getenv("VALUE")
		if value == "" && len(os.Args) > 2 {
			value = os.Args[2]
		}
		var level uint8
		fmt.Sscanf(value, "%d", &level)
		setBrightness(ctx, adptr, nwkAddr, endpoint, level)

	case "color":
		rgb := os.Getenv("RGB")
		if rgb == "" && len(os.Args) > 2 {
			rgb = os.Args[2]
		}
		var r, g, b uint8
		fmt.Sscanf(rgb, "%d,%d,%d", &r, &g, &b)
		setColor(ctx, adptr, nwkAddr, endpoint, r, g, b)

	case "temp":
		kelvin := os.Getenv("KELVIN")
		if kelvin == "" && len(os.Args) > 2 {
			kelvin = os.Args[2]
		}
		var k uint16
		fmt.Sscanf(kelvin, "%d", &k)
		setColorTemp(ctx, adptr, nwkAddr, endpoint, k)

	default: // Default: turn on
		turnOn(ctx, adptr, nwkAddr, endpoint)
	}

	// Read and display final state
	showLightState(ctx, adptr, nwkAddr, endpoint)
}

func turnOn(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Printf("Turning on light at 0x%04X:%d\n", nwkAddr, endpoint)
	if err := adptr.TurnOn(ctx, nwkAddr, endpoint); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success")
	}
}

func turnOff(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Printf("Turning off light at 0x%04X:%d\n", nwkAddr, endpoint)
	if err := adptr.TurnOff(ctx, nwkAddr, endpoint); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success")
	}
}

func toggle(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Printf("Toggling light at 0x%04X:%d\n", nwkAddr, endpoint)
	if err := adptr.Toggle(ctx, nwkAddr, endpoint); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success")
	}
}

func setBrightness(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8, level uint8) {
	fmt.Printf("Setting brightness to %d\n", level)
	if err := adptr.SetBrightness(ctx, nwkAddr, endpoint, level); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success")
	}
}

func setColor(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8, r, g, b uint8) {
	fmt.Printf("Setting color to RGB(%d, %d, %d)\n", r, g, b)
	if err := adptr.SetColor(ctx, nwkAddr, endpoint, r, g, b); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success")
	}
}

func setColorTemp(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8, kelvin uint16) {
	fmt.Printf("Setting color temperature to %dK\n", kelvin)
	if err := adptr.SetColorTemp(ctx, nwkAddr, endpoint, kelvin); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success")
	}
}

func showLightState(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Println("\n=== Current State ===")

	// Read OnOff attribute
	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.AttrOnOff)
	if err == nil && len(attrs) > 0 && attrs[0].Status == zcl.StatusSuccess {
		if onOff, ok := attrs[0].Value.(uint8); ok {
			fmt.Printf("Power: %s\n", stateString(onOff))
		}
	}

	// Read current level
	attrs, err = adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.AttrCurrentLevel)
	if err == nil && len(attrs) > 0 && attrs[0].Status == zcl.StatusSuccess {
		if level, ok := attrs[0].Value.(uint8); ok {
			percentage := int(float64(level) / 254.0 * 100.0)
			fmt.Printf("Brightness: %d (%d%%) (0-254 scale)\n", level, percentage)
		}
	}

	// Read color mode (if color control cluster exists)
	attrs, err = adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.AttrColorMode)
	if err == nil && len(attrs) > 0 && attrs[0].Status == zcl.StatusSuccess {
		if mode, ok := attrs[0].Value.(uint8); ok {
			fmt.Printf("Color Mode: %s\n", colorModeString(mode))
		}
	}

	// Read XY color (if applicable)
	attrs, err = adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.AttrCurrentX, zcl.AttrCurrentY)
	if err == nil && len(attrs) >= 2 {
		var x, y uint16
		var hasColor bool
		if attrs[0].Status == zcl.StatusSuccess {
			if xv, ok := attrs[0].Value.(uint16); ok {
				x = xv
				hasColor = true
			}
		}
		if attrs[1].Status == zcl.StatusSuccess {
			if yv, ok := attrs[1].Value.(uint16); ok {
				y = yv
			}
		}
		if hasColor {
			r, g, b := zcl.XYToRGB(x, y)
			fmt.Printf("XY Color: (%d, %d) ≈ RGB(%d, %d, %d)\n", x, y, r, g, b)
		}
	}

	// Read color temperature (if applicable)
	attrs, err = adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.AttrColorTemperatureMireds)
	if err == nil && len(attrs) > 0 && attrs[0].Status == zcl.StatusSuccess {
		if mireds, ok := attrs[0].Value.(uint16); ok {
			kelvin := zcl.MiredsToKelvin(mireds)
			fmt.Printf("Color Temperature: %d mireds (%dK)\n", mireds, kelvin)
		}
	}
}

func stateString(state uint8) string {
	switch state {
	case 0:
		return "Off"
	case 1:
		return "On"
	default:
		return fmt.Sprintf("Unknown (%d)", state)
	}
}

func colorModeString(mode uint8) string {
	switch mode {
	case 0:
		return "Current Hue and Current Saturation"
	case 1:
		return "Current X and Current Y"
	case 2:
		return "Color Temperature Mireds"
	default:
		return fmt.Sprintf("Unknown (%d)", mode)
	}
}

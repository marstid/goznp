package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

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
  # Identify by name for 5 seconds (default)
  goznp device identify --name "Living Room Light"

  # Identify by address for 10 seconds
  goznp device identify --addr 0xBE87 --duration 10

  # Trigger blink effect
  goznp device identify --name "Bedroom Light" --effect blink`,
	RunE: func(_ *cobra.Command, _ []string) error {
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
		return fmt.Errorf("failed to connect to adapter: %w\n\n"+
			"Troubleshooting:\n"+
			"  - Check serial port connection\n"+
			"  - Try 'goznp connect' to verify adapter availability",
			err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
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

func init() {
	AddConnectionFlags(deviceIdentifyCmd)
	AddDeviceFlags(deviceIdentifyCmd)

	deviceIdentifyCmd.Flags().Uint16Var(&identifyDuration, "duration", 0, "Identify duration in seconds (default: 5)")
	deviceIdentifyCmd.Flags().StringVar(&identifyEffect, "effect", "", "Trigger effect (blink, breathe, okay, channel, stop)")

	// Register as subcommand of deviceCmd
	deviceCmd.AddCommand(deviceIdentifyCmd)
}

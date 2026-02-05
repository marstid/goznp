package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
)

// goznp device sensor -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceSensorCmd = &cobra.Command{
	Use:   "sensor",
	Short: "Read sensor data",
	Long: `Read temperature, humidity, and battery from a sensor device.

Actively queries the device for current sensor readings. Works with most
multisensor devices that support TemperatureMeasurement, RelativeHumidity,
and PowerConfiguration clusters.

Note: Battery-powered devices may be asleep and not respond immediately.
Wake the device (press a button) if needed.

Examples:
  # Read sensor data by name
  goznp device sensor --name "Temperature Sensor"

  # Read sensor data by address
  goznp device sensor --addr 0x1234`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceSensor(ctx)
	},
}

// goznp device listen -p <port>
var deviceListenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for sensor reports",
	Long: `Listen for incoming sensor reports from all devices.

Monitors the network for unsolicited sensor reports and displays them in real-time.
Works with devices that send automatic reports (must be bound to coordinator first).

Receives reports for:
  - Temperature, humidity, battery
  - On/off state changes
  - Power consumption data
  - Other attribute reports

Use Ctrl+C to stop listening.

Examples:
  # Listen for sensor reports (uses GOZNP_PORT env var or --port flag)
  goznp device listen --port /dev/tty.usbserial-110

  # Listen with environment variable set
  export GOZNP_PORT=/dev/tty.usbserial-110 && goznp device listen`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceListen(ctx)
	},
}

func runDeviceListen(ctx context.Context) error {
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

	fmt.Println("Listening for sensor reports... (Ctrl+C to stop)")
	fmt.Println("Tip: Press a button on your sensor to trigger a report")
	fmt.Println()
	fmt.Println("Note: Smart plugs typically don't send automatic power reports unless")
	fmt.Println("      configured to do so. Use 'device power' to actively read power data.")
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			report, err := a.WaitForSensorReport(ctx, 5*time.Second)
			if err != nil {
				continue // Timeout, try again
			}

			// Skip if not a Report Attributes command (0x0A)
			// Most messages are device announcements or other non-report commands
			if report.FrameCommandID != 0x0A && report.ParseError == "" {
				continue
			}

			timestamp := time.Now().Format("15:04:05")
			fmt.Printf("[%s] Device 0x%04X (Cluster: 0x%04X, Endpoint: %d, Cmd: 0x%02X):\n",
				timestamp, report.NwkAddr, report.ClusterID, report.Endpoint, report.FrameCommandID)

			// Show parse errors
			if report.ParseError != "" {
				fmt.Printf("  Parse error: %s\n", report.ParseError)
			}

			// Display sensor data
			if report.Temperature != nil {
				fmt.Printf("  Temperature: %.1f°C\n", *report.Temperature)
			}
			if report.Humidity != nil {
				fmt.Printf("  Humidity:    %.1f%%\n", *report.Humidity)
			}
			if report.Battery != nil {
				fmt.Printf("  Battery:     %d%%\n", *report.Battery)
			}

			// Display on/off state
			if report.OnOff != nil {
				state := "OFF"
				if *report.OnOff {
					state = "ON"
				}
				fmt.Printf("  On/Off:      %s\n", state)
			}

			// Display power consumption data
			if report.Voltage != nil {
				fmt.Printf("  Voltage:     %.1f V\n", *report.Voltage)
			}
			if report.Current != nil {
				fmt.Printf("  Current:     %.3f A\n", *report.Current)
			}
			if report.Power != nil {
				fmt.Printf("  Power:       %.1f W\n", *report.Power)
			}
			if report.Energy != nil {
				fmt.Printf("  Energy:      %.3f kWh\n", *report.Energy)
			}

			// Display raw attributes for debugging if nothing else was displayed
			hasData := report.Temperature != nil || report.Humidity != nil || report.Battery != nil ||
				report.Voltage != nil || report.Current != nil || report.Power != nil || report.Energy != nil ||
				report.OnOff != nil
			if !hasData && len(report.Attributes) > 0 {
				fmt.Println("  Raw attributes:")
				for _, attr := range report.Attributes {
					fmt.Printf("    Attr 0x%04X: %v (type: %T)\n", attr.ID, attr.Value, attr.Value)
				}
			}
			fmt.Println()
		}
	}
}

func runDeviceSensor(ctx context.Context) error {
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

	fmt.Printf("Reading sensor data from device 0x%04X endpoint %d...\n", nwkAddr, deviceEndpoint)
	fmt.Println("(Note: Battery sensors may be asleep and not respond immediately)")
	fmt.Println()

	data, err := a.ReadSensorData(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("failed to read sensor data: %w", err)
	}

	fmt.Println("Sensor Data:")
	if data.Temperature != nil {
		fmt.Printf("  Temperature:  %.1f°C\n", *data.Temperature)
	} else {
		fmt.Println("  Temperature:  - (device may be asleep)")
	}

	if data.Humidity != nil {
		fmt.Printf("  Humidity:     %.1f%%\n", *data.Humidity)
	} else {
		fmt.Println("  Humidity:     -")
	}

	if data.Battery != nil {
		fmt.Printf("  Battery:      %d%%\n", *data.Battery)
	} else {
		fmt.Println("  Battery:      -")
	}

	return nil
}

func init() {
	AddConnectionFlags(deviceSensorCmd)
	AddConnectionFlags(deviceListenCmd)

	AddDeviceFlags(deviceSensorCmd)
	deviceSensorCmd.MarkFlagRequired("addr")

	// Register sensor commands to device command
	deviceCmd.AddCommand(deviceSensorCmd)
	deviceCmd.AddCommand(deviceListenCmd)
}

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/spf13/cobra"
)

// goznp device sensor -p <port> --addr <nwk-addr> [--endpoint <ep>]
var deviceSensorCmd = &cobra.Command{
	Use:   "sensor",
	Short: "Read sensor data",
	Long:  "Read temperature, humidity, and battery from a sensor device",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSensor(ctx)
	},
}

// goznp device listen -p <port>
var deviceListenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for sensor reports",
	Long:  "Listen for incoming sensor reports (temperature, humidity, etc.) from all devices",
	RunE: func(cmd *cobra.Command, args []string) error {
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
	// Device sensor flags
	deviceSensorCmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
	deviceSensorCmd.MarkFlagRequired("addr")
	deviceSensorCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")

	// Add port/baud flags to sensor commands
	for _, cmd := range []*cobra.Command{deviceSensorCmd, deviceListenCmd} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Register sensor commands to device command
	deviceCmd.AddCommand(deviceSensorCmd)
	deviceCmd.AddCommand(deviceListenCmd)
}

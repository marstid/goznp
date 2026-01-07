package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/adapter/quirks"
	"github.com/marstid/goznp/pkg/znp"
	"github.com/spf13/cobra"
)

var quirksCmd = &cobra.Command{
	Use:   "quirks",
	Short: "Device quirks management",
	Long:  "Commands for viewing and managing device-specific quirks",
}

var quirksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered quirks",
	Long:  "Show all device quirks registered in the system",
	Run: func(cmd *cobra.Command, args []string) {
		runQuirksList()
	},
}

func runQuirksList() {
	registry := quirks.DefaultRegistry()

	fmt.Printf("Registered Quirks: %d\n", registry.Count())
	fmt.Println("==================")
	fmt.Println()

	for _, q := range registry.All() {
		fmt.Printf("ID: %s\n", q.ID)
		fmt.Printf("  Type:        %s\n", q.Type)
		fmt.Printf("  Description: %s\n", q.Description)

		// Show matcher criteria
		if q.Matcher.ManufacturerCode != nil {
			fmt.Printf("  Manuf Code:  0x%04X\n", *q.Matcher.ManufacturerCode)
		}
		if q.Matcher.ManufacturerPattern != "" {
			fmt.Printf("  Manuf Match: %s\n", q.Matcher.ManufacturerPattern)
		}
		if q.Matcher.ModelPattern != "" {
			fmt.Printf("  Model Match: %s\n", q.Matcher.ModelPattern)
		}
		fmt.Println()
	}
}

var quirksDeviceCmd = &cobra.Command{
	Use:   "device <address>",
	Short: "Show quirks for a device",
	Long: `Show which quirks apply to a specific device.
Address should be in hex format: 0xABCD`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupSignalHandler()
		if err := runQuirksDevice(ctx, args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func runQuirksDevice(ctx context.Context, addrStr string) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Parse address
	var addr uint64
	_, err = fmt.Sscanf(addrStr, "0x%x", &addr)
	if err != nil {
		_, err = fmt.Sscanf(addrStr, "%d", &addr)
		if err != nil {
			return fmt.Errorf("invalid address format: %s (use 0xABCD or decimal)", addrStr)
		}
	}
	nwkAddr := uint16(addr)

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

	fmt.Printf("Looking up device 0x%04X...\n", nwkAddr)

	// Get device info via interview
	result, err := a.InterviewDevice(ctx, nwkAddr)
	if err != nil {
		return fmt.Errorf("failed to interview device: %w", err)
	}

	fmt.Printf("\nDevice Information:\n")
	fmt.Printf("  Network Address: 0x%04X\n", result.NwkAddr)
	fmt.Printf("  IEEE Address:    %s\n", znp.FormatIEEEAddr(result.IEEEAddr))
	fmt.Printf("  Manufacturer:    %s (0x%04X)\n", result.Manufacturer, result.ManufacturerCode)
	fmt.Printf("  Model:           %s\n", result.Model)
	fmt.Println()

	// Find matching quirks
	registry := quirks.DefaultRegistry()
	deviceInfo := quirks.DeviceInfo{
		ManufacturerCode: result.ManufacturerCode,
		Manufacturer:     result.Manufacturer,
		Model:            result.Model,
	}

	matchingQuirks := registry.FindQuirks(deviceInfo)

	if len(matchingQuirks) == 0 {
		fmt.Println("No quirks apply to this device.")
		return nil
	}

	fmt.Printf("Applied Quirks: %d\n", len(matchingQuirks))
	fmt.Println("================")
	for _, q := range matchingQuirks {
		fmt.Printf("\n%s\n", q.ID)
		fmt.Printf("  Type:        %s\n", q.Type)
		fmt.Printf("  Description: %s\n", q.Description)

		// Show specific quirk details
		if len(q.AttributeOverrides) > 0 {
			fmt.Println("  Attribute Overrides:")
			for _, ao := range q.AttributeOverrides {
				fmt.Printf("    - Cluster 0x%04X, Attr 0x%04X", ao.ClusterID, ao.AttributeID)
				if ao.Divisor != nil {
					fmt.Printf(" (divisor: %.0f)", *ao.Divisor)
				}
				if ao.Multiplier != nil {
					fmt.Printf(" (multiplier: %.2f)", *ao.Multiplier)
				}
				fmt.Println()
			}
		}

		if len(q.ResponseOverrides) > 0 {
			fmt.Println("  Response Overrides:")
			for _, ro := range q.ResponseOverrides {
				fmt.Printf("    - Cluster 0x%04X: cmd 0x%02X also accepts", ro.ClusterID, ro.ExpectedCommand)
				for _, cmd := range ro.AcceptCommands {
					fmt.Printf(" 0x%02X", cmd)
				}
				fmt.Println()
			}
		}

		if len(q.EnergyResetMethods) > 0 {
			fmt.Println("  Energy Reset Methods:")
			for i, em := range q.EnergyResetMethods {
				if em.UseCommand {
					fmt.Printf("    %d. Cluster 0x%04X, Command 0x%02X\n", i+1, em.ClusterID, em.CommandID)
				} else {
					fmt.Printf("    %d. Cluster 0x%04X, Attr 0x%04X\n", i+1, em.ClusterID, em.AttributeID)
				}
			}
		}

		if q.TimingOverride != nil {
			fmt.Println("  Timing Overrides:")
			if q.TimingOverride.InterviewTimeout > 0 {
				fmt.Printf("    Interview Timeout: %ds\n", q.TimingOverride.InterviewTimeout)
			}
			if q.TimingOverride.CommandTimeout > 0 {
				fmt.Printf("    Command Timeout:   %dms\n", q.TimingOverride.CommandTimeout)
			}
			if q.TimingOverride.RetryCount > 0 {
				fmt.Printf("    Retry Count:       %d\n", q.TimingOverride.RetryCount)
			}
			if q.TimingOverride.RetryDelay > 0 {
				fmt.Printf("    Retry Delay:       %dms\n", q.TimingOverride.RetryDelay)
			}
		}
	}

	return nil
}

var quirksMatchCmd = &cobra.Command{
	Use:   "match <manufacturer> <model>",
	Short: "Test quirk matching",
	Long: `Test which quirks would match a device with the given manufacturer and model.
Useful for debugging quirk patterns without connecting to a device.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runQuirksMatch(args[0], args[1])
	},
}

func runQuirksMatch(manufacturer, model string) {
	registry := quirks.DefaultRegistry()
	deviceInfo := quirks.DeviceInfo{
		ManufacturerCode: 0,
		Manufacturer:     manufacturer,
		Model:            model,
	}

	fmt.Printf("Testing quirk match for:\n")
	fmt.Printf("  Manufacturer: %s\n", manufacturer)
	fmt.Printf("  Model:        %s\n", model)
	fmt.Println()

	matchingQuirks := registry.FindQuirks(deviceInfo)

	if len(matchingQuirks) == 0 {
		fmt.Println("No quirks match these criteria.")
		return
	}

	fmt.Printf("Matching Quirks: %d\n", len(matchingQuirks))
	fmt.Println("================")
	for _, q := range matchingQuirks {
		fmt.Printf("  - %s (%s)\n", q.ID, q.Type)
	}
}

func init() {
	// Add port/baud flags
	quirksDeviceCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	quirksDeviceCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")

	quirksCmd.AddCommand(quirksListCmd)
	quirksCmd.AddCommand(quirksDeviceCmd)
	quirksCmd.AddCommand(quirksMatchCmd)
}

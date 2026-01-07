package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/znp"
	"github.com/spf13/cobra"
)

// Device Name Commands

// Parent command for device name operations
var deviceNameCmd = &cobra.Command{
	Use:   "name",
	Short: "Manage custom device names",
	Long:  "Set, get, list, or delete custom names and comments for devices",
}

// goznp device name set --ieee <addr> --name <name> [--comment <comment>]
var deviceNameSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set custom name for a device",
	Long: `Set a custom name and optional comment for a device.

The custom name is stored on the coordinator and associated with the device's
IEEE address. This name persists across network restarts.

Examples:
  goznp device name set --ieee 00:11:22:33:44:55:66:77 --name "Living Room Light"
  goznp device name set --ieee 00:11:22:33:44:55:66:77 --name "Kitchen Sensor" --comment "Temperature and humidity sensor"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameSet(ctx)
	},
}

func runDeviceNameSet(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if deviceIEEE == "" {
		return fmt.Errorf("--ieee is required")
	}

	if deviceName == "" {
		return fmt.Errorf("--name is required")
	}

	ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
	if err != nil {
		return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
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

	fmt.Printf("Setting custom name for device %s...\n", znp.FormatIEEEAddr(ieeeAddr))

	if err := a.SetDeviceName(ctx, ieeeAddr, deviceName, deviceComment); err != nil {
		return fmt.Errorf("failed to set device name: %w", err)
	}

	fmt.Printf("Device name set to: %s\n", deviceName)
	if deviceComment != "" {
		fmt.Printf("Comment: %s\n", deviceComment)
	}

	return nil
}

// goznp device name get --ieee <addr>
var deviceNameGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get custom name for a device",
	Long: `Get the custom name and comment for a device.

Example:
  goznp device name get --ieee 00:11:22:33:44:55:66:77`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameGet(ctx)
	},
}

func runDeviceNameGet(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if deviceIEEE == "" {
		return fmt.Errorf("--ieee is required")
	}

	ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
	if err != nil {
		return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
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

	info, err := a.GetDeviceName(ctx, ieeeAddr)
	if err != nil {
		return fmt.Errorf("failed to get device name: %w", err)
	}

	if info == nil || info.Name == "" {
		fmt.Println("No custom name set")
		return nil
	}

	fmt.Printf("Device: %s\n", znp.FormatIEEEAddr(ieeeAddr))
	fmt.Printf("Name: %s\n", info.Name)
	if info.Comment != "" {
		fmt.Printf("Comment: %s\n", info.Comment)
	}

	return nil
}

// goznp device name list
var deviceNameListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices with custom names",
	Long: `List all devices that have custom names set.

Example:
  goznp device name list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameList(ctx)
	},
}

func runDeviceNameList(ctx context.Context) error {
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

	names, err := a.ListDeviceNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to list device names: %w", err)
	}

	if len(names) == 0 {
		fmt.Println("No devices with custom names")
		return nil
	}

	fmt.Printf("Found %d device(s) with custom names:\n\n", len(names))

	// Create a tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IEEE Address\tName\tComment")
	fmt.Fprintln(w, "------------\t----\t-------")

	for _, device := range names {
		comment := device.Comment
		if comment == "" {
			comment = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			znp.FormatIEEEAddr(device.IEEEAddr),
			device.Name,
			comment,
		)
	}

	w.Flush()

	return nil
}

// goznp device name delete --ieee <addr>
var deviceNameDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete custom name for a device",
	Long: `Delete the custom name and comment for a device.

Example:
  goznp device name delete --ieee 00:11:22:33:44:55:66:77`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceNameDelete(ctx)
	},
}

func runDeviceNameDelete(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if deviceIEEE == "" {
		return fmt.Errorf("--ieee is required")
	}

	ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
	if err != nil {
		return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
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

	fmt.Printf("Deleting custom name for device %s...\n", znp.FormatIEEEAddr(ieeeAddr))

	if err := a.DeleteDeviceName(ctx, ieeeAddr); err != nil {
		return fmt.Errorf("failed to delete device name: %w", err)
	}

	fmt.Println("Device name deleted")

	return nil
}

func init() {
	// Device name flags
	deviceNameSetCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceNameSetCmd.MarkFlagRequired("ieee")
	deviceNameSetCmd.Flags().StringVar(&deviceName, "name", "", "Custom device name")
	deviceNameSetCmd.MarkFlagRequired("name")
	deviceNameSetCmd.Flags().StringVar(&deviceComment, "comment", "", "Optional device comment")

	deviceNameGetCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceNameGetCmd.MarkFlagRequired("ieee")

	deviceNameDeleteCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	deviceNameDeleteCmd.MarkFlagRequired("ieee")

	// Build name subcommand hierarchy
	deviceNameCmd.AddCommand(deviceNameSetCmd)
	deviceNameCmd.AddCommand(deviceNameGetCmd)
	deviceNameCmd.AddCommand(deviceNameListCmd)
	deviceNameCmd.AddCommand(deviceNameDeleteCmd)

	// Register with parent device command
	deviceCmd.AddCommand(deviceNameCmd)
}

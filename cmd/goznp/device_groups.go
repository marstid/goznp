package main

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/spf13/cobra"
)

// Group-related global variables
var removeAllGroups bool

// Groups Cluster Commands

// Parent command for group operations
var deviceGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage device group membership",
	Long:  "Add, remove, or list group memberships for a device",
}

// goznp device group add --addr <nwk-addr> --group <id> [--name <name>]
var deviceGroupAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add device to a group",
	Long: `Add a device endpoint to a Zigbee group.

Groups allow controlling multiple devices with a single command (multicast).
Group IDs are 16-bit values (1-65535). Group 0 is reserved.

Examples:
  goznp device group add --addr 0xBE87 --endpoint 11 --group 1
  goznp device group add --addr 0xBE87 --endpoint 11 --group 1 --name "Living Room"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupAdd(ctx)
	},
}

func runDeviceGroupAdd(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
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

	fmt.Printf("Adding device 0x%04X endpoint %d to group %d", nwkAddr, deviceEndpoint, groupID)
	if groupName != "" {
		fmt.Printf(" (%s)", groupName)
	}
	fmt.Println("...")

	if err := a.AddToGroup(ctx, nwkAddr, deviceEndpoint, groupID, groupName); err != nil {
		return fmt.Errorf("add to group failed: %w", err)
	}

	fmt.Printf("Device added to group %d\n", groupID)
	return nil
}

// goznp device group remove --addr <nwk-addr> --group <id>
var deviceGroupRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove device from a group",
	Long: `Remove a device endpoint from a Zigbee group.

Use --group 0 or --all to remove from all groups.

Examples:
  goznp device group remove --addr 0xBE87 --endpoint 11 --group 1
  goznp device group remove --addr 0xBE87 --endpoint 11 --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupRemove(ctx)
	},
}

func runDeviceGroupRemove(ctx context.Context) error {
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

	if removeAllGroups {
		fmt.Printf("Removing device 0x%04X endpoint %d from all groups...\n", nwkAddr, deviceEndpoint)
		if err := a.RemoveFromAllGroups(ctx, nwkAddr, deviceEndpoint); err != nil {
			return fmt.Errorf("remove from all groups failed: %w", err)
		}
		fmt.Println("Device removed from all groups")
		return nil
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (or use --all)")
	}

	fmt.Printf("Removing device 0x%04X endpoint %d from group %d...\n", nwkAddr, deviceEndpoint, groupID)
	if err := a.RemoveFromGroup(ctx, nwkAddr, deviceEndpoint, groupID); err != nil {
		return fmt.Errorf("remove from group failed: %w", err)
	}

	fmt.Printf("Device removed from group %d\n", groupID)
	return nil
}

// goznp device group list --addr <nwk-addr>
var deviceGroupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List device group memberships",
	Long: `Query which groups a device endpoint belongs to.

Example:
  goznp device group list --addr 0xBE87 --endpoint 11`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupList(ctx)
	},
}

func runDeviceGroupList(ctx context.Context) error {
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

	fmt.Printf("Querying group membership for device 0x%04X endpoint %d...\n\n", nwkAddr, deviceEndpoint)

	membership, err := a.GetGroupMembership(ctx, nwkAddr, deviceEndpoint)
	if err != nil {
		return fmt.Errorf("get group membership failed: %w", err)
	}

	if len(membership.Groups) == 0 {
		fmt.Println("Device is not a member of any groups")
	} else {
		fmt.Printf("Device is a member of %d group(s):\n", len(membership.Groups))
		for _, g := range membership.Groups {
			fmt.Printf("  - Group %d (0x%04X)\n", g, g)
		}
	}

	if membership.Capacity != 0xFF {
		fmt.Printf("\nRemaining capacity: %d groups\n", membership.Capacity)
	}

	return nil
}

// goznp device group on --group <id>
var deviceGroupOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn on all devices in a group",
	Long: `Send On command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.
This is much faster than sending individual commands to each device.

Example:
  goznp device group on --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "on")
	},
}

// goznp device group off --group <id>
var deviceGroupOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn off all devices in a group",
	Long: `Send Off command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.

Example:
  goznp device group off --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "off")
	},
}

// goznp device group toggle --group <id>
var deviceGroupToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle all devices in a group",
	Long: `Send Toggle command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.

Example:
  goznp device group toggle --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "toggle")
	},
}

// goznp device group brightness --group <id> --level <0-254>
var deviceGroupBrightnessCmd = &cobra.Command{
	Use:   "brightness",
	Short: "Set brightness for all devices in a group",
	Long: `Set brightness level on all dimmable devices in a Zigbee group.

Level values:
  0   = Off (will also turn off devices)
  1   = Minimum brightness
  254 = Maximum brightness

Transition time is in milliseconds (default: 0 for instant).

Examples:
  goznp device group brightness --group 1 --level 128                    # Set to 50%
  goznp device group brightness --group 1 --level 254 --transition 1000  # Fade to max over 1s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupBrightness(ctx)
	},
}

// goznp device group scene --group <id> --scene <id>
var deviceGroupSceneCmd = &cobra.Command{
	Use:   "recall-scene",
	Short: "Recall scene on all devices in a group",
	Long: `Recall a scene on all devices in a Zigbee group.

This sends the RecallScene command to all group members simultaneously.
The devices will transition to their saved scene states.

Example:
  goznp device group recall-scene --group 1 --scene 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupScene(ctx)
	},
}

func runDeviceGroupCommand(ctx context.Context, action string) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
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

	fmt.Printf("Sending %s command to group %d...\n", action, groupID)

	switch action {
	case "on":
		err = a.GroupTurnOn(ctx, groupID)
	case "off":
		err = a.GroupTurnOff(ctx, groupID)
	case "toggle":
		err = a.GroupToggle(ctx, groupID)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	fmt.Printf("Command sent to group %d\n", groupID)

	return nil
}

func runDeviceGroupBrightness(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
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

	fmt.Printf("Setting brightness on group %d to %d", groupID, brightnessLevel)
	if transitionTime > 0 {
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.GroupSetBrightness(ctx, groupID, brightnessLevel, transitionTenths); err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	percent := int(brightnessLevel) * 100 / 254
	fmt.Printf("Brightness set to %d (%.0f%%) on group %d\n", brightnessLevel, float64(percent), groupID)

	return nil
}

func runDeviceGroupScene(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)")
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

	fmt.Printf("Recalling scene %d on group %d...\n", sceneID, groupID)

	if err := a.GroupRecallScene(ctx, groupID, sceneID); err != nil {
		return fmt.Errorf("recall scene failed: %w", err)
	}

	fmt.Printf("Scene %d recalled on group %d\n", sceneID, groupID)
	return nil
}

func init() {
	// Device group flags
	for _, cmd := range []*cobra.Command{deviceGroupAddCmd, deviceGroupRemoveCmd, deviceGroupListCmd} {
		cmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
		cmd.MarkFlagRequired("addr")
		cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
	}
	deviceGroupAddCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
	deviceGroupAddCmd.MarkFlagRequired("group")
	deviceGroupAddCmd.Flags().StringVar(&groupName, "name", "", "Optional group name (max 16 chars)")
	deviceGroupRemoveCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID to remove from")
	deviceGroupRemoveCmd.Flags().BoolVar(&removeAllGroups, "all", false, "Remove from all groups")

	// Device group command flags (for on/off/toggle/brightness)
	for _, cmd := range []*cobra.Command{deviceGroupOnCmd, deviceGroupOffCmd, deviceGroupToggleCmd} {
		cmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
		cmd.MarkFlagRequired("group")
	}
	deviceGroupBrightnessCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
	deviceGroupBrightnessCmd.MarkFlagRequired("group")
	deviceGroupBrightnessCmd.Flags().Uint8Var(&brightnessLevel, "level", 0, "Brightness level (0-254)")
	deviceGroupBrightnessCmd.MarkFlagRequired("level")
	deviceGroupBrightnessCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")
	deviceGroupSceneCmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (1-65535)")
	deviceGroupSceneCmd.MarkFlagRequired("group")
	deviceGroupSceneCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	deviceGroupSceneCmd.MarkFlagRequired("scene")

	// Add port/baud flags to all device commands
	for _, cmd := range []*cobra.Command{
		deviceGroupAddCmd, deviceGroupRemoveCmd, deviceGroupListCmd,
		deviceGroupOnCmd, deviceGroupOffCmd, deviceGroupToggleCmd, deviceGroupBrightnessCmd, deviceGroupSceneCmd,
	} {
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}

	// Build group subcommand hierarchy
	deviceGroupCmd.AddCommand(deviceGroupAddCmd)
	deviceGroupCmd.AddCommand(deviceGroupRemoveCmd)
	deviceGroupCmd.AddCommand(deviceGroupListCmd)
	deviceGroupCmd.AddCommand(deviceGroupOnCmd)
	deviceGroupCmd.AddCommand(deviceGroupOffCmd)
	deviceGroupCmd.AddCommand(deviceGroupToggleCmd)
	deviceGroupCmd.AddCommand(deviceGroupBrightnessCmd)
	deviceGroupCmd.AddCommand(deviceGroupSceneCmd)

	// Register group command to device command
	deviceCmd.AddCommand(deviceGroupCmd)
}

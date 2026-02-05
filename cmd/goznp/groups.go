package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
)

// Group-related global variables
var removeAllGroups bool

// Groups command and subcommands
var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Control device groups",
	Long: `Commands for managing Zigbee device groups.

Groups allow controlling multiple devices with a single multicast command.
This is much faster than sending individual commands to each device.

Commands:
  list       List groups a device belongs to
  add        Add device to a group
  remove     Remove device from a group
  on/off      Turn group on/off
  toggle     Toggle group state
  brightness Set brightness for group
  recall     Recall scene on group

Examples:
  goznp groups on --group 1
  goznp groups add --addr 0x1234 --group 1
  goznp groups list --addr 0x1234`,
}

// goznp groups add --addr <nwk-addr> --group <id> [--name <name>]
var groupsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add device to a group",
	Long: `Add a device to a Zigbee group.

Group IDs are 16-bit values (1-65535). Group 0 is reserved.

The device can be specified by:
  - Custom name: --name "Kitchen Light"
  - Network address: --addr 0x1234

Examples:
  goznp groups add --name "Kitchen Light" --group 1
  goznp groups add --addr 0x1234 --group 1 --name "Living Room"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupAdd(ctx)
	},
}

func runDeviceGroupAdd(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)\n\n" +
			"Usage:\n" +
			"  goznp groups add --name \"<name>\" --group <id>\n")
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

	// Resolve device address (supports --name, --ieee, --addr)
	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Adding device to group %d", groupID)
	if groupName != "" {
		fmt.Printf(" (%s)", groupName)
	}
	fmt.Println("...")

	if err := a.AddToGroup(ctx, nwkAddr, deviceEndpoint, groupID, groupName); err != nil {
		return fmt.Errorf("failed to add to group: %w\n\n"+
			"Troubleshooting:\n"+
			"  - Device may support limited group memberships\n"+
			"  - Check if group ID is in valid range (1-65535)",
			err)
	}

	fmt.Printf("Device added to group %d\n", groupID)
	return nil
}

// goznp groups remove --addr <nwk-addr> --group <id>
var groupsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove device from a group",
	Long: `Remove a device from a Zigbee group.

Use --group 0 or --all to remove from all groups.

The device can be specified by:
  - Custom name: --name "Kitchen Light"
  - Network address: --addr 0x1234

Examples:
  goznp groups remove --name "Kitchen Light" --group 1
  goznp groups remove --name "Kitchen Light" --all`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupRemove(ctx)
	},
}

func runDeviceGroupRemove(ctx context.Context) error {
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

	if removeAllGroups {
		fmt.Printf("Removing device from all groups...\n")

		if err := a.RemoveFromAllGroups(ctx, nwkAddr, deviceEndpoint); err != nil {
			return fmt.Errorf("remove from all groups failed: %w", err)
		}

		fmt.Println("Device removed from all groups")
		return nil
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (or use --all)\n\n" +
			"Usage:\n" +
			"  goznp groups remove --name \"<name>\" --group <id>\n" +
			"  goznp groups remove --name \"<name>\" --all")
	}

	fmt.Printf("Removing device from group %d...\n", groupID)

	if err := a.RemoveFromGroup(ctx, nwkAddr, deviceEndpoint, groupID); err != nil {
		return fmt.Errorf("remove from group failed: %w", err)
	}

	fmt.Printf("Device removed from group %d\n", groupID)
	return nil
}

// goznp groups list --addr <nwk-addr>
var groupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List group memberships",
	Long: `Query which groups a device endpoint belongs to.

The device can be specified by:
  - Custom name: --name "Kitchen Light"
  - Network address: --addr 0x1234

Example:
  goznp groups list --name "Kitchen Light"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupList(ctx)
	},
}

func runDeviceGroupList(ctx context.Context) error {
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

	fmt.Printf("Querying group membership for device at 0x%04X endpoint %d...\n\n", nwkAddr, deviceEndpoint)

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

// goznp groups on --group <id>
var groupsOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn on all devices in a group",
	Long: `Send On command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.
This is much faster than sending individual commands to each device.

Example:
  goznp groups on --group 1`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "on")
	},
}

// goznp groups off --group <id>
var groupsOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn off all devices in a group",
	Long: `Send Off command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.

Example:
  goznp groups off --group 1`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "off")
	},
}

// goznp groups toggle --group <id>
var groupsToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle all devices in a group",
	Long: `Send Toggle command to all devices in a Zigbee group.

This uses multicast addressing so all group members receive the command simultaneously.

Example:
  goznp groups toggle --group 1`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupCommand(ctx, "toggle")
	},
}

func runDeviceGroupCommand(ctx context.Context, action string) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)\n\n" +
			"Usage:\n" +
			"  goznp groups <action> --group <id>")
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

// goznp groups brightness --group <id> --level <0-254>
var groupsBrightnessCmd = &cobra.Command{
	Use:   "brightness",
	Short: "Set brightness for all devices in a group",
	Long: `Set brightness level on all dimmable devices in a group.

Level values:
  0   = Off (will also turn off devices)
  1   = Minimum brightness
  254 = Maximum brightness

Transition time is in milliseconds (default: 0 for instant).

Examples:
  goznp groups brightness --group 1 --level 128                    # Set to 50%
  goznp groups brightness --group 1 --level 254 --transition 1000  # Fade to max over 1s`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupBrightness(ctx)
	},
}

func runDeviceGroupBrightness(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)\n\n" +
			"Usage:\n" +
			"  goznp groups brightness --group <id> --level <0-254>")
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

// goznp groups recall --group <id> --scene <id>
var groupsRecallSceneCmd = &cobra.Command{
	Use:   "recall",
	Short: "Recall scene on all devices in a group",
	Long: `Recall a scene on all devices in a Zigbee group.

This sends the RecallScene command to all group members simultaneously.
The devices will transition to their saved scene states.

Example:
  goznp groups recall --group 1 --scene 1`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceGroupScene(ctx)
	},
}

func runDeviceGroupScene(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	if sceneID == 0 {
		return fmt.Errorf("--scene is required (0-255)\n\n" +
			"Usage:\n" +
			"  goznp groups recall --group <id> --scene <id>")
	}

	if groupID == 0 {
		return fmt.Errorf("--group is required (1-65535)\n\n" +
			"Usage:\n" +
			"  goznp groups recall --group <id> --scene <id>")
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

	fmt.Printf("Recalling scene %d on group %d...\n", sceneID, groupID)

	if err := a.GroupRecallScene(ctx, groupID, sceneID); err != nil {
		return fmt.Errorf("recall scene failed: %w", err)
	}

	fmt.Printf("Scene %d recalled on group %d\n", sceneID, groupID)
	return nil
}

func init() {
	AddConnectionFlags(groupsAddCmd)
	AddConnectionFlags(groupsRemoveCmd)
	AddConnectionFlags(groupsListCmd)
	AddConnectionFlags(groupsOnCmd)
	AddConnectionFlags(groupsOffCmd)
	AddConnectionFlags(groupsToggleCmd)
	AddConnectionFlags(groupsBrightnessCmd)
	AddConnectionFlags(groupsRecallSceneCmd)

	AddDeviceFlags(groupsAddCmd)
	AddDeviceFlags(groupsRemoveCmd)
	AddDeviceFlags(groupsListCmd)

	AddGroupFlags(groupsAddCmd)
	AddGroupFlags(groupsRemoveCmd)
	AddGroupFlags(groupsOnCmd)
	AddGroupFlags(groupsOffCmd)
	AddGroupFlags(groupsToggleCmd)
	AddGroupFlags(groupsBrightnessCmd)
	AddGroupFlags(groupsRecallSceneCmd)

	// Mark required flags
	groupsAddCmd.MarkFlagRequired("group")
	groupsRemoveCmd.Flags().BoolVar(&removeAllGroups, "all", false, "Remove from all groups")
	groupsOnCmd.MarkFlagRequired("group")
	groupsOffCmd.MarkFlagRequired("group")
	groupsToggleCmd.MarkFlagRequired("group")
	groupsBrightnessCmd.MarkFlagRequired("group")
	groupsBrightnessCmd.Flags().Uint8Var(&brightnessLevel, "level", 0, "Brightness level (0-254)")
	groupsBrightnessCmd.MarkFlagRequired("level")
	groupsBrightnessCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")
	groupsRecallSceneCmd.MarkFlagRequired("group")
	groupsRecallSceneCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	groupsRecallSceneCmd.MarkFlagRequired("scene")

	// Build groups command hierarchy
	groupsCmd.AddCommand(groupsAddCmd)
	groupsCmd.AddCommand(groupsRemoveCmd)
	groupsCmd.AddCommand(groupsListCmd)
	groupsCmd.AddCommand(groupsOnCmd)
	groupsCmd.AddCommand(groupsOffCmd)
	groupsCmd.AddCommand(groupsToggleCmd)
	groupsCmd.AddCommand(groupsBrightnessCmd)
	groupsRecallSceneCmd.Use = "scene" // Already set, just confirming
	groupsCmd.AddCommand(groupsRecallSceneCmd)
}

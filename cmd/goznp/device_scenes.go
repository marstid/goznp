package main

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/spf13/cobra"
)

var removeAllScenes bool

// Scenes Cluster Commands

// Parent command for scene operations
var deviceSceneCmd = &cobra.Command{
	Use:   "scene",
	Short: "Manage device scenes",
	Long: `Store, recall, and manage scenes on a device.

Scenes save device state (brightness, color, etc.) that can be recalled instantly.
Scenes are stored on the device itself, so they work even when the coordinator is offline.`,
}

// goznp device scene store --addr <nwk-addr> --group <id> --scene <id>
var deviceSceneStoreCmd = &cobra.Command{
	Use:   "store",
	Short: "Store current state as a scene",
	Long: `Store the device's current state as a scene.

The device saves its current attribute values (brightness, color, on/off state, etc.)
to the specified scene ID within the group.

Note: The device must be a member of the group before storing a scene.

Examples:
  goznp device scene store --addr 0xBE87 --endpoint 11 --group 1 --scene 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneStore(ctx)
	},
}

func runDeviceSceneStore(ctx context.Context) error {
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

	fmt.Printf("Storing current state as scene %d in group %d on device 0x%04X endpoint %d...\n",
		sceneID, groupID, nwkAddr, deviceEndpoint)

	if err := a.StoreScene(ctx, nwkAddr, deviceEndpoint, groupID, sceneID); err != nil {
		return fmt.Errorf("store scene failed: %w", err)
	}

	fmt.Printf("Scene %d stored successfully\n", sceneID)
	return nil
}

// goznp device scene recall --addr <nwk-addr> --group <id> --scene <id>
var deviceSceneRecallCmd = &cobra.Command{
	Use:   "recall",
	Short: "Recall/activate a scene",
	Long: `Recall a previously stored scene.

The device transitions to the saved attribute values for the specified scene.

Examples:
  goznp device scene recall --addr 0xBE87 --endpoint 11 --group 1 --scene 1
  goznp device scene recall --addr 0xBE87 --endpoint 11 --group 1 --scene 1 --transition 1000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneRecall(ctx)
	},
}

func runDeviceSceneRecall(ctx context.Context) error {
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

	fmt.Printf("Recalling scene %d from group %d on device 0x%04X endpoint %d", sceneID, groupID, nwkAddr, deviceEndpoint)

	var transTime *uint16
	if transitionTime > 0 {
		// Convert ms to tenths of a second
		tenths := transitionTime / 100
		transTime = &tenths
		fmt.Printf(" (transition: %dms)", transitionTime)
	}
	fmt.Println("...")

	if err := a.RecallScene(ctx, nwkAddr, deviceEndpoint, groupID, sceneID, transTime); err != nil {
		return fmt.Errorf("recall scene failed: %w", err)
	}

	fmt.Printf("Scene %d recalled\n", sceneID)
	return nil
}

// goznp device scene remove --addr <nwk-addr> --group <id> [--scene <id>] [--all]
var deviceSceneRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a scene",
	Long: `Remove a scene from the device.

Use --scene to remove a specific scene, or --all to remove all scenes for the group.

Examples:
  goznp device scene remove --addr 0xBE87 --endpoint 11 --group 1 --scene 1
  goznp device scene remove --addr 0xBE87 --endpoint 11 --group 1 --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneRemove(ctx)
	},
}

func runDeviceSceneRemove(ctx context.Context) error {
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

	if removeAllScenes {
		fmt.Printf("Removing all scenes for group %d on device 0x%04X endpoint %d...\n",
			groupID, nwkAddr, deviceEndpoint)
		if err := a.RemoveAllScenes(ctx, nwkAddr, deviceEndpoint, groupID); err != nil {
			return fmt.Errorf("remove all scenes failed: %w", err)
		}
		fmt.Println("All scenes removed")
		return nil
	}

	fmt.Printf("Removing scene %d from group %d on device 0x%04X endpoint %d...\n",
		sceneID, groupID, nwkAddr, deviceEndpoint)

	if err := a.RemoveScene(ctx, nwkAddr, deviceEndpoint, groupID, sceneID); err != nil {
		return fmt.Errorf("remove scene failed: %w", err)
	}

	fmt.Printf("Scene %d removed\n", sceneID)
	return nil
}

// goznp device scene list --addr <nwk-addr> --group <id>
var deviceSceneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scenes for a group",
	Long: `Query which scenes are stored on the device for a group.

Example:
  goznp device scene list --addr 0xBE87 --endpoint 11 --group 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runDeviceSceneList(ctx)
	},
}

func runDeviceSceneList(ctx context.Context) error {
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

	fmt.Printf("Querying scenes for group %d on device 0x%04X endpoint %d...\n\n",
		groupID, nwkAddr, deviceEndpoint)

	membership, err := a.GetSceneMembership(ctx, nwkAddr, deviceEndpoint, groupID)
	if err != nil {
		return fmt.Errorf("get scene membership failed: %w", err)
	}

	if membership.Status != 0 {
		return fmt.Errorf("device returned error status 0x%02X", membership.Status)
	}

	if len(membership.Scenes) == 0 {
		fmt.Printf("No scenes stored for group %d\n", groupID)
	} else {
		fmt.Printf("Group %d has %d scene(s):\n", groupID, len(membership.Scenes))
		for _, s := range membership.Scenes {
			fmt.Printf("  - Scene %d\n", s)
		}
	}

	if membership.Capacity != 0xFF {
		fmt.Printf("\nRemaining capacity: %d scenes\n", membership.Capacity)
	}

	return nil
}

func init() {
	// Device scene flags
	for _, cmd := range []*cobra.Command{deviceSceneStoreCmd, deviceSceneRecallCmd, deviceSceneRemoveCmd, deviceSceneListCmd} {
		cmd.Flags().StringVar(&deviceAddr, "addr", "", "Device network address (hex, e.g., 0x1234)")
		cmd.MarkFlagRequired("addr")
		cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "Device endpoint")
		cmd.Flags().Uint16Var(&groupID, "group", 0, "Group ID (0 for global scenes)")
		cmd.MarkFlagRequired("group")
		cmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
		cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	}
	deviceSceneStoreCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	deviceSceneStoreCmd.MarkFlagRequired("scene")
	deviceSceneRecallCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID (0-255)")
	deviceSceneRecallCmd.MarkFlagRequired("scene")
	deviceSceneRecallCmd.Flags().Uint16Var(&transitionTime, "transition", 0, "Transition time in milliseconds")
	deviceSceneRemoveCmd.Flags().Uint8Var(&sceneID, "scene", 0, "Scene ID to remove")
	deviceSceneRemoveCmd.Flags().BoolVar(&removeAllScenes, "all", false, "Remove all scenes for the group")

	// Build scene subcommand hierarchy
	deviceSceneCmd.AddCommand(deviceSceneStoreCmd)
	deviceSceneCmd.AddCommand(deviceSceneRecallCmd)
	deviceSceneCmd.AddCommand(deviceSceneRemoveCmd)
	deviceSceneCmd.AddCommand(deviceSceneListCmd)

	// Register scene command with device command
	deviceCmd.AddCommand(deviceSceneCmd)
}

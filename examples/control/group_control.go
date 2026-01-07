//go:build ignore

// Example: Control multiple devices using Zigbee group messaging.
// This demonstrates adding devices to groups and controlling them simultaneously.
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
	// Get serial port from environment
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
		fmt.Printf("GOZNP_PORT not set, using default: %s\n", port)
	}

	// Group ID to control
	groupIDArg := os.Getenv("GROUP_ID")
	if groupIDArg == "" && len(os.Args) > 1 {
		groupIDArg = os.Args[1]
	}
	if groupIDArg == "" {
		groupIDArg = "1" // Default group 1
	}

	var groupID uint16
	if _, err := fmt.Sscanf(groupIDArg, "0x%04X", &groupID); err != nil {
		fmt.Sscanf(groupIDArg, "%d", &groupID)
	}

	// Get command
	command := os.Getenv("COMMAND")
	if command == "" && len(os.Args) > 2 {
		command = os.Args[2]
	}

	// Create and open adapter connection
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	fmt.Printf("=== Group Control (Group 0x%04X) ===\n\n", groupID)

	// Execute the requested command
	switch command {
	case "add":
		addDeviceToGroup(ctx, adptr)

	case "remove":
		removeDeviceFromGroup(ctx, adptr)

	case "view":
		viewGroupMembers(ctx, adptr, groupID)

	case "on", "off", "toggle":
		groupOnOff(ctx, adptr, groupID, command)

	case "identify":
		groupIdentify(ctx, adptr, groupID)

	default:
		printUsage(os.Args[0])
	}
}

func addDeviceToGroup(ctx context.Context, adptr *adapter.Adapter) {
	// Get device address
	dAddr := os.Getenv("DEVICE_ADDR")
	if dAddr == "" {
		fmt.Println("For add/remove, set DEVICE_ADDR environment variable")
		fmt.Println("Example: DEVICE_ADDR=0x1234 go run group_control.go add")
		return
	}

	var nwkAddr uint16
	fmt.Sscanf(dAddr, "%d", &nwkAddr)

	dEndpoint := os.Getenv("ENDPOINT")
	endpoint := uint8(1)
	if dEndpoint != "" {
		fmt.Sscanf(dEndpoint, "%d", &endpoint)
	}

	groupID := os.Getenv("GROUP_ID")
	var gid uint16
	if groupID == "" {
		gid = 1
	} else {
		fmt.Sscanf(groupID, "%d", &gid)
	}

	fmt.Printf("Adding device 0x%04X:%d to group %d\n", nwkAddr, endpoint, gid)

	// Add devices coordinator to group first (required)
	response, err := adptr.AddDeviceToGroup(ctx, adptr.GetEndpointForProfile(0x0104), gid, 0x0000, "All Lights")
	if err != nil {
		log.Printf("Error adding coordinator to group: %v", err)
	} else {
		fmt.Printf("Coordinator response: Status=0x%02X GroupID=0x%04X\n", response.Status, response.GroupID)
	}

	// Add the specified device to the group
	response, err = adptr.AddDeviceToGroup(ctx, nwkAddr, endpoint, gid, "My Group")
	if err != nil {
		log.Printf("Error adding device to group: %v", err)
	} else {
		fmt.Printf("Device response: Status=0x%02X GroupID=0x%04X\n", response.Status, response.GroupID)
		if response.Status == zcl.StatusSuccess {
			fmt.Println("Success! Device added to group")
		} else {
			fmt.Printf("Failed with status: %s\n", statusString(response.Status))
		}
	}
}

func removeDeviceFromGroup(ctx context.Context, adptr *adapter.Adapter) {
	// Get device address
	dAddr := os.Getenv("DEVICE_ADDR")
	if dAddr == "" {
		fmt.Println("For add/remove, set DEVICE_ADDR environment variable")
		fmt.Println("Example: DEVICE_ADDR=0x1234 GO run group_control.go remove")
		return
	}

	var nwkAddr uint16
	fmt.Sscanf(dAddr, "%d", &nwkAddr)

	dEndpoint := os.Getenv("ENDPOINT")
	endpoint := uint8(1)
	if dEndpoint != "" {
		fmt.Sscanf(dEndpoint, "%d", &endpoint)
	}

	groupID := os.Getenv("GROUP_ID")
	var gid uint16
	if groupID == "" {
		gid = 1
	} else {
		fmt.Sscanf(groupID, "%d", &gid)
	}

	fmt.Printf("Removing device 0x%04X:%d from group %d\n", nwkAddr, endpoint, gid)

	// Remove device from group
	response, err := adptr.RemoveDeviceFromGroup(ctx, nwkAddr, endpoint, gid)
	if err != nil {
		log.Printf("Error removing device from group: %v", err)
	} else {
		fmt.Printf("Response: Status=0x%02X GroupID=0x%04X\n", response.Status, response.GroupID)
		if response.Status == zcl.StatusSuccess {
			fmt.Println("Success! Device removed from group")
		} else {
			fmt.Printf("Failed with status: %s\n", statusString(response.Status))
		}
	}
}

func viewGroupMembers(ctx context.Context, adptr *adapter.Adapter, groupID uint16) {
	fmt.Printf("Devices in group %d:\n\n", groupID)

	// Read membership from coordinator endpoint (group name)
	cEP := adptr.GetEndpointForProfile(0x0104)
	names, err := adptr.GetGroupNames(ctx, cEP)
	if err == nil {
		for _, name := range names {
			if name.GroupID == groupID {
				fmt.Printf("From coordinator: %s\n", name.Name)
				break
			}
		}
	}

	// List all devices and check their group membership
	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
		return
	}

	for _, dev := range devices {
		groups, err := adptr.GetDeviceGroups(ctx, dev.NwkAddr)
		if err != nil {
			continue
		}

		for _, g := range groups {
			if g.GroupID == groupID {
				fmt.Printf("  Device: 0x%04X (%s %s)\n", dev.NwkAddr, dev.Manufacturer, dev.Model)
				if devName, _ := adptr.GetDeviceName(ctx, dev.IEEEAddr); devName != nil {
					fmt.Printf("    Name: %s\n", devName.FriendlyName)
				}
				break
			}
		}
	}
}

func groupOnOff(ctx context.Context, adptr *adapter.Adapter, groupID uint16, command string) {
	cEP := adptr.GetEndpointForProfile(0x0104)

	switch command {
	case "on":
		fmt.Printf("Turning on group %d\n", groupID)
		if err := adptr.GroupOn(ctx, groupID, cEP); err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Println("Success")
		}

	case "off":
		fmt.Printf("Turning off group %d\n", groupID)
		if err := adptr.GroupOff(ctx, groupID, cEP); err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Println("Success")
		}

	case "toggle":
		fmt.Printf("Toggling group %d\n", groupID)
		if err := adptr.GroupToggle(ctx, groupID, cEP); err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Println("Success")
		}
	}
}

func groupIdentify(ctx context.Context, adptr *adapter.Adapter, groupID uint16) {
	duration := uint16(10) // 10 seconds
	if arg := os.Getenv("DURATION"); arg != "" {
		fmt.Sscanf(arg, "%d", &duration)
	}

	cEP := adptr.GetEndpointForProfile(0x0104)

	fmt.Printf("Identifying group %d for %d seconds\n", groupID, duration)
	if err := adptr.GroupIdentify(ctx, groupID, cEP, duration); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("Success - devices in group will flash/beep")
	}
}

func printUsage(progName string) {
	fmt.Println("Usage for group control:")
	fmt.Println()
	fmt.Println("Add device to group:")
	fmt.Println("  GROUP_ID=1 DEVICE_ADDR=0x1234 ENDPOINT=1 go run group_control.go add")
	fmt.Println()
	fmt.Println("Remove device from group:")
	fmt.Println("  GROUP_ID=1 DEVICE_ADDR=0x1234 ENDPOINT=1 go run group_control.go remove")
	fmt.Println()
	fmt.Println("View group members:")
	fmt.Println("  go run group_control.go view")
	fmt.Println("  GROUP_ID=2 go run group_control.go view")
	fmt.Println()
	fmt.Println("Control group:")
	fmt.Println("  go run group_control.go on")
	fmt.Println("  go run group_control.go off")
	fmt.Println("  go run group_control.go toggle")
	fmt.Println()
	fmt.Println("Identify group (devices flash):")
	fmt.Println("  go run group_control.go identify")
	fmt.Println("  DURATION=30 go run group_control.go identify")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  GOZNP_PORT     - Serial port (default: /dev/ttyUSB0)")
	fmt.Println("  GROUP_ID      - Group ID to control (default: 1)")
	fmt.Println("  DEVICE_ADDR   - Device network address for add/remove")
	fmt.Println("  ENDPOINT      - Device endpoint (default: 1)")
	fmt.Println("  DURATION      - Identify duration in seconds (default: 10)")
}

func statusString(status uint8) string {
	switch status {
	case zcl.StatusSuccess:
		return "Success"
	case zcl.StatusUnsupportedCommand:
		return "Unsupported Command"
	case zcl.StatusUnsupportedCluster:
		return "Unsupported Cluster"
	case zcl.StatusUnsupportedAttribute:
		return "Unsupported Attribute"
	case zcl.StatusInvalidField:
		return "Invalid Field"
	case zcl.StatusInvalidValue:
		return "Invalid Value"
	case zcl.StatusReadOnly:
		return "Read Only"
	case zcl.StatusInsufficientSpace:
		return "Insufficient Space"
	case zcl.StatusDuplicateExists:
		return "Duplicate Exists"
	case zcl.StatusNotFound:
		return "Not Found"
	case zcl.StatusUnreportableAttribute:
		return "Unreportable Attribute"
	case zcl.StatusHardwareFailure:
		return "Hardware Failure"
	case zcl.StatusMalformedCommand:
		return "Malformed Command"
	case zcl.StatusUnsupportedGeneralCommand:
		return "Unsupported General Command"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", status)
	}
}

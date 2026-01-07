//go:build ignore

// Example: Interview a device to discover its capabilities.
// This demonstrates how to query a device for its endpoints, clusters, and attributes.
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
	// Get serial port from environment variable
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
		fmt.Printf("GOZNP_PORT not set, using default: %s\n", port)
	}

	// Device to interview (network address)
	// Use device_list.go or 'goznp device list' to find addresses
	nwkAddrArg := os.Getenv("DEVICE_ADDR")
	if nwkAddrArg == "" {
		fmt.Println("Usage: DEVICE_ADDR=0x1234 go run device_interview.go")
		fmt.Println("Set DEVICE_ADDR to the network address of the device to interview")
		return
	}

	var nwkAddr uint16
	if _, err := fmt.Sscanf(nwkAddrArg, "0x%04X", &nwkAddr); err != nil {
		// Try as decimal
		if _, err := fmt.Sscanf(nwkAddrArg, "%d", &nwkAddr); err != nil {
			log.Fatalf("Invalid device address: %v", err)
		}
	}

	// Create and open adapter connection
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	fmt.Printf("=== Interviewing device at 0x%04X ===\n\n", nwkAddr)

	// Get device from adapter
	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		log.Fatalf("Failed to get devices: %v", err)
	}

	var target *adapter.Device
	for _, d := range devices {
		if d.NwkAddr == nwkAddr {
			target = d
			break
		}
	}

	if target == nil {
		log.Fatalf("Device 0x%04X not found. Pair it first with 'goznp device permit-join'", nwkAddr)
	}

	// Show basic device info
	fmt.Printf("IEEE Address: %s\n", formatIEEEAddr(target.IEEEAddr))
	fmt.Printf("Manufacturer: %s\n", target.Manufacturer)
	fmt.Printf("Model: %s\n", target.Model)
	fmt.Println()

	// Interview the device
	fmt.Println("Starting device interview...")
	if err := adptr.InterviewDevice(ctx, target); err != nil {
		log.Printf("Interview failed: %v", err)
		fmt.Println("Device may be asleep or some clusters may be unavailable")
	} else {
		fmt.Println("Interview completed successfully")
	}
	fmt.Println()

	// Refresh device data and show results
	devices, _ = adptr.GetDevices(ctx)
	for _, d := range devices {
		if d.IEEEAddr == target.IEEEAddr {
			target = d
			break
		}
	}

	fmt.Println("=== Discovered Endpoints ===")
	if len(target.Endpoints) == 0 {
		fmt.Println("No endpoints discovered")
	} else {
		for _, ep := range target.Endpoints {
			fmt.Printf("\nEndpoint %d\n", ep.ID)
			fmt.Printf("  Profile: 0x%04X (%s)\n", ep.ProfileID, profileName(ep.ProfileID))
			fmt.Printf("  Device: 0x%04X (%s)\n", ep.DeviceID, deviceTypeName(ep.DeviceID))
			fmt.Printf("  Device Version: %d\n", ep.DeviceVersion)

			if len(ep.InClusters) > 0 {
				fmt.Printf("  Input Clusters (%d):\n", len(ep.InClusters))
				for _, clusterID := range ep.InClusters {
					fmt.Printf("    0x%04X - %s\n", clusterID, clusterName(clusterID))

					// Try to read cluster attributes
					if testReadableCluster(clusterID) {
						if attrs, err := adptr.ReadAttributes(ctx, nwkAddr, ep.ID, clusterID); err == nil && len(attrs) > 0 {
							fmt.Printf("      Attributes: ")
							for i, attr := range attrs {
								if i > 0 {
									fmt.Print(", ")
								}
								fmt.Printf("%s=0x%04X", attrIDName(clusterID, attr.AttributeID), attr.AttributeID)
							}
							fmt.Println()
						}
					}
				}
			}

			if len(ep.OutClusters) > 0 {
				fmt.Printf("  Output Clusters (%d):\n", len(ep.OutClusters))
				for _, clusterID := range ep.OutClusters {
					fmt.Printf("    0x%04X - %s\n", clusterID, clusterName(clusterID))
				}
			}
		}
	}

	fmt.Println("\n=== Interview Complete ===")
	fmt.Println("Use the device_list.go example to see all discovered devices")
}

// testReadableCluster returns true if we can try reading from this cluster
func testReadableCluster(clusterID uint16) bool {
	switch clusterID {
	case 0x0000, 0x0001, 0x0003, 0x0006, 0x0008: // Basic, Power, Identify, OnOff, Level
		return true
	}
	return false
}

// formatIEEEAddr formats an 8-byte IEEE address as a hex string with colons
func formatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X",
		addr[7], addr[6], addr[5], addr[4], addr[3], addr[2], addr[1], addr[0])
}

// profileName returns a human-readable name for Zigbee profiles
func profileName(profileID uint16) string {
	switch profileID {
	case 0x0104:
		return "Home Automation"
	case 0x0105:
		return "Home Automation Lite"
	case 0x0107:
		return "Smart Energy"
	case 0x0109:
		return "Remote Control"
	case 0x010A:
		return "Light Link"
	default:
		return fmt.Sprintf("Unknown 0x%04X", profileID)
	}
}

// deviceTypeName returns a human-readable name for Zigbee device types
func deviceTypeName(deviceID uint16) string {
	switch deviceID {
	case 0x0000:
		return "On/Off Switch"
	case 0x0001:
		return "Level Control Switch"
	case 0x0002:
		return "On/Off Output"
	case 0x0003:
		return "Level Control Output"
	case 0x0006:
		return "Remote Control"
	case 0x0008:
		return "Combination Interface"
	case 0x0010:
		return "On/Off Light"
	case 0x0011:
		return "Dimmer Light"
	case 0x0020:
		return "Color Dimmer Light"
	case 0x0022:
		return "Extended Color Light"
	case 0x0050:
		return "Binary Input"
	case 0x0051:
		return "Binary Output"
	case 0x0052:
		return "Analog Input"
	case 0x0053:
		return "Analog Output"
	case 0x0085:
		return "Color Temperature Light"
	default:
		return fmt.Sprintf("Unknown 0x%04X", deviceID)
	}
}

// clusterName returns a human-readable name for Zigbee clusters
func clusterName(clusterID uint16) string {
	switch clusterID {
	case 0x0000:
		return "Basic"
	case 0x0001:
		return "Power Configuration"
	case 0x0002:
		return "Device Temperature"
	case 0x0003:
		return "Identify"
	case 0x0004:
		return "Groups"
	case 0x0005:
		return "Scenes"
	case 0x0006:
		return "On/Off"
	case 0x0008:
		return "Level Control"
	case 0x0101:
		return "Door Lock"
	case 0x0201:
		return "Thermostat"
	case 0x0300:
		return "Color Control"
	case 0x0400:
		return "Illuminance Measurement"
	case 0x0402:
		return "Temperature Measurement"
	case 0x0403:
		return "Pressure Measurement"
	case 0x0405:
		return "Humidity Measurement"
	case 0x0406:
		return "Occupancy Sensing"
	case 0x0500:
		return "IAS Zone"
	case 0x0502:
		return "IAS WD"
	case 0x0702:
		return "Simple Metering"
	case 0x0B04:
		return "Electrical Measurement"
	default:
		return fmt.Sprintf("Unknown 0x%04X", clusterID)
	}
}

// attrIDName returns a human-readable name for common attributes
func attrIDName(clusterID, attrID uint16) string {
	if clusterID == zcl.ClusterBasic {
		switch attrID {
		case 0x0004:
			return "Manufacturer"
		case 0x0005:
			return "Model"
		}
	}
	return "Unknown"
}

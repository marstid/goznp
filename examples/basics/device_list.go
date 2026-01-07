//go:build ignore

// Example: List all paired devices on the Zigbee network.
// This demonstrates device discovery and basic device information retrieval.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
)

func main() {
	// Get serial port from environment variable
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0" // Default for Linux
		fmt.Printf("GOZNP_PORT not set, using default: %s\n", port)
	}

	// Create and open adapter connection
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	// Show adapter information
	info, err := adptr.GetInfo(ctx)
	if err != nil {
		log.Printf("Failed to get adapter info: %v", err)
	} else {
		fmt.Printf("=== Adapter Info ===\n")
		fmt.Printf("Transport: %s\n", info.Version.TransportRev)
		fmt.Printf("Product: %d.%d.%d\n", info.Version.Product, info.Version.MajorRel, info.Version.MinorRel)
		fmt.Printf("Capabilities: 0x%04X\n", info.Capabilities)
		fmt.Println()
	}

	// List all devices
	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		log.Fatalf("Failed to get devices: %v", err)
	}

	fmt.Printf("=== %d Devices ===\n\n", len(devices))

	if len(devices) == 0 {
		fmt.Println("No devices paired. Put devices in pairing mode first:")
		fmt.Println("  goznp device permit-join --seconds 60")
		return
	}

	// Print device information
	for _, dev := range devices {
		fmt.Printf("Device:\n")
		fmt.Printf("  IEEE: %s\n", formatIEEEAddr(dev.IEEEAddr))
		fmt.Printf("  Network Address: 0x%04X\n", dev.NwkAddr)
		fmt.Printf("  Manufacturer: %s\n", dev.Manufacturer)
		fmt.Printf("  Model: %s\n", dev.Model)
		fmt.Printf("  Status: %s\n", dev.State)

		// Get device name if set
		if name, _ := adptr.GetDeviceName(ctx, dev.IEEEAddr); name != nil {
			fmt.Printf("  Name: %s\n", name.FriendlyName)
			if name.Description != "" {
				fmt.Printf("  Description: %s\n", name.Description)
			}
		}

		// Show endpoints and clusters if interviewed
		if len(dev.Endpoints) > 0 {
			fmt.Printf("  Endpoints: %d\n", len(dev.Endpoints))
			for _, ep := range dev.Endpoints {
				fmt.Printf("    Endpoint %d:\n", ep.ID)
				if len(ep.InClusters) > 0 {
					fmt.Printf("      Input Clusters: %d\n", len(ep.InClusters))
					for _, cluster := range ep.InClusters {
						fmt.Printf("        0x%04X - %s\n", cluster, clusterName(cluster))
					}
				}
				if len(ep.OutClusters) > 0 {
					fmt.Printf("      Output Clusters: %d\n", len(ep.OutClusters))
					for _, cluster := range ep.OutClusters {
						fmt.Printf("        0x%04X - %s\n", cluster, clusterName(cluster))
					}
				}
			}
		}

		fmt.Printf("  Last Seen: %s\n", dev.LastSeen.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

// formatIEEEAddr formats an 8-byte IEEE address as a hex string with colons
func formatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X",
		addr[7], addr[6], addr[5], addr[4], addr[3], addr[2], addr[1], addr[0])
}

// clusterName returns a human-readable name for common Zigbee clusters
func clusterName(clusterID uint16) string {
	switch clusterID {
	case 0x0000:
		return "Basic"
	case 0x0001:
		return "Power Configuration"
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
	case 0x0702:
		return "Simple Metering"
	case 0x0B04:
		return "Electrical Measurement"
	default:
		return "Unknown"
	}
}

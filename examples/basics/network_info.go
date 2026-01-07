//go:build ignore

// Example: Get network status and coordinator information.
// This demonstrates how to retrieve network details and verify coordinator state.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/znp"
)

func main() {
	// Get serial port from environment variable
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
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

	// Get comprehensive adapter information
	info, err := adptr.GetInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get adapter info: %v", err)
	}

	fmt.Println("=== Coordinator Information ===")
	fmt.Printf("Transport Revision: %s\n", info.Version.TransportRev)
	fmt.Printf("Product ID: 0x%04X\n", info.Version.Product)
	fmt.Printf("Product Name: %s\n", productName(info.Version.Product))
	fmt.Printf("Major Release: %d\n", info.Version.MajorRel)
	fmt.Printf("Minor Release: %d\n", info.Version.MinorRel)
	fmt.Printf("Maintenance Release: %d\n", info.Version.MaintRel)
	fmt.Printf("Capabilities: 0x%04X\n", info.Capabilities)
	fmt.Printf("  - APP_CNF: %v\n", (info.Capabilities&znp.CapAppCnf) != 0)
	fmt.Printf("  - UTIL:    %v\n", (info.Capabilities&znp.CapUtil) != 0)
	fmt.Printf("  - ZDO:     %v\n", (info.Capabilities&znp.CapZDO) != 0)
	fmt.Printf("  - AF:      %v\n", (info.Capabilities&znp.CapAf) != 0)
	fmt.Printf("  - SYS:     %v\n", (info.Capabilities&znp.CapSys) != 0)
	fmt.Println()

	// Get network details
	nwkInfo, err := adptr.GetNetworkInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get network info: %v", err)
	}

	fmt.Println("=== Network Information ===")
	fmt.Printf("Short Address: 0x%04X\n", nwkInfo.ShortAddr)
	fmt.Printf("PAN ID: 0x%04X\n", nwkInfo.PanID)
	fmt.Printf("Extended PAN ID: %s\n", formatIEEEAddr(nwkInfo.ExtendedPanID))
	fmt.Printf("Channel: %d\n", nwkInfo.Channel)
	fmt.Printf("Device State: %s\n", deviceStateName(nwkInfo.DevState))
	fmt.Printf("Parent Address: 0x%04X\n", nwkInfo.ParentAddr)
	fmt.Println()

	// Get registered profiles
	profiles := adptr.RegisteredProfiles()
	fmt.Println("=== Registered Profiles ===")
	for _, p := range profiles {
		fmt.Printf("  %s (0x%04X)\n", p, p)
	}
	fmt.Println()

	// List devices
	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
	} else {
		fmt.Printf("=== %d Devices Paired ===\n", len(devices))
		fmt.Println()

		// Count devices by type
		byType := make(map[string]int)
		for _, dev := range devices {
			devType := "Other"
			if dev.Model != "" {
				devType = dev.Model
			}
			byType[devType]++
		}

		fmt.Println("Devices by type:")
		for devType, count := range byType {
			fmt.Printf("  %s: %d\n", devType, count)
		}
	}

	// Check if network is formed
	if nwkInfo.DevState&0x0F == 9 { // Coordinator state (DevStateCoord)
		fmt.Println("\nNetwork Status: FORMED - Coordinator is operational")
	} else {
		fmt.Printf("\nNetwork Status: NOT FORMED - Device state: %d\n", nwkInfo.DevState)
		fmt.Println("Use 'goznp network form' to create a new network")
	}
}

// productName returns a human-readable name for Z-Stack products
func productName(product uint16) string {
	switch product {
	case 0:
		return "CC2530"
	case 1:
		return "CC2531"
	case 2:
		return "CC2530ZNP"
	case 3:
		return "CC2630"
	case 4:
		return "CC2538"
	case 5:
		return "CC2650"
	case 8:
		return "CC2652R"
	case 9:
		return "CC2652P"
	case 10:
		return "CC1352P"
	case 12:
		return "CC2652RB"
	default:
		return "Unknown"
	}
}

// formatIEEEAddr formats an 8-byte IEEE address as a hex string with colons
func formatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X",
		addr[7], addr[6], addr[5], addr[4], addr[3], addr[2], addr[1], addr[0])
}

// deviceStateName returns a human-readable name for device states
func deviceStateName(state uint8) string {
	switch state & 0x0F {
	case 0:
		return "Held (not started)"
	case 1:
		return "Initialized (not connected to network)"
	case 2:
		return "Disconnected (left network)"
	case 3:
		return "Reconnecting"
	case 8:
		return "Joined (as router/end device)"
	case 9:
		return "Coordinator (network formed)"
	case 10:
		return "Router (joined network)"
	case 11:
		return "End Device (joined network)"
	default:
		return fmt.Sprintf("Unknown (%d)", state)
	}
}

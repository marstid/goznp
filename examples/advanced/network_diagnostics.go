//go:build ignore

// Example: Perform network health diagnostics on a Zigbee network.
// This demonstrates checking network status, signal strength, and device connectivity.
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
	// Get serial port from environment
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
		fmt.Printf("GOZNP_PORT not set, using default: %s\n", port)
	}

	// Create and open adapter connection
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	fmt.Println("=== Zigbee Network Diagnostics ===\n")

	// 1. Check coordinator status
	checkCoordinatorStatus(ctx, adptr)

	// 2. Check network information
	checkNetworkInfo(ctx, adptr)

	// 3. Check channel energy and quality
	checkChannelQuality(ctx, adptr)

	// 4. Check device states and liveness
	checkDeviceLiveness(ctx, adptr)

	// 5. Check for routing table issues
	checkRoutingTable(ctx, adptr)

	fmt.Println("\n=== Diagnostics Complete ===")
	fmt.Println("Summary of any issues found above")
}

func checkCoordinatorStatus(ctx context.Context, adptr *adapter.Adapter) {
	fmt.Println("1. Coordinator Status")

	info, err := adptr.GetInfo(ctx)
	if err != nil {
		log.Printf("  Error getting info: %v\n", err)
		return
	}

	fmt.Printf("  Firmware Product: 0x%04X\n", info.Version.Product)
	fmt.Printf("  Firmware Version: %d.%d.%d\n",
		info.Version.MajorRel, info.Version.MinorRel, info.Version.MaintRel)
	fmt.Printf("  Transport: %s\n", info.Version.TransportRev)

	// Check available capabilities
	if info.Capabilities&znp.CapAf == 0 {
		fmt.Println("  WARNING: AF capability not available")
	}
	if info.Capabilities&znp.CapZDO == 0 {
		fmt.Println("  WARNING: ZDO capability not available")
	}
	if info.Capabilities&znp.CapSys == 0 {
		fmt.Println("  WARNING: SYS capability not available")
	}

	// Ping coordinator
	caps, err := adptr.Ping(ctx)
	if err != nil {
		log.Printf("  ERROR: Coordinator ping failed: %v\n", err)
	} else {
		fmt.Printf("  Coordinator ping: OK (caps: 0x%04X)\n", caps.Capabilities)
	}

	fmt.Println()
}

func checkNetworkInfo(ctx context.Context, adptr *adapter.Adapter) {
	fmt.Println("2. Network Information")

	nwkInfo, err := adptr.GetNetworkInfo(ctx)
	if err != nil {
		log.Printf("  Error getting network info: %v\n", err)
		return
	}

	fmt.Printf("  Short Address: 0x%04X\n", nwkInfo.ShortAddr)
	fmt.Printf("  PAN ID: 0x%04X\n", nwkInfo.PanID)
	fmt.Printf("  Extended PAN ID: %s\n", formatIEEEAddr(nwkInfo.ExtendedPanID))
	fmt.Printf("  Channel: %d (%s)\n", nwkInfo.Channel, channelDescription(nwkInfo.Channel))
	fmt.Printf("  Device State: %s\n", deviceStateName(nwkInfo.DevState))

	// Check if coordinator is properly initialized
	if nwkInfo.ShortAddr != 0x0000 {
		fmt.Println("  WARNING: Coordinator short address is not 0x0000")
	}

	// Check for valid PAN ID
	if nwkInfo.PanID == 0x0000 || nwkInfo.PanID == 0xFFFF {
		fmt.Println("  ERROR: Invalid PAN ID - network not formed")
	}

	// Check for valid channel
	if nwkInfo.Channel < 11 || nwkInfo.Channel > 26 {
		fmt.Printf("  ERROR: Invalid channel: %d\n", nwkInfo.Channel)
	}

	fmt.Println()
}

func checkChannelQuality(ctx context.Context, adptr *adapter.Adapter) {
	fmt.Println("3. Channel Quality Analysis")

	nwkInfo, err := adptr.GetNetworkInfo(ctx)
	if err != nil {
		log.Printf("  Error: %v\n", err)
		return
	}

	currentChannel := nwkInfo.Channel

	// Get associated devices count (proxy for network size)
	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		return
	}

	fmt.Printf("  Current channel: %d\n", currentChannel)
	fmt.Printf("  Device count: %d\n", len(devices))

	// Check for channel congestion warnings
	if len(devices) > 20 {
		fmt.Printf("  NOTICE: Many devices - consider channel optimization\n")
	}

	// Get device association info
	assocInfo, err := adptr.GetAssocDevices(ctx, adptr.GetEndpointForProfile(0x0104))
	if err == nil {
		fmt.Printf("  Associated devices (neighbors): %d\n", len(assocInfo))
		for i, dev := range assocInfo {
			if i >= 5 { // Show first 5
				fmt.Printf("    ... and %d more\n", len(assocInfo)-5)
				break
			}
			fmt.Printf("    0x%04X (%s)\n", dev.NodeAddr, devRelation(dev.Relation))
		}
	}

	fmt.Println()
}

func checkDeviceLiveness(ctx context.Context, adptr *adapter.Adapter) {
	fmt.Println("4. Device Liveness Check")

	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		log.Printf("  Error: %v\n", err)
		return
	}

	now := time.Now()
	offlineCount := 0
	asleepCount := 0
	routerCount := 0
	endDeviceCount := 0

	for _, dev := range devices {
		sinceLastSeen := now.Sub(dev.LastSeen)

		// Count device types
		if dev.Capabilities&0x02 != 0 { // AC powered
			routerCount++
		} else if dev.Capabilities&0x04 != 0 { // Battery powered
			endDeviceCount++
		}

		// Check battery devices (may be asleep)
		if dev.Capabilities&0x04 != 0 { // Battery
			asleepCount++
		}

		// Check if offline (not seen in > 24 hours for routers, > 7 days for battery)
		offline := false
		threshold := 24 * time.Hour
		if dev.Capabilities&0x04 != 0 { // Battery
			threshold = 7 * 24 * time.Hour
		}

		if sinceLastSeen > threshold {
			offlineCount++
			fmt.Printf("  OFFLINE: %s %s (0x%04X) - last seen %s ago\n",
				dev.Manufacturer, dev.Model, dev.NwkAddr, formatDuration(sinceLastSeen))
		} else if sinceLastSeen > 1*time.Hour {
			fmt.Printf("  STALE: %s %s (0x%04X) - last seen %s ago\n",
				dev.Manufacturer, dev.Model, dev.NwkAddr, formatDuration(sinceLastSeen))
		}
	}

	fmt.Printf("  Total devices: %d\n", len(devices))
	fmt.Printf("  Routers (mains-powered): %d\n", routerCount)
	fmt.Printf("  End devices (battery-powered): %d\n", endDeviceCount)
	fmt.Printf("  Potentially offline: %d\n", offlineCount)
	fmt.Printf("  Battery devices (may be asleep): %d\n", asleepCount)

	if offlineCount > 0 {
		fmt.Println("  WARNING: Some devices may be offline")
	}

	if routerCount < 3 && len(devices) > 5 {
		fmt.Println("  WARNING: Few routers may affect mesh reliability")
	}

	fmt.Println()
}

func checkRoutingTable(ctx context.Context, adptr *adapter.Adapter) {
	fmt.Println("5. Network Infrastructure Check")

	nwkInfo, err := adptr.GetNetworkInfo(ctx)
	if err != nil {
		log.Printf("  Error: %v\n", err)
		return
	}

	devices, err := adptr.GetDevices(ctx)
	if err != nil {
		log.Printf("  Error: %v\n", err)
		return
	}

	routers := make([]*adapter.Device, 0)
	for _, dev := range devices {
		if dev.Capabilities&0x02 != 0 { // AC powered = router
			if dev.NwkAddr != nwkInfo.ShortAddr { // Not coordinator
				routers = append(routers, dev)
			}
		}
	}

	fmt.Printf("  Intermediate routers: %d\n", len(routers))

	// Get association info to see network topology
	assocInfo, err := adptr.GetAssocDevices(ctx, adptr.GetEndpointForProfile(0x0104))
	if err == nil {
		// Count children vs siblings
		children := 0
		siblings := 0
		for _, dev := range assocInfo {
			if dev.Relation == 10 || dev.Relation == 11 { // Child routers or end devices
				children++
			} else if dev.Relation == 8 || dev.Relation == 9 { // Sibling routers
				siblings++
			}
		}

		fmt.Printf("  Direct children of coordinator: %d\n", children)
		fmt.Printf("  Sibling routers in range: %d\n", siblings)

		if children == 0 && len(devices) > 1 {
			fmt.Println("  NOTICE: No direct children - all devices may be multi-hop")
		}
	}

	fmt.Println()
}

func formatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X",
		addr[7], addr[6], addr[5], addr[4], addr[3], addr[2], addr[1], addr[0])
}

func deviceStateName(state uint8) string {
	switch state & 0x0F {
	case 0:
		return "Held (not started)"
	case 1:
		return "Initialized (not connected)"
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

func channelDescription(channel uint8) string {
	if channel < 11 || channel > 26 {
		return "Invalid"
	}
	switch channel {
	case 11, 15, 20, 25:
		return "Recommended"
	case 12, 13, 14:
		return "Lower band"
	case 16, 17, 18, 19:
		return "Middle band"
	case 21, 22, 23, 24:
		return "Upper band"
	case 26:
		return "Often congested"
	default:
		return "Normal"
	}
}

func devRelation(relation uint8) string {
	switch relation {
	case 0:
		return "Parent"
	case 1:
		return "Child"
	case 2:
		return "Sibling"
	case 3:
		return "Unknown"
	case 4:
		return "Previous Child"
	case 7:
		return "Invalid"
	case 8:
		return "Child Routers"
	case 9:
		return "Child End Devices"
	case 10:
		return "Child (all)"
	case 11:
		return "Sibling Routers"
	case 12:
		return "Other"
	default:
		return fmt.Sprintf("Unknown (%d)", relation)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.0fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24.0)
	}
}

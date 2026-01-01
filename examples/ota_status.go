package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

// Example: Query OTA upgrade status from a Zigbee device
// This demonstrates how to read OTA cluster attributes to check firmware version
// and upgrade status.
func main() {
	// Create and open adapter connection
	// Replace with your serial port (e.g., "/dev/ttyUSB0" on Linux, "COM3" on Windows)
	adptr := adapter.New(adapter.WithSerialPath("/dev/ttyACM0"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	// Network address and endpoint of the target device
	// You would get these from device discovery
	nwkAddr := uint16(0x1234) // Example address
	endpoint := uint8(1)      // Typically endpoint 1

	// Example 1: Read full OTA information
	fmt.Println("Reading OTA information...")
	otaInfo, err := adptr.GetOTAInfo(ctx, nwkAddr, endpoint)
	if err != nil {
		log.Printf("Failed to read OTA info: %v", err)
	} else {
		fmt.Printf("Current File Version: 0x%08X\n", otaInfo.CurrentFileVersion)
		fmt.Printf("Upgrade Status: %s (%d)\n", otaInfo.UpgradeStatus, otaInfo.UpgradeStatus)
		fmt.Printf("Manufacturer ID: 0x%04X\n", otaInfo.ManufacturerID)
		fmt.Printf("Image Type ID: 0x%04X\n", otaInfo.ImageTypeID)
		fmt.Printf("File Offset: %d bytes\n", otaInfo.FileOffset)
	}

	fmt.Println()

	// Example 2: Read just the upgrade status
	fmt.Println("Checking upgrade status...")
	status, err := adptr.GetOTAUpgradeStatus(ctx, nwkAddr, endpoint)
	if err != nil {
		log.Printf("Failed to read upgrade status: %v", err)
	} else {
		fmt.Printf("Upgrade Status: %s\n", status)

		// Interpret the status
		switch status {
		case zcl.OTAStatusNormal:
			fmt.Println("Device is in normal operation (no upgrade in progress)")
		case zcl.OTAStatusDownloadInProgress:
			fmt.Println("Firmware download is in progress")
		case zcl.OTAStatusDownloadComplete:
			fmt.Println("Firmware download is complete, ready to upgrade")
		case zcl.OTAStatusWaitingToUpgrade:
			fmt.Println("Waiting to apply the upgrade")
		case zcl.OTAStatusCountDown:
			fmt.Println("Countdown to upgrade in progress")
		case zcl.OTAStatusWaitForMore:
			fmt.Println("Waiting for more data")
		}
	}

	// Example 3: Read specific OTA attributes individually
	fmt.Println("\nReading individual attributes...")
	results, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOTAUpgrade,
		zcl.AttrOTACurrentFileVersion,
		zcl.AttrOTAManufacturerID,
	)
	if err != nil {
		log.Printf("Failed to read attributes: %v", err)
	} else {
		for _, r := range results {
			if r.Status == zcl.StatusSuccess {
				switch r.AttributeID {
				case zcl.AttrOTACurrentFileVersion:
					if ver, ok := r.Value.(uint32); ok {
						fmt.Printf("Firmware Version: 0x%08X\n", ver)
					}
				case zcl.AttrOTAManufacturerID:
					if id, ok := r.Value.(uint16); ok {
						fmt.Printf("Manufacturer ID: 0x%04X\n", id)
					}
				}
			}
		}
	}
}

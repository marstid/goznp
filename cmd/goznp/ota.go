package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/ota"
	"github.com/marstid/goznp/pkg/znp"
)

var (
	// OTA command flags
	otaDirFlag     string
	otaMaxSizeFlag uint8
	otaImageFile   string
)

var otaCmd = &cobra.Command{
	Use:   "ota",
	Short: "OTA (Over-The-Air) firmware update management",
	Long:  "Manage OTA firmware updates for Zigbee devices including listing images, checking compatibility, and serving firmware",
}

var otaListCmd = &cobra.Command{
	Use:   "list [directory]",
	Short: "List available OTA firmware images",
	Long:  "List all OTA firmware images in a directory with version information",
	Args:  cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		dir := otaDirFlag
		if len(args) > 0 {
			dir = args[0]
		}
		if err := runOTAList(dir); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var otaInfoCmd = &cobra.Command{
	Use:   "info <image-file>",
	Short: "Display OTA image information",
	Long:  "Display detailed information about an OTA firmware image file",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		if err := runOTAInfo(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var otaCheckCmd = &cobra.Command{
	Use:   "check <image-file> <manufacturer> <image-type> <device-version>",
	Short: "Check if OTA image is compatible with device",
	Long:  "Check if an OTA firmware image is compatible with a device based on manufacturer code, image type, and version",
	Args:  cobra.ExactArgs(4),
	Run: func(_ *cobra.Command, args []string) {
		var manufacturer uint16
		var imageType uint16
		var deviceVersion uint32

		if _, err := fmt.Sscanf(args[1], "0x%x", &manufacturer); err != nil {
			if _, err := fmt.Sscanf(args[1], "%d", &manufacturer); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid manufacturer code: %s\n", args[1])
				os.Exit(1)
			}
		}
		if _, err := fmt.Sscanf(args[2], "0x%x", &imageType); err != nil {
			if _, err := fmt.Sscanf(args[2], "%d", &imageType); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid image type: %s\n", args[2])
				os.Exit(1)
			}
		}
		if _, err := fmt.Sscanf(args[3], "0x%x", &deviceVersion); err != nil {
			if _, err := fmt.Sscanf(args[3], "%d", &deviceVersion); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid device version: %s\n", args[3])
				os.Exit(1)
			}
		}

		if err := runOTACheck(args[0], manufacturer, imageType, deviceVersion); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var otaServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start OTA server for serving firmware images",
	Long:  "Start an OTA server that responds to firmware update requests from Zigbee devices",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := setupSignalHandler()
		if err := runOTAServer(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var (
	otaImageAddrFlag     uint16
	otaImageEndpointFlag uint8
)

var otaImageCmd = &cobra.Command{
	Use:   "image",
	Short: "Get OTA firmware information from a device",
	Long:  "Query a device for its OTA firmware information including version, manufacturer, and image type",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := setupSignalHandler()
		if err := runOTAImage(ctx, otaImageAddrFlag, otaImageEndpointFlag); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	// OTA list flags
	otaListCmd.Flags().StringVarP(&otaDirFlag, "dir", "d", ".", "Directory containing OTA files")

	// OTA server flags
	otaServerCmd.Flags().StringVarP(&otaDirFlag, "dir", "d", ".", "Directory containing OTA files")
	otaServerCmd.Flags().Uint8VarP(&otaMaxSizeFlag, "max-size", "m", 64, "Maximum block size (16-64)")
	otaServerCmd.Flags().StringVarP(&otaImageFile, "image", "i", "", "Specific image file to trigger update")

	// OTA server needs connection flags
	AddConnectionFlags(otaServerCmd)

	// OTA image needs connection flags
	AddConnectionFlags(otaImageCmd)
	otaImageCmd.Flags().Uint16VarP(&otaImageAddrFlag, "addr", "a", 0, "Device network address")
	otaImageCmd.Flags().Uint8VarP(&otaImageEndpointFlag, "endpoint", "e", 0, "Device endpoint (OTA cluster)")

	// Add subcommands to ota
	otaCmd.AddCommand(otaListCmd)
	otaCmd.AddCommand(otaInfoCmd)
	otaCmd.AddCommand(otaCheckCmd)
	otaCmd.AddCommand(otaServerCmd)
	otaCmd.AddCommand(otaImageCmd)

	// Add ota command to root
	rootCmd.AddCommand(otaCmd)
}

func runOTAList(dir string) error {
	server, err := ota.NewServer(ota.ServerConfig{
		ImagesDir: dir,
	})
	if err != nil {
		return fmt.Errorf("failed to load OTA images: %w", err)
	}

	images := server.ListImages()
	if len(images) == 0 {
		fmt.Printf("No OTA images found in %s\n", dir)
		return nil
	}

	fmt.Printf("OTA images in %s:\n", dir)
	for _, img := range images {
		fmt.Printf("  %s - %s\n", img.HeaderString, img.GetVersionString())
		fmt.Printf("    Manufacturer: 0x%04X, Image Type: 0x%04X\n", img.ManufacturerCode, img.ImageType)
		fmt.Printf("    Size: %d bytes, Stack Version: 0x%04X\n", img.ImageSize, img.StackVersion)
		if img.HeaderFieldControl&ota.FCHardwareVersions != 0 {
			fmt.Printf("    Hardware Versions: %d - %d\n", img.MinHardwareVersion, img.MaxHardwareVersion)
		}
		fmt.Println()
	}

	return nil
}

func runOTAInfo(path string) error {
	img, err := ota.ParseFile(path)
	if err != nil {
		return fmt.Errorf("failed to parse OTA image: %w", err)
	}

	fmt.Printf("OTA Image Information: %s\n", path)
	fmt.Printf("  Version:       %s (0x%08X)\n", img.GetVersionString(), img.FileVersion)
	fmt.Printf("  Manufacturer:  0x%04X\n", img.ManufacturerCode)
	fmt.Printf("  Image Type:    0x%04X\n", img.ImageType)
	fmt.Printf("  Size:          %d bytes\n", img.ImageSize)
	fmt.Printf("  Stack Version: 0x%04X\n", img.StackVersion)
	fmt.Printf("  Header String: %s\n", img.HeaderString)
	fmt.Printf("  Header Length: %d bytes\n", img.HeaderLength)
	fmt.Printf("  Field Control: 0x%04X\n", img.HeaderFieldControl)

	if img.HeaderFieldControl&ota.FCHardwareVersions != 0 {
		fmt.Printf("  Min HW Version: %d\n", img.MinHardwareVersion)
		fmt.Printf("  Max HW Version: %d\n", img.MaxHardwareVersion)
	}

	if len(img.SubElements) > 0 {
		fmt.Printf("  Sub-Elements:  %d\n", len(img.SubElements))
		for _, se := range img.SubElements {
			tag := ota.ImageHeaderTag(se.TagID)
			fmt.Printf("    %s (%d bytes)\n", tag, se.Length)
		}
	}

	signature := img.GetECDSASignature()
	if signature != nil {
		fmt.Printf("  ECDSA Signature: %d bytes\n", len(signature))
	}

	cert := img.GetSigningCertificate()
	if cert != nil {
		fmt.Printf("  Signing Certificate: %d bytes\n", len(cert))
	}

	return nil
}

func runOTACheck(path string, manufacturer uint16, imageType uint16, deviceVersion uint32) error {
	img, err := ota.ParseFile(path)
	if err != nil {
		return fmt.Errorf("failed to parse OTA image: %w", err)
	}

	// Check compatibility
	compatible := img.IsCompatible(manufacturer, imageType, deviceVersion)
	fmt.Printf("Device: Manufacturer=0x%04X, ImageType=0x%04X, Version=%d (0x%08X)\n",
		manufacturer, imageType, deviceVersion, deviceVersion)
	fmt.Printf("Image:  %s (0x%08X)\n", img.GetVersionString(), img.FileVersion)
	fmt.Printf("        Manufacturer=0x%04X, ImageType=0x%04X\n", img.ManufacturerCode, img.ImageType)
	fmt.Printf("        Size=%d bytes\n", img.ImageSize)

	// Validate for upgrade
	validateErr := img.Validate(manufacturer, imageType, deviceVersion)

	fmt.Println()
	if compatible {
		fmt.Println("Result: Compatible ✓")
	} else {
		fmt.Println("Result: Not compatible ✗")
		if img.ManufacturerCode != manufacturer {
			fmt.Printf("  - Manufacturer mismatch (expected 0x%04X, got 0x%04X)\n", img.ManufacturerCode, manufacturer)
		}
		if img.ImageType != imageType {
			fmt.Printf("  - Image type mismatch (expected 0x%04X, got 0x%04X)\n", img.ImageType, imageType)
		}
		if img.HeaderFieldControl&ota.FCHardwareVersions != 0 {
			if deviceVersion < uint32(img.MinHardwareVersion) || deviceVersion > uint32(img.MaxHardwareVersion) {
				fmt.Printf("  - Hardware version out of range (valid: %d-%d)\n", img.MinHardwareVersion, img.MaxHardwareVersion)
			}
		}
	}

	//nolint:gocritic // ifElseChain: conditions use different variables, switch not applicable.
	if validateErr == nil {
		fmt.Println("Upgrade: Valid ✓ (higher version available)")
	} else if compatible {
		fmt.Printf("Upgrade: Valid but %s\n", validateErr)
	} else {
		fmt.Printf("Upgrade: Invalid ✗ - %v\n", validateErr)
	}

	return nil
}

func runOTAServer(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create adapter
	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Get coordinator IEEE address for server address
	networkInfo, err := a.GetNetworkInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network info: %w", err)
	}

	// Create OTA server
	server, err := ota.NewServer(ota.ServerConfig{
		ImagesDir:    otaDirFlag,
		MaxBlockSize: otaMaxSizeFlag,
		ServerAddr:   networkInfo.IEEEAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to create OTA server: %w", err)
	}

	// Set up progress callback
	server.OnProgress = func(update ota.ProgressUpdate) {
		fmt.Printf("[%s] OTA progress: %.1f%% - %d/%d bytes\n",
			update.Device, update.Percentage, update.Offset, update.TotalSize)
	}

	server.OnComplete = func(device string, fileVersion uint32) {
		fmt.Printf("[%s] OTA update complete! Version: 0x%08X\n", device, fileVersion)
	}

	server.OnError = func(device string, err error) {
		fmt.Printf("[%s] OTA update failed: %v\n", device, err)
	}

	fmt.Printf("OTA Server started\n")
	fmt.Printf("  Images directory: %s\n", otaDirFlag)
	fmt.Printf("  Max block size: %d bytes\n", otaMaxSizeFlag)
	fmt.Printf("  Server address: %s\n", znp.FormatIEEEAddr(networkInfo.IEEEAddr))
	fmt.Printf("  Loaded %d firmware images\n", len(server.ListImages()))
	fmt.Println("\nPress Ctrl+C to stop...")

	// Note: For a production OTA server, you'd need to:
	// 1. Register the OTA cluster callback with the adapter
	// 2. Handle QueryImage, ImageBlock, and UpgradeEnd commands
	// 3. Server OTA events until context is canceled

	// For now, just wait for context cancellation
	<-ctx.Done()

	fmt.Println("\nOTA Server stopped")
	return nil
}

func runOTAImage(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Create adapter with timeout
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

	// Get device OTA info
	info, err := a.GetOTAInfo(ctx, nwkAddr, endpoint)
	if err != nil {
		return fmt.Errorf("failed to get OTA info: %v", err)
	}

	fmt.Printf("OTA Information for 0x%04X (endpoint %d)\n", nwkAddr, endpoint)
	fmt.Printf("  Current Version: 0x%08X (%s)\n", info.CurrentFileVersion, formatVersion(info.CurrentFileVersion))
	fmt.Printf("  Manufacturer:    0x%04X\n", info.ManufacturerID)
	fmt.Printf("  Image Type:      0x%04X\n", info.ImageTypeID)
	fmt.Printf("  File Offset:     %d bytes\n", info.FileOffset)

	// Get upgrade status
	status, err := a.GetOTAUpgradeStatus(ctx, nwkAddr, endpoint)
	if err != nil {
		fmt.Printf("  Upgrade Status: Unable to retrieve (%v)\n", err)
	} else {
		fmt.Printf("  Upgrade Status:  0x%02X\n", status)
	}

	// Note: To trigger an OTA update, you'd need to:
	// 1. Query the OTA cluster for the QueryImage command
	// 2. Have an OTA server running
	// 3. Send image update notifications

	return nil
}

// formatVersion converts a 32-bit version to a human-readable string
func formatVersion(version uint32) string {
	major := (version >> 24) & 0xFF
	minor := (version >> 16) & 0xFF
	patch := (version >> 8) & 0xFF
	build := version & 0xFF
	if patch == 0xFF && build == 0xFF {
		patchBuild := version & 0xFFFF
		return fmt.Sprintf("%d.%d.%d", major, minor, patchBuild)
	}
	if build != 0 {
		return fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

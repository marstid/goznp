// Package ota implements Over-The-Air (OTA) firmware update functionality
// for Zigbee devices using the Zigbee OTA Upgrade Cluster (0x0019).
//
// # Overview
//
// OTA upgrades allow Zigbee devices to receive firmware updates over the
// Zigbee network without requiring physical access or a USB connection.
// This package provides both OTA image parsing and an OTA server for serving
// firmware to devices.
//
// # OTA File Format
//
// Zigbee OTA upgrade files follow the Zigbee specification (07-5123-06).
// The file contains:
//
//	[Upgrade File ID] [Header] [Sub-elements] [Image Data]
//
// The Upgrade File ID is a magic number (0x0BEEF11E) that identifies valid
// OTA files. The header contains metadata about the firmware including:
//   - Manufacturer code and image type
//   - Firmware version
//   - Zigbee stack version compatibility
//   - Hardware version ranges
//
// Sub-elements contain optional information like:
//   - Security credentials (ECDH, ECDSA signatures)
//   - Image signing certificates
//   - Manufacturer-specific data
//
// # Basic Usage - Loading OTA Images
//
//	// Load an OTA file
//	img, err := ota.ParseFile("/path/to/firmware.ota")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check image metadata
//	fmt.Printf("Version: %s\n", img.GetVersionString())
//	fmt.Printf("Manufacturer: 0x%04X\n", img.ManufacturerCode)
//	fmt.Printf("Image Type: 0x%04X\n", img.ImageType)
//	fmt.Printf("Size: %d bytes\n", img.ImageSize)
//
//	// Validate compatibility with a device
//	err = img.Validate(deviceManufacturer, deviceImageType, deviceVersion)
//
// # Basic Usage - OTA Server
//
//	// Create an OTA server with images in a directory
//	server, err := ota.NewServer(ota.ServerConfig{
//	    ImagesDir: "/path/to/ota/files",
//	    MaxBlockSize: 64,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Set the server address to coordinator's IEEE address
//	server.SetServerAddr(coordinatorIEEEAddr)
//
//	// Handle Query Image command from device
//	resp, err := server.QueryImage(ctx, ota.QueryImageCommand{
//	    ManufacturerCode: 0x115F, // Tuya
//	    ImageType:        0x2200,
//	    FileVersion:      0x1234,
//	}, "00:11:22:33:44:55:66:77")
//
// // Handling OTA Requests from Devices
//
//	When a device is performing an OTA update, it will send these commands:
//
//	1. Query Image - Device asks if an update is available
//	2. Image Block - Device requests chunks of firmware data
//	3. Upgrade End - Device signals completion (success/failure)
//
//	These are typically handled by the adapter layer which calls into
//	the OTA server. See the goznp adapter package for integration.
//
// # Progress Monitoring
//
//	// Set up progress callback
//	server.OnProgress = func(update ota.ProgressUpdate) {
//	    fmt.Printf("[%s] %.1f%% complete - %d/%d bytes\n",
//	        update.Device, update.Percentage,
//	        update.Offset, update.TotalSize)
//	}
//
//	// Or monitor via channel
//	updates, err := server.MonitorProgress(ctx, deviceIEEEAddr)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for update := range updates {
//	    fmt.Printf("Progress: %.1f%%\n", update.Percentage)
//	}
//
// # Image Management
//
//	// List all available images
//	images := server.ListImages()
//	for _, img := range images {
//	    fmt.Printf("%s - %s (%d bytes)\n",
//	        img.GetVersionString(), img.HeaderString, img.ImageSize)
//	}
//
//	// Add a custom image
//	img, _ := ota.ParseBytes(otaData)
//	server.AddImage(img)
//
//	// Remove an image
//	server.RemoveImage(manufacturer, imageType, fileVersion)
//
// # Sub-elements
//
//	Sub-elements contain optional metadata about the image:
//
//	// Get ECDSA signature
//	signature := img.GetECDSASignature()
//
//	// Get signing certificate
//	cert := img.GetSigningCertificate()
//
//	// Get any sub-element by tag
//	se := img.GetSubElement(ota.TagSigningCertificate)
//
// # Thread Safety
//
// The OTA Server is safe for concurrent use from multiple goroutines.
// All methods use internal mutex protection.
//
// # Security Considerations
//
// OTA images should be verified for authenticity:
//   - Check the upgrade file ID (0x0BEEF11E)
//   - Validate ECDSA signatures if present
//   - Ensure images come from trusted sources
//   - Use install codes for secure device joining
//   - Verify manufacturer and image type before serving
package ota

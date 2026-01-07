// Package ota implements OTA (Over-The-Air) firmware update functionality
// for Zigbee devices using the Zigbee OTA Upgrade cluster (0x0019).
//
// This package provides:
//   - OTA image file parsing (Zigbee OTA Upgrade file format).
//   - OTA server implementation for serving firmware to devices.
//   - Progress tracking for ongoing updates.
package ota

import (
	"fmt"
)

// FileHeader represents the header of an OTA upgrade file.
// Based on Zigbee specification 07-5123-06 (Zigbee Over-The-Air Upgrade).
type FileHeader struct {
	// OTA header fields.
	UpgradeFileID      uint32 // Must be 0x0BEEF11E for upgrade files.
	HeaderVersion      uint16 // Must be 0x0100 for version 1.0.
	HeaderLength       uint16 // Length of this header in bytes.
	HeaderFieldControl uint16 // Bitmask indicating which fields are present.
	ManufacturerCode   uint16 // Manufacturer-specific code.
	ImageType          uint16 // Manufacturer-defined image type identifier.
	FileVersion        uint32 // Version of the firmware image.
	StackVersion       uint16 // Version of the Zigbee stack.
	HeaderString       string // "Image header v" followed by version.
	ImageSize          uint32 // Size of the complete image in bytes.

	// Optional fields (based on HeaderFieldControl).
	SecurityCredentialVers uint8  // Version of security credentials (if BC-01).
	DeviceSpecificFile     bool   // True if this file is device-specific.
	MinHardwareVersion     uint16 // Minimum supported hardware version.
	MaxHardwareVersion     uint16 // Maximum supported hardware version.
	HeaderBitmap           uint32 // Bitmap indicating other optional header fields.

	// More optional fields can be added as needed.
}

// SubElement represents an optional sub-element in the OTA file.
type SubElement struct {
	TagID  uint16 // Identifies the type of sub-element.
	Length uint32 // Length of the data in bytes.
	Data   []byte // Sub-element data.
}

// Image represents a parsed OTA upgrade file.
type Image struct {
	FileHeader
	SubElements []SubElement
	ImageData   []byte // Raw firmware data.
	RawBytes    []byte // Complete file contents.
	Path        string // File path if loaded from disk.
}

// QueryImageCommand represents the OTA cluster Query Image command.
// Sent by the client to request firmware update information.
type QueryImageCommand struct {
	FieldControl     uint8  // Bitmask for optional fields.
	ManufacturerCode uint16 // Manufacturer code.
	ImageType        uint16 // Image type identifier.
	FileVersion      uint32 // Current firmware version.
	HardwareVersion  uint16 // Current hardware version.
}

// QueryImageResponse represents the OTA cluster Query Image response.
// Sent by the server to update availability and file information.
type QueryImageResponse struct {
	Status            uint8   // 0x00=Success, 0x80=Abort, 0x81=Not authorized, etc.
	ManufacturerCode  uint16  // Manufacturer code.
	ImageType         uint16  // Image type identifier.
	FileVersion       uint32  // New firmware file version.
	ImageSize         uint32  // Size of the firmware image in bytes.
	ServerAddress     [8]byte // IEEE address of the OTA server.
	MinimumBlockDelay uint16  // Minimum delay between block requests (ms).
	MaximumBlockDelay uint16  // Maximum delay between block requests (ms).
}

// ImageBlockRequest represents the OTA cluster Image Block request.
// Sent by the client to request a block of firmware data.
type ImageBlockRequest struct {
	FieldControl     uint8  // Bitmask for optional fields.
	ManufacturerCode uint16 // Manufacturer code.
	ImageType        uint16 // Image type identifier.
	FileVersion      uint32 // Firmware version being requested.
	FileOffset       uint32 // Byte offset of requested block.
	MaximumDataSize  uint8  // Maximum data size for this block.
}

// ImageBlockResponse represents the OTA cluster Image Block response.
// Sent by the server containing firmware data.
type ImageBlockResponse struct {
	Status           uint8  // 0x00=Success.
	ManufacturerCode uint16 // Manufacturer code.
	ImageType        uint16 // Image type identifier.
	FileVersion      uint32 // Firmware version.
	FileOffset       uint32 // Byte offset of this block.
	DataSize         uint8  // Length of the following data.
	Data             []byte // Firmware data block.
}

// UpgradeEndRequest represents the OTA cluster Upgrade End request.
// Sent by the client when the upgrade is complete.
type UpgradeEndRequest struct {
	Status           uint8  // 0x00=Success, 0x81=Invalid image, etc.
	ManufacturerCode uint16 // Manufacturer code.
	ImageType        uint16 // Image type identifier.
	FileVersion      uint32 // Firmware version applied.
}

// UpgradeEndResponse represents the OTA cluster Upgrade End response.
// Sent by the server to acknowledge the upgrade completion.
type UpgradeEndResponse struct {
	ManufacturerCode uint16 // Echoed from request.
	ImageType        uint16 // Echoed from request.
	FileVersion      uint32 // Echoed from request.
	Applicability    uint8  // 0x01=Downgraded, 0x02=Same version, 0x03=Upgraded.
}

// OTAStatus represents OTA operation status codes.
//
//nolint:revive // Keep OTAStatus name for clarity despite stuttering
type OTAStatus uint8

const (
	StatusSuccess           OTAStatus = 0x00 // Operation successful.
	StatusInvalidImage      OTAStatus = 0x81 // Invalid image.
	StatusRequiresMoreImage OTAStatus = 0x82 // Requires more images.
	StatusNotAuthorized     OTAStatus = 0x83 // Not authorized.
	StatusAbort             OTAStatus = 0x95 // Abort operation.
	StatusAbortWaitForData  OTAStatus = 0x96 // Abort, wait for data.
	StatusWaitForData       OTAStatus = 0x97 // Wait for more data.
	StatusNoImageAvailable  OTAStatus = 0x98 // No image available.
)

func (s OTAStatus) String() string {
	switch s {
	case StatusSuccess:
		return "Success"
	case StatusInvalidImage:
		return "Invalid Image"
	case StatusRequiresMoreImage:
		return "Requires More Image"
	case StatusNotAuthorized:
		return "Not Authorized"
	case StatusAbort:
		return "Abort"
	case StatusAbortWaitForData:
		return "Abort Wait For Data"
	case StatusWaitForData:
		return "Wait For Data"
	case StatusNoImageAvailable:
		return "No Image Available"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", uint8(s))
	}
}

// UpgradeProgress tracks the progress of an OTA update.
type UpgradeProgress struct {
	IEEEAddress    string              // IEEE address of the device.
	FileVersion    uint32              // File version being downloaded.
	TotalSize      uint32              // Total size of the firmware.
	DownloadedSize uint32              // Bytes downloaded so far.
	Percentage     float64             // Completion percentage (0-100).
	ProgressChan   chan ProgressUpdate // Channel for progress updates.
}

// ProgressUpdate represents a single progress update.
type ProgressUpdate struct {
	Device      string  // IEEE address.
	FileVersion uint32  // File version.
	Offset      uint32  // Current offset.
	TotalSize   uint32  // Total file size.
	Percentage  float64 // Completion percentage.
	Status      string  // Status message.
	Error       error   // Error if any.
}

// ImageHeaderTag identifies optional sub-element types.
type ImageHeaderTag uint16

const (
	TagECDHInfo            ImageHeaderTag = 0x0001 // Elliptic Curve Diffie-Hellman info.
	TagECDSASignature      ImageHeaderTag = 0x0002 // Elliptic Curve Digital Signature Algorithm.
	TagImageLayout         ImageHeaderTag = 0x0003 // Manufacturer-defined layout.
	TagSigningCertificate  ImageHeaderTag = 0x0004 // Certificate for image signing.
	TagWholeImageSignature ImageHeaderTag = 0x0005 // Signature for the entire image.
	TagZigbeePlatformInfo  ImageHeaderTag = 0x0006 // Zigbee platform-specific info.
)

func (t ImageHeaderTag) String() string {
	switch t {
	case TagECDHInfo:
		return "ECDH Info"
	case TagECDSASignature:
		return "ECDSA Signature"
	case TagImageLayout:
		return "Image Layout"
	case TagSigningCertificate:
		return "Signing Certificate"
	case TagWholeImageSignature:
		return "Whole Image Signature"
	case TagZigbeePlatformInfo:
		return "Zigbee Platform Info"
	default:
		return fmt.Sprintf("Unknown(0x%04X)", uint16(t))
	}
}

// FirmwareMatch represents matching firmware for a device.
type FirmwareMatch struct {
	Image        *Image // Parsed OTA image.
	Manufacturer uint16 // Manufacturer code.
	ImageType    uint16 // Image type identifier.
	FileVersion  uint32 // Firmware version.
	Compatible   bool   // Whether this firmware is compatible.
}

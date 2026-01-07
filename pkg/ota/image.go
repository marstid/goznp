package ota

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	// UpgradeFileID marks a valid OTA upgrade file.
	UpgradeFileID = 0x0BEEF11E

	// HeaderVersion supported by this implementation.
	HeaderVersion = 0x0100

	// Field control bit positions.
	FCSecurityCredentialVers uint16 = 0x0001
	FCDeviceSpecificFile     uint16 = 0x0002
	FCHardwareVersions       uint16 = 0x0004
	FCMinApplicable          uint16 = 0x0040
	FCMinAppHardwareVers     uint16 = 0x0080
)

// ParseFile reads and parses an OTA upgrade file from disk.
func ParseFile(path string) (*Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ota: failed to read file: %w", err)
	}

	return ParseBytes(data)
}

// ParseBytes parses an OTA upgrade file from byte data.
func ParseBytes(data []byte) (*Image, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("ota: file too small (%d bytes)", len(data))
	}

	buf := bytes.NewReader(data)
	img := &Image{
		RawBytes: data,
	}

	// Read and validate upgrade file ID.
	if err := binary.Read(buf, binary.LittleEndian, &img.UpgradeFileID); err != nil {
		return nil, fmt.Errorf("ota: failed to read upgrade file ID: %w", err)
	}
	if img.UpgradeFileID != UpgradeFileID {
		return nil, fmt.Errorf("ota: invalid upgrade file ID, expected 0x%08X, got 0x%08X",
			UpgradeFileID, img.UpgradeFileID)
	}

	// Read header version.
	if err := binary.Read(buf, binary.LittleEndian, &img.HeaderVersion); err != nil {
		return nil, fmt.Errorf("ota: failed to read header version: %w", err)
	}
	if img.HeaderVersion != HeaderVersion {
		return nil, fmt.Errorf("ota: unsupported header version 0x%04X, expected 0x%04X",
			img.HeaderVersion, HeaderVersion)
	}

	// Read header length.
	if err := binary.Read(buf, binary.LittleEndian, &img.HeaderLength); err != nil {
		return nil, fmt.Errorf("ota: failed to read header length: %w", err)
	}
	if img.HeaderLength < 32 {
		return nil, fmt.Errorf("ota: invalid header length %d (minimum 32)", img.HeaderLength)
	}

	// Read field control.
	if err := binary.Read(buf, binary.LittleEndian, &img.HeaderFieldControl); err != nil {
		return nil, fmt.Errorf("ota: failed to read field control: %w", err)
	}

	// Read manufacturer code.
	if err := binary.Read(buf, binary.LittleEndian, &img.ManufacturerCode); err != nil {
		return nil, fmt.Errorf("ota: failed to read manufacturer code: %w", err)
	}

	// Read image type.
	if err := binary.Read(buf, binary.LittleEndian, &img.ImageType); err != nil {
		return nil, fmt.Errorf("ota: failed to read image type: %w", err)
	}

	// Read file version.
	if err := binary.Read(buf, binary.LittleEndian, &img.FileVersion); err != nil {
		return nil, fmt.Errorf("ota: failed to read file version: %w", err)
	}

	// Read stack version.
	if err := binary.Read(buf, binary.LittleEndian, &img.StackVersion); err != nil {
		return nil, fmt.Errorf("ota: failed to read stack version: %w", err)
	}

	// Read header string (up to 32 bytes).
	headerStringBytes := make([]byte, 32)
	if _, err := io.ReadFull(buf, headerStringBytes); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("ota: failed to read header string: %w", err)
		}
	}
	img.HeaderString = nullTerminatedString(headerStringBytes)

	// Read image size.
	if err := binary.Read(buf, binary.LittleEndian, &img.ImageSize); err != nil {
		return nil, fmt.Errorf("ota: failed to read image size: %w", err)
	}

	// Read optional fields based on field control.
	if img.HeaderFieldControl&FCSecurityCredentialVers != 0 {
		if err := binary.Read(buf, binary.LittleEndian, &img.SecurityCredentialVers); err != nil {
			return nil, fmt.Errorf("ota: failed to read security credential version: %w", err)
		}
	}

	if img.HeaderFieldControl&FCDeviceSpecificFile != 0 {
		var dsf uint16
		if err := binary.Read(buf, binary.LittleEndian, &dsf); err != nil {
			return nil, fmt.Errorf("ota: failed to read device specific file flag: %w", err)
		}
		img.DeviceSpecificFile = dsf != 0
	}

	if img.HeaderFieldControl&FCHardwareVersions != 0 {
		if err := binary.Read(buf, binary.LittleEndian, &img.MinHardwareVersion); err != nil {
			return nil, fmt.Errorf("ota: failed to read min hardware version: %w", err)
		}
		if err := binary.Read(buf, binary.LittleEndian, &img.MaxHardwareVersion); err != nil {
			return nil, fmt.Errorf("ota: failed to read max hardware version: %w", err)
		}
	}

	// Read remaining header fields if present.
	position := len(data) - buf.Len()
	remainingHeader := int(img.HeaderLength) - position
	if remainingHeader > 0 {
		remainingBytes := make([]byte, remainingHeader)
		if _, err := io.ReadFull(buf, remainingBytes); err != nil {
			return nil, fmt.Errorf("ota: failed to read remaining header fields: %w", err)
		}
		// Parse header bitmap if present.
		if remainingHeader >= 4 {
			img.HeaderBitmap = binary.LittleEndian.Uint32(remainingBytes)
		}
	}

	// Read sub-elements until we hit the image data.
	for {
		if buf.Len() < 6 { // Need at least 2 bytes (tag) + 4 bytes (length).
			break // Not enough data for a sub-element header.
		}

		// Peek at next tag.
		tagID := binary.LittleEndian.Uint16(data[len(data)-buf.Len():])
		length := binary.LittleEndian.Uint32(data[len(data)-buf.Len()+2:])

		// Check if this is end of header (tag ID 0xFFFF).
		if tagID == 0xFFFF {
			// Skip terminator (2-byte tag + 4-byte length = 6 bytes)
			//nolint:errcheck // Seek error intentionally ignored
			_, _ = buf.Seek(6, io.SeekCurrent)
			break
		}

		// Check if we have enough data for this sub-element.
		if int(length) > buf.Len()-6 {
			break
		}

		// Read sub-element.
		se := SubElement{
			TagID:  tagID,
			Length: length,
			Data:   make([]byte, length),
		}
		if _, err := io.ReadFull(buf, se.Data); err != nil {
			return nil, fmt.Errorf("ota: failed to read sub-element: %w", err)
		}
		img.SubElements = append(img.SubElements, se)
	}

	// Remaining data is the image data.
	img.ImageData = make([]byte, buf.Len())
	copy(img.ImageData, data[len(data)-buf.Len():])

	return img, nil
}

// Validate checks if the OTA image is valid for the given device.
func (img *Image) Validate(deviceManufacturer uint16, deviceImageType uint16, deviceVersion uint32) error {
	// Check manufacturer.
	if img.ManufacturerCode != deviceManufacturer {
		return fmt.Errorf("ota: manufacturer mismatch, image 0x%04X, device 0x%04X",
			img.ManufacturerCode, deviceManufacturer)
	}

	// Check image type.
	if img.ImageType != deviceImageType {
		return fmt.Errorf("ota: image type mismatch, image 0x%04X, device 0x%04X",
			img.ImageType, deviceImageType)
	}

	// Check that new version is higher than device version.
	if img.FileVersion <= deviceVersion {
		return fmt.Errorf("ota: file version %d is not higher than device version %d",
			img.FileVersion, deviceVersion)
	}

	return nil
}

// IsCompatible checks if this image is compatible with the given device.
// Unlike Validate, this returns a boolean and doesn't require version to be higher.
func (img *Image) IsCompatible(deviceManufacturer uint16, deviceImageType uint16, deviceVersion uint32) bool {
	// Check manufacturer.
	if img.ManufacturerCode != deviceManufacturer {
		return false
	}

	// Check image type.
	if img.ImageType != deviceImageType {
		return false
	}

	// Check hardware version range if specified.
	if img.HeaderFieldControl&FCHardwareVersions != 0 {
		if deviceVersion < uint32(img.MinHardwareVersion) ||
			deviceVersion > uint32(img.MaxHardwareVersion) {
			return false
		}
	}

	return true
}

// GetVersionString returns a human-readable version string.
// Zigbee OTA version format: major (byte 3), minor (byte 2), patch (byte 1), build (byte 0).
func (img *Image) GetVersionString() string {
	major := (img.FileVersion >> 24) & 0xFF
	minor := (img.FileVersion >> 16) & 0xFF
	patch := (img.FileVersion >> 8) & 0xFF
	build := img.FileVersion & 0xFF
	patchBuild := (img.FileVersion) & 0xFFFF
	if patch == 0xFF && build == 0xFF {
		// Max value: combine patch and build into 16-bit value.
		return fmt.Sprintf("%d.%d.%d", major, minor, patchBuild)
	}
	if build != 0 {
		return fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

// GetSubElement returns a sub-element by its tag ID, or nil if not found.
func (img *Image) GetSubElement(tagID ImageHeaderTag) *SubElement {
	for _, se := range img.SubElements {
		if se.TagID == uint16(tagID) {
			return &se
		}
	}
	return nil
}

// GetECDSASignature returns the ECDSA signature sub-element if present.
func (img *Image) GetECDSASignature() []byte {
	se := img.GetSubElement(TagECDSASignature)
	if se != nil {
		return se.Data
	}
	return nil
}

// GetSigningCertificate returns the signing certificate sub-element if present.
func (img *Image) GetSigningCertificate() []byte {
	se := img.GetSubElement(TagSigningCertificate)
	if se != nil {
		return se.Data
	}
	return nil
}

// BlockSize returns the recommended block size for this image.
// Most devices support block sizes between 16 and 64 bytes.
func (img *Image) BlockSize() uint8 {
	return 64 // Default block size.
}

// NextBlockOffset calculates the next block offset and size for a given offset.
func (img *Image) NextBlockOffset(offset uint32, maxSize uint8) (nextOffset uint32, dataSize uint8, finished bool) {
	if offset >= img.ImageSize {
		return offset, 0, true
	}

	remaining := img.ImageSize - offset
	blockSize := img.BlockSize()
	if blockSize > maxSize {
		blockSize = maxSize
	}

	// Cast to uint32 for comparison to avoid uint8 overflow.
	if remaining < uint32(blockSize) {
		dataSize = uint8(remaining)
	} else {
		dataSize = blockSize
	}
	nextOffset = offset
	finished = offset+uint32(dataSize) >= img.ImageSize
	return nextOffset, dataSize, finished
}

// nullTerminatedString converts a byte slice to a string, stopping at the first null byte.
func nullTerminatedString(b []byte) string {
	null := bytes.IndexByte(b, 0)
	if null == -1 {
		return string(b)
	}
	return string(b[:null])
}

// minUint8 returns the minimum of two uint8 values.
//
//nolint:unused // Reserved for future use
func minUint8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

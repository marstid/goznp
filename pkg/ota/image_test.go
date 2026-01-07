package ota

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestParseBytes tests parsing OTA image byte data.
func TestParseBytes(t *testing.T) {
	// Create a minimal valid OTA image
	buf := new(bytes.Buffer)

	// Upgrade file ID
	binary.Write(buf, binary.LittleEndian, uint32(UpgradeFileID))

	// Header version
	binary.Write(buf, binary.LittleEndian, uint16(HeaderVersion))

	// Header length (minimal header = 32 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(32))

	// Field control (no optional fields)
	binary.Write(buf, binary.LittleEndian, uint16(0))

	// Manufacturer code
	binary.Write(buf, binary.LittleEndian, uint16(0x115F))

	// Image type
	binary.Write(buf, binary.LittleEndian, uint16(0x2200))

	// File version
	binary.Write(buf, binary.LittleEndian, uint32(0x01020304))

	// Stack version
	binary.Write(buf, binary.LittleEndian, uint16(0x0200))

	// Header string
	headerString := []byte("Test OTA Header\x00")
	headerString = append(headerString, make([]byte, 32-len(headerString))...)
	buf.Write(headerString)

	// Image size
	imageData := []byte("firmware data here")
	binary.Write(buf, binary.LittleEndian, uint32(len(imageData)))

	// Sub-element terminator (0xFFFF) to indicate end of header
	// Terminator has 2-byte tag (0xFFFF) + 4-byte length (0x00000000)
	binary.Write(buf, binary.LittleEndian, uint16(0xFFFF))
	binary.Write(buf, binary.LittleEndian, uint32(0))

	// Image data
	buf.Write(imageData)

	// Parse the image
	img, err := ParseBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}

	// Validate parsed data
	if img.UpgradeFileID != UpgradeFileID {
		t.Errorf("UpgradeFileID = 0x%08X, want 0x%08X", img.UpgradeFileID, UpgradeFileID)
	}
	if img.HeaderVersion != HeaderVersion {
		t.Errorf("HeaderVersion = 0x%04X, want 0x%04X", img.HeaderVersion, HeaderVersion)
	}
	if img.ManufacturerCode != 0x115F {
		t.Errorf("ManufacturerCode = 0x%04X, want 0x115F", img.ManufacturerCode)
	}
	if img.ImageType != 0x2200 {
		t.Errorf("ImageType = 0x%04X, want 0x2200", img.ImageType)
	}
	if img.FileVersion != 0x01020304 {
		t.Errorf("FileVersion = 0x%08X, want 0x01020304", img.FileVersion)
	}
	if img.HeaderString != "Test OTA Header" {
		t.Errorf("HeaderString = %q, want %q", img.HeaderString, "Test OTA Header")
	}
	if img.ImageSize != uint32(len(imageData)) {
		t.Errorf("ImageSize = %d, want %d", img.ImageSize, len(imageData))
	}
	if !bytes.Equal(img.ImageData, imageData) {
		t.Errorf("ImageData mismatch: got %q (%d bytes), want %q (%d bytes)", img.ImageData, len(img.ImageData), imageData, len(imageData))
	}
}

// TestParseBytesInvalid tests parsing invalid OTA images.
func TestParseBytesInvalid(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "too small",
			data:    make([]byte, 31),
			wantErr: "too small",
		},
		{
			name: "invalid file ID",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, uint32(0xDEADBEEF)) // Invalid ID
				return buf.Bytes()
			}(),
			wantErr: "invalid upgrade file ID",
		},
		{
			name: "invalid header version",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, uint32(UpgradeFileID))
				binary.Write(buf, binary.LittleEndian, uint16(0x0201)) // Wrong version
				return buf.Bytes()
			}(),
			wantErr: "unsupported header version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes(tt.data)
			if err == nil {
				t.Fatal("ParseBytes() expected error, got nil")
			}
			if tt.wantErr != "" && err == nil {
				t.Errorf("Error should contain %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestValidate tests OTA image validation.
func TestValidate(t *testing.T) {
	img := &Image{
		FileHeader: FileHeader{
			ManufacturerCode: 0x115F,
			ImageType:        0x2200,
			FileVersion:      0x01020305,
		},
	}

	tests := []struct {
		name          string
		manufacturer  uint16
		imageType     uint16
		deviceVersion uint32
		wantErr       bool
	}{
		{
			name:          "valid match",
			manufacturer:  0x115F,
			imageType:     0x2200,
			deviceVersion: 0x01020304,
			wantErr:       false,
		},
		{
			name:          "manufacturer mismatch",
			manufacturer:  0x1234,
			imageType:     0x2200,
			deviceVersion: 0x01020304,
			wantErr:       true,
		},
		{
			name:          "image type mismatch",
			manufacturer:  0x115F,
			imageType:     0x1234,
			deviceVersion: 0x01020304,
			wantErr:       true,
		},
		{
			name:          "version not higher",
			manufacturer:  0x115F,
			imageType:     0x2200,
			deviceVersion: 0x01020305,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := img.Validate(tt.manufacturer, tt.imageType, tt.deviceVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsCompatible tests OTA image compatibility checking.
func TestIsCompatible(t *testing.T) {
	img := &Image{
		FileHeader: FileHeader{
			ManufacturerCode:   0x115F,
			ImageType:          0x2200,
			FileVersion:        0x01020305,
			HeaderFieldControl: FCHardwareVersions,
			MinHardwareVersion: 10,
			MaxHardwareVersion: 20,
		},
	}

	tests := []struct {
		name          string
		manufacturer  uint16
		imageType     uint16
		deviceVersion uint32
		want          bool
	}{
		{
			name:          "compatible",
			manufacturer:  0x115F,
			imageType:     0x2200,
			deviceVersion: 15,
			want:          true,
		},
		{
			name:          "manufacturer mismatch",
			manufacturer:  0x1234,
			imageType:     0x2200,
			deviceVersion: 15,
			want:          false,
		},
		{
			name:          "hardware version too low",
			manufacturer:  0x115F,
			imageType:     0x2200,
			deviceVersion: 5,
			want:          false,
		},
		{
			name:          "hardware version too high",
			manufacturer:  0x115F,
			imageType:     0x2200,
			deviceVersion: 25,
			want:          false,
		},
		{
			name:          "hardware version at max",
			manufacturer:  0x115F,
			imageType:     0x2200,
			deviceVersion: 20,
			want:          true,
		},
	}

	// Test with hardware version check disabled
	t.Run("no field control", func(t *testing.T) {
		img.HeaderFieldControl = 0
		if !img.IsCompatible(0x115F, 0x2200, 255) {
			t.Error("IsCompatible() should return true when no field control is set")
		}
	})

	// Test with hardware version field
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-enable hardware version check
			img.HeaderFieldControl = FCHardwareVersions
			if got := img.IsCompatible(tt.manufacturer, tt.imageType, tt.deviceVersion); got != tt.want {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetVersionString tests version string formatting.
func TestGetVersionString(t *testing.T) {
	tests := []struct {
		name     string
		version  uint32
		expected string
	}{
		{
			name:     "typical version",
			version:  0x01020304,
			expected: "1.2.3.4",
		},
		{
			name:     "major only",
			version:  0x02000000,
			expected: "2.0.0",
		},
		{
			name:     "zero version",
			version:  0x00000000,
			expected: "0.0.0",
		},
		{
			name:     "max values",
			version:  0xFFFFFFFF,
			expected: "255.255.65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := &Image{
				FileHeader: FileHeader{
					FileVersion: tt.version,
				},
			}
			if got := img.GetVersionString(); got != tt.expected {
				t.Errorf("GetVersionString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestGetSubElement tests sub-element retrieval.
func TestGetSubElement(t *testing.T) {
	tag1 := uint16(TagECDSASignature)
	data1 := []byte("signature data")

	tag2 := uint16(TagSigningCertificate)
	data2 := []byte("certificate data")

	img := &Image{
		SubElements: []SubElement{
			{TagID: tag1, Length: uint32(len(data1)), Data: data1},
			{TagID: tag2, Length: uint32(len(data2)), Data: data2},
		},
	}

	tests := []struct {
		name string
		tag  ImageHeaderTag
		want []byte
	}{
		{
			name: "ECDSA signature",
			tag:  TagECDSASignature,
			want: data1,
		},
		{
			name: "signing certificate",
			tag:  TagSigningCertificate,
			want: data2,
		},
		{
			name: "not found",
			tag:  TagECDHInfo,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := img.GetSubElement(tt.tag)
			if (se == nil) != (tt.want == nil) {
				t.Fatalf("GetSubElement() = %v, want nil: %v", se, tt.want == nil)
			}
			if se != nil && !bytes.Equal(se.Data, tt.want) {
				t.Errorf("GetSubElement() data = %v, want %v", se.Data, tt.want)
			}
		})
	}
}

// TestNextBlockOffset tests block offset calculation.
func TestNextBlockOffset(t *testing.T) {
	img := &Image{
		FileHeader: FileHeader{
			ImageSize: 256,
		},
	}

	tests := []struct {
		name          string
		offset        uint32
		maxSize       uint8
		wantOffset    uint32
		wantDataSize  uint8
		wantFinished  bool
	}{
		{
			name:          "first block",
			offset:        0,
			maxSize:       64,
			wantOffset:    0,
			wantDataSize:  64,
			wantFinished:  false,
		},
		{
			name:          "middle block",
			offset:        128,
			maxSize:       64,
			wantOffset:    128,
			wantDataSize:  64,
			wantFinished:  false,
		},
		{
			name:          "last partial block",
			offset:        200,
			maxSize:       64,
			wantOffset:    200,
			wantDataSize:  56,
			wantFinished:  true,
		},
		{
			name:          "exactly at end",
			offset:        256,
			maxSize:       64,
			wantOffset:    256,
			wantDataSize:  0,
			wantFinished:  true,
		},
		{
			name:          "beyond end",
			offset:        300,
			maxSize:       64,
			wantOffset:    300,
			wantDataSize:  0,
			wantFinished:  true,
		},
		{
			name:          "small max size",
			offset:        0,
			maxSize:       16,
			wantOffset:    0,
			wantDataSize:  16,
			wantFinished:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, dataSize, finished := img.NextBlockOffset(tt.offset, tt.maxSize)
			if offset != tt.wantOffset {
				t.Errorf("NextBlockOffset() offset = %d, want %d", offset, tt.wantOffset)
			}
			if dataSize != tt.wantDataSize {
				t.Errorf("NextBlockOffset() dataSize = %d, want %d", dataSize, tt.wantDataSize)
			}
			if finished != tt.wantFinished {
				t.Errorf("NextBlockOffset() finished = %v, want %v", finished, tt.wantFinished)
			}
		})
	}
}

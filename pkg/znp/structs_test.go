package znp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestParseNIB tests parsing of Network Information Base structure.
func TestParseNIB(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *NIB
		wantErr bool
	}{
		{
			name: "valid NIB with typical values",
			data: func() []byte {
				data := make([]byte, 92)
				// SecurityLevel at offset 20
				data[20] = 0x05
				// ExtendedPanID at offset 21-28 (8 bytes)
				copy(data[21:29], []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
				// ChannelList at offset 29-32 (uint32 little-endian)
				binary.LittleEndian.PutUint32(data[29:33], 0x07FFF800) // Channels 11-26
				// PanID at offset 38-39 (uint16 little-endian)
				binary.LittleEndian.PutUint16(data[38:40], 0x1A62)
				// LogicalChannel at offset 42
				data[42] = 11
				// UpdateID at offset 91
				data[91] = 0x00
				return data
			}(),
			want: &NIB{
				SecurityLevel:  0x05,
				ExtendedPanID:  [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				ChannelList:    0x07FFF800,
				PanID:          0x1A62,
				LogicalChannel: 11,
				UpdateID:       0x00,
			},
			wantErr: false,
		},
		{
			name: "NIB with all zeros",
			data: func() []byte {
				return make([]byte, 92)
			}(),
			want: &NIB{
				SecurityLevel:  0x00,
				ExtendedPanID:  [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				ChannelList:    0x00000000,
				PanID:          0x0000,
				LogicalChannel: 0x00,
				UpdateID:       0x00,
			},
			wantErr: false,
		},
		{
			name: "NIB with max values",
			data: func() []byte {
				data := make([]byte, 92)
				data[20] = 0xFF
				copy(data[21:29], []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
				binary.LittleEndian.PutUint32(data[29:33], 0xFFFFFFFF)
				binary.LittleEndian.PutUint16(data[38:40], 0xFFFF)
				data[42] = 0xFF
				data[91] = 0xFF
				return data
			}(),
			want: &NIB{
				SecurityLevel:  0xFF,
				ExtendedPanID:  [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
				ChannelList:    0xFFFFFFFF,
				PanID:          0xFFFF,
				LogicalChannel: 0xFF,
				UpdateID:       0xFF,
			},
			wantErr: false,
		},
		{
			name:    "NIB data too short",
			data:    make([]byte, 50),
			wantErr: true,
		},
		{
			name:    "NIB data minimum - 1 byte",
			data:    make([]byte, 91),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNIB(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNIB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.SecurityLevel != tt.want.SecurityLevel {
				t.Errorf("SecurityLevel = 0x%02X, want 0x%02X", got.SecurityLevel, tt.want.SecurityLevel)
			}
			if got.ExtendedPanID != tt.want.ExtendedPanID {
				t.Errorf("ExtendedPanID = %v, want %v", got.ExtendedPanID, tt.want.ExtendedPanID)
			}
			if got.ChannelList != tt.want.ChannelList {
				t.Errorf("ChannelList = 0x%08X, want 0x%08X", got.ChannelList, tt.want.ChannelList)
			}
			if got.PanID != tt.want.PanID {
				t.Errorf("PanID = 0x%04X, want 0x%04X", got.PanID, tt.want.PanID)
			}
			if got.LogicalChannel != tt.want.LogicalChannel {
				t.Errorf("LogicalChannel = %d, want %d", got.LogicalChannel, tt.want.LogicalChannel)
			}
			if got.UpdateID != tt.want.UpdateID {
				t.Errorf("UpdateID = 0x%02X, want 0x%02X", got.UpdateID, tt.want.UpdateID)
			}
		})
	}
}

// TestParseNwkKeyDescriptor tests parsing of network key descriptor.
func TestParseNwkKeyDescriptor(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *NwkKeyDescriptor
		wantErr bool
	}{
		{
			name: "valid key descriptor",
			data: append(
				[]byte{0x00}, // KeySeqNum
				[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
					0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}..., // 16-byte key
			),
			want: &NwkKeyDescriptor{
				KeySeqNum: 0x00,
				Key: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
					0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			},
			wantErr: false,
		},
		{
			name: "key with all zeros",
			data: make([]byte, 17),
			want: &NwkKeyDescriptor{
				KeySeqNum: 0x00,
				Key:       [16]byte{},
			},
			wantErr: false,
		},
		{
			name: "key with all FFs",
			data: append(
				[]byte{0xFF},
				[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
					0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}...,
			),
			want: &NwkKeyDescriptor{
				KeySeqNum: 0xFF,
				Key: [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
					0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			wantErr: false,
		},
		{
			name:    "data too short",
			data:    make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNwkKeyDescriptor(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNwkKeyDescriptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.KeySeqNum != tt.want.KeySeqNum {
				t.Errorf("KeySeqNum = 0x%02X, want 0x%02X", got.KeySeqNum, tt.want.KeySeqNum)
			}
			if got.Key != tt.want.Key {
				t.Errorf("Key = %v, want %v", got.Key, tt.want.Key)
			}
		})
	}
}

// TestAddressManagerEntry tests address manager entry validation.
func TestAddressManagerEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   AddressManagerEntry
		isValid bool
	}{
		{
			name: "valid entry with association flag",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			isValid: true,
		},
		{
			name: "valid entry with security flag",
			entry: AddressManagerEntry{
				User:    AddrMgrUserSecurity,
				NwkAddr: 0x5678,
				ExtAddr: [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			},
			isValid: true,
		},
		{
			name: "valid entry with combined flags",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc | AddrMgrUserSecurity | AddrMgrUserBinding,
				NwkAddr: 0xABCD,
				ExtAddr: [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
			},
			isValid: true,
		},
		{
			name: "invalid - user is zero",
			entry: AddressManagerEntry{
				User:    0x00,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			isValid: false,
		},
		{
			name: "invalid - ExtAddr all zeros",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			isValid: false,
		},
		{
			name: "invalid - ExtAddr all FFs",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			isValid: false,
		},
		{
			name: "invalid - user zero and ExtAddr zeros",
			entry: AddressManagerEntry{
				User:    0x00,
				NwkAddr: 0x0000,
				ExtAddr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.IsValid()
			if got != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.isValid)
			}
		})
	}
}

// TestAddressManagerUserFlags tests the address manager user flag constants.
func TestAddressManagerUserFlags(t *testing.T) {
	if AddrMgrUserDefault != 0x00 {
		t.Errorf("AddrMgrUserDefault = 0x%02X, want 0x00", AddrMgrUserDefault)
	}
	if AddrMgrUserAssoc != 0x01 {
		t.Errorf("AddrMgrUserAssoc = 0x%02X, want 0x01", AddrMgrUserAssoc)
	}
	if AddrMgrUserSecurity != 0x02 {
		t.Errorf("AddrMgrUserSecurity = 0x%02X, want 0x02", AddrMgrUserSecurity)
	}
	if AddrMgrUserBinding != 0x04 {
		t.Errorf("AddrMgrUserBinding = 0x%02X, want 0x04", AddrMgrUserBinding)
	}

	// Test flag combinations
	combined := AddrMgrUserAssoc | AddrMgrUserSecurity
	if combined != 0x03 {
		t.Errorf("Assoc|Security = 0x%02X, want 0x03", combined)
	}

	allFlags := AddrMgrUserAssoc | AddrMgrUserSecurity | AddrMgrUserBinding
	if allFlags != 0x07 {
		t.Errorf("All flags = 0x%02X, want 0x07", allFlags)
	}
}

// TestApsLinkKeyEntry tests APS link key entry structure.
func TestApsLinkKeyEntry(t *testing.T) {
	entry := ApsLinkKeyEntry{
		Key:       [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		TxFrmCntr: 0x12345678,
		RxFrmCntr: 0x87654321,
	}

	// Test that structure can be created and accessed
	if entry.TxFrmCntr != 0x12345678 {
		t.Errorf("TxFrmCntr = 0x%08X, want 0x12345678", entry.TxFrmCntr)
	}
	if entry.RxFrmCntr != 0x87654321 {
		t.Errorf("RxFrmCntr = 0x%08X, want 0x87654321", entry.RxFrmCntr)
	}
	expectedKey := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	if entry.Key != expectedKey {
		t.Errorf("Key mismatch")
	}
}

// TestSecurityManagerEntry tests security manager entry structure.
func TestSecurityManagerEntry(t *testing.T) {
	entry := SecurityManagerEntry{
		Ami:     0x1234,
		KeyNvId: 0x5678,
	}

	if entry.Ami != 0x1234 {
		t.Errorf("Ami = 0x%04X, want 0x1234", entry.Ami)
	}
	if entry.KeyNvId != 0x5678 {
		t.Errorf("KeyNvId = 0x%04X, want 0x5678", entry.KeyNvId)
	}
}

// TestNIBChannelDecoding tests proper decoding of channel list from NIB.
func TestNIBChannelDecoding(t *testing.T) {
	tests := []struct {
		name        string
		channelMask uint32
		description string
	}{
		{
			name:        "default Zigbee channels 11-26",
			channelMask: 0x07FFF800,
			description: "Channels 11-26",
		},
		{
			name:        "single channel 11",
			channelMask: 0x00000800,
			description: "Channel 11",
		},
		{
			name:        "single channel 26",
			channelMask: 0x04000000,
			description: "Channel 26",
		},
		{
			name:        "all channels",
			channelMask: 0x07FFF800,
			description: "All default channels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 92)
			binary.LittleEndian.PutUint32(data[29:33], tt.channelMask)

			nib, err := ParseNIB(data)
			if err != nil {
				t.Fatalf("ParseNIB() error = %v", err)
			}

			if nib.ChannelList != tt.channelMask {
				t.Errorf("ChannelList = 0x%08X, want 0x%08X", nib.ChannelList, tt.channelMask)
			}
		})
	}
}

// TestNIBPanIDLittleEndian tests that PAN ID is correctly parsed as little-endian.
func TestNIBPanIDLittleEndian(t *testing.T) {
	tests := []struct {
		name      string
		bytes     []byte // Little-endian bytes
		wantPanID uint16
	}{
		{
			name:      "0x1A62",
			bytes:     []byte{0x62, 0x1A}, // Little-endian
			wantPanID: 0x1A62,
		},
		{
			name:      "0xFFFF",
			bytes:     []byte{0xFF, 0xFF},
			wantPanID: 0xFFFF,
		},
		{
			name:      "0x0000",
			bytes:     []byte{0x00, 0x00},
			wantPanID: 0x0000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 92)
			copy(data[38:40], tt.bytes)

			nib, err := ParseNIB(data)
			if err != nil {
				t.Fatalf("ParseNIB() error = %v", err)
			}

			if nib.PanID != tt.wantPanID {
				t.Errorf("PanID = 0x%04X, want 0x%04X", nib.PanID, tt.wantPanID)
			}
		})
	}
}

// TestParseNIBWithExtraData tests that ParseNIB handles data longer than minimum.
func TestParseNIBWithExtraData(t *testing.T) {
	// NIB can have extra data beyond byte 91 for newer versions
	data := make([]byte, 150)
	data[20] = 0x05
	data[42] = 15
	data[91] = 0x01

	nib, err := ParseNIB(data)
	if err != nil {
		t.Fatalf("ParseNIB() should handle extra data, error = %v", err)
	}

	if nib.SecurityLevel != 0x05 {
		t.Errorf("SecurityLevel = 0x%02X, want 0x05", nib.SecurityLevel)
	}
	if nib.LogicalChannel != 15 {
		t.Errorf("LogicalChannel = %d, want 15", nib.LogicalChannel)
	}
	if nib.UpdateID != 0x01 {
		t.Errorf("UpdateID = 0x%02X, want 0x01", nib.UpdateID)
	}
}

// TestParseNwkKeyDescriptorWithExtraData tests handling of extra data.
func TestParseNwkKeyDescriptorWithExtraData(t *testing.T) {
	// Minimum valid size is 17 bytes, test with more
	data := make([]byte, 25)
	data[0] = 0x42
	copy(data[1:17], bytes.Repeat([]byte{0xAA}, 16))

	kd, err := ParseNwkKeyDescriptor(data)
	if err != nil {
		t.Fatalf("ParseNwkKeyDescriptor() should handle extra data, error = %v", err)
	}

	if kd.KeySeqNum != 0x42 {
		t.Errorf("KeySeqNum = 0x%02X, want 0x42", kd.KeySeqNum)
	}
	expectedKey := [16]byte{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
		0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	if kd.Key != expectedKey {
		t.Errorf("Key mismatch")
	}
}

// TestAddressManagerEntryEdgeCases tests edge cases for address validation.
func TestAddressManagerEntryEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		entry   AddressManagerEntry
		isValid bool
	}{
		{
			name: "partial zeros in ExtAddr",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			isValid: true, // Not all zeros, so valid
		},
		{
			name: "partial FFs in ExtAddr",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0xFF, 0xFE, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			isValid: true, // Not all FFs, so valid
		},
		{
			name: "single byte different from zeros",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x0000,
				ExtAddr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			},
			isValid: true,
		},
		{
			name: "single byte different from FFs",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0xFFFF,
				ExtAddr: [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE},
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.IsValid()
			if got != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.isValid)
			}
		})
	}
}

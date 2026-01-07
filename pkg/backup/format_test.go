package backup

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBackupRoundtrip tests JSON marshal/unmarshal roundtrip.
func TestBackupRoundtrip(t *testing.T) {
	original := &Backup{
		Version: 1,
		Created: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Adapter: AdapterInfo{
			ZStackVariant: 3,
			SDKVersion:    "5.30.00.67",
			BuildDate:     20230615,
		},
		Coordinator: CoordinatorInfo{
			IEEEAddress:    "00124b0024c8e8e8",
			NetworkAddress: 0x0000,
		},
		Network: NetworkConfig{
			PanID:              0x1a62,
			ExtendedPanID:      "dddddddddddddddd",
			Channel:            15,
			NetworkKey:         "01030507090b0d0f00020406080a0c0d",
			NetworkKeySequence: 0,
			SecurityLevel:      5,
			UpdateID:           0,
		},
		Security: SecurityInfo{
			FrameCounter:           12345,
			TrustCenterLinkKeySeed: "abcdef1234567890abcdef1234567890",
		},
		Devices: []DeviceEntry{
			{
				IEEEAddress:    "00158d00048e5f9a",
				NetworkAddress: 0x1234,
				Type:           "router",
				LinkKey: &LinkKey{
					Key:       "5a6967426565416c6c69616e63653039",
					TxCounter: 100,
					RxCounter: 200,
				},
			},
			{
				IEEEAddress:    "00158d00048e5f9b",
				NetworkAddress: 0x5678,
				Type:           "endDevice",
				LinkKey:        nil,
			},
		},
	}

	// Marshal to JSON.
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Unmarshal back.
	var restored Backup
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Compare fields.
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", restored.Version, original.Version)
	}
	if !restored.Created.Equal(original.Created) {
		t.Errorf("Created mismatch: got %v, want %v", restored.Created, original.Created)
	}

	// Adapter.
	if restored.Adapter.ZStackVariant != original.Adapter.ZStackVariant {
		t.Errorf("ZStackVariant mismatch: got %d, want %d", restored.Adapter.ZStackVariant, original.Adapter.ZStackVariant)
	}
	if restored.Adapter.SDKVersion != original.Adapter.SDKVersion {
		t.Errorf("SDKVersion mismatch: got %s, want %s", restored.Adapter.SDKVersion, original.Adapter.SDKVersion)
	}
	if restored.Adapter.BuildDate != original.Adapter.BuildDate {
		t.Errorf("BuildDate mismatch: got %d, want %d", restored.Adapter.BuildDate, original.Adapter.BuildDate)
	}

	// Coordinator.
	if restored.Coordinator.IEEEAddress != original.Coordinator.IEEEAddress {
		t.Errorf("Coordinator IEEEAddress mismatch: got %s, want %s", restored.Coordinator.IEEEAddress, original.Coordinator.IEEEAddress)
	}
	if restored.Coordinator.NetworkAddress != original.Coordinator.NetworkAddress {
		t.Errorf("Coordinator NetworkAddress mismatch: got 0x%04x, want 0x%04x", restored.Coordinator.NetworkAddress, original.Coordinator.NetworkAddress)
	}

	// Network.
	if restored.Network.PanID != original.Network.PanID {
		t.Errorf("PanID mismatch: got 0x%04x, want 0x%04x", restored.Network.PanID, original.Network.PanID)
	}
	if restored.Network.ExtendedPanID != original.Network.ExtendedPanID {
		t.Errorf("ExtendedPanID mismatch: got %s, want %s", restored.Network.ExtendedPanID, original.Network.ExtendedPanID)
	}
	if restored.Network.Channel != original.Network.Channel {
		t.Errorf("Channel mismatch: got %d, want %d", restored.Network.Channel, original.Network.Channel)
	}
	if restored.Network.NetworkKey != original.Network.NetworkKey {
		t.Errorf("NetworkKey mismatch: got %s, want %s", restored.Network.NetworkKey, original.Network.NetworkKey)
	}
	if restored.Network.NetworkKeySequence != original.Network.NetworkKeySequence {
		t.Errorf("NetworkKeySequence mismatch: got %d, want %d", restored.Network.NetworkKeySequence, original.Network.NetworkKeySequence)
	}
	if restored.Network.SecurityLevel != original.Network.SecurityLevel {
		t.Errorf("SecurityLevel mismatch: got %d, want %d", restored.Network.SecurityLevel, original.Network.SecurityLevel)
	}
	if restored.Network.UpdateID != original.Network.UpdateID {
		t.Errorf("UpdateID mismatch: got %d, want %d", restored.Network.UpdateID, original.Network.UpdateID)
	}

	// Security.
	if restored.Security.FrameCounter != original.Security.FrameCounter {
		t.Errorf("FrameCounter mismatch: got %d, want %d", restored.Security.FrameCounter, original.Security.FrameCounter)
	}
	if restored.Security.TrustCenterLinkKeySeed != original.Security.TrustCenterLinkKeySeed {
		t.Errorf("TrustCenterLinkKeySeed mismatch: got %s, want %s", restored.Security.TrustCenterLinkKeySeed, original.Security.TrustCenterLinkKeySeed)
	}

	// Devices.
	if len(restored.Devices) != len(original.Devices) {
		t.Fatalf("Devices length mismatch: got %d, want %d", len(restored.Devices), len(original.Devices))
	}
	for i := range original.Devices {
		if restored.Devices[i].IEEEAddress != original.Devices[i].IEEEAddress {
			t.Errorf("Device[%d] IEEEAddress mismatch: got %s, want %s", i, restored.Devices[i].IEEEAddress, original.Devices[i].IEEEAddress)
		}
		if restored.Devices[i].NetworkAddress != original.Devices[i].NetworkAddress {
			t.Errorf("Device[%d] NetworkAddress mismatch: got 0x%04x, want 0x%04x", i, restored.Devices[i].NetworkAddress, original.Devices[i].NetworkAddress)
		}
		if restored.Devices[i].Type != original.Devices[i].Type {
			t.Errorf("Device[%d] Type mismatch: got %s, want %s", i, restored.Devices[i].Type, original.Devices[i].Type)
		}

		// LinkKey comparison.
		switch {
		case original.Devices[i].LinkKey == nil && restored.Devices[i].LinkKey != nil:
			t.Errorf("Device[%d] LinkKey should be nil", i)
		case original.Devices[i].LinkKey != nil && restored.Devices[i].LinkKey == nil:
			t.Errorf("Device[%d] LinkKey should not be nil", i)
		case original.Devices[i].LinkKey != nil && restored.Devices[i].LinkKey != nil:
			if restored.Devices[i].LinkKey.Key != original.Devices[i].LinkKey.Key {
				t.Errorf("Device[%d] LinkKey.Key mismatch: got %s, want %s", i, restored.Devices[i].LinkKey.Key, original.Devices[i].LinkKey.Key)
			}
			if restored.Devices[i].LinkKey.TxCounter != original.Devices[i].LinkKey.TxCounter {
				t.Errorf("Device[%d] LinkKey.TxCounter mismatch: got %d, want %d", i, restored.Devices[i].LinkKey.TxCounter, original.Devices[i].LinkKey.TxCounter)
			}
			if restored.Devices[i].LinkKey.RxCounter != original.Devices[i].LinkKey.RxCounter {
				t.Errorf("Device[%d] LinkKey.RxCounter mismatch: got %d, want %d", i, restored.Devices[i].LinkKey.RxCounter, original.Devices[i].LinkKey.RxCounter)
			}
		}
	}
}

// TestToJSONFromJSON tests the ToJSON and FromJSON helper methods.
func TestToJSONFromJSON(t *testing.T) {
	original := &Backup{
		Version: 1,
		Created: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Coordinator: CoordinatorInfo{
			IEEEAddress:    "00124b0024c8e8e8",
			NetworkAddress: 0x0000,
		},
		Network: NetworkConfig{
			NetworkKey: "01030507090b0d0f00020406080a0c0d",
		},
		Devices: []DeviceEntry{},
	}

	// Convert to JSON.
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	// Verify JSON is indented (should contain newlines).
	if len(data) == 0 {
		t.Fatal("ToJSON returned empty data")
	}

	// Convert back from JSON.
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON error: %v", err)
	}

	// Basic field checks.
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", restored.Version, original.Version)
	}
	if restored.Coordinator.IEEEAddress != original.Coordinator.IEEEAddress {
		t.Errorf("IEEEAddress mismatch: got %s, want %s", restored.Coordinator.IEEEAddress, original.Coordinator.IEEEAddress)
	}
}

// TestValidate tests the Validate method.
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		backup  *Backup
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid backup",
			backup: &Backup{
				Version: 1,
				Coordinator: CoordinatorInfo{
					IEEEAddress: "00124b0024c8e8e8",
				},
				Network: NetworkConfig{
					NetworkKey: "01030507090b0d0f00020406080a0c0d",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid version",
			backup: &Backup{
				Version: 999,
				Coordinator: CoordinatorInfo{
					IEEEAddress: "00124b0024c8e8e8",
				},
				Network: NetworkConfig{
					NetworkKey: "01030507090b0d0f00020406080a0c0d",
				},
			},
			wantErr: true,
			errMsg:  "unsupported backup version",
		},
		{
			name: "missing coordinator IEEE address",
			backup: &Backup{
				Version: 1,
				Network: NetworkConfig{
					NetworkKey: "01030507090b0d0f00020406080a0c0d",
				},
			},
			wantErr: true,
			errMsg:  "missing coordinator IEEE address",
		},
		{
			name: "missing network key",
			backup: &Backup{
				Version: 1,
				Coordinator: CoordinatorInfo{
					IEEEAddress: "00124b0024c8e8e8",
				},
			},
			wantErr: true,
			errMsg:  "missing network key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backup.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestNew tests the New constructor.
func TestNew(t *testing.T) {
	backup := New()

	if backup == nil {
		t.Fatal("New() returned nil")
	}
	if backup.Version != FormatVersion {
		t.Errorf("Version = %d, want %d", backup.Version, FormatVersion)
	}
	if backup.Devices == nil {
		t.Error("Devices should be initialized, not nil")
	}
	if len(backup.Devices) != 0 {
		t.Errorf("Devices length = %d, want 0", len(backup.Devices))
	}
	if backup.Created.IsZero() {
		t.Error("Created should be set to current time")
	}
}

// TestNetworkConfigSerialization tests NetworkConfig hex field serialization.
func TestNetworkConfigSerialization(t *testing.T) {
	config := NetworkConfig{
		PanID:              0x1a62,
		ExtendedPanID:      "dddddddddddddddd",
		Channel:            15,
		NetworkKey:         "01030507090b0d0f00020406080a0c0d",
		NetworkKeySequence: 0,
		SecurityLevel:      5,
		UpdateID:           0,
	}

	// Marshal to JSON.
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Verify JSON contains hex strings in quotes.
	jsonStr := string(data)
	if !contains(jsonStr, "dddddddddddddddd") {
		t.Errorf("JSON should contain ExtendedPanID hex string: %s", jsonStr)
	}
	if !contains(jsonStr, "01030507090b0d0f00020406080a0c0d") {
		t.Errorf("JSON should contain NetworkKey hex string: %s", jsonStr)
	}

	// Unmarshal back.
	var restored NetworkConfig
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Verify fields match.
	if restored.ExtendedPanID != config.ExtendedPanID {
		t.Errorf("ExtendedPanID mismatch: got %s, want %s", restored.ExtendedPanID, config.ExtendedPanID)
	}
	if restored.NetworkKey != config.NetworkKey {
		t.Errorf("NetworkKey mismatch: got %s, want %s", restored.NetworkKey, config.NetworkKey)
	}
	if restored.Channel != config.Channel {
		t.Errorf("Channel mismatch: got %d, want %d", restored.Channel, config.Channel)
	}
}

// TestEncodeDecodeHex tests EncodeHex and DecodeHex functions.
func TestEncodeDecodeHex(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		hex   string
	}{
		{
			name:  "empty",
			bytes: []byte{},
			hex:   "",
		},
		{
			name:  "single byte",
			bytes: []byte{0xab},
			hex:   "ab",
		},
		{
			name:  "multiple bytes",
			bytes: []byte{0x01, 0x03, 0x05, 0x07, 0x09, 0x0b, 0x0d, 0x0f},
			hex:   "01030507090b0d0f",
		},
		{
			name:  "16 bytes (network key)",
			bytes: []byte{0x01, 0x03, 0x05, 0x07, 0x09, 0x0b, 0x0d, 0x0f, 0x00, 0x02, 0x04, 0x06, 0x08, 0x0a, 0x0c, 0x0d},
			hex:   "01030507090b0d0f00020406080a0c0d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encode.
			encoded := EncodeHex(tt.bytes)
			if encoded != tt.hex {
				t.Errorf("EncodeHex() = %s, want %s", encoded, tt.hex)
			}

			// Test decode.
			decoded, err := DecodeHex(tt.hex)
			if err != nil {
				t.Fatalf("DecodeHex() error: %v", err)
			}
			if len(decoded) != len(tt.bytes) {
				t.Errorf("decoded length = %d, want %d", len(decoded), len(tt.bytes))
			}
			for i := range tt.bytes {
				if decoded[i] != tt.bytes[i] {
					t.Errorf("decoded[%d] = 0x%02x, want 0x%02x", i, decoded[i], tt.bytes[i])
				}
			}
		})
	}
}

// TestDecodeHexInvalid tests DecodeHex with invalid input.
func TestDecodeHexInvalid(t *testing.T) {
	tests := []struct {
		name string
		hex  string
	}{
		{
			name: "odd length",
			hex:  "abc",
		},
		{
			name: "invalid characters",
			hex:  "gg",
		},
		{
			name: "mixed case with invalid",
			hex:  "abcdefGZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeHex(tt.hex)
			if err == nil {
				t.Errorf("DecodeHex(%q) expected error, got nil", tt.hex)
			}
		})
	}
}

// TestEncodeDecodeIEEEAddr tests IEEE address encoding/decoding.
func TestEncodeDecodeIEEEAddr(t *testing.T) {
	tests := []struct {
		name string
		addr [8]byte
		hex  string
	}{
		{
			name: "all zeros",
			addr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			hex:  "0000000000000000",
		},
		{
			name: "typical IEEE address",
			addr: [8]byte{0x00, 0x12, 0x4b, 0x00, 0x24, 0xc8, 0xe8, 0xe8},
			hex:  "00124b0024c8e8e8",
		},
		{
			name: "all 0xff",
			addr: [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			hex:  "ffffffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encode.
			encoded := EncodeIEEEAddr(tt.addr)
			if encoded != tt.hex {
				t.Errorf("EncodeIEEEAddr() = %s, want %s", encoded, tt.hex)
			}

			// Test decode.
			decoded, err := DecodeIEEEAddr(tt.hex)
			if err != nil {
				t.Fatalf("DecodeIEEEAddr() error: %v", err)
			}
			if decoded != tt.addr {
				t.Errorf("DecodeIEEEAddr() = %v, want %v", decoded, tt.addr)
			}
		})
	}
}

// TestDecodeIEEEAddrInvalid tests DecodeIEEEAddr with invalid input.
func TestDecodeIEEEAddrInvalid(t *testing.T) {
	tests := []struct {
		name string
		hex  string
	}{
		{
			name: "too short",
			hex:  "00124b00",
		},
		{
			name: "too long",
			hex:  "00124b0024c8e8e8ff",
		},
		{
			name: "invalid characters",
			hex:  "00124b0024c8e8zz",
		},
		{
			name: "odd length",
			hex:  "00124b0024c8e8e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeIEEEAddr(tt.hex)
			if err == nil {
				t.Errorf("DecodeIEEEAddr(%q) expected error, got nil", tt.hex)
			}
		})
	}
}

// TestDeviceEntrySerialization tests DeviceEntry with and without LinkKey.
func TestDeviceEntrySerialization(t *testing.T) {
	tests := []struct {
		name   string
		device DeviceEntry
	}{
		{
			name: "with link key",
			device: DeviceEntry{
				IEEEAddress:    "00158d00048e5f9a",
				NetworkAddress: 0x1234,
				Type:           "router",
				LinkKey: &LinkKey{
					Key:       "5a6967426565416c6c69616e63653039",
					TxCounter: 100,
					RxCounter: 200,
				},
			},
		},
		{
			name: "without link key",
			device: DeviceEntry{
				IEEEAddress:    "00158d00048e5f9b",
				NetworkAddress: 0x5678,
				Type:           "endDevice",
				LinkKey:        nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal.
			data, err := json.Marshal(tt.device)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			// Unmarshal.
			var restored DeviceEntry
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			// Compare.
			if restored.IEEEAddress != tt.device.IEEEAddress {
				t.Errorf("IEEEAddress mismatch: got %s, want %s", restored.IEEEAddress, tt.device.IEEEAddress)
			}
			if restored.NetworkAddress != tt.device.NetworkAddress {
				t.Errorf("NetworkAddress mismatch: got 0x%04x, want 0x%04x", restored.NetworkAddress, tt.device.NetworkAddress)
			}
			if restored.Type != tt.device.Type {
				t.Errorf("Type mismatch: got %s, want %s", restored.Type, tt.device.Type)
			}

			// LinkKey comparison.
			if tt.device.LinkKey == nil {
				if restored.LinkKey != nil {
					t.Error("LinkKey should be nil")
				}
			} else {
				if restored.LinkKey == nil {
					t.Fatal("LinkKey should not be nil")
				}
				if restored.LinkKey.Key != tt.device.LinkKey.Key {
					t.Errorf("LinkKey.Key mismatch: got %s, want %s", restored.LinkKey.Key, tt.device.LinkKey.Key)
				}
				if restored.LinkKey.TxCounter != tt.device.LinkKey.TxCounter {
					t.Errorf("LinkKey.TxCounter mismatch: got %d, want %d", restored.LinkKey.TxCounter, tt.device.LinkKey.TxCounter)
				}
				if restored.LinkKey.RxCounter != tt.device.LinkKey.RxCounter {
					t.Errorf("LinkKey.RxCounter mismatch: got %d, want %d", restored.LinkKey.RxCounter, tt.device.LinkKey.RxCounter)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && containsImpl(s, substr)))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

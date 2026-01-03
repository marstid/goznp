package znp

import (
	"testing"
)

func TestSerializeDeviceNameEntry(t *testing.T) {
	entry := &DeviceNameEntry{
		IEEEAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:        "Living Room Light",
		Description: "Main ceiling light in the living room",
	}

	data := SerializeDeviceNameEntry(entry)

	if len(data) != DeviceNameEntrySize {
		t.Errorf("expected size %d, got %d", DeviceNameEntrySize, len(data))
	}

	// Parse it back
	parsed, err := ParseDeviceNameEntry(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
	}

	if parsed.IEEEAddr != entry.IEEEAddr {
		t.Errorf("IEEE address mismatch: expected %v, got %v", entry.IEEEAddr, parsed.IEEEAddr)
	}
	if parsed.Name != entry.Name {
		t.Errorf("name mismatch: expected %q, got %q", entry.Name, parsed.Name)
	}
	if parsed.Description != entry.Description {
		t.Errorf("description mismatch: expected %q, got %q", entry.Description, parsed.Description)
	}
}

func TestSerializeDeviceNameTable(t *testing.T) {
	table := &DeviceNameTable{
		Version: DeviceNameTableVersion,
		Entries: []DeviceNameEntry{
			{
				IEEEAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Name:        "Device 1",
				Description: "First device",
			},
			{
				IEEEAddr:    [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
				Name:        "Device 2",
				Description: "Second device",
			},
		},
	}

	data := SerializeDeviceNameTable(table)

	expectedSize := DeviceNameHeaderSize + (len(table.Entries) * DeviceNameEntrySize)
	if len(data) != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, len(data))
	}

	// Parse it back
	parsed, err := ParseDeviceNameTable(data)
	if err != nil {
		t.Fatalf("failed to parse table: %v", err)
	}

	if parsed.Version != table.Version {
		t.Errorf("version mismatch: expected %d, got %d", table.Version, parsed.Version)
	}
	if len(parsed.Entries) != len(table.Entries) {
		t.Fatalf("entry count mismatch: expected %d, got %d", len(table.Entries), len(parsed.Entries))
	}

	for i := range table.Entries {
		if parsed.Entries[i].IEEEAddr != table.Entries[i].IEEEAddr {
			t.Errorf("entry %d IEEE address mismatch", i)
		}
		if parsed.Entries[i].Name != table.Entries[i].Name {
			t.Errorf("entry %d name mismatch: expected %q, got %q", i, table.Entries[i].Name, parsed.Entries[i].Name)
		}
		if parsed.Entries[i].Description != table.Entries[i].Description {
			t.Errorf("entry %d description mismatch", i)
		}
	}
}

func TestDeviceNameEntry_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		entry    DeviceNameEntry
		expected bool
	}{
		{
			name:     "empty entry (all zeros)",
			entry:    DeviceNameEntry{},
			expected: true,
		},
		{
			name: "empty with name set",
			entry: DeviceNameEntry{
				Name: "Test",
			},
			expected: true, // IEEE addr still all zeros
		},
		{
			name: "non-empty (first byte set)",
			entry: DeviceNameEntry{
				IEEEAddr: [8]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			expected: false,
		},
		{
			name: "non-empty (last byte set)",
			entry: DeviceNameEntry{
				IEEEAddr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			},
			expected: false,
		},
		{
			name: "fully populated entry",
			entry: DeviceNameEntry{
				IEEEAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Name:        "Test Device",
				Description: "A test device",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.IsEmpty()
			if got != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestSerializeDeviceNameEntry_TruncatesLongName(t *testing.T) {
	longName := "This is a very long name that exceeds the 32 character limit for device names"
	entry := &DeviceNameEntry{
		IEEEAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:        longName,
		Description: "Short desc",
	}

	data := SerializeDeviceNameEntry(entry)
	parsed, err := ParseDeviceNameEntry(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
	}

	if len(parsed.Name) != DeviceNameMaxName {
		t.Errorf("expected truncated name length %d, got %d", DeviceNameMaxName, len(parsed.Name))
	}
	if parsed.Name != longName[:DeviceNameMaxName] {
		t.Errorf("name not truncated correctly")
	}
}

func TestSerializeDeviceNameEntry_TruncatesLongDescription(t *testing.T) {
	longDesc := "This is a very long description that exceeds the 64 character limit for device descriptions in the NVRAM storage"
	entry := &DeviceNameEntry{
		IEEEAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:        "Short name",
		Description: longDesc,
	}

	data := SerializeDeviceNameEntry(entry)
	parsed, err := ParseDeviceNameEntry(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
	}

	if len(parsed.Description) != DeviceNameMaxDescription {
		t.Errorf("expected truncated description length %d, got %d", DeviceNameMaxDescription, len(parsed.Description))
	}
	if parsed.Description != longDesc[:DeviceNameMaxDescription] {
		t.Errorf("description not truncated correctly")
	}
}

func TestParseDeviceNameEntry_ErrorOnShortData(t *testing.T) {
	shortData := make([]byte, DeviceNameEntrySize-1)
	_, err := ParseDeviceNameEntry(shortData)
	if err == nil {
		t.Error("expected error for short data, got nil")
	}
}

func TestParseDeviceNameTable_ErrorOnShortHeader(t *testing.T) {
	shortData := make([]byte, DeviceNameHeaderSize-1)
	_, err := ParseDeviceNameTable(shortData)
	if err == nil {
		t.Error("expected error for short header, got nil")
	}
}

func TestParseDeviceNameTable_ErrorOnInsufficientEntryData(t *testing.T) {
	// Create header saying we have 2 entries but only provide data for 1
	data := make([]byte, DeviceNameHeaderSize+DeviceNameEntrySize)
	data[0] = byte(DeviceNameTableVersion & 0xFF)
	data[1] = byte(DeviceNameTableVersion >> 8)
	data[2] = 2 // count = 2
	data[3] = 0

	_, err := ParseDeviceNameTable(data)
	if err == nil {
		t.Error("expected error for insufficient entry data, got nil")
	}
}

func TestSerializeDeviceNameEntry_EmptyStrings(t *testing.T) {
	entry := &DeviceNameEntry{
		IEEEAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:        "",
		Description: "",
	}

	data := SerializeDeviceNameEntry(entry)
	parsed, err := ParseDeviceNameEntry(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
	}

	if parsed.Name != "" {
		t.Errorf("expected empty name, got %q", parsed.Name)
	}
	if parsed.Description != "" {
		t.Errorf("expected empty description, got %q", parsed.Description)
	}
}

func TestDeviceNameTable_EmptyTable(t *testing.T) {
	table := &DeviceNameTable{
		Version: DeviceNameTableVersion,
		Entries: []DeviceNameEntry{},
	}

	data := SerializeDeviceNameTable(table)
	if len(data) != DeviceNameHeaderSize {
		t.Errorf("expected size %d for empty table, got %d", DeviceNameHeaderSize, len(data))
	}

	parsed, err := ParseDeviceNameTable(data)
	if err != nil {
		t.Fatalf("failed to parse empty table: %v", err)
	}

	if parsed.Version != DeviceNameTableVersion {
		t.Errorf("version mismatch: expected %d, got %d", DeviceNameTableVersion, parsed.Version)
	}
	if len(parsed.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(parsed.Entries))
	}
}

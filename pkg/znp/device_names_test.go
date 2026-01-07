package znp

import (
	"testing"
)

func TestSerializeDeviceNameEntry(t *testing.T) {
	entry := &DeviceNameEntry{
		IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:     "Living Room Light",
		Comment:  "Main ceiling light in the living room",
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
	if parsed.Comment != entry.Comment {
		t.Errorf("comment mismatch: expected %q, got %q", entry.Comment, parsed.Comment)
	}
}

func TestSerializeDeviceNameTable(t *testing.T) {
	table := &DeviceNameTable{
		Version: DeviceNameTableVersion,
		Entries: []DeviceNameEntry{
			{
				IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Name:     "Device 1",
				Comment:  "First device",
			},
			{
				IEEEAddr: [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
				Name:     "Device 2",
				Comment:  "Second device",
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
		if parsed.Entries[i].Comment != table.Entries[i].Comment {
			t.Errorf("entry %d comment mismatch", i)
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
				IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Name:     "Test Device",
				Comment:  "A test device",
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
		IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:     longName,
		Comment:  "Short comment",
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

func TestSerializeDeviceNameEntry_TruncatesLongComment(t *testing.T) {
	longComment := "This is a very long comment that exceeds the 64 character limit for device comments in the NVRAM storage"
	entry := &DeviceNameEntry{
		IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:     "Short name",
		Comment:  longComment,
	}

	data := SerializeDeviceNameEntry(entry)
	parsed, err := ParseDeviceNameEntry(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
	}

	if len(parsed.Comment) != DeviceNameMaxComment {
		t.Errorf("expected truncated comment length %d, got %d", DeviceNameMaxComment, len(parsed.Comment))
	}
	if parsed.Comment != longComment[:DeviceNameMaxComment] {
		t.Errorf("comment not truncated correctly")
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
		IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:     "",
		Comment:  "",
	}

	data := SerializeDeviceNameEntry(entry)
	parsed, err := ParseDeviceNameEntry(data)
	if err != nil {
		t.Fatalf("failed to parse entry: %v", err)
	}

	if parsed.Name != "" {
		t.Errorf("expected empty name, got %q", parsed.Name)
	}
	if parsed.Comment != "" {
		t.Errorf("expected empty comment, got %q", parsed.Comment)
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

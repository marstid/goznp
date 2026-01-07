package znp

import (
	"context"
	"fmt"
)

// Device name table constants.
const (
	// DeviceNameMaxName is the maximum length of a device name.
	DeviceNameMaxName = 32
	// DeviceNameMaxComment is the maximum length of a device comment.
	DeviceNameMaxComment = 64
	// DeviceNameEntrySize is the size of each device name entry in bytes.
	// Layout: IEEE(8) + NameLen(1) + Name(32) + DescLen(1) + Desc(64) + Reserved(2)
	DeviceNameEntrySize = 108
	// DeviceNameTableVersion is the format version of the device name table.
	DeviceNameTableVersion = 0x0001
	// DeviceNameHeaderSize is the size of the table header in bytes.
	// Layout: Version(2) + Count(2)
	DeviceNameHeaderSize = 4

	// Extended NV system identifiers for device names.
	// Using a custom sysId (0x02) separate from Z-Stack's sysId (0x01).
	nvSysUser          uint8  = 0x02   // Custom user system ID
	nvItemDeviceNames  uint16 = 0x0001 // Device names item ID
	nvMaxDeviceEntries        = 200    // Max entries to iterate
)

// DeviceNameEntry represents a custom name and comment for a Zigbee device.
type DeviceNameEntry struct {
	IEEEAddr [8]byte // Device IEEE address
	Name     string  // Custom name (max 32 bytes UTF-8)
	Comment  string  // Custom comment (max 64 bytes UTF-8)
}

// IsEmpty returns true if the entry is empty (all IEEE address bytes are zero).
func (e *DeviceNameEntry) IsEmpty() bool {
	for _, b := range e.IEEEAddr {
		if b != 0 {
			return false
		}
	}
	return true
}

// DeviceNameTable represents the complete device name table stored in NVRAM.
type DeviceNameTable struct {
	Version uint16            // Format version
	Entries []DeviceNameEntry // Device name entries
}

// SerializeDeviceNameEntry serializes a single device name entry to binary format.
// Binary layout (108 bytes):
// - Offset 0: IEEE Address (8 bytes)
// - Offset 8: Name length (1 byte, 0-32)
// - Offset 9: Name (32 bytes, UTF-8, null-padded)
// - Offset 41: Comment length (1 byte, 0-64)
// - Offset 42: Comment (64 bytes, UTF-8, null-padded)
// - Offset 106: Reserved (2 bytes)
func SerializeDeviceNameEntry(entry *DeviceNameEntry) []byte {
	writer := NewBuffaloWriter()

	// IEEE Address (8 bytes)
	writer.WriteIEEEAddr(entry.IEEEAddr)

	// Name field: length byte + 32-byte buffer
	nameBytes := []byte(entry.Name)
	nameLen := len(nameBytes)
	if nameLen > DeviceNameMaxName {
		nameLen = DeviceNameMaxName
	}
	writer.WriteUint8(uint8(nameLen))
	// Write name bytes (up to 32)
	writer.WriteBytes(nameBytes[:nameLen])
	// Pad to 32 bytes
	for i := nameLen; i < DeviceNameMaxName; i++ {
		writer.WriteUint8(0)
	}

	// Comment field: length byte + 64-byte buffer
	commentBytes := []byte(entry.Comment)
	commentLen := len(commentBytes)
	if commentLen > DeviceNameMaxComment {
		commentLen = DeviceNameMaxComment
	}
	writer.WriteUint8(uint8(commentLen))
	// Write comment bytes (up to 64)
	writer.WriteBytes(commentBytes[:commentLen])
	// Pad to 64 bytes
	for i := commentLen; i < DeviceNameMaxComment; i++ {
		writer.WriteUint8(0)
	}

	// Reserved (2 bytes)
	writer.WriteUint16(0)

	return writer.Bytes()
}

// ParseDeviceNameEntry parses a single device name entry from binary format.
// Returns an error if the data is too short.
func ParseDeviceNameEntry(data []byte) (*DeviceNameEntry, error) {
	if len(data) < DeviceNameEntrySize {
		return nil, fmt.Errorf("invalid device name entry size: need %d bytes, have %d", DeviceNameEntrySize, len(data))
	}

	buf := NewBuffalo(data)

	// IEEE Address (8 bytes)
	ieeeAddr, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read IEEE address: %w", err)
	}

	// Name field: length byte + 32-byte buffer
	nameLen, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read name length: %w", err)
	}
	if nameLen > DeviceNameMaxName {
		nameLen = DeviceNameMaxName
	}
	nameBytes, err := buf.ReadBytes(DeviceNameMaxName)
	if err != nil {
		return nil, fmt.Errorf("failed to read name bytes: %w", err)
	}
	name := string(nameBytes[:nameLen])

	// Comment field: length byte + 64-byte buffer
	commentLen, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read comment length: %w", err)
	}
	if commentLen > DeviceNameMaxComment {
		commentLen = DeviceNameMaxComment
	}
	commentBytes, err := buf.ReadBytes(DeviceNameMaxComment)
	if err != nil {
		return nil, fmt.Errorf("failed to read comment bytes: %w", err)
	}
	comment := string(commentBytes[:commentLen])

	// Reserved (2 bytes) - skip
	_, err = buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read reserved bytes: %w", err)
	}

	return &DeviceNameEntry{
		IEEEAddr: ieeeAddr,
		Name:     name,
		Comment:  comment,
	}, nil
}

// SerializeDeviceNameTable serializes the complete device name table to binary format.
// Binary layout:
// - Offset 0: Version (2 bytes, uint16 LE)
// - Offset 2: Entry count (2 bytes, uint16 LE)
// - Offset 4: Entries (N * 108 bytes)
func SerializeDeviceNameTable(table *DeviceNameTable) []byte {
	writer := NewBuffaloWriter()

	// Header: version (2) + count (2)
	writer.WriteUint16(table.Version)
	writer.WriteUint16(uint16(len(table.Entries)))

	// Entries
	for i := range table.Entries {
		entryBytes := SerializeDeviceNameEntry(&table.Entries[i])
		writer.WriteBytes(entryBytes)
	}

	return writer.Bytes()
}

// ParseDeviceNameTable parses the complete device name table from binary format.
// Returns an error if the data format is invalid.
func ParseDeviceNameTable(data []byte) (*DeviceNameTable, error) {
	if len(data) < DeviceNameHeaderSize {
		return nil, fmt.Errorf("invalid device name table: need at least %d bytes for header, have %d", DeviceNameHeaderSize, len(data))
	}

	buf := NewBuffalo(data)

	// Read header
	version, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	count, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read entry count: %w", err)
	}

	// Check expected data size
	expectedSize := DeviceNameHeaderSize + (int(count) * DeviceNameEntrySize)
	if len(data) < expectedSize {
		return nil, fmt.Errorf("invalid device name table: need %d bytes for %d entries, have %d", expectedSize, count, len(data))
	}

	// Parse entries
	entries := make([]DeviceNameEntry, 0, count)
	for i := 0; i < int(count); i++ {
		entryData, err := buf.ReadBytes(DeviceNameEntrySize)
		if err != nil {
			return nil, fmt.Errorf("failed to read entry %d: %w", i, err)
		}

		entry, err := ParseDeviceNameEntry(entryData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse entry %d: %w", i, err)
		}

		entries = append(entries, *entry)
	}

	return &DeviceNameTable{
		Version: version,
		Entries: entries,
	}, nil
}

// ReadDeviceNameTable reads the device name table from NVRAM using Extended NV system.
// Returns an empty table if no entries exist.
func (z *ZNP) ReadDeviceNameTable(ctx context.Context) (*DeviceNameTable, error) {
	table := &DeviceNameTable{
		Version: DeviceNameTableVersion,
		Entries: []DeviceNameEntry{},
	}

	// Read entries from Extended NV, iterating through subIDs
	for subID := uint16(0); subID < nvMaxDeviceEntries; subID++ {
		data, err := z.NvReadEx(ctx, nvSysUser, nvItemDeviceNames, subID, 0, DeviceNameEntrySize)
		if err != nil {
			return nil, fmt.Errorf("failed to read device name entry %d: %w", subID, err)
		}

		// nil means not found - end of table
		if data == nil {
			break
		}

		// Parse the entry
		entry, err := ParseDeviceNameEntry(data)
		if err != nil {
			// Skip corrupt entries
			continue
		}

		table.Entries = append(table.Entries, *entry)
	}

	return table, nil
}

// WriteDeviceNameTable writes the device name table to NVRAM using Extended NV system.
func (z *ZNP) WriteDeviceNameTable(ctx context.Context, table *DeviceNameTable) error {
	// Write each entry to a separate subId
	for i, entry := range table.Entries {
		data := SerializeDeviceNameEntry(&entry)
		if err := z.NvWriteEx(ctx, nvSysUser, nvItemDeviceNames, uint16(i), 0, data); err != nil {
			return fmt.Errorf("failed to write device name entry %d: %w", i, err)
		}
	}

	return nil
}

// GetDeviceName retrieves the name and comment for a specific device.
// Returns nil if the device is not found in the table.
func (z *ZNP) GetDeviceName(ctx context.Context, ieeeAddr [8]byte) (*DeviceNameEntry, error) {
	// Read the table
	table, err := z.ReadDeviceNameTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read device name table: %w", err)
	}

	// Search for the device
	for i := range table.Entries {
		if table.Entries[i].IEEEAddr == ieeeAddr && !table.Entries[i].IsEmpty() {
			return &table.Entries[i], nil
		}
	}

	// Not found - return nil (not an error)
	return nil, nil
}

// SetDeviceName sets or updates the name and comment for a specific device.
// If the device already has an entry, it's updated. Otherwise, a new entry is added.
func (z *ZNP) SetDeviceName(ctx context.Context, ieeeAddr [8]byte, name, comment string) error {
	// Validate name and comment lengths
	if len(name) > DeviceNameMaxName {
		return fmt.Errorf("name too long: %d bytes (max %d)", len(name), DeviceNameMaxName)
	}
	if len(comment) > DeviceNameMaxComment {
		return fmt.Errorf("comment too long: %d bytes (max %d)", len(comment), DeviceNameMaxComment)
	}

	// Read the current table
	table, err := z.ReadDeviceNameTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to read device name table: %w", err)
	}

	// Try to find existing entry or empty slot
	foundIndex := -1
	emptyIndex := -1

	for i := range table.Entries {
		if table.Entries[i].IEEEAddr == ieeeAddr && !table.Entries[i].IsEmpty() {
			// Found existing entry
			foundIndex = i
			break
		}
		if table.Entries[i].IsEmpty() && emptyIndex == -1 {
			// Found first empty slot
			emptyIndex = i
		}
	}

	// Create the new entry
	newEntry := DeviceNameEntry{
		IEEEAddr: ieeeAddr,
		Name:     name,
		Comment:  comment,
	}

	switch {
	case foundIndex != -1:
		// Update existing entry
		table.Entries[foundIndex] = newEntry
	case emptyIndex != -1:
		// Use empty slot
		table.Entries[emptyIndex] = newEntry
	default:
		// Append new entry
		table.Entries = append(table.Entries, newEntry)
	}

	// Write the updated table
	if err := z.WriteDeviceNameTable(ctx, table); err != nil {
		return fmt.Errorf("failed to write updated device name table: %w", err)
	}

	return nil
}

// DeleteDeviceName removes the name and comment for a specific device.
// The entry is marked as empty by zeroing out the IEEE address.
func (z *ZNP) DeleteDeviceName(ctx context.Context, ieeeAddr [8]byte) error {
	// Read the current table
	table, err := z.ReadDeviceNameTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to read device name table: %w", err)
	}

	// Find the entry
	found := false
	for i := range table.Entries {
		if table.Entries[i].IEEEAddr != ieeeAddr || table.Entries[i].IsEmpty() {
			continue
		}
		// Mark as empty by zeroing IEEE address
		table.Entries[i].IEEEAddr = [8]byte{}
		table.Entries[i].Name = ""
		table.Entries[i].Comment = ""
		found = true
		break
	}

	// If not found, nothing to do (not an error)
	if !found {
		return nil
	}

	// Write the updated table
	if err := z.WriteDeviceNameTable(ctx, table); err != nil {
		return fmt.Errorf("failed to write updated device name table: %w", err)
	}

	return nil
}

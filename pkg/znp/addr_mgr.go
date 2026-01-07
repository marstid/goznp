package znp

import (
	"context"
	"fmt"
)

// AddrMgrUserType defines the type flags for address manager entries.
type AddrMgrUserType uint8

// Address Manager entry type flags.
const (
	AddrMgrDefault  AddrMgrUserType = 0x00 // Empty/default entry.
	AddrMgrAssoc    AddrMgrUserType = 0x01 // Associated child device.
	AddrMgrSecurity AddrMgrUserType = 0x02 // Has security key.
	AddrMgrBinding  AddrMgrUserType = 0x04 // Binding table entry.
)

// AddrMgrEntrySize is the size of each Address Manager entry in bytes.
// Legacy OSAL format: 11 bytes (type + nwkAddr + extAddr).
// Extended NV format: 12 bytes (type + padding + nwkAddr + extAddr).
const (
	AddrMgrEntrySize   = 11 // Legacy OSAL format.
	AddrMgrEntrySizeEx = 12 // Extended NV format (with alignment padding).
)

// AddrMgrEntry represents an entry in the Address Manager table.
type AddrMgrEntry struct {
	Type    AddrMgrUserType // Entry type flags.
	NwkAddr uint16          // Network address.
	ExtAddr [8]byte         // IEEE address (all 0xFF = empty).
}

// IsEmpty returns true if the entry is empty (no valid device).
func (e *AddrMgrEntry) IsEmpty() bool {
	// Empty entries have Type=0 or all-FF IEEE address.
	if e.Type == AddrMgrDefault {
		return true
	}
	for _, b := range e.ExtAddr {
		if b != 0xFF {
			return false
		}
	}
	return true
}

// IsAssociated returns true if this is an associated child device.
func (e *AddrMgrEntry) IsAssociated() bool {
	return e.Type&AddrMgrAssoc != 0
}

// HasSecurityKey returns true if the device has a security key.
func (e *AddrMgrEntry) HasSecurityKey() bool {
	return e.Type&AddrMgrSecurity != 0
}

// ParseAddrMgrTable parses raw Address Manager table data into entries.
func ParseAddrMgrTable(data []byte) []AddrMgrEntry {
	if len(data) < AddrMgrEntrySize {
		return nil
	}

	numEntries := len(data) / AddrMgrEntrySize
	entries := make([]AddrMgrEntry, 0, numEntries)

	for i := 0; i < numEntries; i++ {
		offset := i * AddrMgrEntrySize
		entry := AddrMgrEntry{
			Type:    AddrMgrUserType(data[offset]),
			NwkAddr: uint16(data[offset+1]) | uint16(data[offset+2])<<8,
		}
		copy(entry.ExtAddr[:], data[offset+3:offset+11])

		// Only include non-empty entries.
		if !entry.IsEmpty() {
			entries = append(entries, entry)
		}
	}

	return entries
}

// ReadAddrMgrTable reads the Address Manager table from NVRAM.
// Tries Extended NV system first (Z-Stack 3.x), falls back to legacy OSAL.
// Returns all valid (non-empty) device entries.
func (z *ZNP) ReadAddrMgrTable(ctx context.Context) ([]AddrMgrEntry, error) {
	// Try Extended NV system first (Z-Stack 3.x).
	entries, err := z.readAddrMgrEx(ctx)
	if err == nil && len(entries) > 0 {
		return entries, nil
	}

	// Fall back to legacy OSAL NV.
	return z.readAddrMgrOsal(ctx)
}

// readAddrMgrEx reads Address Manager from Extended NV (Z-Stack 3.x).
// Extended NV entries are 12 bytes with alignment padding:
// [type(1), padding(1), nwkAddr(2), extAddr(8)].
func (z *ZNP) readAddrMgrEx(ctx context.Context) ([]AddrMgrEntry, error) {
	rawEntries, err := z.NvReadTable(ctx, NvSysZStack, NvExAddrMgr)
	if err != nil {
		return nil, err
	}

	var entries []AddrMgrEntry
	for _, data := range rawEntries {
		if len(data) < AddrMgrEntrySizeEx {
			continue
		}

		// Extended format: type(1) + padding(1) + nwkAddr(2) + extAddr(8).
		entry := AddrMgrEntry{
			Type:    AddrMgrUserType(data[0]),
			NwkAddr: uint16(data[2]) | uint16(data[3])<<8, // Skip padding byte.
		}
		copy(entry.ExtAddr[:], data[4:12]) // IEEE address starts at offset 4.

		if !entry.IsEmpty() {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// readAddrMgrOsal reads Address Manager from legacy OSAL NV.
func (z *ZNP) readAddrMgrOsal(ctx context.Context) ([]AddrMgrEntry, error) {
	data, err := z.NvReadAll(ctx, NvAddrMgr)
	if err != nil {
		return nil, fmt.Errorf("reading address manager table: %w", err)
	}

	if len(data) == 0 {
		return []AddrMgrEntry{}, nil
	}

	return ParseAddrMgrTable(data), nil
}

// DeleteAddrMgrEntry removes a device from the Address Manager table in NVRAM.
// This is used for force-removing offline/sleepy devices that don't respond to MgmtLeaveReq.
// Returns true if the entry was found and deleted, false if not found.
func (z *ZNP) DeleteAddrMgrEntry(ctx context.Context, ieeeAddr [8]byte) (bool, error) {
	// Try Extended NV first (Z-Stack 3.x).
	deleted, err := z.deleteAddrMgrEntryEx(ctx, ieeeAddr)
	if err == nil && deleted {
		return true, nil
	}

	// Fall back to legacy OSAL NV.
	return z.deleteAddrMgrEntryOsal(ctx, ieeeAddr)
}

// deleteAddrMgrEntryEx deletes from Extended NV (Z-Stack 3.x).
func (z *ZNP) deleteAddrMgrEntryEx(ctx context.Context, ieeeAddr [8]byte) (bool, error) {
	// Read all entries to find the one with matching IEEE address.
	for subID := uint16(0); subID < 1000; subID++ {
		data, err := z.NvReadExAll(ctx, NvSysZStack, NvExAddrMgr, subID)
		if err != nil {
			return false, fmt.Errorf("reading subID %d: %w", subID, err)
		}

		// nil means end of table.
		if data == nil {
			break
		}

		// Check if this entry matches (Extended format: type + padding + nwkAddr + extAddr).
		if len(data) < AddrMgrEntrySizeEx {
			continue
		}

		// IEEE address is at offset 4 (after type, padding, nwkAddr).
		var entryIEEE [8]byte
		copy(entryIEEE[:], data[4:12])

		if entryIEEE == ieeeAddr {
			// Found it! Write an empty entry to delete.
			emptyEntry := make([]byte, AddrMgrEntrySizeEx)
			emptyEntry[0] = 0x00 // Type = empty.
			// Leave nwkAddr as zeros.
			// Set IEEE addr to all 0xFF. (empty marker).
			for i := 4; i < 12; i++ {
				emptyEntry[i] = 0xFF
			}

			if err := z.NvWriteEx(ctx, NvSysZStack, NvExAddrMgr, subID, 0, emptyEntry); err != nil {
				return false, fmt.Errorf("writing empty entry: %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

// deleteAddrMgrEntryOsal deletes from legacy OSAL NV.
func (z *ZNP) deleteAddrMgrEntryOsal(ctx context.Context, ieeeAddr [8]byte) (bool, error) {
	// Read entire table.
	data, err := z.NvReadAll(ctx, NvAddrMgr)
	if err != nil {
		return false, fmt.Errorf("reading address manager table: %w", err)
	}

	if len(data) < AddrMgrEntrySize {
		return false, nil
	}

	numEntries := len(data) / AddrMgrEntrySize

	// Find entry with matching IEEE address.
	for i := 0; i < numEntries; i++ {
		offset := i * AddrMgrEntrySize
		var entryIEEE [8]byte
		copy(entryIEEE[:], data[offset+3:offset+11])

		if entryIEEE == ieeeAddr {
			// Found it! Modify the entry to be empty.
			data[offset] = 0x00 // Type = empty.
			// Set IEEE addr to all 0xFF.
			for j := offset + 3; j < offset+11; j++ {
				data[j] = 0xFF
			}

			// Write back the entire table.
			if err := z.NvWriteAll(ctx, NvAddrMgr, data); err != nil {
				return false, fmt.Errorf("writing address manager table: %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

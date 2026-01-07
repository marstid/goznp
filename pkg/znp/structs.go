package znp

import (
	"encoding/binary"
	"fmt"
)

// NIB represents the Network Information Base stored in NV memory.
// This is a simplified version with the fields we need for backup.
type NIB struct {
	// Offset 0-3: Various config bytes we don't need
	_reserved1 [4]byte //nolint:unused // Required for binary data alignment
	// Offset 4-5: Sequence number
	SequenceNum uint16
	// Offset 6-9: Passive ack timeout and other settings
	_reserved2 [4]byte //nolint:unused // Required for binary data alignment
	// Offset 10: Max broadcast retries
	_reserved3 [1]byte //nolint:unused // Required for binary data alignment
	// Offset 11: Max children
	MaxChildren uint8
	// Offset 12: Max depth
	MaxDepth uint8
	// Offset 13: Max routers
	MaxRouters uint8
	// Offset 14: Dummy neighbor table
	_reserved4 [1]byte //nolint:unused // Required for binary data alignment
	// Offset 15-16: Broadcast delivery time
	_reserved5 [2]byte //nolint:unused // Required for binary data alignment
	// Offset 17: Report constant cost
	_reserved6 [1]byte //nolint:unused // Required for binary data alignment
	// Offset 18: Route discovery retries
	_reserved7 [1]byte //nolint:unused // Required for binary data alignment
	// Offset 19: Reserved
	_reserved8 [1]byte //nolint:unused // Required for binary data alignment
	// Offset 20: Security level
	SecurityLevel uint8
	// Offset 21-28: Extended PAN ID (8 bytes, reversed)
	ExtendedPanID [8]byte
	// Offset 29-32: Channel list (bit mask)
	ChannelList uint32
	// Offset 33-36: Stack profile and other settings
	_reserved9 [4]byte //nolint:unused // Required for binary data alignment
	// Offset 37: Network key switch
	_reserved10 [1]byte //nolint:unused // Required for binary data alignment
	// Offset 38-39: PAN ID
	PanID uint16
	// Offset 40-41: Network manager address
	_reserved11 [2]byte //nolint:unused // Required for binary data alignment
	// Offset 42: Logical channel
	LogicalChannel uint8
	// Offset 43-90: More reserved fields
	_reserved12 [48]byte //nolint:unused // Required for binary data alignment
	// Offset 91: Network update ID
	UpdateID uint8
	// Remaining bytes vary by version
}

// NwkKeyDescriptor represents a network key descriptor.
type NwkKeyDescriptor struct {
	KeySeqNum uint8
	Key       [16]byte
}

// AddressManagerEntry represents an entry in the address manager table.
type AddressManagerEntry struct {
	User    uint8   // Flags: 0x01=Assoc, 0x02=Security, 0x04=Binding
	NwkAddr uint16  // Network (short) address
	ExtAddr [8]byte // IEEE address (reversed byte order)
}

// AddressManagerUser flags
const (
	AddrMgrUserDefault  uint8 = 0x00
	AddrMgrUserAssoc    uint8 = 0x01
	AddrMgrUserSecurity uint8 = 0x02
	AddrMgrUserBinding  uint8 = 0x04
)

// IsValid returns true if this address manager entry is in use.
func (e *AddressManagerEntry) IsValid() bool {
	if e.User == 0 {
		return false
	}
	// Check for all zeros or all 0xFF
	allZero := true
	allFF := true
	for _, b := range e.ExtAddr {
		if b != 0x00 {
			allZero = false
		}
		if b != 0xFF {
			allFF = false
		}
	}
	return !allZero && !allFF
}

// ApsLinkKeyEntry represents an APS link key entry.
type ApsLinkKeyEntry struct {
	Key       [16]byte
	TxFrmCntr uint32
	RxFrmCntr uint32
}

// SecurityManagerEntry maps address index to key NV ID.
type SecurityManagerEntry struct {
	Ami     uint16 // Address manager index
	KeyNvID uint16 // NV ID of the key data
}

// ParseNIB parses a NIB structure from raw NV data.
func ParseNIB(data []byte) (*NIB, error) {
	if len(data) < 92 {
		return nil, fmt.Errorf("NIB data too short: got %d bytes, need at least 92", len(data))
	}
	nib := &NIB{}
	nib.SecurityLevel = data[20]
	copy(nib.ExtendedPanID[:], data[21:29])
	nib.ChannelList = binary.LittleEndian.Uint32(data[29:33])
	nib.PanID = binary.LittleEndian.Uint16(data[38:40])
	nib.LogicalChannel = data[42]
	nib.UpdateID = data[91]
	return nib, nil
}

// ParseNwkKeyDescriptor parses a network key descriptor from raw data.
func ParseNwkKeyDescriptor(data []byte) (*NwkKeyDescriptor, error) {
	if len(data) < 17 {
		return nil, fmt.Errorf("key descriptor data too short: got %d, need 17", len(data))
	}
	kd := &NwkKeyDescriptor{}
	kd.KeySeqNum = data[0]
	copy(kd.Key[:], data[1:17])
	return kd, nil
}

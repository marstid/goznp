package znp

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/unpi"
)

// NV memory operation constants.
const (
	// MaxNvReadSize is the maximum bytes that can be read in one NV read operation.
	MaxNvReadSize = 240
	// MaxNvWriteSize is the maximum bytes that can be written in one NV write operation.
	MaxNvWriteSize = 240
)

// NV status codes.
const (
	NvStatusSuccess      uint8 = 0x00
	NvStatusItemNotFound uint8 = 0x09
	NvStatusBadItemLen   uint8 = 0x0A // Item doesn't exist or wrong length
	NvStatusBadLength    uint8 = 0x0C
)

// NvItemInit initializes/creates an NV item.
// If the item already exists, this is a no-op (returns success).
// itemLen is the total size of the item.
// initData is optional initial data to write (can be nil or shorter than itemLen).
func (z *ZNP) NvItemInit(ctx context.Context, id NvItemID, itemLen uint16, initData []byte) error {
	// Build request: id (uint16), itemLen (uint16), initLen (uint8), initData ([]byte)
	writer := NewBuffaloWriter()
	writer.WriteUint16(uint16(id))
	writer.WriteUint16(itemLen)
	if initData == nil {
		writer.WriteUint8(0)
	} else {
		writer.WriteUint8(uint8(len(initData)))
		writer.WriteBytes(initData)
	}

	// Send CmdSysOsalNvItemInit SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysOsalNvItemInit, writer.Bytes())
	if err != nil {
		return fmt.Errorf("nv item init request failed for item %d: %w", id, err)
	}

	// Parse response: status (uint8)
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return fmt.Errorf("failed to read status: %w", err)
	}

	// Status 0x00 = success, 0x09 = already exists (also success for our purposes)
	if status != NvStatusSuccess && status != NvStatusItemNotFound {
		return fmt.Errorf("nv item init failed with status 0x%02X", status)
	}

	return nil
}

// NvLength returns the length of an NV item.
// Returns 0 if the item doesn't exist.
func (z *ZNP) NvLength(ctx context.Context, id NvItemID) (uint16, error) {
	// Build request: id (uint16)
	writer := NewBuffaloWriter()
	writer.WriteUint16(uint16(id))

	// Send CmdSysOsalNvLength SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysOsalNvLength, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("nv length request failed for item %d: %w", id, err)
	}

	// Parse response: len (uint16)
	buf := NewBuffalo(resp.Data)
	length, err := buf.ReadUint16()
	if err != nil {
		return 0, fmt.Errorf("failed to parse nv length response: %w", err)
	}

	return length, nil
}

// NvRead reads data from an NV item at the specified offset.
// For items larger than MaxNvReadSize, use NvReadAll.
// Returns an empty slice if the item doesn't exist (status == NvStatusItemNotFound).
func (z *ZNP) NvRead(ctx context.Context, id NvItemID, offset uint8) ([]byte, error) {
	// Build request: id (uint16), offset (uint8)
	writer := NewBuffaloWriter()
	writer.WriteUint16(uint16(id))
	writer.WriteUint8(offset)

	// Send CmdSysOsalNvRead SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysOsalNvRead, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("nv read request failed for item %d at offset %d: %w", id, offset, err)
	}

	// Parse response: status (uint8), len (uint8), value ([]byte)
	buf := NewBuffalo(resp.Data)

	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	length, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}

	// Handle item not found gracefully
	if status == NvStatusItemNotFound {
		return []byte{}, nil
	}

	// Check for other errors
	if status != NvStatusSuccess {
		return nil, fmt.Errorf("nv read failed with status 0x%02X", status)
	}

	// Read value bytes
	value, err := buf.ReadBytes(int(length))
	if err != nil {
		return nil, fmt.Errorf("failed to read value: %w", err)
	}

	return value, nil
}

// NvReadAll reads an entire NV item, handling chunking for large items.
// Returns an empty slice if the item doesn't exist.
func (z *ZNP) NvReadAll(ctx context.Context, id NvItemID) ([]byte, error) {
	// First call NvLength to get total length
	totalLength, err := z.NvLength(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get nv item length: %w", err)
	}

	// If length == 0, item doesn't exist
	if totalLength == 0 {
		return []byte{}, nil
	}

	// Allocate result buffer
	result := make([]byte, totalLength)
	offset := uint8(0)
	bytesRead := 0

	// Loop reading MaxNvReadSize chunks
	for bytesRead < int(totalLength) {
		// Calculate remaining bytes to read
		remaining := int(totalLength) - bytesRead
		chunkSize := remaining
		if chunkSize > MaxNvReadSize {
			chunkSize = MaxNvReadSize
		}

		// Call NvRead with current offset
		chunk, err := z.NvRead(ctx, id, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk at offset %d: %w", offset, err)
		}

		// Check if we got an empty chunk (shouldn't happen mid-read)
		if len(chunk) == 0 && bytesRead < int(totalLength) {
			return nil, fmt.Errorf("unexpected empty chunk at offset %d", offset)
		}

		// Copy chunk to result buffer
		copy(result[bytesRead:], chunk)
		bytesRead += len(chunk)
		offset += uint8(len(chunk))
	}

	return result, nil
}

// NvWrite writes data to an NV item at the specified offset.
// For items larger than MaxNvWriteSize, use NvWriteAll.
func (z *ZNP) NvWrite(ctx context.Context, id NvItemID, offset uint8, data []byte) error {
	// Validate data length
	if len(data) > MaxNvWriteSize {
		return fmt.Errorf("data length %d exceeds max write size %d", len(data), MaxNvWriteSize)
	}

	// Build request: id (uint16), offset (uint8), len (uint8), value ([]byte)
	writer := NewBuffaloWriter()
	writer.WriteUint16(uint16(id))
	writer.WriteUint8(offset)
	writer.WriteUint8(uint8(len(data)))
	writer.WriteBytes(data)

	// Send CmdSysOsalNvWrite SREQ
	resp, err := z.Request(ctx, unpi.SYS, CmdSysOsalNvWrite, writer.Bytes())
	if err != nil {
		return fmt.Errorf("nv write request failed for item %d at offset %d: %w", id, offset, err)
	}

	// Parse response: status (uint8)
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return fmt.Errorf("failed to read status: %w", err)
	}

	// Check status
	if status != NvStatusSuccess {
		return fmt.Errorf("nv write failed with status 0x%02X", status)
	}

	return nil
}

// NvWriteAll writes an entire NV item, handling chunking for large items.
func (z *ZNP) NvWriteAll(ctx context.Context, id NvItemID, data []byte) error {
	offset := uint8(0)
	bytesWritten := 0
	totalLength := len(data)

	// Loop writing MaxNvWriteSize chunks
	for bytesWritten < totalLength {
		// Calculate chunk size (min of remaining bytes, MaxNvWriteSize)
		remaining := totalLength - bytesWritten
		chunkSize := remaining
		if chunkSize > MaxNvWriteSize {
			chunkSize = MaxNvWriteSize
		}

		// Get chunk to write
		chunk := data[bytesWritten : bytesWritten+chunkSize]

		// Call NvWrite with current offset
		if err := z.NvWrite(ctx, id, offset, chunk); err != nil {
			return fmt.Errorf("failed to write chunk at offset %d: %w", offset, err)
		}

		bytesWritten += chunkSize
		offset += uint8(chunkSize)
	}

	return nil
}

// Extended NV System (Z-Stack 3.x)
// These commands use a hierarchical SysId/ItemId/SubId structure.

// Extended NV System IDs.
const (
	NvSysZStack uint8 = 0x01 // Z-Stack system
)

// Extended NV Item IDs for Z-Stack system.
const (
	NvExAddrMgr    uint16 = 0x0001 // Address Manager table
	NvExTclkTable  uint16 = 0x0004 // Trust Center Link Key table
	NvExApsKeyData uint16 = 0x0006 // APS Key Data table
)

// NvLengthEx returns the length of an Extended NV item.
// Returns 0 if the item doesn't exist.
func (z *ZNP) NvLengthEx(ctx context.Context, sysId uint8, itemId, subId uint16) (uint8, error) {
	writer := NewBuffaloWriter()
	writer.WriteUint8(sysId)
	writer.WriteUint16(itemId)
	writer.WriteUint16(subId)

	resp, err := z.Request(ctx, unpi.SYS, CmdSysNvLength, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("nv length ex request failed: %w", err)
	}

	buf := NewBuffalo(resp.Data)
	length, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read length: %w", err)
	}

	return length, nil
}

// NvReadEx reads data from an Extended NV item.
// Returns an empty slice if the item doesn't exist.
func (z *ZNP) NvReadEx(ctx context.Context, sysId uint8, itemId, subId, offset uint16, length uint8) ([]byte, error) {
	writer := NewBuffaloWriter()
	writer.WriteUint8(sysId)
	writer.WriteUint16(itemId)
	writer.WriteUint16(subId)
	writer.WriteUint16(offset)
	writer.WriteUint8(length)

	resp, err := z.Request(ctx, unpi.SYS, CmdSysNvRead, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("nv read ex request failed: %w", err)
	}

	buf := NewBuffalo(resp.Data)

	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	readLen, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}

	// Handle item not found (0x09) or bad item length (0x0A - item doesn't exist)
	if status == NvStatusItemNotFound || status == NvStatusBadItemLen {
		return nil, nil
	}

	if status != NvStatusSuccess {
		return nil, fmt.Errorf("nv read ex failed with status 0x%02X", status)
	}

	value, err := buf.ReadBytes(int(readLen))
	if err != nil {
		return nil, fmt.Errorf("failed to read value: %w", err)
	}

	return value, nil
}

// NvReadExAll reads an entire Extended NV item at a specific subId.
func (z *ZNP) NvReadExAll(ctx context.Context, sysId uint8, itemId, subId uint16) ([]byte, error) {
	length, err := z.NvLengthEx(ctx, sysId, itemId, subId)
	if err != nil {
		return nil, err
	}

	if length == 0 {
		return nil, nil
	}

	return z.NvReadEx(ctx, sysId, itemId, subId, 0, length)
}

// NvReadTable reads all entries from an Extended NV table by iterating through SubIds.
// Returns entries until a SubId returns "not found".
func (z *ZNP) NvReadTable(ctx context.Context, sysId uint8, itemId uint16) ([][]byte, error) {
	var entries [][]byte

	for subId := uint16(0); subId < 1000; subId++ { // Safety limit
		data, err := z.NvReadExAll(ctx, sysId, itemId, subId)
		if err != nil {
			return nil, fmt.Errorf("reading subId %d: %w", subId, err)
		}

		// nil means not found - end of table
		if data == nil {
			break
		}

		entries = append(entries, data)
	}

	return entries, nil
}

// NvWriteEx writes data to an Extended NV item.
func (z *ZNP) NvWriteEx(ctx context.Context, sysId uint8, itemId, subId, offset uint16, data []byte) error {
	writer := NewBuffaloWriter()
	writer.WriteUint8(sysId)
	writer.WriteUint16(itemId)
	writer.WriteUint16(subId)
	writer.WriteUint16(offset)
	writer.WriteUint8(uint8(len(data)))
	writer.WriteBytes(data)

	resp, err := z.Request(ctx, unpi.SYS, CmdSysNvWrite, writer.Bytes())
	if err != nil {
		return fmt.Errorf("nv write ex request failed: %w", err)
	}

	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return fmt.Errorf("failed to read status: %w", err)
	}

	if status != NvStatusSuccess {
		return fmt.Errorf("nv write ex failed with status 0x%02X", status)
	}

	return nil
}

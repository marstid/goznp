package znp

import (
	"encoding/binary"
	"fmt"
)

// Sentinel errors for Buffalo operations.
var (
	// ErrBufferOverrun indicates an attempt to read beyond available data.
	ErrBufferOverrun = fmt.Errorf("buffer overrun: not enough data to read")
)

// Buffalo handles binary encoding/decoding for ZNP parameters.
// It provides methods for reading from and writing to byte buffers,
// using little-endian byte order for multi-byte values.
type Buffalo struct {
	data   []byte
	offset int
}

// NewBuffalo creates a new Buffalo reader from existing data.
// The returned Buffalo can be used to parse binary data.
func NewBuffalo(data []byte) *Buffalo {
	return &Buffalo{
		data:   data,
		offset: 0,
	}
}

// NewBuffaloWriter creates an empty Buffalo writer.
// The returned Buffalo can be used to build binary data.
func NewBuffaloWriter() *Buffalo {
	return &Buffalo{
		data:   make([]byte, 0),
		offset: 0,
	}
}

// ReadUint8 reads a single unsigned 8-bit integer.
func (b *Buffalo) ReadUint8() (uint8, error) {
	if b.offset+1 > len(b.data) {
		return 0, fmt.Errorf("%w: need 1 byte, have %d", ErrBufferOverrun, len(b.data)-b.offset)
	}
	val := b.data[b.offset]
	b.offset++
	return val, nil
}

// ReadUint16 reads an unsigned 16-bit integer in little-endian format.
func (b *Buffalo) ReadUint16() (uint16, error) {
	if b.offset+2 > len(b.data) {
		return 0, fmt.Errorf("%w: need 2 bytes, have %d", ErrBufferOverrun, len(b.data)-b.offset)
	}
	val := binary.LittleEndian.Uint16(b.data[b.offset:])
	b.offset += 2
	return val, nil
}

// ReadUint32 reads an unsigned 32-bit integer in little-endian format.
func (b *Buffalo) ReadUint32() (uint32, error) {
	if b.offset+4 > len(b.data) {
		return 0, fmt.Errorf("%w: need 4 bytes, have %d", ErrBufferOverrun, len(b.data)-b.offset)
	}
	val := binary.LittleEndian.Uint32(b.data[b.offset:])
	b.offset += 4
	return val, nil
}

// ReadInt8 reads a signed 8-bit integer.
func (b *Buffalo) ReadInt8() (int8, error) {
	if b.offset+1 > len(b.data) {
		return 0, fmt.Errorf("%w: need 1 byte, have %d", ErrBufferOverrun, len(b.data)-b.offset)
	}
	val := int8(b.data[b.offset])
	b.offset++
	return val, nil
}

// ReadBytes reads n bytes from the buffer.
func (b *Buffalo) ReadBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("invalid read size: %d", n)
	}
	if b.offset+n > len(b.data) {
		return nil, fmt.Errorf("%w: need %d bytes, have %d", ErrBufferOverrun, n, len(b.data)-b.offset)
	}
	val := make([]byte, n)
	copy(val, b.data[b.offset:b.offset+n])
	b.offset += n
	return val, nil
}

// ReadIEEEAddr reads an 8-byte IEEE address.
func (b *Buffalo) ReadIEEEAddr() ([8]byte, error) {
	var addr [8]byte
	if b.offset+8 > len(b.data) {
		return addr, fmt.Errorf("%w: need 8 bytes, have %d", ErrBufferOverrun, len(b.data)-b.offset)
	}
	copy(addr[:], b.data[b.offset:b.offset+8])
	b.offset += 8
	return addr, nil
}

// Remaining returns the number of bytes left to read.
func (b *Buffalo) Remaining() int {
	remaining := len(b.data) - b.offset
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsEOF returns true if there are no more bytes to read.
func (b *Buffalo) IsEOF() bool {
	return b.offset >= len(b.data)
}

// WriteUint8 writes a single unsigned 8-bit integer.
func (b *Buffalo) WriteUint8(v uint8) {
	b.data = append(b.data, v)
}

// WriteUint16 writes an unsigned 16-bit integer in little-endian format.
func (b *Buffalo) WriteUint16(v uint16) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, v)
	b.data = append(b.data, buf...)
}

// WriteUint32 writes an unsigned 32-bit integer in little-endian format.
func (b *Buffalo) WriteUint32(v uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)
	b.data = append(b.data, buf...)
}

// WriteInt8 writes a signed 8-bit integer.
func (b *Buffalo) WriteInt8(v int8) {
	b.data = append(b.data, uint8(v))
}

// WriteBytes writes a byte slice to the buffer.
func (b *Buffalo) WriteBytes(bytes []byte) {
	b.data = append(b.data, bytes...)
}

// WriteIEEEAddr writes an 8-byte IEEE address.
func (b *Buffalo) WriteIEEEAddr(addr [8]byte) {
	b.data = append(b.data, addr[:]...)
}

// Bytes returns the written bytes.
func (b *Buffalo) Bytes() []byte {
	return b.data
}

// Len returns the number of bytes written.
func (b *Buffalo) Len() int {
	return len(b.data)
}

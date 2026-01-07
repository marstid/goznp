package znp

import (
	"bytes"
	"errors"
	"testing"
)

func TestNewBuffalo(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	buf := NewBuffalo(data)

	if buf == nil {
		t.Fatal("NewBuffalo returned nil")
	}
	if buf.offset != 0 {
		t.Errorf("expected offset 0, got %d", buf.offset)
	}
	if !bytes.Equal(buf.data, data) {
		t.Errorf("expected data %v, got %v", data, buf.data)
	}
}

func TestNewBuffaloWriter(t *testing.T) {
	buf := NewBuffaloWriter()

	if buf == nil {
		t.Fatal("NewBuffaloWriter returned nil")
	}
	if buf.offset != 0 {
		t.Errorf("expected offset 0, got %d", buf.offset)
	}
	if len(buf.data) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(buf.data))
	}
}

func TestReadUint8(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint8
		wantErr bool
	}{
		{
			name: "read single byte",
			data: []byte{0x42},
			want: 0x42,
		},
		{
			name: "read from middle",
			data: []byte{0x01, 0x02, 0x03},
			want: 0x01,
		},
		{
			name:    "empty buffer",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadUint8()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint8() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadUint8() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

func TestReadUint16(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint16
		wantErr bool
	}{
		{
			name: "read little-endian",
			data: []byte{0x34, 0x12}, // 0x1234 in little-endian
			want: 0x1234,
		},
		{
			name: "read zeros",
			data: []byte{0x00, 0x00},
			want: 0x0000,
		},
		{
			name:    "insufficient data",
			data:    []byte{0x01},
			wantErr: true,
		},
		{
			name:    "empty buffer",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadUint16()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint16() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadUint16() = 0x%04X, want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestReadUint32(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint32
		wantErr bool
	}{
		{
			name: "read little-endian",
			data: []byte{0x78, 0x56, 0x34, 0x12}, // 0x12345678 in little-endian
			want: 0x12345678,
		},
		{
			name: "read zeros",
			data: []byte{0x00, 0x00, 0x00, 0x00},
			want: 0x00000000,
		},
		{
			name:    "insufficient data",
			data:    []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:    "empty buffer",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadUint32()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadUint32() = 0x%08X, want 0x%08X", got, tt.want)
			}
		})
	}
}

func TestReadInt8(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    int8
		wantErr bool
	}{
		{
			name: "read positive",
			data: []byte{0x42},
			want: 0x42,
		},
		{
			name: "read negative",
			data: []byte{0xFF}, // -1 in two's complement
			want: -1,
		},
		{
			name:    "empty buffer",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadInt8()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadInt8() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadInt8() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestReadBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		n       int
		want    []byte
		wantErr bool
	}{
		{
			name: "read multiple bytes",
			data: []byte{0x01, 0x02, 0x03, 0x04},
			n:    3,
			want: []byte{0x01, 0x02, 0x03},
		},
		{
			name: "read all bytes",
			data: []byte{0xAA, 0xBB},
			n:    2,
			want: []byte{0xAA, 0xBB},
		},
		{
			name: "read zero bytes",
			data: []byte{0x01, 0x02},
			n:    0,
			want: []byte{},
		},
		{
			name:    "read too many bytes",
			data:    []byte{0x01, 0x02},
			n:       5,
			wantErr: true,
		},
		{
			name:    "negative size",
			data:    []byte{0x01},
			n:       -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadBytes(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !bytes.Equal(got, tt.want) {
				t.Errorf("ReadBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadIEEEAddr(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    [8]byte
		wantErr bool
	}{
		{
			name: "read IEEE address",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			want: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
		{
			name:    "insufficient data",
			data:    []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:    "empty buffer",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadIEEEAddr()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadIEEEAddr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadIEEEAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemaining(t *testing.T) {
	buf := NewBuffalo([]byte{0x01, 0x02, 0x03, 0x04, 0x05})

	if buf.Remaining() != 5 {
		t.Errorf("Remaining() = %d, want 5", buf.Remaining())
	}

	//nolint:errcheck // Test intentionally consumes data
	_, _ = buf.ReadUint8()
	if buf.Remaining() != 4 {
		t.Errorf("Remaining() = %d, want 4", buf.Remaining())
	}

	//nolint:errcheck // Test intentionally consumes data
	_, _ = buf.ReadUint16()
	if buf.Remaining() != 2 {
		t.Errorf("Remaining() = %d, want 2", buf.Remaining())
	}

	//nolint:errcheck // Test intentionally consumes data
	_, _ = buf.ReadUint16()
	if buf.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", buf.Remaining())
	}
}

func TestIsEOF(t *testing.T) {
	buf := NewBuffalo([]byte{0x01, 0x02})

	if buf.IsEOF() {
		t.Error("IsEOF() = true, want false at start")
	}

	//nolint:errcheck // Test intentionally consumes data
	_, _ = buf.ReadUint8()
	if buf.IsEOF() {
		t.Error("IsEOF() = true, want false after reading 1 byte")
	}

	//nolint:errcheck // Test intentionally consumes data
	_, _ = buf.ReadUint8()
	if !buf.IsEOF() {
		t.Error("IsEOF() = false, want true at end")
	}
}

func TestWriteUint8(t *testing.T) {
	tests := []struct {
		name   string
		values []uint8
		want   []byte
	}{
		{
			name:   "single byte",
			values: []uint8{0x42},
			want:   []byte{0x42},
		},
		{
			name:   "multiple bytes",
			values: []uint8{0x42, 0xFF},
			want:   []byte{0x42, 0xFF},
		},
		{
			name:   "zero value",
			values: []uint8{0x00},
			want:   []byte{0x00},
		},
		{
			name:   "max value",
			values: []uint8{0xFF},
			want:   []byte{0xFF},
		},
		{
			name:   "sequential values",
			values: []uint8{0x00, 0x01, 0x02, 0xFE, 0xFF},
			want:   []byte{0x00, 0x01, 0x02, 0xFE, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffaloWriter()
			for _, v := range tt.values {
				buf.WriteUint8(v)
			}
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("WriteUint8() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestWriteUint16(t *testing.T) {
	tests := []struct {
		name   string
		values []uint16
		want   []byte
	}{
		{
			name:   "single value",
			values: []uint16{0x1234},
			want:   []byte{0x34, 0x12}, // little-endian
		},
		{
			name:   "zero value",
			values: []uint16{0x0000},
			want:   []byte{0x00, 0x00},
		},
		{
			name:   "max value",
			values: []uint16{0xFFFF},
			want:   []byte{0xFF, 0xFF},
		},
		{
			name:   "multiple values",
			values: []uint16{0x1234, 0xABCD},
			want:   []byte{0x34, 0x12, 0xCD, 0xAB},
		},
		{
			name:   "low byte set",
			values: []uint16{0x00FF},
			want:   []byte{0xFF, 0x00},
		},
		{
			name:   "high byte set",
			values: []uint16{0xFF00},
			want:   []byte{0x00, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffaloWriter()
			for _, v := range tt.values {
				buf.WriteUint16(v)
			}
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("WriteUint16() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestWriteUint32(t *testing.T) {
	tests := []struct {
		name   string
		values []uint32
		want   []byte
	}{
		{
			name:   "single value",
			values: []uint32{0x12345678},
			want:   []byte{0x78, 0x56, 0x34, 0x12}, // little-endian
		},
		{
			name:   "zero value",
			values: []uint32{0x00000000},
			want:   []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:   "max value",
			values: []uint32{0xFFFFFFFF},
			want:   []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name:   "multiple values",
			values: []uint32{0x12345678, 0xABCDEF01},
			want:   []byte{0x78, 0x56, 0x34, 0x12, 0x01, 0xEF, 0xCD, 0xAB},
		},
		{
			name:   "low byte set",
			values: []uint32{0x000000FF},
			want:   []byte{0xFF, 0x00, 0x00, 0x00},
		},
		{
			name:   "high byte set",
			values: []uint32{0xFF000000},
			want:   []byte{0x00, 0x00, 0x00, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffaloWriter()
			for _, v := range tt.values {
				buf.WriteUint32(v)
			}
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("WriteUint32() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestWriteInt8(t *testing.T) {
	buf := NewBuffaloWriter()
	buf.WriteInt8(42)
	buf.WriteInt8(-1)

	want := []byte{0x2A, 0xFF}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("WriteInt8() = %v, want %v", buf.Bytes(), want)
	}
}

func TestWriteBytes(t *testing.T) {
	buf := NewBuffaloWriter()
	buf.WriteBytes([]byte{0x01, 0x02, 0x03})
	buf.WriteBytes([]byte{0xAA, 0xBB})

	want := []byte{0x01, 0x02, 0x03, 0xAA, 0xBB}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("WriteBytes() = %v, want %v", buf.Bytes(), want)
	}
}

func TestWriteIEEEAddr(t *testing.T) {
	tests := []struct {
		name string
		addr [8]byte
		want []byte
	}{
		{
			name: "sequential bytes",
			addr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			want: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
		{
			name: "all zeros",
			addr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "all ones",
			addr: [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			want: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "mixed values",
			addr: [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
			want: []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffaloWriter()
			buf.WriteIEEEAddr(tt.addr)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("WriteIEEEAddr() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestLen(t *testing.T) {
	buf := NewBuffaloWriter()

	if buf.Len() != 0 {
		t.Errorf("Len() = %d, want 0", buf.Len())
	}

	buf.WriteUint8(0x42)
	if buf.Len() != 1 {
		t.Errorf("Len() = %d, want 1", buf.Len())
	}

	buf.WriteUint16(0x1234)
	if buf.Len() != 3 {
		t.Errorf("Len() = %d, want 3", buf.Len())
	}

	buf.WriteUint32(0x12345678)
	if buf.Len() != 7 {
		t.Errorf("Len() = %d, want 7", buf.Len())
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that writing and reading values produces the same results
	writer := NewBuffaloWriter()
	writer.WriteUint8(0x42)
	writer.WriteUint16(0x1234)
	writer.WriteUint32(0x12345678)
	writer.WriteInt8(-42)
	writer.WriteBytes([]byte{0xAA, 0xBB, 0xCC})
	addr := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	writer.WriteIEEEAddr(addr)

	reader := NewBuffalo(writer.Bytes())

	u8, err := reader.ReadUint8()
	if err != nil || u8 != 0x42 {
		t.Errorf("ReadUint8() = 0x%02X, %v; want 0x42, nil", u8, err)
	}

	u16, err := reader.ReadUint16()
	if err != nil || u16 != 0x1234 {
		t.Errorf("ReadUint16() = 0x%04X, %v; want 0x1234, nil", u16, err)
	}

	u32, err := reader.ReadUint32()
	if err != nil || u32 != 0x12345678 {
		t.Errorf("ReadUint32() = 0x%08X, %v; want 0x12345678, nil", u32, err)
	}

	i8, err := reader.ReadInt8()
	if err != nil || i8 != -42 {
		t.Errorf("ReadInt8() = %d, %v; want -42, nil", i8, err)
	}

	b, err := reader.ReadBytes(3)
	if err != nil || !bytes.Equal(b, []byte{0xAA, 0xBB, 0xCC}) {
		t.Errorf("ReadBytes() = %v, %v; want [0xAA 0xBB 0xCC], nil", b, err)
	}

	readAddr, err := reader.ReadIEEEAddr()
	if err != nil || readAddr != addr {
		t.Errorf("ReadIEEEAddr() = %v, %v; want %v, nil", readAddr, err, addr)
	}

	if !reader.IsEOF() {
		t.Error("IsEOF() = false, want true after reading all data")
	}
}

func TestBufferOverrunErrors(t *testing.T) {
	buf := NewBuffalo([]byte{0x01})

	// Try to read beyond buffer
	_, err := buf.ReadUint16()
	if !errors.Is(err, ErrBufferOverrun) {
		t.Errorf("ReadUint16() error = %v, want ErrBufferOverrun", err)
	}

	// Reset buffer
	buf = NewBuffalo([]byte{0x01, 0x02})
	//nolint:errcheck // Test intentionally consumes data
	_, _ = buf.ReadUint16() // consume all data

	_, err = buf.ReadUint8()
	if !errors.Is(err, ErrBufferOverrun) {
		t.Errorf("ReadUint8() after exhausting buffer error = %v, want ErrBufferOverrun", err)
	}
}

func TestSequentialReads(t *testing.T) {
	// Test that sequential reads advance the offset correctly
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	buf := NewBuffalo(data)

	//nolint:errcheck // Test code
	v1, _ := buf.ReadUint8()
	if v1 != 0x01 {
		t.Errorf("First ReadUint8() = 0x%02X, want 0x01", v1)
	}

	//nolint:errcheck // Test code
	v2, _ := buf.ReadUint16()
	if v2 != 0x0302 { // 0x02, 0x03 in little-endian
		t.Errorf("ReadUint16() = 0x%04X, want 0x0302", v2)
	}

	//nolint:errcheck // Test code
	v3, _ := buf.ReadUint8()
	if v3 != 0x04 {
		t.Errorf("Second ReadUint8() = 0x%02X, want 0x04", v3)
	}

	remaining := buf.Remaining()
	if remaining != 4 {
		t.Errorf("Remaining() = %d, want 4", remaining)
	}
}

func TestReadBoundaryErrors(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		operation func(*Buffalo) error
		wantErr   bool
	}{
		{
			name: "ReadUint8 at boundary",
			data: []byte{0x01},
			operation: func(b *Buffalo) error {
				_, err := b.ReadUint8()
				return err
			},
			wantErr: false,
		},
		{
			name: "ReadUint8 past boundary",
			data: []byte{},
			operation: func(b *Buffalo) error {
				_, err := b.ReadUint8()
				return err
			},
			wantErr: true,
		},
		{
			name: "ReadUint16 at boundary",
			data: []byte{0x01, 0x02},
			operation: func(b *Buffalo) error {
				_, err := b.ReadUint16()
				return err
			},
			wantErr: false,
		},
		{
			name: "ReadUint16 one byte short",
			data: []byte{0x01},
			operation: func(b *Buffalo) error {
				_, err := b.ReadUint16()
				return err
			},
			wantErr: true,
		},
		{
			name: "ReadUint32 at boundary",
			data: []byte{0x01, 0x02, 0x03, 0x04},
			operation: func(b *Buffalo) error {
				_, err := b.ReadUint32()
				return err
			},
			wantErr: false,
		},
		{
			name: "ReadUint32 three bytes short",
			data: []byte{0x01},
			operation: func(b *Buffalo) error {
				_, err := b.ReadUint32()
				return err
			},
			wantErr: true,
		},
		{
			name: "ReadIEEEAddr at boundary",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			operation: func(b *Buffalo) error {
				_, err := b.ReadIEEEAddr()
				return err
			},
			wantErr: false,
		},
		{
			name: "ReadIEEEAddr seven bytes short",
			data: []byte{0x01},
			operation: func(b *Buffalo) error {
				_, err := b.ReadIEEEAddr()
				return err
			},
			wantErr: true,
		},
		{
			name: "ReadBytes at boundary",
			data: []byte{0x01, 0x02, 0x03},
			operation: func(b *Buffalo) error {
				_, err := b.ReadBytes(3)
				return err
			},
			wantErr: false,
		},
		{
			name: "ReadBytes past boundary",
			data: []byte{0x01, 0x02},
			operation: func(b *Buffalo) error {
				_, err := b.ReadBytes(3)
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			err := tt.operation(buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("operation error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrBufferOverrun) {
				t.Errorf("expected ErrBufferOverrun, got %v", err)
			}
		})
	}
}

func TestMultipleSequentialReads(t *testing.T) {
	// Test reading multiple values in sequence
	buf := NewBuffaloWriter()
	buf.WriteUint8(0x01)
	buf.WriteUint16(0x0302)
	buf.WriteUint32(0x07060504)
	buf.WriteIEEEAddr([8]byte{0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F})

	reader := NewBuffalo(buf.Bytes())

	v1, err := reader.ReadUint8()
	if err != nil {
		t.Fatalf("ReadUint8() error = %v", err)
	}
	if v1 != 0x01 {
		t.Errorf("ReadUint8() = 0x%02X, want 0x01", v1)
	}

	v2, err := reader.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16() error = %v", err)
	}
	if v2 != 0x0302 {
		t.Errorf("ReadUint16() = 0x%04X, want 0x0302", v2)
	}

	v3, err := reader.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32() error = %v", err)
	}
	if v3 != 0x07060504 {
		t.Errorf("ReadUint32() = 0x%08X, want 0x07060504", v3)
	}

	addr, err := reader.ReadIEEEAddr()
	if err != nil {
		t.Fatalf("ReadIEEEAddr() error = %v", err)
	}
	wantAddr := [8]byte{0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	if addr != wantAddr {
		t.Errorf("ReadIEEEAddr() = %v, want %v", addr, wantAddr)
	}

	if !reader.IsEOF() {
		t.Error("IsEOF() = false, want true after reading all data")
	}
}

func TestPartialReadErrors(t *testing.T) {
	// Test that partial reads don't corrupt the buffer state
	buf := NewBuffalo([]byte{0x01, 0x02})

	// First read succeeds
	v1, err := buf.ReadUint8()
	if err != nil {
		t.Fatalf("First ReadUint8() error = %v", err)
	}
	if v1 != 0x01 {
		t.Errorf("First ReadUint8() = 0x%02X, want 0x01", v1)
	}

	// Second read succeeds
	v2, err := buf.ReadUint8()
	if err != nil {
		t.Fatalf("Second ReadUint8() error = %v", err)
	}
	if v2 != 0x02 {
		t.Errorf("Second ReadUint8() = 0x%02X, want 0x02", v2)
	}

	// Third read should fail
	_, err = buf.ReadUint8()
	if !errors.Is(err, ErrBufferOverrun) {
		t.Errorf("Third ReadUint8() error = %v, want ErrBufferOverrun", err)
	}

	// Verify buffer state
	if !buf.IsEOF() {
		t.Error("IsEOF() = false, want true after exhausting buffer")
	}
	if buf.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", buf.Remaining())
	}
}

func TestMixedWriteOperations(t *testing.T) {
	// Test mixing different write operations
	buf := NewBuffaloWriter()

	buf.WriteUint8(0x01)
	buf.WriteUint16(0x0302)
	buf.WriteBytes([]byte{0x04, 0x05})
	buf.WriteUint32(0x09080706)
	buf.WriteInt8(-1)
	buf.WriteIEEEAddr([8]byte{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11})

	want := []byte{
		0x01,       // uint8
		0x02, 0x03, // uint16 (little-endian)
		0x04, 0x05, // bytes
		0x06, 0x07, 0x08, 0x09, // uint32 (little-endian)
		0xFF,                                           // int8 (-1)
		0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, // IEEE addr
	}

	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("MixedWriteOperations() = %v, want %v", buf.Bytes(), want)
	}

	// Verify length
	if buf.Len() != len(want) {
		t.Errorf("Len() = %d, want %d", buf.Len(), len(want))
	}
}

func TestReadUint8EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		reads   int
		wantErr bool
	}{
		{
			name:    "read after EOF",
			data:    []byte{0x01},
			reads:   2,
			wantErr: true,
		},
		{
			name:    "multiple reads from multi-byte buffer",
			data:    []byte{0x01, 0x02, 0x03},
			reads:   3,
			wantErr: false,
		},
		{
			name:    "exceed buffer by multiple bytes",
			data:    []byte{0x01},
			reads:   5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			var err error
			for i := 0; i < tt.reads; i++ {
				_, err = buf.ReadUint8()
				if err != nil {
					break
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint8() series error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadUint16EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint16
	}{
		{
			name: "max value",
			data: []byte{0xFF, 0xFF},
			want: 0xFFFF,
		},
		{
			name: "min value",
			data: []byte{0x00, 0x00},
			want: 0x0000,
		},
		{
			name: "one value",
			data: []byte{0x01, 0x00},
			want: 0x0001,
		},
		{
			name: "alternating bits",
			data: []byte{0xAA, 0x55},
			want: 0x55AA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadUint16()
			if err != nil {
				t.Fatalf("ReadUint16() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ReadUint16() = 0x%04X, want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestReadUint32EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint32
	}{
		{
			name: "max value",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF},
			want: 0xFFFFFFFF,
		},
		{
			name: "min value",
			data: []byte{0x00, 0x00, 0x00, 0x00},
			want: 0x00000000,
		},
		{
			name: "one value",
			data: []byte{0x01, 0x00, 0x00, 0x00},
			want: 0x00000001,
		},
		{
			name: "alternating bits",
			data: []byte{0xAA, 0x55, 0xAA, 0x55},
			want: 0x55AA55AA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadUint32()
			if err != nil {
				t.Fatalf("ReadUint32() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ReadUint32() = 0x%08X, want 0x%08X", got, tt.want)
			}
		})
	}
}

func TestReadIEEEAddrEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want [8]byte
	}{
		{
			name: "all zeros broadcast address",
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "all ones broadcast address",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			want: [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "alternating pattern",
			data: []byte{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55},
			want: [8]byte{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55},
		},
		{
			name: "real-world MAC example",
			data: []byte{0x00, 0x12, 0x4B, 0x00, 0x1A, 0x2B, 0x3C, 0x4D},
			want: [8]byte{0x00, 0x12, 0x4B, 0x00, 0x1A, 0x2B, 0x3C, 0x4D},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffalo(tt.data)
			got, err := buf.ReadIEEEAddr()
			if err != nil {
				t.Fatalf("ReadIEEEAddr() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ReadIEEEAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

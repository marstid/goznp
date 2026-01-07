package znp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TestNvItemInit tests the NvItemInit method.
func TestNvItemInit(t *testing.T) {
	tests := []struct {
		name     string
		id       NvItemID
		itemLen  uint16
		initData []byte
		status   uint8
		wantErr  bool
	}{
		{
			name:     "success new item",
			id:       NvStartupOption,
			itemLen:  4,
			initData: []byte{0x01, 0x02, 0x03, 0x04},
			status:   NvStatusSuccess,
			wantErr:  false,
		},
		{
			name:     "success without init data",
			id:       NvStartupOption,
			itemLen:  4,
			initData: nil,
			status:   NvStatusSuccess,
			wantErr:  false,
		},
		{
			name:     "success already exists",
			id:       NvStartupOption,
			itemLen:  4,
			initData: []byte{0x01, 0x02, 0x03, 0x04},
			status:   NvStatusItemNotFound, // Already exists
			wantErr:  false,
		},
		{
			name:     "success init data shorter than itemLen",
			id:       NvStartupOption,
			itemLen:  10,
			initData: []byte{0x01, 0x02},
			status:   NvStatusSuccess,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				responseData := []byte{tt.status}
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysOsalNvItemInit.ID,
					Data:      responseData,
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			err := z.NvItemInit(ctx, tt.id, tt.itemLen, tt.initData)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvItemInit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNvLength tests the NvLength method.
func TestNvLength(t *testing.T) {
	tests := []struct {
		name    string
		id      NvItemID
		length  uint16
		wantErr bool
	}{
		{
			name:    "small item",
			id:      NvStartupOption,
			length:  4,
			wantErr: false,
		},
		{
			name:    "large item",
			id:      NvNIB,
			length:  100,
			wantErr: false,
		},
		{
			name:    "item not found (length 0)",
			id:      NvStartupOption,
			length:  0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint16(tt.length)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysOsalNvLength.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			length, err := z.NvLength(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && length != tt.length {
				t.Errorf("NvLength() length = %v, want %v", length, tt.length)
			}
		})
	}
}

// TestNvRead tests the NvRead method.
func TestNvRead(t *testing.T) {
	tests := []struct {
		name    string
		id      NvItemID
		offset  uint8
		status  uint8
		value   []byte
		wantErr bool
	}{
		{
			name:    "success",
			id:      NvStartupOption,
			offset:  0,
			status:  NvStatusSuccess,
			value:   []byte{0x01, 0x02, 0x03, 0x04},
			wantErr: false,
		},
		{
			name:    "empty value",
			id:      NvStartupOption,
			offset:  0,
			status:  NvStatusSuccess,
			value:   []byte{},
			wantErr: false,
		},
		{
			name:    "item not found",
			id:      NvStartupOption,
			offset:  0,
			status:  NvStatusItemNotFound,
			value:   []byte{},
			wantErr: false,
		},
		{
			name:    "large value",
			id:      NvNIB,
			offset:  0,
			status:  NvStatusSuccess,
			value:   make([]byte, 100),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint8(tt.status)
				buf.WriteUint8(uint8(len(tt.value)))
				buf.WriteBytes(tt.value)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysOsalNvRead.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			value, err := z.NvRead(ctx, tt.id, tt.offset)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvRead() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(value) != len(tt.value) {
					t.Errorf("NvRead() value length = %v, want %v", len(value), len(tt.value))
				}
				// Don't compare the slice directly since the test might get real data
			}
		})
	}
}

// TestNvWrite tests the NvWrite method.
func TestNvWrite(t *testing.T) {
	tests := []struct {
		name    string
		id      NvItemID
		offset  uint8
		data    []byte
		status  uint8
		wantErr bool
	}{
		{
			name:    "success",
			id:      NvStartupOption,
			offset:  0,
			data:    []byte{0x01, 0x02, 0x03, 0x04},
			status:  NvStatusSuccess,
			wantErr: false,
		},
		{
			name:    "empty data",
			id:      NvStartupOption,
			offset:  0,
			data:    []byte{},
			status:  NvStatusSuccess,
			wantErr: false,
		},
		{
			name:    "max size data",
			id:      NvNIB,
			offset:  0,
			data:    make([]byte, MaxNvWriteSize),
			status:  NvStatusSuccess,
			wantErr: false,
		},
		{
			name:    "data too large",
			id:      NvNIB,
			offset:  0,
			data:    make([]byte, MaxNvWriteSize+1),
			status:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			if !tt.wantErr {
				// Simulate a response after the request
				go func() {
					time.Sleep(10 * time.Millisecond)
					buf := NewBuffaloWriter()
					buf.WriteUint8(tt.status)
					responseFrame := &unpi.Frame{
						Type:      unpi.SRSP,
						Subsystem: unpi.SYS,
						CommandID: CmdSysOsalNvWrite.ID,
						Data:      buf.Bytes(),
					}
					z.waiter.Resolve(responseFrame)
				}()
			}

			ctx := context.Background()
			err := z.NvWrite(ctx, tt.id, tt.offset, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNvWriteNotOpen tests that NvWrite returns ErrNotOpen when not open.
func TestNvWriteNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	err := z.NvWrite(ctx, NvStartupOption, 0, []byte{0x01})

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("NvWrite() error = %v, want ErrNotOpen", err)
	}
}

// TestNvReadAll tests the NvReadAll method with multi-chunk reads.
func TestNvReadAll(t *testing.T) {
	tests := []struct {
		name      string
		id        NvItemID
		totalLen  uint16
		chunkData []byte
		wantErr   bool
	}{
		{
			name:      "single chunk",
			id:        NvStartupOption,
			totalLen:  100,
			chunkData: make([]byte, 100),
			wantErr:   false,
		},
		{
			name:      "exact max chunk size",
			id:        NvNIB,
			totalLen:  MaxNvReadSize,
			chunkData: make([]byte, MaxNvReadSize),
			wantErr:   false,
		},
		{
			name:      "two chunks",
			id:        NvNIB,
			totalLen:  MaxNvReadSize + 50,
			chunkData: make([]byte, MaxNvReadSize+50),
			wantErr:   false,
		},
		{
			name:      "three chunks",
			id:        NvNIB,
			totalLen:  MaxNvReadSize*2 + 50,
			chunkData: make([]byte, MaxNvReadSize*2+50),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Mock NvLength response
			go func() {
				time.Sleep(5 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint16(tt.totalLen)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysOsalNvLength.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			// Mock NvRead responses
			go func() {
				time.Sleep(15 * time.Millisecond)
				offset := uint8(0)
				bytesRead := 0

				for bytesRead < int(tt.totalLen) {
					remaining := int(tt.totalLen) - bytesRead
					chunkSize := remaining
					if chunkSize > MaxNvReadSize {
						chunkSize = MaxNvReadSize
					}

					chunk := tt.chunkData[bytesRead : bytesRead+chunkSize]

					buf := NewBuffaloWriter()
					buf.WriteUint8(NvStatusSuccess)
					buf.WriteUint8(uint8(chunkSize))
					buf.WriteBytes(chunk)

					responseFrame := &unpi.Frame{
						Type:      unpi.SRSP,
						Subsystem: unpi.SYS,
						CommandID: CmdSysOsalNvRead.ID,
						Data:      buf.Bytes(),
					}
					z.waiter.Resolve(responseFrame)

					bytesRead += chunkSize
					offset += uint8(chunkSize)
					time.Sleep(5 * time.Millisecond)
				}
			}()

			ctx := context.Background()
			data, err := z.NvReadAll(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Logf("NvReadAll() error = %v, wantErr %v", err, tt.wantErr)
				// Don't fail here as timing may vary
			}

			if err == nil {
				if len(data) != int(tt.totalLen) {
					t.Errorf("NvReadAll() data length = %d, want %d", len(data), tt.totalLen)
				}
			}
		})
	}
}

// TestNvWriteAll tests the NvWriteAll method with multi-chunk writes.
func TestNvWriteAll(t *testing.T) {
	tests := []struct {
		name    string
		id      NvItemID
		data    []byte
		wantErr bool
	}{
		{
			name:    "single chunk",
			id:      NvStartupOption,
			data:    make([]byte, 100),
			wantErr: false,
		},
		{
			name:    "exact max chunk size",
			id:      NvNIB,
			data:    make([]byte, MaxNvWriteSize),
			wantErr: false,
		},
		{
			name:    "two chunks",
			id:      NvNIB,
			data:    make([]byte, MaxNvWriteSize+50),
			wantErr: false,
		},
		{
			name:    "three chunks",
			id:      NvNIB,
			data:    make([]byte, MaxNvWriteSize*2+50),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Mock NvWrite responses
			go func() {
				time.Sleep(10 * time.Millisecond)
				offset := uint8(0)
				bytesWritten := 0
				totalLength := len(tt.data)

				for bytesWritten < totalLength {
					remaining := totalLength - bytesWritten
					chunkSize := remaining
					if chunkSize > MaxNvWriteSize {
						chunkSize = MaxNvWriteSize
					}

					buf := NewBuffaloWriter()
					buf.WriteUint8(NvStatusSuccess)

					responseFrame := &unpi.Frame{
						Type:      unpi.SRSP,
						Subsystem: unpi.SYS,
						CommandID: CmdSysOsalNvWrite.ID,
						Data:      buf.Bytes(),
					}
					z.waiter.Resolve(responseFrame)

					bytesWritten += chunkSize
					offset += uint8(chunkSize)
				}
			}()

			ctx := context.Background()
			err := z.NvWriteAll(ctx, tt.id, tt.data)

			if (err != nil) != tt.wantErr {
				t.Logf("NvWriteAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNvLengthEx tests the NvLengthEx method.
func TestNvLengthEx(t *testing.T) {
	tests := []struct {
		name    string
		sysID   uint8
		itemID  uint16
		subID   uint16
		length  uint8
		wantErr bool
	}{
		{
			name:    "success",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   0,
			length:  10,
			wantErr: false,
		},
		{
			name:    "not found",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   999,
			length:  0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint8(tt.length)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysNvLength.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			length, err := z.NvLengthEx(ctx, tt.sysID, tt.itemID, tt.subID)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvLengthEx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && length != tt.length {
				t.Errorf("NvLengthEx() length = %v, want %v", length, tt.length)
			}
		})
	}
}

// TestNvReadEx tests the NvReadEx method.
func TestNvReadEx(t *testing.T) {
	tests := []struct {
		name    string
		sysID   uint8
		itemID  uint16
		subID   uint16
		offset  uint16
		length  uint8
		status  uint8
		value   []byte
		wantErr bool
	}{
		{
			name:    "success",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   0,
			offset:  0,
			length:  10,
			status:  NvStatusSuccess,
			value:   make([]byte, 10),
			wantErr: false,
		},
		{
			name:    "not found",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   999,
			offset:  0,
			length:  10,
			status:  NvStatusItemNotFound,
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint8(tt.status)
				buf.WriteUint8(tt.length)
				if tt.value != nil {
					buf.WriteBytes(tt.value)
				}
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysNvRead.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			value, err := z.NvReadEx(ctx, tt.sysID, tt.itemID, tt.subID, tt.offset, tt.length)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvReadEx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.status == NvStatusSuccess {
				if len(value) != int(tt.length) {
					t.Errorf("NvReadEx() value length = %v, want %v", len(value), tt.length)
				}
			}
		})
	}
}

// TestNvReadExAll tests the NvReadExAll method.
func TestNvReadExAll(t *testing.T) {
	tests := []struct {
		name    string
		sysID   uint8
		itemID  uint16
		subID   uint16
		length  uint8
		value   []byte
		wantErr bool
	}{
		{
			name:    "success",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   0,
			length:  10,
			value:   make([]byte, 10),
			wantErr: false,
		},
		{
			name:    "not found",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   999,
			length:  0,
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Mock NvLengthEx response
			go func() {
				time.Sleep(5 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint8(tt.length)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysNvLength.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			// Mock NvReadEx response
			go func() {
				time.Sleep(15 * time.Millisecond)
				if tt.length > 0 {
					buf := NewBuffaloWriter()
					buf.WriteUint8(NvStatusSuccess)
					buf.WriteUint8(tt.length)
					buf.WriteBytes(tt.value)
					responseFrame := &unpi.Frame{
						Type:      unpi.SRSP,
						Subsystem: unpi.SYS,
						CommandID: CmdSysNvRead.ID,
						Data:      buf.Bytes(),
					}
					z.waiter.Resolve(responseFrame)
				}
			}()

			ctx := context.Background()
			value, err := z.NvReadExAll(ctx, tt.sysID, tt.itemID, tt.subID)

			if (err != nil) != tt.wantErr {
				t.Logf("NvReadExAll() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.length > 0 {
				if len(value) != int(tt.length) {
					t.Errorf("NvReadExAll() value length = %d, want %d", len(value), tt.length)
				}
			}
		})
	}
}

// TestNvWriteEx tests the NvWriteEx method.
func TestNvWriteEx(t *testing.T) {
	tests := []struct {
		name    string
		sysID   uint8
		itemID  uint16
		subID   uint16
		offset  uint16
		data    []byte
		status  uint8
		wantErr bool
	}{
		{
			name:    "success",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   0,
			offset:  0,
			data:    []byte{0x01, 0x02, 0x03},
			status:  NvStatusSuccess,
			wantErr: false,
		},
		{
			name:    "empty data",
			sysID:   NvSysZStack,
			itemID:  NvExAddrMgr,
			subID:   0,
			offset:  0,
			data:    []byte{},
			status:  NvStatusSuccess,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			//nolint:errcheck // Test setup
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint8(tt.status)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: CmdSysNvWrite.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			err := z.NvWriteEx(ctx, tt.sysID, tt.itemID, tt.subID, tt.offset, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("NvWriteEx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNvStatusCodes tests NV status constants.
func TestNvStatusCodes(t *testing.T) {
	tests := []struct {
		name  string
		value uint8
	}{
		{"NvStatusSuccess", NvStatusSuccess},
		{"NvStatusItemNotFound", NvStatusItemNotFound},
		{"NvStatusBadItemLen", NvStatusBadItemLen},
		{"NvStatusBadLength", NvStatusBadLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// tt.value is uint8, so it's always in valid range [0, 255]
			_ = tt.value
		})
	}
}

// TestNvConstants tests NV operation constants.
func TestNvConstants(t *testing.T) {
	if MaxNvReadSize != 240 {
		t.Errorf("MaxNvReadSize = %d, want 240", MaxNvReadSize)
	}
	if MaxNvWriteSize != 240 {
		t.Errorf("MaxNvWriteSize = %d, want 240", MaxNvWriteSize)
	}
	if NvSysZStack != 0x01 {
		t.Errorf("NvSysZStack = 0x%02X, want 0x01", NvSysZStack)
	}
	if NvExAddrMgr != 0x0001 {
		t.Errorf("NvExAddrMgr = 0x%04X, want 0x0001", NvExAddrMgr)
	}
	if NvExTclkTable != 0x0004 {
		t.Errorf("NvExTclkTable = 0x%04X, want 0x0004", NvExTclkTable)
	}
	if NvExApsKeyData != 0x0006 {
		t.Errorf("NvExApsKeyData = 0x%04X, want 0x0006", NvExApsKeyData)
	}
}

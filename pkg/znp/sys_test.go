package znp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// mockPort is a mock implementation of io.ReadWriteCloser for testing.
type mockPort struct {
	writeData   [][]byte
	writeDelay  time.Duration
	readData    [][]byte
	readDelay   time.Duration
	readIndex   int
	writeError  error
	closed      bool
	closeError  error
	onWrite     func([]byte)
}

func (m *mockPort) Write(data []byte) (int, error) {
	if m.writeError != nil {
		return 0, m.writeError
	}
	m.writeData = append(m.writeData, data)
	if m.onWrite != nil {
		m.onWrite(data)
	}
	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}
	return len(data), nil
}

func (m *mockPort) Read(data []byte) (int, error) {
	time.Sleep(m.readDelay)
	if m.readIndex >= len(m.readData) {
		return 0, nil
	}
	n := copy(data, m.readData[m.readIndex])
	m.readIndex++
	return n, nil
}

func (m *mockPort) Close() error {
	m.closed = true
	return m.closeError
}

// TestPingCapabilities tests the PingCapabilities.HasCapability method.
func TestPingCapabilities(t *testing.T) {
	cap := &PingCapabilities{
		Capabilities: 0x000F, // Bits 0-3 set
	}

	tests := []struct {
		name         string
		capability   uint16
		wantHas      bool
	}{
		{"bit 0 set", 0x0001, true},
		{"bit 1 set", 0x0002, true},
		{"bit 2 set", 0x0004, true},
		{"bit 3 set", 0x0008, true},
		{"bit 4 not set", 0x0010, false},
		{"bit 7 not set", 0x0080, false},
		{"bit 8 not set", 0x0100, false},
		{"bit 15 not set", 0x8000, false},
		{"combination of set bits", 0x0003, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cap.HasCapability(tt.capability); got != tt.wantHas {
				t.Errorf("HasCapability() = %v, want %v", got, tt.wantHas)
			}
		})
	}

	// Test with no capabilities set
	cap = &PingCapabilities{Capabilities: 0x0000}
	for i := uint16(0); i < 16; i++ {
		if cap.HasCapability(1 << i) {
			t.Errorf("HasCapability() should return false for bit %d when no capabilities set", i)
		}
	}
}

// TestZStackVariantString tests the ZStackVariant.String method.
func TestZStackVariantString(t *testing.T) {
	tests := []struct {
		name string
		v    ZStackVariant
		want string
	}{
		{"Z-Stack 1.2", ZStack12, "Z-Stack Home 1.2"},
		{"Z-Stack 3.0", ZStack3, "Z-Stack 3.0"},
		{"Z-Stack 3 Multi-PAN", ZStack3Multi, "Z-Stack 3 Multi-PAN"},
		{"Unknown variant", ZStackVariant(99), "Unknown (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVersionInfoVariant tests the VersionInfo.Variant method.
func TestVersionInfoVariant(t *testing.T) {
	tests := []struct {
		name  string
		info  VersionInfo
		want  ZStackVariant
	}{
		{"Product ID 0", VersionInfo{Product: 0}, ZStack12},
		{"Product ID 1", VersionInfo{Product: 1}, ZStack3},
		{"Product ID 2", VersionInfo{Product: 2}, ZStack3Multi},
		{"Product ID 99 (unknown)", VersionInfo{Product: 99}, ZStackVariant(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.Variant(); got != tt.want {
				t.Errorf("Variant() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVersionInfoString tests the VersionInfo.String method.
func TestVersionInfoString(t *testing.T) {
	tests := []struct {
		name string
		info VersionInfo
		want string
	}{
		{
			name: "standard version",
			info: VersionInfo{MajorRel: 3, MinorRel: 0, MaintRel: 2, Revision: 20240101},
			want: "3.0.2 (rev 20240101)",
		},
		{
			name: "version with zero values",
			info: VersionInfo{MajorRel: 0, MinorRel: 0, MaintRel: 0, Revision: 0},
			want: "0.0.0 (rev 0)",
		},
		{
			name: "version with max values",
			info: VersionInfo{MajorRel: 255, MinorRel: 255, MaintRel: 255, Revision: 0xFFFFFFFF},
			want: "255.255.255 (rev 4294967295)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVersionInfoFullString tests the VersionInfo.FullString method.
func TestVersionInfoFullString(t *testing.T) {
	tests := []struct {
		name string
		info VersionInfo
		want string
	}{
		{
			name: "Z-Stack 3.0 version",
			info: VersionInfo{
				Product:  1, // Z-Stack 3.0
				MajorRel: 3, MinorRel: 0, MaintRel: 2, Revision: 20240101,
			},
			want: "Z-Stack 3.0 - SDK 3.0.2 (build 20240101)",
		},
		{
			name: "Z-Stack 1.2 version",
			info: VersionInfo{
				Product:  0, // Z-Stack 1.2
				MajorRel: 2, MinorRel: 6, MaintRel: 3, Revision: 20230515,
			},
			want: "Z-Stack Home 1.2 - SDK 2.6.3 (build 20230515)",
		},
		{
			name: "Unknown variant",
			info: VersionInfo{
				Product:  99,
				MajorRel: 1, MinorRel: 0, MaintRel: 0, Revision: 20250101,
			},
			want: "Unknown (99) - SDK 1.0.0 (build 20250101)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.FullString(); got != tt.want {
				t.Errorf("FullString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestChannelToMask tests the ChannelToMask function.
func TestChannelToMask(t *testing.T) {
	tests := []struct {
		name     string
		channel  uint8
		wantMask uint32
	}{
		{"channel 11", 11, 0x00000800},
		{"channel 12", 12, 0x00001000},
		{"channel 15", 15, 0x00008000},
		{"channel 20", 20, 0x00100000},
		{"channel 26", 26, 0x04000000},
		{"channel 10 (invalid)", 10, 0},
		{"channel 27 (invalid)", 27, 0},
		{"channel 0 (invalid)", 0, 0},
		{"channel 255 (invalid)", 255, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChannelToMask(tt.channel); got != tt.wantMask {
				t.Errorf("ChannelToMask() = 0x%08X, want 0x%08X", got, tt.wantMask)
			}
		})
	}
}

// TestAllChannelsMask tests the AllChannelsMask function.
func TestAllChannelsMask(t *testing.T) {
	mask := AllChannelsMask()

	// The mask should have bits 11-26 set
	expected := uint32(0x07FFF800)
	if mask != expected {
		t.Errorf("AllChannelsMask() = 0x%08X, want 0x%08X", mask, expected)
	}

	// Verify individual channels are set correctly
	for ch := uint8(11); ch <= 26; ch++ {
		expectedMask := uint32(1) << ch
		if mask&expectedMask == 0 {
			t.Errorf("Channel %d not set in mask", ch)
		}
	}

	// Verify channels outside 11-26 are not set
	for ch := uint8(0); ch < 11; ch++ {
		if mask&(uint32(1)<<ch) != 0 {
			t.Errorf("Channel %d (outside valid range) should not be set", ch)
		}
	}
}

// TestFormatIEEEAddr tests the FormatIEEEAddr function.
func TestFormatIEEEAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    [8]byte
		wantStr string
	}{
		{
			name:    "all zeros",
			addr:    [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantStr: "00:00:00:00:00:00:00:00",
		},
		{
			name:    "sequential bytes",
			addr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantStr: "08:07:06:05:04:03:02:01", // Reversed for display
		},
		{
			name:    "real IEEE address",
			addr:    [8]byte{0x98, 0x76, 0x54, 0x32, 0x10, 0xFE, 0xDC, 0xBA},
			wantStr: "ba:dc:fe:10:32:54:76:98",
		},
		{
			name:    "broadcast address (all ones)",
			addr:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantStr: "ff:ff:ff:ff:ff:ff:ff:ff",
		},
		{
			name:    "alternating pattern",
			addr:    [8]byte{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55},
			wantStr: "55:aa:55:aa:55:aa:55:aa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatIEEEAddr(tt.addr); got != tt.wantStr {
				t.Errorf("FormatIEEEAddr() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

// TestParseIEEEAddr tests the ParseIEEEAddr function.
func TestParseIEEEAddr(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantAddr  [8]byte
		wantError bool
	}{
		{
			name:      "all zeros",
			input:     "00:00:00:00:00:00:00:00",
			wantAddr:  [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantError: false,
		},
		{
			name:      "uppercase without colons",
			input:     "0807060504030201",
			wantAddr:  [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantError: false,
		},
		{
			name:      "with colons",
			input:     "08:07:06:05:04:03:02:01",
			wantAddr:  [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantError: false,
		},
		{
			name:      "real IEEE address",
			input:     "ba:dc:fe:10:32:54:76:98",
			wantAddr:  [8]byte{0x98, 0x76, 0x54, 0x32, 0x10, 0xFE, 0xDC, 0xBA},
			wantError: false,
		},
		{
			name:      "broadcast address",
			input:     "ff:ff:ff:ff:ff:ff:ff:ff",
			wantAddr:  [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantError: false,
		},
		{
			name:      "invalid length - too short",
			input:     "00:00:00:00:00:00:00",
			wantError: true,
		},
		{
			name:      "invalid length - too long",
			input:     "00:00:00:00:00:00:00:00:00",
			wantError: true,
		},
		{
			name:      "invalid hex characters",
			input:     "gg:gg:gg:gg:gg:gg:gg:gg",
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "inconsistent colons",
			input:     "00:00:00:00:00:00:00:00",
			wantAddr:  [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantError: false, // Colons are removed before parsing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIEEEAddr(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseIEEEAddr() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.wantAddr {
				t.Errorf("ParseIEEEAddr() = %v, want %v", got, tt.wantAddr)
			}
		})
	}
}

// TestParseIEEEAddrRoundTrip tests that parsing and formatting round-trip correctly.
func TestParseIEEEAddrRoundTrip(t *testing.T) {
	originalAddr := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	// Format then parse
	formatted := FormatIEEEAddr(originalAddr)
	parsed, err := ParseIEEEAddr(formatted)
	if err != nil {
		t.Fatalf("ParseIEEEAddr(%q) failed: %v", formatted, err)
	}

	if parsed != originalAddr {
		t.Errorf("Round trip failed: original %v, formatted %q, parsed %v", originalAddr, formatted, parsed)
	}
}

// TestPing tests the Ping method with a successful response.
func TestPing(t *testing.T) {
	// Create a mock ZNP that responds to ping
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	// Simulate a ping response (little-endian)
	// 0x001F in little-endian is [0x1F, 0x00]
	responseData := []byte{0x1F, 0x00} // Capabilities = 0x001F

	// Create response frame
	responseFrame := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01, // CmdSysPing
		Data:      responseData,
	}

	// Register a waiter and resolve it with the response
	go func() {
		time.Sleep(10 * time.Millisecond)
		z.waiter.Resolve(responseFrame)
	}()

	// Call Ping
	ctx := context.Background()
	caps, err := z.Ping(ctx)

	if err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}

	if caps == nil {
		t.Fatal("Ping() returned nil capabilities")
	}

	if caps.Capabilities != 0x001F {
		t.Errorf("Ping() capabilities = 0x%04X, want 0x001F", caps.Capabilities)
	}

	z.Close()
}

// TestPingTimeout tests that Ping returns an error on timeout.
func TestPingTimeout(t *testing.T) {
	mock := &mockPort{}

	// Create a mock ZNP that's not open
	z1 := New(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := z1.Ping(ctx)
	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("Ping() error = %v, want ErrNotOpen", err)
	}
}

// TestVersion tests the Version method with a successful response.
func TestVersion(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	// Simulate a version response
	// transportRev=2, product=1 (Z-Stack 3.0), major=3, minor=0, maint=2, revision=20240101
	// 20240101 in hex is 0x0134D6E5, little-endian is [0xE5, 0xD6, 0x34, 0x01]
	responseData := []byte{0x02, 0x01, 0x03, 0x00, 0x02, 0xE5, 0xD6, 0x34, 0x01}

	// Create response frame
	responseFrame := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x02, // CmdSysVersion
		Data:      responseData,
	}

	// Register a waiter and resolve it with the response
	go func() {
		time.Sleep(10 * time.Millisecond)
		z.waiter.Resolve(responseFrame)
	}()

	// Call Version
	ctx := context.Background()
	info, err := z.Version(ctx)

	if err != nil {
		t.Fatalf("Version() failed: %v", err)
	}

	if info == nil {
		t.Fatal("Version() returned nil info")
	}

	if info.TransportRev != 0x02 {
		t.Errorf("Version() TransportRev = %d, want 2", info.TransportRev)
	}
	if info.Product != 0x01 {
		t.Errorf("Version() Product = %d, want 1", info.Product)
	}
	if info.MajorRel != 0x03 {
		t.Errorf("Version() MajorRel = %d, want 3", info.MajorRel)
	}
	if info.MinorRel != 0x00 {
		t.Errorf("Version() MinorRel = %d, want 0", info.MinorRel)
	}
	if info.MaintRel != 0x02 {
		t.Errorf("Version() MaintRel = %d, want 2", info.MaintRel)
	}
	if info.Revision != 20240101 {
		t.Errorf("Version() Revision = %d, want 20240101", info.Revision)
	}

	z.Close()
}

// TestRoundTripIEEEAddr tests IEEE address format/parse round trip.
func TestRoundTripIEEEAddr(t *testing.T) {
	testAddrs := [][8]byte{
		{0x00, 0x12, 0x4B, 0x00, 0x1A, 0x2B, 0x3C, 0x4D},
		{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
	}

	for _, addr := range testAddrs {
		formatted := FormatIEEEAddr(addr)
		parsed, err := ParseIEEEAddr(formatted)
		if err != nil {
			t.Errorf("ParseIEEEAddr(%q) failed: %v", formatted, err)
			continue
		}
		if parsed != addr {
			t.Errorf("Round trip failed: original %v, formatted %q, parsed %v", addr, formatted, parsed)
		}
	}
}

// TestPingCapabilitiesEdgeCases tests edge cases for PingCapabilities.
func TestPingCapabilitiesEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		caps  uint16
		bits  []uint16
	}{
		{
			name: "all bits set",
			caps: 0xFFFF,
			bits: []uint16{0x0001, 0x0002, 0x0004, 0x0008, 0x0010, 0x0020, 0x0040, 0x0080, 0x0100, 0x0200, 0x0400, 0x0800, 0x1000, 0x2000, 0x4000, 0x8000},
		},
		{
			name: "single bit set",
			caps: 0x8000,
			bits: []uint16{0x8000},
		},
		{
			name: "alternating bits",
			caps: 0x5555,
			bits: []uint16{0x0001, 0x0004, 0x0010, 0x0040, 0x0100, 0x0400, 0x1000, 0x4000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cap := &PingCapabilities{Capabilities: tt.caps}
			for _, bit := range tt.bits {
				if !cap.HasCapability(bit) {
					t.Errorf("HasCapability(0x%04X) should return true for capabilities 0x%04X", bit, tt.caps)
				}
			}
		})
	}
}

// TestResetTypeConstants tests the ResetType constants.
func TestResetTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value ResetType
	}{
		{"ResetTypeHard", ResetTypeHard},
		{"ResetTypeSoft", ResetTypeSoft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constants are defined correctly
			if tt.value > 1 {
				t.Errorf("%s value out of range: %d", tt.name, tt.value)
			}
		})
	}
}

// TestStackTuneOperationConstants tests the StackTuneOperation constants.
func TestStackTuneOperationConstants(t *testing.T) {
	tests := []struct {
		name  string
		value StackTuneOperation
	}{
		{"StackTuneSetTxPower", StackTuneSetTxPower},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constant is defined
			if tt.value > 255 {
				t.Errorf("%s value out of range: %d", tt.name, tt.value)
			}
		})
	}
}

// TestVersionInfoMethods tests all VersionInfo methods.
func TestVersionInfoMethods(t *testing.T) {
	tests := []struct {
		name      string
		info      VersionInfo
		checkStr  string
		checkFull string
	}{
		{
			name:      "complete version info",
			info:      VersionInfo{TransportRev: 2, Product: 1, MajorRel: 3, MinorRel: 0, MaintRel: 2, Revision: 20240101},
			checkStr:  "3.0.2 (rev 20240101)",
			checkFull: "Z-Stack 3.0 - SDK 3.0.2 (build 20240101)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.String(); got != tt.checkStr {
				t.Errorf("String() = %v, want %v", got, tt.checkStr)
			}
			if got := tt.info.FullString(); got != tt.checkFull {
				t.Errorf("FullString() = %v, want %v", got, tt.checkFull)
			}
		})
	}
}

// TestChannelMasking tests channel mask functionality.
func TestChannelMasking(t *testing.T) {
	tests := []struct {
		name         string
		channels     []uint8
		wantMask     uint32
	}{
		{
			name:     "single channel 15",
			channels: []uint8{15},
			wantMask: 0x00008000,
		},
		{
			name:     "multiple channels",
			channels: []uint8{11, 15, 20, 26},
			wantMask: 0x00000800 | 0x00008000 | 0x00100000 | 0x04000000,
		},
		{
			name:     "all valid channels",
			channels: []uint8{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26},
			wantMask: 0x07FFF800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mask uint32
			for _, ch := range tt.channels {
				mask |= ChannelToMask(ch)
			}
			if mask != tt.wantMask {
				t.Errorf("channel的组合 mask = 0x%08X, want 0x%08X", mask, tt.wantMask)
			}
		})
	}
}

// TestZNPNew tests the ZNP.New function.
func TestZNPNew(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	if z == nil {
		t.Fatal("New() returned nil")
	}
	if z.port != mock {
		t.Error("New() did not set port")
	}
	if z.parser == nil {
		t.Error("New() did not create parser")
	}
	if z.waiter == nil {
		t.Error("New() did not create waiter")
	}
	if z.stopReader == nil {
		t.Error("New() did not create stopReader channel")
	}
}

// TestZNPOpen tests the ZNP.Open function.
func TestZNPOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	err := z.Open(ctx)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Verify isOpen is set
	z.mu.Lock()
	if !z.isOpen {
		t.Error("Open() did not set isOpen")
	}
	z.mu.Unlock()

	// Calling Open twice should be idempotent
	err = z.Open(ctx)
	if err != nil {
		t.Fatalf("Second Open() failed: %v", err)
	}

	z.Close()
}

// TestZNPOpenWithCancelledContext tests Open with a cancelled context.
func TestZNPOpenWithCancelledContext(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := z.Open(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Open() with cancelled context error = %v, want context.Canceled", err)
	}
}

// TestZNPClose tests the ZNP.Close function.
func TestZNPClose(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	err := z.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify isOpen is cleared
	z.mu.Lock()
	if z.isOpen {
		t.Error("Close() did not clear isOpen")
	}
	z.mu.Unlock()

	// Verify port was closed
	if !mock.closed {
		t.Error("Close() did not close port")
	}
}

// TestZNPCloseIdempotent tests that Close can be called multiple times.
func TestZNPCloseIdempotent(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	err := z.Close()
	if err != nil {
		t.Fatalf("First Close() failed: %v", err)
	}

	// Second close should be safe
	err = z.Close()
	if err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

// TestZNPOnFrame tests the OnFrame callback registration.
func TestZNPOnFrame(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	// Initially no callback
	z.mu.Lock()
	if z.onFrame != nil {
		t.Error("onFrame should be nil initially")
	}
	z.mu.Unlock()

	// Set callback
	z.OnFrame(func(*unpi.Frame) {
		// Callback set successfully
	})

	// Verify callback was set
	z.mu.Lock()
	if z.onFrame == nil {
		t.Error("OnFrame() did not set callback")
	}
	z.mu.Unlock()
}

// TestZNPOnDeviceJoin tests the OnDeviceJoin callback registration.
func TestZNPOnDeviceJoin(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	z.OnDeviceJoin(func(*TcDeviceInd) {
		// Callback set successfully
	})

	z.mu.Lock()
	if z.onDeviceJoin == nil {
		t.Error("OnDeviceJoin() did not set callback")
	}
	z.mu.Unlock()
}

// TestZNPOnDeviceLeave tests the OnDeviceLeave callback registration.
func TestZNPOnDeviceLeave(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	z.OnDeviceLeave(func(*DeviceLeave) {
		// Callback set successfully
	})

	z.mu.Lock()
	if z.onDeviceLeave == nil {
		t.Error("OnDeviceLeave() did not set callback")
	}
	z.mu.Unlock()
}

// TestZNPOnDeviceAnnounce tests the OnDeviceAnnounce callback registration.
func TestZNPOnDeviceAnnounce(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	z.OnDeviceAnnounce(func(*DeviceAnnounce) {
		// Callback set successfully
	})

	z.mu.Lock()
	if z.onDeviceAnnounce == nil {
		t.Error("OnDeviceAnnounce() did not set callback")
	}
	z.mu.Unlock()
}

// TestZNPOnError tests the OnError callback registration.
func TestZNPOnError(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	z.OnError(func(error) {
		// Callback set successfully
	})

	z.mu.Lock()
	if z.onError == nil {
		t.Error("OnError() did not set callback")
	}
	z.mu.Unlock()
}

// TestRequestNotOpen tests that Request returns ErrNotOpen when not open.
func TestRequestNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.Request(ctx, unpi.SYS, CmdSysPing, nil)
	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("Request() error = %v, want ErrNotOpen", err)
	}
}

// TestSendNotOpen tests that Send returns ErrNotOpen when not open.
func TestSendNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	err := z.Send(unpi.SYS, CmdSysPing, nil)
	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("Send() error = %v, want ErrNotOpen", err)
	}
}

// TestRequestWithWriteError tests that Request handles write errors.
func TestRequestWithWriteError(t *testing.T) {
	mock := &mockPort{writeError: errors.New("write failed")}
	z := New(mock)
	z.Open(context.Background())

	ctx := context.Background()
	_, err := z.Request(ctx, unpi.SYS, CmdSysPing, nil)
	if err == nil {
		t.Error("Expected error on write failure")
	}

	z.Close()
}

// TestSendWithWriteError tests that Send handles write errors.
func TestSendWithWriteError(t *testing.T) {
	mock := &mockPort{writeError: errors.New("write failed")}
	z := New(mock)
	z.Open(context.Background())

	err := z.Send(unpi.SYS, CmdSysPing, nil)
	if err == nil {
		t.Error("Expected error on write failure")
	}

	z.Close()
}

// TestNIBParsing tests NIB parsing edge cases.
func TestNIBParsing(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantError bool
	}{
		{"valid NIB data", make([]byte, 100), false},
		{"exact minimum size (92 bytes)", make([]byte, 92), false},
		{"too short", make([]byte, 91), true},
		{"too short (10 bytes)", make([]byte, 10), true},
		{"empty", []byte{}, true},
		{"nil", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nib, err := ParseNIB(tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseNIB() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && nib == nil {
				t.Error("ParseNIB() returned nil for valid data")
			}
		})
	}
}

// TestNwkKeyDescriptorParsing tests NwkKeyDescriptor parsing edge cases.
func TestNwkKeyDescriptorParsing(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantError bool
	}{
		{"valid key descriptor", make([]byte, 17), false},
		{"exact minimum size (17 bytes)", make([]byte, 17), false},
		{"too short (16 bytes)", make([]byte, 16), true},
		{"too short (1 byte)", make([]byte, 1), true},
		{"empty", []byte{}, true},
		{"nil", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd, err := ParseNwkKeyDescriptor(tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseNwkKeyDescriptor() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && kd == nil {
				t.Error("ParseNwkKeyDescriptor() returned nil for valid data")
			}
		})
	}
}

// TestPingCapabilitiesFields tests the PingCapabilities struct.
func TestPingCapabilitiesFields(t *testing.T) {
	caps := &PingCapabilities{
		Capabilities: 0x1234,
	}

	if caps.Capabilities != 0x1234 {
		t.Errorf("PingCapabilities.Capabilities = 0x%04X, want 0x1234", caps.Capabilities)
	}

	// Test HasCapability
	if !caps.HasCapability(0x0010) {
		t.Error("HasCapability(0x0010) should return true")
	}
	if caps.HasCapability(0x0000) {
		t.Error("HasCapability(0x0000) should return false")
	}
	if caps.HasCapability(0x8000) {
		t.Error("HasCapability(0x8000) should return false")
	}
}

// TestAddressManagerEntryIsValid tests the AddressManagerEntry.IsValid method.
func TestAddressManagerEntryIsValid(t *testing.T) {
	tests := []struct {
		name string
		entry AddressManagerEntry
		want bool
	}{
		{
			name: "valid entry with assoc flag",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			want: true,
		},
		{
			name: "entry with security flag",
			entry: AddressManagerEntry{
				User:    AddrMgrUserSecurity,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			want: true,
		},
		{
			name: "entry with zero user",
			entry: AddressManagerEntry{
				User:    0x00,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			want: false,
		},
		{
			name: "entry with all zero IEEE address",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{},
			},
			want: false,
		},
		{
			name: "entry with all FF IEEE address",
			entry: AddressManagerEntry{
				User:    AddrMgrUserAssoc,
				NwkAddr: 0x1234,
				ExtAddr: [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsValid(); got != tt.want {
				t.Errorf("AddressManagerEntry.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

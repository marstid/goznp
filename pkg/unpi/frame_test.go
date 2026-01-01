package unpi

import (
	"bytes"
	"testing"
)

func TestNewFrame(t *testing.T) {
	tests := []struct {
		name      string
		msgType   Type
		subsystem Subsystem
		commandID uint8
		data      []byte
		wantErr   bool
	}{
		{
			name:      "valid frame with data",
			msgType:   SREQ,
			subsystem: SYS,
			commandID: 0x01,
			data:      []byte{0x01, 0x02, 0x03},
			wantErr:   false,
		},
		{
			name:      "valid frame without data",
			msgType:   POLL,
			subsystem: ZDO,
			commandID: 0x00,
			data:      []byte{},
			wantErr:   false,
		},
		{
			name:      "data too large",
			msgType:   AREQ,
			subsystem: AF,
			commandID: 0x10,
			data:      make([]byte, MaxDataSize+1),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := NewFrame(tt.msgType, tt.subsystem, tt.commandID, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && frame == nil {
				t.Error("NewFrame() returned nil frame without error")
			}
		})
	}
}

func TestFrameRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		msgType   Type
		subsystem Subsystem
		commandID uint8
		data      []byte
	}{
		{
			name:      "SREQ with data",
			msgType:   SREQ,
			subsystem: SYS,
			commandID: 0x01,
			data:      []byte{0xAA, 0xBB, 0xCC},
		},
		{
			name:      "AREQ without data",
			msgType:   AREQ,
			subsystem: ZDO,
			commandID: 0x42,
			data:      []byte{},
		},
		{
			name:      "SRSP with max data",
			msgType:   SRSP,
			subsystem: AF,
			commandID: 0xFF,
			data:      make([]byte, MaxDataSize),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create frame
			frame, err := NewFrame(tt.msgType, tt.subsystem, tt.commandID, tt.data)
			if err != nil {
				t.Fatalf("NewFrame() error = %v", err)
			}

			// Serialize to bytes
			frameBytes := frame.ToBytes()

			// Verify frame structure
			if frameBytes[0] != SOF {
				t.Errorf("SOF byte = 0x%02X, want 0x%02X", frameBytes[0], SOF)
			}
			if frameBytes[PositionDataLength] != uint8(len(tt.data)) {
				t.Errorf("Length = %d, want %d", frameBytes[PositionDataLength], len(tt.data))
			}

			// Parse back
			parsed, err := ParseFrame(frameBytes)
			if err != nil {
				t.Fatalf("ParseFrame() error = %v", err)
			}

			// Verify fields match
			if parsed.Type != tt.msgType {
				t.Errorf("Type = %v, want %v", parsed.Type, tt.msgType)
			}
			if parsed.Subsystem != tt.subsystem {
				t.Errorf("Subsystem = %v, want %v", parsed.Subsystem, tt.subsystem)
			}
			if parsed.CommandID != tt.commandID {
				t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, tt.commandID)
			}
			if !bytes.Equal(parsed.Data, tt.data) {
				t.Errorf("Data = %v, want %v", parsed.Data, tt.data)
			}
		})
	}
}

func TestParseFrame_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name:    "frame too short",
			data:    []byte{0xFE, 0x00, 0x01},
			wantErr: ErrFrameTooShort,
		},
		{
			name:    "invalid SOF",
			data:    []byte{0xFF, 0x00, 0x01, 0x02, 0x03},
			wantErr: ErrInvalidSOF,
		},
		{
			name:    "invalid checksum",
			data:    []byte{0xFE, 0x01, 0x21, 0x01, 0xAA, 0xFF},
			wantErr: ErrInvalidChecksum,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFrame(tt.data)
			if err == nil {
				t.Fatal("ParseFrame() expected error, got nil")
			}
			// Check if error is of expected type (using error wrapping)
			if tt.wantErr != nil && !isErrorType(err, tt.wantErr) {
				t.Errorf("ParseFrame() error = %v, want error type %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateFCS(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint8
	}{
		{
			name: "simple XOR",
			data: []byte{0x01, 0x02, 0x03},
			want: 0x00, // 0x01 ^ 0x02 ^ 0x03 = 0x00
		},
		{
			name: "real frame data",
			data: []byte{0x00, 0x21, 0x01},
			want: 0x20, // 0x00 ^ 0x21 ^ 0x01 = 0x20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateFCS(tt.data)
			if got != tt.want {
				t.Errorf("calculateFCS() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		msgType Type
		want    string
	}{
		{POLL, "POLL"},
		{SREQ, "SREQ"},
		{AREQ, "AREQ"},
		{SRSP, "SRSP"},
		{Type(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.msgType.String()
			if got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubsystemString(t *testing.T) {
	tests := []struct {
		subsystem Subsystem
		want      string
	}{
		{SYS, "SYS"},
		{ZDO, "ZDO"},
		{AF, "AF"},
		{GREENPOWER, "GREENPOWER"},
		{Subsystem(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.subsystem.String()
			if got != tt.want {
				t.Errorf("Subsystem.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// isErrorType checks if err wraps or equals target error
func isErrorType(err, target error) bool {
	if err == target {
		return true
	}
	// Simple check for error message containing target message
	if err != nil && target != nil {
		return bytes.Contains([]byte(err.Error()), []byte(target.Error()))
	}
	return false
}

// TestFrameChecksum verifies checksum calculation for various frame configurations.
func TestFrameChecksum(t *testing.T) {
	tests := []struct {
		name      string
		msgType   Type
		subsystem Subsystem
		commandID uint8
		data      []byte
	}{
		{
			name:      "empty payload",
			msgType:   SREQ,
			subsystem: SYS,
			commandID: 0x01,
			data:      []byte{},
		},
		{
			name:      "single byte payload",
			msgType:   SREQ,
			subsystem: SYS,
			commandID: 0x01,
			data:      []byte{0x00},
		},
		{
			name:      "all zeros payload",
			msgType:   SREQ,
			subsystem: SYS,
			commandID: 0x00,
			data:      []byte{0x00, 0x00, 0x00},
		},
		{
			name:      "all ones payload",
			msgType:   SREQ,
			subsystem: SYS,
			commandID: 0xFF,
			data:      []byte{0xFF, 0xFF, 0xFF},
		},
		{
			name:      "alternating pattern",
			msgType:   AREQ,
			subsystem: ZDO,
			commandID: 0xAA,
			data:      []byte{0xAA, 0x55, 0xAA, 0x55},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := NewFrame(tt.msgType, tt.subsystem, tt.commandID, tt.data)
			if err != nil {
				t.Fatalf("NewFrame() error = %v", err)
			}

			frameBytes := frame.ToBytes()

			// Calculate expected checksum manually
			dataLen := len(tt.data)
			checksumData := frameBytes[PositionDataLength : DataStart+dataLen]
			expectedChecksum := calculateFCS(checksumData)

			// Verify the checksum in the frame matches
			actualChecksum := frameBytes[len(frameBytes)-1]
			if actualChecksum != expectedChecksum {
				t.Errorf("checksum = 0x%02X, want 0x%02X", actualChecksum, expectedChecksum)
			}

			// Verify frame can be parsed back (checksum validation)
			parsed, err := ParseFrame(frameBytes)
			if err != nil {
				t.Errorf("ParseFrame() error = %v (checksum validation failed)", err)
			}
			if parsed != nil && !bytes.Equal(parsed.Data, tt.data) {
				t.Errorf("parsed data = %v, want %v", parsed.Data, tt.data)
			}
		})
	}
}

// TestFrameMaxPayloadSize verifies behavior at payload size boundaries.
func TestFrameMaxPayloadSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "max size exactly",
			size:    MaxDataSize,
			wantErr: false,
		},
		{
			name:    "max size minus one",
			size:    MaxDataSize - 1,
			wantErr: false,
		},
		{
			name:    "max size plus one",
			size:    MaxDataSize + 1,
			wantErr: true,
		},
		{
			name:    "double max size",
			size:    MaxDataSize * 2,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			frame, err := NewFrame(SREQ, SYS, 0x01, data)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && frame != nil {
				// Verify frame can be serialized and parsed
				frameBytes := frame.ToBytes()
				if frameBytes[PositionDataLength] != uint8(tt.size) {
					t.Errorf("length field = %d, want %d", frameBytes[PositionDataLength], tt.size)
				}

				parsed, err := ParseFrame(frameBytes)
				if err != nil {
					t.Errorf("ParseFrame() error = %v", err)
				}
				if len(parsed.Data) != tt.size {
					t.Errorf("parsed data length = %d, want %d", len(parsed.Data), tt.size)
				}
			}
		})
	}
}

// TestFrameEmptyPayload verifies handling of frames with no data payload.
func TestFrameEmptyPayload(t *testing.T) {
	tests := []struct {
		name      string
		msgType   Type
		subsystem Subsystem
		commandID uint8
	}{
		{
			name:      "POLL empty",
			msgType:   POLL,
			subsystem: SYS,
			commandID: 0x00,
		},
		{
			name:      "SREQ empty",
			msgType:   SREQ,
			subsystem: ZDO,
			commandID: 0x42,
		},
		{
			name:      "AREQ empty",
			msgType:   AREQ,
			subsystem: AF,
			commandID: 0xFF,
		},
		{
			name:      "SRSP empty",
			msgType:   SRSP,
			subsystem: UTIL,
			commandID: 0x01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := NewFrame(tt.msgType, tt.subsystem, tt.commandID, []byte{})
			if err != nil {
				t.Fatalf("NewFrame() error = %v", err)
			}

			frameBytes := frame.ToBytes()

			// Verify frame structure
			if len(frameBytes) != MinMessageLength {
				t.Errorf("frame length = %d, want %d", len(frameBytes), MinMessageLength)
			}
			if frameBytes[PositionDataLength] != 0 {
				t.Errorf("data length = %d, want 0", frameBytes[PositionDataLength])
			}

			// Verify checksum is calculated correctly for empty payload
			checksumData := frameBytes[PositionDataLength:DataStart]
			expectedChecksum := calculateFCS(checksumData)
			actualChecksum := frameBytes[len(frameBytes)-1]
			if actualChecksum != expectedChecksum {
				t.Errorf("checksum = 0x%02X, want 0x%02X", actualChecksum, expectedChecksum)
			}

			// Parse and verify
			parsed, err := ParseFrame(frameBytes)
			if err != nil {
				t.Fatalf("ParseFrame() error = %v", err)
			}
			if len(parsed.Data) != 0 {
				t.Errorf("parsed data length = %d, want 0", len(parsed.Data))
			}
			if parsed.Type != tt.msgType {
				t.Errorf("type = %v, want %v", parsed.Type, tt.msgType)
			}
			if parsed.Subsystem != tt.subsystem {
				t.Errorf("subsystem = %v, want %v", parsed.Subsystem, tt.subsystem)
			}
			if parsed.CommandID != tt.commandID {
				t.Errorf("commandID = 0x%02X, want 0x%02X", parsed.CommandID, tt.commandID)
			}
		})
	}
}

// TestParseFrameInvalidLength verifies error handling for length mismatches.
func TestParseFrameInvalidLength(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name: "length field larger than actual data",
			data: []byte{
				0xFE,       // SOF
				0x05,       // Length = 5
				0x21,       // Cmd0
				0x01,       // Cmd1
				0xAA, 0xBB, // Only 2 data bytes instead of 5
				0x00, // FCS (incorrect)
			},
			wantErr: ErrFrameTooShort,
		},
		{
			name: "zero length but has data bytes",
			data: []byte{
				0xFE, // SOF
				0x00, // Length = 0
				0x21, // Cmd0
				0x01, // Cmd1
				0x20, // FCS
			},
			wantErr: nil, // Should parse successfully as zero-length frame
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFrame(tt.data)
			if tt.wantErr == nil && err != nil {
				t.Errorf("ParseFrame() unexpected error = %v", err)
			}
			if tt.wantErr != nil && err == nil {
				t.Error("ParseFrame() expected error, got nil")
			}
			if tt.wantErr != nil && err != nil && !isErrorType(err, tt.wantErr) {
				t.Errorf("ParseFrame() error = %v, want error type %v", err, tt.wantErr)
			}
		})
	}
}

// TestFrameTypeAndSubsystemEncoding verifies correct encoding of all type/subsystem combinations.
func TestFrameTypeAndSubsystemEncoding(t *testing.T) {
	types := []Type{POLL, SREQ, AREQ, SRSP}
	subsystems := []Subsystem{SYS, MAC, NWK, AF, ZDO, SAPI, UTIL, DEBUG, APP, GREENPOWER}

	for _, msgType := range types {
		for _, subsystem := range subsystems {
			t.Run(msgType.String()+"-"+subsystem.String(), func(t *testing.T) {
				frame, err := NewFrame(msgType, subsystem, 0x42, []byte{0xAA})
				if err != nil {
					t.Fatalf("NewFrame() error = %v", err)
				}

				frameBytes := frame.ToBytes()

				// Verify Cmd0 encoding: (Type << 5) | (Subsystem & 0x1F)
				cmd0 := frameBytes[PositionCmd0]
				decodedType := Type(cmd0 >> 5)
				decodedSubsystem := Subsystem(cmd0 & 0x1F)

				if decodedType != msgType {
					t.Errorf("decoded type = %v, want %v", decodedType, msgType)
				}
				if decodedSubsystem != subsystem {
					t.Errorf("decoded subsystem = %v, want %v", decodedSubsystem, subsystem)
				}

				// Verify round-trip
				parsed, err := ParseFrame(frameBytes)
				if err != nil {
					t.Fatalf("ParseFrame() error = %v", err)
				}
				if parsed.Type != msgType {
					t.Errorf("parsed type = %v, want %v", parsed.Type, msgType)
				}
				if parsed.Subsystem != subsystem {
					t.Errorf("parsed subsystem = %v, want %v", parsed.Subsystem, subsystem)
				}
			})
		}
	}
}

// TestCalculateFCSEdgeCases verifies checksum calculation for edge cases.
func TestCalculateFCSEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint8
	}{
		{
			name: "empty data",
			data: []byte{},
			want: 0x00,
		},
		{
			name: "single zero",
			data: []byte{0x00},
			want: 0x00,
		},
		{
			name: "single byte",
			data: []byte{0x42},
			want: 0x42,
		},
		{
			name: "two identical bytes",
			data: []byte{0xFF, 0xFF},
			want: 0x00, // 0xFF ^ 0xFF = 0x00
		},
		{
			name: "three identical bytes",
			data: []byte{0xAA, 0xAA, 0xAA},
			want: 0xAA, // 0xAA ^ 0xAA ^ 0xAA = 0xAA
		},
		{
			name: "max values",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF},
			want: 0x00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateFCS(tt.data)
			if got != tt.want {
				t.Errorf("calculateFCS() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

package unpi

import (
	"errors"
	"fmt"
)

// Sentinel errors for frame operations.
var (
	// ErrInvalidChecksum indicates the frame checksum validation failed.
	ErrInvalidChecksum = errors.New("invalid frame checksum")
	// ErrInvalidSOF indicates the frame does not start with the SOF byte.
	ErrInvalidSOF = errors.New("invalid start of frame")
	// ErrDataTooLarge indicates the data payload exceeds MaxDataSize.
	ErrDataTooLarge = errors.New("data size exceeds maximum")
	// ErrFrameTooShort indicates the frame is shorter than MinMessageLength.
	ErrFrameTooShort = errors.New("frame too short")
)

// Frame represents a UNPI protocol frame.
type Frame struct {
	// Type is the message type (POLL, SREQ, AREQ, SRSP).
	Type Type
	// Subsystem identifies the target subsystem.
	Subsystem Subsystem
	// CommandID is the command identifier within the subsystem.
	CommandID uint8
	// Data is the payload data for the command.
	Data []byte
}

// NewFrame creates a new Frame with the specified parameters.
// It validates that the data size does not exceed MaxDataSize.
func NewFrame(msgType Type, subsystem Subsystem, commandID uint8, data []byte) (*Frame, error) {
	if len(data) > MaxDataSize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrDataTooLarge, len(data))
	}

	return &Frame{
		Type:      msgType,
		Subsystem: subsystem,
		CommandID: commandID,
		Data:      data,
	}, nil
}

// ToBytes serializes the Frame to its wire format representation.
// Format: [SOF][Len][Cmd0][Cmd1][Data...][FCS]
// where Cmd0 encodes Type and Subsystem, Cmd1 is CommandID.
func (f *Frame) ToBytes() []byte {
	dataLen := len(f.Data)
	frameSize := MinMessageLength + dataLen
	frame := make([]byte, frameSize)

	// SOF byte
	frame[0] = SOF

	// Data length
	frame[PositionDataLength] = uint8(dataLen)

	// Cmd0: (Type << 5) | (Subsystem & 0x1F)
	frame[PositionCmd0] = (uint8(f.Type) << 5) | (uint8(f.Subsystem) & 0x1F)

	// Cmd1: CommandID
	frame[PositionCmd1] = f.CommandID

	// Data payload
	copy(frame[DataStart:], f.Data)

	// FCS (checksum)
	fcs := calculateFCS(frame[PositionDataLength : DataStart+dataLen])
	frame[frameSize-1] = fcs

	return frame
}

// ParseFrame parses and validates a UNPI frame from raw bytes.
// It validates the SOF byte, frame length, and checksum.
func ParseFrame(data []byte) (*Frame, error) {
	if len(data) < MinMessageLength {
		return nil, fmt.Errorf("%w: got %d bytes", ErrFrameTooShort, len(data))
	}

	// Validate SOF
	if data[0] != SOF {
		return nil, fmt.Errorf("%w: expected 0x%02X, got 0x%02X", ErrInvalidSOF, SOF, data[0])
	}

	// Extract data length
	dataLen := int(data[PositionDataLength])

	// Validate total frame length
	expectedLen := MinMessageLength + dataLen
	if len(data) < expectedLen {
		return nil, fmt.Errorf("%w: expected %d bytes for data length %d, got %d bytes",
			ErrFrameTooShort, expectedLen, dataLen, len(data))
	}

	// Validate checksum
	fcsPos := DataStart + dataLen
	expectedFCS := calculateFCS(data[PositionDataLength:fcsPos])
	actualFCS := data[fcsPos]
	if actualFCS != expectedFCS {
		return nil, fmt.Errorf("%w: expected 0x%02X, got 0x%02X", ErrInvalidChecksum, expectedFCS, actualFCS)
	}

	// Decode Cmd0 to extract Type and Subsystem
	cmd0 := data[PositionCmd0]
	msgType := Type(cmd0 >> 5)
	subsystem := Subsystem(cmd0 & 0x1F)

	// Extract CommandID
	commandID := data[PositionCmd1]

	// Extract payload data
	payload := make([]byte, dataLen)
	copy(payload, data[DataStart:fcsPos])

	return &Frame{
		Type:      msgType,
		Subsystem: subsystem,
		CommandID: commandID,
		Data:      payload,
	}, nil
}

// calculateFCS computes the XOR checksum of the provided bytes.
// The checksum is calculated from the Length field through the Data field.
func calculateFCS(data []byte) uint8 {
	var fcs uint8
	for _, b := range data {
		fcs ^= b
	}
	return fcs
}

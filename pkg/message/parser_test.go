package message

import (
	"testing"
)

func TestParserNew(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser returned nil")
	}
}

func TestDeviceIdentifier(t *testing.T) {
	identifier := DeviceIdentifier{
		IEEEAddr: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		NwkAddr:  0x1234,
	}

	if identifier.IEEEAddr[0] != 0x01 {
		t.Error("IEEEAddr not set correctly")
	}

	if identifier.NwkAddr != 0x1234 {
		t.Error("NwkAddr not set correctly")
	}
}

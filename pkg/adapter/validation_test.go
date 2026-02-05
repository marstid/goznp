package adapter

import (
	"testing"
)

func TestValidateIEEEAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    [8]byte
		wantErr bool
	}{
		{
			name:    "valid address",
			addr:    [8]byte{0x00, 0x12, 0x4B, 0x00, 0x01, 0x02, 0x03, 0x04},
			wantErr: false,
		},
		{
			name:    "all zeros",
			addr:    [8]byte{},
			wantErr: true,
		},
		{
			name:    "broadcast address",
			addr:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIEEEAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIEEEAddr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint uint8
		wantErr  bool
	}{
		{
			name:     "valid endpoint 1",
			endpoint: 1,
			wantErr:  false,
		},
		{
			name:     "valid endpoint 240",
			endpoint: 240,
			wantErr:  false,
		},
		{
			name:     "invalid endpoint 0",
			endpoint: 0,
			wantErr:  true,
		},
		{
			name:     "invalid endpoint 241",
			endpoint: 241,
			wantErr:  true,
		},
		{
			name:     "invalid endpoint 255",
			endpoint: 255,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEndpoint(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEndpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNetworkAddress(t *testing.T) {
	tests := []struct {
		name    string
		nwkAddr uint16
		wantErr bool
	}{
		{
			name:    "valid address",
			nwkAddr: 0x1234,
			wantErr: false,
		},
		{
			name:    "valid address 0x0000",
			nwkAddr: 0x0000,
			wantErr: false,
		},
		{
			name:    "broadcast address 0xFFFF",
			nwkAddr: 0xFFFF,
			wantErr: true,
		},
		{
			name:    "invalid address 0xFFFE",
			nwkAddr: 0xFFFE,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNetworkAddress(tt.nwkAddr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNetworkAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package adapter

import (
	"testing"

	"github.com/marstid/goznp/pkg/zcl"
)

// GetFirstResult Tests

func TestGetFirstResult(t *testing.T) {
	tests := []struct {
		name    string
		results []AttributeResult
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty results",
			results: []AttributeResult{},
			wantErr: true,
			errMsg:  "no attributes returned",
		},
		{
			name: "success status",
			results: []AttributeResult{
				{
					AttributeID: 0x0000,
					Status:      zcl.StatusSuccess,
					Value:       uint8(42),
				},
			},
			wantErr: false,
		},
		{
			name: "failure status",
			results: []AttributeResult{
				{
					AttributeID: 0x0000,
					Status:      zcl.StatusFailure,
					Value:       nil,
				},
			},
			wantErr: true,
			errMsg:  "attribute read returned status 0x01",
		},
		{
			name: "unsupported attribute status",
			results: []AttributeResult{
				{
					AttributeID: 0x0000,
					Status:      zcl.StatusUnsupportedAttribute,
					Value:       nil,
				},
			},
			wantErr: true,
			errMsg:  "attribute read returned status 0x86",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetFirstResult(tt.results)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFirstResult() expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("GetFirstResult() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetFirstResult() unexpected error: %v", err)
				}
				if result == nil {
					t.Errorf("GetFirstResult() returned nil result")
				} else if result.Status != zcl.StatusSuccess {
					t.Errorf("GetFirstResult() status = 0x%02X, want 0x%02X", result.Status, zcl.StatusSuccess)
				}
			}
		})
	}
}

// Value Extraction Tests

func TestExtractUint8(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  uint8
		ok    bool
	}{
		{"uint8", uint8(42), 42, true},
		{"uint16 in range", uint16(100), 100, true},
		{"uint16 out of range", uint16(300), 0, false},
		{"int in range", int(50), 50, true},
		{"int negative", int(-1), 0, false},
		{"int out of range", int(256), 0, false},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractUint8(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractUint8(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractUint8(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractUint16(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  uint16
		ok    bool
	}{
		{"uint16", uint16(1000), 1000, true},
		{"int16", int16(500), 500, true},
		{"uint8", uint8(42), 42, true},
		{"int", int(1234), 1234, true},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractUint16(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractUint16(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractUint16(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractInt16(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int16
		ok    bool
	}{
		{"int16 positive", int16(1000), 1000, true},
		{"int16 negative", int16(-500), -500, true},
		{"uint16", uint16(500), 500, true},
		{"int", int(-1234), -1234, true},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractInt16(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractInt16(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractInt16(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractInt8(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int8
		ok    bool
	}{
		{"int8 positive", int8(42), 42, true},
		{"int8 negative", int8(-42), -42, true},
		{"uint8", uint8(100), 100, true},
		{"int", int(-50), -50, true},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractInt8(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractInt8(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractInt8(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractUint32(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  uint32
		ok    bool
	}{
		{"uint32", uint32(100000), 100000, true},
		{"int32", int32(50000), 50000, true},
		{"uint16", uint16(1000), 1000, true},
		{"int", int(123456), 123456, true},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractUint32(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractUint32(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractUint32(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractInt32(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int32
		ok    bool
	}{
		{"int32 positive", int32(100000), 100000, true},
		{"int32 negative", int32(-50000), -50000, true},
		{"uint32", uint32(50000), 50000, true},
		{"int", int(-123456), -123456, true},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractInt32(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractInt32(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractInt32(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractUint48(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  uint64
		ok    bool
	}{
		{"uint64", uint64(1000000000), 1000000000, true},
		{"int64", int64(500000000), 500000000, true},
		{"uint32", uint32(100000), 100000, true},
		{"int", int(123456), 123456, true},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractUint48(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractUint48(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractUint48(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractBool(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
		ok    bool
	}{
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"uint8 zero", uint8(0), false, true},
		{"uint8 non-zero", uint8(1), true, true},
		{"uint8 large", uint8(255), true, true},
		{"uint16 zero", uint16(0), false, true},
		{"uint16 non-zero", uint16(1), true, true},
		{"int zero", int(0), false, true},
		{"int non-zero", int(1), true, true},
		{"string", "test", false, false},
		{"nil", nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractBool(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractBool(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractBool(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
		ok    bool
	}{
		{"string", "hello", "hello", true},
		{"empty string", "", "", true},
		{"uint8", uint8(42), "", false},
		{"int", int(123), "", false},
		{"nil", nil, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractString(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractString(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractString(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestExtractFloat32(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  float32
		ok    bool
	}{
		{"float32", float32(3.14), 3.14, true},
		{"float32 zero", float32(0), 0, true},
		{"float32 negative", float32(-2.5), -2.5, true},
		{"int", int(123), 0, false},
		{"string", "test", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractFloat32(tt.value)
			if ok != tt.ok {
				t.Errorf("ExtractFloat32(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ExtractFloat32(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

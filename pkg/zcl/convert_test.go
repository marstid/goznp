package zcl

import "testing"

func TestToUint8(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    uint8
		wantOk  bool
	}{
		{"valid uint8", uint8(42), 42, true},
		{"zero uint8", uint8(0), 0, true},
		{"max uint8", uint8(255), 255, true},
		{"wrong type uint16", uint16(42), 0, false},
		{"wrong type int", int(42), 0, false},
		{"wrong type string", "42", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToUint8(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ToUint8() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ToUint8() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToUint16(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   uint16
		wantOk bool
	}{
		{"valid uint16", uint16(1000), 1000, true},
		{"zero uint16", uint16(0), 0, true},
		{"max uint16", uint16(65535), 65535, true},
		{"wrong type uint8", uint8(42), 0, false},
		{"wrong type int", int(1000), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToUint16(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ToUint16() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ToUint16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToInt16(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   int16
		wantOk bool
	}{
		{"positive int16", int16(1000), 1000, true},
		{"negative int16", int16(-500), -500, true},
		{"zero int16", int16(0), 0, true},
		{"wrong type uint16", uint16(1000), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToInt16(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ToInt16() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ToInt16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   bool
		wantOk bool
	}{
		{"true", true, true, true},
		{"false", false, false, true},
		{"wrong type uint8", uint8(1), false, false},
		{"wrong type string", "true", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToBool(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ToBool() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ToBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToBoolFromUint8(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   bool
		wantOk bool
	}{
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"uint8 zero", uint8(0), false, true},
		{"uint8 one", uint8(1), true, true},
		{"uint8 nonzero", uint8(255), true, true},
		{"wrong type string", "1", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToBoolFromUint8(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ToBoolFromUint8() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ToBoolFromUint8() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   string
		wantOk bool
	}{
		{"valid string", "hello", "hello", true},
		{"empty string", "", "", true},
		{"wrong type int", 42, "", false},
		{"wrong type []byte", []byte("hello"), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToString(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ToString() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustUint8(t *testing.T) {
	if got := MustUint8(uint8(42), 0); got != 42 {
		t.Errorf("MustUint8() = %v, want 42", got)
	}
	if got := MustUint8("invalid", 99); got != 99 {
		t.Errorf("MustUint8() with default = %v, want 99", got)
	}
}

func TestMustUint16(t *testing.T) {
	if got := MustUint16(uint16(1000), 0); got != 1000 {
		t.Errorf("MustUint16() = %v, want 1000", got)
	}
	if got := MustUint16("invalid", 999); got != 999 {
		t.Errorf("MustUint16() with default = %v, want 999", got)
	}
}

func TestMustBool(t *testing.T) {
	if got := MustBool(true, false); got != true {
		t.Errorf("MustBool(true) = %v, want true", got)
	}
	if got := MustBool(uint8(1), false); got != true {
		t.Errorf("MustBool(uint8(1)) = %v, want true", got)
	}
	if got := MustBool(uint8(0), true); got != false {
		t.Errorf("MustBool(uint8(0)) = %v, want false", got)
	}
	if got := MustBool("invalid", true); got != true {
		t.Errorf("MustBool() with default = %v, want true", got)
	}
}

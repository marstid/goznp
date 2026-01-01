package zcl

import (
	"math"
	"reflect"
	"testing"
)

func TestReadValueIntegers(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		// Uint8
		{"uint8", []byte{0x42}, TypeUint8, uint8(0x42), 1, false},
		{"uint8 zero", []byte{0x00}, TypeUint8, uint8(0x00), 1, false},
		{"uint8 max", []byte{0xFF}, TypeUint8, uint8(0xFF), 1, false},
		{"uint8 insufficient", []byte{}, TypeUint8, nil, 0, true},

		// Uint16
		{"uint16", []byte{0x34, 0x12}, TypeUint16, uint16(0x1234), 2, false},
		{"uint16 zero", []byte{0x00, 0x00}, TypeUint16, uint16(0x0000), 2, false},
		{"uint16 max", []byte{0xFF, 0xFF}, TypeUint16, uint16(0xFFFF), 2, false},
		{"uint16 insufficient", []byte{0x34}, TypeUint16, nil, 0, true},

		// Uint32
		{"uint32", []byte{0x78, 0x56, 0x34, 0x12}, TypeUint32, uint32(0x12345678), 4, false},
		{"uint32 zero", []byte{0x00, 0x00, 0x00, 0x00}, TypeUint32, uint32(0x00000000), 4, false},
		{"uint32 max", []byte{0xFF, 0xFF, 0xFF, 0xFF}, TypeUint32, uint32(0xFFFFFFFF), 4, false},
		{"uint32 insufficient", []byte{0x78, 0x56, 0x34}, TypeUint32, nil, 0, true},

		// Int8
		{"int8 positive", []byte{0x42}, TypeInt8, int8(0x42), 1, false},
		{"int8 negative", []byte{0xFF}, TypeInt8, int8(-1), 1, false},
		{"int8 zero", []byte{0x00}, TypeInt8, int8(0x00), 1, false},
		{"int8 min", []byte{0x80}, TypeInt8, int8(-128), 1, false},
		{"int8 max", []byte{0x7F}, TypeInt8, int8(127), 1, false},
		{"int8 insufficient", []byte{}, TypeInt8, nil, 0, true},

		// Int16
		{"int16 positive", []byte{0x34, 0x12}, TypeInt16, int16(0x1234), 2, false},
		{"int16 negative", []byte{0xFF, 0xFF}, TypeInt16, int16(-1), 2, false},
		{"int16 zero", []byte{0x00, 0x00}, TypeInt16, int16(0x0000), 2, false},
		{"int16 min", []byte{0x00, 0x80}, TypeInt16, int16(-32768), 2, false},
		{"int16 max", []byte{0xFF, 0x7F}, TypeInt16, int16(32767), 2, false},
		{"int16 insufficient", []byte{0x34}, TypeInt16, nil, 0, true},

		// Int32
		{"int32 positive", []byte{0x78, 0x56, 0x34, 0x12}, TypeInt32, int32(0x12345678), 4, false},
		{"int32 negative", []byte{0xFF, 0xFF, 0xFF, 0xFF}, TypeInt32, int32(-1), 4, false},
		{"int32 zero", []byte{0x00, 0x00, 0x00, 0x00}, TypeInt32, int32(0x00000000), 4, false},
		{"int32 min", []byte{0x00, 0x00, 0x00, 0x80}, TypeInt32, int32(-2147483648), 4, false},
		{"int32 max", []byte{0xFF, 0xFF, 0xFF, 0x7F}, TypeInt32, int32(2147483647), 4, false},
		{"int32 insufficient", []byte{0x78, 0x56, 0x34}, TypeInt32, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueBoolean(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		{"true (0x01)", []byte{0x01}, true, 1, false},
		{"true (0xFF)", []byte{0xFF}, true, 1, false},
		{"true (0x42)", []byte{0x42}, true, 1, false},
		{"false (0x00)", []byte{0x00}, false, 1, false},
		{"insufficient", []byte{}, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, TypeBoolean)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got != tt.want {
					t.Errorf("ReadValue() got = %v, want %v", got, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueStrings(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		// CharString
		{"char empty", []byte{0x00}, TypeCharString, "", 1, false},
		{"char normal", []byte{0x05, 'h', 'e', 'l', 'l', 'o'}, TypeCharString, "hello", 6, false},
		{"char invalid (0xFF)", []byte{0xFF}, TypeCharString, nil, 1, false},
		{"char insufficient length", []byte{}, TypeCharString, nil, 0, true},
		{"char insufficient data", []byte{0x05, 'h', 'i'}, TypeCharString, nil, 0, true},

		// OctetString
		{"octet empty", []byte{0x00}, TypeOctetString, []byte{}, 1, false},
		{"octet normal", []byte{0x03, 0x01, 0x02, 0x03}, TypeOctetString, []byte{0x01, 0x02, 0x03}, 4, false},
		{"octet invalid (0xFF)", []byte{0xFF}, TypeOctetString, nil, 1, false},
		{"octet insufficient length", []byte{}, TypeOctetString, nil, 0, true},
		{"octet insufficient data", []byte{0x05, 0x01, 0x02}, TypeOctetString, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %v, want %v", got, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueFloats(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		// Float32
		{"float32 zero", []byte{0x00, 0x00, 0x00, 0x00}, TypeSingle, float32(0.0), 4, false},
		{"float32 one", []byte{0x00, 0x00, 0x80, 0x3F}, TypeSingle, float32(1.0), 4, false},
		{"float32 pi", []byte{0xDB, 0x0F, 0x49, 0x40}, TypeSingle, float32(3.14159265), 4, false},
		{"float32 insufficient", []byte{0x00, 0x00, 0x80}, TypeSingle, nil, 0, true},

		// Float64
		{"float64 zero", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, TypeDouble, float64(0.0), 8, false},
		{"float64 one", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F}, TypeDouble, float64(1.0), 8, false},
		{"float64 insufficient", []byte{0x00, 0x00, 0x00, 0x00}, TypeDouble, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.dataType == TypeSingle {
					gotFloat := got.(float32)
					wantFloat := tt.want.(float32)
					if math.Abs(float64(gotFloat-wantFloat)) > 0.0001 {
						t.Errorf("ReadValue() got = %v, want %v", got, tt.want)
					}
				} else {
					gotFloat := got.(float64)
					wantFloat := tt.want.(float64)
					if math.Abs(gotFloat-wantFloat) > 0.0000001 {
						t.Errorf("ReadValue() got = %v, want %v", got, tt.want)
					}
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueExtendedIntegers(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		// Uint24
		{"uint24", []byte{0x78, 0x56, 0x34}, TypeUint24, uint32(0x345678), 3, false},
		{"uint24 max", []byte{0xFF, 0xFF, 0xFF}, TypeUint24, uint32(0xFFFFFF), 3, false},
		{"uint24 insufficient", []byte{0x78, 0x56}, TypeUint24, nil, 0, true},

		// Uint64
		{"uint64", []byte{0xEF, 0xCD, 0xAB, 0x90, 0x78, 0x56, 0x34, 0x12}, TypeUint64, uint64(0x1234567890ABCDEF), 8, false},
		{"uint64 insufficient", []byte{0xEF, 0xCD, 0xAB, 0x90, 0x78, 0x56, 0x34}, TypeUint64, nil, 0, true},

		// Int24 with sign extension
		{"int24 positive", []byte{0x78, 0x56, 0x34}, TypeInt24, int32(0x345678), 3, false},
		{"int24 negative", []byte{0xFF, 0xFF, 0xFF}, TypeInt24, int32(-1), 3, false},
		{"int24 insufficient", []byte{0x78, 0x56}, TypeInt24, nil, 0, true},

		// Int64
		{"int64 positive", []byte{0x78, 0x56, 0x34, 0x12, 0x00, 0x00, 0x00, 0x00}, TypeInt64, int64(0x12345678), 8, false},
		{"int64 negative", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, TypeInt64, int64(-1), 8, false},
		{"int64 insufficient", []byte{0x78, 0x56, 0x34, 0x12, 0x00, 0x00, 0x00}, TypeInt64, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueLongStrings(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		// LongChar
		{"long char empty", []byte{0x00, 0x00}, TypeLongChar, "", 2, false},
		{"long char normal", []byte{0x05, 0x00, 'h', 'e', 'l', 'l', 'o'}, TypeLongChar, "hello", 7, false},
		{"long char invalid (0xFFFF)", []byte{0xFF, 0xFF}, TypeLongChar, nil, 2, false},
		{"long char insufficient length", []byte{0x05}, TypeLongChar, nil, 0, true},
		{"long char insufficient data", []byte{0x05, 0x00, 'h', 'i'}, TypeLongChar, nil, 0, true},

		// LongOctet
		{"long octet empty", []byte{0x00, 0x00}, TypeLongOctet, []byte{}, 2, false},
		{"long octet normal", []byte{0x03, 0x00, 0x01, 0x02, 0x03}, TypeLongOctet, []byte{0x01, 0x02, 0x03}, 5, false},
		{"long octet invalid (0xFFFF)", []byte{0xFF, 0xFF}, TypeLongOctet, nil, 2, false},
		{"long octet insufficient length", []byte{0x05}, TypeLongOctet, nil, 0, true},
		{"long octet insufficient data", []byte{0x05, 0x00, 0x01, 0x02}, TypeLongOctet, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %v, want %v", got, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueSpecialTypes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		// NoData
		{"nodata", []byte{}, TypeNoData, nil, 0, false},
		{"nodata with extra", []byte{0x01, 0x02}, TypeNoData, nil, 0, false},

		// IEEEAddr
		{"ieee addr", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, TypeIEEEAddr, [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, 8, false},
		{"ieee addr insufficient", []byte{0x01, 0x02, 0x03, 0x04}, TypeIEEEAddr, nil, 0, true},

		// SecKey
		{"seckey", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}, TypeSecKey,
			[16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}, 16, false},
		{"seckey insufficient", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, TypeSecKey, nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %v, want %v", got, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueArray(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		{
			name:     "array empty",
			data:     []byte{0x20, 0x00, 0x00}, // TypeUint8, count=0
			dataType: TypeArray,
			want:     &ArrayValue{ElementType: TypeUint8, Elements: []interface{}{}},
			wantN:    3,
			wantErr:  false,
		},
		{
			name:     "array uint8",
			data:     []byte{0x20, 0x03, 0x00, 0x01, 0x02, 0x03}, // TypeUint8, count=3, [1,2,3]
			dataType: TypeArray,
			want:     &ArrayValue{ElementType: TypeUint8, Elements: []interface{}{uint8(0x01), uint8(0x02), uint8(0x03)}},
			wantN:    6,
			wantErr:  false,
		},
		{
			name:     "array invalid (0xFFFF count)",
			data:     []byte{0x20, 0xFF, 0xFF}, // TypeUint8, count=0xFFFF (invalid)
			dataType: TypeArray,
			want:     &ArrayValue{ElementType: TypeUint8, Elements: nil},
			wantN:    3,
			wantErr:  false,
		},
		{
			name:     "array insufficient header",
			data:     []byte{0x20, 0x03}, // Missing full header
			dataType: TypeArray,
			want:     nil,
			wantN:    0,
			wantErr:  true,
		},
		{
			name:     "array insufficient elements",
			data:     []byte{0x20, 0x03, 0x00, 0x01}, // TypeUint8, count=3, only 1 element
			dataType: TypeArray,
			want:     nil,
			wantN:    0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %+v, want %+v", got, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueStruct(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dataType DataType
		want     interface{}
		wantN    int
		wantErr  bool
	}{
		{
			name:     "struct empty",
			data:     []byte{0x00, 0x00}, // count=0
			dataType: TypeStruct,
			want:     &StructValue{Fields: []StructField{}},
			wantN:    2,
			wantErr:  false,
		},
		{
			name:     "struct single field",
			data:     []byte{0x01, 0x00, 0x20, 0x42}, // count=1, TypeUint8, value=0x42
			dataType: TypeStruct,
			want:     &StructValue{Fields: []StructField{{Type: TypeUint8, Value: uint8(0x42)}}},
			wantN:    4,
			wantErr:  false,
		},
		{
			name:     "struct multiple fields",
			data:     []byte{0x02, 0x00, 0x20, 0x42, 0x10, 0x01}, // count=2, TypeUint8=0x42, TypeBoolean=true
			dataType: TypeStruct,
			want: &StructValue{Fields: []StructField{
				{Type: TypeUint8, Value: uint8(0x42)},
				{Type: TypeBoolean, Value: true},
			}},
			wantN:   6,
			wantErr: false,
		},
		{
			name:     "struct invalid (0xFFFF count)",
			data:     []byte{0xFF, 0xFF}, // count=0xFFFF (invalid)
			dataType: TypeStruct,
			want:     &StructValue{Fields: nil},
			wantN:    2,
			wantErr:  false,
		},
		{
			name:     "struct insufficient header",
			data:     []byte{0x01}, // Missing full header
			dataType: TypeStruct,
			want:     nil,
			wantN:    0,
			wantErr:  true,
		},
		{
			name:     "struct insufficient field data",
			data:     []byte{0x01, 0x00, 0x20}, // count=1, TypeUint8, but no value
			dataType: TypeStruct,
			want:     nil,
			wantN:    0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := ReadValue(tt.data, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReadValue() got = %+v, want %+v", got, tt.want)
				}
				if n != tt.wantN {
					t.Errorf("ReadValue() n = %v, want %v", n, tt.wantN)
				}
			}
		})
	}
}

func TestReadValueUnsupportedType(t *testing.T) {
	// Test with an unsupported/unknown data type
	data := []byte{0x01, 0x02, 0x03}
	unknownType := DataType(0xFF)

	_, _, err := ReadValue(data, unknownType)
	if err == nil {
		t.Error("ReadValue() expected error for unsupported type, got nil")
	}
}

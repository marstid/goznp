package zcl

import (
	"encoding/binary"
	"fmt"
	"math"
)

// DataType defines ZCL data types.
type DataType uint8

const (
	TypeNoData      DataType = 0x00
	TypeData8       DataType = 0x08
	TypeData16      DataType = 0x09
	TypeData24      DataType = 0x0A
	TypeData32      DataType = 0x0B
	TypeData40      DataType = 0x0C
	TypeData48      DataType = 0x0D
	TypeData56      DataType = 0x0E
	TypeData64      DataType = 0x0F
	TypeBoolean     DataType = 0x10
	TypeBitmap8     DataType = 0x18
	TypeBitmap16    DataType = 0x19
	TypeBitmap24    DataType = 0x1A
	TypeBitmap32    DataType = 0x1B
	TypeBitmap40    DataType = 0x1C
	TypeBitmap48    DataType = 0x1D
	TypeBitmap56    DataType = 0x1E
	TypeBitmap64    DataType = 0x1F
	TypeUint8       DataType = 0x20
	TypeUint16      DataType = 0x21
	TypeUint24      DataType = 0x22
	TypeUint32      DataType = 0x23
	TypeUint40      DataType = 0x24
	TypeUint48      DataType = 0x25
	TypeUint56      DataType = 0x26
	TypeUint64      DataType = 0x27
	TypeInt8        DataType = 0x28
	TypeInt16       DataType = 0x29
	TypeInt24       DataType = 0x2A
	TypeInt32       DataType = 0x2B
	TypeInt40       DataType = 0x2C
	TypeInt48       DataType = 0x2D
	TypeInt56       DataType = 0x2E
	TypeInt64       DataType = 0x2F
	TypeEnum8       DataType = 0x30
	TypeEnum16      DataType = 0x31
	TypeSingle      DataType = 0x39 // float32
	TypeDouble      DataType = 0x3A // float64
	TypeOctetString DataType = 0x41
	TypeCharString  DataType = 0x42
	TypeLongOctet   DataType = 0x43 // Long octet string (2-byte length)
	TypeLongChar    DataType = 0x44 // Long char string (2-byte length)
	TypeArray       DataType = 0x48
	TypeStruct      DataType = 0x4C
	TypeSet         DataType = 0x50
	TypeBag         DataType = 0x51
	TypeTimeOfDay   DataType = 0xE0
	TypeDate        DataType = 0xE1
	TypeUTCTime     DataType = 0xE2
	TypeClusterID   DataType = 0xE8
	TypeAttrID      DataType = 0xE9
	TypeIEEEAddr    DataType = 0xF0
	TypeSecKey      DataType = 0xF1
)

// String returns human-readable type name.
func (t DataType) String() string {
	switch t {
	case TypeNoData:
		return "NoData"
	case TypeData8:
		return "Data8"
	case TypeData16:
		return "Data16"
	case TypeData24:
		return "Data24"
	case TypeData32:
		return "Data32"
	case TypeData40:
		return "Data40"
	case TypeData48:
		return "Data48"
	case TypeData56:
		return "Data56"
	case TypeData64:
		return "Data64"
	case TypeBoolean:
		return "Boolean"
	case TypeBitmap8:
		return "Bitmap8"
	case TypeBitmap16:
		return "Bitmap16"
	case TypeBitmap24:
		return "Bitmap24"
	case TypeBitmap32:
		return "Bitmap32"
	case TypeBitmap40:
		return "Bitmap40"
	case TypeBitmap48:
		return "Bitmap48"
	case TypeBitmap56:
		return "Bitmap56"
	case TypeBitmap64:
		return "Bitmap64"
	case TypeUint8:
		return "Uint8"
	case TypeUint16:
		return "Uint16"
	case TypeUint24:
		return "Uint24"
	case TypeUint32:
		return "Uint32"
	case TypeUint40:
		return "Uint40"
	case TypeUint48:
		return "Uint48"
	case TypeUint56:
		return "Uint56"
	case TypeUint64:
		return "Uint64"
	case TypeInt8:
		return "Int8"
	case TypeInt16:
		return "Int16"
	case TypeInt24:
		return "Int24"
	case TypeInt32:
		return "Int32"
	case TypeInt40:
		return "Int40"
	case TypeInt48:
		return "Int48"
	case TypeInt56:
		return "Int56"
	case TypeInt64:
		return "Int64"
	case TypeEnum8:
		return "Enum8"
	case TypeEnum16:
		return "Enum16"
	case TypeSingle:
		return "Single"
	case TypeDouble:
		return "Double"
	case TypeOctetString:
		return "OctetString"
	case TypeCharString:
		return "CharString"
	case TypeLongOctet:
		return "LongOctetString"
	case TypeLongChar:
		return "LongCharString"
	case TypeArray:
		return "Array"
	case TypeStruct:
		return "Struct"
	case TypeSet:
		return "Set"
	case TypeBag:
		return "Bag"
	case TypeTimeOfDay:
		return "TimeOfDay"
	case TypeDate:
		return "Date"
	case TypeUTCTime:
		return "UTCTime"
	case TypeClusterID:
		return "ClusterID"
	case TypeAttrID:
		return "AttrID"
	case TypeIEEEAddr:
		return "IEEEAddr"
	case TypeSecKey:
		return "SecKey"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", uint8(t))
	}
}

// Size returns the fixed size of the type, or 0 for variable-length types.
func (t DataType) Size() int {
	switch t {
	case TypeNoData:
		return 0
	case TypeData8, TypeBoolean, TypeBitmap8, TypeUint8, TypeInt8, TypeEnum8:
		return 1
	case TypeData16, TypeBitmap16, TypeUint16, TypeInt16, TypeEnum16, TypeClusterID, TypeAttrID:
		return 2
	case TypeData24, TypeBitmap24, TypeUint24, TypeInt24:
		return 3
	case TypeData32, TypeBitmap32, TypeUint32, TypeInt32, TypeSingle, TypeTimeOfDay, TypeDate, TypeUTCTime:
		return 4
	case TypeData40, TypeBitmap40, TypeUint40, TypeInt40:
		return 5
	case TypeData48, TypeBitmap48, TypeUint48, TypeInt48:
		return 6
	case TypeData56, TypeBitmap56, TypeUint56, TypeInt56:
		return 7
	case TypeData64, TypeBitmap64, TypeUint64, TypeInt64, TypeDouble, TypeIEEEAddr:
		return 8
	case TypeSecKey:
		return 16
	case TypeOctetString, TypeCharString, TypeLongOctet, TypeLongChar, TypeArray, TypeStruct, TypeSet, TypeBag:
		return 0 // Variable length
	default:
		return 0
	}
}

// ArrayValue represents a ZCL array with element type and values.
type ArrayValue struct {
	ElementType DataType
	Elements    []interface{}
}

// StructValue represents a ZCL struct with typed fields.
type StructValue struct {
	Fields []StructField
}

// StructField represents a single field in a ZCL struct.
type StructField struct {
	Type  DataType
	Value interface{}
}

// ReadValue reads a value of the given type from a byte slice.
// Returns the value and the number of bytes consumed.
func ReadValue(data []byte, dataType DataType) (interface{}, int, error) {
	switch dataType {
	case TypeNoData:
		return nil, 0, nil

	case TypeBoolean:
		if len(data) < 1 {
			return nil, 0, fmt.Errorf("insufficient data for boolean: need 1 byte, have %d", len(data))
		}
		return data[0] != 0, 1, nil

	case TypeUint8, TypeData8, TypeBitmap8, TypeEnum8:
		if len(data) < 1 {
			return nil, 0, fmt.Errorf("insufficient data for uint8: need 1 byte, have %d", len(data))
		}
		return data[0], 1, nil

	case TypeUint16, TypeData16, TypeBitmap16, TypeEnum16, TypeClusterID, TypeAttrID:
		if len(data) < 2 {
			return nil, 0, fmt.Errorf("insufficient data for uint16: need 2 bytes, have %d", len(data))
		}
		return binary.LittleEndian.Uint16(data), 2, nil

	case TypeUint24, TypeData24, TypeBitmap24:
		if len(data) < 3 {
			return nil, 0, fmt.Errorf("insufficient data for uint24: need 3 bytes, have %d", len(data))
		}
		val := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
		return val, 3, nil

	case TypeUint32, TypeData32, TypeBitmap32, TypeTimeOfDay, TypeDate, TypeUTCTime:
		if len(data) < 4 {
			return nil, 0, fmt.Errorf("insufficient data for uint32: need 4 bytes, have %d", len(data))
		}
		return binary.LittleEndian.Uint32(data), 4, nil

	case TypeUint40, TypeData40, TypeBitmap40:
		if len(data) < 5 {
			return nil, 0, fmt.Errorf("insufficient data for uint40: need 5 bytes, have %d", len(data))
		}
		val := uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 |
			uint64(data[3])<<24 | uint64(data[4])<<32
		return val, 5, nil

	case TypeUint48, TypeData48, TypeBitmap48:
		if len(data) < 6 {
			return nil, 0, fmt.Errorf("insufficient data for uint48: need 6 bytes, have %d", len(data))
		}
		val := uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 |
			uint64(data[3])<<24 | uint64(data[4])<<32 | uint64(data[5])<<40
		return val, 6, nil

	case TypeUint56, TypeData56, TypeBitmap56:
		if len(data) < 7 {
			return nil, 0, fmt.Errorf("insufficient data for uint56: need 7 bytes, have %d", len(data))
		}
		val := uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 |
			uint64(data[3])<<24 | uint64(data[4])<<32 | uint64(data[5])<<40 |
			uint64(data[6])<<48
		return val, 7, nil

	case TypeUint64, TypeData64, TypeBitmap64:
		if len(data) < 8 {
			return nil, 0, fmt.Errorf("insufficient data for uint64: need 8 bytes, have %d", len(data))
		}
		return binary.LittleEndian.Uint64(data), 8, nil

	case TypeInt8:
		if len(data) < 1 {
			return nil, 0, fmt.Errorf("insufficient data for int8: need 1 byte, have %d", len(data))
		}
		return int8(data[0]), 1, nil

	case TypeInt16:
		if len(data) < 2 {
			return nil, 0, fmt.Errorf("insufficient data for int16: need 2 bytes, have %d", len(data))
		}
		return int16(binary.LittleEndian.Uint16(data)), 2, nil

	case TypeInt24:
		if len(data) < 3 {
			return nil, 0, fmt.Errorf("insufficient data for int24: need 3 bytes, have %d", len(data))
		}
		val := int32(data[0]) | int32(data[1])<<8 | int32(data[2])<<16
		// Sign extend from 24 bits using arithmetic shift
		val = (val << 8) >> 8
		return val, 3, nil

	case TypeInt32:
		if len(data) < 4 {
			return nil, 0, fmt.Errorf("insufficient data for int32: need 4 bytes, have %d", len(data))
		}
		return int32(binary.LittleEndian.Uint32(data)), 4, nil

	case TypeInt40:
		if len(data) < 5 {
			return nil, 0, fmt.Errorf("insufficient data for int40: need 5 bytes, have %d", len(data))
		}
		val := int64(data[0]) | int64(data[1])<<8 | int64(data[2])<<16 |
			int64(data[3])<<24 | int64(data[4])<<32
		// Sign extend from 40 bits using arithmetic shift
		val = (val << 24) >> 24
		return val, 5, nil

	case TypeInt48:
		if len(data) < 6 {
			return nil, 0, fmt.Errorf("insufficient data for int48: need 6 bytes, have %d", len(data))
		}
		val := int64(data[0]) | int64(data[1])<<8 | int64(data[2])<<16 |
			int64(data[3])<<24 | int64(data[4])<<32 | int64(data[5])<<40
		// Sign extend from 48 bits using arithmetic shift
		val = (val << 16) >> 16
		return val, 6, nil

	case TypeInt56:
		if len(data) < 7 {
			return nil, 0, fmt.Errorf("insufficient data for int56: need 7 bytes, have %d", len(data))
		}
		val := int64(data[0]) | int64(data[1])<<8 | int64(data[2])<<16 |
			int64(data[3])<<24 | int64(data[4])<<32 | int64(data[5])<<40 |
			int64(data[6])<<48
		// Sign extend from 56 bits using arithmetic shift
		val = (val << 8) >> 8
		return val, 7, nil

	case TypeInt64:
		if len(data) < 8 {
			return nil, 0, fmt.Errorf("insufficient data for int64: need 8 bytes, have %d", len(data))
		}
		return int64(binary.LittleEndian.Uint64(data)), 8, nil

	case TypeSingle:
		if len(data) < 4 {
			return nil, 0, fmt.Errorf("insufficient data for float32: need 4 bytes, have %d", len(data))
		}
		bits := binary.LittleEndian.Uint32(data)
		return math.Float32frombits(bits), 4, nil

	case TypeDouble:
		if len(data) < 8 {
			return nil, 0, fmt.Errorf("insufficient data for float64: need 8 bytes, have %d", len(data))
		}
		bits := binary.LittleEndian.Uint64(data)
		return math.Float64frombits(bits), 8, nil

	case TypeOctetString, TypeCharString:
		if len(data) < 1 {
			return nil, 0, fmt.Errorf("insufficient data for string length: need 1 byte, have %d", len(data))
		}
		length := int(data[0])
		if length == 0xFF { // Invalid/null string
			return nil, 1, nil
		}
		if len(data) < 1+length {
			return nil, 0, fmt.Errorf("insufficient data for string: need %d bytes, have %d", 1+length, len(data))
		}
		if dataType == TypeCharString {
			return string(data[1 : 1+length]), 1 + length, nil
		}
		result := make([]byte, length)
		copy(result, data[1:1+length])
		return result, 1 + length, nil

	case TypeLongOctet, TypeLongChar:
		if len(data) < 2 {
			return nil, 0, fmt.Errorf("insufficient data for long string length: need 2 bytes, have %d", len(data))
		}
		length := int(binary.LittleEndian.Uint16(data))
		if length == 0xFFFF { // Invalid/null string
			return nil, 2, nil
		}
		if len(data) < 2+length {
			return nil, 0, fmt.Errorf("insufficient data for long string: need %d bytes, have %d", 2+length, len(data))
		}
		if dataType == TypeLongChar {
			return string(data[2 : 2+length]), 2 + length, nil
		}
		result := make([]byte, length)
		copy(result, data[2:2+length])
		return result, 2 + length, nil

	case TypeArray, TypeSet, TypeBag:
		// Array format: element_type (1 byte) + count (2 bytes) + elements
		if len(data) < 3 {
			return nil, 0, fmt.Errorf("insufficient data for array header: need 3 bytes, have %d", len(data))
		}
		elemType := DataType(data[0])
		count := int(binary.LittleEndian.Uint16(data[1:3]))

		// Handle invalid count
		if count == 0xFFFF {
			return &ArrayValue{ElementType: elemType, Elements: nil}, 3, nil
		}

		offset := 3
		elements := make([]interface{}, 0, count)

		for i := 0; i < count; i++ {
			if offset >= len(data) {
				return nil, 0, fmt.Errorf("insufficient data for array element %d", i)
			}
			elem, consumed, err := ReadValue(data[offset:], elemType)
			if err != nil {
				return nil, 0, fmt.Errorf("reading array element %d: %w", i, err)
			}
			elements = append(elements, elem)
			offset += consumed
		}

		return &ArrayValue{ElementType: elemType, Elements: elements}, offset, nil

	case TypeStruct:
		// Struct format: count (2 bytes) + (type (1 byte) + value) for each field
		if len(data) < 2 {
			return nil, 0, fmt.Errorf("insufficient data for struct header: need 2 bytes, have %d", len(data))
		}
		count := int(binary.LittleEndian.Uint16(data))

		// Handle invalid count
		if count == 0xFFFF {
			return &StructValue{Fields: nil}, 2, nil
		}

		offset := 2
		fields := make([]StructField, 0, count)

		for i := 0; i < count; i++ {
			if offset >= len(data) {
				return nil, 0, fmt.Errorf("insufficient data for struct field %d type", i)
			}
			fieldType := DataType(data[offset])
			offset++

			val, consumed, err := ReadValue(data[offset:], fieldType)
			if err != nil {
				return nil, 0, fmt.Errorf("reading struct field %d: %w", i, err)
			}
			fields = append(fields, StructField{Type: fieldType, Value: val})
			offset += consumed
		}

		return &StructValue{Fields: fields}, offset, nil

	case TypeIEEEAddr:
		if len(data) < 8 {
			return nil, 0, fmt.Errorf("insufficient data for IEEE address: need 8 bytes, have %d", len(data))
		}
		var addr [8]byte
		copy(addr[:], data[:8])
		return addr, 8, nil

	case TypeSecKey:
		if len(data) < 16 {
			return nil, 0, fmt.Errorf("insufficient data for security key: need 16 bytes, have %d", len(data))
		}
		var key [16]byte
		copy(key[:], data[:16])
		return key, 16, nil

	default:
		return nil, 0, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// WriteValue writes a value to a byte slice for the given type.
func WriteValue(value interface{}, dataType DataType) ([]byte, error) {
	switch dataType {
	case TypeNoData:
		return []byte{}, nil

	case TypeBoolean:
		v, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool for boolean type, got %T", value)
		}
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil

	case TypeUint8, TypeData8, TypeBitmap8, TypeEnum8:
		v, ok := value.(uint8)
		if !ok {
			return nil, fmt.Errorf("expected uint8, got %T", value)
		}
		return []byte{v}, nil

	case TypeUint16, TypeData16:
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("expected uint16, got %T", value)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, v)
		return buf, nil

	case TypeBitmap16:
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("expected uint16, got %T", value)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, v)
		return buf, nil

	case TypeEnum16:
		v, ok := value.(uint16)
		if !ok {
			return nil, fmt.Errorf("expected uint16, got %T", value)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, v)
		return buf, nil

	case TypeUint24:
		v, ok := value.(uint32)
		if !ok {
			return nil, fmt.Errorf("expected uint32 for uint24, got %T", value)
		}
		return []byte{byte(v), byte(v >> 8), byte(v >> 16)}, nil

	case TypeBitmap24:
		v, ok := value.(uint32)
		if !ok {
			return nil, fmt.Errorf("expected uint32, got %T", value)
		}
		return []byte{byte(v), byte(v >> 8), byte(v >> 16)}, nil

	case TypeUint32, TypeData32:
		v, ok := value.(uint32)
		if !ok {
			return nil, fmt.Errorf("expected uint32, got %T", value)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, v)
		return buf, nil

	case TypeBitmap32:
		v, ok := value.(uint32)
		if !ok {
			return nil, fmt.Errorf("expected uint32, got %T", value)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, v)
		return buf, nil

	case TypeBitmap40:
		v, ok := value.(uint64)
		if !ok {
			return nil, fmt.Errorf("expected uint64, got %T", value)
		}
		return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24), byte(v >> 32)}, nil

	case TypeUint48:
		v, ok := value.(uint64)
		if !ok {
			return nil, fmt.Errorf("expected uint64 for uint48, got %T", value)
		}
		return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24), byte(v >> 32), byte(v >> 40)}, nil

	case TypeBitmap48:
		v, ok := value.(uint64)
		if !ok {
			return nil, fmt.Errorf("expected uint64, got %T", value)
		}
		return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24), byte(v >> 32), byte(v >> 40)}, nil

	case TypeBitmap56:
		v, ok := value.(uint64)
		if !ok {
			return nil, fmt.Errorf("expected uint64, got %T", value)
		}
		return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24), byte(v >> 32), byte(v >> 40), byte(v >> 48)}, nil

	case TypeBitmap64:
		v, ok := value.(uint64)
		if !ok {
			return nil, fmt.Errorf("expected uint64, got %T", value)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, v)
		return buf, nil

	case TypeInt8:
		v, ok := value.(int8)
		if !ok {
			return nil, fmt.Errorf("expected int8, got %T", value)
		}
		return []byte{byte(v)}, nil

	case TypeInt16:
		v, ok := value.(int16)
		if !ok {
			return nil, fmt.Errorf("expected int16, got %T", value)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(v))
		return buf, nil

	case TypeInt32:
		v, ok := value.(int32)
		if !ok {
			return nil, fmt.Errorf("expected int32, got %T", value)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(v))
		return buf, nil

	case TypeSingle:
		v, ok := value.(float32)
		if !ok {
			return nil, fmt.Errorf("expected float32, got %T", value)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
		return buf, nil

	case TypeOctetString:
		v, ok := value.([]byte)
		if !ok {
			return nil, fmt.Errorf("expected []byte for octet string, got %T", value)
		}
		if len(v) > 255 {
			return nil, fmt.Errorf("octet string too long: %d bytes (max 255)", len(v))
		}
		result := make([]byte, 1+len(v))
		result[0] = byte(len(v))
		copy(result[1:], v)
		return result, nil

	case TypeCharString:
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for char string, got %T", value)
		}
		if len(v) > 255 {
			return nil, fmt.Errorf("char string too long: %d bytes (max 255)", len(v))
		}
		result := make([]byte, 1+len(v))
		result[0] = byte(len(v))
		copy(result[1:], v)
		return result, nil

	case TypeIEEEAddr:
		v, ok := value.([8]byte)
		if !ok {
			return nil, fmt.Errorf("expected [8]byte for IEEE address, got %T", value)
		}
		return v[:], nil

	case TypeSecKey:
		v, ok := value.([16]byte)
		if !ok {
			return nil, fmt.Errorf("expected [16]byte for security key, got %T", value)
		}
		return v[:], nil

	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

package zcl

// Type conversion helpers for ZCL attribute values.
// These functions provide safe type assertions with default values,
// reducing boilerplate in attribute parsing code.

// ToUint8 safely converts an interface{} to uint8.
// Returns the value and true if successful, or 0 and false otherwise.
func ToUint8(v interface{}) (uint8, bool) {
	if val, ok := v.(uint8); ok {
		return val, true
	}
	return 0, false
}

// ToUint16 safely converts an interface{} to uint16.
// Returns the value and true if successful, or 0 and false otherwise.
func ToUint16(v interface{}) (uint16, bool) {
	if val, ok := v.(uint16); ok {
		return val, true
	}
	return 0, false
}

// ToUint32 safely converts an interface{} to uint32.
// Returns the value and true if successful, or 0 and false otherwise.
func ToUint32(v interface{}) (uint32, bool) {
	if val, ok := v.(uint32); ok {
		return val, true
	}
	return 0, false
}

// ToUint64 safely converts an interface{} to uint64.
// Returns the value and true if successful, or 0 and false otherwise.
func ToUint64(v interface{}) (uint64, bool) {
	if val, ok := v.(uint64); ok {
		return val, true
	}
	return 0, false
}

// ToInt8 safely converts an interface{} to int8.
// Returns the value and true if successful, or 0 and false otherwise.
func ToInt8(v interface{}) (int8, bool) {
	if val, ok := v.(int8); ok {
		return val, true
	}
	return 0, false
}

// ToInt16 safely converts an interface{} to int16.
// Returns the value and true if successful, or 0 and false otherwise.
func ToInt16(v interface{}) (int16, bool) {
	if val, ok := v.(int16); ok {
		return val, true
	}
	return 0, false
}

// ToInt32 safely converts an interface{} to int32.
// Returns the value and true if successful, or 0 and false otherwise.
func ToInt32(v interface{}) (int32, bool) {
	if val, ok := v.(int32); ok {
		return val, true
	}
	return 0, false
}

// ToInt64 safely converts an interface{} to int64.
// Returns the value and true if successful, or 0 and false otherwise.
func ToInt64(v interface{}) (int64, bool) {
	if val, ok := v.(int64); ok {
		return val, true
	}
	return 0, false
}

// ToFloat32 safely converts an interface{} to float32.
// Returns the value and true if successful, or 0 and false otherwise.
func ToFloat32(v interface{}) (float32, bool) {
	if val, ok := v.(float32); ok {
		return val, true
	}
	return 0, false
}

// ToFloat64 safely converts an interface{} to float64.
// Returns the value and true if successful, or 0 and false otherwise.
func ToFloat64(v interface{}) (float64, bool) {
	if val, ok := v.(float64); ok {
		return val, true
	}
	return 0, false
}

// ToBool safely converts an interface{} to bool.
// Returns the value and true if successful, or false and false otherwise.
func ToBool(v interface{}) (bool, bool) {
	if val, ok := v.(bool); ok {
		return val, true
	}
	return false, false
}

// ToString safely converts an interface{} to string.
// Returns the value and true if successful, or "" and false otherwise.
func ToString(v interface{}) (string, bool) {
	if val, ok := v.(string); ok {
		return val, true
	}
	return "", false
}

// ToBytes safely converts an interface{} to []byte.
// Returns the value and true if successful, or nil and false otherwise.
func ToBytes(v interface{}) ([]byte, bool) {
	if val, ok := v.([]byte); ok {
		return val, true
	}
	return nil, false
}

// ToBoolFromUint8 converts a uint8 value to bool (0 = false, non-zero = true).
// This is useful for ZCL boolean attributes that are encoded as uint8.
// Returns the boolean value and true if the input was uint8 or bool.
func ToBoolFromUint8(v interface{}) (bool, bool) {
	if val, ok := v.(bool); ok {
		return val, true
	}
	if val, ok := v.(uint8); ok {
		return val != 0, true
	}
	return false, false
}

// MustUint8 returns the uint8 value or the provided default.
func MustUint8(v interface{}, def uint8) uint8 {
	if val, ok := v.(uint8); ok {
		return val
	}
	return def
}

// MustUint16 returns the uint16 value or the provided default.
func MustUint16(v interface{}, def uint16) uint16 {
	if val, ok := v.(uint16); ok {
		return val
	}
	return def
}

// MustUint32 returns the uint32 value or the provided default.
func MustUint32(v interface{}, def uint32) uint32 {
	if val, ok := v.(uint32); ok {
		return val
	}
	return def
}

// MustInt16 returns the int16 value or the provided default.
func MustInt16(v interface{}, def int16) int16 {
	if val, ok := v.(int16); ok {
		return val
	}
	return def
}

// MustString returns the string value or the provided default.
func MustString(v interface{}, def string) string {
	if val, ok := v.(string); ok {
		return val
	}
	return def
}

// MustBool returns the bool value or the provided default.
// Also handles uint8 values (0 = false, non-zero = true).
func MustBool(v interface{}, def bool) bool {
	if val, ok := v.(bool); ok {
		return val
	}
	if val, ok := v.(uint8); ok {
		return val != 0
	}
	return def
}

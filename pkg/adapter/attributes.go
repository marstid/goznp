package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// Attribute Reading Helpers
// from Zigbee devices. They handle the common pattern of:
//   1. Calling ReadAttributes
//   2. Checking for errors and results
//   3. Verifying status
//   4. Type asserting/converting the value
//
// Example:
//   // Before (15 lines):
//   results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.AttrLevelCurrentLevel)
//   if err != nil {
//       return 0, fmt.Errorf("failed to read brightness level: %w", err)
//   }
//   if len(results) == 0 {
//       return 0, fmt.Errorf("no attributes returned")
//   }
//   result := results[0]
//   if result.Status != zcl.StatusSuccess {
//       return 0, fmt.Errorf("read brightness returned status 0x%02X", result.Status)
//   }
//   if level, ok := result.Value.(uint8); ok {
//       return level, nil
//   }
//   return 0, fmt.Errorf("unexpected value type: %T", result.Value)
//
//   // After (1 line):
//   level, err := a.ReadAttributeUint8(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.AttrLevelCurrentLevel)

// GetFirstResult returns the first successful attribute result, or an error.
// This is a helper for extracting and validating a single attribute result.
func GetFirstResult(results []AttributeResult) (*AttributeResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no attributes returned")
	}
	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return nil, fmt.Errorf("attribute read returned status 0x%02X", uint8(result.Status))
	}
	return &result, nil
}

// Value Extraction Helpers
//
// These functions extract typed values from interface{} attribute values.
// They handle type coercion where appropriate (e.g., uint8 from uint16).

// ExtractUint8 extracts a uint8 from an attribute result value.
// Returns false if the value cannot be converted to uint8.
func ExtractUint8(v interface{}) (uint8, bool) {
	switch val := v.(type) {
	case uint8:
		return val, true
	case uint16:
		if val <= 255 {
			return uint8(val), true
		}
	case int:
		if val >= 0 && val <= 255 {
			return uint8(val), true
		}
	}
	return 0, false
}

// ExtractUint16 extracts a uint16 from an attribute result value.
// Uses the existing toUint16 helper which handles uint16, int16, uint8, and int.
func ExtractUint16(v interface{}) (uint16, bool) {
	return toUint16(v)
}

// ExtractInt16 extracts an int16 from an attribute result value.
// Uses the existing toInt16 helper which handles int16, uint16, and int.
func ExtractInt16(v interface{}) (int16, bool) {
	return toInt16(v)
}

// ExtractInt8 extracts an int8 from an attribute result value.
// Uses the existing toInt8 helper which handles int8, uint8, and int.
func ExtractInt8(v interface{}) (int8, bool) {
	return toInt8(v)
}

// ExtractUint32 extracts a uint32 from an attribute result value.
// Uses the existing toUint32 helper which handles uint32, int32, uint16, and int.
func ExtractUint32(v interface{}) (uint32, bool) {
	return toUint32(v)
}

// ExtractInt32 extracts an int32 from an attribute result value.
// Uses the existing toInt32 helper which handles int32, uint32, and int.
func ExtractInt32(v interface{}) (int32, bool) {
	return toInt32(v)
}

// ExtractUint48 extracts a uint48 (as uint64) from an attribute result value.
// Uses the existing toUint48 helper which handles uint64, int64, uint32, and int.
func ExtractUint48(v interface{}) (uint64, bool) {
	return toUint48(v)
}

// ExtractBool extracts a bool from an attribute result value.
// Handles both native bool and uint8 representations (0 = false, non-zero = true).
func ExtractBool(v interface{}) (bool, bool) {
	switch val := v.(type) {
	case bool:
		return val, true
	case uint8:
		return val != 0, true
	case uint16:
		return val != 0, true
	case int:
		return val != 0, true
	}
	return false, false
}

// ExtractString extracts a string from an attribute result value.
// Returns false if the value is not a string.
func ExtractString(v interface{}) (string, bool) {
	if str, ok := v.(string); ok {
		return str, true
	}
	return "", false
}

// ExtractFloat32 extracts a float32 from an attribute result value.
// Returns false if the value is not a float32.
func ExtractFloat32(v interface{}) (float32, bool) {
	if f, ok := v.(float32); ok {
		return f, true
	}
	return 0, false
}

// Type-Safe Single Attribute Readers
//
// These functions read a single attribute and return its typed value.
// They combine ReadAttributes, GetFirstResult, and type extraction into
// a single convenient call.

// ReadAttributeUint8 reads a single uint8 attribute from a device.
// Returns an error if the attribute cannot be read or converted to uint8.
func (a *Adapter) ReadAttributeUint8(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (uint8, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractUint8(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeUint16 reads a single uint16 attribute from a device.
// Returns an error if the attribute cannot be read or converted to uint16.
func (a *Adapter) ReadAttributeUint16(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (uint16, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractUint16(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeInt16 reads a single int16 attribute from a device.
// Returns an error if the attribute cannot be read or converted to int16.
func (a *Adapter) ReadAttributeInt16(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (int16, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractInt16(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeInt8 reads a single int8 attribute from a device.
// Returns an error if the attribute cannot be read or converted to int8.
func (a *Adapter) ReadAttributeInt8(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (int8, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractInt8(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeUint32 reads a single uint32 attribute from a device.
// Returns an error if the attribute cannot be read or converted to uint32.
func (a *Adapter) ReadAttributeUint32(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (uint32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractUint32(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeInt32 reads a single int32 attribute from a device.
// Returns an error if the attribute cannot be read or converted to int32.
func (a *Adapter) ReadAttributeInt32(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (int32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractInt32(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeUint48 reads a single uint48 (as uint64) attribute from a device.
// Returns an error if the attribute cannot be read or converted to uint64.
func (a *Adapter) ReadAttributeUint48(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (uint64, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractUint48(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeBool reads a single bool attribute from a device.
// Handles both native bool and uint8 representations (0 = false, non-zero = true).
// Returns an error if the attribute cannot be read or converted to bool.
func (a *Adapter) ReadAttributeBool(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (bool, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return false, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return false, err
	}

	val, ok := ExtractBool(result.Value)
	if !ok {
		return false, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeString reads a single string attribute from a device.
// Returns an error if the attribute cannot be read or is not a string.
func (a *Adapter) ReadAttributeString(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (string, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return "", fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return "", err
	}

	val, ok := ExtractString(result.Value)
	if !ok {
		return "", fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

// ReadAttributeFloat32 reads a single float32 attribute from a device.
// Returns an error if the attribute cannot be read or is not a float32.
func (a *Adapter) ReadAttributeFloat32(ctx context.Context, nwkAddr uint16, endpoint uint8, cluster zcl.ClusterID, attr zcl.AttributeID) (float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, cluster, attr)
	if err != nil {
		return 0, fmt.Errorf("failed to read attribute 0x%04X: %w", attr, err)
	}

	result, err := GetFirstResult(results)
	if err != nil {
		return 0, err
	}

	val, ok := ExtractFloat32(result.Value)
	if !ok {
		return 0, fmt.Errorf("attribute 0x%04X has unexpected type %T", attr, result.Value)
	}

	return val, nil
}

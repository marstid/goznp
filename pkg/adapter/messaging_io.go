package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// BinaryInput Cluster (0x000F)

// BinaryInputInfo contains binary input status information.
type BinaryInputInfo struct {
	PresentValue bool              // Current input value (active/inactive)
	OutOfService bool              // Out of service flag
	StatusFlags  uint8             // Status bitmap
	Reliability  zcl.IOReliability // Reliability state
}

// GetBinaryInput reads the binary input status from a device.
// Returns the present value, out-of-service flag, status flags, and reliability.
// This provides complete information about a binary input sensor.
func (a *Adapter) GetBinaryInput(ctx context.Context, nwkAddr uint16, endpoint uint8) (*BinaryInputInfo, error) {
	// Define attributes to read
	attrs := []zcl.AttributeID{
		zcl.AttrBinaryInputPresentValue,
		zcl.AttrBinaryInputOutOfService,
		zcl.AttrBinaryInputStatusFlags,
		zcl.AttrBinaryInputReliability,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryInput, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary input attributes: %w", err)
	}

	info := &BinaryInputInfo{}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrBinaryInputPresentValue:
			if v, ok := r.Value.(bool); ok {
				info.PresentValue = v
			} else if v, ok := r.Value.(uint8); ok {
				info.PresentValue = v != 0
			}
		case zcl.AttrBinaryInputOutOfService:
			if v, ok := r.Value.(bool); ok {
				info.OutOfService = v
			} else if v, ok := r.Value.(uint8); ok {
				info.OutOfService = v != 0
			}
		case zcl.AttrBinaryInputStatusFlags:
			if v, ok := r.Value.(uint8); ok {
				info.StatusFlags = v
			}
		case zcl.AttrBinaryInputReliability:
			if v, ok := r.Value.(uint8); ok {
				info.Reliability = zcl.IOReliability(v)
			}
		}
	}

	return info, nil
}

// GetBinaryInputValue reads only the present value from a binary input device.
// Returns the current input state (true=active, false=inactive).
// This is a convenience method when you only need the input value.
func (a *Adapter) GetBinaryInputValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (bool, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryInput, zcl.AttrBinaryInputPresentValue)
	if err != nil {
		return false, fmt.Errorf("failed to read binary input value: %w", err)
	}

	if len(results) == 0 {
		return false, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return false, fmt.Errorf("read binary input value returned status 0x%02X", result.Status)
	}

	// Handle both bool and uint8 types
	switch v := result.Value.(type) {
	case bool:
		return v, nil
	case uint8:
		return v != 0, nil
	default:
		return false, fmt.Errorf("unexpected value type: %T", result.Value)
	}
}

// AnalogInput Cluster (0x000C)

// AnalogInputInfo contains analog input sensor information.
type AnalogInputInfo struct {
	PresentValue     float32              // Current analog value
	OutOfService     bool                 // Out of service flag
	StatusFlags      uint8                // Status bitmap
	EngineeringUnits zcl.EngineeringUnits // Engineering units
	MinValue         float32              // Minimum value
	MaxValue         float32              // Maximum value
}

// GetAnalogInput reads all key analog input attributes from a device.
// Returns comprehensive information about the analog input including current value,
// status, units, and range.
func (a *Adapter) GetAnalogInput(ctx context.Context, nwkAddr uint16, endpoint uint8) (*AnalogInputInfo, error) {
	// Read analog input attributes
	attrs := []zcl.AttributeID{
		zcl.AttrAnalogInputPresentValue,
		zcl.AttrAnalogInputOutOfService,
		zcl.AttrAnalogInputStatusFlags,
		zcl.AttrAnalogInputEngineeringUnits,
		zcl.AttrAnalogInputMinPresentValue,
		zcl.AttrAnalogInputMaxPresentValue,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogInput, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read analog input attributes: %w", err)
	}

	info := &AnalogInputInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrAnalogInputPresentValue:
			if val, ok := r.Value.(float32); ok {
				info.PresentValue = val
			}
		case zcl.AttrAnalogInputOutOfService:
			if val, ok := r.Value.(bool); ok {
				info.OutOfService = val
			} else if val, ok := r.Value.(uint8); ok {
				info.OutOfService = val != 0
			}
		case zcl.AttrAnalogInputStatusFlags:
			if val, ok := r.Value.(uint8); ok {
				info.StatusFlags = val
			}
		case zcl.AttrAnalogInputEngineeringUnits:
			if val, ok := toUint16(r.Value); ok {
				info.EngineeringUnits = zcl.EngineeringUnits(val)
			}
		case zcl.AttrAnalogInputMinPresentValue:
			if val, ok := r.Value.(float32); ok {
				info.MinValue = val
			}
		case zcl.AttrAnalogInputMaxPresentValue:
			if val, ok := r.Value.(float32); ok {
				info.MaxValue = val
			}
		}
	}

	return info, nil
}

// GetAnalogInputValue reads just the present value from an analog input.
// This is a simplified method for quickly reading the current analog value.
// Returns the analog value as float32.
func (a *Adapter) GetAnalogInputValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogInput, zcl.AttrAnalogInputPresentValue)
	if err != nil {
		return 0, fmt.Errorf("failed to read analog input value: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read analog input value returned status 0x%02X", result.Status)
	}

	if val, ok := result.Value.(float32); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// AnalogOutput Cluster (0x000D)

// AnalogOutputInfo contains analog output control information.
type AnalogOutputInfo struct {
	PresentValue     float32              // Current output value
	OutOfService     bool                 // Out of service flag
	StatusFlags      uint8                // Status bitmap
	EngineeringUnits zcl.EngineeringUnits // Engineering units
	MinValue         float32              // Minimum value
	MaxValue         float32              // Maximum value
}

// GetAnalogOutput reads all key analog output attributes from a device.
// Returns comprehensive information about the analog output including current value,
// status, units, and range.
func (a *Adapter) GetAnalogOutput(ctx context.Context, nwkAddr uint16, endpoint uint8) (*AnalogOutputInfo, error) {
	// Read analog output attributes
	attrs := []zcl.AttributeID{
		zcl.AttrAnalogOutputPresentValue,
		zcl.AttrAnalogOutputOutOfService,
		zcl.AttrAnalogOutputStatusFlags,
		zcl.AttrAnalogOutputEngineeringUnits,
		zcl.AttrAnalogOutputMinPresentValue,
		zcl.AttrAnalogOutputMaxPresentValue,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogOutput, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read analog output attributes: %w", err)
	}

	info := &AnalogOutputInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrAnalogOutputPresentValue:
			if val, ok := r.Value.(float32); ok {
				info.PresentValue = val
			}
		case zcl.AttrAnalogOutputOutOfService:
			if val, ok := r.Value.(bool); ok {
				info.OutOfService = val
			} else if val, ok := r.Value.(uint8); ok {
				info.OutOfService = val != 0
			}
		case zcl.AttrAnalogOutputStatusFlags:
			if val, ok := r.Value.(uint8); ok {
				info.StatusFlags = val
			}
		case zcl.AttrAnalogOutputEngineeringUnits:
			if val, ok := toUint16(r.Value); ok {
				info.EngineeringUnits = zcl.EngineeringUnits(val)
			}
		case zcl.AttrAnalogOutputMinPresentValue:
			if val, ok := r.Value.(float32); ok {
				info.MinValue = val
			}
		case zcl.AttrAnalogOutputMaxPresentValue:
			if val, ok := r.Value.(float32); ok {
				info.MaxValue = val
			}
		}
	}

	return info, nil
}

// SetAnalogOutput sets the output value on an analog output device.
// value is the desired analog output value to set.
// This writes to the PresentValue attribute (0x0055) which is writable.
func (a *Adapter) SetAnalogOutput(ctx context.Context, nwkAddr uint16, endpoint uint8, value float32) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrAnalogOutputPresentValue: {
			Type:  zcl.TypeSingle,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogOutput, values)
}

// GetAnalogOutputValue reads just the present value from an analog output.
// This is a simplified method for quickly reading the current analog output value.
// Returns the analog value as float32.
func (a *Adapter) GetAnalogOutputValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogOutput, zcl.AttrAnalogOutputPresentValue)
	if err != nil {
		return 0, fmt.Errorf("failed to read analog output value: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read analog output value returned status 0x%02X", result.Status)
	}

	if val, ok := result.Value.(float32); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// AnalogValue Cluster (0x000E)

// AnalogValueInfo contains analog value information.
type AnalogValueInfo struct {
	PresentValue     float32              // Current analog value (read/write)
	OutOfService     bool                 // Out of service flag
	StatusFlags      uint8                // Status bitmap
	EngineeringUnits zcl.EngineeringUnits // Engineering units
}

// GetAnalogValue reads all key analog value attributes from a device.
// Returns comprehensive information about the analog value including current value,
// status, and units.
func (a *Adapter) GetAnalogValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (*AnalogValueInfo, error) {
	// Read analog value attributes
	attrs := []zcl.AttributeID{
		zcl.AttrAnalogValuePresentValue,
		zcl.AttrAnalogValueOutOfService,
		zcl.AttrAnalogValueStatusFlags,
		zcl.AttrAnalogValueEngineeringUnits,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogValue, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read analog value attributes: %w", err)
	}

	info := &AnalogValueInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrAnalogValuePresentValue:
			if val, ok := r.Value.(float32); ok {
				info.PresentValue = val
			}
		case zcl.AttrAnalogValueOutOfService:
			if val, ok := r.Value.(bool); ok {
				info.OutOfService = val
			} else if val, ok := r.Value.(uint8); ok {
				info.OutOfService = val != 0
			}
		case zcl.AttrAnalogValueStatusFlags:
			if val, ok := r.Value.(uint8); ok {
				info.StatusFlags = val
			}
		case zcl.AttrAnalogValueEngineeringUnits:
			if val, ok := toUint16(r.Value); ok {
				info.EngineeringUnits = zcl.EngineeringUnits(val)
			}
		}
	}

	return info, nil
}

// SetAnalogValue sets the present value of an analog value object.
// value is the desired float32 value to set.
func (a *Adapter) SetAnalogValue(ctx context.Context, nwkAddr uint16, endpoint uint8, value float32) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrAnalogValuePresentValue: {
			Type:  zcl.TypeSingle,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAnalogValue, values)
}
// BinaryValue Cluster (0x0011)

// BinaryValueInfo contains binary value information.
type BinaryValueInfo struct {
	PresentValue bool              // Current value (read/write)
	OutOfService bool              // Out of service flag
	StatusFlags  uint8             // Status bitmap
	Reliability  zcl.IOReliability // Reliability status
}

// GetBinaryValue reads all key binary value attributes from a device.
// Returns comprehensive information about the binary value including current value,
// status, and reliability.
func (a *Adapter) GetBinaryValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (*BinaryValueInfo, error) {
	// Read binary value attributes
	attrs := []zcl.AttributeID{
		zcl.AttrBinaryValuePresentValue,
		zcl.AttrBinaryValueOutOfService,
		zcl.AttrBinaryValueStatusFlags,
		zcl.AttrBinaryValueReliability,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryValue, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary value attributes: %w", err)
	}

	info := &BinaryValueInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrBinaryValuePresentValue:
			if val, ok := r.Value.(bool); ok {
				info.PresentValue = val
			} else if val, ok := r.Value.(uint8); ok {
				info.PresentValue = val != 0
			}
		case zcl.AttrBinaryValueOutOfService:
			if val, ok := r.Value.(bool); ok {
				info.OutOfService = val
			} else if val, ok := r.Value.(uint8); ok {
				info.OutOfService = val != 0
			}
		case zcl.AttrBinaryValueStatusFlags:
			if val, ok := r.Value.(uint8); ok {
				info.StatusFlags = val
			}
		case zcl.AttrBinaryValueReliability:
			if val, ok := r.Value.(uint8); ok {
				info.Reliability = zcl.IOReliability(val)
			}
		}
	}

	return info, nil
}

// SetBinaryValue sets the present value of a binary value object.
// value is the desired boolean state to set.
func (a *Adapter) SetBinaryValue(ctx context.Context, nwkAddr uint16, endpoint uint8, value bool) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrBinaryValuePresentValue: {
			Type:  zcl.TypeBoolean,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryValue, values)
}

// MultistateInput Cluster (0x0012)

// MultistateInputInfo contains multistate input information.
type MultistateInputInfo struct {
	PresentValue   uint16            // Current state value (1-based)
	NumberOfStates uint16            // Number of possible states
	OutOfService   bool              // Out of service flag
	StatusFlags    uint8             // Status bitmap
	Reliability    zcl.IOReliability // Reliability state
}

// GetMultistateInput reads all key multistate input attributes from a device.
// Returns comprehensive information about the multistate input including current value,
// number of states, status, and reliability.
func (a *Adapter) GetMultistateInput(ctx context.Context, nwkAddr uint16, endpoint uint8) (*MultistateInputInfo, error) {
	// Read multistate input attributes
	attrs := []zcl.AttributeID{
		zcl.AttrMultistateInputPresentValue,
		zcl.AttrMultistateInputNumberOfStates,
		zcl.AttrMultistateInputOutOfService,
		zcl.AttrMultistateInputStatusFlags,
		zcl.AttrMultistateInputReliability,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMultistateInput, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read multistate input attributes: %w", err)
	}

	info := &MultistateInputInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrMultistateInputPresentValue:
			if val, ok := toUint16(r.Value); ok {
				info.PresentValue = val
			}
		case zcl.AttrMultistateInputNumberOfStates:
			if val, ok := toUint16(r.Value); ok {
				info.NumberOfStates = val
			}
		case zcl.AttrMultistateInputOutOfService:
			if val, ok := r.Value.(bool); ok {
				info.OutOfService = val
			} else if val, ok := r.Value.(uint8); ok {
				info.OutOfService = val != 0
			}
		case zcl.AttrMultistateInputStatusFlags:
			if val, ok := r.Value.(uint8); ok {
				info.StatusFlags = val
			}
		case zcl.AttrMultistateInputReliability:
			if val, ok := r.Value.(uint8); ok {
				info.Reliability = zcl.IOReliability(val)
			}
		}
	}

	return info, nil
}

// GetMultistateInputValue reads just the current state value from a multistate input.
// This is a simplified method for quickly reading the current state.
// Returns the state value as uint16 (1-based indexing).
func (a *Adapter) GetMultistateInputValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (uint16, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMultistateInput, zcl.AttrMultistateInputPresentValue)
	if err != nil {
		return 0, fmt.Errorf("failed to read multistate input value: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read multistate input value returned status 0x%02X", result.Status)
	}

	if val, ok := toUint16(result.Value); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// BinaryOutput Cluster (0x0010)

// BinaryOutputInfo contains the status information for a BinaryOutput device.
type BinaryOutputInfo struct {
	PresentValue bool  // Current output value
	OutOfService bool  // Out of service flag
	StatusFlags  uint8 // Status flags bitmap
	Polarity     uint8 // 0=normal, 1=reversed
}

// GetBinaryOutput reads the current status from a BinaryOutput device.
// Returns present value, out of service flag, status flags, and polarity.
func (a *Adapter) GetBinaryOutput(ctx context.Context, nwkAddr uint16, endpoint uint8) (*BinaryOutputInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrBinaryOutputPresentValue,
		zcl.AttrBinaryOutputOutOfService,
		zcl.AttrBinaryOutputStatusFlags,
		zcl.AttrBinaryOutputPolarity,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryOutput, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary output attributes: %w", err)
	}

	info := &BinaryOutputInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrBinaryOutputPresentValue:
			if v, ok := r.Value.(bool); ok {
				info.PresentValue = v
			} else if v, ok := r.Value.(uint8); ok {
				info.PresentValue = v != 0
			}
		case zcl.AttrBinaryOutputOutOfService:
			if v, ok := r.Value.(bool); ok {
				info.OutOfService = v
			} else if v, ok := r.Value.(uint8); ok {
				info.OutOfService = v != 0
			}
		case zcl.AttrBinaryOutputStatusFlags:
			if v, ok := r.Value.(uint8); ok {
				info.StatusFlags = v
			}
		case zcl.AttrBinaryOutputPolarity:
			if v, ok := r.Value.(uint8); ok {
				info.Polarity = v
			}
		}
	}

	return info, nil
}

// SetBinaryOutput sets the output value on a BinaryOutput device.
// value specifies the desired output state (true=active, false=inactive).
// This writes to the PresentValue attribute (0x0055) which is writable.
func (a *Adapter) SetBinaryOutput(ctx context.Context, nwkAddr uint16, endpoint uint8, value bool) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrBinaryOutputPresentValue: {
			Type:  zcl.TypeBoolean,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryOutput, values)
}

// GetBinaryOutputValue reads just the current output value from a BinaryOutput device.
// This is a convenience method that returns only the PresentValue attribute.
// Returns true for active state, false for inactive state.
func (a *Adapter) GetBinaryOutputValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (bool, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBinaryOutput, zcl.AttrBinaryOutputPresentValue)
	if err != nil {
		return false, fmt.Errorf("failed to read binary output value: %w", err)
	}

	if len(results) == 0 {
		return false, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return false, fmt.Errorf("read binary output value returned status 0x%02X", result.Status)
	}

	// Convert value to bool
	switch v := result.Value.(type) {
	case bool:
		return v, nil
	case uint8:
		return v != 0, nil
	default:
		return false, fmt.Errorf("unexpected value type: %T", result.Value)
	}
}

// MultistateOutput Cluster (0x0013)

// MultistateOutputInfo contains the status information for a MultistateOutput device.
type MultistateOutputInfo struct {
	PresentValue   uint16 // Current state value
	NumberOfStates uint16 // Number of possible states
	OutOfService   bool   // Out of service flag
	StatusFlags    uint8  // Status flags bitmap
}

// GetMultistateOutput reads the current status from a MultistateOutput device.
// Returns present value, number of states, out of service flag, and status flags.
func (a *Adapter) GetMultistateOutput(ctx context.Context, nwkAddr uint16, endpoint uint8) (*MultistateOutputInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrMultistateOutputPresentValue,
		zcl.AttrMultistateOutputNumberOfStates,
		zcl.AttrMultistateOutputOutOfService,
		zcl.AttrMultistateOutputStatusFlags,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMultistateOutput, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read multistate output attributes: %w", err)
	}

	info := &MultistateOutputInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrMultistateOutputPresentValue:
			if v, ok := toUint16(r.Value); ok {
				info.PresentValue = v
			}
		case zcl.AttrMultistateOutputNumberOfStates:
			if v, ok := toUint16(r.Value); ok {
				info.NumberOfStates = v
			}
		case zcl.AttrMultistateOutputOutOfService:
			if v, ok := r.Value.(bool); ok {
				info.OutOfService = v
			} else if v, ok := r.Value.(uint8); ok {
				info.OutOfService = v != 0
			}
		case zcl.AttrMultistateOutputStatusFlags:
			if v, ok := r.Value.(uint8); ok {
				info.StatusFlags = v
			}
		}
	}

	return info, nil
}

// SetMultistateOutput sets the output state on a MultistateOutput device.
// value specifies the desired state (1 to NumberOfStates).
// This writes to the PresentValue attribute (0x0055) which is writable.
func (a *Adapter) SetMultistateOutput(ctx context.Context, nwkAddr uint16, endpoint uint8, value uint16) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrMultistateOutputPresentValue: {
			Type:  zcl.TypeUint16,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMultistateOutput, values)
}
// MultistateValue Cluster (0x0014)

// MultistateValueInfo contains the status information for a MultistateValue device.
type MultistateValueInfo struct {
	PresentValue   uint16            // Current value (read/write)
	NumberOfStates uint16            // Number of possible states
	OutOfService   bool              // Out of service flag
	StatusFlags    uint8             // Status flags bitmap
	Reliability    zcl.IOReliability // Reliability state
}

// GetMultistateValue reads the current status from a MultistateValue device.
// Returns present value, number of states, out of service flag, status flags, and reliability.
func (a *Adapter) GetMultistateValue(ctx context.Context, nwkAddr uint16, endpoint uint8) (*MultistateValueInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrMultistateValuePresentValue,
		zcl.AttrMultistateValueNumberOfStates,
		zcl.AttrMultistateValueOutOfService,
		zcl.AttrMultistateValueStatusFlags,
		zcl.AttrMultistateValueReliability,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMultistateValue, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read multistate value attributes: %w", err)
	}

	info := &MultistateValueInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrMultistateValuePresentValue:
			if v, ok := toUint16(r.Value); ok {
				info.PresentValue = v
			}
		case zcl.AttrMultistateValueNumberOfStates:
			if v, ok := toUint16(r.Value); ok {
				info.NumberOfStates = v
			}
		case zcl.AttrMultistateValueOutOfService:
			if v, ok := r.Value.(bool); ok {
				info.OutOfService = v
			} else if v, ok := r.Value.(uint8); ok {
				info.OutOfService = v != 0
			}
		case zcl.AttrMultistateValueStatusFlags:
			if v, ok := r.Value.(uint8); ok {
				info.StatusFlags = v
			}
		case zcl.AttrMultistateValueReliability:
			if v, ok := r.Value.(uint8); ok {
				info.Reliability = zcl.IOReliability(v)
			}
		}
	}

	return info, nil
}

// SetMultistateValue sets the value on a MultistateValue device.
// value specifies the desired state (1 to NumberOfStates).
// This writes to the PresentValue attribute (0x0055) which is read/write.
func (a *Adapter) SetMultistateValue(ctx context.Context, nwkAddr uint16, endpoint uint8, value uint16) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrMultistateValuePresentValue: {
			Type:  zcl.TypeUint16,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMultistateValue, values)
}

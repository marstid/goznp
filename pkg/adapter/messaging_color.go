package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// Color Control Cluster (0x0300)

// SetColorTemperature sets the color temperature on a tunable white light.
// tempMireds is the color temperature in mireds (1,000,000 / Kelvin).
// For example: 2700K = 370 mireds, 6500K = 154 mireds.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) SetColorTemperature(ctx context.Context, nwkAddr uint16, endpoint uint8, tempMireds uint16, transitionTime uint16) error {
	// MoveToColorTemp payload: colorTempMireds (2 bytes LE) + transitionTime (2 bytes LE)
	payload := []byte{
		byte(tempMireds & 0xFF),
		byte(tempMireds >> 8),
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.CmdColorMoveToColorTemp, payload)
}

// SetColorKelvin sets the color temperature using Kelvin values.
// kelvin is the color temperature in Kelvin (typically 2000-6500K for Hue lights).
// The function converts Kelvin to mireds using the conversion function from the zcl package.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
// Returns an error if the kelvin value is outside the valid range (1500-10000K).
func (a *Adapter) SetColorKelvin(ctx context.Context, nwkAddr uint16, endpoint uint8, kelvin uint16, transitionTime uint16) error {
	// Validate kelvin range (1500-10000K is a reasonable range for most lighting)
	// Hue lights typically support 2000-6500K
	if kelvin < 1500 || kelvin > 10000 {
		return fmt.Errorf("kelvin value %d out of valid range (1500-10000)", kelvin)
	}

	// Convert Kelvin to mireds
	mireds := zcl.KelvinToMireds(kelvin)

	// Use the existing SetColorTemperature method
	return a.SetColorTemperature(ctx, nwkAddr, endpoint, mireds, transitionTime)
}

// GetColorTemperature reads the current color temperature from a device.
// Returns the color temperature in mireds.
func (a *Adapter) GetColorTemperature(ctx context.Context, nwkAddr uint16, endpoint uint8) (uint16, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.AttrColorTempMireds)
	if err != nil {
		return 0, fmt.Errorf("failed to read color temperature: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read color temperature returned status 0x%02X", result.Status)
	}

	if temp, ok := result.Value.(uint16); ok {
		return temp, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// ColorTempInfo contains color temperature range information.
type ColorTempInfo struct {
	CurrentMireds uint16 // Current color temp in mireds
	MinMireds     uint16 // Minimum (warmest, lowest Kelvin)
	MaxMireds     uint16 // Maximum (coolest, highest Kelvin)
}

// GetColorTempInfo reads color temperature and range from a device.
func (a *Adapter) GetColorTempInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*ColorTempInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrColorTempMireds,
		zcl.AttrColorTempMin,
		zcl.AttrColorTempMax,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read color temp info: %w", err)
	}

	info := &ColorTempInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}
		val, ok := r.Value.(uint16)
		if !ok {
			continue
		}
		switch r.AttributeID {
		case zcl.AttrColorTempMireds:
			info.CurrentMireds = val
		case zcl.AttrColorTempMin:
			info.MinMireds = val
		case zcl.AttrColorTempMax:
			info.MaxMireds = val
		}
	}

	return info, nil
}

// ColorState contains the current color state of a device.
type ColorState struct {
	Hue        uint8  // Current hue (0-254)
	Saturation uint8  // Current saturation (0-254)
	X          uint16 // Current X chromaticity coordinate
	Y          uint16 // Current Y chromaticity coordinate
	ColorMode  uint8  // 0=HS, 1=XY, 2=ColorTemp
}

// SetHueSaturation sets both the hue and saturation of a color-capable light.
// hue is 0-254 (0=red, 85=green, 170=blue), saturation is 0-254 (0=white, 254=fully saturated).
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) SetHueSaturation(ctx context.Context, nwkAddr uint16, endpoint uint8, hue uint8, saturation uint8, transitionTime uint16) error {
	// MoveToHueSaturation payload: hue (1 byte) + saturation (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		hue,
		saturation,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.CmdColorMoveToHueSaturation, payload)
}

// SetColorXY sets the color using CIE XY chromaticity coordinates.
// x and y are 0-65535 representing the CIE 1931 color space coordinates.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) SetColorXY(ctx context.Context, nwkAddr uint16, endpoint uint8, x uint16, y uint16, transitionTime uint16) error {
	// MoveToColor payload: x (2 bytes LE) + y (2 bytes LE) + transitionTime (2 bytes LE)
	payload := []byte{
		byte(x & 0xFF),
		byte(x >> 8),
		byte(y & 0xFF),
		byte(y >> 8),
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.CmdColorMoveToColor, payload)
}

// SetColorRGB sets the color using RGB values.
// RGB values are 0-255. The function converts RGB to CIE XY color space
// using the conversion function from the zcl package.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) SetColorRGB(ctx context.Context, nwkAddr uint16, endpoint uint8, r, g, b uint8, transitionTime uint16) error {
	// Convert RGB to XY coordinates
	x, y := zcl.RGBToXY(r, g, b)

	// Use the existing SetColorXY method
	return a.SetColorXY(ctx, nwkAddr, endpoint, x, y, transitionTime)
}

// SetHue sets only the hue of a color-capable light.
// hue is 0-254 (0=red, 85=green, 170=blue).
// direction: 0=shortest, 1=longest, 2=up, 3=down.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) SetHue(ctx context.Context, nwkAddr uint16, endpoint uint8, hue uint8, direction uint8, transitionTime uint16) error {
	// MoveToHue payload: hue (1 byte) + direction (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		hue,
		direction,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.CmdColorMoveToHue, payload)
}

// SetSaturation sets only the saturation of a color-capable light.
// saturation is 0-254 (0=white, 254=fully saturated).
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) SetSaturation(ctx context.Context, nwkAddr uint16, endpoint uint8, saturation uint8, transitionTime uint16) error {
	// MoveToSaturation payload: saturation (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		saturation,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, zcl.CmdColorMoveToSaturation, payload)
}

// GetColor reads the current color state from a device.
// Returns the current hue, saturation, X/Y coordinates, and color mode.
func (a *Adapter) GetColor(ctx context.Context, nwkAddr uint16, endpoint uint8) (*ColorState, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrColorCurrentHue,
		zcl.AttrColorCurrentSaturation,
		zcl.AttrColorCurrentX,
		zcl.AttrColorCurrentY,
		zcl.AttrColorMode,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterColorControl, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read color state: %w", err)
	}

	state := &ColorState{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrColorCurrentHue:
			if val, ok := r.Value.(uint8); ok {
				state.Hue = val
			}
		case zcl.AttrColorCurrentSaturation:
			if val, ok := r.Value.(uint8); ok {
				state.Saturation = val
			}
		case zcl.AttrColorCurrentX:
			if val, ok := r.Value.(uint16); ok {
				state.X = val
			}
		case zcl.AttrColorCurrentY:
			if val, ok := r.Value.(uint16); ok {
				state.Y = val
			}
		case zcl.AttrColorMode:
			if val, ok := r.Value.(uint8); ok {
				state.ColorMode = val
			}
		}
	}

	return state, nil
}

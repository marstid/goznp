package adapter

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

// Default coordinator endpoint (Home Automation profile).
const CoordinatorEndpoint = 1

// transactionID counter for ZCL messages.
var transactionID uint32

// nextTransactionID returns the next transaction ID.
func nextTransactionID() uint8 {
	return uint8(atomic.AddUint32(&transactionID, 1))
}

// AttributeResult represents a single attribute read result.
type AttributeResult struct {
	AttributeID zcl.AttributeID
	Status      uint8
	DataType    zcl.DataType
	Value       interface{}
}

// ReadAttributes reads attributes from a device endpoint.
func (a *Adapter) ReadAttributes(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, attributeIDs ...zcl.AttributeID) ([]AttributeResult, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build read attributes request
	seqNum := nextTransactionID()
	frame := zcl.BuildReadAttributesRequest(seqNum, attributeIDs...)

	// Send request and wait for response
	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(clusterID), frame)
	if err != nil {
		return nil, fmt.Errorf("read attributes failed: %w", err)
	}

	// Parse response
	if respFrame.CommandID != uint8(zcl.CmdReadAttributesResponse) {
		return nil, fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	readResults, err := zcl.ParseReadAttributesResponse(respFrame.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse read attributes response: %w", err)
	}

	// Convert to AttributeResult
	results := make([]AttributeResult, len(readResults))
	for i, r := range readResults {
		results[i] = AttributeResult{
			AttributeID: r.AttributeID,
			Status:      r.Status,
			DataType:    r.DataType,
			Value:       r.Value,
		}
	}

	return results, nil
}

// WriteAttributes writes attributes to a device endpoint.
func (a *Adapter) WriteAttributes(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, values map[zcl.AttributeID]zcl.AttributeValue) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	a.mu.Unlock()

	// Build write attributes request
	seqNum := nextTransactionID()
	frame := zcl.BuildWriteAttributesRequest(seqNum, values)

	// Send request and wait for response
	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(clusterID), frame)
	if err != nil {
		return fmt.Errorf("write attributes failed: %w", err)
	}

	// Parse response
	if respFrame.CommandID != uint8(zcl.CmdWriteAttributesResponse) {
		return fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	writeResults, err := zcl.ParseWriteAttributesResponse(respFrame.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse write attributes response: %w", err)
	}

	// Check if any writes failed
	for _, r := range writeResults {
		if r.Status != zcl.StatusSuccess {
			return fmt.Errorf("write attribute 0x%04X failed with status 0x%02X", r.AttributeID, r.Status)
		}
	}

	return nil
}

// ConfigureReporting configures a device to automatically report attribute changes.
// This allows devices to push updates instead of requiring polling.
//
// Parameters:
//   - nwkAddr: Network address of the target device
//   - endpoint: Endpoint number on the device
//   - clusterID: Cluster ID containing the attribute
//   - attributeID: Attribute ID to configure
//   - dataType: Data type of the attribute
//   - minInterval: Minimum reporting interval in seconds (0 = no minimum)
//   - maxInterval: Maximum reporting interval in seconds (0xFFFF = no periodic reports)
//   - reportableChange: Minimum change to trigger a report (for analog types, nil for discrete types)
//
// Example: Configure temperature sensor to report every 60-300s or on 0.5°C change:
//
//	err := adapter.ConfigureReporting(ctx, nwkAddr, endpoint, zcl.ClusterTempMeasurement,
//	    zcl.AttrTempMeasuredValue, zcl.TypeInt16, 60, 300, int16(50)) // 50 = 0.5°C in 0.01 units
func (a *Adapter) ConfigureReporting(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, attributeID zcl.AttributeID, dataType zcl.DataType, minInterval, maxInterval uint16, reportableChange interface{}) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	a.mu.Unlock()

	// Build configure reporting request
	seqNum := nextTransactionID()
	config := zcl.ReportingConfig{
		Direction:        0x00, // Device reports to coordinator
		AttributeID:      attributeID,
		DataType:         dataType,
		MinInterval:      minInterval,
		MaxInterval:      maxInterval,
		ReportableChange: reportableChange,
	}

	frame, err := zcl.BuildConfigureReportingRequest(seqNum, []zcl.ReportingConfig{config})
	if err != nil {
		return fmt.Errorf("failed to build configure reporting request: %w", err)
	}

	// Send request and wait for response
	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(clusterID), frame)
	if err != nil {
		return fmt.Errorf("configure reporting failed: %w", err)
	}

	// Parse response
	// Some devices (like Tuya) respond with Report Attributes (0x0A) instead of
	// ConfigureReportingResponse (0x07). We treat this as success since it means
	// the device is sending attribute reports.
	if respFrame.CommandID == uint8(zcl.CmdReportAttributes) {
		// Device responded with an attribute report - treat as success
		return nil
	}

	if respFrame.CommandID != uint8(zcl.CmdConfigureReportingResp) {
		return fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	results, err := zcl.ParseConfigureReportingResponse(respFrame.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse configure reporting response: %w", err)
	}

	// Check if configuration succeeded
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			return fmt.Errorf("configure reporting for attribute 0x%04X failed with status 0x%02X", r.AttributeID, r.Status)
		}
	}

	return nil
}

// ReportingConfigResult contains the reporting configuration for a single attribute.
type ReportingConfigResult struct {
	Status           uint8
	Direction        uint8
	AttributeID      zcl.AttributeID
	DataType         zcl.DataType
	MinInterval      uint16
	MaxInterval      uint16
	ReportableChange interface{}
	TimeoutPeriod    uint16
}

// ReadReportingConfig reads the current reporting configuration for attributes on a device.
// This queries the device for how it's configured to send attribute reports.
func (a *Adapter) ReadReportingConfig(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, attributeIDs ...zcl.AttributeID) ([]ReportingConfigResult, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build read reporting config request
	seqNum := nextTransactionID()
	records := make([]zcl.ReadReportingConfigRecord, len(attributeIDs))
	for i, attrID := range attributeIDs {
		records[i] = zcl.ReadReportingConfigRecord{
			Direction:   0x00, // Query outgoing reports (device -> coordinator)
			AttributeID: attrID,
		}
	}

	frame := zcl.BuildReadReportingConfigRequest(seqNum, records)

	// Send request and wait for response
	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(clusterID), frame)
	if err != nil {
		return nil, fmt.Errorf("read reporting config failed: %w", err)
	}

	// Parse response
	if respFrame.CommandID != uint8(zcl.CmdReadReportingConfigResp) {
		return nil, fmt.Errorf("unexpected response command: 0x%02X (expected 0x%02X)", respFrame.CommandID, zcl.CmdReadReportingConfigResp)
	}

	zclRecords, err := zcl.ParseReadReportingConfigResponse(respFrame.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse read reporting config response: %w", err)
	}

	// Convert to adapter types
	results := make([]ReportingConfigResult, len(zclRecords))
	for i, r := range zclRecords {
		results[i] = ReportingConfigResult{
			Status:           r.Status,
			Direction:        r.Direction,
			AttributeID:      r.AttributeID,
			DataType:         r.DataType,
			MinInterval:      r.MinInterval,
			MaxInterval:      r.MaxInterval,
			ReportableChange: r.ReportableChange,
			TimeoutPeriod:    r.TimeoutPeriod,
		}
	}

	return results, nil
}

// SendClusterCommand sends a cluster-specific command to a device.
func (a *Adapter) SendClusterCommand(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, commandID uint8, payload []byte) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Build cluster command frame
	seqNum := nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, commandID, payload)

	// Build AF data request
	req := znp.DataRequest{
		DstAddr:     nwkAddr,
		DstEndpoint: endpoint,
		SrcEndpoint: CoordinatorEndpoint,
		ClusterID:   uint16(clusterID),
		TransID:     seqNum,
		Options:     znp.AfOptionAckRequest,
		Radius:      30,
		Data:        frame.ToBytes(),
	}

	// Send request
	status, err := znpClient.AfDataRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("data request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("data request returned status 0x%02X", status)
	}

	// Wait for data confirm
	confirm, err := znpClient.WaitForDataConfirm(ctx, seqNum, 5*time.Second)
	if err != nil {
		return fmt.Errorf("waiting for data confirm: %w", err)
	}

	if confirm.Status != 0 {
		return fmt.Errorf("data confirm returned status 0x%02X", confirm.Status)
	}

	return nil
}

// TurnOn sends On command to a device.
func (a *Adapter) TurnOn(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOn, nil)
}

// TurnOff sends Off command to a device.
func (a *Adapter) TurnOff(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOff, nil)
}

// Toggle sends Toggle command to a device.
func (a *Adapter) Toggle(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffToggle, nil)
}

// OffWithEffect turns off a device with a visual effect (typically for lights).
// effectID specifies the effect type:
//   - 0x00: DelayedAllOff - fade to off over 0.8 seconds
//   - 0x01: DyingLight - 50% dim down in 0.8s then fade to off in 12s
//
// effectVariant is an effect-specific variant value (typically 0 for default).
func (a *Adapter) OffWithEffect(ctx context.Context, nwkAddr uint16, endpoint uint8, effectID, effectVariant uint8) error {
	// OffWithEffect payload: effectId (1 byte) + effectVariant (1 byte)
	payload := []byte{effectID, effectVariant}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOffWithEffect, payload)
}

// OnWithTimedOff turns on a device for a specified duration.
// The device will automatically turn off after the specified time.
// onTime and offWaitTime are in 1/10th seconds (e.g., 10 = 1 second).
//
// Parameters:
//   - onTime: Duration the device stays on (1/10th seconds)
//   - offWaitTime: Additional delay before turning off (1/10th seconds)
//
// The onOffControl parameter is typically 0x00 for normal operation.
func (a *Adapter) OnWithTimedOff(ctx context.Context, nwkAddr uint16, endpoint uint8, onTime, offWaitTime uint16) error {
	// OnWithTimedOff payload: onOffControl (1 byte) + onTime (2 bytes LE) + offWaitTime (2 bytes LE)
	payload := []byte{
		0x00, // onOffControl (0x00 = accept command only if device is currently off)
		byte(onTime & 0xFF),
		byte(onTime >> 8),
		byte(offWaitTime & 0xFF),
		byte(offWaitTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.CmdOnOffOnWithTimedOff, payload)
}

// SetStartupOnOff configures the device behavior when it powers on.
// value specifies the startup behavior:
//   - 0x00: Turn off when powered on
//   - 0x01: Turn on when powered on
//   - 0x02: Toggle state when powered on
//   - 0xFF: Restore previous state when powered on
//
// This setting is persistent across power cycles.
func (a *Adapter) SetStartupOnOff(ctx context.Context, nwkAddr uint16, endpoint uint8, value uint8) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrStartUpOnOff: {
			Type:  zcl.TypeEnum8,
			Value: value,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, values)
}

// SetBrightness sets the brightness level on a dimmable device.
// level is 0-254 (0=off, 254=full brightness).
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
// Uses MoveToLevelWithOnOff command which also handles on/off state.
func (a *Adapter) SetBrightness(ctx context.Context, nwkAddr uint16, endpoint uint8, level uint8, transitionTime uint16) error {
	// MoveToLevelWithOnOff payload: level (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		level,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelMoveToLevelWithOnOff, payload)
}

// GetBrightness reads the current brightness level from a device.
// Returns 0-254 (0=off, 254=full brightness).
func (a *Adapter) GetBrightness(ctx context.Context, nwkAddr uint16, endpoint uint8) (uint8, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.AttrLevelCurrentLevel)
	if err != nil {
		return 0, fmt.Errorf("failed to read brightness level: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read brightness returned status 0x%02X", result.Status)
	}

	if level, ok := result.Value.(uint8); ok {
		return level, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// MoveLevel starts continuously changing the level at the specified rate.
// The level will continue changing until it reaches the limit or StopLevel is called.
//
// Parameters:
//   - moveMode: Direction of movement (zcl.MoveModeUp or zcl.MoveModeDown)
//   - rate: Units per second to change the level (0 = use device's DefaultMoveRate attribute)
//
// Example: Increase brightness at 50 units/second:
//
//	err := adapter.MoveLevel(ctx, nwkAddr, endpoint, uint8(zcl.MoveModeUp), 50)
func (a *Adapter) MoveLevel(ctx context.Context, nwkAddr uint16, endpoint uint8, moveMode, rate uint8) error {
	// Move payload: moveMode (1 byte) + rate (1 byte)
	payload := []byte{moveMode, rate}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelMove, payload)
}

// StepLevel changes the level by a fixed amount over a transition time.
// This is useful for dimming/brightening by a specific increment.
//
// Parameters:
//   - stepMode: Direction of step (zcl.StepModeUp or zcl.StepModeDown)
//   - stepSize: Amount to change the level (0-254)
//   - transitionTime: Time for the transition in tenths of a second (e.g., 10 = 1 second)
//
// Example: Decrease brightness by 20 units over 0.5 seconds:
//
//	err := adapter.StepLevel(ctx, nwkAddr, endpoint, uint8(zcl.StepModeDown), 20, 5)
func (a *Adapter) StepLevel(ctx context.Context, nwkAddr uint16, endpoint uint8, stepMode, stepSize uint8, transitionTime uint16) error {
	// Step payload: stepMode (1 byte) + stepSize (1 byte) + transitionTime (2 bytes LE)
	payload := []byte{
		stepMode,
		stepSize,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelStep, payload)
}

// StopLevel stops any ongoing level change started by MoveLevel or StepLevel.
// This command immediately halts the level transition and maintains the current level.
//
// Example:
//
//	err := adapter.StopLevel(ctx, nwkAddr, endpoint)
func (a *Adapter) StopLevel(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// Stop command has no payload
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, zcl.CmdLevelStop, nil)
}

// SetStartupLevel configures the brightness level the device uses when powered on.
// This persists across power cycles, allowing devices to remember their preferred startup state.
//
// Parameters:
//   - level: The startup level (0-254 for specific level, 255 to restore previous level)
//
// Special values:
//   - 0-254: Set to a specific brightness level on power-up
//   - 255 (0xFF): Restore the level from before power loss
//
// Example: Set device to turn on at 50% brightness:
//
//	err := adapter.SetStartupLevel(ctx, nwkAddr, endpoint, 127)
//
// Example: Set device to restore previous level on power-up:
//
//	err := adapter.SetStartupLevel(ctx, nwkAddr, endpoint, 255)
func (a *Adapter) SetStartupLevel(ctx context.Context, nwkAddr uint16, endpoint uint8, level uint8) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrLevelStartupLevel: {
			Type:  zcl.TypeUint8,
			Value: level,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterLevelControl, values)
}

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

// ============================================================================
// Identify Cluster (0x0003)
// ============================================================================

// Identify starts the identify mode on a device for the specified duration.
// The device will perform a visual/audible identification (e.g., flashing lights).
// Duration is in seconds. Use 0 to stop identifying.
func (a *Adapter) Identify(ctx context.Context, nwkAddr uint16, endpoint uint8, durationSecs uint16) error {
	// Identify payload: identifyTime (2 bytes LE)
	payload := []byte{
		byte(durationSecs & 0xFF),
		byte(durationSecs >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIdentify, zcl.CmdIdentify, payload)
}

// TriggerEffect triggers a specific identification effect on a device.
// Effects: Blink (0x00), Breathe (0x01), Okay (0x02), ChannelChange (0x0B),
// FinishEffect (0xFE), StopEffect (0xFF)
func (a *Adapter) TriggerEffect(ctx context.Context, nwkAddr uint16, endpoint uint8, effectID uint8, effectVariant uint8) error {
	// TriggerEffect payload: effectId (1 byte) + effectVariant (1 byte)
	payload := []byte{effectID, effectVariant}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIdentify, zcl.CmdTriggerEffect, payload)
}

// ============================================================================
// Groups Cluster (0x0004)
// ============================================================================

// AddToGroup adds a device endpoint to a group.
// groupID is the 16-bit group address (1-65535).
// groupName is optional (can be empty string).
func (a *Adapter) AddToGroup(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, groupName string) error {
	// AddGroup payload: groupId (2 bytes LE) + groupName (string with length prefix)
	nameBytes := []byte(groupName)
	if len(nameBytes) > 16 {
		nameBytes = nameBytes[:16] // ZCL limits group names to 16 chars
	}

	payload := make([]byte, 3+len(nameBytes))
	payload[0] = byte(groupID & 0xFF)
	payload[1] = byte(groupID >> 8)
	payload[2] = byte(len(nameBytes))
	copy(payload[3:], nameBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterGroups, zcl.CmdGroupsAdd, payload)
}

// RemoveFromGroup removes a device endpoint from a group.
func (a *Adapter) RemoveFromGroup(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16) error {
	// RemoveGroup payload: groupId (2 bytes LE)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterGroups, zcl.CmdGroupsRemove, payload)
}

// RemoveFromAllGroups removes a device endpoint from all groups.
func (a *Adapter) RemoveFromAllGroups(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterGroups, zcl.CmdGroupsRemoveAll, nil)
}

// GroupMembership contains the groups a device belongs to.
type GroupMembership struct {
	Capacity uint8    // Remaining capacity for group memberships (0xFF = unknown)
	Groups   []uint16 // List of group IDs the device belongs to
}

// GetGroupMembership queries which groups a device endpoint belongs to.
func (a *Adapter) GetGroupMembership(ctx context.Context, nwkAddr uint16, endpoint uint8) (*GroupMembership, error) {
	// GetGroupMembership payload: groupCount (1 byte) + groupList (empty = query all)
	payload := []byte{0x00} // Query all groups

	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build and send request
	seqNum := nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, zcl.CmdGroupsGetMembership, payload)

	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(zcl.ClusterGroups), frame)
	if err != nil {
		return nil, fmt.Errorf("get group membership failed: %w", err)
	}

	// Parse response (command 0x02)
	if respFrame.CommandID != zcl.CmdGroupsGetMembershipResponse {
		return nil, fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	if len(respFrame.Payload) < 2 {
		return nil, fmt.Errorf("response too short")
	}

	result := &GroupMembership{
		Capacity: respFrame.Payload[0],
		Groups:   make([]uint16, 0),
	}

	groupCount := respFrame.Payload[1]
	offset := 2
	for i := uint8(0); i < groupCount && offset+1 < len(respFrame.Payload); i++ {
		groupID := uint16(respFrame.Payload[offset]) | uint16(respFrame.Payload[offset+1])<<8
		result.Groups = append(result.Groups, groupID)
		offset += 2
	}

	return result, nil
}

// SendGroupCommand sends a cluster command to all devices in a group.
// This uses group addressing (multicast) so the command is received by all
// group members simultaneously. groupID is the 16-bit group address.
// clusterID is the cluster to send the command on, commandID is the
// cluster-specific command, and payload is the command payload.
//
// Example: Turn on all lights in group 1:
//
//	err := adapter.SendGroupCommand(ctx, 1, zcl.ClusterOnOff, zcl.CmdOnOffOn, nil)
func (a *Adapter) SendGroupCommand(ctx context.Context, groupID uint16, clusterID zcl.ClusterID, commandID uint8, payload []byte) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Build cluster command frame
	seqNum := nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, commandID, payload)

	// Build AF data request with group addressing
	// For group addressing, we use the group ID as the destination address
	// and broadcast endpoint (0xFF) as the destination endpoint
	req := znp.DataRequest{
		DstAddr:     groupID, // Group ID as destination
		DstEndpoint: 0xFF,    // Broadcast endpoint for group commands
		SrcEndpoint: CoordinatorEndpoint,
		ClusterID:   uint16(clusterID),
		TransID:     seqNum,
		Options:     znp.AfOptionNone, // No ACK for group commands
		Radius:      30,
		Data:        frame.ToBytes(),
	}

	// Send request
	status, err := znpClient.AfDataRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("group data request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("group data request returned status 0x%02X", status)
	}

	// Wait for data confirm (no response expected from devices)
	confirm, err := znpClient.WaitForDataConfirm(ctx, seqNum, 5*time.Second)
	if err != nil {
		return fmt.Errorf("waiting for data confirm: %w", err)
	}

	if confirm.Status != 0 {
		return fmt.Errorf("data confirm returned status 0x%02X", confirm.Status)
	}

	return nil
}

// GroupTurnOn sends On command to all devices in a group.
func (a *Adapter) GroupTurnOn(ctx context.Context, groupID uint16) error {
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterOnOff, zcl.CmdOnOffOn, nil)
}

// GroupTurnOff sends Off command to all devices in a group.
func (a *Adapter) GroupTurnOff(ctx context.Context, groupID uint16) error {
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterOnOff, zcl.CmdOnOffOff, nil)
}

// GroupToggle sends Toggle command to all devices in a group.
func (a *Adapter) GroupToggle(ctx context.Context, groupID uint16) error {
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterOnOff, zcl.CmdOnOffToggle, nil)
}

// GroupSetBrightness sets the brightness level on all dimmable devices in a group.
// level is 0-254 (0=off, 254=full brightness).
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) GroupSetBrightness(ctx context.Context, groupID uint16, level uint8, transitionTime uint16) error {
	payload := []byte{
		level,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterLevelControl, zcl.CmdLevelMoveToLevelWithOnOff, payload)
}

// GroupRecallScene recalls a scene on all devices in a group.
// sceneID is the scene to recall (0-255).
func (a *Adapter) GroupRecallScene(ctx context.Context, groupID uint16, sceneID uint8) error {
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterScenes, zcl.CmdScenesRecall, payload)
}

// ============================================================================
// Scenes Cluster (0x0005)
// ============================================================================

// StoreScene stores the current device state as a scene.
// The device saves its current attribute values (brightness, color, etc.) to the scene.
// groupID must be a valid group the device belongs to (or 0x0000 for global scenes).
// sceneID is 0-255.
func (a *Adapter) StoreScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8) error {
	// StoreScene payload: groupId (2 bytes LE) + sceneId (1 byte)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesStore, payload)
}

// AddScene adds a scene with explicit parameters.
// This is an advanced command that allows specifying scene details explicitly
// rather than capturing current device state like StoreScene does.
// groupID must be a valid group the device belongs to (or 0x0000 for global scenes).
// sceneID is 0-255.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
// sceneName is optional (max 16 characters).
// Note: This only creates the scene metadata. To capture device state, use StoreScene instead.
func (a *Adapter) AddScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8, transitionTime uint16, sceneName string) error {
	// AddScene payload: groupId (2 bytes LE) + sceneId (1 byte) + transitionTime (2 bytes LE) + sceneName (string with length prefix)
	nameBytes := []byte(sceneName)
	if len(nameBytes) > 16 {
		nameBytes = nameBytes[:16] // ZCL limits scene names to 16 chars
	}

	payload := make([]byte, 6+len(nameBytes))
	payload[0] = byte(groupID & 0xFF)
	payload[1] = byte(groupID >> 8)
	payload[2] = sceneID
	payload[3] = byte(transitionTime & 0xFF)
	payload[4] = byte(transitionTime >> 8)
	payload[5] = byte(len(nameBytes))
	copy(payload[6:], nameBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesAdd, payload)
}

// RecallScene activates a previously stored scene.
// The device transitions to the saved attribute values.
// transitionTime is optional (0xFFFF = use scene's stored transition time).
func (a *Adapter) RecallScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8, transitionTime *uint16) error {
	// RecallScene payload: groupId (2 bytes LE) + sceneId (1 byte) + [transitionTime (2 bytes LE)]
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}

	// Add optional transition time
	if transitionTime != nil {
		payload = append(payload, byte(*transitionTime&0xFF), byte(*transitionTime>>8))
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesRecall, payload)
}

// RemoveScene removes a specific scene from the device.
func (a *Adapter) RemoveScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8) error {
	// RemoveScene payload: groupId (2 bytes LE) + sceneId (1 byte)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesRemove, payload)
}

// RemoveAllScenes removes all scenes for a group from the device.
func (a *Adapter) RemoveAllScenes(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16) error {
	// RemoveAllScenes payload: groupId (2 bytes LE)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesRemoveAll, payload)
}

// SceneMembership contains the scenes stored on a device for a group.
type SceneMembership struct {
	Status   uint8   // ZCL status (0 = success)
	Capacity uint8   // Remaining capacity for scenes (0xFF = unknown)
	GroupID  uint16  // The group these scenes belong to
	Scenes   []uint8 // List of scene IDs
}

// GetSceneMembership queries which scenes are stored on a device for a group.
func (a *Adapter) GetSceneMembership(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16) (*SceneMembership, error) {
	// GetSceneMembership payload: groupId (2 bytes LE)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
	}

	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build and send request
	seqNum := nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, zcl.CmdScenesGetMembership, payload)

	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(zcl.ClusterScenes), frame)
	if err != nil {
		return nil, fmt.Errorf("get scene membership failed: %w", err)
	}

	// Parse response (command 0x06)
	if respFrame.CommandID != zcl.CmdScenesGetMembershipResponse {
		return nil, fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	if len(respFrame.Payload) < 4 {
		return nil, fmt.Errorf("response too short")
	}

	result := &SceneMembership{
		Status:   respFrame.Payload[0],
		Capacity: respFrame.Payload[1],
		GroupID:  uint16(respFrame.Payload[2]) | uint16(respFrame.Payload[3])<<8,
		Scenes:   make([]uint8, 0),
	}

	// Only parse scene list if status is success
	if result.Status == zcl.StatusSuccess && len(respFrame.Payload) > 4 {
		sceneCount := respFrame.Payload[4]
		for i := uint8(0); i < sceneCount && int(5+i) < len(respFrame.Payload); i++ {
			result.Scenes = append(result.Scenes, respFrame.Payload[5+i])
		}
	}

	return result, nil
}

// ============================================================================
// Basic Cluster (0x0000)
// ============================================================================

// ResetToFactoryDefaults sends the reset to factory defaults command to a device.
// This resets all writeable attributes in the Basic cluster to their factory default values.
// Note: The device behavior may vary - some devices may reset all settings, while others
// may only reset specific attributes. Refer to the device documentation for details.
func (a *Adapter) ResetToFactoryDefaults(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterBasic, zcl.CmdBasicResetToFactoryDefaults, nil)
}

// DeviceInfo contains all Basic cluster attributes for a device.
type DeviceInfo struct {
	// Core Device Information
	ZCLVersion       *uint8  // ZCL version supported
	AppVersion       *uint8  // Application version
	StackVersion     *uint8  // Stack version
	HWVersion        *uint8  // Hardware version
	ManufacturerName *string // Manufacturer name
	ModelIdentifier  *string // Model identifier
	DateCode         *string // Manufacturing date code (YYYYMMDD format)
	PowerSource      *zcl.PowerSource
	SWBuildID        *string // Software build ID

	// Extended Device Information
	ProductCode                *string // Product code (octet string)
	ProductURL                 *string // Product URL
	ManufacturerVersionDetails *string // Manufacturer version details
	SerialNumber               *string // Serial number
	ProductLabel               *string // Product label

	// Optional Device Information
	LocationDescription *string // Physical location description (max 16 chars)
	PhysicalEnvironment *uint8  // Physical environment type (enum8)
	DeviceEnabled       *bool   // Whether device is enabled
	AlarmMask           *uint8  // Alarm mask (bitmap8)
	DisableLocalConfig  *uint8  // Local config disable mask (bitmap8)

	// Generic Device Information (ZCL 7+)
	GenericDeviceClass *uint8 // Generic device class (enum8)
	GenericDeviceType  *uint8 // Generic device type (enum8)
}

// ReadDeviceInfo reads all Basic cluster attributes from a device.
// Returns a DeviceInfo struct with all available attributes. Attributes that
// are not supported or fail to read will be nil.
func (a *Adapter) ReadDeviceInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DeviceInfo, error) {
	// Define all Basic cluster attributes to read
	attrs := []zcl.AttributeID{
		zcl.AttrBasicZCLVersion,
		zcl.AttrBasicAppVersion,
		zcl.AttrBasicStackVersion,
		zcl.AttrBasicHWVersion,
		zcl.AttrBasicManufacturerName,
		zcl.AttrBasicModelIdentifier,
		zcl.AttrBasicDateCode,
		zcl.AttrBasicPowerSource,
		zcl.AttrBasicProductCode,
		zcl.AttrBasicProductURL,
		zcl.AttrBasicManufacturerVersionDetails,
		zcl.AttrBasicSerialNumber,
		zcl.AttrBasicProductLabel,
		zcl.AttrBasicLocationDescription,
		zcl.AttrBasicPhysicalEnvironment,
		zcl.AttrBasicDeviceEnabled,
		zcl.AttrBasicAlarmMask,
		zcl.AttrBasicDisableLocalConfig,
		zcl.AttrBasicSWBuildID,
		zcl.AttrBasicGenericDeviceClass,
		zcl.AttrBasicGenericDeviceType,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBasic, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read basic cluster attributes: %w", err)
	}

	info := &DeviceInfo{}

	// Parse results and populate DeviceInfo struct
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue // Skip attributes that failed to read
		}

		switch r.AttributeID {
		case zcl.AttrBasicZCLVersion:
			if v, ok := r.Value.(uint8); ok {
				info.ZCLVersion = &v
			}
		case zcl.AttrBasicAppVersion:
			if v, ok := r.Value.(uint8); ok {
				info.AppVersion = &v
			}
		case zcl.AttrBasicStackVersion:
			if v, ok := r.Value.(uint8); ok {
				info.StackVersion = &v
			}
		case zcl.AttrBasicHWVersion:
			if v, ok := r.Value.(uint8); ok {
				info.HWVersion = &v
			}
		case zcl.AttrBasicManufacturerName:
			if v, ok := r.Value.(string); ok {
				info.ManufacturerName = &v
			}
		case zcl.AttrBasicModelIdentifier:
			if v, ok := r.Value.(string); ok {
				info.ModelIdentifier = &v
			}
		case zcl.AttrBasicDateCode:
			if v, ok := r.Value.(string); ok {
				info.DateCode = &v
			}
		case zcl.AttrBasicPowerSource:
			if v, ok := r.Value.(uint8); ok {
				ps := zcl.PowerSource(v)
				info.PowerSource = &ps
			}
		case zcl.AttrBasicLocationDescription:
			if v, ok := r.Value.(string); ok {
				info.LocationDescription = &v
			}
		case zcl.AttrBasicPhysicalEnvironment:
			if v, ok := r.Value.(uint8); ok {
				info.PhysicalEnvironment = &v
			}
		case zcl.AttrBasicDeviceEnabled:
			if v, ok := r.Value.(bool); ok {
				info.DeviceEnabled = &v
			} else if v, ok := r.Value.(uint8); ok {
				enabled := v != 0
				info.DeviceEnabled = &enabled
			}
		case zcl.AttrBasicAlarmMask:
			if v, ok := r.Value.(uint8); ok {
				info.AlarmMask = &v
			}
		case zcl.AttrBasicDisableLocalConfig:
			if v, ok := r.Value.(uint8); ok {
				info.DisableLocalConfig = &v
			}
		case zcl.AttrBasicSWBuildID:
			if v, ok := r.Value.(string); ok {
				info.SWBuildID = &v
			}
		case zcl.AttrBasicProductCode:
			if v, ok := r.Value.(string); ok {
				info.ProductCode = &v
			}
		case zcl.AttrBasicProductURL:
			if v, ok := r.Value.(string); ok {
				info.ProductURL = &v
			}
		case zcl.AttrBasicManufacturerVersionDetails:
			if v, ok := r.Value.(string); ok {
				info.ManufacturerVersionDetails = &v
			}
		case zcl.AttrBasicSerialNumber:
			if v, ok := r.Value.(string); ok {
				info.SerialNumber = &v
			}
		case zcl.AttrBasicProductLabel:
			if v, ok := r.Value.(string); ok {
				info.ProductLabel = &v
			}
		case zcl.AttrBasicGenericDeviceClass:
			if v, ok := r.Value.(uint8); ok {
				info.GenericDeviceClass = &v
			}
		case zcl.AttrBasicGenericDeviceType:
			if v, ok := r.Value.(uint8); ok {
				info.GenericDeviceType = &v
			}
		}
	}

	return info, nil
}

// ============================================================================
// On/Off Cluster (0x0006)
// ============================================================================

// GetOnOffState reads the current on/off state.
func (a *Adapter) GetOnOffState(ctx context.Context, nwkAddr uint16, endpoint uint8) (bool, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.AttrOnOff)
	if err != nil {
		return false, fmt.Errorf("failed to read on/off state: %w", err)
	}

	if len(results) == 0 {
		return false, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return false, fmt.Errorf("read on/off state returned status 0x%02X", result.Status)
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

// DeviceStatus contains common device status information.
type DeviceStatus struct {
	NwkAddr        uint16
	Manufacturer   string
	Model          string
	PowerSource    zcl.PowerSource
	BatteryPercent *uint8 // nil if not available
	OnOff          *bool  // nil if not on/off device
}

// GetDeviceStatus queries common status attributes from a device.
func (a *Adapter) GetDeviceStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DeviceStatus, error) {
	status := &DeviceStatus{
		NwkAddr: nwkAddr,
	}

	// Read basic cluster attributes
	basicAttrs := []zcl.AttributeID{
		zcl.AttrBasicManufacturerName,
		zcl.AttrBasicModelIdentifier,
		zcl.AttrBasicPowerSource,
	}

	basicResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBasic, basicAttrs...)
	if err == nil {
		for _, r := range basicResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}

			switch r.AttributeID {
			case zcl.AttrBasicManufacturerName:
				if s, ok := r.Value.(string); ok {
					status.Manufacturer = s
				}
			case zcl.AttrBasicModelIdentifier:
				if s, ok := r.Value.(string); ok {
					status.Model = s
				}
			case zcl.AttrBasicPowerSource:
				if ps, ok := r.Value.(uint8); ok {
					status.PowerSource = zcl.PowerSource(ps)
				}
			}
		}
	}

	// Try to read battery percentage (may not be supported)
	batteryResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, zcl.AttrPowerBatteryPercentage)
	if err == nil && len(batteryResults) > 0 && batteryResults[0].Status == zcl.StatusSuccess {
		if bp, ok := batteryResults[0].Value.(uint8); ok {
			status.BatteryPercent = &bp
		}
	}

	// Try to read on/off state (may not be supported)
	onOffState, err := a.GetOnOffState(ctx, nwkAddr, endpoint)
	if err == nil {
		status.OnOff = &onOffState
	}

	return status, nil
}

// SensorData contains temperature and humidity sensor readings.
type SensorData struct {
	Temperature *float32 // Celsius, nil if not available
	Humidity    *float32 // Percent, nil if not available
	Battery     *uint8   // Percent, nil if not available
}

// SensorReport contains a sensor report with device address.
type SensorReport struct {
	NwkAddr     uint16
	ClusterID   uint16 // For debugging
	Endpoint    uint8  // For debugging
	Temperature *float32
	Humidity    *float32
	Battery     *uint8
	// Power consumption data from smart plugs
	Voltage *float64 // RMS Voltage in volts
	Current *float64 // RMS Current in amps
	Power   *float64 // Active power in watts
	Energy  *float64 // Total energy in kWh
	// On/Off state
	OnOff *bool
	// Debug info
	FrameCommandID uint8
	ParseError     string
	Attributes     []ReportedAttribute
}

// ReportedAttribute contains raw attribute data for debugging.
type ReportedAttribute struct {
	ID    uint16
	Value interface{}
}

// ReadSensorData reads temperature, humidity, and battery from a sensor.
func (a *Adapter) ReadSensorData(ctx context.Context, nwkAddr uint16, endpoint uint8) (*SensorData, error) {
	data := &SensorData{}

	// Read temperature (cluster 0x0402, attr 0x0000)
	// Value is int16 in units of 0.01°C
	tempResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTempMeasurement, zcl.AttrTempMeasuredValue)
	if err == nil && len(tempResults) > 0 && tempResults[0].Status == zcl.StatusSuccess {
		if raw, ok := tempResults[0].Value.(int16); ok {
			temp := float32(raw) / 100.0
			data.Temperature = &temp
		}
	}

	// Read humidity (cluster 0x0405, attr 0x0000)
	// Value is uint16 in units of 0.01%
	humResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterHumidityMeas, zcl.AttrHumidityMeasuredValue)
	if err == nil && len(humResults) > 0 && humResults[0].Status == zcl.StatusSuccess {
		if raw, ok := humResults[0].Value.(uint16); ok {
			hum := float32(raw) / 100.0
			data.Humidity = &hum
		}
	}

	// Read battery percentage (cluster 0x0001, attr 0x0021)
	// Value is uint8, 0-200 representing 0-100% (0.5% steps)
	battResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, zcl.AttrPowerBatteryPercentage)
	if err == nil && len(battResults) > 0 && battResults[0].Status == zcl.StatusSuccess {
		if raw, ok := battResults[0].Value.(uint8); ok {
			batt := raw / 2 // Convert from 0-200 to 0-100%
			data.Battery = &batt
		}
	}

	return data, nil
}

// PressureData contains atmospheric pressure readings.
type PressureData struct {
	Pressure    *float64 // Pressure in hPa (hectopascals)
	MinPressure *float64 // Minimum measurable pressure in hPa
	MaxPressure *float64 // Maximum measurable pressure in hPa
}

// GetPressure reads atmospheric pressure from a device.
// Returns pressure in hPa (hectopascals), also known as millibars.
// The device reports pressure in units of 10 Pa (0.1 hPa), which is converted to hPa.
// Some devices support scaled values with a scale attribute (10^scale multiplier).
func (a *Adapter) GetPressure(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PressureData, error) {
	data := &PressureData{}

	// Try to read regular pressure attributes first
	regularAttrs := []zcl.AttributeID{
		zcl.AttrPressureMeasuredValue,
		zcl.AttrPressureMinMeasuredValue,
		zcl.AttrPressureMaxMeasuredValue,
	}

	regularResults, regularErr := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPressureMeas, regularAttrs...)

	// Try to read scaled pressure attributes and scale
	scaledAttrs := []zcl.AttributeID{
		zcl.AttrPressureScaledValue,
		zcl.AttrPressureMinScaledValue,
		zcl.AttrPressureMaxScaledValue,
		zcl.AttrPressureScale,
	}

	scaledResults, scaledErr := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPressureMeas, scaledAttrs...)

	// If both failed, return error
	if regularErr != nil && scaledErr != nil {
		return nil, fmt.Errorf("failed to read pressure attributes: %w", regularErr)
	}

	// Check if we have a scale attribute for scaled values
	var scale int8 = 0
	hasScale := false
	if scaledErr == nil {
		for _, r := range scaledResults {
			if r.AttributeID == zcl.AttrPressureScale && r.Status == zcl.StatusSuccess {
				if s, ok := toInt8(r.Value); ok {
					scale = s
					hasScale = true
				}
			}
		}
	}

	// Calculate scale multiplier: 10^scale
	// For example: scale=-1 means multiply by 0.1, scale=0 means multiply by 1
	scaleMultiplier := 1.0
	if hasScale {
		for i := int8(0); i < scale; i++ {
			scaleMultiplier *= 10.0
		}
		for i := int8(0); i > scale; i-- {
			scaleMultiplier /= 10.0
		}
	}

	// Try scaled values first if scale is available and we got successful reads
	if hasScale && scaledErr == nil {
		for _, r := range scaledResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}

			switch r.AttributeID {
			case zcl.AttrPressureScaledValue:
				if raw, ok := toInt16(r.Value); ok {
					// Apply scale multiplier to get hPa
					pressure := float64(raw) * scaleMultiplier
					data.Pressure = &pressure
				}
			case zcl.AttrPressureMinScaledValue:
				if raw, ok := toInt16(r.Value); ok {
					minPressure := float64(raw) * scaleMultiplier
					data.MinPressure = &minPressure
				}
			case zcl.AttrPressureMaxScaledValue:
				if raw, ok := toInt16(r.Value); ok {
					maxPressure := float64(raw) * scaleMultiplier
					data.MaxPressure = &maxPressure
				}
			}
		}
	}

	// If scaled values didn't work or aren't available, use regular values
	// Regular values are in units of 10 Pa (0.1 hPa), so divide by 10 to get hPa
	if data.Pressure == nil && regularErr == nil {
		for _, r := range regularResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}

			switch r.AttributeID {
			case zcl.AttrPressureMeasuredValue:
				if raw, ok := toInt16(r.Value); ok {
					// Convert from 10 Pa (0.1 hPa) to hPa
					pressure := float64(raw) / 10.0
					data.Pressure = &pressure
				}
			case zcl.AttrPressureMinMeasuredValue:
				if raw, ok := toInt16(r.Value); ok {
					minPressure := float64(raw) / 10.0
					data.MinPressure = &minPressure
				}
			case zcl.AttrPressureMaxMeasuredValue:
				if raw, ok := toInt16(r.Value); ok {
					maxPressure := float64(raw) / 10.0
					data.MaxPressure = &maxPressure
				}
			}
		}
	}

	return data, nil
}

// WaitForSensorReport waits for an incoming sensor report from any device.
// Returns when a temperature, humidity, or battery report is received.
// Duplicate messages (same source, cluster, and sequence number within
// the deduplication window) are automatically filtered out.
func (a *Adapter) WaitForSensorReport(ctx context.Context, timeout time.Duration) (*SensorReport, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	dedupe := a.dedupe
	a.mu.Unlock()

	// Track deadline for proper timeout handling across retries
	deadline := time.Now().Add(timeout)

	for {
		// Check if we've exceeded the overall timeout
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("wait for sensor report: timeout")
		}

		// Wait for any incoming message (srcAddr=0, clusterID=0 matches all)
		msg, err := znpClient.WaitForIncomingMsg(ctx, 0, 0, remaining)
		if err != nil {
			return nil, err
		}

		report := &SensorReport{
			NwkAddr: msg.SrcAddr,
		}

		// Parse ZCL frame
		frame, err := zcl.ParseFrame(msg.Data)
		if err != nil {
			// Store error for debugging
			report.ParseError = fmt.Sprintf("parsing ZCL frame: %v", err)
			return report, nil
		}

		// Store frame info for debugging
		report.FrameCommandID = frame.CommandID

		// Check for duplicates using (srcAddr, clusterID, transSeqNum)
		// This filters out mesh network retransmissions
		if dedupe != nil && dedupe.isDuplicate(msg.SrcAddr, msg.ClusterID, frame.TransSeqNum) {
			// Duplicate message, skip and wait for next
			continue
		}

		// Only handle Report Attributes command
		if frame.CommandID != uint8(zcl.CmdReportAttributes) {
			return report, nil
		}

		results, err := zcl.ParseReportAttributesPayload(frame.Payload)
		if err != nil {
			report.ParseError = fmt.Sprintf("parsing attributes: %v", err)
			return report, nil
		}

		// Store cluster and endpoint for debugging
		report.ClusterID = uint16(msg.ClusterID)
		report.Endpoint = msg.SrcEndpoint

		// Store raw attributes for debugging
		for _, r := range results {
			report.Attributes = append(report.Attributes, ReportedAttribute{
				ID:    uint16(r.AttributeID),
				Value: r.Value,
			})
		}

		// Check cluster ID and parse accordingly
		switch zcl.ClusterID(msg.ClusterID) {
		case zcl.ClusterOnOff:
			for _, r := range results {
				if r.AttributeID == zcl.AttrOnOff {
					var state bool
					switch v := r.Value.(type) {
					case bool:
						state = v
					case uint8:
						state = v != 0
					}
					report.OnOff = &state
				}
			}

		case zcl.ClusterTempMeasurement:
			for _, r := range results {
				if r.AttributeID == zcl.AttrTempMeasuredValue {
					if raw, ok := r.Value.(int16); ok {
						temp := float32(raw) / 100.0
						report.Temperature = &temp
					}
				}
			}

		case zcl.ClusterHumidityMeas:
			for _, r := range results {
				if r.AttributeID == zcl.AttrHumidityMeasuredValue {
					if raw, ok := r.Value.(uint16); ok {
						hum := float32(raw) / 100.0
						report.Humidity = &hum
					}
				}
			}

		case zcl.ClusterPowerConfig:
			for _, r := range results {
				if r.AttributeID == zcl.AttrPowerBatteryPercentage {
					if raw, ok := r.Value.(uint8); ok {
						batt := raw / 2
						report.Battery = &batt
					}
				}
			}

		case zcl.ClusterElectricalMeas:
			// Parse power consumption attributes
			for _, r := range results {
				switch r.AttributeID {
				case zcl.AttrElecRMSVoltage:
					if raw, ok := toUint16(r.Value); ok {
						v := float64(raw) / 1.0 // Tuya reports actual voltage
						report.Voltage = &v
					}
				case zcl.AttrElecRMSCurrent:
					if raw, ok := toUint16(r.Value); ok {
						c := float64(raw) / 1000.0 // Tuya reports current in mA
						report.Current = &c
					}
				case zcl.AttrElecActivePower:
					if raw, ok := toInt16(r.Value); ok {
						p := float64(raw) / 1.0 // Tuya reports actual power
						report.Power = &p
					}
				}
			}

		case zcl.ClusterMeteringSimple:
			// Parse energy metering attributes
			for _, r := range results {
				if r.AttributeID == zcl.AttrMeterCurrentSummation {
					if raw, ok := toUint48(r.Value); ok {
						// Tuya reports energy in Wh, convert to kWh
						e := float64(raw) / 1000.0
						report.Energy = &e
					}
				}
			}
		}

		return report, nil
	}
}

// PowerData contains power consumption readings from a smart plug.
type PowerData struct {
	// Electrical Measurement cluster (0x0B04) - instantaneous values
	Voltage     *float64 // RMS Voltage in volts
	Current     *float64 // RMS Current in amps
	ActivePower *float64 // Active power in watts
	PowerFactor *int8    // Power factor (-100 to 100%)

	// Simple Metering cluster (0x0702) - cumulative values
	TotalEnergy  *float64 // Total energy in kWh
	InstantPower *float64 // Instantaneous demand in watts (from metering)

	// Debug info
	RawVoltage uint16
	RawCurrent uint16
	RawPower   int16
	RawEnergy  uint64
}

// ReadPowerData reads power consumption data from a smart plug.
func (a *Adapter) ReadPowerData(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PowerData, error) {
	data := &PowerData{}

	// Try to read from Electrical Measurement cluster (0x0B04)
	elecAttrs := []zcl.AttributeID{
		zcl.AttrElecRMSVoltage,
		zcl.AttrElecRMSCurrent,
		zcl.AttrElecActivePower,
		zcl.AttrElecPowerFactor,
		zcl.AttrElecACVoltageDivisor,
		zcl.AttrElecACCurrentDivisor,
		zcl.AttrElecACPowerDivisor,
	}

	elecResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterElectricalMeas, elecAttrs...)
	if err == nil {
		// Parse divisors first (defaults for when device doesn't report them)
		// Tuya TS011F typically reports:
		// - Voltage: actual value (divisor 1)
		// - Current: in mA (divisor 1000)
		// - Power: actual value (divisor 1)
		voltageDivisor := uint16(1)
		currentDivisor := uint16(1000) // Most Tuya plugs report current in mA
		powerDivisor := uint16(1)

		for _, r := range elecResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrElecACVoltageDivisor:
				if v, ok := toUint16(r.Value); ok && v > 0 {
					voltageDivisor = v
				}
			case zcl.AttrElecACCurrentDivisor:
				if v, ok := toUint16(r.Value); ok && v > 0 {
					currentDivisor = v
				}
			case zcl.AttrElecACPowerDivisor:
				if v, ok := toUint16(r.Value); ok && v > 0 {
					powerDivisor = v
				}
			}
		}

		// Now parse measurement values
		for _, r := range elecResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrElecRMSVoltage:
				if v, ok := toUint16(r.Value); ok {
					data.RawVoltage = v
					voltage := float64(v) / float64(voltageDivisor)
					data.Voltage = &voltage
				}
			case zcl.AttrElecRMSCurrent:
				if v, ok := toUint16(r.Value); ok {
					data.RawCurrent = v
					current := float64(v) / float64(currentDivisor)
					data.Current = &current
				}
			case zcl.AttrElecActivePower:
				if v, ok := toInt16(r.Value); ok {
					data.RawPower = v
					power := float64(v) / float64(powerDivisor)
					data.ActivePower = &power
				}
			case zcl.AttrElecPowerFactor:
				if v, ok := toInt8(r.Value); ok {
					data.PowerFactor = &v
				}
			}
		}
	}

	// Try to read from Simple Metering cluster (0x0702)
	meterAttrs := []zcl.AttributeID{
		zcl.AttrMeterCurrentSummation,
		zcl.AttrMeterInstantDemand,
		zcl.AttrMeterMultiplier,
		zcl.AttrMeterDivisor,
	}

	meterResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMeteringSimple, meterAttrs...)
	if err == nil {
		// Parse multiplier/divisor first (defaults)
		multiplier := uint32(1)
		divisor := uint32(1)

		for _, r := range meterResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrMeterMultiplier:
				if v, ok := toUint32(r.Value); ok && v > 0 {
					multiplier = v
				}
			case zcl.AttrMeterDivisor:
				if v, ok := toUint32(r.Value); ok && v > 0 {
					divisor = v
				}
			}
		}

		// Now parse measurement values
		for _, r := range meterResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}
			switch r.AttributeID {
			case zcl.AttrMeterCurrentSummation:
				if v, ok := toUint48(r.Value); ok {
					data.RawEnergy = v
					// Apply multiplier/divisor to get kWh
					energy := float64(v) * float64(multiplier) / float64(divisor)
					data.TotalEnergy = &energy
				}
			case zcl.AttrMeterInstantDemand:
				if v, ok := toInt32(r.Value); ok {
					// Apply multiplier/divisor to get watts
					power := float64(v) * float64(multiplier) / float64(divisor) * 1000 // Convert kW to W
					data.InstantPower = &power
				}
			}
		}
	}

	return data, nil
}

// Helper functions to convert interface{} to specific types
func toUint16(v interface{}) (uint16, bool) {
	switch val := v.(type) {
	case uint16:
		return val, true
	case int16:
		return uint16(val), true
	case uint8:
		return uint16(val), true
	case int:
		return uint16(val), true
	}
	return 0, false
}

func toInt16(v interface{}) (int16, bool) {
	switch val := v.(type) {
	case int16:
		return val, true
	case uint16:
		return int16(val), true
	case int:
		return int16(val), true
	}
	return 0, false
}

func toInt8(v interface{}) (int8, bool) {
	switch val := v.(type) {
	case int8:
		return val, true
	case uint8:
		return int8(val), true
	case int:
		return int8(val), true
	}
	return 0, false
}

func toUint32(v interface{}) (uint32, bool) {
	switch val := v.(type) {
	case uint32:
		return val, true
	case int32:
		return uint32(val), true
	case uint16:
		return uint32(val), true
	case int:
		return uint32(val), true
	}
	return 0, false
}

func toInt32(v interface{}) (int32, bool) {
	switch val := v.(type) {
	case int32:
		return val, true
	case uint32:
		return int32(val), true
	case int:
		return int32(val), true
	}
	return 0, false
}

func toUint48(v interface{}) (uint64, bool) {
	switch val := v.(type) {
	case uint64:
		return val, true
	case int64:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case int:
		return uint64(val), true
	}
	return 0, false
}

// ============================================================================
// PowerConfiguration Cluster (0x0001)
// ============================================================================

// BatteryInfo contains battery status information.
type BatteryInfo struct {
	Voltage    *float64 // Voltage in V (nil if not available)
	Percentage *uint8   // Percentage remaining 0-100% (nil if not available)
	LowBattery bool     // True if battery alarm is active
}

// GetBatteryInfo reads battery status from a device.
// Returns battery voltage, percentage, and alarm status from the PowerConfiguration cluster.
// Voltage is converted from device units (0.1V) to volts.
// Percentage is converted from device range (0-200 = 0-100%) to 0-100%.
func (a *Adapter) GetBatteryInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*BatteryInfo, error) {
	info := &BatteryInfo{}

	// Read battery attributes
	attrs := []zcl.AttributeID{
		zcl.AttrPowerBatteryVoltage,
		zcl.AttrPowerBatteryPercentage,
		zcl.AttrPowerBatteryAlarmState,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read battery attributes: %w", err)
	}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrPowerBatteryVoltage:
			if raw, ok := r.Value.(uint8); ok {
				// Convert from 0.1V units to volts
				voltage := float64(raw) / 10.0
				info.Voltage = &voltage
			}
		case zcl.AttrPowerBatteryPercentage:
			if raw, ok := r.Value.(uint8); ok {
				// Convert from 0-200 (0.5% steps) to 0-100%
				percentage := raw / 2
				info.Percentage = &percentage
			}
		case zcl.AttrPowerBatteryAlarmState:
			// Check if any alarm bit is set
			if alarmState, ok := toUint32(r.Value); ok {
				info.LowBattery = alarmState != 0
			}
		}
	}

	return info, nil
}

// MainsInfo contains mains power information.
type MainsInfo struct {
	Voltage   *float64 // Voltage in V (nil if not available)
	Frequency *uint8   // Frequency in Hz (nil if not available)
}

// GetMainsInfo reads mains power information from a device.
// Returns mains voltage and frequency from the PowerConfiguration cluster.
// Voltage is converted from device units (0.1V) to volts.
// Frequency is converted from device units (2Hz increments) to Hz.
func (a *Adapter) GetMainsInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*MainsInfo, error) {
	info := &MainsInfo{}

	// Read mains attributes
	attrs := []zcl.AttributeID{
		zcl.AttrPowerMainsVoltage,
		zcl.AttrPowerMainsFrequency,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read mains attributes: %w", err)
	}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrPowerMainsVoltage:
			if raw, ok := toUint16(r.Value); ok {
				// Convert from 0.1V units to volts
				voltage := float64(raw) / 10.0
				info.Voltage = &voltage
			}
		case zcl.AttrPowerMainsFrequency:
			if raw, ok := r.Value.(uint8); ok {
				// Convert from 2Hz increments to Hz
				frequency := raw * 2
				info.Frequency = &frequency
			}
		}
	}

	return info, nil
}

// ============================================================================
// BinaryInput Cluster (0x000F)
// ============================================================================

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

// ============================================================================
// Energy Management
// ============================================================================

// ResetEnergy attempts to reset the energy counter on a smart plug.
// This uses manufacturer-specific methods and may not work on all devices.
// For Tuya TS011F plugs, zigbee2mqtt uses Basic cluster resetFactDefault command.
func (a *Adapter) ResetEnergy(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	var lastErr error

	// Method 1: Basic cluster (0x0000) command 0 (resetFactDefault)
	// This is what zigbee2mqtt uses for TS011F energy reset
	if err := a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterBasic, 0x00, nil); err == nil {
		return nil
	} else {
		lastErr = err
	}

	// Method 2: Tuya cluster 0xE001, attribute 0xD004, value 1
	tuyaValues1 := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTuyaEnergyReset: {
			Type:  zcl.TypeEnum8,
			Value: uint8(1),
		},
	}
	if err := a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTuyaSpecific, tuyaValues1); err == nil {
		return nil
	} else {
		lastErr = err
	}

	// Method 3: Tuya cluster 0xE000, attribute 0xD004
	tuyaValues2 := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTuyaEnergyReset: {
			Type:  zcl.TypeEnum8,
			Value: uint8(1),
		},
	}
	if err := a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterID(0xE000), tuyaValues2); err == nil {
		return nil
	} else {
		lastErr = err
	}

	// Method 4: Try writing 0 to the metering summation attribute (standard but rarely works)
	meterValues := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrMeterCurrentSummation: {
			Type:  zcl.TypeUint48,
			Value: uint64(0),
		},
	}
	if err := a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterMeteringSimple, meterValues); err == nil {
		return nil
	} else {
		lastErr = err
	}

	return fmt.Errorf("energy reset failed (tried 4 methods): %w", lastErr)
}

// sendZclRequest sends a ZCL request and waits for response.
func (a *Adapter) sendZclRequest(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID uint16, frame *zcl.Frame) (*zcl.Frame, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Build AF data request
	req := znp.DataRequest{
		DstAddr:     nwkAddr,
		DstEndpoint: endpoint,
		SrcEndpoint: CoordinatorEndpoint,
		ClusterID:   clusterID,
		TransID:     frame.TransSeqNum,
		Options:     znp.AfOptionAckRequest,
		Radius:      30,
		Data:        frame.ToBytes(),
	}

	// Send request
	status, err := znpClient.AfDataRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("data request failed: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("data request returned status 0x%02X", status)
	}

	// Wait for data confirm
	confirm, err := znpClient.WaitForDataConfirm(ctx, frame.TransSeqNum, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("waiting for data confirm: %w", err)
	}

	if confirm.Status != 0 {
		return nil, fmt.Errorf("data confirm returned status 0x%02X", confirm.Status)
	}

	// Wait for incoming response
	msg, err := znpClient.WaitForIncomingMsg(ctx, nwkAddr, clusterID, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("waiting for response: %w", err)
	}

	// Parse ZCL frame
	respFrame, err := zcl.ParseFrame(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ZCL frame: %w", err)
	}

	return respFrame, nil
}

// ============================================================================
// Alarms Cluster (0x0009)
// ============================================================================

// AlarmEntry represents an entry in the device's alarm log.
// Alarms can be generated by various conditions on the device (e.g., low battery, temperature threshold).
type AlarmEntry struct {
	AlarmCode uint8         // Alarm code identifying the type of alarm
	ClusterID zcl.ClusterID // Cluster that generated the alarm
	Timestamp uint32        // Time when alarm occurred (Zigbee time, seconds since 2000-01-01)
}

// GetAlarmCount reads the number of alarm entries currently stored in the device's alarm log.
// Returns the count of active alarms.
func (a *Adapter) GetAlarmCount(ctx context.Context, nwkAddr uint16, endpoint uint8) (uint16, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterAlarms, zcl.AttrAlarmCount)
	if err != nil {
		return 0, fmt.Errorf("failed to read alarm count: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read alarm count returned status 0x%02X", result.Status)
	}

	if count, ok := toUint16(result.Value); ok {
		return count, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// ResetAlarm clears a specific alarm from the device's alarm log.
// The alarm is identified by its alarm code and the cluster that generated it.
//
// Parameters:
//   - alarmCode: The alarm code to reset (device-specific)
//   - clusterID: The cluster ID that generated the alarm
//
// Example: Reset a low battery alarm (code 0x00) from PowerConfiguration cluster:
//
//	err := adapter.ResetAlarm(ctx, nwkAddr, endpoint, 0x00, zcl.ClusterPowerConfig)
func (a *Adapter) ResetAlarm(ctx context.Context, nwkAddr uint16, endpoint uint8, alarmCode uint8, clusterID zcl.ClusterID) error {
	// ResetAlarm payload: alarmCode (1 byte) + clusterID (2 bytes LE)
	payload := []byte{
		alarmCode,
		byte(clusterID & 0xFF),
		byte(clusterID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterAlarms, zcl.CmdAlarmsResetAlarm, payload)
}

// ResetAllAlarms clears all alarms from the device's alarm log.
// This removes all active alarm entries.
func (a *Adapter) ResetAllAlarms(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// ResetAllAlarms command has no payload
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterAlarms, zcl.CmdAlarmsResetAllAlarms, nil)
}

// GetAlarm retrieves the oldest alarm entry from the device's alarm log.
// The device returns the first alarm in its log (FIFO order).
// Returns nil if there are no alarms in the log.
//
// The alarm is not removed from the log by this command; use ResetAlarm or ResetAllAlarms to clear it.
func (a *Adapter) GetAlarm(ctx context.Context, nwkAddr uint16, endpoint uint8) (*AlarmEntry, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build and send GetAlarm request (no payload)
	seqNum := nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, zcl.CmdAlarmsGetAlarm, nil)

	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(zcl.ClusterAlarms), frame)
	if err != nil {
		return nil, fmt.Errorf("get alarm failed: %w", err)
	}

	// Parse GetAlarmResponse (command 0x01)
	if respFrame.CommandID != zcl.CmdAlarmsGetAlarmResponse {
		return nil, fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	// GetAlarmResponse payload:
	// - status (1 byte): 0x00 = success, others = error/no alarms
	// - alarmCode (1 byte) - only if status is success
	// - clusterID (2 bytes LE) - only if status is success
	// - timestamp (4 bytes LE) - only if status is success
	if len(respFrame.Payload) < 1 {
		return nil, fmt.Errorf("response too short")
	}

	status := respFrame.Payload[0]
	if status != zcl.StatusSuccess {
		// No alarms in log or error
		if status == 0x01 { // NOT_FOUND
			return nil, nil
		}
		return nil, fmt.Errorf("get alarm returned status 0x%02X", status)
	}

	// Parse successful response
	if len(respFrame.Payload) < 8 {
		return nil, fmt.Errorf("response payload too short for alarm entry")
	}

	entry := &AlarmEntry{
		AlarmCode: respFrame.Payload[1],
		ClusterID: zcl.ClusterID(uint16(respFrame.Payload[2]) | uint16(respFrame.Payload[3])<<8),
		Timestamp: uint32(respFrame.Payload[4]) | uint32(respFrame.Payload[5])<<8 |
			uint32(respFrame.Payload[6])<<16 | uint32(respFrame.Payload[7])<<24,
	}

	return entry, nil
}

// ResetAlarmLog clears the entire alarm log on the device.
// This removes all alarm history but does not prevent new alarms from being generated.
func (a *Adapter) ResetAlarmLog(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// ResetAlarmLog command has no payload
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterAlarms, zcl.CmdAlarmsResetAlarmLog, nil)
}

// ============================================================================
// Air Quality Measurement Clusters
// ============================================================================

// AirQualityData contains air quality sensor readings.
type AirQualityData struct {
	CO           *float32 // Carbon monoxide in ppm
	CO2          *float32 // Carbon dioxide in ppm
	PM25         *float32 // PM2.5 in µg/m³
	Formaldehyde *float32 // Formaldehyde in ppm
}

// GetAirQuality reads air quality measurements from a device.
// It will read from whichever clusters the device supports.
// Values are nil if the device doesn't support that measurement type.
func (a *Adapter) GetAirQuality(ctx context.Context, nwkAddr uint16, endpoint uint8) (*AirQualityData, error) {
	data := &AirQualityData{}

	// Try to read CO (cluster 0x040C, attr 0x0000)
	// Value is float32 (single) in ppm
	coResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterCarbonMonoxide, zcl.AttrCOMeasuredValue)
	if err == nil && len(coResults) > 0 && coResults[0].Status == zcl.StatusSuccess {
		if val, ok := coResults[0].Value.(float32); ok {
			data.CO = &val
		}
	}

	// Try to read CO2 (cluster 0x040D, attr 0x0000)
	// Value is float32 (single) in ppm
	co2Results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterCarbonDioxide, zcl.AttrCO2MeasuredValue)
	if err == nil && len(co2Results) > 0 && co2Results[0].Status == zcl.StatusSuccess {
		if val, ok := co2Results[0].Value.(float32); ok {
			data.CO2 = &val
		}
	}

	// Try to read PM2.5 (cluster 0x042A, attr 0x0000)
	// Value is float32 (single) in µg/m³
	pm25Results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPM25Measurement, zcl.AttrPM25MeasuredValue)
	if err == nil && len(pm25Results) > 0 && pm25Results[0].Status == zcl.StatusSuccess {
		if val, ok := pm25Results[0].Value.(float32); ok {
			data.PM25 = &val
		}
	}

	// Try to read Formaldehyde (cluster 0x042B, attr 0x0000)
	// Value is float32 (single) in ppm
	formaldehydeResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFormaldehydeMeasurement, zcl.AttrFormaldehydeMeasuredValue)
	if err == nil && len(formaldehydeResults) > 0 && formaldehydeResults[0].Status == zcl.StatusSuccess {
		if val, ok := formaldehydeResults[0].Value.(float32); ok {
			data.Formaldehyde = &val
		}
	}

	return data, nil
}

// GetCO2Level reads carbon dioxide concentration.
// Returns the CO2 level in ppm, or nil if not available.
func (a *Adapter) GetCO2Level(ctx context.Context, nwkAddr uint16, endpoint uint8) (*float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterCarbonDioxide, zcl.AttrCO2MeasuredValue)
	if err != nil {
		return nil, fmt.Errorf("failed to read CO2 level: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return nil, fmt.Errorf("read CO2 level returned status 0x%02X", result.Status)
	}

	if val, ok := result.Value.(float32); ok {
		return &val, nil
	}
	return nil, fmt.Errorf("unexpected value type: %T", result.Value)
}

// GetPM25Level reads PM2.5 particulate concentration.
// Returns the PM2.5 level in µg/m³, or nil if not available.
func (a *Adapter) GetPM25Level(ctx context.Context, nwkAddr uint16, endpoint uint8) (*float32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPM25Measurement, zcl.AttrPM25MeasuredValue)
	if err != nil {
		return nil, fmt.Errorf("failed to read PM2.5 level: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return nil, fmt.Errorf("read PM2.5 level returned status 0x%02X", result.Status)
	}

	if val, ok := result.Value.(float32); ok {
		return &val, nil
	}
	return nil, fmt.Errorf("unexpected value type: %T", result.Value)
}

// ============================================================================
// AnalogInput Cluster (0x000C)
// ============================================================================

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

// ============================================================================
// AnalogOutput Cluster (0x000D)
// ============================================================================

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

// ============================================================================
// AnalogValue Cluster (0x000E)
// ============================================================================

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

// ============================================================================
// BinaryValue Cluster (0x0011)
// ============================================================================

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

// ============================================================================
// MultistateInput Cluster (0x0012)
// ============================================================================

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

// ============================================================================
// IASZone Cluster (0x0500) - Security Sensors
// ============================================================================

// IASZoneStatus contains current zone sensor state.
type IASZoneStatus struct {
	ZoneState  uint8           // 0=not enrolled, 1=enrolled
	ZoneType   zcl.IASZoneType // Type of sensor
	ZoneStatus uint16          // Bitmap of current alarms and states
	ZoneID     *uint8          // Zone ID (0-254), nil if not available
	CIEAddress *[8]byte        // IEEE address of CIE, nil if not available

	// Parsed status flags for convenience
	Alarm1        bool // Alarm 1 active (e.g., motion detected, door open)
	Alarm2        bool // Alarm 2 active (secondary alarm)
	Tamper        bool // Tamper detected
	LowBattery    bool // Low battery warning
	Trouble       bool // Sensor trouble/failure
	ACMains       bool // AC mains fault
	Test          bool // Sensor in test mode
	BatteryDefect bool // Battery defect detected
}

// GetIASZoneStatus reads the current status from an IAS Zone device.
// This queries all relevant attributes and parses the zone status bitmap.
func (a *Adapter) GetIASZoneStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*IASZoneStatus, error) {
	// Read IAS Zone attributes
	attrs := []zcl.AttributeID{
		zcl.AttrIASZoneState,
		zcl.AttrIASZoneType,
		zcl.AttrIASZoneStatus,
		zcl.AttrIASZoneID,
		zcl.AttrIASZoneCIEAddr,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read IAS Zone attributes: %w", err)
	}

	status := &IASZoneStatus{}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrIASZoneState:
			if v, ok := r.Value.(uint8); ok {
				status.ZoneState = v
			}
		case zcl.AttrIASZoneType:
			if v, ok := toUint16(r.Value); ok {
				status.ZoneType = zcl.IASZoneType(v)
			}
		case zcl.AttrIASZoneStatus:
			if v, ok := toUint16(r.Value); ok {
				status.ZoneStatus = v
				// Parse status bitmap
				status.Alarm1 = (v & zcl.IASZoneStatusAlarm1) != 0
				status.Alarm2 = (v & zcl.IASZoneStatusAlarm2) != 0
				status.Tamper = (v & zcl.IASZoneStatusTamper) != 0
				status.LowBattery = (v & zcl.IASZoneStatusBattery) != 0
				status.Trouble = (v & zcl.IASZoneStatusTrouble) != 0
				status.ACMains = (v & zcl.IASZoneStatusACMains) != 0
				status.Test = (v & zcl.IASZoneStatusTest) != 0
				status.BatteryDefect = (v & zcl.IASZoneStatusBatteryDefect) != 0
			}
		case zcl.AttrIASZoneID:
			if v, ok := r.Value.(uint8); ok {
				status.ZoneID = &v
			}
		case zcl.AttrIASZoneCIEAddr:
			// IEEE address is 8 bytes
			if v, ok := r.Value.([8]byte); ok {
				status.CIEAddress = &v
			}
		}
	}

	return status, nil
}

// EnrollIASZone sends an enroll response to a zone device.
// This must be called after receiving an enroll request from the device.
//
// Parameters:
//   - responseCode: Enrollment response code
//     0x00 = Success
//     0x01 = Not supported
//     0x02 = No enroll permit
//     0x03 = Too many zones
//   - zoneID: Assigned zone ID (0-254)
//
// The device will not send zone status change notifications until it is enrolled.
func (a *Adapter) EnrollIASZone(ctx context.Context, nwkAddr uint16, endpoint uint8, responseCode uint8, zoneID uint8) error {
	// EnrollResponse payload: responseCode (1 byte) + zoneID (1 byte)
	payload := []byte{responseCode, zoneID}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, zcl.CmdIASZoneEnrollResponse, payload)
}

// WriteIASZoneCIEAddress writes the coordinator's IEEE address to the zone device.
// This is required before a device can be enrolled. The device needs to know
// which coordinator (CIE - Control and Indicating Equipment) to send notifications to.
//
// The cieAddress should be the coordinator's IEEE address, which can be obtained
// from the network configuration or device information.
func (a *Adapter) WriteIASZoneCIEAddress(ctx context.Context, nwkAddr uint16, endpoint uint8, cieAddress [8]byte) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrIASZoneCIEAddr: {
			Type:  zcl.TypeIEEEAddr,
			Value: cieAddress,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, values)
}

// IASZoneNotification represents a zone status change notification from a device.
// These are sent automatically by enrolled IAS Zone devices when their status changes
// (e.g., motion detected, door opened, tamper alert).
type IASZoneNotification struct {
	SrcAddr    uint16 // Source device network address
	Endpoint   uint8  // Source endpoint
	ZoneStatus uint16 // Status bitmap
	ExtStatus  uint8  // Extended status
	ZoneID     uint8  // Zone ID
	Delay      uint16 // Delay in quarter-seconds

	// Parsed status flags for convenience
	Alarm1        bool // Alarm 1 active
	Alarm2        bool // Alarm 2 active
	Tamper        bool // Tamper detected
	LowBattery    bool // Low battery
	Trouble       bool // Trouble/failure
	ACMains       bool // AC mains fault
	Test          bool // Test mode
	BatteryDefect bool // Battery defect
}

// WaitForIASZoneNotification waits for an incoming IAS Zone status change notification.
// Returns when a zone notification is received or timeout occurs.
//
// Note: The device must be enrolled before it will send notifications.
// Use EnrollIASZone to enroll the device first.
func (a *Adapter) WaitForIASZoneNotification(ctx context.Context, timeout time.Duration) (*IASZoneNotification, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	dedupe := a.dedupe
	a.mu.Unlock()

	deadline := time.Now().Add(timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("wait for IAS Zone notification: timeout")
		}

		// Wait for incoming message on IASZone cluster
		msg, err := znpClient.WaitForIncomingMsg(ctx, 0, uint16(zcl.ClusterIASZone), remaining)
		if err != nil {
			return nil, err
		}

		// Parse ZCL frame
		frame, err := zcl.ParseFrame(msg.Data)
		if err != nil {
			continue
		}

		// Check for duplicates
		if dedupe != nil && dedupe.isDuplicate(msg.SrcAddr, msg.ClusterID, frame.TransSeqNum) {
			continue
		}

		// Check if this is a status change notification
		if frame.CommandID != zcl.CmdIASZoneStatusChangeNotification {
			continue
		}

		// Parse notification payload
		// Payload format: zoneStatus (2 bytes LE) + extendedStatus (1 byte) + zoneID (1 byte) + delay (2 bytes LE)
		if len(frame.Payload) < 6 {
			continue
		}

		notification := &IASZoneNotification{
			SrcAddr:    msg.SrcAddr,
			Endpoint:   msg.SrcEndpoint,
			ZoneStatus: uint16(frame.Payload[0]) | uint16(frame.Payload[1])<<8,
			ExtStatus:  frame.Payload[2],
			ZoneID:     frame.Payload[3],
			Delay:      uint16(frame.Payload[4]) | uint16(frame.Payload[5])<<8,
		}

		// Parse status flags
		notification.Alarm1 = (notification.ZoneStatus & zcl.IASZoneStatusAlarm1) != 0
		notification.Alarm2 = (notification.ZoneStatus & zcl.IASZoneStatusAlarm2) != 0
		notification.Tamper = (notification.ZoneStatus & zcl.IASZoneStatusTamper) != 0
		notification.LowBattery = (notification.ZoneStatus & zcl.IASZoneStatusBattery) != 0
		notification.Trouble = (notification.ZoneStatus & zcl.IASZoneStatusTrouble) != 0
		notification.ACMains = (notification.ZoneStatus & zcl.IASZoneStatusACMains) != 0
		notification.Test = (notification.ZoneStatus & zcl.IASZoneStatusTest) != 0
		notification.BatteryDefect = (notification.ZoneStatus & zcl.IASZoneStatusBatteryDefect) != 0

		return notification, nil
	}
}

// ============================================================================
// DoorLock Cluster (0x0101)
// ============================================================================

// DoorLockStatus contains current lock state.
type DoorLockStatus struct {
	LockState       zcl.DoorLockState
	DoorState       *zcl.DoorState
	ActuatorEnabled bool
	AutoRelockTime  *uint32
}

// GetDoorLockStatus reads the current door lock status from a device.
// Returns the lock state, door state (if available), actuator status,
// and auto-relock time (if configured).
func (a *Adapter) GetDoorLockStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DoorLockStatus, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrDoorLockState,
		zcl.AttrDoorLockDoorState,
		zcl.AttrDoorLockActuatorEnabled,
		zcl.AttrDoorLockAutoRelockTime,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read door lock attributes: %w", err)
	}

	status := &DoorLockStatus{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrDoorLockState:
			if v, ok := r.Value.(uint8); ok {
				status.LockState = zcl.DoorLockState(v)
			}
		case zcl.AttrDoorLockDoorState:
			if v, ok := r.Value.(uint8); ok {
				doorState := zcl.DoorState(v)
				status.DoorState = &doorState
			}
		case zcl.AttrDoorLockActuatorEnabled:
			if v, ok := r.Value.(bool); ok {
				status.ActuatorEnabled = v
			} else if v, ok := r.Value.(uint8); ok {
				status.ActuatorEnabled = v != 0
			}
		case zcl.AttrDoorLockAutoRelockTime:
			if v, ok := toUint32(r.Value); ok {
				status.AutoRelockTime = &v
			}
		}
	}

	return status, nil
}

// LockDoor sends a lock command to the door lock device.
func (a *Adapter) LockDoor(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockLock, nil)
}

// UnlockDoor sends an unlock command to the door lock device.
func (a *Adapter) UnlockDoor(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockUnlock, nil)
}

// UnlockDoorWithTimeout unlocks the door and automatically relocks after the specified timeout.
// timeoutSeconds is the number of seconds before the door automatically relocks.
func (a *Adapter) UnlockDoorWithTimeout(ctx context.Context, nwkAddr uint16, endpoint uint8, timeoutSeconds uint16) error {
	// UnlockWithTimeout payload: timeout (2 bytes LE)
	payload := []byte{
		byte(timeoutSeconds & 0xFF),
		byte(timeoutSeconds >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockUnlockWithTimeout, payload)
}

// ToggleDoorLock toggles the lock state (locked <-> unlocked).
func (a *Adapter) ToggleDoorLock(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockToggle, nil)
}

// SetDoorLockPIN sets a PIN code for a user.
// userID is the user identifier (0-65535).
// pin is the PIN code string.
// The userStatus and userType are set to default values (enabled user with unrestricted access).
func (a *Adapter) SetDoorLockPIN(ctx context.Context, nwkAddr uint16, endpoint uint8, userID uint16, pin string) error {
	// SetPINCode payload: userID (2 bytes LE) + userStatus (1 byte) + userType (1 byte) + pin (string with length prefix)
	pinBytes := []byte(pin)
	payload := make([]byte, 5+len(pinBytes))
	payload[0] = byte(userID & 0xFF)
	payload[1] = byte(userID >> 8)
	payload[2] = 0x01 // userStatus: enabled (0x01)
	payload[3] = 0x00 // userType: unrestricted (0x00)
	payload[4] = byte(len(pinBytes))
	copy(payload[5:], pinBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockSetPINCode, payload)
}

// ClearDoorLockPIN removes a user's PIN code.
// userID is the user identifier to clear.
func (a *Adapter) ClearDoorLockPIN(ctx context.Context, nwkAddr uint16, endpoint uint8, userID uint16) error {
	// ClearPINCode payload: userID (2 bytes LE)
	payload := []byte{
		byte(userID & 0xFF),
		byte(userID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockClearPINCode, payload)
}

// ============================================================================
// Thermostat Cluster (0x0201)
// ============================================================================

// ThermostatStatus contains current thermostat state.
type ThermostatStatus struct {
	LocalTemperature *float64                  // Current temperature in °C
	CoolingSetpoint  *float64                  // Cooling setpoint in °C
	HeatingSetpoint  *float64                  // Heating setpoint in °C
	SystemMode       *zcl.ThermostatSystemMode // Current operating mode
	RunningState     *uint16                   // Bitmap of running states
	CoolingDemand    *uint8                    // 0-100%
	HeatingDemand    *uint8                    // 0-100%
}

// GetThermostatStatus reads the current thermostat status from a device.
// Returns the current temperature, setpoints, mode, and demand values.
// Temperature values are converted from centidegrees (0.01°C) to degrees Celsius.
func (a *Adapter) GetThermostatStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*ThermostatStatus, error) {
	// Define attributes to read
	attrs := []zcl.AttributeID{
		zcl.AttrThermostatLocalTemp,
		zcl.AttrThermostatOccupiedCoolingSetpoint,
		zcl.AttrThermostatOccupiedHeatingSetpoint,
		zcl.AttrThermostatSystemMode,
		zcl.AttrThermostatRunningState,
		zcl.AttrThermostatPICoolingDemand,
		zcl.AttrThermostatPIHeatingDemand,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read thermostat attributes: %w", err)
	}

	status := &ThermostatStatus{}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrThermostatLocalTemp:
			if raw, ok := toInt16(r.Value); ok {
				// Convert from centidegrees (0.01°C) to degrees
				temp := float64(raw) / 100.0
				status.LocalTemperature = &temp
			}
		case zcl.AttrThermostatOccupiedCoolingSetpoint:
			if raw, ok := toInt16(r.Value); ok {
				// Convert from centidegrees (0.01°C) to degrees
				setpoint := float64(raw) / 100.0
				status.CoolingSetpoint = &setpoint
			}
		case zcl.AttrThermostatOccupiedHeatingSetpoint:
			if raw, ok := toInt16(r.Value); ok {
				// Convert from centidegrees (0.01°C) to degrees
				setpoint := float64(raw) / 100.0
				status.HeatingSetpoint = &setpoint
			}
		case zcl.AttrThermostatSystemMode:
			if mode, ok := r.Value.(uint8); ok {
				m := zcl.ThermostatSystemMode(mode)
				status.SystemMode = &m
			}
		case zcl.AttrThermostatRunningState:
			if state, ok := toUint16(r.Value); ok {
				status.RunningState = &state
			}
		case zcl.AttrThermostatPICoolingDemand:
			if demand, ok := r.Value.(uint8); ok {
				status.CoolingDemand = &demand
			}
		case zcl.AttrThermostatPIHeatingDemand:
			if demand, ok := r.Value.(uint8); ok {
				status.HeatingDemand = &demand
			}
		}
	}

	return status, nil
}

// SetThermostatMode sets the operating mode of a thermostat.
// mode specifies the desired operating mode (Off, Auto, Cool, Heat, etc.).
func (a *Adapter) SetThermostatMode(ctx context.Context, nwkAddr uint16, endpoint uint8, mode zcl.ThermostatSystemMode) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrThermostatSystemMode: {
			Type:  zcl.TypeEnum8,
			Value: uint8(mode),
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, values)
}

// SetThermostatSetpoint sets the heating and cooling setpoints on a thermostat.
// Temperatures are in degrees Celsius and are converted to centidegrees (0.01°C) for the device.
// heating: Desired heating setpoint in °C
// cooling: Desired cooling setpoint in °C
func (a *Adapter) SetThermostatSetpoint(ctx context.Context, nwkAddr uint16, endpoint uint8, heating, cooling float64) error {
	// Convert from degrees to centidegrees (0.01°C)
	heatingCentidegrees := int16(heating * 100)
	coolingCentidegrees := int16(cooling * 100)

	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrThermostatOccupiedHeatingSetpoint: {
			Type:  zcl.TypeInt16,
			Value: heatingCentidegrees,
		},
		zcl.AttrThermostatOccupiedCoolingSetpoint: {
			Type:  zcl.TypeInt16,
			Value: coolingCentidegrees,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, values)
}

// AdjustThermostatSetpoint raises or lowers the thermostat setpoint.
// This uses the SetpointRaiseLower command to adjust the setpoint by a relative amount.
//
// Parameters:
//   - mode: 0=Heat, 1=Cool, 2=Both
//   - amount: Adjustment amount in 0.1°C increments (positive to raise, negative to lower)
//
// Example: Raise heating setpoint by 1°C:
//
//	err := adapter.AdjustThermostatSetpoint(ctx, nwkAddr, endpoint, 0, 10)
func (a *Adapter) AdjustThermostatSetpoint(ctx context.Context, nwkAddr uint16, endpoint uint8, mode uint8, amount int8) error {
	// SetpointRaiseLower payload: mode (1 byte) + amount (1 byte, signed)
	payload := []byte{mode, uint8(amount)}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterThermostat, zcl.CmdThermostatSetpointRaiseLower, payload)
}

// ============================================================================
// WindowCovering Cluster (0x0102)
// ============================================================================

// WindowCoveringStatus contains current covering position.
type WindowCoveringStatus struct {
	Type         *zcl.WindowCoveringType
	LiftPercent  *uint8 // 0=fully open, 100=fully closed
	TiltPercent  *uint8 // 0=fully open, 100=fully closed
	ConfigStatus *uint8
}

// GetWindowCoveringStatus reads the current window covering status from a device.
// Returns the covering type, lift percentage (0=fully open, 100=fully closed),
// tilt percentage, and configuration status.
func (a *Adapter) GetWindowCoveringStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*WindowCoveringStatus, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrWindowCoveringType,
		zcl.AttrWindowCoveringCurrentPositionLiftPercent,
		zcl.AttrWindowCoveringCurrentPositionTiltPercent,
		zcl.AttrWindowCoveringConfigStatus,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read window covering status: %w", err)
	}

	status := &WindowCoveringStatus{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrWindowCoveringType:
			if val, ok := r.Value.(uint8); ok {
				coveringType := zcl.WindowCoveringType(val)
				status.Type = &coveringType
			}
		case zcl.AttrWindowCoveringCurrentPositionLiftPercent:
			if val, ok := r.Value.(uint8); ok {
				status.LiftPercent = &val
			}
		case zcl.AttrWindowCoveringCurrentPositionTiltPercent:
			if val, ok := r.Value.(uint8); ok {
				status.TiltPercent = &val
			}
		case zcl.AttrWindowCoveringConfigStatus:
			if val, ok := r.Value.(uint8); ok {
				status.ConfigStatus = &val
			}
		}
	}

	return status, nil
}

// OpenWindowCovering fully opens (raises) the covering.
// This sends the UpOpen command to the device.
func (a *Adapter) OpenWindowCovering(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringUpOpen, nil)
}

// CloseWindowCovering fully closes (lowers) the covering.
// This sends the DownClose command to the device.
func (a *Adapter) CloseWindowCovering(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringDownClose, nil)
}

// StopWindowCovering stops any ongoing movement.
// This immediately halts the covering's motion and maintains the current position.
func (a *Adapter) StopWindowCovering(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringStop, nil)
}

// SetWindowCoveringPosition sets the lift position as percentage (0=open, 100=closed).
// The covering will move to the specified position.
// percent is 0-100 where 0 is fully open and 100 is fully closed.
func (a *Adapter) SetWindowCoveringPosition(ctx context.Context, nwkAddr uint16, endpoint uint8, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("percent must be 0-100, got %d", percent)
	}
	// GoToLiftPercent payload: percentageLiftValue (1 byte)
	payload := []byte{percent}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringGoToLiftPercent, payload)
}

// SetWindowCoveringTilt sets the tilt angle as percentage (0=open, 100=closed).
// The covering slats will tilt to the specified angle.
// percent is 0-100 where 0 is fully open and 100 is fully closed.
func (a *Adapter) SetWindowCoveringTilt(ctx context.Context, nwkAddr uint16, endpoint uint8, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("percent must be 0-100, got %d", percent)
	}
	// GoToTiltPercent payload: percentageTiltValue (1 byte)
	payload := []byte{percent}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterWindowCovering, zcl.CmdWindowCoveringGoToTiltPercent, payload)
}

// ============================================================================
// Fan Cluster (0x0202)
// ============================================================================

// FanStatus contains current fan state.
type FanStatus struct {
	Mode           *zcl.FanMode         // Current fan mode
	ModeSequence   *zcl.FanModeSequence // Supported fan mode sequence
	PercentSetting *uint8               // Fan speed setting as percentage (0-100%)
	PercentCurrent *uint8               // Current fan speed as percentage (0-100%)
	SpeedMax       *uint8               // Maximum fan speed
	SpeedSetting   *uint8               // Fan speed setting
	SpeedCurrent   *uint8               // Current fan speed
}

// GetFanStatus reads the current fan status from a device.
// Returns all available fan attributes including mode, speed, and percentages.
func (a *Adapter) GetFanStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*FanStatus, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrFanMode,
		zcl.AttrFanModeSequence,
		zcl.AttrFanPercentSetting,
		zcl.AttrFanPercentCurrent,
		zcl.AttrFanSpeedMax,
		zcl.AttrFanSpeedSetting,
		zcl.AttrFanSpeedCurrent,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFan, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read fan attributes: %w", err)
	}

	status := &FanStatus{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrFanMode:
			if val, ok := r.Value.(uint8); ok {
				mode := zcl.FanMode(val)
				status.Mode = &mode
			}
		case zcl.AttrFanModeSequence:
			if val, ok := r.Value.(uint8); ok {
				seq := zcl.FanModeSequence(val)
				status.ModeSequence = &seq
			}
		case zcl.AttrFanPercentSetting:
			if val, ok := r.Value.(uint8); ok {
				status.PercentSetting = &val
			}
		case zcl.AttrFanPercentCurrent:
			if val, ok := r.Value.(uint8); ok {
				status.PercentCurrent = &val
			}
		case zcl.AttrFanSpeedMax:
			if val, ok := r.Value.(uint8); ok {
				status.SpeedMax = &val
			}
		case zcl.AttrFanSpeedSetting:
			if val, ok := r.Value.(uint8); ok {
				status.SpeedSetting = &val
			}
		case zcl.AttrFanSpeedCurrent:
			if val, ok := r.Value.(uint8); ok {
				status.SpeedCurrent = &val
			}
		}
	}

	return status, nil
}

// SetFanMode sets the operating mode of a fan.
// mode specifies the desired fan mode (Off, Low, Medium, High, On, Auto, Smart).
func (a *Adapter) SetFanMode(ctx context.Context, nwkAddr uint16, endpoint uint8, mode zcl.FanMode) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrFanMode: {
			Type:  zcl.TypeEnum8,
			Value: uint8(mode),
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFan, values)
}

// SetFanSpeed sets the fan speed as a percentage.
// percent is 0-100 where 0 is off and 100 is maximum speed.
func (a *Adapter) SetFanSpeed(ctx context.Context, nwkAddr uint16, endpoint uint8, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("percent must be 0-100, got %d", percent)
	}
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrFanPercentSetting: {
			Type:  zcl.TypeUint8,
			Value: percent,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterFan, values)
}

// ============================================================================
// Time Cluster (0x000A)
// ============================================================================

// Zigbee epoch is January 1, 2000 00:00:00 UTC (946684800 Unix seconds).
const zigbeeEpoch = 946684800

// ZigbeeTimeToUnix converts Zigbee time (seconds since 2000-01-01) to Unix time.
// Zigbee time is uint32 seconds since January 1, 2000 00:00:00 UTC.
// Unix time is int64 seconds since January 1, 1970 00:00:00 UTC.
func ZigbeeTimeToUnix(zigbeeTime uint32) time.Time {
	unixSeconds := int64(zigbeeTime) + zigbeeEpoch
	return time.Unix(unixSeconds, 0)
}

// UnixToZigbeeTime converts Unix time to Zigbee time (seconds since 2000-01-01).
// Returns uint32 seconds since January 1, 2000 00:00:00 UTC.
func UnixToZigbeeTime(t time.Time) uint32 {
	unixSeconds := t.Unix()
	if unixSeconds < zigbeeEpoch {
		return 0 // Time before Zigbee epoch
	}
	return uint32(unixSeconds - zigbeeEpoch)
}

// TimeInfo contains time information from a Time cluster device.
type TimeInfo struct {
	Time       uint32 // UTC time since 2000-01-01 (Zigbee epoch)
	TimeStatus uint8  // Status bitmap
	TimeZone   int32  // Timezone offset in seconds
	LocalTime  uint32 // Local time
}

// GetTime reads the current time and status from a Time cluster device.
// Returns time information including UTC time, status flags, timezone offset, and local time.
// Use ZigbeeTimeToUnix() to convert the Time field to a standard time.Time value.
func (a *Adapter) GetTime(ctx context.Context, nwkAddr uint16, endpoint uint8) (*TimeInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrTimeTime,
		zcl.AttrTimeStatus,
		zcl.AttrTimeZone,
		zcl.AttrTimeLocalTime,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read time attributes: %w", err)
	}

	info := &TimeInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrTimeTime:
			if val, ok := toUint32(r.Value); ok {
				info.Time = val
			}
		case zcl.AttrTimeStatus:
			if val, ok := r.Value.(uint8); ok {
				info.TimeStatus = val
			}
		case zcl.AttrTimeZone:
			if val, ok := toInt32(r.Value); ok {
				info.TimeZone = val
			}
		case zcl.AttrTimeLocalTime:
			if val, ok := toUint32(r.Value); ok {
				info.LocalTime = val
			}
		}
	}

	return info, nil
}

// SetTime sets the UTC time on a Time cluster device.
// zigbeeTime is seconds since January 1, 2000 00:00:00 UTC.
// Use UnixToZigbeeTime() to convert from time.Time to Zigbee time.
//
// Example: Set device to current time:
//
//	zigbeeTime := adapter.UnixToZigbeeTime(time.Now())
//	err := adapter.SetTime(ctx, nwkAddr, endpoint, zigbeeTime)
func (a *Adapter) SetTime(ctx context.Context, nwkAddr uint16, endpoint uint8, zigbeeTime uint32) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTimeTime: {
			Type:  zcl.TypeUint32,
			Value: zigbeeTime,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, values)
}

// GetTimeZone reads the timezone offset from a Time cluster device.
// Returns the timezone offset in seconds from UTC.
// For example: UTC+2 = 7200 seconds, UTC-5 = -18000 seconds.
func (a *Adapter) GetTimeZone(ctx context.Context, nwkAddr uint16, endpoint uint8) (int32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, zcl.AttrTimeZone)
	if err != nil {
		return 0, fmt.Errorf("failed to read timezone: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read timezone returned status 0x%02X", result.Status)
	}

	if offset, ok := toInt32(result.Value); ok {
		return offset, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// SetTimeZone sets the timezone offset on a Time cluster device.
// offsetSeconds is the timezone offset in seconds from UTC.
// For example: UTC+2 = 7200 seconds, UTC-5 = -18000 seconds.
//
// Example: Set timezone to UTC+1 (Central European Time):
//
//	err := adapter.SetTimeZone(ctx, nwkAddr, endpoint, 3600)
func (a *Adapter) SetTimeZone(ctx context.Context, nwkAddr uint16, endpoint uint8, offsetSeconds int32) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTimeZone: {
			Type:  zcl.TypeInt32,
			Value: offsetSeconds,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, values)
}

// ============================================================================
// DRLC (Demand Response and Load Control) Cluster (0x0701)
// ============================================================================

// DRLCEventInfo contains information about a demand response load control event.
// This represents a request from the utility to reduce or shift energy consumption.
type DRLCEventInfo struct {
	IssuerEventID    uint32 // Unique event ID from utility
	DeviceClass      uint16 // Device class bitmap (which devices should respond)
	UtilityGroupID   uint8  // Utility enrolment group
	StartTime        uint32 // Event start time (Zigbee time, seconds since 2000-01-01)
	Duration         uint16 // Event duration in minutes
	CriticalityLevel uint8  // Criticality level (0-15, higher = more critical)
	CoolingOffset    uint8  // Temperature offset for cooling (°C)
	HeatingOffset    uint8  // Temperature offset for heating (°C)
}

// GetDRLCInfo reads the current DRLC configuration from a device.
// This retrieves the device's demand response settings including which utility group
// it belongs to, the device class, and randomization settings.
//
// The device class bitmap indicates which types of loads this device can control.
// Use the zcl.DRLCDeviceClass* constants to check which classes are supported.
//
// Returns the utility enrolment group, device class, and randomization settings.
func (a *Adapter) GetDRLCInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DRLCEventInfo, error) {
	// Read DRLC attributes
	attrs := []zcl.AttributeID{
		zcl.AttrDRLCUtilityEnrolmentGroup,
		zcl.AttrDRLCStartRandomizeMinutes,
		zcl.AttrDRLCDurationRandomizeMinutes,
		zcl.AttrDRLCDeviceClassValue,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterDRLC, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read DRLC attributes: %w", err)
	}

	info := &DRLCEventInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrDRLCUtilityEnrolmentGroup:
			if v, ok := r.Value.(uint8); ok {
				info.UtilityGroupID = v
			}
		case zcl.AttrDRLCDeviceClassValue:
			if v, ok := toUint16(r.Value); ok {
				info.DeviceClass = v
			}
		}
	}

	return info, nil
}

// ============================================================================
// IAS Warning Device Cluster (0x0502)
// ============================================================================

// StartWarning activates the warning device (siren/strobe).
// This sends a command to the device to start sounding an alarm with the specified parameters.
//
// Parameters:
//   - mode: Type of warning (Burglar, Fire, Emergency, etc.)
//   - useStrobe: Whether to activate the strobe light
//   - sirenLevel: Loudness of the siren (Low, Medium, High, VeryHigh)
//   - durationSecs: How long to sound the alarm (max depends on device, typically 240 seconds)
//   - strobeDutyCycle: Duty cycle percentage for strobe (0-100), where 100 = always on
//   - strobeLevel: Brightness of the strobe (Low, Medium, High, VeryHigh)
//
// Example: Sound burglar alarm for 30 seconds with high siren and medium strobe:
//
//	err := adapter.StartWarning(ctx, nwkAddr, endpoint,
//	    zcl.WarningModeBurglar, true, zcl.SirenLevelHigh,
//	    30, 50, zcl.StrobeLevelMedium)
func (a *Adapter) StartWarning(ctx context.Context, nwkAddr uint16, endpoint uint8, mode zcl.WarningMode, useStrobe bool, sirenLevel zcl.SirenLevel, durationSecs uint16, strobeDutyCycle uint8, strobeLevel zcl.StrobeLevel) error {
	// Build the first byte: (mode & 0x0F) | ((strobe & 0x03) << 4) | ((sirenLevel & 0x03) << 6)
	var strobeBits uint8
	if useStrobe {
		strobeBits = 0x02 // Use strobe
	} else {
		strobeBits = 0x00 // No strobe
	}

	firstByte := (uint8(mode) & 0x0F) | ((strobeBits & 0x03) << 4) | ((uint8(sirenLevel) & 0x03) << 6)

	// Build payload: firstByte (1 byte) + warningDuration (2 bytes LE) + strobeDutyCycle (1 byte) + strobeLevel (1 byte)
	payload := []byte{
		firstByte,
		byte(durationSecs & 0xFF),
		byte(durationSecs >> 8),
		strobeDutyCycle,
		uint8(strobeLevel),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASWD, zcl.CmdIASWDStartWarning, payload)
}

// StopWarning stops any active warning on the device.
// This immediately silences the alarm and turns off any strobe.
//
// This is equivalent to calling StartWarning with WarningModeStop.
func (a *Adapter) StopWarning(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// To stop warning, send StartWarning command with mode=Stop and duration=0
	payload := []byte{
		uint8(zcl.WarningModeStop), // mode=Stop, no strobe, siren level=Low
		0x00,                       // duration = 0 (LE)
		0x00,
		0x00, // duty cycle = 0
		0x00, // strobe level = Low
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASWD, zcl.CmdIASWDStartWarning, payload)
}

// Squawk makes a brief sound on the device (for arming/disarming feedback).
// This is typically used to provide audio feedback when arming or disarming a security system.
//
// Parameters:
//   - mode: Squawk mode (0=armed, 1=disarmed)
//   - useStrobe: Whether to flash the strobe
//   - level: Volume level (Low, Medium, High, VeryHigh)
//
// Example: Play disarmed squawk with medium volume and strobe:
//
//	err := adapter.Squawk(ctx, nwkAddr, endpoint, 1, true, zcl.SirenLevelMedium)
func (a *Adapter) Squawk(ctx context.Context, nwkAddr uint16, endpoint uint8, mode uint8, useStrobe bool, level zcl.SirenLevel) error {
	// Build the byte: (mode & 0x0F) | ((strobe & 0x01) << 4) | ((level & 0x03) << 6)
	var strobeBit uint8
	if useStrobe {
		strobeBit = 0x01
	}

	squawkByte := (mode & 0x0F) | ((strobeBit & 0x01) << 4) | ((uint8(level) & 0x03) << 6)

	payload := []byte{squawkByte}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASWD, zcl.CmdIASWDSquawk, payload)
}

// ============================================================================
// OTA Upgrade Cluster (0x0019)
// ============================================================================

// OTAInfo contains OTA upgrade status information.
type OTAInfo struct {
	CurrentFileVersion uint32               // Current firmware version
	UpgradeStatus      zcl.OTAUpgradeStatus // Current upgrade status
	ManufacturerID     uint16               // Manufacturer ID
	ImageTypeID        uint16               // Image type ID
	FileOffset         uint32               // Current file offset
}

// GetOTAInfo reads OTA upgrade status information from a device.
// Returns current firmware version, upgrade status, manufacturer/image IDs, and file offset.
// This provides a comprehensive view of the device's OTA upgrade state.
func (a *Adapter) GetOTAInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*OTAInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrOTACurrentFileVersion,
		zcl.AttrOTAImageUpgradeStatus,
		zcl.AttrOTAManufacturerID,
		zcl.AttrOTAImageTypeID,
		zcl.AttrOTAFileOffset,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOTAUpgrade, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read OTA attributes: %w", err)
	}

	info := &OTAInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrOTACurrentFileVersion:
			if val, ok := toUint32(r.Value); ok {
				info.CurrentFileVersion = val
			}
		case zcl.AttrOTAImageUpgradeStatus:
			if val, ok := r.Value.(uint8); ok {
				info.UpgradeStatus = zcl.OTAUpgradeStatus(val)
			}
		case zcl.AttrOTAManufacturerID:
			if val, ok := toUint16(r.Value); ok {
				info.ManufacturerID = val
			}
		case zcl.AttrOTAImageTypeID:
			if val, ok := toUint16(r.Value); ok {
				info.ImageTypeID = val
			}
		case zcl.AttrOTAFileOffset:
			if val, ok := toUint32(r.Value); ok {
				info.FileOffset = val
			}
		}
	}

	return info, nil
}

// GetOTAUpgradeStatus reads the current OTA upgrade status from a device.
// Returns the upgrade status enum indicating the device's current state
// (Normal, DownloadInProgress, DownloadComplete, etc.).
func (a *Adapter) GetOTAUpgradeStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (zcl.OTAUpgradeStatus, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOTAUpgrade, zcl.AttrOTAImageUpgradeStatus)
	if err != nil {
		return 0, fmt.Errorf("failed to read OTA upgrade status: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read OTA upgrade status returned status 0x%02X", result.Status)
	}

	if status, ok := result.Value.(uint8); ok {
		return zcl.OTAUpgradeStatus(status), nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// ============================================================================
// PollControl Cluster (0x0020)
// ============================================================================

// PollControlInfo contains poll control configuration for a sleepy end device.
type PollControlInfo struct {
	CheckInInterval   uint32 // In quarter seconds
	LongPollInterval  uint32 // In quarter seconds
	ShortPollInterval uint16 // In quarter seconds
	FastPollTimeout   uint16 // In quarter seconds
}

// GetPollControlInfo reads poll control configuration from a device.
// Returns the check-in interval, long poll interval, short poll interval,
// and fast poll timeout. All intervals are in quarter seconds.
//
// To convert to seconds, use QuarterSecondsToSeconds():
//
//	info, err := adapter.GetPollControlInfo(ctx, nwkAddr, endpoint)
//	if err != nil {
//		return err
//	}
//	checkInSecs := QuarterSecondsToSeconds(info.CheckInInterval)
func (a *Adapter) GetPollControlInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PollControlInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrPollControlCheckInInterval,
		zcl.AttrPollControlLongPollInterval,
		zcl.AttrPollControlShortPollInterval,
		zcl.AttrPollControlFastPollTimeout,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read poll control attributes: %w", err)
	}

	info := &PollControlInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrPollControlCheckInInterval:
			if val, ok := toUint32(r.Value); ok {
				info.CheckInInterval = val
			}
		case zcl.AttrPollControlLongPollInterval:
			if val, ok := toUint32(r.Value); ok {
				info.LongPollInterval = val
			}
		case zcl.AttrPollControlShortPollInterval:
			if val, ok := toUint16(r.Value); ok {
				info.ShortPollInterval = val
			}
		case zcl.AttrPollControlFastPollTimeout:
			if val, ok := toUint16(r.Value); ok {
				info.FastPollTimeout = val
			}
		}
	}

	return info, nil
}

// SendCheckInResponse responds to a check-in notification from a sleepy device.
// This tells the device whether to enter fast polling mode and for how long.
//
// Parameters:
//   - startFastPolling: Whether the device should enter fast polling mode
//   - fastPollTimeout: Duration of fast polling in quarter seconds (0 = use device default)
//
// The device will poll more frequently during fast polling, allowing the coordinator
// to send pending commands. After the timeout, the device returns to normal polling.
//
// Example: Enable fast polling for 10 seconds (40 quarter seconds):
//
//	err := adapter.SendCheckInResponse(ctx, nwkAddr, endpoint, true, 40)
func (a *Adapter) SendCheckInResponse(ctx context.Context, nwkAddr uint16, endpoint uint8, startFastPolling bool, fastPollTimeout uint16) error {
	// CheckInResponse payload: startFastPolling (1 byte bool) + fastPollTimeout (2 bytes LE)
	var startFastPoll uint8
	if startFastPolling {
		startFastPoll = 0x01
	} else {
		startFastPoll = 0x00
	}

	payload := []byte{
		startFastPoll,
		byte(fastPollTimeout & 0xFF),
		byte(fastPollTimeout >> 8),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlCheckInResponse, payload)
}

// StopFastPolling instructs a sleepy device to stop fast polling immediately.
// The device will return to its normal long poll interval.
//
// This is useful to conserve battery when you no longer have pending commands
// for the device.
func (a *Adapter) StopFastPolling(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlFastPollStop, nil)
}

// SetLongPollInterval sets the long poll interval on a sleepy device.
// The long poll interval is how often the device wakes up to check for messages
// when it's in normal (non-fast-polling) mode.
//
// intervalQuarterSecs is the interval in quarter seconds (4 = 1 second).
//
// Example: Set long poll interval to 5 minutes (1200 quarter seconds):
//
//	err := adapter.SetLongPollInterval(ctx, nwkAddr, endpoint, 1200)
func (a *Adapter) SetLongPollInterval(ctx context.Context, nwkAddr uint16, endpoint uint8, intervalQuarterSecs uint32) error {
	// SetLongPollInterval payload: newLongPollInterval (4 bytes LE)
	payload := []byte{
		byte(intervalQuarterSecs & 0xFF),
		byte((intervalQuarterSecs >> 8) & 0xFF),
		byte((intervalQuarterSecs >> 16) & 0xFF),
		byte((intervalQuarterSecs >> 24) & 0xFF),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlSetLongPollInterval, payload)
}

// SetShortPollInterval sets the short poll interval (fast poll interval) on a sleepy device.
// The short poll interval is how often the device polls when in fast polling mode.
//
// intervalQuarterSecs is the interval in quarter seconds (4 = 1 second).
//
// Example: Set short poll interval to 250ms (1 quarter second):
//
//	err := adapter.SetShortPollInterval(ctx, nwkAddr, endpoint, 1)
func (a *Adapter) SetShortPollInterval(ctx context.Context, nwkAddr uint16, endpoint uint8, intervalQuarterSecs uint16) error {
	// SetShortPollInterval payload: newShortPollInterval (2 bytes LE)
	payload := []byte{
		byte(intervalQuarterSecs & 0xFF),
		byte(intervalQuarterSecs >> 8),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlSetShortPollInterval, payload)
}

// QuarterSecondsToSeconds converts quarter seconds to seconds.
// Quarter seconds are commonly used in Zigbee for timing intervals.
// 4 quarter seconds = 1 second.
func QuarterSecondsToSeconds(quarterSecs uint32) float64 {
	return float64(quarterSecs) / 4.0
}

// SecondsToQuarterSeconds converts seconds to quarter seconds.
// Quarter seconds are commonly used in Zigbee for timing intervals.
// 1 second = 4 quarter seconds.
func SecondsToQuarterSeconds(secs float64) uint32 {
	return uint32(secs * 4.0)
}

// ============================================================================
// BinaryOutput Cluster (0x0010)
// ============================================================================

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

// ============================================================================
// MultistateOutput Cluster (0x0013)
// ============================================================================

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

// ============================================================================
// MultistateValue Cluster (0x0014)
// ============================================================================

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

// ============================================================================
// Price Cluster (0x0700) - Smart Energy Profile
// ============================================================================

// PriceInfo contains pricing information from the Price cluster.
// This is part of the Smart Energy profile used for dynamic pricing and
// demand response applications.
type PriceInfo struct {
	ProviderId         uint32 // Unique identifier for the commodity supplier
	RateLabel          string // User-defined rate label (e.g., "Peak", "Off-Peak")
	IssuerEventId      uint32 // Unique identifier for this pricing event
	CurrentTime        uint32 // Current time in Zigbee time (seconds since 2000-01-01)
	UnitOfMeasure      uint8  // Unit of measurement (kWh, m3, etc.)
	Currency           uint16 // ISO 4217 currency code (e.g., 840 = USD)
	PriceTrailingDigit uint8  // Number of digits to right of decimal point
	NumberOfTiers      uint8  // Number of price tiers in use
	Price              uint32 // Price in currency units (apply trailing digit)
	StartTime          uint32 // When this price becomes effective (Zigbee time)
	DurationInMinutes  uint16 // How long this price is valid
}

// GetCurrentPrice requests the current price from a smart energy device.
// This sends a GetCurrentPrice command to the device and waits for a
// PublishPrice response containing the current pricing information.
//
// The device must support the Price cluster (0x0700) for this to work.
// This is commonly used with smart meters and energy management systems.
//
// Returns PriceInfo with pricing details, or an error if the request fails.
func (a *Adapter) GetCurrentPrice(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PriceInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build GetCurrentPrice command (command ID 0x00, no payload for basic request)
	// The full command payload can include CommandOptions (1 byte), but for a simple
	// request for current price, we use an empty payload or 0x00 for default options.
	seqNum := nextTransactionID()
	payload := []byte{0x00} // CommandOptions: 0x00 = RequestorRxOnWhenIdle
	frame := zcl.BuildClusterCommand(seqNum, zcl.CmdPriceGetCurrentPrice, payload)

	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(zcl.ClusterPrice), frame)
	if err != nil {
		return nil, fmt.Errorf("get current price failed: %w", err)
	}

	// Parse PublishPrice response (command 0x00 from server)
	if respFrame.CommandID != zcl.CmdPricePublishPrice {
		return nil, fmt.Errorf("unexpected response command: 0x%02X (expected PublishPrice 0x00)", respFrame.CommandID)
	}

	// Parse PublishPrice payload
	// Format (minimum): providerId (4) + rateLabel (var) + issuerEventId (4) + currentTime (4) +
	//                   unitOfMeasure (1) + currency (2) + priceTrailingDigit (1) +
	//                   numberOfPriceTiers (1) + registerTier (1) + startTime (4) +
	//                   durationInMinutes (2) + price (4) + ...
	if len(respFrame.Payload) < 28 {
		return nil, fmt.Errorf("publish price response too short: %d bytes", len(respFrame.Payload))
	}

	info := &PriceInfo{}
	offset := 0

	// ProviderId (uint32)
	info.ProviderId = uint32(respFrame.Payload[offset]) |
		uint32(respFrame.Payload[offset+1])<<8 |
		uint32(respFrame.Payload[offset+2])<<16 |
		uint32(respFrame.Payload[offset+3])<<24
	offset += 4

	// RateLabel (octet string with length prefix)
	if offset >= len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at rateLabel")
	}
	labelLen := int(respFrame.Payload[offset])
	offset++
	if offset+labelLen > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated in rateLabel")
	}
	info.RateLabel = string(respFrame.Payload[offset : offset+labelLen])
	offset += labelLen

	// IssuerEventId (uint32)
	if offset+4 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at issuerEventId")
	}
	info.IssuerEventId = uint32(respFrame.Payload[offset]) |
		uint32(respFrame.Payload[offset+1])<<8 |
		uint32(respFrame.Payload[offset+2])<<16 |
		uint32(respFrame.Payload[offset+3])<<24
	offset += 4

	// CurrentTime (uint32)
	if offset+4 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at currentTime")
	}
	info.CurrentTime = uint32(respFrame.Payload[offset]) |
		uint32(respFrame.Payload[offset+1])<<8 |
		uint32(respFrame.Payload[offset+2])<<16 |
		uint32(respFrame.Payload[offset+3])<<24
	offset += 4

	// UnitOfMeasure (uint8)
	if offset+1 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at unitOfMeasure")
	}
	info.UnitOfMeasure = respFrame.Payload[offset]
	offset++

	// Currency (uint16)
	if offset+2 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at currency")
	}
	info.Currency = uint16(respFrame.Payload[offset]) | uint16(respFrame.Payload[offset+1])<<8
	offset += 2

	// PriceTrailingDigit (upper 4 bits) and PriceTier (lower 4 bits) in one byte
	if offset+1 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at priceTrailingDigit")
	}
	priceTrailingDigitAndTier := respFrame.Payload[offset]
	info.PriceTrailingDigit = (priceTrailingDigitAndTier >> 4) & 0x0F
	offset++

	// NumberOfPriceTiersAndRegisterTier (upper 4 bits = number of tiers, lower 4 bits = register tier)
	if offset+1 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at numberOfTiers")
	}
	numberOfTiersAndRegisterTier := respFrame.Payload[offset]
	info.NumberOfTiers = (numberOfTiersAndRegisterTier >> 4) & 0x0F
	offset++

	// StartTime (uint32)
	if offset+4 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at startTime")
	}
	info.StartTime = uint32(respFrame.Payload[offset]) |
		uint32(respFrame.Payload[offset+1])<<8 |
		uint32(respFrame.Payload[offset+2])<<16 |
		uint32(respFrame.Payload[offset+3])<<24
	offset += 4

	// DurationInMinutes (uint16)
	if offset+2 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at duration")
	}
	info.DurationInMinutes = uint16(respFrame.Payload[offset]) | uint16(respFrame.Payload[offset+1])<<8
	offset += 2

	// Price (uint32)
	if offset+4 > len(respFrame.Payload) {
		return nil, fmt.Errorf("payload truncated at price")
	}
	info.Price = uint32(respFrame.Payload[offset]) |
		uint32(respFrame.Payload[offset+1])<<8 |
		uint32(respFrame.Payload[offset+2])<<16 |
		uint32(respFrame.Payload[offset+3])<<24

	return info, nil
}

// ============================================================================
// Smart Energy Messaging Cluster (0x0703)
// ============================================================================

// SEMessage represents a Smart Energy message to be displayed on a device.
// This is used with the Messaging cluster (0x0703) in the Smart Energy profile.
type SEMessage struct {
	MessageID uint32 // Unique message identifier
	Control   uint8  // MessageControl bitmap (transmission, importance, confirmation)
	StartTime uint32 // Zigbee time when message should be displayed (0 = now)
	Duration  uint16 // Duration to display message in minutes (0xFFFF = until explicitly cancelled)
	Message   string // Message text to display
}

// DisplaySEMessage sends a message to be displayed on a Smart Energy device.
// The message is sent using the Messaging cluster (0x0703) DisplayMessage command.
//
// Parameters:
//   - msg: The message to display, including message ID, control flags, timing, and text
//
// The Control field is a bitmap specifying:
//   - Transmission type (normal or anonymous)
//   - Importance level (low, medium, high, critical)
//   - Whether confirmation is required
//
// Use zcl.SEMsgCtrl* constants to build the control byte.
//
// Example: Display a high-importance message immediately for 60 minutes:
//
//	msg := adapter.SEMessage{
//	    MessageID: 1,
//	    Control:   zcl.SEMsgCtrlImportanceHigh | zcl.SEMsgCtrlConfirmationRequired,
//	    StartTime: 0, // Display now
//	    Duration:  60, // 60 minutes
//	    Message:   "High electricity demand. Please reduce usage.",
//	}
//	err := adapter.DisplaySEMessage(ctx, nwkAddr, endpoint, msg)
func (a *Adapter) DisplaySEMessage(ctx context.Context, nwkAddr uint16, endpoint uint8, msg SEMessage) error {
	// DisplayMessage payload format:
	// - messageId (4 bytes LE)
	// - messageControl (1 byte)
	// - startTime (4 bytes LE)
	// - durationInMinutes (2 bytes LE)
	// - message (string with 1-byte length prefix)

	msgBytes := []byte(msg.Message)
	if len(msgBytes) > 255 {
		msgBytes = msgBytes[:255] // Maximum message length per ZCL spec
	}

	payload := make([]byte, 12+len(msgBytes))

	// MessageID (4 bytes LE)
	payload[0] = byte(msg.MessageID & 0xFF)
	payload[1] = byte((msg.MessageID >> 8) & 0xFF)
	payload[2] = byte((msg.MessageID >> 16) & 0xFF)
	payload[3] = byte((msg.MessageID >> 24) & 0xFF)

	// MessageControl (1 byte)
	payload[4] = msg.Control

	// StartTime (4 bytes LE)
	payload[5] = byte(msg.StartTime & 0xFF)
	payload[6] = byte((msg.StartTime >> 8) & 0xFF)
	payload[7] = byte((msg.StartTime >> 16) & 0xFF)
	payload[8] = byte((msg.StartTime >> 24) & 0xFF)

	// Duration (2 bytes LE)
	payload[9] = byte(msg.Duration & 0xFF)
	payload[10] = byte(msg.Duration >> 8)

	// Message string (length + text)
	payload[11] = byte(len(msgBytes))
	copy(payload[12:], msgBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterSEMessaging, zcl.CmdSEMessagingDisplayMessage, payload)
}

// CancelSEMessage cancels a specific message on a Smart Energy device.
// This sends a CancelMessage command to remove a previously displayed message.
//
// Parameters:
//   - messageID: The unique identifier of the message to cancel
//
// Example: Cancel message with ID 1:
//
//	err := adapter.CancelSEMessage(ctx, nwkAddr, endpoint, 1)
func (a *Adapter) CancelSEMessage(ctx context.Context, nwkAddr uint16, endpoint uint8, messageID uint32) error {
	// CancelMessage payload: messageId (4 bytes LE) + messageControl (1 byte)
	// We use control byte 0x00 for normal cancellation
	payload := []byte{
		byte(messageID & 0xFF),
		byte((messageID >> 8) & 0xFF),
		byte((messageID >> 16) & 0xFF),
		byte((messageID >> 24) & 0xFF),
		0x00, // messageControl (not used for cancel, set to 0)
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterSEMessaging, zcl.CmdSEMessagingCancelMessage, payload)
}

// GetLastSEMessage requests the last message from a Smart Energy device.
// This sends a GetLastMessage command and waits for the device to respond with
// the last message it received.
//
// Returns the last message, or nil if no message is available.
//
// Note: This is an asynchronous operation. The device will respond with a
// DisplayMessage command containing the last message. This method sends the
// request but does not wait for the response. You'll need to listen for
// incoming DisplayMessage commands to receive the message.
//
// Example:
//
//	err := adapter.GetLastSEMessage(ctx, nwkAddr, endpoint)
//	// Listen for DisplayMessage response separately
func (a *Adapter) GetLastSEMessage(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// GetLastMessage command has no payload
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterSEMessaging, zcl.CmdSEMessagingGetLastMessage, nil)
}

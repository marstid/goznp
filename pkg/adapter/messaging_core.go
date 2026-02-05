package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

// Default coordinator endpoint (Home Automation profile).
const CoordinatorEndpoint = 1

// AttributeResult represents a single attribute read result.
type AttributeResult struct {
	AttributeID zcl.AttributeID
	Status      zcl.Status
	DataType    zcl.DataType
	Value       interface{}
}

// ReadAttributes reads one or more attributes from a device endpoint.
//
// This method sends a ZCL read attributes request to the specified device and
// waits for the response. Multiple attributes can be requested in a single call
// to improve efficiency.
//
// Parameters:
//   - ctx: Context for the operation (for cancellation/timeouts)
//   - nwkAddr: Network address of the target device (16-bit)
//   - endpoint: Endpoint number on the device (1-240)
//   - clusterID: Cluster ID containing the attributes
//   - attributeIDs: One or more attribute IDs to read
//
// Returns:
//   - A slice of AttributeResult, one per requested attribute
//   - An error if the request fails
//
// The returned results include the status, data type, and value for each
// attribute. If an attribute read fails, the status field will indicate
// the error (e.g., unsupported attribute, unauthorized read).
//
// Example:
//
//	// Read OnOff attribute from a bulb
//	results, err := adapter.ReadAttributes(ctx, 0x1234, 1, zcl.ClusterOnOff, zcl.AttrOnOff)
//	if err != nil {
//	    return err
//	}
//	for _, r := range results {
//	    if r.Status == zcl.StatusSuccess && r.AttributeID == zcl.AttrOnOff {
//	        isOn := r.Value.(bool)
//	        fmt.Printf("Bulb is %s\n", map[bool]string{true: "on", false: "off"}[isOn])
//	    }
//	}
func (a *Adapter) ReadAttributes(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, attributeIDs ...zcl.AttributeID) ([]AttributeResult, error) {
	if err := a.checkOpen(ctx); err != nil {
		return nil, err
	}
	if err := validateNetworkAddress(nwkAddr); err != nil {
		return nil, err
	}
	if err := validateEndpoint(endpoint); err != nil {
		return nil, err
	}
	if len(attributeIDs) == 0 {
		return nil, fmt.Errorf("at least one attribute ID must be specified")
	}

	// Build read attributes request
	seqNum := a.nextTransactionID()
	frame := zcl.BuildReadAttributesRequest(seqNum, attributeIDs...)

	// Send request and wait for response
	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(clusterID), frame)
	if err != nil {
		return nil, fmt.Errorf("read attributes failed [nwk:0x%04X ep:%d cluster:0x%04X]: %w",
			nwkAddr, endpoint, clusterID, err)
	}

	// Parse response
	if respFrame.CommandID != uint8(zcl.CmdReadAttributesResponse) {
		return nil, fmt.Errorf("unexpected response command [nwk:0x%04X ep:%d cluster:0x%04X]: 0x%02X",
			nwkAddr, endpoint, clusterID, respFrame.CommandID)
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
// WriteAttributes writes one or more attributes to a device endpoint.
//
// This method sends a ZCL write attributes request to the specified device.
// Multiple attributes can be written in a single call for efficiency.
//
// Parameters:
//   - ctx: Context for the operation (for cancellation/timeouts)
//   - nwkAddr: Network address of the target device (16-bit)
//   - endpoint: Endpoint number on the device (1-240)
//   - clusterID: Cluster ID containing the attributes
//   - values: Map of attribute ID to AttributeValue (value and type)
//
// Returns:
//   - An error if any attribute write fails
//
// All attributes in the request must be successfully written, or an error
// is returned. The response from the device indicates the status for each
// attribute write.
//
// Example:
//
//	// Turn on a bulb and set brightness
//	values := map[zcl.AttributeID]zcl.AttributeValue{
//	    zcl.AttrOnOff: {Type: zcl.TypeBoolean, Value: true},
//	    zcl.AttrLevelCurrentLevel: {Type: zcl.TypeUint8, Value: uint8(200)},
//	}
//	if err := adapter.WriteAttributes(ctx, 0x1234, 1, zcl.ClusterOnOff, values); err != nil {
//	    return err
//	}
func (a *Adapter) WriteAttributes(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, values map[zcl.AttributeID]zcl.AttributeValue) error {
	if err := a.checkOpen(ctx); err != nil {
		return err
	}
	if err := validateNetworkAddress(nwkAddr); err != nil {
		return err
	}
	if err := validateEndpoint(endpoint); err != nil {
		return err
	}
	if len(values) == 0 {
		return fmt.Errorf("at least one attribute value must be specified")
	}

	// Build write attributes request
	seqNum := a.nextTransactionID()
	frame, err := zcl.BuildWriteAttributesRequest(seqNum, values)
	if err != nil {
		return fmt.Errorf("failed to build write attributes request: %w", err)
	}

	// Send request and wait for response
	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(clusterID), frame)
	if err != nil {
		return fmt.Errorf("write attributes failed [nwk:0x%04X ep:%d cluster:0x%04X]: %w",
			nwkAddr, endpoint, clusterID, err)
	}

	// Parse response
	if respFrame.CommandID != uint8(zcl.CmdWriteAttributesResponse) {
		return fmt.Errorf("unexpected response command [nwk:0x%04X ep:%d cluster:0x%04X]: 0x%02X",
			nwkAddr, endpoint, clusterID, respFrame.CommandID)
	}

	writeResults, err := zcl.ParseWriteAttributesResponse(respFrame.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse write attributes response [nwk:0x%04X ep:%d cluster:0x%04X]: %w",
			nwkAddr, endpoint, clusterID, err)
	}

	// Check if any writes failed
	for _, r := range writeResults {
		if r.Status != zcl.StatusSuccess {
			return fmt.Errorf("write attribute 0x%04X failed [nwk:0x%04X ep:%d cluster:0x%04X] with status 0x%02X",
				r.AttributeID, nwkAddr, endpoint, clusterID, r.Status)
		}
	}

	return nil
}

// ConfigureReporting configures a device to automatically report attribute changes.
//
// This method configures a device to push attribute updates instead of requiring
// polling, which is more efficient and reduces network traffic. The device will
// send reports when:
//  1. The attribute changes by at least reportableChange (for analog types)
//  2. The time since the last report exceeds maxInterval (periodic reports)
//  3. The time since the last report is at least minInterval (rate limiting)
//
// Parameters:
//   - ctx: Context for the operation
//   - nwkAddr: Network address of the target device (16-bit)
//   - endpoint: Endpoint number on the device (1-240)
//   - clusterID: Cluster ID containing the attribute
//   - attributeID: Attribute ID to configure
//   - dataType: ZCL data type of the attribute
//   - minInterval: Minimum reporting interval in seconds (0 = no minimum)
//   - maxInterval: Maximum reporting interval in seconds (0xFFFF = no periodic reports)
//   - reportableChange: Minimum delta to trigger report (for analog), nil for discrete
//
// Returns:
//   - An error if configuration fails
//
// Notes:
//   - Use appropriate reportableChange values based on the attribute's precision
//   - For temperature in 0.01°C units: value 50 = 0.5°C change
//   - For discrete attributes (enums, booleans), use nil for reportableChange
//   - Some sleepy devices may not support reporting and will need polling
//
// Example:
//
//	// Configure temperature sensor to report every 60-300s or on 0.5°C change
//	err := adapter.ConfigureReporting(ctx, 0x1234, 1, zcl.ClusterTempMeasurement,
//	    zcl.AttrTempMeasuredValue, zcl.TypeInt16, 60, 300, int16(50)) // 50 = 0.5°C
//	if err != nil {
//	    return fmt.Errorf("failed to configure reporting: %w", err)
//	}
func (a *Adapter) ConfigureReporting(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, attributeID zcl.AttributeID, dataType zcl.DataType, minInterval, maxInterval uint16, reportableChange interface{}) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	a.mu.Unlock()

	// Build configure reporting request
	seqNum := a.nextTransactionID()
	config := zcl.ReportingConfig{
		Direction:        ReportingDirectionDeviceToCoordinator,
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
		return fmt.Errorf("configure reporting failed [nwk:0x%04X ep:%d cluster:0x%04X attr:0x%04X]: %w",
			nwkAddr, endpoint, clusterID, attributeID, err)
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
		return fmt.Errorf("unexpected response command [nwk:0x%04X ep:%d cluster:0x%04X attr:0x%04X]: 0x%02X",
			nwkAddr, endpoint, clusterID, attributeID, respFrame.CommandID)
	}

	var results []zcl.ConfigureReportingResult
	results, err = zcl.ParseConfigureReportingResponse(respFrame.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse configure reporting response [nwk:0x%04X ep:%d cluster:0x%04X attr:0x%04X]: %w",
			nwkAddr, endpoint, clusterID, attributeID, err)
	}

	// Check if configuration succeeded
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			return fmt.Errorf("configure reporting for attribute 0x%04X failed [nwk:0x%04X ep:%d cluster:0x%04X] with status 0x%02X",
				r.AttributeID, nwkAddr, endpoint, clusterID, r.Status)
		}
	}

	return nil
}

// ReportingConfigResult contains the reporting configuration for a single attribute.
type ReportingConfigResult struct {
	Status           zcl.Status
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
	seqNum := a.nextTransactionID()
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
	seqNum := a.nextTransactionID()
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

// sendZclRequest sends a ZCL request and waits for the response.
// This is a helper function used by various ZCL operations.
func (a *Adapter) sendZclRequest(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID uint16, frame *zcl.Frame) (*zcl.Frame, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Determine retry configuration
	maxAttempts := a.options.ZCLRetryAttempts + 1
	retryDelay := a.options.ZCLRetryDelay

	var lastErr error
	var respFrame *zcl.Frame

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Skip delay on first attempt
		if attempt > 0 {
			select {
			case <-time.After(retryDelay):
				// Delay elapsed
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			// Calculate exponential backoff for next retry
			retryDelay *= 2
		}

		// Build new transaction ID for each attempt
		frame.TransSeqNum = a.nextTransactionID()

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
			lastErr = fmt.Errorf("data request failed: %w", err)
			if attempt < maxAttempts-1 {
				a.options.Logger.Debugf("ZCL request attempt %d/%d failed: %v, retrying...",
					attempt+1, maxAttempts, err)
				continue
			}
			break
		}

		if status != 0 {
			lastErr = fmt.Errorf("data request returned status 0x%02X", status)
			if attempt < maxAttempts-1 {
				a.options.Logger.Debugf("ZCL request attempt %d/%d failed with status 0x%02X, retrying...",
					attempt+1, maxAttempts, status)
				continue
			}
			break
		}

		// Wait for data confirm
		confirm, err := znpClient.WaitForDataConfirm(ctx, frame.TransSeqNum, 5*time.Second)
		if err != nil {
			lastErr = fmt.Errorf("waiting for data confirm: %w", err)
			if attempt < maxAttempts-1 {
				a.options.Logger.Debugf("ZCL request attempt %d/%d failed waiting for data confirm: %v, retrying...",
					attempt+1, maxAttempts, err)
				continue
			}
			break
		}

		if confirm.Status != 0 {
			lastErr = fmt.Errorf("data confirm returned status 0x%02X", confirm.Status)
			if attempt < maxAttempts-1 {
				a.options.Logger.Debugf("ZCL request attempt %d/%d failed with data confirm status 0x%02X, retrying...",
					attempt+1, maxAttempts, confirm.Status)
				continue
			}
			break
		}

		// Wait for incoming response
		msg, err := znpClient.WaitForIncomingMsg(ctx, nwkAddr, clusterID, 5*time.Second)
		if err != nil {
			lastErr = fmt.Errorf("waiting for response: %w", err)
			if attempt < maxAttempts-1 {
				a.options.Logger.Debugf("ZCL request attempt %d/%d failed waiting for response: %v, retrying...",
					attempt+1, maxAttempts, err)
				continue
			}
			break
		}

		// Parse ZCL frame
		respFrame, err = zcl.ParseFrame(msg.Data)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse ZCL frame: %w", err)
			if attempt < maxAttempts-1 {
				a.options.Logger.Debugf("ZCL request attempt %d/%d failed to parse response: %v, retrying...",
					attempt+1, maxAttempts, err)
				continue
			}
			break
		}

		// Success!
		return respFrame, nil
	}

	// All retries exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("ZCL request failed after %d attempts: %w", maxAttempts, lastErr)
	}
	return nil, fmt.Errorf("ZCL request failed after %d attempts", maxAttempts)
}

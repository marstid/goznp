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

// ReadAttributes reads attributes from a device endpoint.
func (a *Adapter) ReadAttributes(ctx context.Context, nwkAddr uint16, endpoint uint8, clusterID zcl.ClusterID, attributeIDs ...zcl.AttributeID) ([]AttributeResult, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build read attributes request
	seqNum := a.nextTransactionID()
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
	seqNum := a.nextTransactionID()
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
	seqNum := a.nextTransactionID()
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

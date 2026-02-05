package message

import (
	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

// ParsedMessage represents a parsed ZCL message from a device.
type ParsedMessage struct {
	Source      *DeviceIdentifier
	ClusterID   uint16
	Endpoint    uint8
	CommandID   uint8
	TransSeqNum uint8
	Attributes  []adapter.ReportedAttribute
	Error       error
}

// DeviceIdentifier identifies the source of a message.
type DeviceIdentifier struct {
	IEEEAddr [8]byte
	NwkAddr  uint16
}

// Parser parses ZCL messages and extracts device state updates.
type Parser struct {
	// Any parser-specific configuration can go here
}

// NewParser creates a new message parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses an incoming ZCL message and returns the parsed result.
func (p *Parser) Parse(msg *znp.IncomingMessage) *ParsedMessage {
	result := &ParsedMessage{
		ClusterID:   msg.ClusterID,
		Endpoint:    msg.SrcEndpoint,
		TransSeqNum: msg.TransSeqNum,
	}

	frame, err := zcl.ParseFrame(msg.Data)
	if err != nil {
		result.Error = err
		return result
	}

	result.CommandID = frame.CommandID

	switch frame.CommandID {
	case uint8(zcl.CmdReportAttributes):
		return p.parseReportAttributes(result, msg, frame)
	case uint8(zcl.CmdReadAttributesResponse):
		return p.parseReadResponse(result, msg, frame)
	case uint8(zcl.CmdWriteAttributesResponse):
		return p.parseWriteResponse(result, msg, frame)
	case uint8(zcl.CmdDefaultResponse):
		return p.parseDefaultResponse(result, msg, frame)
	default:
		// Unknown or unimplemented command
		return result
	}
}

// parseReportAttributes parses a Report Attributes command.
func (p *Parser) parseReportAttributes(result *ParsedMessage, msg *znp.IncomingMessage, frame *zcl.Frame) *ParsedMessage {
	attrs, err := zcl.ParseReportAttributesPayload(frame.Payload)
	if err != nil {
		result.Error = err
		return result
	}

	result.Attributes = make([]adapter.ReportedAttribute, len(attrs))
	for i, attr := range attrs {
		result.Attributes[i] = adapter.ReportedAttribute{
			ID:    uint16(attr.AttributeID),
			Value: attr.Value,
		}
	}

	return result
}

// parseReadResponse parses a Read Attributes Response command.
func (p *Parser) parseReadResponse(result *ParsedMessage, msg *znp.IncomingMessage, frame *zcl.Frame) *ParsedMessage {
	attrs, err := zcl.ParseReadAttributesResponse(frame.Payload)
	if err != nil {
		result.Error = err
		return result
	}

	result.Attributes = make([]adapter.ReportedAttribute, len(attrs))
	for i, attr := range attrs {
		result.Attributes[i] = adapter.ReportedAttribute{
			ID:    uint16(attr.AttributeID),
			Value: attr.Value,
		}
	}

	return result
}

// parseWriteResponse parses a Write Attributes Response command.
func (p *Parser) parseWriteResponse(result *ParsedMessage, msg *znp.IncomingMessage, frame *zcl.Frame) *ParsedMessage {
	results, err := zcl.ParseWriteAttributesResponse(frame.Payload)
	if err != nil {
		result.Error = err
		return result
	}

	// Write response contains status codes
	result.Attributes = make([]adapter.ReportedAttribute, 0, len(results))
	for _, res := range results {
		if res.AttributeID != 0 {
			result.Attributes = append(result.Attributes, adapter.ReportedAttribute{
				ID:    uint16(res.AttributeID),
				Value: nil,
			})
		}
	}

	return result
}

// parseDefaultResponse parses a Default Response command.
func (p *Parser) parseDefaultResponse(result *ParsedMessage, msg *znp.IncomingMessage, frame *zcl.Frame) *ParsedMessage {
	// Default response contains a status code
	// We don't need to extract attributes, just acknowledge it was received
	return result
}

package zcl

import (
	"encoding/binary"
	"fmt"
)

// FrameType defines ZCL frame types.
type FrameType uint8

const (
	FrameTypeGlobal  FrameType = 0x00 // Profile-wide commands
	FrameTypeCluster FrameType = 0x01 // Cluster-specific commands
)

// Direction defines command direction.
type Direction uint8

const (
	DirectionClientToServer Direction = 0
	DirectionServerToClient Direction = 1
)

// GlobalCommand defines profile-wide ZCL commands.
type GlobalCommand uint8

const (
	CmdReadAttributes           GlobalCommand = 0x00
	CmdReadAttributesResponse   GlobalCommand = 0x01
	CmdWriteAttributes          GlobalCommand = 0x02
	CmdWriteAttributesUndivided GlobalCommand = 0x03
	CmdWriteAttributesResponse  GlobalCommand = 0x04
	CmdWriteAttributesNoResp    GlobalCommand = 0x05
	CmdConfigureReporting       GlobalCommand = 0x06
	CmdConfigureReportingResp   GlobalCommand = 0x07
	CmdReadReportingConfig      GlobalCommand = 0x08
	CmdReadReportingConfigResp  GlobalCommand = 0x09
	CmdReportAttributes         GlobalCommand = 0x0A
	CmdDefaultResponse          GlobalCommand = 0x0B
	CmdDiscoverAttributes       GlobalCommand = 0x0C
	CmdDiscoverAttributesResp   GlobalCommand = 0x0D
)

// On/Off Cluster Commands
const (
	CmdOnOffOff                     uint8 = 0x00
	CmdOnOffOn                      uint8 = 0x01
	CmdOnOffToggle                  uint8 = 0x02
	CmdOnOffOffWithEffect           uint8 = 0x40
	CmdOnOffOnWithRecallGlobalScene uint8 = 0x41
	CmdOnOffOnWithTimedOff          uint8 = 0x42
)

// Level Control Cluster Commands
const (
	CmdLevelMoveToLevel          uint8 = 0x00
	CmdLevelMove                 uint8 = 0x01
	CmdLevelStep                 uint8 = 0x02
	CmdLevelStop                 uint8 = 0x03
	CmdLevelMoveToLevelWithOnOff uint8 = 0x04
	CmdLevelMoveWithOnOff        uint8 = 0x05
	CmdLevelStepWithOnOff        uint8 = 0x06
	CmdLevelStopWithOnOff        uint8 = 0x07
	CmdLevelMoveToClosestFreq    uint8 = 0x08
)

// ZCL Status Codes
const (
	StatusSuccess           uint8 = 0x00
	StatusFailure           uint8 = 0x01
	StatusUnsupportedAttr   uint8 = 0x86
	StatusInvalidField      uint8 = 0x85
	StatusInvalidValue      uint8 = 0x87
	StatusReadOnly          uint8 = 0x88
	StatusInsufficientSpace uint8 = 0x89
	StatusNotFound          uint8 = 0x8B
	StatusTimeout           uint8 = 0x94
)

// FrameControl defines the ZCL frame control byte.
type FrameControl struct {
	FrameType              FrameType
	ManufacturerSpecific   bool
	Direction              Direction
	DisableDefaultResponse bool
}

// ToByte encodes frame control to a byte.
func (fc FrameControl) ToByte() uint8 {
	var b uint8
	b |= uint8(fc.FrameType & 0x03)
	if fc.ManufacturerSpecific {
		b |= 0x04
	}
	b |= uint8(fc.Direction&0x01) << 3
	if fc.DisableDefaultResponse {
		b |= 0x10
	}
	return b
}

// ParseFrameControl decodes frame control from a byte.
func ParseFrameControl(b uint8) FrameControl {
	return FrameControl{
		FrameType:              FrameType(b & 0x03),
		ManufacturerSpecific:   (b & 0x04) != 0,
		Direction:              Direction((b >> 3) & 0x01),
		DisableDefaultResponse: (b & 0x10) != 0,
	}
}

// Frame represents a ZCL frame.
type Frame struct {
	Control     FrameControl
	ManufCode   uint16 // Only present if ManufacturerSpecific
	TransSeqNum uint8
	CommandID   uint8
	Payload     []byte
}

// ToBytes encodes the frame to bytes.
func (f *Frame) ToBytes() []byte {
	size := 3 // control + seqnum + commandID
	if f.Control.ManufacturerSpecific {
		size += 2 // manufacturer code
	}
	size += len(f.Payload)

	result := make([]byte, 0, size)
	result = append(result, f.Control.ToByte())

	if f.Control.ManufacturerSpecific {
		manufCode := make([]byte, 2)
		binary.LittleEndian.PutUint16(manufCode, f.ManufCode)
		result = append(result, manufCode...)
	}

	result = append(result, f.TransSeqNum)
	result = append(result, f.CommandID)
	result = append(result, f.Payload...)

	return result
}

// ParseFrame decodes a ZCL frame from bytes.
func ParseFrame(data []byte) (*Frame, error) {
	if len(data) < 3 {
		return nil, fmt.Errorf("ZCL frame too short: need at least 3 bytes, have %d", len(data))
	}

	frame := &Frame{
		Control: ParseFrameControl(data[0]),
	}

	offset := 1

	if frame.Control.ManufacturerSpecific {
		if len(data) < 5 {
			return nil, fmt.Errorf("ZCL frame too short for manufacturer code: need at least 5 bytes, have %d", len(data))
		}
		frame.ManufCode = binary.LittleEndian.Uint16(data[offset:])
		offset += 2
	}

	frame.TransSeqNum = data[offset]
	offset++

	frame.CommandID = data[offset]
	offset++

	frame.Payload = data[offset:]

	return frame, nil
}

// BuildReadAttributesRequest creates a read attributes request frame.
func BuildReadAttributesRequest(seqNum uint8, attributeIDs ...AttributeID) *Frame {
	payload := make([]byte, len(attributeIDs)*2)
	for i, attrID := range attributeIDs {
		binary.LittleEndian.PutUint16(payload[i*2:], uint16(attrID))
	}

	return &Frame{
		Control: FrameControl{
			FrameType:              FrameTypeGlobal,
			ManufacturerSpecific:   false,
			Direction:              DirectionClientToServer,
			DisableDefaultResponse: false,
		},
		TransSeqNum: seqNum,
		CommandID:   uint8(CmdReadAttributes),
		Payload:     payload,
	}
}

// AttributeValue represents an attribute with its type and value.
type AttributeValue struct {
	Type  DataType
	Value interface{}
}

// BuildWriteAttributesRequest creates a write attributes request.
// values is a map of attribute ID to AttributeValue pairs.
func BuildWriteAttributesRequest(seqNum uint8, values map[AttributeID]AttributeValue) *Frame {
	payload := make([]byte, 0)

	for attrID, attrVal := range values {
		// Attribute ID (2 bytes)
		attrIDBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(attrIDBytes, uint16(attrID))
		payload = append(payload, attrIDBytes...)

		// Data type (1 byte)
		payload = append(payload, uint8(attrVal.Type))

		// Value
		valueBytes, err := WriteValue(attrVal.Value, attrVal.Type)
		if err != nil {
			// In production code, you'd want to handle this error properly
			// For now, skip this attribute
			continue
		}
		payload = append(payload, valueBytes...)
	}

	return &Frame{
		Control: FrameControl{
			FrameType:              FrameTypeGlobal,
			ManufacturerSpecific:   false,
			Direction:              DirectionClientToServer,
			DisableDefaultResponse: false,
		},
		TransSeqNum: seqNum,
		CommandID:   uint8(CmdWriteAttributes),
		Payload:     payload,
	}
}

// BuildClusterCommand creates a cluster-specific command frame.
func BuildClusterCommand(seqNum uint8, commandID uint8, payload []byte) *Frame {
	return &Frame{
		Control: FrameControl{
			FrameType:              FrameTypeCluster,
			ManufacturerSpecific:   false,
			Direction:              DirectionClientToServer,
			DisableDefaultResponse: false,
		},
		TransSeqNum: seqNum,
		CommandID:   commandID,
		Payload:     payload,
	}
}

// ReadAttributeResult represents a single attribute in a read response.
type ReadAttributeResult struct {
	AttributeID AttributeID
	Status      uint8 // 0 = success
	DataType    DataType
	Value       interface{}
}

// ParseReadAttributesResponse parses a read attributes response.
// Format: [attrID:2][status:1][dataType:1 if success][value if success]...
func ParseReadAttributesResponse(payload []byte) ([]ReadAttributeResult, error) {
	results := make([]ReadAttributeResult, 0)
	offset := 0

	for offset < len(payload) {
		if offset+3 > len(payload) {
			return nil, fmt.Errorf("insufficient data for attribute result header at offset %d", offset)
		}

		result := ReadAttributeResult{
			AttributeID: AttributeID(binary.LittleEndian.Uint16(payload[offset:])),
			Status:      payload[offset+2],
		}
		offset += 3

		if result.Status == StatusSuccess {
			if offset >= len(payload) {
				return nil, fmt.Errorf("insufficient data for data type at offset %d", offset)
			}

			result.DataType = DataType(payload[offset])
			offset++

			value, consumed, err := ReadValue(payload[offset:], result.DataType)
			if err != nil {
				return nil, fmt.Errorf("failed to read value for attribute 0x%04X: %w", result.AttributeID, err)
			}

			result.Value = value
			offset += consumed
		}

		results = append(results, result)
	}

	return results, nil
}

// WriteAttributeResult represents a write result in a write response.
type WriteAttributeResult struct {
	Status      uint8
	AttributeID AttributeID // Only present if status != 0
}

// ParseWriteAttributesResponse parses a write attributes response.
// Format: [status:1][attrID:2 if status != 0]...
func ParseWriteAttributesResponse(payload []byte) ([]WriteAttributeResult, error) {
	results := make([]WriteAttributeResult, 0)
	offset := 0

	// If the first status is 0x00, it means all writes succeeded
	if len(payload) > 0 && payload[0] == StatusSuccess {
		results = append(results, WriteAttributeResult{
			Status: StatusSuccess,
		})
		return results, nil
	}

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		result := WriteAttributeResult{
			Status: payload[offset],
		}
		offset++

		if result.Status != StatusSuccess {
			if offset+2 > len(payload) {
				return nil, fmt.Errorf("insufficient data for attribute ID at offset %d", offset)
			}
			result.AttributeID = AttributeID(binary.LittleEndian.Uint16(payload[offset:]))
			offset += 2
		}

		results = append(results, result)
	}

	return results, nil
}

// ReportAttributeResult represents a single attribute in a report.
type ReportAttributeResult struct {
	AttributeID AttributeID
	DataType    DataType
	Value       interface{}
}

// ParseReportAttributesPayload parses a report attributes payload.
// Format: [attrID:2][dataType:1][value]...
// Unlike read response, reports don't include status (implicitly success).
func ParseReportAttributesPayload(payload []byte) ([]ReportAttributeResult, error) {
	results := make([]ReportAttributeResult, 0)
	offset := 0

	for offset < len(payload) {
		if offset+3 > len(payload) {
			return nil, fmt.Errorf("insufficient data for attribute header at offset %d", offset)
		}

		result := ReportAttributeResult{
			AttributeID: AttributeID(binary.LittleEndian.Uint16(payload[offset:])),
			DataType:    DataType(payload[offset+2]),
		}
		offset += 3

		value, consumed, err := ReadValue(payload[offset:], result.DataType)
		if err != nil {
			return nil, fmt.Errorf("failed to read value for attribute 0x%04X: %w", result.AttributeID, err)
		}

		result.Value = value
		offset += consumed

		results = append(results, result)
	}

	return results, nil
}

// ReportingConfig represents the configuration for a single attribute reporting.
type ReportingConfig struct {
	Direction        uint8 // 0x00 = reported (send reports), 0x01 = received (receive reports)
	AttributeID      AttributeID
	DataType         DataType    // Only for direction 0x00
	MinInterval      uint16      // Minimum reporting interval in seconds (only for direction 0x00)
	MaxInterval      uint16      // Maximum reporting interval in seconds (only for direction 0x00)
	ReportableChange interface{} // Reportable change value (only for analog types, direction 0x00)
}

// BuildConfigureReportingRequest creates a configure reporting request frame.
// For most use cases, use direction=0x00 (device reports to coordinator).
// reportableChange should match the data type (e.g., int16 for temperature).
func BuildConfigureReportingRequest(seqNum uint8, configs []ReportingConfig) (*Frame, error) {
	payload := make([]byte, 0)

	for _, cfg := range configs {
		// Direction (1 byte)
		payload = append(payload, cfg.Direction)

		// Attribute ID (2 bytes)
		attrIDBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(attrIDBytes, uint16(cfg.AttributeID))
		payload = append(payload, attrIDBytes...)

		if cfg.Direction == 0x00 {
			// For "reported" direction (device sends reports)

			// Data type (1 byte)
			payload = append(payload, uint8(cfg.DataType))

			// Min reporting interval (2 bytes)
			minBytes := make([]byte, 2)
			binary.LittleEndian.PutUint16(minBytes, cfg.MinInterval)
			payload = append(payload, minBytes...)

			// Max reporting interval (2 bytes)
			maxBytes := make([]byte, 2)
			binary.LittleEndian.PutUint16(maxBytes, cfg.MaxInterval)
			payload = append(payload, maxBytes...)

			// Reportable change (variable, only for analog/discrete types)
			if cfg.ReportableChange != nil {
				changeBytes, err := WriteValue(cfg.ReportableChange, cfg.DataType)
				if err != nil {
					return nil, fmt.Errorf("failed to write reportable change for attribute 0x%04X: %w", cfg.AttributeID, err)
				}
				payload = append(payload, changeBytes...)
			}
		} else if cfg.Direction == 0x01 {
			// For "received" direction (device receives reports)
			// Only timeout period (2 bytes)
			timeoutBytes := make([]byte, 2)
			binary.LittleEndian.PutUint16(timeoutBytes, cfg.MaxInterval)
			payload = append(payload, timeoutBytes...)
		}
	}

	return &Frame{
		Control: FrameControl{
			FrameType:              FrameTypeGlobal,
			ManufacturerSpecific:   false,
			Direction:              DirectionClientToServer,
			DisableDefaultResponse: false,
		},
		TransSeqNum: seqNum,
		CommandID:   uint8(CmdConfigureReporting),
		Payload:     payload,
	}, nil
}

// ConfigureReportingResult represents a single attribute result in a configure reporting response.
type ConfigureReportingResult struct {
	Status      uint8       // 0x00 = success
	Direction   uint8       // Only present if status != 0x00
	AttributeID AttributeID // Only present if status != 0x00
}

// ParseConfigureReportingResponse parses a configure reporting response.
// Format: [status:1][direction:1 if status!=0][attrID:2 if status!=0]...
func ParseConfigureReportingResponse(payload []byte) ([]ConfigureReportingResult, error) {
	results := make([]ConfigureReportingResult, 0)
	offset := 0

	// If first status is 0x00, all configurations succeeded
	if len(payload) > 0 && payload[0] == StatusSuccess {
		results = append(results, ConfigureReportingResult{
			Status: StatusSuccess,
		})
		return results, nil
	}

	// Parse individual results
	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		result := ConfigureReportingResult{
			Status: payload[offset],
		}
		offset++

		if result.Status != StatusSuccess {
			// Read direction and attribute ID if status indicates failure
			if offset+3 > len(payload) {
				return nil, fmt.Errorf("insufficient data for failed result at offset %d", offset)
			}
			result.Direction = payload[offset]
			offset++

			result.AttributeID = AttributeID(binary.LittleEndian.Uint16(payload[offset:]))
			offset += 2
		}

		results = append(results, result)
	}

	return results, nil
}

// ReadReportingConfigRecord specifies which attribute's reporting config to read.
type ReadReportingConfigRecord struct {
	Direction   uint8 // 0x00 = reported (device sends reports), 0x01 = received
	AttributeID AttributeID
}

// BuildReadReportingConfigRequest creates a read reporting configuration request frame.
// This queries the device for its current reporting configuration for the specified attributes.
func BuildReadReportingConfigRequest(seqNum uint8, records []ReadReportingConfigRecord) *Frame {
	payload := make([]byte, 0, len(records)*3)

	for _, rec := range records {
		// Direction (1 byte)
		payload = append(payload, rec.Direction)

		// Attribute ID (2 bytes)
		attrIDBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(attrIDBytes, uint16(rec.AttributeID))
		payload = append(payload, attrIDBytes...)
	}

	return &Frame{
		Control: FrameControl{
			FrameType:              FrameTypeGlobal,
			ManufacturerSpecific:   false,
			Direction:              DirectionClientToServer,
			DisableDefaultResponse: false,
		},
		TransSeqNum: seqNum,
		CommandID:   uint8(CmdReadReportingConfig),
		Payload:     payload,
	}
}

// ReportingConfigRecord represents a single attribute's reporting configuration from the response.
type ReportingConfigRecord struct {
	Status           uint8 // 0x00 = success
	Direction        uint8 // 0x00 = reported, 0x01 = received
	AttributeID      AttributeID
	DataType         DataType    // Only present if Direction == 0x00 and Status == 0x00
	MinInterval      uint16      // Only present if Direction == 0x00 and Status == 0x00
	MaxInterval      uint16      // Only present if Direction == 0x00 and Status == 0x00
	ReportableChange interface{} // Only present for analog types, Direction == 0x00 and Status == 0x00
	TimeoutPeriod    uint16      // Only present if Direction == 0x01 and Status == 0x00
}

// ParseReadReportingConfigResponse parses a read reporting configuration response.
// Format varies based on status and direction.
func ParseReadReportingConfigResponse(payload []byte) ([]ReportingConfigRecord, error) {
	records := make([]ReportingConfigRecord, 0)
	offset := 0

	for offset < len(payload) {
		if offset+4 > len(payload) {
			return nil, fmt.Errorf("insufficient data at offset %d", offset)
		}

		record := ReportingConfigRecord{}

		// Status (1 byte)
		record.Status = payload[offset]
		offset++

		// Direction (1 byte)
		record.Direction = payload[offset]
		offset++

		// Attribute ID (2 bytes)
		record.AttributeID = AttributeID(binary.LittleEndian.Uint16(payload[offset:]))
		offset += 2

		// If status is not success, that's all we get
		if record.Status != StatusSuccess {
			records = append(records, record)
			continue
		}

		// Parse based on direction
		if record.Direction == 0x00 {
			// Reported direction: device sends reports to us
			if offset+5 > len(payload) {
				return nil, fmt.Errorf("insufficient data for reporting config at offset %d", offset)
			}

			// Data type (1 byte)
			record.DataType = DataType(payload[offset])
			offset++

			// Min interval (2 bytes)
			record.MinInterval = binary.LittleEndian.Uint16(payload[offset:])
			offset += 2

			// Max interval (2 bytes)
			record.MaxInterval = binary.LittleEndian.Uint16(payload[offset:])
			offset += 2

			// Reportable change (variable, only for analog types)
			if isAnalogType(record.DataType) {
				typeSize := dataTypeSize(record.DataType)
				if typeSize > 0 && offset+typeSize <= len(payload) {
					value, _, err := ReadValue(payload[offset:], record.DataType)
					if err == nil {
						record.ReportableChange = value
					}
					offset += typeSize
				}
			}
		} else if record.Direction == 0x01 {
			// Received direction: device receives reports from us
			if offset+2 > len(payload) {
				return nil, fmt.Errorf("insufficient data for timeout period at offset %d", offset)
			}

			// Timeout period (2 bytes)
			record.TimeoutPeriod = binary.LittleEndian.Uint16(payload[offset:])
			offset += 2
		}

		records = append(records, record)
	}

	return records, nil
}

// isAnalogType returns true if the data type is analog (has reportable change).
func isAnalogType(dt DataType) bool {
	switch dt {
	case TypeUint8, TypeUint16, TypeUint24, TypeUint32, TypeUint48,
		TypeInt8, TypeInt16, TypeInt24, TypeInt32,
		TypeSingle, TypeDouble,
		TypeTimeOfDay, TypeDate, TypeUTCTime:
		return true
	default:
		return false
	}
}

// dataTypeSize returns the size in bytes for a data type, or 0 if variable/unknown.
func dataTypeSize(dt DataType) int {
	switch dt {
	case TypeBoolean, TypeUint8, TypeInt8, TypeEnum8, TypeBitmap8:
		return 1
	case TypeUint16, TypeInt16, TypeEnum16, TypeBitmap16, TypeClusterID, TypeAttrID:
		return 2
	case TypeUint24, TypeInt24:
		return 3
	case TypeUint32, TypeInt32, TypeSingle, TypeTimeOfDay, TypeDate, TypeUTCTime, TypeBitmap32:
		return 4
	case TypeUint48:
		return 6
	case TypeDouble, TypeIEEEAddr:
		return 8
	default:
		return 0
	}
}

package zcl

import (
	"encoding/binary"
	"testing"
)

// TestBuildReadAttributesRequest tests building read attributes requests.
func TestBuildReadAttributesRequest(t *testing.T) {
	t.Run("single attribute", func(t *testing.T) {
		frame := BuildReadAttributesRequest(0x01, AttributeID(0x0000))
		data := frame.ToBytes()

		// Verify frame header (3 bytes)
		if data[0] != 0x00 { // FrameControl: global, client-to-server
			t.Errorf("wrong frame control: got 0x%02X, want 0x00", data[0])
		}
		if data[1] != 0x01 { // SeqNum
			t.Errorf("wrong seq num: got 0x%02X, want 0x01", data[1])
		}
		if data[2] != 0x00 { // Command: ReadAttributes
			t.Errorf("wrong command: got 0x%02X, want 0x00", data[2])
		}

		// Verify attribute ID (little-endian)
		if len(data) != 5 {
			t.Fatalf("wrong frame length: got %d, want 5", len(data))
		}
		if data[3] != 0x00 || data[4] != 0x00 {
			t.Errorf("wrong attr ID: got [0x%02X, 0x%02X], want [0x00, 0x00]", data[3], data[4])
		}
	})

	t.Run("multiple attributes", func(t *testing.T) {
		frame := BuildReadAttributesRequest(0x42, AttributeID(0x0000), AttributeID(0x0001), AttributeID(0x0005))
		data := frame.ToBytes()

		// Verify frame header
		if data[0] != 0x00 {
			t.Errorf("wrong frame control: got 0x%02X", data[0])
		}
		if data[1] != 0x42 {
			t.Errorf("wrong seq num: got 0x%02X, want 0x42", data[1])
		}
		if data[2] != 0x00 {
			t.Errorf("wrong command: got 0x%02X", data[2])
		}

		// Verify length (3 header bytes + 3*2 attribute bytes)
		if len(data) != 9 {
			t.Fatalf("wrong frame length: got %d, want 9", len(data))
		}

		// Verify first attribute ID: 0x0000
		if data[3] != 0x00 || data[4] != 0x00 {
			t.Errorf("wrong first attr ID: got 0x%04X", binary.LittleEndian.Uint16(data[3:]))
		}

		// Verify second attribute ID: 0x0001
		if data[5] != 0x01 || data[6] != 0x00 {
			t.Errorf("wrong second attr ID: got 0x%04X", binary.LittleEndian.Uint16(data[5:]))
		}

		// Verify third attribute ID: 0x0005
		if data[7] != 0x05 || data[8] != 0x00 {
			t.Errorf("wrong third attr ID: got 0x%04X", binary.LittleEndian.Uint16(data[7:]))
		}
	})

	t.Run("frame structure", func(t *testing.T) {
		frame := BuildReadAttributesRequest(0xFF, AttributeID(0x1234))

		// Verify frame control fields
		if frame.Control.FrameType != FrameTypeGlobal {
			t.Errorf("wrong frame type: got %d, want %d", frame.Control.FrameType, FrameTypeGlobal)
		}
		if frame.Control.Direction != DirectionClientToServer {
			t.Errorf("wrong direction: got %d, want %d", frame.Control.Direction, DirectionClientToServer)
		}
		if frame.Control.ManufacturerSpecific {
			t.Error("manufacturer specific should be false")
		}
		if frame.Control.DisableDefaultResponse {
			t.Error("disable default response should be false")
		}

		// Verify command ID
		if frame.CommandID != uint8(CmdReadAttributes) {
			t.Errorf("wrong command ID: got 0x%02X, want 0x%02X", frame.CommandID, CmdReadAttributes)
		}

		// Verify sequence number
		if frame.TransSeqNum != 0xFF {
			t.Errorf("wrong seq num: got 0x%02X, want 0xFF", frame.TransSeqNum)
		}
	})
}

// TestParseReadAttributesResponse tests parsing read attributes responses.
func TestParseReadAttributesResponse(t *testing.T) {
	t.Run("success single attribute", func(t *testing.T) {
		// Response: attrID=0x0000, status=success, type=uint8, value=42
		payload := []byte{
			0x00, 0x00, // Attribute ID: 0x0000
			0x00, // Status: success
			0x20, // Data type: uint8
			0x2A, // Value: 42
		}

		results, err := ParseReadAttributesResponse(payload)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("wrong result count: got %d, want 1", len(results))
		}

		r := results[0]
		if r.AttributeID != AttributeID(0x0000) {
			t.Errorf("wrong attr ID: got 0x%04X", r.AttributeID)
		}
		if r.Status != StatusSuccess {
			t.Errorf("wrong status: got 0x%02X", r.Status)
		}
		if r.DataType != TypeUint8 {
			t.Errorf("wrong data type: got 0x%02X", r.DataType)
		}
		if val, ok := r.Value.(uint8); !ok || val != 42 {
			t.Errorf("wrong value: got %v, want 42", r.Value)
		}
	})

	t.Run("success multiple attributes", func(t *testing.T) {
		// Two successful reads
		payload := []byte{
			0x00, 0x00, // Attribute ID: 0x0000
			0x00,       // Status: success
			0x10,       // Data type: boolean
			0x01,       // Value: true
			0x01, 0x00, // Attribute ID: 0x0001
			0x00,       // Status: success
			0x21,       // Data type: uint16
			0x34, 0x12, // Value: 0x1234 (little-endian)
		}

		results, err := ParseReadAttributesResponse(payload)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("wrong result count: got %d, want 2", len(results))
		}

		// First attribute
		if results[0].AttributeID != AttributeID(0x0000) {
			t.Errorf("wrong first attr ID: got 0x%04X", results[0].AttributeID)
		}
		if val, ok := results[0].Value.(bool); !ok || !val {
			t.Errorf("wrong first value: got %v, want true", results[0].Value)
		}

		// Second attribute
		if results[1].AttributeID != AttributeID(0x0001) {
			t.Errorf("wrong second attr ID: got 0x%04X", results[1].AttributeID)
		}
		if val, ok := results[1].Value.(uint16); !ok || val != 0x1234 {
			t.Errorf("wrong second value: got %v, want 0x1234", results[1].Value)
		}
	})

	t.Run("failure response", func(t *testing.T) {
		// Response: attrID=0x0005, status=unsupported (no data type/value)
		payload := []byte{
			0x05, 0x00, // Attribute ID: 0x0005
			0x86, // Status: unsupported attribute
		}

		results, err := ParseReadAttributesResponse(payload)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("wrong result count: got %d, want 1", len(results))
		}

		r := results[0]
		if r.AttributeID != AttributeID(0x0005) {
			t.Errorf("wrong attr ID: got 0x%04X", r.AttributeID)
		}
		if r.Status != StatusUnsupportedAttr {
			t.Errorf("wrong status: got 0x%02X, want 0x86", r.Status)
		}
		if r.Value != nil {
			t.Errorf("value should be nil for failed read, got %v", r.Value)
		}
	})

	t.Run("mixed success and failure", func(t *testing.T) {
		payload := []byte{
			0x00, 0x00, // Attribute ID: 0x0000
			0x00,       // Status: success
			0x20,       // Data type: uint8
			0xFF,       // Value: 255
			0x01, 0x00, // Attribute ID: 0x0001
			0x86,       // Status: unsupported
			0x02, 0x00, // Attribute ID: 0x0002
			0x00, // Status: success
			0x10, // Data type: boolean
			0x00, // Value: false
		}

		results, err := ParseReadAttributesResponse(payload)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("wrong result count: got %d, want 3", len(results))
		}

		// First: success
		if results[0].Status != StatusSuccess {
			t.Errorf("first status should be success")
		}
		if val, ok := results[0].Value.(uint8); !ok || val != 255 {
			t.Errorf("first value wrong: got %v", results[0].Value)
		}

		// Second: failure
		if results[1].Status != StatusUnsupportedAttr {
			t.Errorf("second status should be unsupported")
		}
		if results[1].Value != nil {
			t.Errorf("second value should be nil")
		}

		// Third: success
		if results[2].Status != StatusSuccess {
			t.Errorf("third status should be success")
		}
		if val, ok := results[2].Value.(bool); !ok || val {
			t.Errorf("third value wrong: got %v, want false", results[2].Value)
		}
	})

	t.Run("insufficient data", func(t *testing.T) {
		// Too short for header
		payload := []byte{0x00, 0x00}
		_, err := ParseReadAttributesResponse(payload)
		if err == nil {
			t.Error("should error on insufficient data")
		}
	})
}

// TestBuildWriteAttributesRequest tests building write attributes requests.
func TestBuildWriteAttributesRequest(t *testing.T) {
	t.Run("single uint8 attribute", func(t *testing.T) {
		values := map[AttributeID]AttributeValue{
			AttributeID(0x0000): {Type: TypeUint8, Value: uint8(100)},
		}

		frame, err := BuildWriteAttributesRequest(0x10, values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := frame.ToBytes()

		// Verify frame header
		if data[0] != 0x00 {
			t.Errorf("wrong frame control: got 0x%02X", data[0])
		}
		if data[1] != 0x10 {
			t.Errorf("wrong seq num: got 0x%02X, want 0x10", data[1])
		}
		if data[2] != 0x02 { // CmdWriteAttributes
			t.Errorf("wrong command: got 0x%02X, want 0x02", data[2])
		}

		// Verify payload: attrID(2) + type(1) + value(1) = 4 bytes
		if len(data) != 7 {
			t.Fatalf("wrong frame length: got %d, want 7", len(data))
		}

		// Attribute ID: 0x0000
		attrID := binary.LittleEndian.Uint16(data[3:5])
		if attrID != 0x0000 {
			t.Errorf("wrong attr ID: got 0x%04X", attrID)
		}

		// Data type: uint8
		if data[5] != uint8(TypeUint8) {
			t.Errorf("wrong data type: got 0x%02X, want 0x20", data[5])
		}

		// Value: 100
		if data[6] != 100 {
			t.Errorf("wrong value: got %d, want 100", data[6])
		}
	})

	t.Run("multiple attributes different types", func(t *testing.T) {
		values := map[AttributeID]AttributeValue{
			AttributeID(0x0000): {Type: TypeBoolean, Value: true},
			AttributeID(0x0001): {Type: TypeUint16, Value: uint16(0x1234)},
		}

		frame, err := BuildWriteAttributesRequest(0x20, values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := frame.ToBytes()

		// Verify frame header
		if data[1] != 0x20 {
			t.Errorf("wrong seq num: got 0x%02X", data[1])
		}
		if data[2] != uint8(CmdWriteAttributes) {
			t.Errorf("wrong command: got 0x%02X", data[2])
		}

		// Expected payload size: 2 attributes
		// Attr1: attrID(2) + type(1) + value(1) = 4 bytes
		// Attr2: attrID(2) + type(1) + value(2) = 5 bytes
		// Total: 3 (header) + 4 + 5 = 12 bytes
		if len(data) != 12 {
			t.Fatalf("wrong frame length: got %d, want 12", len(data))
		}

		// Note: map iteration order is random, so we need to parse both attributes
		// and check they're both present
		foundBoolean := false
		foundUint16 := false

		offset := 3
		for offset < len(data) {
			attrID := AttributeID(binary.LittleEndian.Uint16(data[offset : offset+2]))
			dataType := DataType(data[offset+2])

			switch {
			case attrID == AttributeID(0x0000) && dataType == TypeBoolean:
				if data[offset+3] != 1 {
					t.Errorf("wrong boolean value: got %d, want 1", data[offset+3])
				}
				foundBoolean = true
				offset += 4
			case attrID == AttributeID(0x0001) && dataType == TypeUint16:
				val := binary.LittleEndian.Uint16(data[offset+3 : offset+5])
				if val != 0x1234 {
					t.Errorf("wrong uint16 value: got 0x%04X, want 0x1234", val)
				}
				foundUint16 = true
				offset += 5
			default:
				t.Errorf("unexpected attribute: ID=0x%04X, type=0x%02X", attrID, dataType)
			}
		}

		if !foundBoolean {
			t.Error("boolean attribute not found in payload")
		}
		if !foundUint16 {
			t.Error("uint16 attribute not found in payload")
		}
	})

	t.Run("string attribute", func(t *testing.T) {
		values := map[AttributeID]AttributeValue{
			AttributeID(0x0010): {Type: TypeCharString, Value: "test"},
		}

		frame, err := BuildWriteAttributesRequest(0x30, values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := frame.ToBytes()

		// Verify command
		if data[2] != uint8(CmdWriteAttributes) {
			t.Errorf("wrong command: got 0x%02X", data[2])
		}

		// Expected: header(3) + attrID(2) + type(1) + length(1) + string(4) = 11 bytes
		if len(data) != 11 {
			t.Fatalf("wrong frame length: got %d, want 11", len(data))
		}

		// Verify attribute ID
		attrID := binary.LittleEndian.Uint16(data[3:5])
		if attrID != 0x0010 {
			t.Errorf("wrong attr ID: got 0x%04X", attrID)
		}

		// Verify data type
		if data[5] != uint8(TypeCharString) {
			t.Errorf("wrong data type: got 0x%02X", data[5])
		}

		// Verify string length
		if data[6] != 4 {
			t.Errorf("wrong string length: got %d, want 4", data[6])
		}

		// Verify string content
		str := string(data[7:11])
		if str != "test" {
			t.Errorf("wrong string value: got %q, want %q", str, "test")
		}
	})

	t.Run("frame structure", func(t *testing.T) {
		values := map[AttributeID]AttributeValue{
			AttributeID(0x0000): {Type: TypeUint8, Value: uint8(1)},
		}

		frame, err := BuildWriteAttributesRequest(0x99, values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify frame control fields
		if frame.Control.FrameType != FrameTypeGlobal {
			t.Errorf("wrong frame type")
		}
		if frame.Control.Direction != DirectionClientToServer {
			t.Errorf("wrong direction")
		}
		if frame.Control.ManufacturerSpecific {
			t.Error("manufacturer specific should be false")
		}

		// Verify command ID
		if frame.CommandID != uint8(CmdWriteAttributes) {
			t.Errorf("wrong command ID: got 0x%02X", frame.CommandID)
		}

		// Verify sequence number
		if frame.TransSeqNum != 0x99 {
			t.Errorf("wrong seq num: got 0x%02X", frame.TransSeqNum)
		}
	})

	t.Run("error on invalid value type", func(t *testing.T) {
		values := map[AttributeID]AttributeValue{
			AttributeID(0x0000): {Type: TypeUint8, Value: struct{}{}},
		}

		_, err := BuildWriteAttributesRequest(0x01, values)
		if err == nil {
			t.Error("expected error for invalid value type, got nil")
		}
	})
}

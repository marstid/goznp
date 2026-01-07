// Package zcl provides Zigbee Cluster Library primitives for encoding/decoding
// ZCL frames and working with standard clusters.
//
// # ZCL Frame Structure
//
// A ZCL frame has the following structure:
//
//	Frame Control (1 byte)
//	- Frame Type (2 bits): global (0x00), cluster-specific (0x01)
//	- Manufacturer Specific (1 bit)
//	- Direction (1 bit): client→server (0), server→client (1)
//	- Disable Default Response (1 bit)
//	- Reserved (3 bits)
//
//	Transaction Sequence Number (1 byte)
//
//	Command (1 byte): varies by frame type
//
//	Data (variable): command-specific payload
//
// # Creating Frames
//
// Create a read attributes request:
//
//	frame := zcl.Frame{
//	    FrameControl: zcl.FrameTypeGlobal,
//	    Direction:    zcl.ClientToServer,
//	    TransSeqNum:  1,
//	    Command:      zcl.CmdReadAttributes,
//	    Data: attributes,
//	}
//
// Encode to bytes:
//
//	data, err := zcl.Encode(frame)
//
// # Data Types
//
// ZCL defines various data types for attribute values:
//
//	Boolean:      zcl.TypeData8, zcl.TypeBoolean
//	Integer:      zcl.TypeData8, TypeData16, TypeData24, TypeData32, TypeData40, TypeData48
//	Float:        zcl.TypeSingle, TypeDouble, TypeHalf
//	Strings:      zcl.TypeString, TypeOctetString
//	Arrays:       zcl.TypeArray, TypeArray16
//	Special:      zcl.TypeBitmap8, zcl.TypeEnumeration8, zcl.TypeUTC
//
// Encoding a value:
//
//	encoded, err := zcl.WriteValue(uint16(1234))
//
// Decoding a value:
//
//	decoded, remaining, err := zcl.ReadValue(data, zcl.TypeData16)
//
// # Clusters
//
// Standard Zigbee clusters are defined as constants in clusters.go:
//
//	Basic:              0x0000
//	Power Configuration: 0x0001
//	Identify:           0x0003
//	Groups:             0x0004
//	Scenes:             0x0005
//	On/Off:             0x0006
//	Level Control:      0x0008
//	Color Control:      0x0300
//	Temperature:        0x0402
//	Humidity:           0x0405
//	Occupancy:          0x0406
//	Electrical Measure: 0x0B04
//	Simple Metering:    0x0702
//
// # Attributes
//
// Each cluster has standard attributes. Read attributes to query device state:
//
//	results, err := adapter.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, []zcl.AttributeID{
//	    zcl.AttrOnOff,
//	})
//
// # Global Commands
//
// Work with any cluster:
//
//	Read Attributes:               zcl.CmdReadAttributes
//	Read Attributes Response:      zcl.CmdReadAttributesResponse
//	Write Attributes:              zcl.CmdWriteAttributes
//	Write Attributes Response:     zcl.CmdWriteAttributesResponse
//	Configure Reporting:          zcl.CmdConfigureReporting
//	Configure Reporting Response: zcl.CmdConfigureReportingResponse
//	Discover Attributes:          zcl.CmdDiscoverAttributes
//
// # Color Control
//
// Utilities for color conversion:
//
//	Kelvin to mireds:
//		mireds := zcl.KelvinToMireds(2700)
//
//	mireds to Kelvin:
//		kelvin := zcl.MiredsToKelvin(370)
//
//	RGB to XY:
//		x, y := zcl.RGBToXY(255, 0, 0)
//
//	HSV to Hue/Saturation (0-254):
//		hue, sat := zcl.HSVToZigbee(120, 100)
//
// # Status Codes
//
// ZCL operations return status codes:
//
//	Success:        0x00
//	UnsupportedAttr: 0x86
//	InvalidValue:     0x87
//	ReadOnly:         0x82
//
// # Manufacturer-Specific Clusters
//
// Set manufacturer code in frame control:
//
//	frame.FrameControl.ManufacturerSpecific = true
//	frame.ManufacturerCode = 0x115F // Tuya
//
// # Thread Safety
//
// Frame encoding/decoding functions are stateless and safe for concurrent use.
package zcl

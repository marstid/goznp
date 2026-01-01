package unpi

// Protocol constants for UNPI frame structure.
const (
	// SOF is the Start of Frame delimiter byte.
	SOF byte = 0xFE

	// Frame position constants indicate byte offsets in the UNPI frame.
	PositionDataLength = 1 // Position of the data length field
	PositionCmd0       = 2 // Position of the first command byte
	PositionCmd1       = 3 // Position of the second command byte
	DataStart          = 4 // Position where data payload begins

	// MinMessageLength is the minimum valid frame size (SOF + Len + Cmd0 + Cmd1 + FCS).
	MinMessageLength = 5

	// MaxDataSize is the maximum payload size allowed in a frame.
	MaxDataSize = 250
)

// Type represents the UNPI message type field.
type Type uint8

const (
	// POLL represents a synchronous request that expects no response data.
	POLL Type = 0
	// SREQ represents a synchronous request that expects a response.
	SREQ Type = 1
	// AREQ represents an asynchronous request (no response expected).
	AREQ Type = 2
	// SRSP represents a synchronous response to an SREQ.
	SRSP Type = 3
)

// String returns the string representation of the Type.
func (t Type) String() string {
	switch t {
	case POLL:
		return "POLL"
	case SREQ:
		return "SREQ"
	case AREQ:
		return "AREQ"
	case SRSP:
		return "SRSP"
	default:
		return "UNKNOWN"
	}
}

// Subsystem represents the UNPI subsystem identifier.
type Subsystem uint8

const (
	// RESERVED is a reserved subsystem value.
	RESERVED Subsystem = 0
	// SYS is the system subsystem.
	SYS Subsystem = 1
	// MAC is the MAC layer subsystem.
	MAC Subsystem = 2
	// NWK is the network layer subsystem.
	NWK Subsystem = 3
	// AF is the Application Framework subsystem.
	AF Subsystem = 4
	// ZDO is the Zigbee Device Object subsystem.
	ZDO Subsystem = 5
	// SAPI is the Simple API subsystem.
	SAPI Subsystem = 6
	// UTIL is the utility subsystem.
	UTIL Subsystem = 7
	// DEBUG is the debug subsystem.
	DEBUG Subsystem = 8
	// APP is the application subsystem.
	APP Subsystem = 9
	// APPCNF is the application configuration subsystem.
	APPCNF Subsystem = 15
	// GREENPOWER is the Green Power subsystem.
	GREENPOWER Subsystem = 21
)

// String returns the string representation of the Subsystem.
func (s Subsystem) String() string {
	switch s {
	case RESERVED:
		return "RESERVED"
	case SYS:
		return "SYS"
	case MAC:
		return "MAC"
	case NWK:
		return "NWK"
	case AF:
		return "AF"
	case ZDO:
		return "ZDO"
	case SAPI:
		return "SAPI"
	case UTIL:
		return "UTIL"
	case DEBUG:
		return "DEBUG"
	case APP:
		return "APP"
	case APPCNF:
		return "APPCNF"
	case GREENPOWER:
		return "GREENPOWER"
	default:
		return "UNKNOWN"
	}
}

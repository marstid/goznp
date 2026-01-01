package znp

import (
	"fmt"

	"github.com/marstid/goznp/pkg/unpi"
)

// ParameterType defines parameter types for encoding.
type ParameterType uint8

const (
	// ParamUint8 represents an unsigned 8-bit integer parameter.
	ParamUint8 ParameterType = iota
	// ParamUint16 represents an unsigned 16-bit integer parameter.
	ParamUint16
	// ParamUint32 represents an unsigned 32-bit integer parameter.
	ParamUint32
	// ParamInt8 represents a signed 8-bit integer parameter.
	ParamInt8
	// ParamIEEEAddr represents an 8-byte IEEE address parameter.
	ParamIEEEAddr
	// ParamBuffer represents a variable-length byte buffer parameter.
	ParamBuffer
)

// Parameter defines a command parameter with its name and type.
type Parameter struct {
	// Name is the parameter identifier.
	Name string
	// Type specifies the parameter's data type.
	Type ParameterType
}

// Command defines a ZNP command structure.
type Command struct {
	// Name is the command identifier.
	Name string
	// ID is the command ID byte.
	ID uint8
	// Type is the UNPI message type (SREQ, AREQ, etc.).
	Type unpi.Type
	// Request defines the parameters for command requests.
	Request []Parameter
	// Response defines the parameters for command responses.
	Response []Parameter
}

// ResetType defines the type of reset to perform.
type ResetType uint8

const (
	// ResetTypeHard performs a hardware reset.
	ResetTypeHard ResetType = 0
	// ResetTypeSoft performs a software reset.
	ResetTypeSoft ResetType = 1
)

// SYS subsystem commands for Phase 1.
var (
	// CmdSysPing sends a ping request to verify communication.
	CmdSysPing = Command{
		Name: "ping",
		ID:   0x01,
		Type: unpi.SREQ,
		Response: []Parameter{
			{Name: "capabilities", Type: ParamUint16},
		},
	}

	// CmdSysVersion requests the firmware version information.
	CmdSysVersion = Command{
		Name: "version",
		ID:   0x02,
		Type: unpi.SREQ,
		Response: []Parameter{
			{Name: "transportRev", Type: ParamUint8},
			{Name: "product", Type: ParamUint8},
			{Name: "majorRel", Type: ParamUint8},
			{Name: "minorRel", Type: ParamUint8},
			{Name: "maintRel", Type: ParamUint8},
			{Name: "revision", Type: ParamUint32},
		},
	}

	// CmdSysResetReq requests a system reset.
	CmdSysResetReq = Command{
		Name: "resetReq",
		ID:   0x00,
		Type: unpi.AREQ,
		Request: []Parameter{
			{Name: "type", Type: ParamUint8},
		},
	}

	// CmdSysResetInd is the asynchronous reset indication.
	CmdSysResetInd = Command{
		Name: "resetInd",
		ID:   0x80,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "reason", Type: ParamUint8},
			{Name: "transportRev", Type: ParamUint8},
			{Name: "product", Type: ParamUint8},
			{Name: "majorRel", Type: ParamUint8},
			{Name: "minorRel", Type: ParamUint8},
			{Name: "maintRel", Type: ParamUint8},
		},
	}

	// CmdSysOsalNvItemInit initializes/creates an NV item (legacy).
	// If the item already exists, this is a no-op.
	CmdSysOsalNvItemInit = Command{
		Name: "osalNvItemInit",
		ID:   0x07,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "id", Type: ParamUint16},
			{Name: "itemLen", Type: ParamUint16},
			{Name: "initLen", Type: ParamUint8},
			{Name: "initData", Type: ParamBuffer},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdSysOsalNvRead reads from NV memory (legacy).
	CmdSysOsalNvRead = Command{
		Name: "osalNvRead",
		ID:   0x08,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "id", Type: ParamUint16},
			{Name: "offset", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
			{Name: "len", Type: ParamUint8},
			{Name: "value", Type: ParamBuffer},
		},
	}

	// CmdSysOsalNvWrite writes to NV memory (legacy).
	CmdSysOsalNvWrite = Command{
		Name: "osalNvWrite",
		ID:   0x09,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "id", Type: ParamUint16},
			{Name: "offset", Type: ParamUint8},
			{Name: "len", Type: ParamUint8},
			{Name: "value", Type: ParamBuffer},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdSysOsalNvLength gets the length of an NV item (legacy).
	CmdSysOsalNvLength = Command{
		Name: "osalNvLength",
		ID:   0x13,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "id", Type: ParamUint16},
		},
		Response: []Parameter{
			{Name: "len", Type: ParamUint16},
		},
	}

	// CmdSysNvLength gets the length of an NV item (extended).
	CmdSysNvLength = Command{
		Name: "nvLength",
		ID:   0x32,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "sysId", Type: ParamUint8},
			{Name: "itemId", Type: ParamUint16},
			{Name: "subId", Type: ParamUint16},
		},
		Response: []Parameter{
			{Name: "len", Type: ParamUint8},
		},
	}

	// CmdSysNvRead reads from NV memory (extended).
	CmdSysNvRead = Command{
		Name: "nvRead",
		ID:   0x33,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "sysId", Type: ParamUint8},
			{Name: "itemId", Type: ParamUint16},
			{Name: "subId", Type: ParamUint16},
			{Name: "offset", Type: ParamUint16},
			{Name: "len", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
			{Name: "len", Type: ParamUint8},
			{Name: "value", Type: ParamBuffer},
		},
	}

	// CmdSysNvWrite writes to NV memory (extended).
	CmdSysNvWrite = Command{
		Name: "nvWrite",
		ID:   0x34,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "sysId", Type: ParamUint8},
			{Name: "itemId", Type: ParamUint16},
			{Name: "subId", Type: ParamUint16},
			{Name: "offset", Type: ParamUint16},
			{Name: "len", Type: ParamUint8},
			{Name: "value", Type: ParamBuffer},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdUtilGetDeviceInfo retrieves device information including IEEE address.
	CmdUtilGetDeviceInfo = Command{
		Name: "getDeviceInfo",
		ID:   0x00,
		Type: unpi.SREQ,
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
			{Name: "ieeeAddr", Type: ParamIEEEAddr},
			{Name: "shortAddr", Type: ParamUint16},
			{Name: "deviceType", Type: ParamUint8},
			{Name: "deviceState", Type: ParamUint8},
			{Name: "numAssocDevices", Type: ParamUint8},
			// assocDevicesList is variable length, handled separately
		},
	}

	// CmdUtilAssocCount returns the count of associated devices.
	CmdUtilAssocCount = Command{
		Name: "assocCount",
		ID:   0x48,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "startRelation", Type: ParamUint8},
			{Name: "startIndex", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "cnt", Type: ParamUint16},
		},
	}

	// CmdUtilAssocFindDevice finds a device by index in the association table.
	CmdUtilAssocFindDevice = Command{
		Name: "assocFindDevice",
		ID:   0x49,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "number", Type: ParamUint8},
		},
		// Response is a complex struct, handled in util.go
	}

	// CmdZdoExtNwkInfo retrieves extended network information.
	CmdZdoExtNwkInfo = Command{
		Name: "extNwkInfo",
		ID:   0x50,
		Type: unpi.SREQ,
		Response: []Parameter{
			{Name: "shortAddr", Type: ParamUint16},
			{Name: "devState", Type: ParamUint8},
			{Name: "panId", Type: ParamUint16},
			{Name: "parentAddr", Type: ParamUint16},
			{Name: "extendedPanId", Type: ParamIEEEAddr},
			{Name: "parentExtAddr", Type: ParamIEEEAddr},
			{Name: "channel", Type: ParamUint8},
		},
	}

	// CmdSysSetTxPower sets the TX power level (legacy, may not work on all firmware).
	CmdSysSetTxPower = Command{
		Name: "setTxPower",
		ID:   0x14,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "txPower", Type: ParamInt8},
		},
		Response: []Parameter{
			{Name: "txPower", Type: ParamInt8},
		},
	}

	// CmdSysStackTune tunes stack parameters including TX power.
	// This is the preferred method for setting TX power on Z-Stack 3.0.
	// Operation 0 = set TX power, value = power in dBm.
	CmdSysStackTune = Command{
		Name: "stackTune",
		ID:   0x0F,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "operation", Type: ParamUint8},
			{Name: "value", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "value", Type: ParamUint8},
		},
	}

	// CmdZdoStartupFromApp starts the network from application.
	CmdZdoStartupFromApp = Command{
		Name: "startupFromApp",
		ID:   0x40,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "startDelay", Type: ParamUint16},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoStateChangeInd is the device state change indication.
	CmdZdoStateChangeInd = Command{
		Name: "stateChangeInd",
		ID:   0xC0,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "state", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtNwkUpdateReq requests a network update (channel change).
	CmdZdoMgmtNwkUpdateReq = Command{
		Name: "mgmtNwkUpdateReq",
		ID:   0x37,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "dstAddrMode", Type: ParamUint8},
			{Name: "channelMask", Type: ParamUint32},
			{Name: "scanDuration", Type: ParamUint8},
			{Name: "scanCount", Type: ParamUint8},
			{Name: "nwkManagerAddr", Type: ParamUint16},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtPermitJoinReq opens/closes network for device joining.
	CmdZdoMgmtPermitJoinReq = Command{
		Name: "mgmtPermitJoinReq",
		ID:   0x36,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "addrMode", Type: ParamUint8},
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "duration", Type: ParamUint8},
			{Name: "tcSignificance", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoEndDeviceAnnceInd is device join notification (AREQ).
	CmdZdoEndDeviceAnnceInd = Command{
		Name: "endDeviceAnnceInd",
		ID:   0xC1,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "srcAddr", Type: ParamUint16},
			{Name: "nwkAddr", Type: ParamUint16},
			{Name: "ieeeAddr", Type: ParamIEEEAddr},
			{Name: "capabilities", Type: ParamUint8},
		},
	}

	// CmdZdoTcDevInd is Trust Center device indication (AREQ).
	CmdZdoTcDevInd = Command{
		Name: "tcDevInd",
		ID:   0xCA,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "nwkAddr", Type: ParamUint16},
			{Name: "ieeeAddr", Type: ParamIEEEAddr},
			{Name: "parentAddr", Type: ParamUint16},
		},
	}

	// CmdZdoLeaveInd is device leave indication (AREQ).
	CmdZdoLeaveInd = Command{
		Name: "leaveInd",
		ID:   0xC9,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "srcAddr", Type: ParamUint16},
			{Name: "ieeeAddr", Type: ParamIEEEAddr},
			{Name: "request", Type: ParamUint8},
			{Name: "remove", Type: ParamUint8},
			{Name: "rejoin", Type: ParamUint8},
		},
	}

	// CmdZdoIeeeAddrReq requests IEEE address from a device by network address.
	CmdZdoIeeeAddrReq = Command{
		Name: "ieeeAddrReq",
		ID:   0x01,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "shortAddr", Type: ParamUint16},
			{Name: "reqType", Type: ParamUint8},
			{Name: "startIndex", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoIeeeAddrRsp is IEEE address response (AREQ).
	CmdZdoIeeeAddrRsp = Command{
		Name: "ieeeAddrRsp",
		ID:   0x81,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
			{Name: "ieeeAddr", Type: ParamIEEEAddr},
			{Name: "nwkAddr", Type: ParamUint16},
			{Name: "startIndex", Type: ParamUint8},
			{Name: "numAssocDev", Type: ParamUint8},
		},
	}

	// CmdZdoActiveEpReq requests active endpoints from a device.
	CmdZdoActiveEpReq = Command{
		Name: "activeEpReq",
		ID:   0x05,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "nwkAddrOfInterest", Type: ParamUint16},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoActiveEpRsp is active endpoints response (AREQ).
	CmdZdoActiveEpRsp = Command{
		Name: "activeEpRsp",
		ID:   0x85,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "srcAddr", Type: ParamUint16},
			{Name: "status", Type: ParamUint8},
			{Name: "nwkAddr", Type: ParamUint16},
			{Name: "activeEpCount", Type: ParamUint8},
			{Name: "activeEpList", Type: ParamBuffer},
		},
	}

	// CmdZdoNodeDescReq requests node descriptor from a device.
	CmdZdoNodeDescReq = Command{
		Name: "nodeDescReq",
		ID:   0x02,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "nwkAddrOfInterest", Type: ParamUint16},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoNodeDescRsp is node descriptor response (AREQ).
	CmdZdoNodeDescRsp = Command{
		Name: "nodeDescRsp",
		ID:   0x82,
		Type: unpi.AREQ,
		// Response is variable length, handled in code
	}

	// CmdZdoSimpleDescReq requests simple descriptor from a device endpoint.
	CmdZdoSimpleDescReq = Command{
		Name: "simpleDescReq",
		ID:   0x04,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "nwkAddrOfInterest", Type: ParamUint16},
			{Name: "endpoint", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoSimpleDescRsp is simple descriptor response (AREQ).
	CmdZdoSimpleDescRsp = Command{
		Name: "simpleDescRsp",
		ID:   0x84,
		Type: unpi.AREQ,
		// Response is variable length, handled in code
	}

	// CmdZdoMgmtLqiReq requests neighbor table (LQI) from a device.
	// This is used to discover mesh network topology.
	CmdZdoMgmtLqiReq = Command{
		Name: "mgmtLqiReq",
		ID:   0x31,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "startIndex", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtLqiRsp is neighbor table response (AREQ).
	CmdZdoMgmtLqiRsp = Command{
		Name: "mgmtLqiRsp",
		ID:   0xB1,
		Type: unpi.AREQ,
		// Response is variable length, handled in code
	}

	// CmdZdoMgmtLeaveReq requests a device to leave the network.
	CmdZdoMgmtLeaveReq = Command{
		Name: "mgmtLeaveReq",
		ID:   0x34,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "ieeeAddr", Type: ParamIEEEAddr},
			{Name: "options", Type: ParamUint8}, // bit0=removeChildren, bit1=rejoin
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtLeaveRsp is the management leave response (AREQ).
	CmdZdoMgmtLeaveRsp = Command{
		Name: "mgmtLeaveRsp",
		ID:   0xB4,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "srcAddr", Type: ParamUint16},
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtRtgReq requests routing table from a device.
	CmdZdoMgmtRtgReq = Command{
		Name: "mgmtRtgReq",
		ID:   0x32,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "startIndex", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtRtgRsp is routing table response (AREQ).
	CmdZdoMgmtRtgRsp = Command{
		Name: "mgmtRtgRsp",
		ID:   0xB2,
		Type: unpi.AREQ,
		// Response is variable length, handled in code
	}

	// CmdZdoMgmtBindReq requests binding table from a device.
	CmdZdoMgmtBindReq = Command{
		Name: "mgmtBindReq",
		ID:   0x33,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "startIndex", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoMgmtBindRsp is binding table response (AREQ).
	CmdZdoMgmtBindRsp = Command{
		Name: "mgmtBindRsp",
		ID:   0xB3,
		Type: unpi.AREQ,
		// Response is variable length, handled in code
	}

	// CmdZdoBindReq creates a binding entry on a device.
	// Binding tells a device where to send reports (e.g., to coordinator).
	CmdZdoBindReq = Command{
		Name: "bindReq",
		ID:   0x21,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "srcAddr", Type: ParamIEEEAddr},
			{Name: "srcEndpoint", Type: ParamUint8},
			{Name: "clusterId", Type: ParamUint16},
			{Name: "dstAddrMode", Type: ParamUint8},
			{Name: "dstAddr", Type: ParamIEEEAddr},  // Or uint16 if mode is 0x01 (group)
			{Name: "dstEndpoint", Type: ParamUint8}, // Only if dstAddrMode is 0x03
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoBindRsp is bind response (AREQ).
	CmdZdoBindRsp = Command{
		Name: "bindRsp",
		ID:   0xA1,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "srcAddr", Type: ParamUint16},
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoUnbindReq removes a binding entry from a device.
	CmdZdoUnbindReq = Command{
		Name: "unbindReq",
		ID:   0x22,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "dstAddr", Type: ParamUint16},
			{Name: "srcAddr", Type: ParamIEEEAddr},
			{Name: "srcEndpoint", Type: ParamUint8},
			{Name: "clusterId", Type: ParamUint16},
			{Name: "dstAddrMode", Type: ParamUint8},
			{Name: "dstAddr", Type: ParamIEEEAddr},  // Or uint16 if mode is 0x01 (group)
			{Name: "dstEndpoint", Type: ParamUint8}, // Only if dstAddrMode is 0x03
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdZdoUnbindRsp is unbind response (AREQ).
	CmdZdoUnbindRsp = Command{
		Name: "unbindRsp",
		ID:   0xA2,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "srcAddr", Type: ParamUint16},
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdUtilSetChannels sets the channel mask.
	CmdUtilSetChannels = Command{
		Name: "setChannels",
		ID:   0x03,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "channels", Type: ParamUint32},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAppCnfBdbStartCommissioning starts BDB commissioning.
	CmdAppCnfBdbStartCommissioning = Command{
		Name: "bdbStartCommissioning",
		ID:   0x05,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "mode", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAppCnfBdbSetChannel sets the BDB channel.
	CmdAppCnfBdbSetChannel = Command{
		Name: "bdbSetChannel",
		ID:   0x08,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "isPrimary", Type: ParamUint8},
			{Name: "channel", Type: ParamUint32},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAfRegister registers an endpoint on the device.
	CmdAfRegister = Command{
		Name: "register",
		ID:   0x00,
		Type: unpi.SREQ,
		// Request parameters handled dynamically due to variable cluster lists
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAfDelete unregisters/deletes an endpoint.
	CmdAfDelete = Command{
		Name: "delete",
		ID:   0x04,
		Type: unpi.SREQ,
		Request: []Parameter{
			{Name: "endpoint", Type: ParamUint8},
		},
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAfDataRequest sends data to a device.
	CmdAfDataRequest = Command{
		Name: "dataRequest",
		ID:   0x01,
		Type: unpi.SREQ,
		// Request parameters handled dynamically
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAfDataRequestExt sends extended data request (for larger payloads).
	CmdAfDataRequestExt = Command{
		Name: "dataRequestExt",
		ID:   0x02,
		Type: unpi.SREQ,
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
		},
	}

	// CmdAfDataConfirm is async confirmation of data send.
	CmdAfDataConfirm = Command{
		Name: "dataConfirm",
		ID:   0x80,
		Type: unpi.AREQ,
		Response: []Parameter{
			{Name: "status", Type: ParamUint8},
			{Name: "endpoint", Type: ParamUint8},
			{Name: "transId", Type: ParamUint8},
		},
	}

	// CmdAfIncomingMsg is async incoming message.
	CmdAfIncomingMsg = Command{
		Name: "incomingMsg",
		ID:   0x81,
		Type: unpi.AREQ,
		// Response handled dynamically due to variable data length
	}

	// CmdAfIncomingMsgExt is extended async incoming message.
	CmdAfIncomingMsgExt = Command{
		Name: "incomingMsgExt",
		ID:   0x82,
		Type: unpi.AREQ,
	}
)

// NvItemID represents NV memory item identifiers.
type NvItemID uint16

const (
	// NvExtAddr is the coordinator IEEE address.
	NvExtAddr NvItemID = 1
	// NvStartupOption controls device startup behavior.
	// 0x00 = normal startup, 0x03 = clear state and config.
	NvStartupOption NvItemID = 3
	// NvNIB is the Network Information Base.
	NvNIB NvItemID = 33
	// NvAddrMgr is the address manager table.
	NvAddrMgr NvItemID = 35
	// NvNwkActiveKeyInfo is the active network key.
	NvNwkActiveKeyInfo NvItemID = 58
	// NvNwkAlternKeyInfo is the alternate network key.
	NvNwkAlternKeyInfo NvItemID = 59
	// NvApsLinkKeyTable is the security manager table.
	NvApsLinkKeyTable NvItemID = 76
	// NvNwkSecMaterialTableStart is the network security material.
	NvNwkSecMaterialTableStart NvItemID = 117
	// NvTclkSeed is the Trust center link key seed.
	NvTclkSeed NvItemID = 257
	// NvApsLinkKeyDataStart is the APS link key data start.
	NvApsLinkKeyDataStart NvItemID = 513
	// NvApsLinkKeyDataEnd is the APS link key data end.
	NvApsLinkKeyDataEnd NvItemID = 767
	// NvPrecfgKey is the pre-configured network key.
	NvPrecfgKey NvItemID = 46
	// NvPrecfgKeysEnable enables pre-configured keys.
	NvPrecfgKeysEnable NvItemID = 47
	// NvLogicalType is the device logical type (0=Coordinator).
	NvLogicalType NvItemID = 55
	// NvZdoDirectCb enables ZDO callbacks.
	NvZdoDirectCb NvItemID = 57
	// NvTransmitPower stores the TX power level (int8 dBm).
	// Used by Koenkk firmware for persistent TX power storage.
	NvTransmitPower NvItemID = 0x0010
	// NvZStackProfile is a custom NV item for storing the application profile.
	// Uses user data range (0x0F00+). Stores 2-byte profile ID.
	NvZStackProfile NvItemID = 0x0F00
)

// ApplicationProfile represents a Zigbee application profile.
type ApplicationProfile uint16

const (
	// ProfileUnknown indicates profile is not set or unknown.
	ProfileUnknown ApplicationProfile = 0x0000
	// ProfileHomeAutomation is the Home Automation profile (0x0104).
	ProfileHomeAutomation ApplicationProfile = 0x0104
	// ProfileSmartEnergy is the Smart Energy profile (0x0109).
	ProfileSmartEnergy ApplicationProfile = 0x0109
	// ProfileGreenPower is the Green Power profile (0xA1E0).
	ProfileGreenPower ApplicationProfile = 0xA1E0
	// ProfileLightLink is the Light Link / Zigbee 3.0 profile (0xC05E).
	ProfileLightLink ApplicationProfile = 0xC05E
)

// String returns the human-readable profile name.
func (p ApplicationProfile) String() string {
	switch uint16(p) {
	case 0x0104:
		return "Home Automation (HA)"
	case 0x0109:
		return "Smart Energy (SE)"
	case 0xA1E0:
		return "Green Power (GP)"
	case 0xC05E:
		return "Light Link / Zigbee 3.0 (ZLL)"
	case 0x0000:
		return "Unknown"
	default:
		return fmt.Sprintf("Custom (0x%04X)", uint16(p))
	}
}

// ShortName returns the short profile identifier.
func (p ApplicationProfile) ShortName() string {
	switch uint16(p) {
	case 0x0104:
		return "ha"
	case 0x0109:
		return "se"
	case 0xA1E0:
		return "gp"
	case 0xC05E:
		return "zll"
	default:
		return "unknown"
	}
}

// ParseProfile parses a profile string into an ApplicationProfile.
func ParseProfile(s string) (ApplicationProfile, error) {
	switch s {
	case "ha", "home-automation", "homeautomation":
		return ApplicationProfile(0x0104), nil
	case "se", "smart-energy", "smartenergy":
		return ApplicationProfile(0x0109), nil
	case "gp", "green-power", "greenpower":
		return ApplicationProfile(0xA1E0), nil
	case "zll", "light-link", "lightlink", "zigbee3":
		return ApplicationProfile(0xC05E), nil
	default:
		return ApplicationProfile(0), fmt.Errorf("unknown profile: %s", s)
	}
}

// AvailableProfiles returns all supported profiles.
func AvailableProfiles() []ApplicationProfile {
	return []ApplicationProfile{
		ProfileHomeAutomation,
		ProfileSmartEnergy,
		ProfileGreenPower,
		ProfileLightLink,
	}
}

// NvSystemID represents NV system identifiers for extended commands.
type NvSystemID uint8

const (
	// NvSysIdZStack is the Z-Stack system identifier.
	NvSysIdZStack NvSystemID = 1
)

// DevState represents the device network state.
type DevState uint8

const (
	// DevStateHold is initialized but not started.
	DevStateHold DevState = 0
	// DevStateInit is initializing.
	DevStateInit DevState = 1
	// DevStateNwkDisc is discovering PAN to join.
	DevStateNwkDisc DevState = 2
	// DevStateNwkJoining is joining a PAN.
	DevStateNwkJoining DevState = 3
	// DevStateNwkRejoin is rejoining a PAN.
	DevStateNwkRejoin DevState = 4
	// DevStateEndDeviceUnauth is started as device, waiting for auth.
	DevStateEndDeviceUnauth DevState = 5
	// DevStateEndDevice is started as end device.
	DevStateEndDevice DevState = 6
	// DevStateRouter is started as router.
	DevStateRouter DevState = 7
	// DevStateCoordStarting is starting as coordinator.
	DevStateCoordStarting DevState = 8
	// DevStateZbCoord is started as coordinator.
	DevStateZbCoord DevState = 9
	// DevStateNwkOrphan is device lost network.
	DevStateNwkOrphan DevState = 10
)

// String returns the device state name.
func (s DevState) String() string {
	switch s {
	case DevStateHold:
		return "Hold"
	case DevStateInit:
		return "Initializing"
	case DevStateNwkDisc:
		return "Discovering"
	case DevStateNwkJoining:
		return "Joining"
	case DevStateNwkRejoin:
		return "Rejoining"
	case DevStateEndDeviceUnauth:
		return "EndDevice (Unauth)"
	case DevStateEndDevice:
		return "EndDevice"
	case DevStateRouter:
		return "Router"
	case DevStateCoordStarting:
		return "Coordinator Starting"
	case DevStateZbCoord:
		return "Coordinator"
	case DevStateNwkOrphan:
		return "Orphan"
	default:
		return fmt.Sprintf("Unknown (%d)", s)
	}
}

// BdbCommissioningMode represents BDB commissioning modes.
type BdbCommissioningMode uint8

const (
	// BdbCommissioningModeNetworkSteering performs network steering.
	BdbCommissioningModeNetworkSteering BdbCommissioningMode = 0x02
	// BdbCommissioningModeNetworkFormation forms a new network.
	BdbCommissioningModeNetworkFormation BdbCommissioningMode = 0x04
	// BdbCommissioningModeFindingBinding performs finding and binding.
	BdbCommissioningModeFindingBinding BdbCommissioningMode = 0x08
)

// AddressMode defines ZDO addressing modes.
type AddressMode uint8

const (
	// AddrModeNotPresent indicates no address present.
	AddrModeNotPresent AddressMode = 0x00
	// AddrModeGroup is group addressing.
	AddrModeGroup AddressMode = 0x01
	// AddrMode16Bit is 16-bit network address.
	AddrMode16Bit AddressMode = 0x02
	// AddrMode64Bit is 64-bit IEEE address.
	AddrMode64Bit AddressMode = 0x03
	// AddrModeBroadcast is broadcast address.
	AddrModeBroadcast AddressMode = 0x0F
)

package znp

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TcDeviceInd represents a Trust Center device indication (device joined).
type TcDeviceInd struct {
	NwkAddr    uint16
	IEEEAddr   [8]byte
	ParentAddr uint16
}

// ParseTcDeviceInd parses a tcDeviceInd AREQ frame.
func ParseTcDeviceInd(data []byte) (*TcDeviceInd, error) {
	buf := NewBuffalo(data)

	nwkAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read nwkAddr: %w", err)
	}

	ieeeAddr, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read ieeeAddr: %w", err)
	}

	parentAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read parentAddr: %w", err)
	}

	return &TcDeviceInd{
		NwkAddr:    nwkAddr,
		IEEEAddr:   ieeeAddr,
		ParentAddr: parentAddr,
	}, nil
}

// WaitForTcDeviceInd waits for a trust center device indication (device join).
func (z *ZNP) WaitForTcDeviceInd(ctx context.Context, timeout time.Duration) (*TcDeviceInd, error) {
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoTcDevInd.ID,
	}

	frame, err := z.waiter.WaitFor(ctx, matcher, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait for tcDeviceInd: %w", err)
	}

	return ParseTcDeviceInd(frame.Data)
}

// ParseLeaveInd parses a leaveInd AREQ frame.
func ParseLeaveInd(data []byte) (*DeviceLeave, error) {
	buf := NewBuffalo(data)

	srcAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read srcAddr: %w", err)
	}

	ieeeAddr, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read ieeeAddr: %w", err)
	}

	request, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read request: %w", err)
	}

	remove, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read remove: %w", err)
	}

	rejoin, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read rejoin: %w", err)
	}

	return &DeviceLeave{
		SrcAddr:  srcAddr,
		IEEEAddr: ieeeAddr,
		Request:  request != 0,
		Remove:   remove != 0,
		Rejoin:   rejoin != 0,
	}, nil
}

// ParseEndDeviceAnnceInd parses an endDeviceAnnceInd AREQ frame.
// This is sent when a device announces itself on the network (join or rejoin).
func ParseEndDeviceAnnceInd(data []byte) (*DeviceAnnounce, error) {
	buf := NewBuffalo(data)

	srcAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read srcAddr: %w", err)
	}

	nwkAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read nwkAddr: %w", err)
	}

	ieeeAddr, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read ieeeAddr: %w", err)
	}

	capabilities, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read capabilities: %w", err)
	}

	return &DeviceAnnounce{
		SrcAddr:      srcAddr,
		NwkAddr:      nwkAddr,
		IEEEAddr:     ieeeAddr,
		Capabilities: capabilities,
	}, nil
}

// WaitForLeaveInd waits for a device leave indication.
func (z *ZNP) WaitForLeaveInd(ctx context.Context, timeout time.Duration) (*DeviceLeave, error) {
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoLeaveInd.ID,
	}

	frame, err := z.waiter.WaitFor(ctx, matcher, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait for leaveInd: %w", err)
	}

	return ParseLeaveInd(frame.Data)
}

// IeeeAddrReq requests the IEEE address of a device by its network address.
func (z *ZNP) IeeeAddrReq(ctx context.Context, nwkAddr uint16) ([8]byte, error) {
	var emptyAddr [8]byte

	// Build request
	writer := NewBuffaloWriter()
	writer.WriteUint16(nwkAddr) // shortAddr
	writer.WriteUint8(0)        // reqType: 0=single device response
	writer.WriteUint8(0)        // startIndex

	// Send SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoIeeeAddrReq, writer.Bytes())
	if err != nil {
		return emptyAddr, fmt.Errorf("ieee addr request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return emptyAddr, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return emptyAddr, fmt.Errorf("ieee addr request returned status %d", status)
	}

	// Wait for AREQ response (ieeeAddrRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoIeeeAddrRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return emptyAddr, fmt.Errorf("waiting for ieee addr response failed: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	// Read status
	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return emptyAddr, fmt.Errorf("failed to read response status: %w", err)
	}

	if respStatus != 0 {
		return emptyAddr, fmt.Errorf("ieee addr response returned status %d", respStatus)
	}

	// Read IEEE address
	ieeeAddr, err := areqBuf.ReadIEEEAddr()
	if err != nil {
		return emptyAddr, fmt.Errorf("failed to read IEEE address: %w", err)
	}

	return ieeeAddr, nil
}

// DeviceAnnounce represents a device join announcement.
type DeviceAnnounce struct {
	SrcAddr      uint16
	NwkAddr      uint16
	IEEEAddr     [8]byte
	Capabilities uint8
}

// DeviceCapability bit flags for device capabilities.
const (
	// CapAltPANCoord indicates alternate PAN coordinator capability.
	CapAltPANCoord uint8 = 0x01
	// CapDeviceType indicates device type (0=EndDevice, 1=Router).
	CapDeviceType uint8 = 0x02
	// CapPowerSource indicates power source (0=Battery, 1=Mains).
	CapPowerSource uint8 = 0x04
	// CapRxOnWhenIdle indicates receiver on when idle.
	CapRxOnWhenIdle uint8 = 0x08
	// CapSecurityCap indicates security capability.
	CapSecurityCap uint8 = 0x40
	// CapAllocAddr indicates address allocation capability.
	CapAllocAddr uint8 = 0x80
)

// IsRouter returns true if capabilities indicate the device is a router.
func (d *DeviceAnnounce) IsRouter() bool {
	return (d.Capabilities & CapDeviceType) != 0
}

// IsBatteryPowered returns true if the device is battery powered.
func (d *DeviceAnnounce) IsBatteryPowered() bool {
	return (d.Capabilities & CapPowerSource) == 0
}

// DeviceLeave represents a device leave event.
type DeviceLeave struct {
	SrcAddr  uint16
	IEEEAddr [8]byte
	Request  bool
	Remove   bool
	Rejoin   bool
}

// ActiveEndpoints represents active endpoints response.
type ActiveEndpoints struct {
	SrcAddr   uint16
	Status    uint8
	NwkAddr   uint16
	Endpoints []uint8
}

// SimpleDescriptor represents a simple descriptor.
type SimpleDescriptor struct {
	Endpoint      uint8
	ProfileID     uint16
	DeviceID      uint16
	DeviceVersion uint8
	InClusters    []uint16
	OutClusters   []uint16
}

// NodeDescriptor contains device type and capabilities from NodeDescReq.
type NodeDescriptor struct {
	NwkAddr          uint16
	Status           uint8
	LogicalType      uint8  // 0=Coordinator, 1=Router, 2=EndDevice
	ComplexDescAvail bool   // Complex descriptor available
	UserDescAvail    bool   // User descriptor available
	APSFlags         uint8  // APS flags (reserved)
	FreqBand         uint8  // Frequency band (0=868MHz, 1=reserved, 2=2.4GHz)
	MacCapFlags      uint8  // MAC capability flags
	ManufCode        uint16 // Manufacturer code
	MaxBufSize       uint8  // Maximum buffer size
	MaxInSize        uint16 // Maximum incoming transfer size
	ServerMask       uint16 // Server mask (supported server capabilities)
	MaxOutSize       uint16 // Maximum outgoing transfer size
	DescCapFlags     uint8  // Descriptor capability flags
}

// NodeDescReq requests node descriptor from a device.
// Returns device type, manufacturer code, and capability flags.
func (z *ZNP) NodeDescReq(ctx context.Context, dstAddr uint16) (*NodeDescriptor, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr) // dstAddr
	writer.WriteUint16(dstAddr) // nwkAddrOfInterest (same as dst)

	// Send CmdZdoNodeDescReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoNodeDescReq, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("node desc request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("node desc request returned status %d", status)
	}

	// Wait for AREQ response (nodeDescRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoNodeDescRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for node desc response failed: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	srcAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read src address: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read response status: %w", err)
	}

	if respStatus != 0 {
		return nil, fmt.Errorf("node desc response returned status %d", respStatus)
	}

	nwkAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read nwk address: %w", err)
	}

	// Verify addresses match
	if srcAddr != dstAddr || nwkAddr != dstAddr {
		return nil, fmt.Errorf("address mismatch: expected %04X, got src=%04X nwk=%04X", dstAddr, srcAddr, nwkAddr)
	}

	// Parse node descriptor fields
	// Byte 1: LogicalType(3) | ComplexDescAvail(1) | UserDescAvail(1) | Reserved(3)
	typeFlags, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read type flags: %w", err)
	}

	logicalType := typeFlags & 0x07
	complexDescAvail := (typeFlags & 0x08) != 0
	userDescAvail := (typeFlags & 0x10) != 0

	// Byte 2: APSFlags(3) | FreqBand(5)
	apsFreq, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read aps/freq: %w", err)
	}

	apsFlags := apsFreq & 0x07
	freqBand := (apsFreq >> 3) & 0x1F

	// Byte 3: MAC capability flags
	macCapFlags, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read mac cap flags: %w", err)
	}

	// Bytes 4-5: Manufacturer code
	manufCode, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read manufacturer code: %w", err)
	}

	// Byte 6: Max buffer size
	maxBufSize, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read max buf size: %w", err)
	}

	// Bytes 7-8: Max incoming transfer size
	maxInSize, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read max in size: %w", err)
	}

	// Bytes 9-10: Server mask
	serverMask, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read server mask: %w", err)
	}

	// Bytes 11-12: Max outgoing transfer size
	maxOutSize, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read max out size: %w", err)
	}

	// Byte 13: Descriptor capability flags
	descCapFlags, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read desc cap flags: %w", err)
	}

	return &NodeDescriptor{
		NwkAddr:          nwkAddr,
		Status:           respStatus,
		LogicalType:      logicalType,
		ComplexDescAvail: complexDescAvail,
		UserDescAvail:    userDescAvail,
		APSFlags:         apsFlags,
		FreqBand:         freqBand,
		MacCapFlags:      macCapFlags,
		ManufCode:        manufCode,
		MaxBufSize:       maxBufSize,
		MaxInSize:        maxInSize,
		ServerMask:       serverMask,
		MaxOutSize:       maxOutSize,
		DescCapFlags:     descCapFlags,
	}, nil
}

// MgmtPermitJoinReq opens or closes the network for device joining.
// duration: 0=close, 1-254=seconds to remain open, 255=always open.
// Returns status (0=success).
func (z *ZNP) MgmtPermitJoinReq(ctx context.Context, duration uint8) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint8(uint8(AddrModeBroadcast)) // addrMode: broadcast
	writer.WriteUint16(0xFFFC)                  // dstAddr: all routers and coordinator
	writer.WriteUint8(duration)                 // duration
	writer.WriteUint8(0)                        // tcSignificance

	// Send CmdZdoMgmtPermitJoinReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoMgmtPermitJoinReq, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("mgmt permit join request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	return status, nil
}

// ActiveEpReq requests active endpoints from a device.
// Returns the active endpoints or an error.
func (z *ZNP) ActiveEpReq(ctx context.Context, dstAddr uint16) (*ActiveEndpoints, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr) // dstAddr
	writer.WriteUint16(dstAddr) // nwkAddrOfInterest (same as dst)

	// Send CmdZdoActiveEpReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoActiveEpReq, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("active ep request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("active ep request returned status %d", status)
	}

	// Wait for AREQ response (activeEpRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoActiveEpRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for active ep response failed: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	srcAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read src address: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read response status: %w", err)
	}

	nwkAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read nwk address: %w", err)
	}

	epCount, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read endpoint count: %w", err)
	}

	// Read endpoint list
	endpoints := make([]uint8, epCount)
	for i := uint8(0); i < epCount; i++ {
		ep, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read endpoint %d: %w", i, err)
		}
		endpoints[i] = ep
	}

	return &ActiveEndpoints{
		SrcAddr:   srcAddr,
		Status:    respStatus,
		NwkAddr:   nwkAddr,
		Endpoints: endpoints,
	}, nil
}

// NeighborEntry represents a single entry in the neighbor table.
type NeighborEntry struct {
	ExtPanID   [8]byte // Extended PAN ID
	ExtAddr    [8]byte // IEEE address of neighbor
	NwkAddr    uint16  // Network address of neighbor
	DeviceType uint8   // 0=Coordinator, 1=Router, 2=EndDevice
	RxOnIdle   uint8   // Receiver on when idle
	Relation   uint8   // 0=Parent, 1=Child, 2=Sibling, 3=None, 4=Previous Child
	PermitJoin uint8   // Permit joining
	Depth      uint8   // Tree depth
	LQI        uint8   // Link Quality Indicator (0-255)
}

// NeighborTable represents the response from a MgmtLqi request.
type NeighborTable struct {
	SrcAddr          uint16
	Status           uint8
	NeighborTableLen uint8           // Total entries in table
	StartIndex       uint8           // Index of first entry returned
	Neighbors        []NeighborEntry // Returned entries
}

// MgmtLqiReq requests the neighbor table from a device.
// startIndex specifies which entry to start from (for pagination).
func (z *ZNP) MgmtLqiReq(ctx context.Context, dstAddr uint16, startIndex uint8) (*NeighborTable, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)
	writer.WriteUint8(startIndex)

	// Send SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoMgmtLqiReq, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("mgmt lqi request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("mgmt lqi request returned status %d", status)
	}

	// Wait for AREQ response
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoMgmtLqiRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for mgmt lqi response: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	srcAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read src addr: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	neighborTableLen, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read neighbor table len: %w", err)
	}

	neighborStartIndex, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read start index: %w", err)
	}

	neighborCount, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read neighbor count: %w", err)
	}

	// Parse neighbor entries
	neighbors := make([]NeighborEntry, neighborCount)
	for i := uint8(0); i < neighborCount; i++ {
		entry := NeighborEntry{}

		// Extended PAN ID (8 bytes)
		extPanID, err := areqBuf.ReadIEEEAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to read ext pan id: %w", err)
		}
		entry.ExtPanID = extPanID

		// Extended address (8 bytes)
		extAddr, err := areqBuf.ReadIEEEAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to read ext addr: %w", err)
		}
		entry.ExtAddr = extAddr

		// Network address (2 bytes)
		nwkAddr, err := areqBuf.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("failed to read nwk addr: %w", err)
		}
		entry.NwkAddr = nwkAddr

		// Packed byte: DeviceType(2) | RxOnIdle(2) | Relation(3) | Reserved(1)
		packed1, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read packed byte 1: %w", err)
		}
		entry.DeviceType = packed1 & 0x03
		entry.RxOnIdle = (packed1 >> 2) & 0x03
		entry.Relation = (packed1 >> 4) & 0x07

		// Packed byte: PermitJoin(2) | Reserved(6)
		packed2, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read packed byte 2: %w", err)
		}
		entry.PermitJoin = packed2 & 0x03

		// Depth
		depth, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read depth: %w", err)
		}
		entry.Depth = depth

		// LQI
		lqi, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read lqi: %w", err)
		}
		entry.LQI = lqi

		neighbors[i] = entry
	}

	return &NeighborTable{
		SrcAddr:          srcAddr,
		Status:           respStatus,
		NeighborTableLen: neighborTableLen,
		StartIndex:       neighborStartIndex,
		Neighbors:        neighbors,
	}, nil
}

// GetAllNeighbors retrieves the complete neighbor table from a device.
// It handles pagination automatically.
func (z *ZNP) GetAllNeighbors(ctx context.Context, dstAddr uint16) ([]NeighborEntry, error) {
	var allNeighbors []NeighborEntry
	startIndex := uint8(0)

	for {
		table, err := z.MgmtLqiReq(ctx, dstAddr, startIndex)
		if err != nil {
			return nil, err
		}

		if table.Status != 0 {
			return nil, fmt.Errorf("neighbor table request returned status %d", table.Status)
		}

		allNeighbors = append(allNeighbors, table.Neighbors...)

		// Check if we got all entries
		if uint8(len(allNeighbors)) >= table.NeighborTableLen {
			break
		}

		startIndex = uint8(len(allNeighbors))
	}

	return allNeighbors, nil
}

// RoutingEntry represents a single routing table entry.
type RoutingEntry struct {
	DstAddr      uint16 // Destination network address
	Status       uint8  // 0=Active, 1=Discovery Underway, 2=Discovery Failed, 3=Inactive, 4=Validation Underway
	NextHop      uint16 // Next hop network address
	Concentrator bool   // True if route is to a concentrator
	RouteRecord  bool   // True if route record table entry
	ManyToOne    bool   // True if many-to-one route discovery
}

// RoutingTable contains the routing table response.
type RoutingTable struct {
	SrcAddr         uint16         // Source address that responded
	Status          uint8          // Status of the response
	RoutingTableLen uint8          // Total entries in routing table
	StartIndex      uint8          // Index of first entry returned
	Entries         []RoutingEntry // Routing table entries
}

// MgmtRtgReq requests the routing table from a device (ZDO command 0x0032).
// startIndex specifies which entry to start from (for pagination).
func (z *ZNP) MgmtRtgReq(ctx context.Context, dstAddr uint16, startIndex uint8) (*RoutingTable, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)
	writer.WriteUint8(startIndex)

	// Send SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoMgmtRtgReq, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("mgmt rtg request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("mgmt rtg request returned status %d", status)
	}

	// Wait for AREQ response
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoMgmtRtgRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for mgmt rtg response: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	srcAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read src addr: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	routingTableLen, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read routing table len: %w", err)
	}

	routingStartIndex, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read start index: %w", err)
	}

	routingCount, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read routing count: %w", err)
	}

	// Parse routing entries
	entries := make([]RoutingEntry, routingCount)
	for i := uint8(0); i < routingCount; i++ {
		entry := RoutingEntry{}

		// Destination address (2 bytes)
		dstAddr, err := areqBuf.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("failed to read dst addr: %w", err)
		}
		entry.DstAddr = dstAddr

		// Status (1 byte): Status(3 bits) | MemoryConstrained(1) | ManyToOne(1) | RouteRecordRequired(1) | GroupIdFlag(1) | Reserved(1)
		statusByte, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read status byte: %w", err)
		}
		entry.Status = statusByte & 0x07
		entry.Concentrator = (statusByte & 0x08) != 0 // bit 3
		entry.ManyToOne = (statusByte & 0x10) != 0    // bit 4
		entry.RouteRecord = (statusByte & 0x20) != 0  // bit 5

		// Next hop address (2 bytes)
		nextHop, err := areqBuf.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("failed to read next hop: %w", err)
		}
		entry.NextHop = nextHop

		entries[i] = entry
	}

	return &RoutingTable{
		SrcAddr:         srcAddr,
		Status:          respStatus,
		RoutingTableLen: routingTableLen,
		StartIndex:      routingStartIndex,
		Entries:         entries,
	}, nil
}

// GetAllRoutes retrieves the complete routing table from a device.
// It handles pagination automatically.
func (z *ZNP) GetAllRoutes(ctx context.Context, dstAddr uint16) ([]RoutingEntry, error) {
	var allRoutes []RoutingEntry
	startIndex := uint8(0)

	for {
		table, err := z.MgmtRtgReq(ctx, dstAddr, startIndex)
		if err != nil {
			return nil, err
		}

		if table.Status != 0 {
			return nil, fmt.Errorf("routing table request returned status %d", table.Status)
		}

		allRoutes = append(allRoutes, table.Entries...)

		// Check if we got all entries
		if uint8(len(allRoutes)) >= table.RoutingTableLen {
			break
		}

		startIndex = uint8(len(allRoutes))
	}

	return allRoutes, nil
}

// BindingEntry represents a single entry in the binding table.
type BindingEntry struct {
	SrcAddr     [8]byte // Source IEEE address
	SrcEndpoint uint8   // Source endpoint
	ClusterID   uint16  // Cluster ID
	DstAddrMode uint8   // 0x01=group, 0x03=IEEE address
	DstAddr     [8]byte // Destination IEEE address (or group ID in first 2 bytes)
	DstEndpoint uint8   // Destination endpoint (only if DstAddrMode is 0x03)
}

// BindingTable represents the response from a MgmtBind request.
type BindingTable struct {
	SrcAddr         uint16
	Status          uint8
	BindingTableLen uint8          // Total entries in table
	StartIndex      uint8          // Index of first entry returned
	Bindings        []BindingEntry // Returned entries
}

// MgmtBindReq requests the binding table from a device.
// startIndex specifies which entry to start from (for pagination).
func (z *ZNP) MgmtBindReq(ctx context.Context, dstAddr uint16, startIndex uint8) (*BindingTable, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)
	writer.WriteUint8(startIndex)

	// Send SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoMgmtBindReq, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("mgmt bind request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("mgmt bind request returned status %d", status)
	}

	// Wait for AREQ response
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoMgmtBindRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for mgmt bind response: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	srcAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read src addr: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	bindingTableLen, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read binding table len: %w", err)
	}

	bindingStartIndex, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read start index: %w", err)
	}

	bindingCount, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read binding count: %w", err)
	}

	// Parse binding entries
	bindings := make([]BindingEntry, 0, bindingCount)
	for i := uint8(0); i < bindingCount; i++ {
		entry := BindingEntry{}

		// Source IEEE address (8 bytes)
		srcIEEE, err := areqBuf.ReadIEEEAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to read src ieee addr: %w", err)
		}
		entry.SrcAddr = srcIEEE

		// Source endpoint
		srcEp, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read src endpoint: %w", err)
		}
		entry.SrcEndpoint = srcEp

		// Cluster ID
		clusterID, err := areqBuf.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("failed to read cluster id: %w", err)
		}
		entry.ClusterID = clusterID

		// Destination address mode
		dstAddrMode, err := areqBuf.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("failed to read dst addr mode: %w", err)
		}
		entry.DstAddrMode = dstAddrMode

		// Destination address (always 8 bytes in Z-Stack response)
		dstAddr, err := areqBuf.ReadIEEEAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to read dst addr: %w", err)
		}
		entry.DstAddr = dstAddr

		// Destination endpoint (only present if addr mode is 0x03)
		if dstAddrMode == 0x03 {
			dstEp, err := areqBuf.ReadUint8()
			if err != nil {
				return nil, fmt.Errorf("failed to read dst endpoint: %w", err)
			}
			entry.DstEndpoint = dstEp
		}

		bindings = append(bindings, entry)
	}

	return &BindingTable{
		SrcAddr:         srcAddr,
		Status:          respStatus,
		BindingTableLen: bindingTableLen,
		StartIndex:      bindingStartIndex,
		Bindings:        bindings,
	}, nil
}

// GetAllBindings retrieves the complete binding table from a device.
// It handles pagination automatically.
func (z *ZNP) GetAllBindings(ctx context.Context, dstAddr uint16) ([]BindingEntry, error) {
	var allBindings []BindingEntry
	startIndex := uint8(0)

	for {
		table, err := z.MgmtBindReq(ctx, dstAddr, startIndex)
		if err != nil {
			return nil, err
		}

		if table.Status != 0 {
			return nil, fmt.Errorf("binding table request returned status %d", table.Status)
		}

		allBindings = append(allBindings, table.Bindings...)

		// Check if we got all entries
		if uint8(len(allBindings)) >= table.BindingTableLen {
			break
		}

		startIndex = uint8(len(allBindings))
	}

	return allBindings, nil
}

// SimpleDescReq requests simple descriptor from device endpoint.
// Returns the simple descriptor or an error.
func (z *ZNP) SimpleDescReq(ctx context.Context, dstAddr uint16, endpoint uint8) (*SimpleDescriptor, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr) // dstAddr
	writer.WriteUint16(dstAddr) // nwkAddrOfInterest (same as dst)
	writer.WriteUint8(endpoint) // endpoint

	// Send CmdZdoSimpleDescReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoSimpleDescReq, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("simple desc request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return nil, fmt.Errorf("simple desc request returned status %d", status)
	}

	// Wait for AREQ response (simpleDescRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoSimpleDescRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for simple desc response failed: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	srcAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read src address: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read response status: %w", err)
	}

	if respStatus != 0 {
		return nil, fmt.Errorf("simple desc response returned status %d", respStatus)
	}

	nwkAddr, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read nwk address: %w", err)
	}

	// Verify addresses match
	if srcAddr != dstAddr || nwkAddr != dstAddr {
		return nil, fmt.Errorf("address mismatch: expected %04X, got src=%04X nwk=%04X", dstAddr, srcAddr, nwkAddr)
	}

	// Read length byte (descriptor length)
	descLen, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read descriptor length: %w", err)
	}

	if descLen == 0 {
		return nil, fmt.Errorf("invalid descriptor length: 0")
	}

	// Read endpoint
	ep, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read endpoint: %w", err)
	}

	// Read profile ID
	profileID, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read profile ID: %w", err)
	}

	// Read device ID
	deviceID, err := areqBuf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read device ID: %w", err)
	}

	// Read device version (4 bits) + reserved (4 bits)
	deviceVersionByte, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read device version: %w", err)
	}
	deviceVersion := deviceVersionByte & 0x0F

	// Read input cluster count
	inClusterCount, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read input cluster count: %w", err)
	}

	// Read input clusters
	inClusters := make([]uint16, inClusterCount)
	for i := uint8(0); i < inClusterCount; i++ {
		cluster, err := areqBuf.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("failed to read input cluster %d: %w", i, err)
		}
		inClusters[i] = cluster
	}

	// Read output cluster count
	outClusterCount, err := areqBuf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read output cluster count: %w", err)
	}

	// Read output clusters
	outClusters := make([]uint16, outClusterCount)
	for i := uint8(0); i < outClusterCount; i++ {
		cluster, err := areqBuf.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("failed to read output cluster %d: %w", i, err)
		}
		outClusters[i] = cluster
	}

	return &SimpleDescriptor{
		Endpoint:      ep,
		ProfileID:     profileID,
		DeviceID:      deviceID,
		DeviceVersion: deviceVersion,
		InClusters:    inClusters,
		OutClusters:   outClusters,
	}, nil
}

// MgmtLeaveReq sends a management leave request to tell a device to leave the network.
// Options: bit0 (0x01) = removeChildren, bit1 (0x02) = rejoin
// Returns status (0=success).
func (z *ZNP) MgmtLeaveReq(ctx context.Context, dstAddr uint16, ieeeAddr [8]byte, removeChildren, rejoin bool) (uint8, error) {
	// Build options byte
	var options uint8
	if removeChildren {
		options |= 0x01
	}
	if rejoin {
		options |= 0x02
	}

	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)
	writer.WriteIEEEAddr(ieeeAddr)
	writer.WriteUint8(options)

	// Send CmdZdoMgmtLeaveReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoMgmtLeaveReq, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("mgmt leave request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return status, nil
	}

	// Wait for AREQ response (mgmtLeaveRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoMgmtLeaveRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return 0, fmt.Errorf("waiting for mgmt leave response: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	_, err = areqBuf.ReadUint16() // srcAddr
	if err != nil {
		return 0, fmt.Errorf("failed to read src address: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read response status: %w", err)
	}

	return respStatus, nil
}

// BindReq creates a binding entry on a device.
// Binding tells a device where to send reports (e.g., attribute reports, commands).
// For most use cases, bind a device's cluster to the coordinator's IEEE address.
//
// Parameters:
//   - dstAddr: Network address of the device to configure binding on
//   - srcIEEEAddr: IEEE address of the source device (the device being bound)
//   - srcEndpoint: Source endpoint on the device
//   - clusterID: Cluster ID to bind
//   - dstIEEEAddr: Destination IEEE address (e.g., coordinator's IEEE address)
//   - dstEndpoint: Destination endpoint (e.g., 1 for coordinator)
//
// Returns status (0=success).
func (z *ZNP) BindReq(ctx context.Context, dstAddr uint16, srcIEEEAddr [8]byte, srcEndpoint uint8, clusterID uint16, dstIEEEAddr [8]byte, dstEndpoint uint8) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)       // Network address to send command to
	writer.WriteIEEEAddr(srcIEEEAddr) // Source IEEE address (device to bind)
	writer.WriteUint8(srcEndpoint)    // Source endpoint
	writer.WriteUint16(clusterID)     // Cluster ID to bind
	writer.WriteUint8(0x03)           // Destination address mode: 0x03 = 64-bit IEEE address
	writer.WriteIEEEAddr(dstIEEEAddr) // Destination IEEE address (where reports go)
	writer.WriteUint8(dstEndpoint)    // Destination endpoint

	// Send CmdZdoBindReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoBindReq, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("bind request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return status, nil
	}

	// Wait for AREQ response (bindRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoBindRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return 0, fmt.Errorf("waiting for bind response: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	_, err = areqBuf.ReadUint16() // srcAddr
	if err != nil {
		return 0, fmt.Errorf("failed to read src address: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read response status: %w", err)
	}

	return respStatus, nil
}

// UnbindReq removes a binding entry from a device.
// This stops the device from sending reports to the specified destination.
//
// Parameters:
//   - dstAddr: Network address of the device to remove binding from
//   - srcIEEEAddr: IEEE address of the source device (the device being unbound)
//   - srcEndpoint: Source endpoint on the device
//   - clusterID: Cluster ID to unbind
//   - dstIEEEAddr: Destination IEEE address (e.g., coordinator's IEEE address)
//   - dstEndpoint: Destination endpoint (e.g., 1 for coordinator)
//
// Returns status (0=success).
func (z *ZNP) UnbindReq(ctx context.Context, dstAddr uint16, srcIEEEAddr [8]byte, srcEndpoint uint8, clusterID uint16, dstIEEEAddr [8]byte, dstEndpoint uint8) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint16(dstAddr)       // Network address to send command to
	writer.WriteIEEEAddr(srcIEEEAddr) // Source IEEE address (device to unbind)
	writer.WriteUint8(srcEndpoint)    // Source endpoint
	writer.WriteUint16(clusterID)     // Cluster ID to unbind
	writer.WriteUint8(0x03)           // Destination address mode: 0x03 = 64-bit IEEE address
	writer.WriteIEEEAddr(dstIEEEAddr) // Destination IEEE address
	writer.WriteUint8(dstEndpoint)    // Destination endpoint

	// Send CmdZdoUnbindReq SREQ
	resp, err := z.Request(ctx, unpi.ZDO, CmdZdoUnbindReq, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("unbind request failed: %w", err)
	}

	// Parse immediate SREQ response
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	if status != 0 {
		return status, nil
	}

	// Wait for AREQ response (unbindRsp)
	matcher := FrameMatcher{
		Type:      unpi.AREQ,
		Subsystem: unpi.ZDO,
		CommandID: CmdZdoUnbindRsp.ID,
	}

	areqResp, err := z.waiter.WaitFor(ctx, matcher, DefaultSREQTimeout)
	if err != nil {
		return 0, fmt.Errorf("waiting for unbind response: %w", err)
	}

	// Parse AREQ response
	areqBuf := NewBuffalo(areqResp.Data)

	_, err = areqBuf.ReadUint16() // srcAddr
	if err != nil {
		return 0, fmt.Errorf("failed to read src address: %w", err)
	}

	respStatus, err := areqBuf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read response status: %w", err)
	}

	return respStatus, nil
}

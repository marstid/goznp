package znp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/marstid/goznp/pkg/unpi"
)

// DeviceInfo contains information about the coordinator device.
type DeviceInfo struct {
	IEEEAddr        [8]byte
	ShortAddr       uint16
	DeviceType      uint8 // 0=Coordinator, 1=Router, 2=EndDevice
	DeviceState     uint8
	NumAssocDevices uint8
	AssocDevices    []uint16 // Network addresses of associated devices
}

// DeviceType constants
const (
	DeviceTypeCoordinator uint8 = 0
	DeviceTypeRouter      uint8 = 1
	DeviceTypeEndDevice   uint8 = 2
)

// AssocDevice represents an associated device entry.
type AssocDevice struct {
	ShortAddr    uint16
	IEEEAddr     [8]byte
	NodeRelation uint8
	Age          uint8
	LinkInfo     struct {
		TxCounter     uint8
		TxCost        uint8
		RxLqi         uint8
		InKeySeqNum   uint8
		InFrmCntrHigh uint16
	}
}

// GetDeviceInfo retrieves information about this device (coordinator).
func (z *ZNP) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	// Send CmdUtilGetDeviceInfo SREQ
	resp, err := z.Request(ctx, unpi.UTIL, CmdUtilGetDeviceInfo, nil)
	if err != nil {
		return nil, fmt.Errorf("get device info request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)

	// Read status byte
	status, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	// Check for success (status 0)
	if status != 0 {
		return nil, fmt.Errorf("device info request failed with status: %d", status)
	}

	// Read IEEE address (8 bytes)
	ieeeAddr, err := buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read IEEE address: %w", err)
	}

	// Read short address (uint16)
	shortAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read short address: %w", err)
	}

	// Read device type (uint8)
	deviceType, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read device type: %w", err)
	}

	// Read device state (uint8)
	deviceState, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read device state: %w", err)
	}

	// Read number of associated devices (uint8)
	numAssocDevices, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read num assoc devices: %w", err)
	}

	// Read associated devices list (network addresses)
	assocDevices := make([]uint16, 0, numAssocDevices)
	for i := uint8(0); i < numAssocDevices; i++ {
		addr, err := buf.ReadUint16()
		if err != nil {
			// Stop reading if we run out of data
			break
		}
		assocDevices = append(assocDevices, addr)
	}

	return &DeviceInfo{
		IEEEAddr:        ieeeAddr,
		ShortAddr:       shortAddr,
		DeviceType:      deviceType,
		DeviceState:     deviceState,
		NumAssocDevices: numAssocDevices,
		AssocDevices:    assocDevices,
	}, nil
}

// GetAssocCount returns the count of associated devices.
func (z *ZNP) GetAssocCount(ctx context.Context) (uint16, error) {
	// Build request: startRelation=0, startIndex=0
	writer := NewBuffaloWriter()
	writer.WriteUint8(0) // startRelation
	writer.WriteUint8(0) // startIndex

	// Send CmdUtilAssocCount SREQ
	resp, err := z.Request(ctx, unpi.UTIL, CmdUtilAssocCount, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("assoc count request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)

	// Read count (uint16)
	cnt, err := buf.ReadUint16()
	if err != nil {
		return 0, fmt.Errorf("failed to read count: %w", err)
	}

	return cnt, nil
}

// GetAssocDevice retrieves an associated device by index.
func (z *ZNP) GetAssocDevice(ctx context.Context, index uint8) (*AssocDevice, error) {
	// Build request: number=index
	writer := NewBuffaloWriter()
	writer.WriteUint8(index)

	// Send CmdUtilAssocFindDevice SREQ
	resp, err := z.Request(ctx, unpi.UTIL, CmdUtilAssocFindDevice, writer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("assoc find device request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)

	// Read short address (uint16)
	shortAddr, err := buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read short address: %w", err)
	}

	// Read addrIdx (uint16) - skip
	_, err = buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read addrIdx: %w", err)
	}

	// Read node relation (uint8)
	nodeRelation, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read node relation: %w", err)
	}

	// Read devStatus (uint8) - skip
	_, err = buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read devStatus: %w", err)
	}

	// Read assocCnt (uint8) - skip
	_, err = buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read assocCnt: %w", err)
	}

	// Read age (uint8)
	age, err := buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read age: %w", err)
	}

	// Read linkInfo struct
	var device AssocDevice
	device.ShortAddr = shortAddr
	device.NodeRelation = nodeRelation
	device.Age = age

	// txCounter (uint8)
	device.LinkInfo.TxCounter, err = buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read txCounter: %w", err)
	}

	// txCost (uint8)
	device.LinkInfo.TxCost, err = buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read txCost: %w", err)
	}

	// rxLqi (uint8)
	device.LinkInfo.RxLqi, err = buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read rxLqi: %w", err)
	}

	// inKeySeqNum (uint8)
	device.LinkInfo.InKeySeqNum, err = buf.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to read inKeySeqNum: %w", err)
	}

	// inFrmCntrHigh (uint16)
	device.LinkInfo.InFrmCntrHigh, err = buf.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("failed to read inFrmCntrHigh: %w", err)
	}

	// Read IEEE address (8 bytes) - at the end
	device.IEEEAddr, err = buf.ReadIEEEAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to read IEEE address: %w", err)
	}

	return &device, nil
}

// FormatIEEEAddr formats an IEEE address as a colon-separated hex string.
func FormatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X",
		addr[0], addr[1], addr[2], addr[3], addr[4], addr[5], addr[6], addr[7])
}

// ParseIEEEAddr parses a colon-separated hex string into an IEEE address.
func ParseIEEEAddr(s string) ([8]byte, error) {
	var addr [8]byte

	// Remove colons and parse
	s = strings.ReplaceAll(s, ":", "")
	if len(s) != 16 {
		return addr, fmt.Errorf("invalid IEEE address length: %s", s)
	}

	for i := 0; i < 8; i++ {
		b, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return addr, fmt.Errorf("invalid hex byte in IEEE address: %w", err)
		}
		addr[i] = uint8(b)
	}

	return addr, nil
}

// SetChannels sets the channel mask for the device.
// The channelMask is a bit mask where bit N enables channel N.
// Valid Zigbee channels are 11-26 (bits 11-26).
// Example: 0x00008000 = channel 15, 0x07FFF800 = all channels 11-26.
func (z *ZNP) SetChannels(ctx context.Context, channelMask uint32) (uint8, error) {
	// Build request data
	writer := NewBuffaloWriter()
	writer.WriteUint32(channelMask)

	// Send CmdUtilSetChannels SREQ
	resp, err := z.Request(ctx, unpi.UTIL, CmdUtilSetChannels, writer.Bytes())
	if err != nil {
		return 0, fmt.Errorf("set channels request failed: %w", err)
	}

	// Parse response using Buffalo
	buf := NewBuffalo(resp.Data)
	status, err := buf.ReadUint8()
	if err != nil {
		return 0, fmt.Errorf("failed to read status: %w", err)
	}

	return status, nil
}

// ChannelToMask converts a single channel number to a channel mask.
// Valid channels are 11-26.
func ChannelToMask(channel uint8) uint32 {
	if channel < 11 || channel > 26 {
		return 0
	}
	return 1 << channel
}

// AllChannelsMask returns a mask with all valid Zigbee channels enabled (11-26).
func AllChannelsMask() uint32 {
	return 0x07FFF800
}

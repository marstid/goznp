package adapter

import (
	"context"
	"time"

	"github.com/marstid/goznp/pkg/znp"
)

// ZNPClient defines the interface for interacting with the Z-Stack ZNP layer.
// This interface enables mocking the ZNP layer in tests, allowing unit tests
// of the adapter package without requiring a physical Zigbee coordinator.
//
// The *znp.ZNP type implements this interface, so existing code using *znp.ZNP
// can seamlessly switch to using this interface type instead.
type ZNPClient interface {
	// Lifecycle methods manage the ZNP connection lifecycle.
	Open(ctx context.Context) error
	Close() error

	// System commands query basic coordinator information and perform resets.
	Ping(ctx context.Context) (*znp.PingCapabilities, error)
	Version(ctx context.Context) (*znp.VersionInfo, error)
	Reset(ctx context.Context, resetType znp.ResetType) (*znp.ResetIndication, error)
	GetDeviceInfo(ctx context.Context) (*znp.DeviceInfo, error)
	StartupFromApp(ctx context.Context, startDelay uint16) (uint8, error)
	SetTxPower(ctx context.Context, power int8) (int8, error)

	// Network management configures and monitors network state.
	ExtNwkInfo(ctx context.Context) (*znp.ExtNetworkInfo, error)
	BdbSetChannel(ctx context.Context, isPrimary bool, channel uint32) (uint8, error)
	BdbStartCommissioning(ctx context.Context, mode znp.BdbCommissioningMode) (uint8, error)
	WaitForStateChange(ctx context.Context, timeout time.Duration) (znp.DevState, error)

	// AF (Application Framework) layer manages endpoints and data transmission.
	AfRegister(ctx context.Context, config znp.EndpointConfig) (uint8, error)
	AfDelete(ctx context.Context, endpoint uint8) (uint8, error)
	AfDataRequest(ctx context.Context, req znp.DataRequest) (uint8, error)
	WaitForDataConfirm(ctx context.Context, transID uint8, timeout time.Duration) (*znp.DataConfirm, error)
	WaitForIncomingMsg(ctx context.Context, srcAddr uint16, clusterID uint16, timeout time.Duration) (*znp.IncomingMessage, error)

	// ZDO (Zigbee Device Objects) commands perform device discovery and management.
	MgmtPermitJoinReq(ctx context.Context, duration uint8) (uint8, error)
	MgmtLeaveReq(ctx context.Context, dstAddr uint16, ieeeAddr [8]byte, removeChildren, rejoin bool) (uint8, error)
	MgmtNwkUpdateReq(ctx context.Context, dstAddr uint16, dstAddrMode uint8, channelMask uint32, scanDuration uint8, scanCount uint8, nwkManagerAddr uint16) (uint8, error)
	ActiveEpReq(ctx context.Context, dstAddr uint16) (*znp.ActiveEndpoints, error)
	SimpleDescReq(ctx context.Context, dstAddr uint16, endpoint uint8) (*znp.SimpleDescriptor, error)
	NodeDescReq(ctx context.Context, dstAddr uint16) (*znp.NodeDescriptor, error)
	IeeeAddrReq(ctx context.Context, nwkAddr uint16) ([8]byte, error)
	BindReq(ctx context.Context, dstAddr uint16, srcIEEEAddr [8]byte, srcEndpoint uint8, clusterID uint16, dstIEEEAddr [8]byte, dstEndpoint uint8) (uint8, error)
	UnbindReq(ctx context.Context, dstAddr uint16, srcIEEEAddr [8]byte, srcEndpoint uint8, clusterID uint16, dstIEEEAddr [8]byte, dstEndpoint uint8) (uint8, error)

	// Network topology queries retrieve neighbor, routing, and binding tables.
	GetAllNeighbors(ctx context.Context, dstAddr uint16) ([]znp.NeighborEntry, error)
	GetAllBindings(ctx context.Context, dstAddr uint16) ([]znp.BindingEntry, error)
	GetAllRoutes(ctx context.Context, dstAddr uint16) ([]znp.RoutingEntry, error)

	// NVRAM operations provide persistent storage access.
	NvLength(ctx context.Context, id znp.NvItemID) (uint16, error)
	NvRead(ctx context.Context, id znp.NvItemID, offset uint8) ([]byte, error)
	NvWrite(ctx context.Context, id znp.NvItemID, offset uint8, data []byte) error
	NvReadAll(ctx context.Context, id znp.NvItemID) ([]byte, error)
	NvWriteAll(ctx context.Context, id znp.NvItemID, data []byte) error
	NvItemInit(ctx context.Context, id znp.NvItemID, itemLen uint16, initData []byte) error

	// Address Manager operations manage the coordinator's device table.
	ReadAddrMgrTable(ctx context.Context) ([]znp.AddrMgrEntry, error)
	DeleteAddrMgrEntry(ctx context.Context, ieeeAddr [8]byte) (bool, error)

	// Device name operations provide human-readable device naming.
	SetDeviceName(ctx context.Context, ieeeAddr [8]byte, name, comment string) error
	GetDeviceName(ctx context.Context, ieeeAddr [8]byte) (*znp.DeviceNameEntry, error)
	DeleteDeviceName(ctx context.Context, ieeeAddr [8]byte) error
	ReadDeviceNameTable(ctx context.Context) (*znp.DeviceNameTable, error)

	// Event callbacks register handlers for asynchronous device events.
	OnDeviceJoin(handler func(*znp.TcDeviceInd))
	OnDeviceLeave(handler func(*znp.DeviceLeave))
	OnDeviceAnnounce(handler func(*znp.DeviceAnnounce))
}

package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/znp"
)

// mockZNPClient is a simple mock implementation of ZNPClient for testing.
// In production tests, use a more sophisticated mocking library like gomock or testify/mock.
type mockZNPClient struct {
	openCalled  bool
	closeCalled bool
	versionInfo *znp.VersionInfo
}

// Lifecycle methods
func (m *mockZNPClient) Open(ctx context.Context) error {
	m.openCalled = true
	return nil
}

func (m *mockZNPClient) Close() error {
	m.closeCalled = true
	return nil
}

// System commands
func (m *mockZNPClient) Ping(ctx context.Context) (*znp.PingCapabilities, error) {
	return &znp.PingCapabilities{Capabilities: 0x01}, nil
}

func (m *mockZNPClient) Version(ctx context.Context) (*znp.VersionInfo, error) {
	return m.versionInfo, nil
}

func (m *mockZNPClient) Reset(ctx context.Context, resetType znp.ResetType) (*znp.ResetIndication, error) {
	return &znp.ResetIndication{}, nil
}

func (m *mockZNPClient) GetDeviceInfo(ctx context.Context) (*znp.DeviceInfo, error) {
	return &znp.DeviceInfo{}, nil
}

func (m *mockZNPClient) StartupFromApp(ctx context.Context, startDelay uint16) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) SetTxPower(ctx context.Context, power int8) (int8, error) {
	return power, nil
}

// Network management
func (m *mockZNPClient) ExtNwkInfo(ctx context.Context) (*znp.ExtNetworkInfo, error) {
	return &znp.ExtNetworkInfo{}, nil
}

func (m *mockZNPClient) BdbSetChannel(ctx context.Context, isPrimary bool, channel uint32) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) BdbStartCommissioning(ctx context.Context, mode znp.BdbCommissioningMode) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) WaitForStateChange(ctx context.Context, timeout time.Duration) (znp.DevState, error) {
	return 0, nil
}

// AF layer
func (m *mockZNPClient) AfRegister(ctx context.Context, config znp.EndpointConfig) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) AfDelete(ctx context.Context, endpoint uint8) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) AfDataRequest(ctx context.Context, req znp.DataRequest) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) WaitForDataConfirm(ctx context.Context, transID uint8, timeout time.Duration) (*znp.DataConfirm, error) {
	return &znp.DataConfirm{}, nil
}

func (m *mockZNPClient) WaitForIncomingMsg(ctx context.Context, srcAddr uint16, clusterID uint16, timeout time.Duration) (*znp.IncomingMessage, error) {
	return &znp.IncomingMessage{}, nil
}

// ZDO commands
func (m *mockZNPClient) MgmtPermitJoinReq(ctx context.Context, duration uint8) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) MgmtLeaveReq(ctx context.Context, dstAddr uint16, ieeeAddr [8]byte, removeChildren, rejoin bool) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) MgmtNwkUpdateReq(ctx context.Context, dstAddr uint16, dstAddrMode uint8, channelMask uint32, scanDuration uint8, scanCount uint8, nwkManagerAddr uint16) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) ActiveEpReq(ctx context.Context, dstAddr uint16) (*znp.ActiveEndpoints, error) {
	return &znp.ActiveEndpoints{}, nil
}

func (m *mockZNPClient) SimpleDescReq(ctx context.Context, dstAddr uint16, endpoint uint8) (*znp.SimpleDescriptor, error) {
	return &znp.SimpleDescriptor{}, nil
}

func (m *mockZNPClient) NodeDescReq(ctx context.Context, dstAddr uint16) (*znp.NodeDescriptor, error) {
	return &znp.NodeDescriptor{}, nil
}

func (m *mockZNPClient) IeeeAddrReq(ctx context.Context, nwkAddr uint16) ([8]byte, error) {
	return [8]byte{}, nil
}

func (m *mockZNPClient) BindReq(ctx context.Context, dstAddr uint16, srcIEEEAddr [8]byte, srcEndpoint uint8, clusterID uint16, dstIEEEAddr [8]byte, dstEndpoint uint8) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) UnbindReq(ctx context.Context, dstAddr uint16, srcIEEEAddr [8]byte, srcEndpoint uint8, clusterID uint16, dstIEEEAddr [8]byte, dstEndpoint uint8) (uint8, error) {
	return 0, nil
}

// Network topology
func (m *mockZNPClient) GetAllNeighbors(ctx context.Context, dstAddr uint16) ([]znp.NeighborEntry, error) {
	return []znp.NeighborEntry{}, nil
}

func (m *mockZNPClient) GetAllBindings(ctx context.Context, dstAddr uint16) ([]znp.BindingEntry, error) {
	return []znp.BindingEntry{}, nil
}

func (m *mockZNPClient) GetAllRoutes(ctx context.Context, dstAddr uint16) ([]znp.RoutingEntry, error) {
	return []znp.RoutingEntry{}, nil
}

// NVRAM operations
func (m *mockZNPClient) NvRead(ctx context.Context, id znp.NvItemID, offset uint8) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockZNPClient) NvWrite(ctx context.Context, id znp.NvItemID, offset uint8, data []byte) error {
	return nil
}

func (m *mockZNPClient) NvReadAll(ctx context.Context, id znp.NvItemID) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockZNPClient) NvWriteAll(ctx context.Context, id znp.NvItemID, data []byte) error {
	return nil
}

func (m *mockZNPClient) NvItemInit(ctx context.Context, id znp.NvItemID, itemLen uint16, initData []byte) error {
	return nil
}

// Address Manager
func (m *mockZNPClient) ReadAddrMgrTable(ctx context.Context) ([]znp.AddrMgrEntry, error) {
	return []znp.AddrMgrEntry{}, nil
}

func (m *mockZNPClient) DeleteAddrMgrEntry(ctx context.Context, ieeeAddr [8]byte) (bool, error) {
	return true, nil
}

// Device names
func (m *mockZNPClient) SetDeviceName(ctx context.Context, ieeeAddr [8]byte, name, comment string) error {
	return nil
}

func (m *mockZNPClient) GetDeviceName(ctx context.Context, ieeeAddr [8]byte) (*znp.DeviceNameEntry, error) {
	return &znp.DeviceNameEntry{}, nil
}

func (m *mockZNPClient) DeleteDeviceName(ctx context.Context, ieeeAddr [8]byte) error {
	return nil
}

func (m *mockZNPClient) ReadDeviceNameTable(ctx context.Context) (*znp.DeviceNameTable, error) {
	return &znp.DeviceNameTable{}, nil
}

// Event callbacks
func (m *mockZNPClient) OnDeviceJoin(handler func(*znp.TcDeviceInd))       {}
func (m *mockZNPClient) OnDeviceLeave(handler func(*znp.DeviceLeave))      {}
func (m *mockZNPClient) OnDeviceAnnounce(handler func(*znp.DeviceAnnounce)) {}

// TestZNPClientInterface verifies that the mockZNPClient implements ZNPClient.
// This is a compile-time check to ensure the mock stays in sync with the interface.
func TestZNPClientInterface(t *testing.T) {
	// This test will fail to compile if mockZNPClient doesn't implement ZNPClient
	var _ ZNPClient = (*mockZNPClient)(nil)

	// Also verify that *znp.ZNP implements ZNPClient
	var _ ZNPClient = (*znp.ZNP)(nil)

	t.Log("ZNPClient interface correctly implemented by both mock and *znp.ZNP")
}

// TestMockUsage demonstrates how to use the mock in tests.
func TestMockUsage(t *testing.T) {
	// Create a mock ZNP client
	mock := &mockZNPClient{
		versionInfo: &znp.VersionInfo{
			TransportRev: 2,
			Product:      1,
			MajorRel:     2,
			MinorRel:     7,
			MaintRel:     1,
		},
	}

	// Create an adapter using the mock
	// In a real test, you would inject the mock through the adapter's constructor
	// or by directly setting the adapter's znp field (requires exposing it or using reflection)
	adapter := &Adapter{
		znp: mock,
	}

	// Verify the mock is used
	if adapter.znp == nil {
		t.Fatal("adapter znp client is nil")
	}

	// Call a method that uses the mock
	ctx := context.Background()
	_, err := adapter.znp.Ping(ctx)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Verify mock interactions (in real tests, use a mocking library with expectations)
	if !mock.openCalled && !mock.closeCalled {
		// This is expected in this simple test since we didn't call Open/Close
		t.Log("Open/Close not called, as expected")
	}
}

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
func (m *mockZNPClient) Open(_ context.Context) error {
	m.openCalled = true
	return nil
}

func (m *mockZNPClient) Close() error {
	m.closeCalled = true
	return nil
}

// System commands
func (m *mockZNPClient) Ping(_ context.Context) (*znp.PingCapabilities, error) {
	return &znp.PingCapabilities{Capabilities: 0x01}, nil
}

func (m *mockZNPClient) Version(_ context.Context) (*znp.VersionInfo, error) {
	return m.versionInfo, nil
}

func (m *mockZNPClient) Reset(_ context.Context, _ znp.ResetType) (*znp.ResetIndication, error) {
	return &znp.ResetIndication{}, nil
}

func (m *mockZNPClient) GetDeviceInfo(_ context.Context) (*znp.DeviceInfo, error) {
	return &znp.DeviceInfo{}, nil
}

func (m *mockZNPClient) StartupFromApp(_ context.Context, _ uint16) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) SetTxPower(_ context.Context, power int8) (int8, error) {
	return power, nil
}

// Network management
func (m *mockZNPClient) ExtNwkInfo(_ context.Context) (*znp.ExtNetworkInfo, error) {
	return &znp.ExtNetworkInfo{}, nil
}

func (m *mockZNPClient) BdbSetChannel(_ context.Context, _ bool, _ uint32) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) BdbStartCommissioning(_ context.Context, _ znp.BdbCommissioningMode) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) WaitForStateChange(_ context.Context, _ time.Duration) (znp.DevState, error) {
	return 0, nil
}

// AF layer.
func (m *mockZNPClient) AfRegister(_ context.Context, _ znp.EndpointConfig) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) AfDelete(_ context.Context, _ uint8) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) AfDataRequest(_ context.Context, _ znp.DataRequest) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) WaitForDataConfirm(_ context.Context, _ uint8, _ time.Duration) (*znp.DataConfirm, error) {
	return &znp.DataConfirm{}, nil
}

func (m *mockZNPClient) WaitForIncomingMsg(_ context.Context, _ uint16, _ uint16, _ time.Duration) (*znp.IncomingMessage, error) {
	return &znp.IncomingMessage{}, nil
}

// ZDO commands.
func (m *mockZNPClient) MgmtPermitJoinReq(_ context.Context, _ uint8) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) MgmtLeaveReq(_ context.Context, _ uint16, _ [8]byte, _, _ bool) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) MgmtNwkUpdateReq(_ context.Context, _ uint16, _ uint8, _ uint32, _ uint8, _ uint8, _ uint16) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) ActiveEpReq(_ context.Context, _ uint16) (*znp.ActiveEndpoints, error) {
	return &znp.ActiveEndpoints{}, nil
}

func (m *mockZNPClient) SimpleDescReq(_ context.Context, _ uint16, _ uint8) (*znp.SimpleDescriptor, error) {
	return &znp.SimpleDescriptor{}, nil
}

func (m *mockZNPClient) NodeDescReq(_ context.Context, _ uint16) (*znp.NodeDescriptor, error) {
	return &znp.NodeDescriptor{}, nil
}

func (m *mockZNPClient) IeeeAddrReq(_ context.Context, _ uint16) ([8]byte, error) {
	return [8]byte{}, nil
}

func (m *mockZNPClient) BindReq(_ context.Context, _ uint16, _ [8]byte, _ uint8, _ uint16, _ [8]byte, _ uint8) (uint8, error) {
	return 0, nil
}

func (m *mockZNPClient) UnbindReq(_ context.Context, _ uint16, _ [8]byte, _ uint8, _ uint16, _ [8]byte, _ uint8) (uint8, error) {
	return 0, nil
}

// Network topology.
func (m *mockZNPClient) GetAllNeighbors(_ context.Context, _ uint16) ([]znp.NeighborEntry, error) {
	return []znp.NeighborEntry{}, nil
}

func (m *mockZNPClient) GetAllBindings(_ context.Context, _ uint16) ([]znp.BindingEntry, error) {
	return []znp.BindingEntry{}, nil
}

func (m *mockZNPClient) GetAllRoutes(_ context.Context, _ uint16) ([]znp.RoutingEntry, error) {
	return []znp.RoutingEntry{}, nil
}

func (m *mockZNPClient) NvLength(_ context.Context, _ znp.NvItemID) (uint16, error) {
	return 0, nil
}

func (m *mockZNPClient) NvRead(_ context.Context, _ znp.NvItemID, _ uint8) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockZNPClient) NvWrite(_ context.Context, _ znp.NvItemID, _ uint8, _ []byte) error {
	return nil
}

func (m *mockZNPClient) NvReadAll(_ context.Context, _ znp.NvItemID) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockZNPClient) NvWriteAll(_ context.Context, _ znp.NvItemID, _ []byte) error {
	return nil
}

func (m *mockZNPClient) NvItemInit(_ context.Context, _ znp.NvItemID, _ uint16, _ []byte) error {
	return nil
}

// Address Manager
func (m *mockZNPClient) ReadAddrMgrTable(_ context.Context) ([]znp.AddrMgrEntry, error) {
	return []znp.AddrMgrEntry{}, nil
}

func (m *mockZNPClient) DeleteAddrMgrEntry(_ context.Context, _ [8]byte) (bool, error) {
	return true, nil
}

// Device names
func (m *mockZNPClient) SetDeviceName(_ context.Context, _ [8]byte, _, _ string) error {
	return nil
}

func (m *mockZNPClient) GetDeviceName(_ context.Context, _ [8]byte) (*znp.DeviceNameEntry, error) {
	return &znp.DeviceNameEntry{}, nil
}

func (m *mockZNPClient) DeleteDeviceName(_ context.Context, _ [8]byte) error {
	return nil
}

func (m *mockZNPClient) ReadDeviceNameTable(_ context.Context) (*znp.DeviceNameTable, error) {
	return &znp.DeviceNameTable{}, nil
}

// Event callbacks
func (m *mockZNPClient) OnDeviceJoin(_ func(*znp.TcDeviceInd))        {}
func (m *mockZNPClient) OnDeviceLeave(_ func(*znp.DeviceLeave))       {}
func (m *mockZNPClient) OnDeviceAnnounce(_ func(*znp.DeviceAnnounce)) {}

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

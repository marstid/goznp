package adapter

import (
	"context"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
	"github.com/marstid/goznp/pkg/znp"
)

// mockZNP implements a minimal ZNP interface for testing.
// It allows tests to control responses and errors for various ZNP operations.
type mockZNP struct {
	pingResp    *znp.PingCapabilities
	pingErr     error
	versionResp *znp.VersionInfo
	versionErr  error
	resetResp   *znp.ResetIndication
	resetErr    error
	afRegStatus uint8
	afRegErr    error
	afDelStatus uint8
	afDelErr    error
	startupResp uint8
	startupErr  error

	// Callback handlers
	onDeviceJoinHandler  func(*znp.TcDeviceInd)
	onDeviceLeaveHandler func(*znp.DeviceLeave)
	onErrorHandler       func(error)

	// Track method calls for verification
	openCalled        bool
	closeCalled       bool
	afRegisterCalls   []znp.EndpointConfig
	afDeleteCalls     []uint8
	startupFromAppArg uint16
}

func (m *mockZNP) Open(ctx context.Context) error {
	m.openCalled = true
	return nil
}

func (m *mockZNP) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockZNP) Ping(ctx context.Context) (*znp.PingCapabilities, error) {
	return m.pingResp, m.pingErr
}

func (m *mockZNP) Version(ctx context.Context) (*znp.VersionInfo, error) {
	return m.versionResp, m.versionErr
}

func (m *mockZNP) Reset(ctx context.Context, resetType znp.ResetType) (*znp.ResetIndication, error) {
	return m.resetResp, m.resetErr
}

func (m *mockZNP) AfRegister(ctx context.Context, config znp.EndpointConfig) (uint8, error) {
	m.afRegisterCalls = append(m.afRegisterCalls, config)
	return m.afRegStatus, m.afRegErr
}

func (m *mockZNP) AfDelete(ctx context.Context, endpoint uint8) (uint8, error) {
	m.afDeleteCalls = append(m.afDeleteCalls, endpoint)
	return m.afDelStatus, m.afDelErr
}

func (m *mockZNP) StartupFromApp(ctx context.Context, startDelay uint16) (uint8, error) {
	m.startupFromAppArg = startDelay
	return m.startupResp, m.startupErr
}

func (m *mockZNP) OnDeviceJoin(handler func(*znp.TcDeviceInd)) {
	m.onDeviceJoinHandler = handler
}

func (m *mockZNP) OnDeviceLeave(handler func(*znp.DeviceLeave)) {
	m.onDeviceLeaveHandler = handler
}

func (m *mockZNP) OnError(handler func(error)) {
	m.onErrorHandler = handler
}

func (m *mockZNP) OnFrame(handler func(*unpi.Frame)) {
	// Not used in current tests
}

// Helper to create a successful mock with reasonable defaults
func newSuccessfulMockZNP() *mockZNP {
	return &mockZNP{
		pingResp: &znp.PingCapabilities{
			Capabilities: 0x0001,
		},
		versionResp: &znp.VersionInfo{
			TransportRev: 2,
			Product:      1,
			MajorRel:     3,
			MinorRel:     0,
			MaintRel:     1,
		},
		resetResp: &znp.ResetIndication{
			Reason:       0,
			TransportRev: 2,
			Product:      1,
			MajorRel:     3,
			MinorRel:     0,
			MaintRel:     1,
		},
		afRegStatus: StatusSuccess,
		afDelStatus: StatusSuccess,
		startupResp: 0, // Success
	}
}

// mockPort implements a minimal serial.Port interface for testing
type mockPort struct {
	closeCalled bool
}

func (m *mockPort) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (m *mockPort) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockPort) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockPort) Flush() error {
	return nil
}

func (m *mockPort) SetReadDeadline(t time.Time) error {
	return nil
}

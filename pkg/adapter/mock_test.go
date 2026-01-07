package adapter

import (
	"context"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
	"github.com/marstid/goznp/pkg/znp"
)

// mockZNP implements a minimal ZNP interface for testing.
// It allows tests to control responses and errors for various ZNP operations.
//
//nolint:unused // Test helper for future use.
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
	onDeviceJoinHandler     func(*znp.TcDeviceInd)
	onDeviceLeaveHandler    func(*znp.DeviceLeave)
	onDeviceAnnounceHandler func(*znp.DeviceAnnounce)
	onHandler               func(error)

	// Track method calls for verification
	openCalled        bool
	closeCalled       bool
	afRegisterCalls   []znp.EndpointConfig
	afDeleteCalls     []uint8
	startupFromAppArg uint16
}

//nolint:unused // Test helper for future use
func (m *mockZNP) Open(_ context.Context) error {
	m.openCalled = true
	return nil
}

//nolint:unused // Test helper for future use.
func (m *mockZNP) Close() error {
	m.closeCalled = true
	return nil
}

//nolint:unused // Test helper for future use
func (m *mockZNP) Ping(_ context.Context) (*znp.PingCapabilities, error) {
	return m.pingResp, m.pingErr
}

//nolint:unused // Test helper for future use
func (m *mockZNP) Version(_ context.Context) (*znp.VersionInfo, error) {
	return m.versionResp, m.versionErr
}

//nolint:unused // Test helper for future use
func (m *mockZNP) Reset(_ context.Context, _ znp.ResetType) (*znp.ResetIndication, error) {
	return m.resetResp, m.resetErr
}

//nolint:unused // Test helper for future use
func (m *mockZNP) AfRegister(_ context.Context, config znp.EndpointConfig) (uint8, error) {
	m.afRegisterCalls = append(m.afRegisterCalls, config)
	return m.afRegStatus, m.afRegErr
}

//nolint:unused // Test helper for future use
func (m *mockZNP) AfDelete(_ context.Context, endpoint uint8) (uint8, error) {
	m.afDeleteCalls = append(m.afDeleteCalls, endpoint)
	return m.afDelStatus, m.afDelErr
}

//nolint:unused // Test helper for future use
func (m *mockZNP) StartupFromApp(_ context.Context, startDelay uint16) (uint8, error) {
	m.startupFromAppArg = startDelay
	return m.startupResp, m.startupErr
}

//nolint:unused // Test helper for future use.
func (m *mockZNP) OnDeviceJoin(handler func(*znp.TcDeviceInd)) {
	m.onDeviceJoinHandler = handler
}

//nolint:unused // Test helper for future use.
func (m *mockZNP) OnDeviceLeave(handler func(*znp.DeviceLeave)) {
	m.onDeviceLeaveHandler = handler
}

//nolint:unused // Test helper for future use.
func (m *mockZNP) OnError(handler func(error)) {
	m.onHandler = handler
}

//nolint:unused // Test helper for future use.
func (m *mockZNP) OnDeviceAnnounce(handler func(*znp.DeviceAnnounce)) {
	m.onDeviceAnnounceHandler = handler
}

//nolint:unused // Test helper for future use
func (m *mockZNP) OnFrame(_ func(*unpi.Frame)) {
	// Not used in current tests
}

// Helper to create a successful mock with reasonable defaults
//
//nolint:unused // Test helper for future use.
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
//
//nolint:unused // Test helper for future use.
type mockPort struct {
	closeCalled bool
}

//nolint:unused // Test helper for future use.
func (m *mockPort) Read(_ []byte) (n int, err error) {
	return 0, nil
}

//nolint:unused // Test helper for future use.
func (m *mockPort) Write(p []byte) (n int, err error) {
	return len(p), nil
}

//nolint:unused // Test helper for future use.
func (m *mockPort) Close() error {
	m.closeCalled = true
	return nil
}

//nolint:unused // Test helper for future use.
func (m *mockPort) Flush() error {
	return nil
}

//nolint:unused // Test helper for future use.
func (m *mockPort) SetReadDeadline(_ time.Time) error {
	return nil
}

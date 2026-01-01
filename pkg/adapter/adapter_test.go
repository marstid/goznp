package adapter

import (
	"testing"
	"time"
)

// TestNew verifies that New creates an adapter with the given options.
func TestNew(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
	}{
		{
			name: "no options",
			opts: nil,
		},
		{
			name: "with serial path",
			opts: []Option{WithSerialPath("/dev/ttyUSB0")},
		},
		{
			name: "with multiple options",
			opts: []Option{
				WithSerialPath("/dev/test"),
				WithBaudRate(115200),
				WithPingRetries(5),
				WithPingTimeout(500 * time.Millisecond),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.opts...)
			if a == nil {
				t.Fatal("New returned nil")
			}
			if a.IsOpen() {
				t.Error("new adapter should not be open")
			}
			if a.dedupe == nil {
				t.Error("dedupe cache should be initialized")
			}
		})
	}
}

// TestIsOpen verifies that IsOpen returns correct state.
func TestIsOpen(t *testing.T) {
	a := New()

	// Initially should not be open
	if a.IsOpen() {
		t.Error("new adapter should not be open")
	}

	// Manually set open state for testing
	a.mu.Lock()
	a.isOpen = true
	a.mu.Unlock()

	if !a.IsOpen() {
		t.Error("adapter should be open after setting isOpen=true")
	}

	// Set back to closed
	a.mu.Lock()
	a.isOpen = false
	a.mu.Unlock()

	if a.IsOpen() {
		t.Error("adapter should not be open after setting isOpen=false")
	}
}

// TestWithSerialPath verifies that WithSerialPath option works correctly.
func TestWithSerialPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedPath string
	}{
		{
			name:         "standard USB path",
			path:         "/dev/ttyUSB0",
			expectedPath: "/dev/ttyUSB0",
		},
		{
			name:         "ACM path",
			path:         "/dev/ttyACM0",
			expectedPath: "/dev/ttyACM0",
		},
		{
			name:         "custom path",
			path:         "/dev/test",
			expectedPath: "/dev/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithSerialPath(tt.path))
			if a.options.SerialConfig.Path != tt.expectedPath {
				t.Errorf("expected path %s, got %s", tt.expectedPath, a.options.SerialConfig.Path)
			}
		})
	}
}

// TestWithBaudRate verifies that WithBaudRate option works correctly.
func TestWithBaudRate(t *testing.T) {
	tests := []struct {
		name         string
		rate         int
		expectedRate int
	}{
		{
			name:         "115200 baud",
			rate:         115200,
			expectedRate: 115200,
		},
		{
			name:         "57600 baud",
			rate:         57600,
			expectedRate: 57600,
		},
		{
			name:         "9600 baud",
			rate:         9600,
			expectedRate: 9600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithBaudRate(tt.rate))
			if a.options.SerialConfig.BaudRate != tt.expectedRate {
				t.Errorf("expected baud rate %d, got %d", tt.expectedRate, a.options.SerialConfig.BaudRate)
			}
		})
	}
}

// TestWithPingRetries verifies that WithPingRetries option works correctly.
func TestWithPingRetries(t *testing.T) {
	tests := []struct {
		name            string
		retries         int
		expectedRetries int
	}{
		{
			name:            "3 retries",
			retries:         3,
			expectedRetries: 3,
		},
		{
			name:            "5 retries",
			retries:         5,
			expectedRetries: 5,
		},
		{
			name:            "10 retries",
			retries:         10,
			expectedRetries: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithPingRetries(tt.retries))
			if a.options.PingRetries != tt.expectedRetries {
				t.Errorf("expected %d retries, got %d", tt.expectedRetries, a.options.PingRetries)
			}
		})
	}
}

// TestWithPingTimeout verifies that WithPingTimeout option works correctly.
func TestWithPingTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "250ms timeout",
			timeout:         250 * time.Millisecond,
			expectedTimeout: 250 * time.Millisecond,
		},
		{
			name:            "500ms timeout",
			timeout:         500 * time.Millisecond,
			expectedTimeout: 500 * time.Millisecond,
		},
		{
			name:            "1s timeout",
			timeout:         time.Second,
			expectedTimeout: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithPingTimeout(tt.timeout))
			if a.options.PingTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, a.options.PingTimeout)
			}
		})
	}
}

// TestWithResetTimeout verifies that WithResetTimeout option works correctly.
func TestWithResetTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "30s timeout",
			timeout:         30 * time.Second,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "60s timeout",
			timeout:         60 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "10s timeout",
			timeout:         10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithResetTimeout(tt.timeout))
			if a.options.ResetTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, a.options.ResetTimeout)
			}
		})
	}
}

// TestWithRTSCTSFlow verifies that WithRTSCTSFlow option works correctly.
func TestWithRTSCTSFlow(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "RTS/CTS enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "RTS/CTS disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithRTSCTSFlow(tt.enabled))
			if a.options.SerialConfig.RTSCTSFlow != tt.expected {
				t.Errorf("expected RTSCTSFlow %v, got %v", tt.expected, a.options.SerialConfig.RTSCTSFlow)
			}
		})
	}
}

// TestCombinedOptions verifies that multiple options can be applied together.
func TestCombinedOptions(t *testing.T) {
	a := New(
		WithSerialPath("/dev/ttyUSB0"),
		WithBaudRate(57600),
		WithPingRetries(5),
		WithPingTimeout(500*time.Millisecond),
		WithResetTimeout(45*time.Second),
		WithRTSCTSFlow(true),
	)

	if a.options.SerialConfig.Path != "/dev/ttyUSB0" {
		t.Errorf("expected path /dev/ttyUSB0, got %s", a.options.SerialConfig.Path)
	}
	if a.options.SerialConfig.BaudRate != 57600 {
		t.Errorf("expected baud rate 57600, got %d", a.options.SerialConfig.BaudRate)
	}
	if a.options.PingRetries != 5 {
		t.Errorf("expected 5 retries, got %d", a.options.PingRetries)
	}
	if a.options.PingTimeout != 500*time.Millisecond {
		t.Errorf("expected timeout 500ms, got %v", a.options.PingTimeout)
	}
	if a.options.ResetTimeout != 45*time.Second {
		t.Errorf("expected reset timeout 45s, got %v", a.options.ResetTimeout)
	}
	if !a.options.SerialConfig.RTSCTSFlow {
		t.Error("expected RTS/CTS flow to be enabled")
	}
}

// TestVersion verifies that Version returns nil when adapter is not open.
func TestVersion(t *testing.T) {
	a := New()

	version := a.Version()
	if version != nil {
		t.Error("expected nil version when adapter is not open")
	}
}

// TestDefaultOptions verifies that default options are reasonable.
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.PingRetries <= 0 {
		t.Error("default ping retries should be positive")
	}
	if opts.PingTimeout <= 0 {
		t.Error("default ping timeout should be positive")
	}
	if opts.ResetTimeout <= 0 {
		t.Error("default reset timeout should be positive")
	}
	if opts.SerialConfig.BaudRate <= 0 {
		t.Error("default baud rate should be positive")
	}
}

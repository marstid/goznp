package adapter

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"
)

// TestLogger implements Logger for testing
type TestLogger struct {
	debug buf
	info  buf
	warn  buf
	error buf
}

type buf struct{ bytes.Buffer }

func (b *buf) String() string { return b.Buffer.String() }
func (b *buf) Writef(format string, args ...interface{}) {
	fmt.Fprintf(&b.Buffer, format, args...)
	b.Buffer.WriteByte('\n')
}

func (l *TestLogger) Debugf(format string, args ...interface{}) { l.debug.Writef(format, args...) }
func (l *TestLogger) Infof(format string, args ...interface{})  { l.info.Writef(format, args...) }
func (l *TestLogger) Warnf(format string, args ...interface{})  { l.warn.Writef(format, args...) }
func (l *TestLogger) Errorf(format string, args ...interface{}) { l.error.Writef(format, args...) }

func TestDefaultLogger(t *testing.T) {
	logger := &DefaultLogger{}

	// Should not panic
	logger.Debugf("test %s", "debug")
	logger.Infof("test %s", "info")
	logger.Warnf("test %s", "warn")
	logger.Errorf("test %s", "error")
}

func TestWithLogger(t *testing.T) {
	customLogger := &TestLogger{}
	a := New(WithLogger(customLogger))

	if a.options.Logger != customLogger {
		t.Error("WithLogger() did not set custom logger")
	}
}

func TestWithZCLRetryAttempts(t *testing.T) {
	tests := []struct {
		name             string
		attempts         int
		expectedAttempts int
	}{
		{
			name:             "zero retries",
			attempts:         0,
			expectedAttempts: 0,
		},
		{
			name:             "three retries",
			attempts:         3,
			expectedAttempts: 3,
		},
		{
			name:             "negative retries (should be ignored)",
			attempts:         -1,
			expectedAttempts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithZCLRetryAttempts(tt.attempts))
			if a.options.ZCLRetryAttempts != tt.expectedAttempts {
				t.Errorf("WithZCLRetryAttempts(%d) = %d, want %d",
					tt.attempts, a.options.ZCLRetryAttempts, tt.expectedAttempts)
			}
		})
	}
}

func TestWithZCLRetryDelay(t *testing.T) {
	tests := []struct {
		name          string
		delay         time.Duration
		expectedDelay time.Duration
	}{
		{
			name:          "100ms delay",
			delay:         100 * time.Millisecond,
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name:          "1 second delay",
			delay:         1 * time.Second,
			expectedDelay: 1 * time.Second,
		},
		{
			name:          "zero delay (should be ignored)",
			delay:         0,
			expectedDelay: 100 * time.Millisecond, // Default value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithZCLRetryDelay(tt.delay))
			if a.options.ZCLRetryDelay != tt.expectedDelay {
				t.Errorf("WithZCLRetryDelay(%v) = %v, want %v",
					tt.delay, a.options.ZCLRetryDelay, tt.expectedDelay)
			}
		})
	}
}

func TestWithZDOInitDelay(t *testing.T) {
	tests := []struct {
		name          string
		delay         time.Duration
		expectedDelay time.Duration
	}{
		{
			name:          "200ms delay",
			delay:         200 * time.Millisecond,
			expectedDelay: 200 * time.Millisecond,
		},
		{
			name:          "0 delay (disabled)",
			delay:         0,
			expectedDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(WithZDOInitDelay(tt.delay))
			if a.options.ZDOInitDelay != tt.expectedDelay {
				t.Errorf("WithZDOInitDelay(%v) = %v, want %v",
					tt.delay, a.options.ZDOInitDelay, tt.expectedDelay)
			}
		})
	}
}

func TestCheckOpen(t *testing.T) {
	a := New()

	// Test with closed adapter
	ctx := context.Background()
	err := a.checkOpen(ctx)
	if err != ErrNotOpen {
		t.Errorf("checkOpen() on closed adapter = %v, want ErrNotOpen", err)
	}

	// Test with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = a.checkOpen(canceledCtx)
	if err == nil {
		t.Error("checkOpen() with canceled context should return error")
	}
}

func TestConstants(t *testing.T) {
	// Verify that constants have expected values
	tests := []struct {
		name   string
		value  uint8
		expect uint8
	}{
		{"StatusSuccess", StatusSuccess, 0x00},
		{"StatusAlreadyRegistered", StatusAlreadyRegistered, 0xB8},
		{"ReportingDirectionDeviceToCoordinator", ReportingDirectionDeviceToCoordinator, 0x00},
		{"AddressModeGroup", AddressModeGroup, 0x01},
		{"AddressModeIEEE", AddressModeIEEE, 0x03},
		{"LogicalTypeCoordinator", LogicalTypeCoordinator, 0x00},
		{"StartupOptionClearState", StartupOptionClearState, 0x03},
		{"ZdoDirectCbEnabled", ZdoDirectCbEnabled, 0x01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expect {
				t.Errorf("%s = 0x%02X, want 0x%02X", tt.name, tt.value, tt.expect)
			}
		})
	}
}

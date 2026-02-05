// Package adapter provides a high-level API for Z-Stack Zigbee adapters.
package adapter

import (
	"time"

	"github.com/marstid/goznp/pkg/serial"
)

// Options configures the adapter.
type Options struct {
	SerialConfig     serial.Config
	PingRetries      int
	PingTimeout      time.Duration
	ResetTimeout     time.Duration
	ZDOInitDelay     time.Duration // Delay after StartupFromApp to allow ZDO layer initialization
	Logger           Logger        // Logger for adapter events and warnings
	ZCLRetryAttempts int           // Number of retry attempts for ZCL requests (excluding initial attempt)
	ZCLRetryDelay    time.Duration // Initial delay before first retry, with exponential backoff
}

// DefaultOptions returns default adapter options.
func DefaultOptions() Options {
	return Options{
		SerialConfig:     serial.DefaultConfig(),
		PingRetries:      3,
		PingTimeout:      250 * time.Millisecond,
		ResetTimeout:     30 * time.Second,
		ZDOInitDelay:     200 * time.Millisecond,
		Logger:           &DefaultLogger{},
		ZCLRetryAttempts: 0, // No retry by default
		ZCLRetryDelay:    100 * time.Millisecond,
	}
}

// Option is a functional option for configuring the adapter.
type Option func(*Options)

// WithSerialPath sets the serial port path.
func WithSerialPath(path string) Option {
	return func(o *Options) {
		o.SerialConfig.Path = path
	}
}

// WithBaudRate sets the serial port baud rate.
func WithBaudRate(rate int) Option {
	return func(o *Options) {
		o.SerialConfig.BaudRate = rate
	}
}

// WithPingRetries sets the number of ping retry attempts.
func WithPingRetries(retries int) Option {
	return func(o *Options) {
		o.PingRetries = retries
	}
}

// WithPingTimeout sets the timeout for ping operations.
func WithPingTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.PingTimeout = timeout
	}
}

// WithResetTimeout sets the timeout for reset operations.
func WithResetTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.ResetTimeout = timeout
	}
}

// WithRTSCTSFlow enables or disables RTS/CTS hardware flow control.
func WithRTSCTSFlow(enabled bool) Option {
	return func(o *Options) {
		o.SerialConfig.RTSCTSFlow = enabled
	}
}

// WithZDOInitDelay sets the delay after StartupFromApp to allow ZDO layer initialization.
// Default is 200ms. Set to 0 to disable the delay if you're sure initialization is not needed.
func WithZDOInitDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.ZDOInitDelay = delay
	}
}

// WithLogger sets the logger for adapter events and warnings.
// By default, a no-op logger is used which discards all logs.
func WithLogger(logger Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithZCLRetryAttempts sets the number of retry attempts for ZCL requests.
// The value is the number of retries (additional attempts after the initial attempt).
// For example, 3 means: initial attempt + 3 retries = 4 total attempts.
// Default is 0 (no retry).
func WithZCLRetryAttempts(attempts int) Option {
	return func(o *Options) {
		if attempts >= 0 {
			o.ZCLRetryAttempts = attempts
		}
	}
}

// WithZCLRetryDelay sets the initial delay before the first ZCL retry.
// Subsequent retries use exponential backoff (delay, 2x delay, 4x delay, etc.).
// Default is 100ms.
func WithZCLRetryDelay(delay time.Duration) Option {
	return func(o *Options) {
		if delay > 0 {
			o.ZCLRetryDelay = delay
		}
	}
}

// Package adapter provides a high-level API for Z-Stack Zigbee adapters.
package adapter

import (
	"time"

	"github.com/marstid/goznp/pkg/serial"
)

// Options configures the adapter.
type Options struct {
	SerialConfig serial.Config
	PingRetries  int
	PingTimeout  time.Duration
	ResetTimeout time.Duration
}

// DefaultOptions returns default adapter options.
func DefaultOptions() Options {
	return Options{
		SerialConfig: serial.DefaultConfig(),
		PingRetries:  3,
		PingTimeout:  250 * time.Millisecond,
		ResetTimeout: 30 * time.Second,
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

package daemon

import (
	"os"
	"strconv"
	"time"

	"github.com/marstid/goznp/pkg/serial"
)

// Config holds configuration for goznpd.
// All values are loaded from environment variables with reasonable defaults.
type Config struct {
	// Adapter configuration
	Port     string // GOZNP_PORT (required)
	BaudRate int    // GOZNP_BAUD_RATE (default: 115200)

	// HTTP server configuration
	HTTPAddr string // GOZNP_HTTP_ADDR (default: :8080)

	// Logging configuration
	LogLevel string // GOZNP_LOG_LEVEL (default: info)

	// Device polling configuration
	PollInterval time.Duration // GOZNP_POLL_INTERVAL (default: 5m)

	// Message listener configuration
	ListenTimeout time.Duration // GOZNP_LISTEN_TIMEOUT (default: 100ms)

	// Network configuration
	PermitJoinDuration uint8 // GOZNP_PERMIT_JOIN_DURATION (default: 255 = always)
}

// Load loads configuration from environment variables.
// Returns an error if required variables are missing.
func Load() (*Config, error) {
	cfg := &Config{
		BaudRate:           115200,
		HTTPAddr:           ":8080",
		LogLevel:           "info",
		PollInterval:       5 * time.Minute,
		ListenTimeout:      100 * time.Millisecond,
		PermitJoinDuration: 255,
	}

	// Required: Port path
	if port := os.Getenv("GOZNP_PORT"); port != "" {
		cfg.Port = port
	} else {
		return nil, ErrPortRequired
	}

	// Optional: Baud rate
	if baud := os.Getenv("GOZNP_BAUD_RATE"); baud != "" {
		if b, err := strconv.Atoi(baud); err == nil {
			cfg.BaudRate = b
		}
	}

	// Optional: HTTP bind address
	if addr := os.Getenv("GOZNP_HTTP_ADDR"); addr != "" {
		cfg.HTTPAddr = addr
	}

	// Optional: Log level
	if level := os.Getenv("GOZNP_LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	// Optional: Poll interval
	if poll := os.Getenv("GOZNP_POLL_INTERVAL"); poll != "" {
		if d, err := time.ParseDuration(poll); err == nil {
			cfg.PollInterval = d
		}
	}

	// Optional: Listen timeout
	if timeout := os.Getenv("GOZNP_LISTEN_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.ListenTimeout = d
		}
	}

	// Optional: Permit join duration
	if duration := os.Getenv("GOZNP_PERMIT_JOIN_DURATION"); duration != "" {
		if d, err := strconv.ParseUint(duration, 10, 8); err == nil {
			cfg.PermitJoinDuration = uint8(d)
		}
	}

	return cfg, nil
}

// ToSerialConfig converts Config to serial.Config.
func (c *Config) ToSerialConfig() serial.Config {
	return serial.Config{
		Path:        c.Port,
		BaudRate:    c.BaudRate,
		RTSCTSFlow:  false, // Default to no hardware flow control
		ReadTimeout: 5 * time.Second,
	}
}

var (
	// ErrPortRequired is returned when GOZNP_PORT is not set.
	ErrPortRequired = ConfigError("GOZNP_PORT environment variable is required")
)

// ConfigError represents a configuration error.
type ConfigError string

func (e ConfigError) Error() string {
	return string(e)
}

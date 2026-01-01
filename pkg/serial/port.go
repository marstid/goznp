// Package serial provides serial port abstraction for Z-Stack communication.
package serial

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// ErrInvalidPortPath indicates the port path is invalid or potentially unsafe.
var ErrInvalidPortPath = errors.New("invalid port path")

// Port interface for serial port operations.
// Extends io.ReadWriteCloser with additional serial port specific methods.
type Port interface {
	io.ReadWriteCloser
	SetReadTimeout(timeout time.Duration) error
	SetDTR(dtr bool) error
	SetRTS(rts bool) error
	ResetInputBuffer() error
}

// Config holds serial port configuration.
type Config struct {
	Path        string        // Device path (e.g., /dev/ttyUSB0, COM3)
	BaudRate    int           // Baud rate (default 115200)
	ReadTimeout time.Duration // Read timeout (default 100ms)
	RTSCTSFlow  bool          // Enable RTS/CTS hardware flow control
}

// USBPortInfo contains detailed information about a USB serial port.
type USBPortInfo struct {
	Name         string // Port name (e.g., /dev/ttyUSB0)
	IsUSB        bool   // Whether this is a USB port
	VID          string // USB Vendor ID (hex string)
	PID          string // USB Product ID (hex string)
	SerialNumber string // USB serial number
	Product      string // Product description
}

// serialPort wraps go.bug.st/serial.Port to implement our Port interface.
type serialPort struct {
	port        serial.Port
	readTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults for Z-Stack communication.
func DefaultConfig() Config {
	return Config{
		BaudRate:    115200,
		ReadTimeout: 100 * time.Millisecond,
		RTSCTSFlow:  false,
	}
}

// Open opens a serial port with the given configuration.
// It validates the port path to prevent path traversal and ensure it's a valid serial port path.
func Open(cfg Config) (Port, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("serial: port path is required")
	}

	// Validate port path for security
	if err := validatePortPath(cfg.Path); err != nil {
		return nil, fmt.Errorf("serial: %w: %s", ErrInvalidPortPath, err)
	}

	// Apply defaults if not set
	if cfg.BaudRate == 0 {
		cfg.BaudRate = 115200
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 100 * time.Millisecond
	}

	mode := &serial.Mode{
		BaudRate: cfg.BaudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(cfg.Path, mode)
	if err != nil {
		return nil, fmt.Errorf("serial: failed to open port %s: %w", cfg.Path, err)
	}

	// Set read timeout
	if err := port.SetReadTimeout(cfg.ReadTimeout); err != nil {
		port.Close()
		return nil, fmt.Errorf("serial: failed to set read timeout: %w", err)
	}

	sp := &serialPort{
		port:        port,
		readTimeout: cfg.ReadTimeout,
	}

	// Set DTR and RTS high by default
	if err := sp.SetDTR(true); err != nil {
		port.Close()
		return nil, fmt.Errorf("serial: failed to set DTR: %w", err)
	}
	if err := sp.SetRTS(true); err != nil {
		port.Close()
		return nil, fmt.Errorf("serial: failed to set RTS: %w", err)
	}

	return sp, nil
}

// Read reads data from the serial port.
func (p *serialPort) Read(b []byte) (int, error) {
	return p.port.Read(b)
}

// Write writes data to the serial port.
func (p *serialPort) Write(b []byte) (int, error) {
	return p.port.Write(b)
}

// Close closes the serial port.
func (p *serialPort) Close() error {
	if err := p.port.Close(); err != nil {
		return fmt.Errorf("serial: failed to close port: %w", err)
	}
	return nil
}

// SetReadTimeout sets the read timeout for the serial port.
func (p *serialPort) SetReadTimeout(timeout time.Duration) error {
	if err := p.port.SetReadTimeout(timeout); err != nil {
		return fmt.Errorf("serial: failed to set read timeout: %w", err)
	}
	p.readTimeout = timeout
	return nil
}

// SetDTR sets the Data Terminal Ready signal.
func (p *serialPort) SetDTR(dtr bool) error {
	if err := p.port.SetDTR(dtr); err != nil {
		return fmt.Errorf("serial: failed to set DTR: %w", err)
	}
	return nil
}

// SetRTS sets the Request To Send signal.
func (p *serialPort) SetRTS(rts bool) error {
	if err := p.port.SetRTS(rts); err != nil {
		return fmt.Errorf("serial: failed to set RTS: %w", err)
	}
	return nil
}

// ResetInputBuffer discards data in the input buffer.
func (p *serialPort) ResetInputBuffer() error {
	if err := p.port.ResetInputBuffer(); err != nil {
		return fmt.Errorf("serial: failed to reset input buffer: %w", err)
	}
	return nil
}

// ListPorts returns a list of available serial port names.
func ListPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("serial: failed to list ports: %w", err)
	}
	return ports, nil
}

// ListUSBPorts returns detailed information about available USB serial ports.
func ListUSBPorts() ([]USBPortInfo, error) {
	details, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("serial: failed to get detailed port list: %w", err)
	}

	usbPorts := make([]USBPortInfo, 0, len(details))
	for _, detail := range details {
		info := USBPortInfo{
			Name:         detail.Name,
			IsUSB:        detail.IsUSB,
			VID:          detail.VID,
			PID:          detail.PID,
			SerialNumber: detail.SerialNumber,
			Product:      detail.Product,
		}
		usbPorts = append(usbPorts, info)
	}

	return usbPorts, nil
}

// IsCC2652Adapter checks if a USBPortInfo represents a known CC2652/SONOFF adapter.
// Recognized adapters:
//   - CH340: VID 1A86, PID 55D4
//   - CP2102: VID 10C4, PID EA60
func IsCC2652Adapter(info USBPortInfo) bool {
	if !info.IsUSB {
		return false
	}

	// Normalize VID/PID to uppercase for comparison
	vid := toUpperHex(info.VID)
	pid := toUpperHex(info.PID)

	// CH340 adapter
	if vid == "1A86" && pid == "55D4" {
		return true
	}

	// CP2102 adapter
	if vid == "10C4" && pid == "EA60" {
		return true
	}

	return false
}

// FindCC2652Adapters returns all detected CC2652/SONOFF adapters.
func FindCC2652Adapters() ([]USBPortInfo, error) {
	ports, err := ListUSBPorts()
	if err != nil {
		return nil, err
	}

	var adapters []USBPortInfo
	for _, port := range ports {
		if IsCC2652Adapter(port) {
			adapters = append(adapters, port)
		}
	}

	return adapters, nil
}

// toUpperHex normalizes a hex string to uppercase.
func toUpperHex(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'f' {
			result[i] = c - 32 // Convert to uppercase
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// validatePortPath validates that a port path is safe and appears to be a valid serial port.
// It checks for path traversal attacks and verifies the path format matches expected serial port patterns.
func validatePortPath(path string) error {
	// Clean the path to normalize it
	cleaned := filepath.Clean(path)

	// Check for path traversal
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path contains traversal sequence")
	}

	// Platform-specific validation
	switch runtime.GOOS {
	case "windows":
		// Windows: COM1, COM2, etc. or \\.\COM1 format
		upper := strings.ToUpper(cleaned)
		if strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "\\\\.\\COM") {
			return nil
		}
		return fmt.Errorf("expected COM port format (e.g., COM1, \\\\.\\COM1)")

	case "darwin":
		// macOS: /dev/tty.* or /dev/cu.*
		if strings.HasPrefix(cleaned, "/dev/tty.") || strings.HasPrefix(cleaned, "/dev/cu.") {
			return nil
		}
		return fmt.Errorf("expected /dev/tty.* or /dev/cu.* format")

	default:
		// Linux and other Unix: /dev/ttyUSB*, /dev/ttyACM*, /dev/ttyS*, /dev/serial/*
		validPrefixes := []string{
			"/dev/ttyUSB",
			"/dev/ttyACM",
			"/dev/ttyS",
			"/dev/ttyAMA",
			"/dev/serial/",
		}
		for _, prefix := range validPrefixes {
			if strings.HasPrefix(cleaned, prefix) {
				return nil
			}
		}
		return fmt.Errorf("expected /dev/tty* or /dev/serial/* format")
	}
}

//go:build integration

// Package adapter integration tests require actual Zigbee hardware.
//
// To run these tests:
//
//	GOZNP_PORT=/dev/ttyUSB0 go test -tags=integration ./pkg/adapter/...
//
// Or use the Makefile target:
//
//	GOZNP_PORT=/dev/ttyUSB0 make test-integration
//
// Environment variables:
//   - GOZNP_PORT: Serial port path (required, e.g., /dev/ttyUSB0, /dev/tty.usbserial-110)
//   - GOZNP_BAUD: Baud rate (optional, default: 115200)
package adapter

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// testAdapter is a shared adapter instance for integration tests.
// It's initialized once and reused across tests to avoid repeated open/close cycles.
var testAdapter *Adapter

// getTestAdapter returns a connected adapter for integration tests.
// It initializes the adapter on first call and reuses it for subsequent calls.
func getTestAdapter(t *testing.T) *Adapter {
	t.Helper()

	if testAdapter != nil && testAdapter.IsOpen() {
		return testAdapter
	}

	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		t.Skip("GOZNP_PORT not set, skipping hardware integration test")
	}

	baudRate := 115200
	if baudStr := os.Getenv("GOZNP_BAUD"); baudStr != "" {
		if b, err := strconv.Atoi(baudStr); err == nil {
			baudRate = b
		}
	}

	adapter := New(
		WithSerialPath(port),
		WithBaudRate(baudRate),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := adapter.Open(ctx); err != nil {
		t.Fatalf("Failed to open adapter on %s: %v", port, err)
	}

	testAdapter = adapter
	return testAdapter
}

// TestIntegration_Ping verifies basic connectivity to the coordinator.
func TestIntegration_Ping(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	caps, err := a.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	t.Logf("Ping successful, capabilities: 0x%04X", caps.Capabilities)
}

// TestIntegration_Version retrieves firmware version information.
func TestIntegration_Version(t *testing.T) {
	a := getTestAdapter(t)

	version := a.Version()
	if version == nil {
		t.Fatal("Version returned nil")
	}

	t.Logf("Firmware Version:")
	t.Logf("  Transport: %d", version.TransportRev)
	t.Logf("  Product: %d (%s)", version.Product, version.Variant())
	t.Logf("  Version: %d.%d.%d", version.MajorRel, version.MinorRel, version.MaintRel)
	t.Logf("  Revision: %d", version.Revision)
}

// TestIntegration_GetInfo retrieves coordinator information.
func TestIntegration_GetInfo(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := a.GetInfo(ctx)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	t.Logf("Coordinator Info:")
	t.Logf("  Capabilities: 0x%04X", info.Capabilities)
	if info.Version != nil {
		t.Logf("  Version: %d.%d.%d", info.Version.MajorRel, info.Version.MinorRel, info.Version.MaintRel)
	}
}

// TestIntegration_GetDevices lists paired devices.
func TestIntegration_GetDevices(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := a.GetDevices(ctx)
	if err != nil {
		t.Fatalf("GetDevices failed: %v", err)
	}

	t.Logf("Found %d paired device(s)", len(devices))
	for i, dev := range devices {
		t.Logf("  [%d] IEEE: %s, NwkAddr: 0x%04X",
			i, formatIEEE(dev.IEEEAddr), dev.NwkAddr)
	}
}

// TestIntegration_GetNetworkHealth checks network health status.
func TestIntegration_GetNetworkHealth(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	health, err := a.GetNetworkHealth(ctx)
	if err != nil {
		t.Fatalf("GetNetworkHealth failed: %v", err)
	}

	t.Logf("Network Health:")
	t.Logf("  Channel: %d", health.Channel)
	t.Logf("  PAN ID: 0x%04X", health.PanID)
	t.Logf("  Device Count: %d", health.DeviceCount)
	t.Logf("  Router Count: %d", health.RouterCount)
	t.Logf("  End Device Count: %d", health.EndDeviceCount)
	t.Logf("  Average LQI: %d", health.AverageLQI)
	if len(health.WeakLinks) > 0 {
		t.Logf("  Weak Links: %d", len(health.WeakLinks))
	}
}

// TestIntegration_RegisteredProfiles checks registered application profiles.
func TestIntegration_RegisteredProfiles(t *testing.T) {
	a := getTestAdapter(t)

	profiles := a.RegisteredProfiles()
	t.Logf("Registered %d application profile(s)", len(profiles))
	for _, p := range profiles {
		ep := a.GetEndpointForProfile(p)
		t.Logf("  Profile 0x%04X on endpoint %d", p, ep)
	}
}

// formatIEEE formats an IEEE address as a colon-separated hex string.
func formatIEEE(addr [8]byte) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		addr[7], addr[6], addr[5], addr[4],
		addr[3], addr[2], addr[1], addr[0])
}

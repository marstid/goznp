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

// =============================================================================
// NVRAM Integration Tests
// =============================================================================

// TestIntegration_NvReadNIB reads the Network Information Base from NVRAM.
func TestIntegration_NvReadNIB(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Read the NIB (Network Information Base) - a core NVRAM item.
	data, err := a.znp.NvReadAll(ctx, 0x0021) // NvNIB
	if err != nil {
		t.Fatalf("NvReadAll(NIB) failed: %v", err)
	}

	t.Logf("NIB data length: %d bytes", len(data))
	if len(data) > 0 {
		t.Logf("NIB first 16 bytes: %x", data[:min(16, len(data))])
	}
}

// TestIntegration_NvReadExtAddr reads the coordinator's extended address from NVRAM.
func TestIntegration_NvReadExtAddr(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Read the extended address (IEEE address) from NVRAM.
	data, err := a.znp.NvReadAll(ctx, 0x0001) // NvExtAddr
	if err != nil {
		t.Fatalf("NvReadAll(ExtAddr) failed: %v", err)
	}

	if len(data) != 8 {
		t.Fatalf("ExtAddr should be 8 bytes, got %d", len(data))
	}

	var addr [8]byte
	copy(addr[:], data)
	t.Logf("Coordinator IEEE Address: %s", formatIEEE(addr))
}

// TestIntegration_NvLength tests reading NVRAM item lengths.
func TestIntegration_NvLength(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test a few known NVRAM items.
	items := []struct {
		name string
		id   uint16
	}{
		{"ExtAddr", 0x0001},
		{"NIB", 0x0021},
		{"NwkActiveKeyInfo", 0x003F},
	}

	for _, item := range items {
		length, err := a.znp.NvLength(ctx, item.id)
		if err != nil {
			t.Errorf("NvLength(%s) failed: %v", item.name, err)
			continue
		}
		t.Logf("NVRAM item %s (0x%04X): %d bytes", item.name, item.id, length)
	}
}

// TestIntegration_DeviceNames tests device name storage in NVRAM.
func TestIntegration_DeviceNames(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List existing device names.
	names, err := a.ListDeviceNames(ctx)
	if err != nil {
		t.Fatalf("ListDeviceNames failed: %v", err)
	}

	t.Logf("Found %d device name(s)", len(names))
	for _, n := range names {
		t.Logf("  %s: %q (comment: %q)", formatIEEE(n.IEEEAddr), n.Name, n.Comment)
	}
}

// TestIntegration_DeviceNameRoundTrip tests writing and reading back a device name.
func TestIntegration_DeviceNameRoundTrip(t *testing.T) {
	a := getTestAdapter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get a device to test with.
	devices, err := a.GetDevices(ctx)
	if err != nil {
		t.Fatalf("GetDevices failed: %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No devices paired, skipping device name round-trip test")
	}

	// Use the first device for testing.
	testDevice := devices[0]
	testName := fmt.Sprintf("IntTest_%d", time.Now().Unix()%10000)
	testComment := "Integration test"

	t.Logf("Testing device name round-trip on %s", formatIEEE(testDevice.IEEEAddr))

	// Save current name (if any) to restore later.
	originalName, _ := a.GetDeviceName(ctx, testDevice.IEEEAddr)

	// Set a new name.
	err = a.SetDeviceName(ctx, testDevice.IEEEAddr, testName, testComment)
	if err != nil {
		t.Fatalf("SetDeviceName failed: %v", err)
	}
	t.Logf("Set device name to %q", testName)

	// Read it back.
	retrieved, err := a.GetDeviceName(ctx, testDevice.IEEEAddr)
	if err != nil {
		t.Fatalf("GetDeviceName failed: %v", err)
	}

	if retrieved.Name != testName {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, testName)
	}
	if retrieved.Comment != testComment {
		t.Errorf("Comment mismatch: got %q, want %q", retrieved.Comment, testComment)
	}
	t.Logf("Verified name round-trip: %q", retrieved.Name)

	// Restore original name or clear if there wasn't one.
	if originalName != nil && originalName.Name != "" {
		//nolint:errcheck // Best effort cleanup
		_ = a.SetDeviceName(ctx, testDevice.IEEEAddr, originalName.Name, originalName.Comment)
		t.Logf("Restored original name: %q", originalName.Name)
	} else {
		//nolint:errcheck // Best effort cleanup
		_ = a.DeleteDeviceName(ctx, testDevice.IEEEAddr)
		t.Log("Cleared test name")
	}
}

package adapter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/marstid/goznp/pkg/serial"
	"github.com/marstid/goznp/pkg/znp"
)

// ZNP status codes returned by various operations.
const (
	// StatusSuccess indicates the operation completed successfully.
	StatusSuccess uint8 = 0x00
	// StatusAlreadyRegistered indicates the endpoint is already registered.
	// This is returned by AfRegister when trying to register an existing endpoint.
	StatusAlreadyRegistered uint8 = 0xB8
)

// dedupeKey uniquely identifies a Zigbee message for deduplication.
// The combination of source address, cluster, and transaction sequence number
// allows us to detect retransmitted messages in the mesh network.
type dedupeKey struct {
	srcAddr     uint16
	clusterID   uint16
	transSeqNum uint8
}

// dedupeEntry stores metadata for a seen message.
type dedupeEntry struct {
	seenAt time.Time
}

// dedupeCache provides time-windowed deduplication for incoming messages.
type dedupeCache struct {
	mu      sync.Mutex
	entries map[dedupeKey]dedupeEntry
	window  time.Duration
}

// newDedupeCache creates a new deduplication cache with the given time window.
func newDedupeCache(window time.Duration) *dedupeCache {
	return &dedupeCache{
		entries: make(map[dedupeKey]dedupeEntry),
		window:  window,
	}
}

// isDuplicate checks if a message is a duplicate and records it if not.
// Returns true if this message was seen within the deduplication window.
// A zero window disables deduplication entirely.
func (c *dedupeCache) isDuplicate(srcAddr, clusterID uint16, transSeqNum uint8) bool {
	// Zero window means no deduplication.
	if c.window == 0 {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := dedupeKey{
		srcAddr:     srcAddr,
		clusterID:   clusterID,
		transSeqNum: transSeqNum,
	}

	now := time.Now()

	for k, entry := range c.entries {
		if now.Sub(entry.seenAt) > c.window {
			delete(c.entries, k)
		}
	}

	if entry, exists := c.entries[key]; exists {
		if now.Sub(entry.seenAt) <= c.window {
			return true
		}
	}

	c.entries[key] = dedupeEntry{seenAt: now}
	return false
}

// Adapter represents a Z-Stack Zigbee adapter.
//
// The Adapter provides a high-level API for managing a Zigbee network coordinator,
// including device pairing, ZCL messaging, and network management. It sits on top
// of the ZNP (Zigbee Network Processor) protocol layer, handling the low-level
// communication details.
//
// Thread Safety: All public methods are thread-safe and safe for concurrent
// use from multiple goroutines.
//
// Lifecycle:
//  1. Create adapter with New() and configuration options
//  2. Call Open() to connect to the adapter
//  3. Use adapter methods for device/network operations
//  4. Call Close() when done to release resources
//
// Example:
//
//	adapter := adapter.New(
//	    adapter.WithSerialPath("/dev/tty.usbserial-110"),
//	    adapter.WithBaudRate(115200),
//	)
//	if err := adapter.Open(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer adapter.Close()
type Adapter struct {
	options             Options
	port                serial.Port
	znp                 ZNPClient
	version             *znp.VersionInfo
	deviceMgr           *deviceManager
	registeredEndpoints *RegisteredEndpoints
	dedupe              *dedupeCache
	transactionID       uint32          // Per-adapter ZCL transaction ID counter.
	ctx                 context.Context // Stores the context from Open()

	mu     sync.Mutex
	isOpen bool
	wg     sync.WaitGroup // Tracks event handler goroutines for clean shutdown
}

// nextTransactionID returns the next ZCL transaction sequence number.
// This is per-adapter to avoid ID collisions when multiple adapters are used.
func (a *Adapter) nextTransactionID() uint8 {
	return uint8(atomic.AddUint32(&a.transactionID, 1))
}

// Info contains comprehensive adapter information.
type Info struct {
	Version      *znp.VersionInfo
	Capabilities uint16
}

// DedupeWindow is the default time window for message deduplication.
// Messages with the same (srcAddr, clusterID, transSeqNum) within this
// window are considered duplicates and filtered out.
const DedupeWindow = 2 * time.Second

// New creates a new adapter with the given options.
//
// Use functional options to configure the adapter. Common options include:
//   - WithSerialPath: Set the serial port path (e.g., "/dev/tty.usbserial-110")
//   - WithBaudRate: Set baud rate (e.g., 115200)
//   - WithLogger: Set a logger for debug output
//   - WithZCLRetryAttempts: Set retry attempts for ZCL requests
//
// The adapter is not opened until Open() is called.
//
// Example:
//
//	adapter := adapter.New(
//	    adapter.WithSerialPath("/dev/tty.usbserial-110"),
//	    adapter.WithBaudRate(115200),
//	    adapter.WithLogger(myLogger),
//	    adapter.WithZCLRetryAttempts(3),
//	)
func New(opts ...Option) *Adapter {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return &Adapter{
		options: options,
		dedupe:  newDedupeCache(DedupeWindow),
	}
}

// Open opens the adapter and establishes communication with the Z-Stack coordinator.
//
// This method performs the following initialization steps:
//  1. Opens the serial port specified in SerialConfig
//  2. Initializes the ZNP protocol layer
//  3. Sends handshake pings to verify communication
//  4. Retrieves version information from the adapter
//  5. Registers coordinator endpoints for different profiles (HA, SE, GP, ZLL)
//  6. Starts the Zigbee network from NV memory
//  7. Sets up device event callbacks for join/leave notifications
//
// The adapter must be opened before performing any device or network operations.
// Call Close() when done to release all resources.
//
// If the network has not been configured, Open() will return an error indicating
// that FormNetwork() must be called first.
//
// Example:
//
//	if err := adapter.Open(ctx); err != nil {
//	    return fmt.Errorf("failed to open adapter: %w", err)
//	}
//	defer adapter.Close()
func (a *Adapter) Open(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isOpen {
		return nil
	}

	a.ctx = ctx

	port, err := serial.Open(a.options.SerialConfig)
	if err != nil {
		return fmt.Errorf("adapter: failed to open serial port: %w", err)
	}
	a.port = port

	a.znp = znp.New(port)
	if err := a.znp.Open(ctx); err != nil {
		a.port.Close()
		a.port = nil
		a.znp = nil
		return fmt.Errorf("adapter: failed to start ZNP client: %w", err)
	}

	if err := a.pingHandshake(ctx); err != nil {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		return fmt.Errorf("adapter: ping handshake failed: %w", err)
	}

	version, err := a.znp.Version(ctx)
	if err != nil {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		return fmt.Errorf("adapter: failed to get version info: %w", err)
	}
	a.version = version

	a.deviceMgr = newDeviceManager(&a.wg)
	a.setupDeviceEventCallbacks()

	if err := a.registerCoordinatorEndpoints(ctx); err != nil {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		a.deviceMgr = nil
		return fmt.Errorf("adapter: failed to register coordinator endpoints: %w", err)
	}

	// Start ZDO layer and restore network from NVRAM.
	status, err := a.znp.StartupFromApp(ctx, 100)
	if err != nil {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		a.deviceMgr = nil
		return fmt.Errorf("adapter: network startup failed: %w", err)
	}
	if status == 2 {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		a.deviceMgr = nil
		return fmt.Errorf("adapter: StartupFromApp failed - network not configured (use 'network form' to create)")
	}

	// Brief delay for ZDO layer initialization.
	// This gives the Z-Stack ZDO layer time to initialize before we start
	// sending ZDO commands. The delay is configurable via WithZDOInitDelay.
	if a.options.ZDOInitDelay > 0 {
		time.Sleep(a.options.ZDOInitDelay)
	}

	a.isOpen = true
	return nil
}

func (a *Adapter) setupDeviceEventCallbacks() {
	a.znp.OnDeviceJoin(func(ind *znp.TcDeviceInd) {
		dev := &Device{
			IEEEAddr: ind.IEEEAddr,
			NwkAddr:  ind.NwkAddr,
			LastSeen: time.Now(),
		}
		a.deviceMgr.addDevice(dev)
		a.deviceMgr.notifyEvent(DeviceEvent{
			Type:   DeviceEventJoined,
			Device: dev,
		})
	})

	a.znp.OnDeviceLeave(func(ind *znp.DeviceLeave) {
		a.deviceMgr.removeDevice(ind.IEEEAddr)
		a.deviceMgr.notifyEvent(DeviceEvent{
			Type:     DeviceEventLeft,
			IEEEAddr: ind.IEEEAddr,
		})

		a.wg.Add(1)
		go func(ieeeAddr [8]byte) {
			defer a.wg.Done()
			if a.ctx == nil {
				return
			}
			//nolint:errcheck // Device name cleanup is best-effort, errors intentionally ignored.
			_ = a.DeleteDeviceName(a.ctx, ieeeAddr)
		}(ind.IEEEAddr)
	})

	a.znp.OnDeviceAnnounce(func(ind *znp.DeviceAnnounce) {
		updated := a.deviceMgr.updateDeviceNwkAddr(ind.IEEEAddr, ind.NwkAddr)
		if !updated {
			dev := &Device{
				IEEEAddr:     ind.IEEEAddr,
				NwkAddr:      ind.NwkAddr,
				Capabilities: ind.Capabilities,
				LastSeen:     time.Now(),
			}
			a.deviceMgr.addDevice(dev)
		}
	})
}

// Close closes the adapter and releases all resources.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isOpen {
		return nil
	}

	var firstErr error

	if a.znp != nil {
		if err := a.znp.Close(); err != nil {
			firstErr = fmt.Errorf("adapter: failed to close ZNP client: %w", err)
		}
		a.znp = nil
	}

	if a.port != nil {
		if err := a.port.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("adapter: failed to close serial port: %w", err)
		}
		a.port = nil
	}

	// Wait for all event handler goroutines to complete.
	a.wg.Wait()

	a.isOpen = false
	a.ctx = nil
	a.version = nil
	a.deviceMgr = nil
	a.registeredEndpoints = nil

	return firstErr
}

// IsOpen returns true if the adapter is open and ready for communication.
func (a *Adapter) IsOpen() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isOpen
}

// GetZNP returns the underlying ZNP client.
// This provides access to the ZNP layer for advanced operations like
// listening for incoming messages. The returned ZNPClient is only
// valid while the adapter is open.
func (a *Adapter) GetZNP() ZNPClient {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.znp
}

// checkOpen verifies that the adapter is open and the context is valid.
// Returns an error if the adapter is not open or the context is canceled.
func (a *Adapter) checkOpen(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isOpen {
		return ErrNotOpen
	}
	return nil
}

// Version returns the cached version information.
// Returns nil if the adapter is not open.
func (a *Adapter) Version() *znp.VersionInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.version
}

// Ping sends a ping request to the adapter and returns capabilities.
func (a *Adapter) Ping(ctx context.Context) (*znp.PingCapabilities, error) {
	if err := a.checkOpen(ctx); err != nil {
		return nil, err
	}

	a.mu.Lock()
	znpClient := a.znp
	a.mu.Unlock()

	caps, err := znpClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("adapter: ping failed: %w", err)
	}

	return caps, nil
}

// Reset performs a soft reset of the adapter.
func (a *Adapter) Reset(ctx context.Context) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Create context with reset timeout.
	resetCtx, cancel := context.WithTimeout(ctx, a.options.ResetTimeout)
	defer cancel()

	_, err := znpClient.Reset(resetCtx, znp.ResetTypeSoft)
	if err != nil {
		return fmt.Errorf("adapter: reset failed: %w", err)
	}

	// Update version info after reset.
	version, err := znpClient.Version(ctx)
	if err != nil {
		return fmt.Errorf("adapter: failed to get version info after reset: %w", err)
	}

	a.mu.Lock()
	a.version = version
	a.mu.Unlock()

	return nil
}

// GetInfo returns comprehensive adapter information including version and capabilities.
func (a *Adapter) GetInfo(ctx context.Context) (*Info, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	cachedVersion := a.version
	a.mu.Unlock()

	// Get capabilities via ping.
	caps, err := znpClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("adapter: failed to get capabilities: %w", err)
	}

	return &Info{
		Version:      cachedVersion,
		Capabilities: caps.Capabilities,
	}, nil
}

// pingHandshake performs the initial ping handshake with retries.
func (a *Adapter) pingHandshake(ctx context.Context) error {
	var lastErr error

	for attempt := 0; attempt < a.options.PingRetries; attempt++ {
		// Create context with ping timeout.
		pingCtx, cancel := context.WithTimeout(ctx, a.options.PingTimeout)

		_, err := a.znp.Ping(pingCtx)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err

		// Don't sleep after the last attempt.
		if attempt < a.options.PingRetries-1 {
			// Small delay before retry.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
	}

	return fmt.Errorf("ping failed after %d attempts: %w", a.options.PingRetries, lastErr)
}

// registerCoordinatorEndpoints registers all coordinator endpoints.
// This follows the zigbee-herdsman pattern of registering multiple endpoints
// with different profiles to support communication with various device types.
func (a *Adapter) registerCoordinatorEndpoints(ctx context.Context) error {
	a.registeredEndpoints = NewRegisteredEndpoints()

	for _, epDef := range CoordinatorEndpoints {
		// Try to delete existing registration first (ignore errors)
		//nolint:errcheck // AfDelete errors are expected and intentionally ignored
		_, _ = a.znp.AfDelete(ctx, epDef.Endpoint)

		config := znp.EndpointConfig{
			Endpoint:    epDef.Endpoint,
			AppProfID:   uint16(epDef.ProfileID),
			AppDeviceID: epDef.DeviceID,
			AppDevVer:   1,
			InClusters:  epDef.InClusters,
			OutClusters: epDef.OutClusters,
		}

		status, err := a.znp.AfRegister(ctx, config)
		if err != nil {
			a.options.Logger.Warnf("failed to register endpoint %d: %v", epDef.Endpoint, err)
			continue
		}
		if status != StatusSuccess && status != StatusAlreadyRegistered {
			a.options.Logger.Warnf("endpoint %d registration returned status 0x%02X", epDef.Endpoint, status)
			continue
		}

		a.registeredEndpoints.Add(epDef.Endpoint, epDef.ProfileID)
	}

	// At minimum, endpoint 1 (HA) must be registered.
	if len(a.registeredEndpoints.Endpoints) == 0 {
		return fmt.Errorf("failed to register any coordinator endpoints")
	}

	return nil
}

// RegisteredProfiles returns the list of Application Profiles supported by registered endpoints.
func (a *Adapter) RegisteredProfiles() []znp.ApplicationProfile {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.registeredEndpoints == nil {
		return []znp.ApplicationProfile{znp.ProfileHomeAutomation}
	}
	return a.registeredEndpoints.GetProfiles()
}

// GetEndpointForProfile returns the coordinator endpoint to use for a given device profile.
func (a *Adapter) GetEndpointForProfile(profile znp.ApplicationProfile) uint8 {
	return GetEndpointForProfile(profile)
}

package adapter

import (
	"context"
	"fmt"
	"sync"
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
	// Zero window means no deduplication
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
type Adapter struct {
	options             Options
	port                serial.Port
	znp                 *znp.ZNP
	version             *znp.VersionInfo
	deviceMgr           *deviceManager
	registeredEndpoints *RegisteredEndpoints
	dedupe              *dedupeCache

	mu     sync.Mutex
	isOpen bool
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
// It opens the serial port, initializes the ZNP protocol layer, verifies connectivity,
// registers coordinator endpoints, and starts the Zigbee network.
func (a *Adapter) Open(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isOpen {
		return nil
	}

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

	a.deviceMgr = newDeviceManager()
	a.setupDeviceEventCallbacks()

	if err := a.registerCoordinatorEndpoints(ctx); err != nil {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		a.deviceMgr = nil
		return fmt.Errorf("adapter: failed to register coordinator endpoints: %w", err)
	}

	// Start ZDO layer and restore network from NVRAM
	status, err := a.znp.StartupFromApp(ctx, 100)
	if err != nil {
		// Non-fatal - might already be started
	} else if status == 2 {
		a.znp.Close()
		a.port.Close()
		a.port = nil
		a.znp = nil
		a.deviceMgr = nil
		return fmt.Errorf("adapter: StartupFromApp failed - network not configured (use 'network form' to create)")
	}

	// Brief delay for ZDO layer initialization
	time.Sleep(200 * time.Millisecond)

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

		// Clean up device name in background (non-fatal, best-effort)
		go func() {
			ctx := context.Background()
			_ = a.DeleteDeviceName(ctx, ind.IEEEAddr)
		}()
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
		if err := a.znp.Close(); err != nil && firstErr == nil {
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

	a.isOpen = false
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

// Version returns the cached version information.
// Returns nil if the adapter is not open.
func (a *Adapter) Version() *znp.VersionInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.version
}

// Ping sends a ping request to the adapter and returns capabilities.
func (a *Adapter) Ping(ctx context.Context) (*znp.PingCapabilities, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
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

	// Create context with reset timeout
	resetCtx, cancel := context.WithTimeout(ctx, a.options.ResetTimeout)
	defer cancel()

	_, err := znpClient.Reset(resetCtx, znp.ResetTypeSoft)
	if err != nil {
		return fmt.Errorf("adapter: reset failed: %w", err)
	}

	// Update version info after reset
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

	// Get capabilities via ping
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
		// Create context with ping timeout
		pingCtx, cancel := context.WithTimeout(ctx, a.options.PingTimeout)

		_, err := a.znp.Ping(pingCtx)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err

		// Don't sleep after the last attempt
		if attempt < a.options.PingRetries-1 {
			// Small delay before retry
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
			// Log warning but continue - some endpoints may fail
			fmt.Printf("Warning: failed to register endpoint %d: %v\n", epDef.Endpoint, err)
			continue
		}
		if status != StatusSuccess && status != StatusAlreadyRegistered {
			fmt.Printf("Warning: endpoint %d registration returned status 0x%02X\n", epDef.Endpoint, status)
			continue
		}

		a.registeredEndpoints.Add(epDef.Endpoint, epDef.ProfileID)
	}

	// At minimum, endpoint 1 (HA) must be registered
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

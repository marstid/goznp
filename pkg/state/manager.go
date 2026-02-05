package state

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
)

// Manager manages device state and sensor cache.
// It provides thread-safe access to device information and coordinates with the event system.
type Manager struct {
	mu         sync.RWMutex
	adapter    *adapter.Adapter
	devices    map[[8]byte]*DeviceState
	slugIndex  map[string][8]byte // slug -> ieeeAddr
	sensors    *SensorCache
	eventBus   EventBus
	logger     *slog.Logger
	pollTimer  *time.Ticker
	pollCtx    context.Context
	pollCancel context.CancelFunc
}

// EventBus interface for event publishing.
// Implemented by pkg/event.Bus.
type EventBus interface {
	Publish(ctx context.Context, event interface{}) error
}

// NewManager creates a new state manager.
func NewManager(ctx context.Context, adapter *adapter.Adapter, eventBus EventBus, pollInterval time.Duration, logger *slog.Logger) *Manager {
	mgr := &Manager{
		adapter:   adapter,
		devices:   make(map[[8]byte]*DeviceState),
		slugIndex: make(map[string][8]byte),
		sensors:   NewSensorCache(),
		eventBus:  eventBus,
		logger:    logger,
	}

	// Setup device event listener
	adapter.OnDeviceEvent(mgr.handleDeviceEvent)

	// Start polling sleepy devices if interval > 0
	if pollInterval > 0 {
		mgr.pollCtx, mgr.pollCancel = context.WithCancel(context.Background())
		mgr.pollTimer = time.NewTicker(pollInterval)
		go mgr.pollSleepyDevices(mgr.pollCtx)
	}

	return mgr
}

// Close shuts down the state manager.
func (mgr *Manager) Close() {
	if mgr.pollCancel != nil {
		mgr.pollCancel()
		mgr.pollTimer.Stop()
	}
}

// GetDevice returns a device by IEEE address.
func (mgr *Manager) GetDevice(ieeeAddr [8]byte) (*DeviceState, bool) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	dev, ok := mgr.devices[ieeeAddr]
	return dev, ok
}

// GetDeviceByNwkAddr returns a device by network address.
func (mgr *Manager) GetDeviceByNwkAddr(nwkAddr uint16) (*DeviceState, bool) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	for _, dev := range mgr.devices {
		if dev.NwkAddr == nwkAddr {
			return dev, true
		}
	}
	return nil, false
}

// GetDeviceByIdentifier looks up device by slug or IEEE address.
// Priority: 1) Try as slug, 2) Try as IEEE address.
func (mgr *Manager) GetDeviceByIdentifier(identifier string) (*DeviceState, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	// First, try as slug
	if ieeeAddr, ok := mgr.slugIndex[identifier]; ok {
		if dev, exists := mgr.devices[ieeeAddr]; exists {
			return dev, nil
		}
	}

	// Then, try as IEEE address
	if ieeeAddr, err := ParseIEEEAddr(identifier); err == nil {
		if dev, exists := mgr.devices[ieeeAddr]; exists {
			return dev, nil
		}
	}

	return nil, ErrDeviceNotFound
}

// GetDevices returns all devices.
func (mgr *Manager) GetDevices() []*DeviceState {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	devices := make([]*DeviceState, 0, len(mgr.devices))
	for _, dev := range mgr.devices {
		devices = append(devices, dev)
	}
	return devices
}

// AddDevice adds a device to the state manager.
// Returns the device's IEEE address and generated slug.
func (mgr *Manager) AddDevice(ieeeAddr [8]byte, nwkAddr uint16, capabilities uint8) ([8]byte, string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if _, exists := mgr.devices[ieeeAddr]; !exists {
		dev := NewDeviceState(ieeeAddr, nwkAddr, capabilities)
		mgr.devices[ieeeAddr] = dev

		// Add default slug
		defaultSlug := GetDefaultSlugForIEEE(ieeeAddr)
		mgr.slugIndex[defaultSlug] = ieeeAddr

		mgr.logger.Info("Device added", "ieeeAddr", FormatIEEEAddr(ieeeAddr), "nwkAddr", fmt.Sprintf("0x%04X", nwkAddr), "slug", defaultSlug)
	}

	return ieeeAddr, GetDefaultSlugForIEEE(ieeeAddr)
}

// PopulateDevices initializes the device list from adapter.Device objects.
// Used during startup to populate the state manager with existing devices.
func (mgr *Manager) PopulateDevices(devices []*adapter.Device) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	for _, dev := range devices {
		if _, exists := mgr.devices[dev.IEEEAddr]; !exists {
			newDev := NewDeviceState(dev.IEEEAddr, dev.NwkAddr, dev.Capabilities)
			mgr.devices[dev.IEEEAddr] = newDev

			defaultSlug := GetDefaultSlugForIEEE(dev.IEEEAddr)
			mgr.slugIndex[defaultSlug] = dev.IEEEAddr

			mgr.logger.Debug("Device populated",
				"ieeeAddr", FormatIEEEAddr(dev.IEEEAddr),
				"nwkAddr", fmt.Sprintf("0x%04X", dev.NwkAddr),
				"slug", defaultSlug)
		}
	}
}

// RemoveDevice removes a device from the state manager.
func (mgr *Manager) RemoveDevice(ieeeAddr [8]byte) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if dev, ok := mgr.devices[ieeeAddr]; ok {
		// Remove from slug index
		slug := dev.Slug()
		delete(mgr.slugIndex, slug)

		delete(mgr.devices, ieeeAddr)
		mgr.sensors.RemoveDevice(ieeeAddr)

		mgr.logger.Info("Device removed", "ieeeAddr", FormatIEEEAddr(ieeeAddr), "slug", slug)
	}
}

// SetDeviceName sets a friendly name and generates a unique slug.
// Returns the generated slug.
// Errors:
//   - ErrDeviceNotFound: device doesn't exist
//   - ErrNameInUse: slug already assigned to another device
func (mgr *Manager) SetDeviceName(ieeeAddr [8]byte, name string) (string, error) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	dev, ok := mgr.devices[ieeeAddr]
	if !ok {
		return "", ErrDeviceNotFound
	}

	oldSlug := dev.Slug()
	dev.Name = name

	newSlug := SlugFormat(name)
	if newSlug == "" {
		newSlug = GetDefaultSlugForIEEE(ieeeAddr)
	}

	if oldSlug == newSlug {
		return newSlug, nil // No change
	}

	// Check for conflicts
	if existingIEEE, exists := mgr.slugIndex[newSlug]; exists && existingIEEE != ieeeAddr {
		// Conflict! Revert the name change
		dev.Name = "" // Revert to old name
		return "", fmt.Errorf("%w: device name '%s' conflicts with existing device (slug '%s' already in use)", ErrNameInUse, name, newSlug)
	}

	// No conflict, update slug index
	if oldSlug != "" {
		delete(mgr.slugIndex, oldSlug)
	}
	mgr.slugIndex[newSlug] = ieeeAddr

	mgr.logger.Info("Device name changed", "ieeeAddr", FormatIEEEAddr(ieeeAddr), "oldSlug", oldSlug, "newSlug", newSlug, "name", name)
	return newSlug, nil
}

// SetDeviceNameByIdentifier sets a device name using an identifier (slug or IEEE).
// Returns the generated slug.
func (mgr *Manager) SetDeviceNameByIdentifier(identifier string, name string) (string, error) {
	mgr.mu.RLock()
	// Look up IEEE from identifiers first
	ieeeAddr, err := mgr.resolveIdentifierLocked(identifier)
	mgr.mu.RUnlock()

	if err != nil {
		return "", err
	}

	return mgr.SetDeviceName(ieeeAddr, name)
}

// GetSlugForIEEE returns the slug for a device by its IEEE address.
func (mgr *Manager) GetSlugForIEEE(ieeeAddr [8]byte) string {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if dev, ok := mgr.devices[ieeeAddr]; ok {
		return dev.Slug()
	}
	return GetDefaultSlugForIEEE(ieeeAddr)
}

// GetSensorData returns cached sensor readings for a device.
func (mgr *Manager) GetSensorData(ieeeAddr [8]byte) *DeviceSensors {
	return mgr.sensors.GetLatest(ieeeAddr)
}

// UpdateState updates device state from incoming attribute reports.
// Returns true if state changed.
func (mgr *Manager) UpdateState(ieeeAddr [8]byte, clusterID uint16, attrs []adapter.ReportedAttribute) bool {
	mgr.mu.RLock()
	dev, ok := mgr.devices[ieeeAddr]
	mgr.mu.RUnlock()

	if !ok {
		return false
	}

	// Update state values
	changed := false
	for _, attr := range attrs {
		if dev.SetState(clusterID, uint16(attr.ID), attr.Value) {
			changed = true
		}
	}

	if changed {
		dev.LastUpdated = time.Now()
	}

	// Update sensor cache
	_ = mgr.sensors.UpdateFromAttributes(ieeeAddr, clusterID, attrs)

	return changed
}

// UpdateNwkAddr updates the network address for a device.
// Called when a device rejoins with a new address.
func (mgr *Manager) UpdateNwkAddr(ieeeAddr [8]byte, newNwkAddr uint16) bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if dev, ok := mgr.devices[ieeeAddr]; ok {
		if dev.NwkAddr != newNwkAddr {
			oldAddr := dev.NwkAddr
			dev.NwkAddr = newNwkAddr
			dev.LastSeen = time.Now()
			mgr.logger.Info("Device network address changed", "ieeeAddr", FormatIEEEAddr(ieeeAddr), "oldAddr", fmt.Sprintf("0x%04X", oldAddr), "newAddr", fmt.Sprintf("0x%04X", newNwkAddr))
			return true
		}
	}
	return false
}

// RefreshDevice interviews a device and updates its state.
// Returns error if interview fails.
func (mgr *Manager) RefreshDevice(ctx context.Context, ieeeAddr [8]byte) error {
	mgr.mu.RLock()
	dev, ok := mgr.devices[ieeeAddr]
	nwkAddr := dev.NwkAddr
	mgr.mu.RUnlock()

	if !ok {
		return ErrDeviceNotFound
	}

	// Get active endpoints
	endpoints, err := mgr.adapter.GetActiveEndpoints(ctx, nwkAddr)
	if err != nil {
		return fmt.Errorf("failed to get active endpoints: %w", err)
	}

	mgr.mu.Lock()
	dev.Endpoints = endpoints
	dev.LastSeen = time.Now()
	mgr.mu.Unlock()

	return nil
}

// handleDeviceEvent processes device join/leave events.
func (mgr *Manager) handleDeviceEvent(e adapter.DeviceEvent) {
	switch e.Type {
	case adapter.DeviceEventJoined:
		if e.Device != nil {
			mgr.AddDevice(e.Device.IEEEAddr, e.Device.NwkAddr, e.Device.Capabilities)
		}
	case adapter.DeviceEventLeft:
		mgr.RemoveDevice(e.IEEEAddr)
	}
}

// pollSleepyDevices periodically polls battery-powered devices.
func (mgr *Manager) pollSleepyDevices(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-mgr.pollTimer.C:
			mgr.pollOnce(ctx)
		}
	}
}

// pollOnce polls all battery-powered devices once.
func (mgr *Manager) pollOnce(ctx context.Context) {
	mgr.mu.RLock()
	devices := make([]*DeviceState, 0, len(mgr.devices))
	for _, dev := range mgr.devices {
		devices = append(devices, dev)
	}
	mgr.mu.RUnlock()

	for _, dev := range devices {
		// Only poll battery-powered (end) devices, not routers
		if dev.IsBatteryPowered() {
			pollCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := mgr.RefreshDevice(pollCtx, dev.IEEEAddr)
			cancel()

			if err != nil {
				mgr.logger.Debug("Failed to poll sleepy device", "ieeeAddr", FormatIEEEAddr(dev.IEEEAddr), "slug", dev.Slug(), "error", err)
			}
		}
	}
}

// resolveIdentifierLocked resolves an identifier to an IEEE address.
// Caller must hold RLock.
func (mgr *Manager) resolveIdentifierLocked(identifier string) ([8]byte, error) {
	// First, try as slug
	if ieeeAddr, ok := mgr.slugIndex[identifier]; ok {
		return ieeeAddr, nil
	}

	// Then, try as IEEE address
	if ieeeAddr, err := ParseIEEEAddr(identifier); err == nil {
		return ieeeAddr, nil
	}

	return [8]byte{}, ErrDeviceNotFound
}

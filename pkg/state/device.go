package state

import (
	"sync"
	"time"
)

// DeviceStateMap represents device state as cluster ID -> attribute ID -> value.
type DeviceStateMap map[uint16]map[uint16]interface{}

// DeviceState represents the complete state of a device.
// It extends adapter.Device with additional state tracking.
type DeviceState struct {
	// From adapter.Device
	IEEEAddr     [8]byte
	NwkAddr      uint16
	Capabilities uint8
	Endpoints    []uint8
	LastSeen     time.Time

	// Enhanced state
	Name         string    // Friendly name (optional)
	Manufacturer string    // From interview
	Model        string    // From interview
	JoinedAt     time.Time // When device joined network

	// State cache
	State       DeviceStateMap
	StateMu     sync.RWMutex
	LastUpdated time.Time
}

// IsRouter returns true if device is a router.
func (ds *DeviceState) IsRouter() bool {
	return ds.Capabilities&0x01 != 0
}

// IsBatteryPowered returns true if device is battery powered.
func (ds *DeviceState) IsBatteryPowered() bool {
	return ds.Capabilities&0x04 == 0
}

// Slug returns the slug for this device.
// - If Name is set, returns slugified name
// - If Name is empty, returns default slug from IEEE address
func (ds *DeviceState) Slug() string {
	if ds.Name == "" {
		return GetDefaultSlugForIEEE(ds.IEEEAddr)
	}
	return SlugFormat(ds.Name)
}

// GetState retrieves cached state for a specific cluster/attribute.
// Returns the value and true if found, nil and false otherwise.
func (ds *DeviceState) GetState(clusterID, attrID uint16) (interface{}, bool) {
	ds.StateMu.RLock()
	defer ds.StateMu.RUnlock()

	cluster, ok := ds.State[clusterID]
	if !ok {
		return nil, false
	}

	value, ok := cluster[attrID]
	return value, ok
}

// SetState updates cached state and returns true if value changed.
// Returns false if value unchanged.
func (ds *DeviceState) SetState(clusterID, attrID uint16, value interface{}) bool {
	ds.StateMu.Lock()
	defer ds.StateMu.Unlock()

	if ds.State == nil {
		ds.State = make(DeviceStateMap)
	}

	cluster, ok := ds.State[clusterID]
	if !ok {
		cluster = make(map[uint16]interface{})
		ds.State[clusterID] = cluster
	}

	// Check if value changed
	oldValue, exists := cluster[attrID]
	if exists && oldValue == value {
		return false
	}

	cluster[attrID] = value
	ds.LastUpdated = time.Now()
	return true
}

// GetOnOffState returns the on/off state from the OnOff cluster (0x0006).
// Returns nil if not available.
func (ds *DeviceState) GetOnOffState() *bool {
	val, ok := ds.GetState(0x0006, 0x0000)
	if !ok {
		return nil
	}

	b, ok := val.(bool)
	if ok {
		return &b
	}

	// Also accept uint8 (some devices report as 0/1)
	if u, ok := val.(uint8); ok {
		result := u != 0
		return &result
	}

	return nil
}

// GetBrightness returns the brightness level from LevelControl cluster (0x0008).
// Returns nil if not available.
func (ds *DeviceState) GetBrightness() *uint8 {
	val, ok := ds.GetState(0x0008, 0x0000)
	if !ok {
		return nil
	}

	b, ok := val.(uint8)
	if ok {
		return &b
	}

	return nil
}

// GetStateMap returns a copy of the entire state map.
func (ds *DeviceState) GetStateMap() DeviceStateMap {
	ds.StateMu.RLock()
	defer ds.StateMu.RUnlock()

	result := make(DeviceStateMap)
	for clusterID, cluster := range ds.State {
		result[clusterID] = make(map[uint16]interface{})
		for attrID, value := range cluster {
			result[clusterID][attrID] = value
		}
	}
	return result
}

// NewDeviceState creates a new DeviceState from a device event or discovery.
func NewDeviceState(ieeeAddr [8]byte, nwkAddr uint16, capabilities uint8) *DeviceState {
	now := time.Now()
	return &DeviceState{
		IEEEAddr:     ieeeAddr,
		NwkAddr:      nwkAddr,
		Capabilities: capabilities,
		Endpoints:    []uint8{},
		LastSeen:     now,
		JoinedAt:     now,
		State:        make(DeviceStateMap),
	}
}

// DeviceSummary provides a summary of device state for API responses.
type DeviceSummary struct {
	IEEEAddr     string    `json:"ieee_addr"`
	NwkAddr      string    `json:"nwk_addr"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name,omitempty"`
	Model        string    `json:"model,omitempty"`
	Manufacturer string    `json:"manufacturer,omitempty"`
	LastSeen     time.Time `json:"last_seen"`
	JoinedAt     time.Time `json:"joined_at"`
	IsRouter     bool      `json:"is_router"`
	IsBattery    bool      `json:"is_battery_powered"`
}

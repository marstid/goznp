package adapter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/marstid/goznp/pkg/znp"
)

// Device represents a paired Zigbee device.
type Device struct {
	IEEEAddr     [8]byte
	NwkAddr      uint16
	Capabilities uint8
	Endpoints    []uint8
	LastSeen     time.Time
}

// IsRouter returns true if device is a router.
func (d *Device) IsRouter() bool {
	return d.Capabilities&znp.CapDeviceType != 0
}

// IsBatteryPowered returns true if battery powered.
func (d *Device) IsBatteryPowered() bool {
	return d.Capabilities&znp.CapPowerSource == 0
}

// DeviceEventType defines device event types.
type DeviceEventType int

const (
	DeviceEventJoined DeviceEventType = iota
	DeviceEventLeft
)

// DeviceEvent represents a device join/leave event.
type DeviceEvent struct {
	Type     DeviceEventType
	Device   *Device
	IEEEAddr [8]byte // For leave events.
}

// deviceManager manages paired devices.
type deviceManager struct {
	mu      sync.RWMutex
	devices map[[8]byte]*Device
	handler func(DeviceEvent)
	wg      *sync.WaitGroup // Tracks event handler goroutines
}

// newDeviceManager creates a new device manager.
func newDeviceManager(wg *sync.WaitGroup) *deviceManager {
	return &deviceManager{
		devices: make(map[[8]byte]*Device),
		wg:      wg,
	}
}

// addDevice adds or updates a device in the manager.
func (dm *deviceManager) addDevice(dev *Device) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.devices[dev.IEEEAddr] = dev
}

// removeDevice removes a device from the manager.
func (dm *deviceManager) removeDevice(ieeeAddr [8]byte) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	delete(dm.devices, ieeeAddr)
}

// getDevice returns a device by IEEE address.
func (dm *deviceManager) getDevice(ieeeAddr [8]byte) *Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.devices[ieeeAddr]
}

// getDeviceByNwkAddr returns a device by network address.
func (dm *deviceManager) getDeviceByNwkAddr(nwkAddr uint16) *Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	for _, dev := range dm.devices {
		if dev.NwkAddr == nwkAddr {
			return dev
		}
	}
	return nil
}

// updateDeviceNwkAddr updates the network address for a device.
// This is called when a device announces with a new address (e.g., after rejoin).
// Returns true if the device was found and updated, false otherwise.
func (dm *deviceManager) updateDeviceNwkAddr(ieeeAddr [8]byte, newNwkAddr uint16) bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dev, exists := dm.devices[ieeeAddr]; exists {
		dev.NwkAddr = newNwkAddr
		dev.LastSeen = time.Now()
		return true
	}
	return false
}

// setHandler sets the device event handler.
func (dm *deviceManager) setHandler(h func(DeviceEvent)) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.handler = h
}

// notifyEvent calls the event handler if set.
// The event is deeply copied to avoid data races with the handler goroutine.
func (dm *deviceManager) notifyEvent(event DeviceEvent) {
	dm.mu.RLock()
	handler := dm.handler
	dm.mu.RUnlock()

	if handler != nil {
		// Deep copy event to avoid potential data races with event data.
		eventCopy := event
		// Deep copy the Device struct if present to avoid concurrent modification.
		if event.Device != nil {
			deviceCopy := *event.Device
			// Deep copy the Endpoints slice to avoid concurrent modification.
			if len(deviceCopy.Endpoints) > 0 {
				deviceCopy.Endpoints = make([]uint8, len(event.Device.Endpoints))
				copy(deviceCopy.Endpoints, event.Device.Endpoints)
			}
			eventCopy.Device = &deviceCopy
		}
		// Call handler in a goroutine to avoid blocking.
		if dm.wg != nil {
			dm.wg.Add(1)
			go func(e DeviceEvent) {
				defer dm.wg.Done()
				handler(e)
			}(eventCopy)
		} else {
			go handler(eventCopy)
		}
	}
}

// PermitJoin opens or closes the network for device joining.
// duration: 0=close, 1-254=seconds, 255=always open
func (a *Adapter) PermitJoin(ctx context.Context, duration uint8) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	status, err := znpClient.MgmtPermitJoinReq(ctx, duration)
	if err != nil {
		return fmt.Errorf("permit join request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("permit join returned status 0x%02X", status)
	}

	return nil
}

// GetDevices returns all devices from the coordinator's NVRAM Address Manager table.
func (a *Adapter) GetDevices(ctx context.Context) ([]*Device, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Read devices from NVRAM Address Manager table.
	entries, err := znpClient.ReadAddrMgrTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading address manager table: %w", err)
	}

	devices := make([]*Device, 0, len(entries))
	for _, entry := range entries {
		// Skip coordinator (address 0x0000).
		if entry.NwkAddr == CoordinatorAddressNwk {
			continue
		}

		dev := &Device{
			IEEEAddr: entry.ExtAddr,
			NwkAddr:  entry.NwkAddr,
		}
		devices = append(devices, dev)
	}

	return devices, nil
}

// GetDevice returns a device by IEEE address.
func (a *Adapter) GetDevice(ieeeAddr [8]byte) *Device {
	if a.deviceMgr == nil {
		return nil
	}
	return a.deviceMgr.getDevice(ieeeAddr)
}

// GetDeviceByNwkAddr returns a device by network address.
func (a *Adapter) GetDeviceByNwkAddr(nwkAddr uint16) *Device {
	if a.deviceMgr == nil {
		return nil
	}
	return a.deviceMgr.getDeviceByNwkAddr(nwkAddr)
}

// OnDeviceEvent sets a callback for device join/leave events.
func (a *Adapter) OnDeviceEvent(handler func(DeviceEvent)) {
	if a.deviceMgr != nil {
		a.deviceMgr.setHandler(handler)
	}
}

// GetActiveEndpoints queries a device for its active endpoints.
func (a *Adapter) GetActiveEndpoints(ctx context.Context, nwkAddr uint16) ([]uint8, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	activeEp, err := znpClient.ActiveEpReq(ctx, nwkAddr)
	if err != nil {
		return nil, fmt.Errorf("active endpoint request failed: %w", err)
	}

	if activeEp.Status != 0 {
		return nil, fmt.Errorf("active endpoint request returned status 0x%02X", activeEp.Status)
	}

	return activeEp.Endpoints, nil
}

// EndpointDescriptor contains cluster information for an endpoint.
type EndpointDescriptor struct {
	Endpoint      uint8
	ProfileID     uint16
	DeviceID      uint16
	DeviceVersion uint8
	InClusters    []uint16
	OutClusters   []uint16
}

// GetSimpleDescriptor queries a device endpoint for its simple descriptor.
func (a *Adapter) GetSimpleDescriptor(ctx context.Context, nwkAddr uint16, endpoint uint8) (*EndpointDescriptor, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	desc, err := znpClient.SimpleDescReq(ctx, nwkAddr, endpoint)
	if err != nil {
		return nil, fmt.Errorf("simple descriptor request failed: %w", err)
	}

	return &EndpointDescriptor{
		Endpoint:      desc.Endpoint,
		ProfileID:     desc.ProfileID,
		DeviceID:      desc.DeviceID,
		DeviceVersion: desc.DeviceVersion,
		InClusters:    desc.InClusters,
		OutClusters:   desc.OutClusters,
	}, nil
}

// DeviceCapabilities contains full device capability information.
type DeviceCapabilities struct {
	NwkAddr   uint16
	Endpoints []*EndpointDescriptor
}

// GetDeviceCapabilities queries all endpoints and their clusters from a device.
func (a *Adapter) GetDeviceCapabilities(ctx context.Context, nwkAddr uint16) (*DeviceCapabilities, error) {
	// First get active endpoints.
	endpoints, err := a.GetActiveEndpoints(ctx, nwkAddr)
	if err != nil {
		return nil, fmt.Errorf("getting active endpoints: %w", err)
	}

	caps := &DeviceCapabilities{
		NwkAddr:   nwkAddr,
		Endpoints: make([]*EndpointDescriptor, 0, len(endpoints)),
	}

	// Query each endpoint for its simple descriptor.
	for _, ep := range endpoints {
		desc, err := a.GetSimpleDescriptor(ctx, nwkAddr, ep)
		if err != nil {
			// Log error but continue with other endpoints.
			continue
		}
		caps.Endpoints = append(caps.Endpoints, desc)
	}

	return caps, nil
}

// NeighborInfo contains information about a neighbor in the mesh network.
type NeighborInfo struct {
	NwkAddr    uint16
	IEEEAddr   [8]byte
	DeviceType uint8 // 0=Coordinator, 1=Router, 2=EndDevice.
	Relation   uint8 // 0=Parent, 1=Child, 2=Sibling, 3=None, 4=PreviousChild.
	LQI        uint8 // Link Quality (0-255, higher is better).
	Depth      uint8 // Tree depth from coordinator.
}

// GetNeighborTable retrieves the neighbor table from a device.
// This shows which devices this node can directly communicate with.
func (a *Adapter) GetNeighborTable(ctx context.Context, nwkAddr uint16) ([]NeighborInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	entries, err := znpClient.GetAllNeighbors(ctx, nwkAddr)
	if err != nil {
		return nil, fmt.Errorf("getting neighbor table: %w", err)
	}

	neighbors := make([]NeighborInfo, len(entries))
	for i, e := range entries {
		neighbors[i] = NeighborInfo{
			NwkAddr:    e.NwkAddr,
			IEEEAddr:   e.ExtAddr,
			DeviceType: e.DeviceType,
			Relation:   e.Relation,
			LQI:        e.LQI,
			Depth:      e.Depth,
		}
	}

	return neighbors, nil
}

// BindingInfo contains information about a binding entry on a device.
type BindingInfo struct {
	SrcAddr     [8]byte // Source device IEEE address.
	SrcEndpoint uint8   // Source endpoint.
	ClusterID   uint16  // Cluster ID.
	DstAddrMode uint8   // 0x01=group, 0x03=IEEE address.
	DstAddr     [8]byte // Destination IEEE address (or group ID).
	DstEndpoint uint8   // Destination endpoint.
}

// GetBindingTable retrieves the binding table from a device.
// This shows where the device will send reports for each cluster.
func (a *Adapter) GetBindingTable(ctx context.Context, nwkAddr uint16) ([]BindingInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	entries, err := znpClient.GetAllBindings(ctx, nwkAddr)
	if err != nil {
		return nil, fmt.Errorf("getting binding table: %w", err)
	}

	bindings := make([]BindingInfo, len(entries))
	for i, e := range entries {
		bindings[i] = BindingInfo{
			SrcAddr:     e.SrcAddr,
			SrcEndpoint: e.SrcEndpoint,
			ClusterID:   e.ClusterID,
			DstAddrMode: e.DstAddrMode,
			DstAddr:     e.DstAddr,
			DstEndpoint: e.DstEndpoint,
		}
	}

	return bindings, nil
}

// RemoveDevice sends a leave request to remove a device from the network.
// If removeChildren is true, the device will also remove any children it has.
// If rejoin is false, the device will not attempt to rejoin the network.
func (a *Adapter) RemoveDevice(ctx context.Context, nwkAddr uint16, ieeeAddr [8]byte, removeChildren, rejoin bool) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	status, err := znpClient.MgmtLeaveReq(ctx, nwkAddr, ieeeAddr, removeChildren, rejoin)
	if err != nil {
		return fmt.Errorf("remove device request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("remove device returned status 0x%02X", status)
	}

	// Remove from local device manager.
	a.deviceMgr.removeDevice(ieeeAddr)

	// Clean up device name (non-fatal, best-effort)
	//nolint:errcheck // Device name cleanup is best-effort, errors intentionally ignored
	_ = a.DeleteDeviceName(ctx, ieeeAddr)

	return nil
}

// ForceRemoveDevice removes a device from the coordinator's NVRAM directly.
// This is used when a device is offline/asleep and doesn't respond to MgmtLeaveReq.
// Unlike RemoveDevice, this only removes the coordinator's record - the device
// itself still thinks it's joined and will need to be factory reset.
func (a *Adapter) ForceRemoveDevice(ctx context.Context, ieeeAddr [8]byte) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	deleted, err := znpClient.DeleteAddrMgrEntry(ctx, ieeeAddr)
	if err != nil {
		return fmt.Errorf("failed to delete from NVRAM: %w", err)
	}

	if !deleted {
		return ErrDeviceNotFound
	}

	// Remove from local device manager.
	a.deviceMgr.removeDevice(ieeeAddr)

	// Clean up device name (non-fatal, best-effort)
	//nolint:errcheck // Device name cleanup is best-effort, errors intentionally ignored
	_ = a.DeleteDeviceName(ctx, ieeeAddr)

	return nil
}

// Bind creates a binding entry on a device, telling it where to send reports.
// This is essential for devices to automatically send attribute reports to the coordinator.
// Without binding, devices don't know where to send their reports.
//
// Typical usage: Bind a device's cluster to the coordinator so it reports state changes.
//
// Parameters:
//   - deviceIEEEAddr: IEEE address of the device to configure binding on
//   - deviceNwkAddr: Network address of the device
//   - deviceEndpoint: Source endpoint on the device
//   - clusterID: Cluster ID to bind (e.g., OnOff, Temperature)
//   - coordinatorIEEEAddr: IEEE address of the coordinator (destination)
//   - coordinatorEndpoint: Coordinator endpoint (typically 1)
//
// Example:
//
//	// Bind device's OnOff cluster to coordinator
//	err := adapter.Bind(ctx, deviceIEEE, deviceNwk, 1, zcl.ClusterOnOff, coordIEEE, 1)
func (a *Adapter) Bind(ctx context.Context, deviceIEEEAddr [8]byte, deviceNwkAddr uint16, deviceEndpoint uint8, clusterID uint16, coordinatorIEEEAddr [8]byte, coordinatorEndpoint uint8) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	status, err := znpClient.BindReq(ctx, deviceNwkAddr, deviceIEEEAddr, deviceEndpoint, clusterID, coordinatorIEEEAddr, coordinatorEndpoint)
	if err != nil {
		return fmt.Errorf("bind request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("bind returned status 0x%02X", status)
	}

	return nil
}

// Unbind removes a binding entry from a device.
// This stops the device from sending reports to the specified destination.
//
// Parameters:
//   - deviceIEEEAddr: IEEE address of the device to remove binding from
//   - deviceNwkAddr: Network address of the device
//   - deviceEndpoint: Source endpoint on the device
//   - clusterID: Cluster ID to unbind
//   - coordinatorIEEEAddr: IEEE address of the coordinator (destination)
//   - coordinatorEndpoint: Coordinator endpoint (typically 1)
func (a *Adapter) Unbind(ctx context.Context, deviceIEEEAddr [8]byte, deviceNwkAddr uint16, deviceEndpoint uint8, clusterID uint16, coordinatorIEEEAddr [8]byte, coordinatorEndpoint uint8) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	status, err := znpClient.UnbindReq(ctx, deviceNwkAddr, deviceIEEEAddr, deviceEndpoint, clusterID, coordinatorIEEEAddr, coordinatorEndpoint)
	if err != nil {
		return fmt.Errorf("unbind request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("unbind returned status 0x%02X", status)
	}

	return nil
}

package adapter

import (
	"context"
	"fmt"
	"time"
)

// WeakLink represents a link with poor signal quality.
type WeakLink struct {
	FromAddr uint16
	ToAddr   uint16
	LQI      uint8
}

// TopologyNode represents a device in the network topology.
type TopologyNode struct {
	NwkAddr    uint16
	IEEEAddr   [8]byte
	DeviceType uint8  // 0=Coordinator, 1=Router, 2=EndDevice.
	ParentAddr uint16 // Network address of parent.
	Depth      uint8
	LQI        uint8 // LQI to parent.
	Children   []uint16
}

// NetworkHealth contains overall network health metrics.
type NetworkHealth struct {
	Timestamp      time.Time
	Channel        uint8
	PanID          uint16
	DeviceCount    int
	RouterCount    int
	EndDeviceCount int
	AverageLQI     uint8
	MinLQI         uint8
	MaxLQI         uint8
	WeakLinks      []WeakLink // LQI < 100.
	Topology       []TopologyNode
}

// DeviceHealth contains health info for a single device.
type DeviceHealth struct {
	NwkAddr       uint16
	IEEEAddr      [8]byte
	LastSeen      time.Time
	LQI           uint8
	Depth         uint8
	NeighborCount int
	RouteCount    int
	IsReachable   bool
}

// RoutingInfo contains routing table information from a device.
type RoutingInfo struct {
	DstAddr      uint16
	Status       uint8 // 0=Active, 1=Discovery, 2=Failed, 3=Inactive, 4=Validation.
	NextHop      uint16
	Concentrator bool
	RouteRecord  bool
	ManyToOne    bool
}

// GetRoutingTable returns routing table from a device.
// Only routers and coordinators have routing tables; end devices will return an error.
func (a *Adapter) GetRoutingTable(ctx context.Context, nwkAddr uint16) ([]RoutingInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	entries, err := znpClient.GetAllRoutes(ctx, nwkAddr)
	if err != nil {
		return nil, fmt.Errorf("getting routing table: %w", err)
	}

	routes := make([]RoutingInfo, len(entries))
	for i, e := range entries {
		routes[i] = RoutingInfo{
			DstAddr:      e.DstAddr,
			Status:       e.Status,
			NextHop:      e.NextHop,
			Concentrator: e.Concentrator,
			RouteRecord:  e.RouteRecord,
			ManyToOne:    e.ManyToOne,
		}
	}

	return routes, nil
}

// GetNetworkHealth gathers comprehensive network health information.
// This queries all routers in the network and builds a topology map.
func (a *Adapter) GetNetworkHealth(ctx context.Context) (*NetworkHealth, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Get network info (channel, PAN ID).
	netInfo, err := a.GetNetworkInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting network info: %w", err)
	}

	// Get all devices from NVRAM.
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	health := &NetworkHealth{
		Timestamp:   time.Now(),
		Channel:     netInfo.Channel,
		PanID:       netInfo.PanID,
		DeviceCount: len(devices),
		MinLQI:      255,
		MaxLQI:      0,
		WeakLinks:   make([]WeakLink, 0),
		Topology:    make([]TopologyNode, 0),
	}

	if len(devices) == 0 {
		return health, nil
	}

	// Build topology by querying neighbor tables.
	// We'll also collect LQI statistics.
	var totalLQI int
	var lqiCount int

	// Query coordinator's neighbor table first.
	coordNeighbors, err := a.GetNeighborTable(ctx, 0x0000)
	if err == nil {
		coordNode := TopologyNode{
			NwkAddr:    0x0000,
			IEEEAddr:   netInfo.IEEEAddr,
			DeviceType: 0, // Coordinator.
			ParentAddr: 0xFFFF,
			Depth:      0,
			LQI:        255,
			Children:   make([]uint16, 0),
		}

		// Process coordinator's neighbors.
		for _, n := range coordNeighbors {
			totalLQI += int(n.LQI)
			lqiCount++

			if n.LQI < health.MinLQI {
				health.MinLQI = n.LQI
			}
			if n.LQI > health.MaxLQI {
				health.MaxLQI = n.LQI
			}

			if n.LQI < 100 {
				health.WeakLinks = append(health.WeakLinks, WeakLink{
					FromAddr: 0x0000,
					ToAddr:   n.NwkAddr,
					LQI:      n.LQI,
				})
			}

			// Track children..
			if n.Relation == 1 { // Child relationship..
				coordNode.Children = append(coordNode.Children, n.NwkAddr)
			}
		}

		health.Topology = append(health.Topology, coordNode)
	}

	// Query each device.
	for _, dev := range devices {
		// Try to determine if device is a router by querying neighbor table.
		// End devices typically don't respond or timeout.
		queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		neighbors, err := a.GetNeighborTable(queryCtx, dev.NwkAddr)
		cancel()

		isRouter := (err == nil)
		deviceType := uint8(2) // EndDevice.
		if isRouter {
			deviceType = 1 // Router.
			health.RouterCount++
		} else {
			health.EndDeviceCount++
		}

		node := TopologyNode{
			NwkAddr:    dev.NwkAddr,
			IEEEAddr:   dev.IEEEAddr,
			DeviceType: deviceType,
			ParentAddr: 0xFFFF, // Unknown by default.
			Depth:      0,
			LQI:        0,
			Children:   make([]uint16, 0),
		}

		// Process neighbors if this is a router.
		if isRouter {
			for _, n := range neighbors {
				totalLQI += int(n.LQI)
				lqiCount++

				if n.LQI < health.MinLQI {
					health.MinLQI = n.LQI
				}
				if n.LQI > health.MaxLQI {
					health.MaxLQI = n.LQI
				}

				if n.LQI < 100 {
					health.WeakLinks = append(health.WeakLinks, WeakLink{
						FromAddr: dev.NwkAddr,
						ToAddr:   n.NwkAddr,
						LQI:      n.LQI,
					})
				}

				// Determine parent (relation 0 = parent).
				if n.Relation == 0 {
					node.ParentAddr = n.NwkAddr
					node.LQI = n.LQI
					node.Depth = n.Depth + 1
				}

				// Track children.
				if n.Relation == 1 { // Child relationship.
					node.Children = append(node.Children, n.NwkAddr)
				}
			}
		}

		health.Topology = append(health.Topology, node)
	}

	// Calculate average LQI.
	if lqiCount > 0 {
		health.AverageLQI = uint8(totalLQI / lqiCount)
	}

	// If no LQI data collected, reset min to 0.
	if lqiCount == 0 {
		health.MinLQI = 0
	}

	return health, nil
}

// GetDeviceHealth gets health info for a specific device.
func (a *Adapter) GetDeviceHealth(ctx context.Context, nwkAddr uint16) (*DeviceHealth, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Try to get device from NVRAM.
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	var foundDevice *Device
	for _, dev := range devices {
		if dev.NwkAddr == nwkAddr {
			foundDevice = dev
			break
		}
	}

	if foundDevice == nil && nwkAddr != 0x0000 {
		return nil, fmt.Errorf("device 0x%04X not found", nwkAddr)
	}

	health := &DeviceHealth{
		NwkAddr:     nwkAddr,
		LastSeen:    time.Now(),
		IsReachable: false,
	}

	if foundDevice != nil {
		health.IEEEAddr = foundDevice.IEEEAddr
	} else if nwkAddr == 0x0000 {
		// Special case for coordinator.
		netInfo, err := a.GetNetworkInfo(ctx)
		if err == nil {
			health.IEEEAddr = netInfo.IEEEAddr
		}
	}

	// Try to query neighbor table (only routers and coordinator respond).
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	neighbors, err := a.GetNeighborTable(queryCtx, nwkAddr)
	cancel()

	if err == nil {
		health.IsReachable = true
		health.NeighborCount = len(neighbors)

		// Find parent and get LQI.
		for _, n := range neighbors {
			if n.Relation == 0 { // Parent.
				health.LQI = n.LQI
				health.Depth = n.Depth + 1
				break
			}
		}
	}

	// Try to get routing table count.
	routeCtx, routeCancel := context.WithTimeout(ctx, 3*time.Second)
	routes, err := a.GetRoutingTable(routeCtx, nwkAddr)
	routeCancel()

	if err == nil {
		health.RouteCount = len(routes)
	}

	return health, nil
}

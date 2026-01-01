package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
)

// DeviceType represents the logical type of a Zigbee device.
type DeviceType uint8

const (
	DeviceTypeCoordinator DeviceType = 0
	DeviceTypeRouter      DeviceType = 1
	DeviceTypeEndDevice   DeviceType = 2
)

// String returns human-readable device type name.
func (dt DeviceType) String() string {
	switch dt {
	case DeviceTypeCoordinator:
		return "Coordinator"
	case DeviceTypeRouter:
		return "Router"
	case DeviceTypeEndDevice:
		return "EndDevice"
	default:
		return fmt.Sprintf("Unknown(%d)", dt)
	}
}

// PowerSource represents how a device is powered.
type PowerSource uint8

const (
	PowerSourceUnknown          PowerSource = 0x00
	PowerSourceMains            PowerSource = 0x01
	PowerSourceBattery          PowerSource = 0x03
	PowerSourceDC               PowerSource = 0x04
	PowerSourceEmergencyMains   PowerSource = 0x05
	PowerSourceEmergencyBattery PowerSource = 0x06
)

// String returns human-readable power source name.
func (ps PowerSource) String() string {
	switch ps {
	case PowerSourceMains:
		return "Mains"
	case PowerSourceBattery:
		return "Battery"
	case PowerSourceDC:
		return "DC"
	case PowerSourceEmergencyMains:
		return "Emergency Mains"
	case PowerSourceEmergencyBattery:
		return "Emergency Battery"
	default:
		return "Unknown"
	}
}

// EndpointInfo contains cluster information for an endpoint.
type EndpointInfo struct {
	Endpoint      uint8
	ProfileID     uint16
	DeviceID      uint16
	DeviceVersion uint8
	InClusters    []uint16
	OutClusters   []uint16
}

// HasCluster returns true if the endpoint has the specified input cluster.
func (e *EndpointInfo) HasCluster(clusterID uint16) bool {
	for _, c := range e.InClusters {
		if c == clusterID {
			return true
		}
	}
	return false
}

// InterviewResult contains all discovered information about a device.
type InterviewResult struct {
	// Device addressing
	IEEEAddr [8]byte
	NwkAddr  uint16

	// From NodeDescriptor
	DeviceType       DeviceType
	ManufacturerCode uint16

	// From Basic cluster
	Manufacturer string
	Model        string
	PowerSource  PowerSource
	SWBuildID    string // Software build ID (optional)

	// All endpoints with their clusters
	Endpoints []EndpointInfo

	// Interview metadata
	InterviewedAt time.Time
	Success       bool
	Errors        []string // Non-fatal errors encountered during interview
}

// IEEEAddrString returns the IEEE address as a hex string.
func (r *InterviewResult) IEEEAddrString() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		r.IEEEAddr[7], r.IEEEAddr[6], r.IEEEAddr[5], r.IEEEAddr[4],
		r.IEEEAddr[3], r.IEEEAddr[2], r.IEEEAddr[1], r.IEEEAddr[0])
}

// FindEndpointWithCluster returns the first endpoint that has the specified cluster.
func (r *InterviewResult) FindEndpointWithCluster(clusterID uint16) *EndpointInfo {
	for i := range r.Endpoints {
		if r.Endpoints[i].HasCluster(clusterID) {
			return &r.Endpoints[i]
		}
	}
	return nil
}

// InterviewOptions configures the interview process.
type InterviewOptions struct {
	// Timeout per individual ZDO/ZCL request (default: 10s)
	RequestTimeout time.Duration

	// Skip reading Basic cluster attributes (faster but less info)
	SkipBasicCluster bool

	// Number of retries for failed requests (default: 1)
	Retries int
}

// DefaultInterviewOptions returns sensible defaults for interviewing.
func DefaultInterviewOptions() InterviewOptions {
	return InterviewOptions{
		RequestTimeout: 10 * time.Second,
		Retries:        1,
	}
}

// InterviewDevice performs a full device interview to discover capabilities.
// This queries the device for its node descriptor, endpoints, clusters, and
// basic identification attributes (manufacturer, model).
func (a *Adapter) InterviewDevice(ctx context.Context, nwkAddr uint16) (*InterviewResult, error) {
	var emptyIEEE [8]byte
	return a.interviewDeviceInternal(ctx, nwkAddr, emptyIEEE, DefaultInterviewOptions())
}

// InterviewDeviceWithOptions performs device interview with custom options.
func (a *Adapter) InterviewDeviceWithOptions(ctx context.Context, nwkAddr uint16, opts InterviewOptions) (*InterviewResult, error) {
	var emptyIEEE [8]byte
	return a.interviewDeviceInternal(ctx, nwkAddr, emptyIEEE, opts)
}

// InterviewDeviceWithAddr performs interview when both addresses are known.
// This skips the IeeeAddrReq call which is useful for sleeping end devices.
func (a *Adapter) InterviewDeviceWithAddr(ctx context.Context, nwkAddr uint16, ieeeAddr [8]byte) (*InterviewResult, error) {
	return a.interviewDeviceInternal(ctx, nwkAddr, ieeeAddr, DefaultInterviewOptions())
}

// interviewDeviceInternal is the core interview implementation.
func (a *Adapter) interviewDeviceInternal(ctx context.Context, nwkAddr uint16, knownIEEE [8]byte, opts InterviewOptions) (*InterviewResult, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	result := &InterviewResult{
		NwkAddr:       nwkAddr,
		InterviewedAt: time.Now(),
		Success:       true,
	}

	// Helper to add non-fatal error
	addError := func(msg string) {
		result.Errors = append(result.Errors, msg)
	}

	// Helper for request with timeout
	reqCtx := func() (context.Context, context.CancelFunc) {
		if opts.RequestTimeout > 0 {
			return context.WithTimeout(ctx, opts.RequestTimeout)
		}
		return ctx, func() {}
	}

	// Step 1: Get IEEE address
	// If we have a known IEEE address, use it as fallback if the request fails
	// (sleeping devices won't respond, but routers will)
	var emptyIEEE [8]byte
	rctx, cancel := reqCtx()
	ieeeAddr, err := znpClient.IeeeAddrReq(rctx, nwkAddr)
	cancel()
	if err != nil {
		if knownIEEE != emptyIEEE {
			// Use the known address as fallback
			result.IEEEAddr = knownIEEE
			addError(fmt.Sprintf("ieee addr request: %v (using known address)", err))
		} else {
			// No fallback available - this is fatal
			return nil, fmt.Errorf("failed to get IEEE address: %w", err)
		}
	} else {
		result.IEEEAddr = ieeeAddr
	}

	// Step 2: Get node descriptor (for device type and manufacturer code)
	rctx, cancel = reqCtx()
	nodeDesc, err := znpClient.NodeDescReq(rctx, nwkAddr)
	cancel()
	if err != nil {
		addError(fmt.Sprintf("node descriptor: %v", err))
	} else {
		result.DeviceType = DeviceType(nodeDesc.LogicalType)
		result.ManufacturerCode = nodeDesc.ManufCode
	}

	// Step 3: Get active endpoints
	rctx, cancel = reqCtx()
	activeEp, err := znpClient.ActiveEpReq(rctx, nwkAddr)
	cancel()
	if err != nil {
		addError(fmt.Sprintf("active endpoints: %v", err))
		// No endpoints means we can't continue with cluster discovery
		result.Success = len(result.Errors) == 0
		return result, nil
	}

	if activeEp.Status != 0 {
		addError(fmt.Sprintf("active endpoints status: 0x%02X", activeEp.Status))
		result.Success = len(result.Errors) == 0
		return result, nil
	}

	// Step 4: Get simple descriptor for each endpoint
	for _, ep := range activeEp.Endpoints {
		rctx, cancel = reqCtx()
		desc, err := znpClient.SimpleDescReq(rctx, nwkAddr, ep)
		cancel()
		if err != nil {
			addError(fmt.Sprintf("endpoint %d simple desc: %v", ep, err))
			continue
		}

		result.Endpoints = append(result.Endpoints, EndpointInfo{
			Endpoint:      desc.Endpoint,
			ProfileID:     desc.ProfileID,
			DeviceID:      desc.DeviceID,
			DeviceVersion: desc.DeviceVersion,
			InClusters:    desc.InClusters,
			OutClusters:   desc.OutClusters,
		})
	}

	// Step 5: Read Basic cluster attributes (if not skipped)
	if !opts.SkipBasicCluster && len(result.Endpoints) > 0 {
		// Find endpoint with Basic cluster, prefer endpoint 1
		var basicEndpoint uint8
		for _, ep := range result.Endpoints {
			if ep.HasCluster(uint16(zcl.ClusterBasic)) {
				basicEndpoint = ep.Endpoint
				// Prefer endpoint 1 if it has Basic cluster
				if ep.Endpoint == 1 {
					break
				}
			}
		}

		if basicEndpoint > 0 {
			a.readBasicAttributes(ctx, nwkAddr, basicEndpoint, result, opts)
		} else {
			addError("no endpoint with Basic cluster found")
		}
	}

	result.Success = len(result.Errors) == 0
	return result, nil
}

// readBasicAttributes reads manufacturer, model, and power source from Basic cluster.
func (a *Adapter) readBasicAttributes(ctx context.Context, nwkAddr uint16, endpoint uint8, result *InterviewResult, opts InterviewOptions) {
	addError := func(msg string) {
		result.Errors = append(result.Errors, msg)
	}

	// Read multiple attributes in one request
	attrs := []zcl.AttributeID{
		zcl.AttrBasicManufacturerName,
		zcl.AttrBasicModelIdentifier,
		zcl.AttrBasicPowerSource,
		zcl.AttrBasicSWBuildID,
	}

	var readCtx context.Context
	var cancel context.CancelFunc
	if opts.RequestTimeout > 0 {
		readCtx, cancel = context.WithTimeout(ctx, opts.RequestTimeout)
	} else {
		readCtx, cancel = ctx, func() {}
	}
	defer cancel()

	results, err := a.ReadAttributes(readCtx, nwkAddr, endpoint, zcl.ClusterBasic, attrs...)
	if err != nil {
		addError(fmt.Sprintf("basic cluster read: %v", err))
		return
	}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrBasicManufacturerName:
			if s, ok := r.Value.(string); ok {
				result.Manufacturer = s
			}
		case zcl.AttrBasicModelIdentifier:
			if s, ok := r.Value.(string); ok {
				result.Model = s
			}
		case zcl.AttrBasicPowerSource:
			if ps, ok := r.Value.(uint8); ok {
				result.PowerSource = PowerSource(ps)
			}
		case zcl.AttrBasicSWBuildID:
			if s, ok := r.Value.(string); ok {
				result.SWBuildID = s
			}
		}
	}
}

// InterviewDeviceByIEEE performs interview when IEEE address is known.
// It first resolves the network address from the device manager.
func (a *Adapter) InterviewDeviceByIEEE(ctx context.Context, ieeeAddr [8]byte) (*InterviewResult, error) {
	// Try to find device in manager
	dev := a.deviceMgr.getDevice(ieeeAddr)
	if dev == nil {
		return nil, fmt.Errorf("device %02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x not found",
			ieeeAddr[7], ieeeAddr[6], ieeeAddr[5], ieeeAddr[4],
			ieeeAddr[3], ieeeAddr[2], ieeeAddr[1], ieeeAddr[0])
	}

	return a.InterviewDevice(ctx, dev.NwkAddr)
}

// InterviewAllDevices interviews all known devices from the NVRAM.
// Returns results for each device. Errors for individual devices are recorded
// in the InterviewResult.Errors field rather than failing the entire operation.
func (a *Adapter) InterviewAllDevices(ctx context.Context) ([]*InterviewResult, error) {
	return a.InterviewAllDevicesWithOptions(ctx, DefaultInterviewOptions())
}

// InterviewAllDevicesWithOptions interviews all devices with custom options.
func (a *Adapter) InterviewAllDevicesWithOptions(ctx context.Context, opts InterviewOptions) ([]*InterviewResult, error) {
	// Get device list from NVRAM
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting device list: %w", err)
	}

	results := make([]*InterviewResult, 0, len(devices))

	for _, dev := range devices {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		// Use known IEEE address to skip IeeeAddrReq (helps with sleeping devices)
		result, err := a.interviewDeviceInternal(ctx, dev.NwkAddr, dev.IEEEAddr, opts)
		if err != nil {
			// Create a result with the error
			result = &InterviewResult{
				IEEEAddr:      dev.IEEEAddr,
				NwkAddr:       dev.NwkAddr,
				InterviewedAt: time.Now(),
				Success:       false,
				Errors:        []string{err.Error()},
			}
		}

		results = append(results, result)
	}

	return results, nil
}

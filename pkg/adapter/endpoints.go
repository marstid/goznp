package adapter

import (
	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

// Endpoint definitions for the coordinator.
// Following zigbee-herdsman pattern: register multiple endpoints with different profiles
// so we can communicate with devices of any profile type.
//
// Reference: https://github.com/Koenkk/zigbee-herdsman/blob/master/src/adapter/z-stack/adapter/endpoints.ts

// EndpointDef defines a coordinator endpoint configuration.
type EndpointDef struct {
	Endpoint    uint8
	ProfileID   znp.ApplicationProfile
	DeviceID    uint16
	InClusters  []uint16
	OutClusters []uint16
}

// Standard coordinator endpoints.
// Each endpoint is registered with a specific Application Profile to enable
// communication with devices using that profile.
var CoordinatorEndpoints = []EndpointDef{
	// Endpoint 1: Home Automation (0x0104) - Primary endpoint
	{
		Endpoint:  1,
		ProfileID: znp.ProfileHomeAutomation,
		DeviceID:  0x0005, // Configuration Tool
		InClusters: []uint16{
			uint16(zcl.ClusterBasic),
		},
		OutClusters: []uint16{
			uint16(zcl.ClusterBasic),
			uint16(zcl.ClusterOnOff),
			uint16(zcl.ClusterLevelControl),
			uint16(zcl.ClusterColorControl),
			uint16(zcl.ClusterPowerConfig),
		},
	},
	// Endpoint 2: Smart Energy (0x0109)
	{
		Endpoint:  2,
		ProfileID: znp.ProfileSmartEnergy,
		DeviceID:  0x0005,
		InClusters: []uint16{
			uint16(zcl.ClusterBasic),
		},
		OutClusters: []uint16{
			uint16(zcl.ClusterBasic),
		},
	},
	// Endpoint 3: Green Power (0xA1E0)
	{
		Endpoint:  3,
		ProfileID: znp.ProfileGreenPower,
		DeviceID:  0x0005,
		InClusters: []uint16{
			uint16(zcl.ClusterBasic),
		},
		OutClusters: []uint16{
			uint16(zcl.ClusterBasic),
		},
	},
	// Endpoint 11: Green Power Proxy (standard GP endpoint)
	{
		Endpoint:  11,
		ProfileID: znp.ProfileGreenPower,
		DeviceID:  0x0066, // GP Proxy
		InClusters: []uint16{
			0x0021, // Green Power cluster
		},
		OutClusters: []uint16{
			0x0021,
		},
	},
	// Endpoint 110: ZLL/Touchlink (0xC05E)
	{
		Endpoint:  110,
		ProfileID: znp.ProfileLightLink,
		DeviceID:  0x0005,
		InClusters: []uint16{
			uint16(zcl.ClusterBasic),
		},
		OutClusters: []uint16{
			uint16(zcl.ClusterBasic),
			uint16(zcl.ClusterOnOff),
			uint16(zcl.ClusterLevelControl),
			uint16(zcl.ClusterColorControl),
		},
	},
	// Endpoint 242: Green Power Sink
	{
		Endpoint:  242,
		ProfileID: znp.ProfileGreenPower,
		DeviceID:  0x0061, // GP Target
		InClusters: []uint16{
			0x0021,
		},
		OutClusters: []uint16{
			0x0021,
		},
	},
}

// ProfileToEndpoint maps application profiles to their primary endpoint.
var ProfileToEndpoint = map[znp.ApplicationProfile]uint8{
	znp.ProfileHomeAutomation: 1,
	znp.ProfileSmartEnergy:    2,
	znp.ProfileGreenPower:     3,
	znp.ProfileLightLink:      110,
}

// GetEndpointForProfile returns the coordinator endpoint to use for a given profile.
// Returns endpoint 1 (HA) as default for unknown profiles.
func GetEndpointForProfile(profile znp.ApplicationProfile) uint8 {
	if ep, ok := ProfileToEndpoint[profile]; ok {
		return ep
	}
	return 1 // Default to HA endpoint
}

// RegisteredEndpoints holds the list of successfully registered endpoints.
type RegisteredEndpoints struct {
	Endpoints []uint8
	Profiles  map[uint8]znp.ApplicationProfile
}

// NewRegisteredEndpoints creates a new RegisteredEndpoints tracker.
func NewRegisteredEndpoints() *RegisteredEndpoints {
	return &RegisteredEndpoints{
		Endpoints: make([]uint8, 0),
		Profiles:  make(map[uint8]znp.ApplicationProfile),
	}
}

// Add records a successfully registered endpoint.
func (r *RegisteredEndpoints) Add(endpoint uint8, profile znp.ApplicationProfile) {
	r.Endpoints = append(r.Endpoints, endpoint)
	r.Profiles[endpoint] = profile
}

// HasProfile returns true if any endpoint with the given profile is registered.
func (r *RegisteredEndpoints) HasProfile(profile znp.ApplicationProfile) bool {
	for _, p := range r.Profiles {
		if p == profile {
			return true
		}
	}
	return false
}

// GetProfiles returns all registered profiles.
func (r *RegisteredEndpoints) GetProfiles() []znp.ApplicationProfile {
	seen := make(map[znp.ApplicationProfile]bool)
	profiles := make([]znp.ApplicationProfile, 0)
	for _, p := range r.Profiles {
		if !seen[p] {
			seen[p] = true
			profiles = append(profiles, p)
		}
	}
	return profiles
}

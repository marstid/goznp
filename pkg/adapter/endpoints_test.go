package adapter

import (
	"testing"

	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

func TestNewRegisteredEndpoints(t *testing.T) {
	re := NewRegisteredEndpoints()
	if re == nil {
		t.Fatal("NewRegisteredEndpoints returned nil")
	}

	if re.Endpoints == nil {
		t.Error("Endpoints slice is nil")
	}

	if re.Profiles == nil {
		t.Error("Profiles map is nil")
	}

	if len(re.Endpoints) != 0 {
		t.Errorf("expected empty Endpoints slice, got length %d", len(re.Endpoints))
	}

	if len(re.Profiles) != 0 {
		t.Errorf("expected empty Profiles map, got length %d", len(re.Profiles))
	}
}

func TestRegisteredEndpoints_Add(t *testing.T) {
	tests := []struct {
		name     string
		endpoint uint8
		profile  znp.ApplicationProfile
	}{
		{
			name:     "add home automation endpoint",
			endpoint: 1,
			profile:  znp.ProfileHomeAutomation,
		},
		{
			name:     "add smart energy endpoint",
			endpoint: 2,
			profile:  znp.ProfileSmartEnergy,
		},
		{
			name:     "add green power endpoint",
			endpoint: 242,
			profile:  znp.ProfileGreenPower,
		},
		{
			name:     "add light link endpoint",
			endpoint: 110,
			profile:  znp.ProfileLightLink,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := NewRegisteredEndpoints()
			re.Add(tt.endpoint, tt.profile)

			if len(re.Endpoints) != 1 {
				t.Errorf("expected 1 endpoint, got %d", len(re.Endpoints))
			}

			if re.Endpoints[0] != tt.endpoint {
				t.Errorf("expected endpoint %d, got %d", tt.endpoint, re.Endpoints[0])
			}

			if re.Profiles[tt.endpoint] != tt.profile {
				t.Errorf("expected profile %v for endpoint %d, got %v", tt.profile, tt.endpoint, re.Profiles[tt.endpoint])
			}
		})
	}
}

func TestRegisteredEndpoints_AddMultiple(t *testing.T) {
	re := NewRegisteredEndpoints()

	re.Add(1, znp.ProfileHomeAutomation)
	re.Add(2, znp.ProfileSmartEnergy)
	re.Add(3, znp.ProfileGreenPower)

	if len(re.Endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(re.Endpoints))
	}

	if len(re.Profiles) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(re.Profiles))
	}

	// Verify all endpoints are present
	expectedEndpoints := map[uint8]bool{1: true, 2: true, 3: true}
	for _, ep := range re.Endpoints {
		if !expectedEndpoints[ep] {
			t.Errorf("unexpected endpoint %d in list", ep)
		}
	}

	// Verify profile mappings
	if re.Profiles[1] != znp.ProfileHomeAutomation {
		t.Error("endpoint 1 should have Home Automation profile")
	}
	if re.Profiles[2] != znp.ProfileSmartEnergy {
		t.Error("endpoint 2 should have Smart Energy profile")
	}
	if re.Profiles[3] != znp.ProfileGreenPower {
		t.Error("endpoint 3 should have Green Power profile")
	}
}

func TestRegisteredEndpoints_GetProfiles(t *testing.T) {
	tests := []struct {
		name      string
		endpoints []struct {
			ep      uint8
			profile znp.ApplicationProfile
		}
		wantProfilesCount int
		wantProfiles      map[znp.ApplicationProfile]bool
	}{
		{
			name: "empty endpoints",
			endpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{},
			wantProfilesCount: 0,
			wantProfiles:      map[znp.ApplicationProfile]bool{},
		},
		{
			name: "single profile",
			endpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{
				{1, znp.ProfileHomeAutomation},
			},
			wantProfilesCount: 1,
			wantProfiles: map[znp.ApplicationProfile]bool{
				znp.ProfileHomeAutomation: true,
			},
		},
		{
			name: "multiple unique profiles",
			endpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{
				{1, znp.ProfileHomeAutomation},
				{2, znp.ProfileSmartEnergy},
				{3, znp.ProfileGreenPower},
			},
			wantProfilesCount: 3,
			wantProfiles: map[znp.ApplicationProfile]bool{
				znp.ProfileHomeAutomation: true,
				znp.ProfileSmartEnergy:    true,
				znp.ProfileGreenPower:     true,
			},
		},
		{
			name: "duplicate profiles",
			endpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{
				{1, znp.ProfileHomeAutomation},
				{11, znp.ProfileGreenPower},
				{242, znp.ProfileGreenPower},
			},
			wantProfilesCount: 2,
			wantProfiles: map[znp.ApplicationProfile]bool{
				znp.ProfileHomeAutomation: true,
				znp.ProfileGreenPower:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := NewRegisteredEndpoints()
			for _, ep := range tt.endpoints {
				re.Add(ep.ep, ep.profile)
			}

			profiles := re.GetProfiles()

			if len(profiles) != tt.wantProfilesCount {
				t.Errorf("expected %d profiles, got %d", tt.wantProfilesCount, len(profiles))
			}

			// Verify all returned profiles are expected
			for _, profile := range profiles {
				if !tt.wantProfiles[profile] {
					t.Errorf("unexpected profile %v in result", profile)
				}
			}
		})
	}
}

func TestRegisteredEndpoints_HasProfile(t *testing.T) {
	tests := []struct {
		name         string
		addEndpoints []struct {
			ep      uint8
			profile znp.ApplicationProfile
		}
		checkProfile   znp.ApplicationProfile
		expectedResult bool
	}{
		{
			name: "empty endpoints - check HA",
			addEndpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{},
			checkProfile:   znp.ProfileHomeAutomation,
			expectedResult: false,
		},
		{
			name: "has home automation",
			addEndpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{
				{1, znp.ProfileHomeAutomation},
			},
			checkProfile:   znp.ProfileHomeAutomation,
			expectedResult: true,
		},
		{
			name: "does not have smart energy",
			addEndpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{
				{1, znp.ProfileHomeAutomation},
			},
			checkProfile:   znp.ProfileSmartEnergy,
			expectedResult: false,
		},
		{
			name: "has green power from multiple endpoints",
			addEndpoints: []struct {
				ep      uint8
				profile znp.ApplicationProfile
			}{
				{1, znp.ProfileHomeAutomation},
				{11, znp.ProfileGreenPower},
				{242, znp.ProfileGreenPower},
			},
			checkProfile:   znp.ProfileGreenPower,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := NewRegisteredEndpoints()
			for _, ep := range tt.addEndpoints {
				re.Add(ep.ep, ep.profile)
			}

			result := re.HasProfile(tt.checkProfile)
			if result != tt.expectedResult {
				t.Errorf("HasProfile(%v) = %v, want %v", tt.checkProfile, result, tt.expectedResult)
			}
		})
	}
}

func TestGetEndpointForProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile znp.ApplicationProfile
		want    uint8
	}{
		{
			name:    "home automation profile",
			profile: znp.ProfileHomeAutomation,
			want:    1,
		},
		{
			name:    "smart energy profile",
			profile: znp.ProfileSmartEnergy,
			want:    2,
		},
		{
			name:    "green power profile",
			profile: znp.ProfileGreenPower,
			want:    3,
		},
		{
			name:    "light link profile",
			profile: znp.ProfileLightLink,
			want:    110,
		},
		{
			name:    "unknown profile defaults to HA",
			profile: znp.ApplicationProfile(0x9999),
			want:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEndpointForProfile(tt.profile)
			if got != tt.want {
				t.Errorf("GetEndpointForProfile(%v) = %d, want %d", tt.profile, got, tt.want)
			}
		})
	}
}

func TestProfileToEndpoint(t *testing.T) {
	// Verify the ProfileToEndpoint map has expected mappings
	expectedMappings := map[znp.ApplicationProfile]uint8{
		znp.ProfileHomeAutomation: 1,
		znp.ProfileSmartEnergy:    2,
		znp.ProfileGreenPower:     3,
		znp.ProfileLightLink:      110,
	}

	if len(ProfileToEndpoint) != len(expectedMappings) {
		t.Errorf("ProfileToEndpoint has %d entries, expected %d", len(ProfileToEndpoint), len(expectedMappings))
	}

	for profile, expectedEP := range expectedMappings {
		if ep, ok := ProfileToEndpoint[profile]; !ok {
			t.Errorf("ProfileToEndpoint missing mapping for profile %v", profile)
		} else if ep != expectedEP {
			t.Errorf("ProfileToEndpoint[%v] = %d, want %d", profile, ep, expectedEP)
		}
	}
}

func TestCoordinatorEndpoints(t *testing.T) {
	// Verify we have the expected number of endpoint definitions
	expectedCount := 6
	if len(CoordinatorEndpoints) != expectedCount {
		t.Errorf("CoordinatorEndpoints has %d entries, expected %d", len(CoordinatorEndpoints), expectedCount)
	}

	// Test each endpoint definition
	tests := []struct {
		name        string
		endpoint    uint8
		profileID   znp.ApplicationProfile
		deviceID    uint16
		hasInBasic  bool
		hasOutBasic bool
	}{
		{
			name:        "Endpoint 1 - Home Automation",
			endpoint:    1,
			profileID:   znp.ProfileHomeAutomation,
			deviceID:    0x0005,
			hasInBasic:  true,
			hasOutBasic: true,
		},
		{
			name:        "Endpoint 2 - Smart Energy",
			endpoint:    2,
			profileID:   znp.ProfileSmartEnergy,
			deviceID:    0x0005,
			hasInBasic:  true,
			hasOutBasic: true,
		},
		{
			name:        "Endpoint 3 - Green Power",
			endpoint:    3,
			profileID:   znp.ProfileGreenPower,
			deviceID:    0x0005,
			hasInBasic:  true,
			hasOutBasic: true,
		},
		{
			name:        "Endpoint 11 - Green Power Proxy",
			endpoint:    11,
			profileID:   znp.ProfileGreenPower,
			deviceID:    0x0066,
			hasInBasic:  false,
			hasOutBasic: false,
		},
		{
			name:        "Endpoint 110 - Light Link",
			endpoint:    110,
			profileID:   znp.ProfileLightLink,
			deviceID:    0x0005,
			hasInBasic:  true,
			hasOutBasic: true,
		},
		{
			name:        "Endpoint 242 - Green Power Sink",
			endpoint:    242,
			profileID:   znp.ProfileGreenPower,
			deviceID:    0x0061,
			hasInBasic:  false,
			hasOutBasic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find the endpoint definition
			var def *EndpointDef
			for i := range CoordinatorEndpoints {
				if CoordinatorEndpoints[i].Endpoint == tt.endpoint {
					def = &CoordinatorEndpoints[i]
					break
				}
			}

			if def == nil {
				t.Fatalf("Endpoint %d not found in CoordinatorEndpoints", tt.endpoint)
			}

			// Verify endpoint properties
			if def.Endpoint != tt.endpoint {
				t.Errorf("Endpoint = %d, want %d", def.Endpoint, tt.endpoint)
			}

			if def.ProfileID != tt.profileID {
				t.Errorf("ProfileID = %v, want %v", def.ProfileID, tt.profileID)
			}

			if def.DeviceID != tt.deviceID {
				t.Errorf("DeviceID = 0x%04X, want 0x%04X", def.DeviceID, tt.deviceID)
			}

			// Verify cluster configurations
			if def.InClusters == nil {
				t.Error("InClusters is nil")
			}

			if def.OutClusters == nil {
				t.Error("OutClusters is nil")
			}

			// Check for Basic cluster presence
			hasInBasic := containsCluster(def.InClusters, uint16(zcl.ClusterBasic))
			if hasInBasic != tt.hasInBasic {
				t.Errorf("InClusters contains Basic = %v, want %v", hasInBasic, tt.hasInBasic)
			}

			hasOutBasic := containsCluster(def.OutClusters, uint16(zcl.ClusterBasic))
			if hasOutBasic != tt.hasOutBasic {
				t.Errorf("OutClusters contains Basic = %v, want %v", hasOutBasic, tt.hasOutBasic)
			}
		})
	}
}

func TestCoordinatorEndpoints_HAEndpoint(t *testing.T) {
	// Test Home Automation endpoint in detail (most commonly used)
	var haEndpoint *EndpointDef
	for i := range CoordinatorEndpoints {
		if CoordinatorEndpoints[i].Endpoint == 1 {
			haEndpoint = &CoordinatorEndpoints[i]
			break
		}
	}

	if haEndpoint == nil {
		t.Fatal("Home Automation endpoint (1) not found")
	}

	// Verify it has essential output clusters for HA devices
	requiredOutClusters := []uint16{
		uint16(zcl.ClusterBasic),
		uint16(zcl.ClusterOnOff),
		uint16(zcl.ClusterLevelControl),
		uint16(zcl.ClusterColorControl),
	}

	for _, cluster := range requiredOutClusters {
		if !containsCluster(haEndpoint.OutClusters, cluster) {
			t.Errorf("HA endpoint missing required output cluster 0x%04X", cluster)
		}
	}
}

func TestCoordinatorEndpoints_GreenPowerEndpoints(t *testing.T) {
	// Verify we have multiple Green Power endpoints
	gpEndpoints := make([]EndpointDef, 0)
	for _, def := range CoordinatorEndpoints {
		if def.ProfileID == znp.ProfileGreenPower {
			gpEndpoints = append(gpEndpoints, def)
		}
	}

	if len(gpEndpoints) != 3 {
		t.Errorf("expected 3 Green Power endpoints, found %d", len(gpEndpoints))
	}

	// Verify each has the Green Power cluster (0x0021)
	gpCluster := uint16(0x0021)
	for _, def := range gpEndpoints {
		hasInGP := containsCluster(def.InClusters, gpCluster)
		hasOutGP := containsCluster(def.OutClusters, gpCluster)

		// GP endpoints should have 0x0021 in their clusters
		if def.Endpoint == 11 || def.Endpoint == 242 {
			if !hasInGP {
				t.Errorf("GP endpoint %d missing Green Power cluster in InClusters", def.Endpoint)
			}
			if !hasOutGP {
				t.Errorf("GP endpoint %d missing Green Power cluster in OutClusters", def.Endpoint)
			}
		}
	}
}

func TestCoordinatorEndpoints_UniqueEndpointNumbers(t *testing.T) {
	// Verify all endpoint numbers are unique
	seen := make(map[uint8]bool)
	for _, def := range CoordinatorEndpoints {
		if seen[def.Endpoint] {
			t.Errorf("duplicate endpoint number %d found", def.Endpoint)
		}
		seen[def.Endpoint] = true
	}
}

// Helper function to check if a cluster ID exists in a slice
func containsCluster(clusters []uint16, cluster uint16) bool {
	for _, c := range clusters {
		if c == cluster {
			return true
		}
	}
	return false
}

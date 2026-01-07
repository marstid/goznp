package quirks

import (
	"testing"

	"github.com/marstid/goznp/pkg/zcl"
)

func TestDeviceMatcher_Matches(t *testing.T) {
	tests := []struct {
		name          string
		matcher       DeviceMatcher
		manufCode     uint16
		manufacturer  string
		model         string
		expectedMatch bool
	}{
		{
			name: "exact manufacturer code match",
			matcher: DeviceMatcher{
				ManufacturerCode: ptrUint16(0x117C),
			},
			manufCode:     0x117C,
			manufacturer:  "IKEA",
			model:         "TRADFRI",
			expectedMatch: true,
		},
		{
			name: "manufacturer code mismatch",
			matcher: DeviceMatcher{
				ManufacturerCode: ptrUint16(0x117C),
			},
			manufCode:     0x1002,
			manufacturer:  "Tuya",
			model:         "TS011F",
			expectedMatch: false,
		},
		{
			name: "manufacturer pattern match",
			matcher: DeviceMatcher{
				ManufacturerPattern: "_TZ*",
			},
			manufCode:     0x1002,
			manufacturer:  "_TZ3000",
			model:         "TS011F",
			expectedMatch: true,
		},
		{
			name: "manufacturer pattern no match",
			matcher: DeviceMatcher{
				ManufacturerPattern: "_TZ*",
			},
			manufCode:     0x117C,
			manufacturer:  "IKEA",
			model:         "TRADFRI",
			expectedMatch: false,
		},
		{
			name: "model pattern match",
			matcher: DeviceMatcher{
				ModelPattern: "TS011F*",
			},
			manufCode:     0x1002,
			manufacturer:  "_TZ3000",
			model:         "TS011F_plug",
			expectedMatch: true,
		},
		{
			name: "model pattern exact match",
			matcher: DeviceMatcher{
				ModelPattern: "TS011F",
			},
			manufCode:     0x1002,
			manufacturer:  "_TZ3000",
			model:         "TS011F",
			expectedMatch: true,
		},
		{
			name: "model pattern no match",
			matcher: DeviceMatcher{
				ModelPattern: "TS011F*",
			},
			manufCode:     0x117C,
			manufacturer:  "IKEA",
			model:         "TRADFRI_bulb",
			expectedMatch: false,
		},
		{
			name: "combined manufacturer and model match",
			matcher: DeviceMatcher{
				ManufacturerPattern: "_TZ*",
				ModelPattern:        "TS011F*",
			},
			manufCode:     0x1002,
			manufacturer:  "_TZ3000",
			model:         "TS011F_plug",
			expectedMatch: true,
		},
		{
			name:          "empty matcher matches all",
			matcher:       DeviceMatcher{},
			manufCode:     0x1002,
			manufacturer:  "AnyManuf",
			model:         "AnyModel",
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.matcher.Compile()
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			result := tt.matcher.Matches(tt.manufCode, tt.manufacturer, tt.model)
			if result != tt.expectedMatch {
				t.Errorf("Matches() = %v, want %v", result, tt.expectedMatch)
			}
		})
	}
}

func TestRegistry_FindQuirks(t *testing.T) {
	registry := NewRegistry()

	// Register test quirks
	quirk1 := &DeviceQuirk{
		ID:   "test-tuya",
		Type: QuirkResponseOverride,
		Matcher: DeviceMatcher{
			ManufacturerPattern: "_TZ*",
		},
	}
	quirk2 := &DeviceQuirk{
		ID:   "test-ts011f",
		Type: QuirkAttributeOverride,
		Matcher: DeviceMatcher{
			ModelPattern: "TS011F*",
		},
	}
	quirk3 := &DeviceQuirk{
		ID:   "test-ikea",
		Type: QuirkBinding,
		Matcher: DeviceMatcher{
			ManufacturerCode: ptrUint16(0x117C),
		},
	}

	if err := registry.RegisterAll([]*DeviceQuirk{quirk1, quirk2, quirk3}); err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	tests := []struct {
		name          string
		info          DeviceInfo
		expectedCount int
		expectedIDs   []string
	}{
		{
			name: "tuya device matches tuya quirks",
			info: DeviceInfo{
				ManufacturerCode: 0x1002,
				Manufacturer:     "_TZ3000",
				Model:            "TS011F_plug",
			},
			expectedCount: 2,
			expectedIDs:   []string{"test-tuya", "test-ts011f"},
		},
		{
			name: "ikea device matches ikea quirk",
			info: DeviceInfo{
				ManufacturerCode: 0x117C,
				Manufacturer:     "IKEA",
				Model:            "TRADFRI",
			},
			expectedCount: 1,
			expectedIDs:   []string{"test-ikea"},
		},
		{
			name: "unknown device no matches",
			info: DeviceInfo{
				ManufacturerCode: 0xFFFF,
				Manufacturer:     "Unknown",
				Model:            "Unknown",
			},
			expectedCount: 0,
			expectedIDs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quirks := registry.FindQuirks(tt.info)
			if len(quirks) != tt.expectedCount {
				t.Errorf("FindQuirks() returned %d quirks, want %d", len(quirks), tt.expectedCount)
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, q := range quirks {
					if q.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected quirk %s not found", expectedID)
				}
			}
		})
	}
}

func TestAppliedQuirks_ApplyAttributeOverride(t *testing.T) {
	divisor := 1000.0
	multiplier := 2.0
	offset := 5.0

	applied := &AppliedQuirks{
		AttributeOverrides: map[zcl.ClusterID]map[zcl.AttributeID]*AttributeOverride{
			zcl.ClusterElectricalMeas: {
				zcl.AttrElecRMSCurrent: {
					ClusterID:   zcl.ClusterElectricalMeas,
					AttributeID: zcl.AttrElecRMSCurrent,
					Divisor:     &divisor,
				},
				zcl.AttrElecRMSVoltage: {
					ClusterID:   zcl.ClusterElectricalMeas,
					AttributeID: zcl.AttrElecRMSVoltage,
					Multiplier:  &multiplier,
					Offset:      &offset,
				},
			},
		},
	}

	tests := []struct {
		name      string
		clusterID zcl.ClusterID
		attrID    zcl.AttributeID
		input     float64
		expected  float64
	}{
		{
			name:      "apply divisor",
			clusterID: zcl.ClusterElectricalMeas,
			attrID:    zcl.AttrElecRMSCurrent,
			input:     1500.0,
			expected:  1.5, // 1500 / 1000
		},
		{
			name:      "apply multiplier and offset",
			clusterID: zcl.ClusterElectricalMeas,
			attrID:    zcl.AttrElecRMSVoltage,
			input:     100.0,
			expected:  205.0, // (100 * 2) + 5
		},
		{
			name:      "no override returns original",
			clusterID: zcl.ClusterOnOff,
			attrID:    zcl.AttrOnOff,
			input:     1.0,
			expected:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applied.ApplyAttributeOverride(tt.clusterID, tt.attrID, tt.input)
			if result != tt.expected {
				t.Errorf("ApplyAttributeOverride() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAppliedQuirks_IsAcceptableResponse(t *testing.T) {
	applied := &AppliedQuirks{
		ResponseOverrides: map[zcl.ClusterID]map[uint8][]uint8{
			zcl.ClusterOnOff: {
				0x07: {0x0A, 0x01},
			},
		},
	}

	tests := []struct {
		name      string
		clusterID zcl.ClusterID
		expected  uint8
		actual    uint8
		want      bool
	}{
		{
			name:      "exact match always acceptable",
			clusterID: zcl.ClusterOnOff,
			expected:  0x07,
			actual:    0x07,
			want:      true,
		},
		{
			name:      "override command acceptable",
			clusterID: zcl.ClusterOnOff,
			expected:  0x07,
			actual:    0x0A,
			want:      true,
		},
		{
			name:      "another override command acceptable",
			clusterID: zcl.ClusterOnOff,
			expected:  0x07,
			actual:    0x01,
			want:      true,
		},
		{
			name:      "unlisted command not acceptable",
			clusterID: zcl.ClusterOnOff,
			expected:  0x07,
			actual:    0xFF,
			want:      false,
		},
		{
			name:      "no override for cluster",
			clusterID: zcl.ClusterLevelControl,
			expected:  0x07,
			actual:    0x0A,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applied.IsAcceptableResponse(tt.clusterID, tt.expected, tt.actual)
			if result != tt.want {
				t.Errorf("IsAcceptableResponse() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestDefaultRegistry(t *testing.T) {
	registry := DefaultRegistry()

	if registry.Count() == 0 {
		t.Error("DefaultRegistry() should have built-in quirks")
	}

	// Check that built-in quirks are registered
	tuyaQuirk := registry.GetQuirkByID("tuya-reporting-response")
	if tuyaQuirk == nil {
		t.Error("Expected tuya-reporting-response quirk to be registered")
	}

	ikeaQuirk := registry.GetQuirkByID("ikea-binding")
	if ikeaQuirk == nil {
		t.Error("Expected ikea-binding quirk to be registered")
	}
}

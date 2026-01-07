package devices

import "testing"

func TestLookup(t *testing.T) {
	tests := []struct {
		name         string
		manufacturer string
		model        string
		wantVendor   string
		wantModel    string
		wantNil      bool
	}{
		{
			name:         "NOUS A6Z outdoor socket",
			manufacturer: "_TZ3000_266azbg3",
			model:        "TS011F",
			wantVendor:   "NOUS",
			wantModel:    "A6Z",
		},
		{
			name:         "GLEDOPTO dimmer",
			manufacturer: "GLEDOPTO",
			model:        "GL-SD-301P",
			wantVendor:   "GLEDOPTO",
			wantModel:    "GL-SD-301P",
		},
		{
			name:         "IKEA TRADFRI outlet",
			manufacturer: "IKEA of Sweden",
			model:        "TRADFRI control outlet",
			wantVendor:   "IKEA",
			wantModel:    "E1603",
		},
		{
			name:         "Aqara temperature sensor",
			manufacturer: "LUMI",
			model:        "lumi.weather",
			wantVendor:   "Aqara",
			wantModel:    "WSDCGQ11LM",
		},
		{
			name:         "Sonoff motion sensor",
			manufacturer: "SONOFF",
			model:        "SNZB-03",
			wantVendor:   "Sonoff",
			wantModel:    "SNZB-03",
		},
		{
			name:         "Philips Hue bulb",
			manufacturer: "Philips",
			model:        "LCT015",
			wantVendor:   "Philips Hue",
			wantModel:    "LCT015",
		},
		{
			name:         "unknown device",
			manufacturer: "Unknown_Manufacturer",
			model:        "Unknown_Model",
			wantNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := Lookup(tt.manufacturer, tt.model)
			if tt.wantNil {
				if info != nil {
					t.Errorf("Lookup() = %v, want nil", info)
				}
				return
			}
			if info == nil {
				t.Fatalf("Lookup() = nil, want non-nil")
			}
			if info.Vendor != tt.wantVendor {
				t.Errorf("Vendor = %q, want %q", info.Vendor, tt.wantVendor)
			}
			if info.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", info.Model, tt.wantModel)
			}
		})
	}
}

func TestLookupByManufacturer(t *testing.T) {
	// Should find any GLEDOPTO device.
	info := LookupByManufacturer("GLEDOPTO")
	if info == nil {
		t.Fatal("LookupByManufacturer(GLEDOPTO) = nil, want non-nil")
	}
	if info.Vendor != "GLEDOPTO" {
		t.Errorf("Vendor = %q, want GLEDOPTO", info.Vendor)
	}

	// Unknown manufacturer should return nil.
	info = LookupByManufacturer("Unknown_Manufacturer")
	if info != nil {
		t.Errorf("LookupByManufacturer(Unknown) = %v, want nil", info)
	}
}

func TestLookupWithFallback(t *testing.T) {
	// Exact match should work.
	info := LookupWithFallback("GLEDOPTO", "GL-SD-301P")
	if info == nil || info.Model != "GL-SD-301P" {
		t.Error("exact match failed")
	}

	// Fallback to manufacturer when model doesn't match.
	info = LookupWithFallback("GLEDOPTO", "UNKNOWN_MODEL")
	if info == nil {
		t.Error("manufacturer fallback failed")
	} else if info.Vendor != "GLEDOPTO" {
		t.Errorf("Vendor = %q, want GLEDOPTO", info.Vendor)
	}

	// Unknown manufacturer should return nil.
	info = LookupWithFallback("Unknown", "Unknown")
	if info != nil {
		t.Errorf("LookupWithFallback(Unknown, Unknown) = %v, want nil", info)
	}
}

func TestCount(t *testing.T) {
	count := Count()
	if count < 50 {
		t.Errorf("Count() = %d, want >= 50 devices", count)
	}
}

func TestAllFingerprints(t *testing.T) {
	fps := AllFingerprints()
	if len(fps) != Count() {
		t.Errorf("AllFingerprints() returned %d, Count() = %d", len(fps), Count())
	}
}

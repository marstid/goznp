package quirks

import "github.com/marstid/goznp/pkg/zcl"

// Manufacturer codes for common device manufacturers.
const (
	ManufTuya    uint16 = 0x1002 // Tuya
	ManufIKEA    uint16 = 0x117C // IKEA
	ManufPhilips uint16 = 0x100B // Philips (Signify)
	ManufAqara   uint16 = 0x115F // Aqara / Xiaomi
	ManufSonoff  uint16 = 0x1286 // Sonoff / ITEAD
)

// divisor1000 is a helper for creating divisor pointers.
func divisor1000() *float64 {
	v := 1000.0
	return &v
}

// BuiltinQuirks contains all compiled-in device quirks.
var BuiltinQuirks = []*DeviceQuirk{
	// Tuya: Accept Report Attributes as ConfigureReporting response
	{
		ID:          "tuya-reporting-response",
		Description: "Tuya devices respond with Report Attributes (0x0A) instead of ConfigureReportingResponse (0x07)",
		Type:        QuirkResponseOverride,
		Matcher: DeviceMatcher{
			ManufacturerPattern: "_TZ*", // Tuya manufacturer strings start with _TZ
		},
		ResponseOverrides: []ResponseOverride{
			{
				ClusterID:       zcl.ClusterOnOff,
				ExpectedCommand: 0x07,          // ConfigureReportingResponse
				AcceptCommands:  []uint8{0x0A}, // Report Attributes
			},
			{
				ClusterID:       zcl.ClusterLevelControl,
				ExpectedCommand: 0x07,
				AcceptCommands:  []uint8{0x0A},
			},
			{
				ClusterID:       zcl.ClusterElectricalMeas,
				ExpectedCommand: 0x07,
				AcceptCommands:  []uint8{0x0A},
			},
			{
				ClusterID:       zcl.ClusterMeteringSimple,
				ExpectedCommand: 0x07,
				AcceptCommands:  []uint8{0x0A},
			},
		},
	},

	// Tuya TS011F smart plugs: Electrical measurement scaling
	{
		ID:          "tuya-ts011f-electrical",
		Description: "Tuya TS011F plugs report current in mA, needs conversion to A",
		Type:        QuirkAttributeOverride,
		Matcher: DeviceMatcher{
			ModelPattern: "TS011F*",
		},
		AttributeOverrides: []AttributeOverride{
			{
				ClusterID:   zcl.ClusterElectricalMeas,
				AttributeID: zcl.AttrElecRMSCurrent,
				Divisor:     divisor1000(), // Convert mA to A
			},
		},
	},

	// Tuya TS011F: Energy reset methods
	{
		ID:          "tuya-ts011f-energy-reset",
		Description: "Tuya TS011F plugs support multiple energy reset methods",
		Type:        QuirkEnergyReset,
		Matcher: DeviceMatcher{
			ModelPattern: "TS011F*",
		},
		EnergyResetMethods: []EnergyResetMethod{
			// Method 1: Basic cluster reset command
			{
				UseCommand: true,
				ClusterID:  zcl.ClusterBasic,
				CommandID:  0x00, // resetFactDefault
			},
			// Method 2: Tuya cluster 0xE001, attribute 0xD004
			{
				ClusterID:   zcl.ClusterID(0xE001),
				AttributeID: zcl.AttributeID(0xD004),
				Value:       uint8(1),
			},
			// Method 3: Tuya cluster 0xE000, attribute 0xD004
			{
				ClusterID:   zcl.ClusterID(0xE000),
				AttributeID: zcl.AttributeID(0xD004),
				Value:       uint8(1),
			},
		},
	},

	// Tuya general: Accept non-standard response formats
	{
		ID:          "tuya-general-responses",
		Description: "General Tuya quirks for non-standard ZCL responses",
		Type:        QuirkResponseOverride,
		Matcher: DeviceMatcher{
			ManufacturerPattern: "_TZ*",
		},
		ResponseOverrides: []ResponseOverride{
			// Many Tuya devices use 0x0A for various responses
			{
				ClusterID:       zcl.ClusterBasic,
				ExpectedCommand: 0x07,
				AcceptCommands:  []uint8{0x0A},
			},
		},
	},

	// IKEA: Special binding requirements
	{
		ID:          "ikea-binding",
		Description: "IKEA devices may require specific binding order",
		Type:        QuirkBinding,
		Matcher: DeviceMatcher{
			ManufacturerCode: ptrUint16(ManufIKEA),
		},
	},

	// IKEA TRADFRI: Timing adjustments for slow devices
	{
		ID:          "ikea-tradfri-timing",
		Description: "IKEA TRADFRI devices need longer timeouts",
		Type:        QuirkTimingAdjust,
		Matcher: DeviceMatcher{
			ModelPattern: "TRADFRI*",
		},
		TimingOverride: &TimingOverride{
			CommandTimeout: 10000, // 10 seconds
			RetryCount:     3,
			RetryDelay:     1000, // 1 second
		},
	},

	// Aqara: Attribute reporting quirks
	{
		ID:          "aqara-reporting",
		Description: "Aqara devices use non-standard attribute reporting",
		Type:        QuirkResponseOverride,
		Matcher: DeviceMatcher{
			ManufacturerCode: ptrUint16(ManufAqara),
		},
		ResponseOverrides: []ResponseOverride{
			{
				ClusterID:       zcl.ClusterOnOff,
				ExpectedCommand: 0x07,
				AcceptCommands:  []uint8{0x0A, 0x01},
			},
		},
	},

	// Aqara sensors: Longer interview timeout for sleepy devices
	{
		ID:          "aqara-sensor-timing",
		Description: "Aqara sensors are sleepy and need longer timeouts",
		Type:        QuirkTimingAdjust,
		Matcher: DeviceMatcher{
			ManufacturerPattern: "LUMI*",
		},
		TimingOverride: &TimingOverride{
			InterviewTimeout: 30, // 30 seconds
			CommandTimeout:   15000,
			RetryCount:       5,
			RetryDelay:       2000,
		},
	},

	// Philips Hue: Enhanced color control
	{
		ID:          "philips-hue-color",
		Description: "Philips Hue bulbs support enhanced color control commands",
		Type:        QuirkCommandOverride,
		Matcher: DeviceMatcher{
			ManufacturerCode: ptrUint16(ManufPhilips),
		},
	},

	// Sonoff: Standard Tuya-like behavior
	{
		ID:          "sonoff-responses",
		Description: "Sonoff devices have Tuya-like response behavior",
		Type:        QuirkResponseOverride,
		Matcher: DeviceMatcher{
			ManufacturerCode: ptrUint16(ManufSonoff),
		},
		ResponseOverrides: []ResponseOverride{
			{
				ClusterID:       zcl.ClusterOnOff,
				ExpectedCommand: 0x07,
				AcceptCommands:  []uint8{0x0A},
			},
		},
	},

	// Generic Tuya temperature sensors
	{
		ID:          "tuya-temp-sensor",
		Description: "Tuya temperature sensors report in 0.01C units",
		Type:        QuirkAttributeOverride,
		Matcher: DeviceMatcher{
			ModelPattern: "TS0201*",
		},
		AttributeOverrides: []AttributeOverride{
			{
				ClusterID:   zcl.ClusterTempMeasurement,
				AttributeID: zcl.AttrTempMeasuredValue,
				Divisor:     ptr(100.0), // Convert from 0.01C to C
			},
		},
	},

	// Tuya humidity sensors
	{
		ID:          "tuya-humidity-sensor",
		Description: "Tuya humidity sensors report in 0.01% units",
		Type:        QuirkAttributeOverride,
		Matcher: DeviceMatcher{
			ModelPattern: "TS0201*",
		},
		AttributeOverrides: []AttributeOverride{
			{
				ClusterID:   zcl.ClusterHumidityMeas,
				AttributeID: zcl.AttrHumidityMeasuredValue,
				Divisor:     ptr(100.0), // Convert from 0.01% to %
			},
		},
	},
}

// ptrUint16 returns a pointer to the given uint16 value.
func ptrUint16(v uint16) *uint16 {
	return &v
}

// ptr returns a pointer to the given float64 value.
func ptr(v float64) *float64 {
	return &v
}

// DefaultRegistry returns a registry pre-populated with built-in quirks.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	_ = r.RegisterAll(BuiltinQuirks) // Ignore errors for built-in quirks
	return r
}

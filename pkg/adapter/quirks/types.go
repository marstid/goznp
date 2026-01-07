package quirks

import (
	"regexp"

	"github.com/marstid/goznp/pkg/zcl"
)

// QuirkType identifies the category of a quirk.
type QuirkType uint8

const (
	// QuirkAttributeOverride modifies attribute behavior (scaling, type conversion).
	QuirkAttributeOverride QuirkType = iota
	// QuirkCommandOverride uses a different command or accepts non-standard responses.
	QuirkCommandOverride
	// QuirkEndpointRemap remaps endpoint numbers.
	QuirkEndpointRemap
	// QuirkClusterRemap remaps cluster IDs.
	QuirkClusterRemap
	// QuirkTimingAdjust adjusts timeouts or delays.
	QuirkTimingAdjust
	// QuirkResponseOverride accepts non-standard command responses.
	QuirkResponseOverride
	// QuirkEnergyReset specifies method for energy counter reset.
	QuirkEnergyReset
	// QuirkBinding specifies special binding requirements.
	QuirkBinding
)

// String returns the string representation of a QuirkType.
func (q QuirkType) String() string {
	switch q {
	case QuirkAttributeOverride:
		return "AttributeOverride"
	case QuirkCommandOverride:
		return "CommandOverride"
	case QuirkEndpointRemap:
		return "EndpointRemap"
	case QuirkClusterRemap:
		return "ClusterRemap"
	case QuirkTimingAdjust:
		return "TimingAdjust"
	case QuirkResponseOverride:
		return "ResponseOverride"
	case QuirkEnergyReset:
		return "EnergyReset"
	case QuirkBinding:
		return "Binding"
	default:
		return "Unknown"
	}
}

// DeviceMatcher defines criteria for matching devices.
type DeviceMatcher struct {
	// ManufacturerCode matches exact manufacturer code (nil = any).
	ManufacturerCode *uint16

	// ManufacturerPattern matches manufacturer string using glob pattern.
	ManufacturerPattern string

	// ModelPattern matches model string using glob pattern.
	ModelPattern string

	// compiledManuf is the compiled regex for manufacturer pattern.
	compiledManuf *regexp.Regexp

	// compiledModel is the compiled regex for model pattern.
	compiledModel *regexp.Regexp
}

// Compile prepares the matcher patterns for use.
func (m *DeviceMatcher) Compile() error {
	if m.ManufacturerPattern != "" {
		re, err := globToRegex(m.ManufacturerPattern)
		if err != nil {
			return err
		}
		m.compiledManuf = re
	}
	if m.ModelPattern != "" {
		re, err := globToRegex(m.ModelPattern)
		if err != nil {
			return err
		}
		m.compiledModel = re
	}
	return nil
}

// Matches checks if the device matches the criteria.
func (m *DeviceMatcher) Matches(manufacturerCode uint16, manufacturer, model string) bool {
	// Check manufacturer code if specified
	if m.ManufacturerCode != nil && *m.ManufacturerCode != manufacturerCode {
		return false
	}

	// Check manufacturer pattern if specified
	if m.compiledManuf != nil && !m.compiledManuf.MatchString(manufacturer) {
		return false
	}

	// Check model pattern if specified
	if m.compiledModel != nil && !m.compiledModel.MatchString(model) {
		return false
	}

	return true
}

// globToRegex converts a glob pattern to a regex.
func globToRegex(pattern string) (*regexp.Regexp, error) {
	// Convert glob to regex
	regex := "^"
	for _, c := range pattern {
		switch c {
		case '*':
			regex += ".*"
		case '?':
			regex += "."
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			regex += "\\" + string(c)
		default:
			regex += string(c)
		}
	}
	regex += "$"
	return regexp.Compile(regex)
}

// AttributeOverride defines how to override an attribute's behavior.
type AttributeOverride struct {
	ClusterID   zcl.ClusterID
	AttributeID zcl.AttributeID
	Divisor     *float64 // Divide value by this
	Multiplier  *float64 // Multiply value by this
	Offset      *float64 // Add this to value
	Invert      bool     // Invert boolean values
}

// ResponseOverride defines alternate acceptable command responses.
type ResponseOverride struct {
	ClusterID       zcl.ClusterID
	ExpectedCommand uint8   // Original expected command
	AcceptCommands  []uint8 // Also accept these commands as valid responses
}

// EnergyResetMethod defines how to reset energy counter.
type EnergyResetMethod struct {
	ClusterID   zcl.ClusterID
	AttributeID zcl.AttributeID
	Value       interface{}
	UseCommand  bool  // If true, use SendClusterCommand instead of WriteAttributes
	CommandID   uint8 // Command ID if UseCommand is true
}

// TimingOverride defines custom timing parameters.
type TimingOverride struct {
	InterviewTimeout int // Interview timeout in seconds
	CommandTimeout   int // Command timeout in milliseconds
	RetryCount       int // Number of retries
	RetryDelay       int // Delay between retries in milliseconds
}

// DeviceQuirk represents a specific quirk for a device type.
type DeviceQuirk struct {
	// ID is a unique identifier for this quirk.
	ID string

	// Description explains what this quirk does.
	Description string

	// Type categorizes the quirk.
	Type QuirkType

	// Matcher defines which devices this quirk applies to.
	Matcher DeviceMatcher

	// AttributeOverrides for QuirkAttributeOverride type.
	AttributeOverrides []AttributeOverride

	// ResponseOverrides for QuirkResponseOverride type.
	ResponseOverrides []ResponseOverride

	// EnergyResetMethods for QuirkEnergyReset type.
	EnergyResetMethods []EnergyResetMethod

	// TimingOverride for QuirkTimingAdjust type.
	TimingOverride *TimingOverride
}

// DeviceInfo contains device identification data for quirk matching.
type DeviceInfo struct {
	ManufacturerCode uint16
	Manufacturer     string
	Model            string
}

// AppliedQuirks tracks which quirks apply to a device.
type AppliedQuirks struct {
	DeviceAddr         uint16
	IEEEAddr           [8]byte
	Quirks             []*DeviceQuirk
	AttributeOverrides map[zcl.ClusterID]map[zcl.AttributeID]*AttributeOverride
	ResponseOverrides  map[zcl.ClusterID]map[uint8][]uint8
	EnergyResetMethods []EnergyResetMethod
	TimingOverride     *TimingOverride
}

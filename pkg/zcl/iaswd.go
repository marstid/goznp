package zcl

import "fmt"

// IAS Warning Device Cluster Attributes (0x0502)
const (
	AttrIASWDMaxDuration AttributeID = 0x0000 // Maximum warning duration in seconds (uint16)
)

// WarningMode represents the type of warning for IASWD devices.
type WarningMode uint8

const (
	WarningModeStop           WarningMode = 0x00 // Stop warning
	WarningModeBurglar        WarningMode = 0x01 // Burglar alarm
	WarningModeFire           WarningMode = 0x02 // Fire alarm
	WarningModeEmergency      WarningMode = 0x03 // Emergency alarm
	WarningModePolicePanic    WarningMode = 0x04 // Police panic
	WarningModeFirePanic      WarningMode = 0x05 // Fire panic
	WarningModeEmergencyPanic WarningMode = 0x06 // Emergency panic
)

// String returns human-readable warning mode name.
func (w WarningMode) String() string {
	switch w {
	case WarningModeStop:
		return "Stop"
	case WarningModeBurglar:
		return "Burglar"
	case WarningModeFire:
		return "Fire"
	case WarningModeEmergency:
		return "Emergency"
	case WarningModePolicePanic:
		return "PolicePanic"
	case WarningModeFirePanic:
		return "FirePanic"
	case WarningModeEmergencyPanic:
		return "EmergencyPanic"
	default:
		return fmt.Sprintf("WarningMode(0x%02X)", uint8(w))
	}
}

// SirenLevel represents the loudness level of the siren.
type SirenLevel uint8

const (
	SirenLevelLow      SirenLevel = 0x00 // Low volume
	SirenLevelMedium   SirenLevel = 0x01 // Medium volume
	SirenLevelHigh     SirenLevel = 0x02 // High volume
	SirenLevelVeryHigh SirenLevel = 0x03 // Very high volume
)

// String returns human-readable siren level name.
func (s SirenLevel) String() string {
	switch s {
	case SirenLevelLow:
		return "Low"
	case SirenLevelMedium:
		return "Medium"
	case SirenLevelHigh:
		return "High"
	case SirenLevelVeryHigh:
		return "VeryHigh"
	default:
		return fmt.Sprintf("SirenLevel(0x%02X)", uint8(s))
	}
}

// StrobeLevel represents the brightness level of the strobe light.
type StrobeLevel uint8

const (
	StrobeLevelLow      StrobeLevel = 0x00 // Low brightness
	StrobeLevelMedium   StrobeLevel = 0x01 // Medium brightness
	StrobeLevelHigh     StrobeLevel = 0x02 // High brightness
	StrobeLevelVeryHigh StrobeLevel = 0x03 // Very high brightness
)

// String returns human-readable strobe level name.
func (s StrobeLevel) String() string {
	switch s {
	case StrobeLevelLow:
		return "Low"
	case StrobeLevelMedium:
		return "Medium"
	case StrobeLevelHigh:
		return "High"
	case StrobeLevelVeryHigh:
		return "VeryHigh"
	default:
		return fmt.Sprintf("StrobeLevel(0x%02X)", uint8(s))
	}
}

// IAS Warning Device Cluster Commands (0x0502)
const (
	CmdIASWDStartWarning uint8 = 0x00 // Start warning (siren/strobe)
	CmdIASWDSquawk       uint8 = 0x01 // Squawk (brief sound for arming/disarming)
)

package zcl

import "testing"

func TestClusterIDString(t *testing.T) {
	tests := []struct {
		c    ClusterID
		want string
	}{
		{ClusterBasic, "Basic"},
		{ClusterPowerConfig, "PowerConfiguration"},
		{ClusterDeviceTemp, "DeviceTemperature"},
		{ClusterIdentify, "Identify"},
		{ClusterGroups, "Groups"},
		{ClusterScenes, "Scenes"},
		{ClusterOnOff, "OnOff"},
		{ClusterOnOffConfig, "OnOffConfiguration"},
		{ClusterLevelControl, "LevelControl"},
		{ClusterAlarms, "Alarms"},
		{ClusterDoorLock, "DoorLock"},
		{ClusterWindowCovering, "WindowCovering"},
		{ClusterColorControl, "ColorControl"},
		{ClusterTime, "Time"},
		{ClusterAnalogInput, "AnalogInput"},
		{ClusterAnalogOutput, "AnalogOutput"},
		{ClusterBinaryInput, "BinaryInput"},
		{ClusterBinaryOutput, "BinaryOutput"},
		{ClusterBinaryValue, "BinaryValue"},
		{ClusterMultistateInput, "MultistateInput"},
		{ClusterOTAUpgrade, "OTAUpgrade"},
		{ClusterPollControl, "PollControl"},
		{ClusterElectricalMeas, "ElectricalMeasurement"},
		{ClusterMeteringSimple, "SimpleMetering"},
		{ClusterTempMeasurement, "TemperatureMeasurement"},
		{ClusterPressureMeas, "PressureMeasurement"},
		{ClusterHumidityMeas, "RelativeHumidityMeasurement"},
		{ClusterIlluminanceMeas, "IlluminanceMeasurement"},
		{ClusterCarbonMonoxide, "CarbonMonoxideMeasurement"},
		{ClusterCarbonDioxide, "CarbonDioxideMeasurement"},
		{ClusterPM25Measurement, "PM25Measurement"},
		{ClusterFormaldehydeMeasurement, "FormaldehydeMeasurement"},
		{ClusterThermostat, "Thermostat"},
		{ClusterFan, "Fan"},
		{ClusterOccupancySensing, "OccupancySensing"},
		{ClusterIASZone, "IASZone"},
		{ClusterIASWD, "IASWD"},
		{ClusterID(0xFFFF), "Cluster(0xFFFF)"},
		{ClusterID(0x1234), "Cluster(0x1234)"},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("ClusterID(%#x).String() = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestPowerSourceString(t *testing.T) {
	tests := []struct {
		p    PowerSource
		want string
	}{
		{PowerSourceUnknown, "Unknown"},
		{PowerSourceMains, "MainsSinglePhase"},
		{PowerSourceBattery, "Battery"},
		{PowerSourceDC, "DCSource"},
		{PowerSourceEmergencyMains, "EmergencyMainsAlwaysOn"},
		{PowerSourceEmergencyDC, "EmergencyMainsTransferSwitch"},
		{PowerSource(0xFF), "PowerSource(0xFF)"},
		{PowerSource(0x99), "PowerSource(0x99)"},
	}
	for _, tt := range tests {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("PowerSource(%#x).String() = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestThermostatSystemModeString(t *testing.T) {
	tests := []struct {
		m    ThermostatSystemMode
		want string
	}{
		{ThermostatModeOff, "Off"},
		{ThermostatModeAuto, "Auto"},
		{ThermostatModeCool, "Cool"},
		{ThermostatModeHeat, "Heat"},
		{ThermostatModeEmergencyHeat, "EmergencyHeat"},
		{ThermostatModePrecooling, "Precooling"},
		{ThermostatModeFanOnly, "FanOnly"},
		{ThermostatModeDry, "Dry"},
		{ThermostatModeSleep, "Sleep"},
		{ThermostatSystemMode(0xFF), "ThermostatSystemMode(0xFF)"},
		{ThermostatSystemMode(0x0A), "ThermostatSystemMode(0x0A)"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("ThermostatSystemMode(%#x).String() = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestFanModeString(t *testing.T) {
	tests := []struct {
		f    FanMode
		want string
	}{
		{FanModeOff, "Off"},
		{FanModeLow, "Low"},
		{FanModeMedium, "Medium"},
		{FanModeHigh, "High"},
		{FanModeOn, "On"},
		{FanModeAuto, "Auto"},
		{FanModeSmart, "Smart"},
		{FanMode(0xFF), "FanMode(0xFF)"},
		{FanMode(0x10), "FanMode(0x10)"},
	}
	for _, tt := range tests {
		if got := tt.f.String(); got != tt.want {
			t.Errorf("FanMode(%#x).String() = %q, want %q", tt.f, got, tt.want)
		}
	}
}

func TestFanModeSequenceString(t *testing.T) {
	tests := []struct {
		f    FanModeSequence
		want string
	}{
		{FanSequenceLowMedHigh, "LowMedHigh"},
		{FanSequenceLowHigh, "LowHigh"},
		{FanSequenceLowMedHighAuto, "LowMedHighAuto"},
		{FanSequenceLowHighAuto, "LowHighAuto"},
		{FanSequenceOnAuto, "OnAuto"},
		{FanModeSequence(0xFF), "FanModeSequence(0xFF)"},
		{FanModeSequence(0x10), "FanModeSequence(0x10)"},
	}
	for _, tt := range tests {
		if got := tt.f.String(); got != tt.want {
			t.Errorf("FanModeSequence(%#x).String() = %q, want %q", tt.f, got, tt.want)
		}
	}
}

func TestDoorLockStateString(t *testing.T) {
	tests := []struct {
		d    DoorLockState
		want string
	}{
		{DoorLockStateNotFullyLocked, "NotFullyLocked"},
		{DoorLockStateLocked, "Locked"},
		{DoorLockStateUnlocked, "Unlocked"},
		{DoorLockStateUndefined, "Undefined"},
		{DoorLockState(0x10), "DoorLockState(0x10)"},
		{DoorLockState(0xFE), "DoorLockState(0xFE)"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("DoorLockState(%#x).String() = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestDoorStateString(t *testing.T) {
	tests := []struct {
		d    DoorState
		want string
	}{
		{DoorStateOpen, "Open"},
		{DoorStateClosed, "Closed"},
		{DoorStateJammed, "Jammed"},
		{DoorStateForcedOpen, "ForcedOpen"},
		{DoorStateUnspecified, "Unspecified"},
		{DoorState(0xFF), "DoorState(0xFF)"},
		{DoorState(0x10), "DoorState(0x10)"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("DoorState(%#x).String() = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestWindowCoveringTypeString(t *testing.T) {
	tests := []struct {
		w    WindowCoveringType
		want string
	}{
		{WindowCoveringTypeRollerShade, "RollerShade"},
		{WindowCoveringTypeRollerShade2Motor, "RollerShade2Motor"},
		{WindowCoveringTypeRollerShadeExterior, "RollerShadeExterior"},
		{WindowCoveringTypeRollerShadeExterior2Motor, "RollerShadeExterior2Motor"},
		{WindowCoveringTypeDrapery, "Drapery"},
		{WindowCoveringTypeAwning, "Awning"},
		{WindowCoveringTypeShutter, "Shutter"},
		{WindowCoveringTypeTiltBlindTiltOnly, "TiltBlindTiltOnly"},
		{WindowCoveringTypeTiltBlindLiftAndTilt, "TiltBlindLiftAndTilt"},
		{WindowCoveringTypeProjectorScreen, "ProjectorScreen"},
		{WindowCoveringTypeUnknown, "Unknown"},
		{WindowCoveringType(0x10), "WindowCoveringType(0x10)"},
		{WindowCoveringType(0xFE), "WindowCoveringType(0xFE)"},
	}
	for _, tt := range tests {
		if got := tt.w.String(); got != tt.want {
			t.Errorf("WindowCoveringType(%#x).String() = %q, want %q", tt.w, got, tt.want)
		}
	}
}

func TestMoveModeString(t *testing.T) {
	tests := []struct {
		m    MoveMode
		want string
	}{
		{MoveModeUp, "Up"},
		{MoveModeDown, "Down"},
		{MoveMode(0xFF), "MoveMode(0xFF)"},
		{MoveMode(0x10), "MoveMode(0x10)"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("MoveMode(%#x).String() = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestStepModeString(t *testing.T) {
	tests := []struct {
		s    StepMode
		want string
	}{
		{StepModeUp, "Up"},
		{StepModeDown, "Down"},
		{StepMode(0xFF), "StepMode(0xFF)"},
		{StepMode(0x10), "StepMode(0x10)"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("StepMode(%#x).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestStartUpOnOffString(t *testing.T) {
	tests := []struct {
		s    StartUpOnOff
		want string
	}{
		{StartUpOnOffOff, "Off"},
		{StartUpOnOffOn, "On"},
		{StartUpOnOffToggle, "Toggle"},
		{StartUpOnOffPrevious, "Previous"},
		{StartUpOnOff(0x10), "StartUpOnOff(0x10)"},
		{StartUpOnOff(0xFE), "StartUpOnOff(0xFE)"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("StartUpOnOff(%#x).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestBatterySizeString(t *testing.T) {
	tests := []struct {
		b    BatterySize
		want string
	}{
		{BatterySizeNoBattery, "NoBattery"},
		{BatterySizeBuiltIn, "BuiltIn"},
		{BatterySizeOther, "Other"},
		{BatterySizeAA, "AA"},
		{BatterySizeAAA, "AAA"},
		{BatterySizeC, "C"},
		{BatterySizeD, "D"},
		{BatterySizeCR2, "CR2"},
		{BatterySizeCR123A, "CR123A"},
		{BatterySizeUnknown, "Unknown"},
		{BatterySize(0x10), "BatterySize(0x10)"},
		{BatterySize(0xFE), "BatterySize(0xFE)"},
	}
	for _, tt := range tests {
		if got := tt.b.String(); got != tt.want {
			t.Errorf("BatterySize(%#x).String() = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestIASZoneTypeString(t *testing.T) {
	tests := []struct {
		z    IASZoneType
		want string
	}{
		{IASZoneTypeStandardCIE, "StandardCIE"},
		{IASZoneTypeMotionSensor, "MotionSensor"},
		{IASZoneTypeContactSwitch, "ContactSwitch"},
		{IASZoneTypeDoorWindow, "DoorWindow"},
		{IASZoneTypeFireSensor, "FireSensor"},
		{IASZoneTypeWaterSensor, "WaterSensor"},
		{IASZoneTypeCOSensor, "COSensor"},
		{IASZoneTypePersonalEmergency, "PersonalEmergency"},
		{IASZoneTypeVibrationMovement, "VibrationMovement"},
		{IASZoneTypeRemoteControl, "RemoteControl"},
		{IASZoneTypeKeyFob, "KeyFob"},
		{IASZoneTypeKeypad, "Keypad"},
		{IASZoneTypeStandardWarning, "StandardWarning"},
		{IASZoneTypeGlassBreak, "GlassBreak"},
		{IASZoneTypeSecurityRepeater, "SecurityRepeater"},
		{IASZoneTypeInvalidZone, "InvalidZone"},
		{IASZoneType(0x1234), "IASZoneType(0x1234)"},
		{IASZoneType(0x0001), "IASZoneType(0x0001)"},
	}
	for _, tt := range tests {
		if got := tt.z.String(); got != tt.want {
			t.Errorf("IASZoneType(%#x).String() = %q, want %q", tt.z, got, tt.want)
		}
	}
}

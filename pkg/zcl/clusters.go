package zcl

import "fmt"

// ClusterID defines standard Zigbee cluster IDs.
type ClusterID uint16

const (
	ClusterBasic                   ClusterID = 0x0000
	ClusterPowerConfig             ClusterID = 0x0001
	ClusterDeviceTemp              ClusterID = 0x0002
	ClusterIdentify                ClusterID = 0x0003
	ClusterGroups                  ClusterID = 0x0004
	ClusterScenes                  ClusterID = 0x0005
	ClusterOnOff                   ClusterID = 0x0006
	ClusterOnOffConfig             ClusterID = 0x0007
	ClusterLevelControl            ClusterID = 0x0008
	ClusterAlarms                  ClusterID = 0x0009
	ClusterDoorLock                ClusterID = 0x0101
	ClusterWindowCovering          ClusterID = 0x0102
	ClusterColorControl            ClusterID = 0x0300
	ClusterTime                    ClusterID = 0x000A
	ClusterAnalogInput             ClusterID = 0x000C
	ClusterAnalogOutput            ClusterID = 0x000D
	ClusterAnalogValue             ClusterID = 0x000E
	ClusterBinaryInput             ClusterID = 0x000F
	ClusterBinaryOutput            ClusterID = 0x0010
	ClusterBinaryValue             ClusterID = 0x0011
	ClusterMultistateInput         ClusterID = 0x0012
	ClusterMultistateOutput        ClusterID = 0x0013
	ClusterMultistateValue         ClusterID = 0x0014
	ClusterOTAUpgrade              ClusterID = 0x0019
	ClusterPollControl             ClusterID = 0x0020
	ClusterElectricalMeas          ClusterID = 0x0B04
	ClusterPrice                   ClusterID = 0x0700
	ClusterDRLC                    ClusterID = 0x0701
	ClusterMeteringSimple          ClusterID = 0x0702
	ClusterSEMessaging             ClusterID = 0x0703
	ClusterTempMeasurement         ClusterID = 0x0402
	ClusterPressureMeas            ClusterID = 0x0403
	ClusterFlowMeasurement         ClusterID = 0x0404
	ClusterHumidityMeas            ClusterID = 0x0405
	ClusterIlluminanceMeas         ClusterID = 0x0400
	ClusterCarbonMonoxide          ClusterID = 0x040C
	ClusterCarbonDioxide           ClusterID = 0x040D
	ClusterPM25Measurement         ClusterID = 0x042A
	ClusterFormaldehydeMeasurement ClusterID = 0x042B
	ClusterThermostat              ClusterID = 0x0201
	ClusterFan                     ClusterID = 0x0202
	ClusterOccupancySensing        ClusterID = 0x0406
	ClusterIASZone                 ClusterID = 0x0500
	ClusterIASWD                   ClusterID = 0x0502
)

// String returns human-readable cluster name.
func (c ClusterID) String() string {
	switch c {
	case ClusterBasic:
		return "Basic"
	case ClusterPowerConfig:
		return "PowerConfiguration"
	case ClusterDeviceTemp:
		return "DeviceTemperature"
	case ClusterIdentify:
		return "Identify"
	case ClusterGroups:
		return "Groups"
	case ClusterScenes:
		return "Scenes"
	case ClusterOnOff:
		return "OnOff"
	case ClusterOnOffConfig:
		return "OnOffConfiguration"
	case ClusterLevelControl:
		return "LevelControl"
	case ClusterColorControl:
		return "ColorControl"
	case ClusterAlarms:
		return "Alarms"
	case ClusterDoorLock:
		return "DoorLock"
	case ClusterWindowCovering:
		return "WindowCovering"
	case ClusterTime:
		return "Time"
	case ClusterAnalogInput:
		return "AnalogInput"
	case ClusterAnalogOutput:
		return "AnalogOutput"
	case ClusterAnalogValue:
		return "AnalogValue"
	case ClusterBinaryInput:
		return "BinaryInput"
	case ClusterBinaryOutput:
		return "BinaryOutput"
	case ClusterBinaryValue:
		return "BinaryValue"
	case ClusterMultistateInput:
		return "MultistateInput"
	case ClusterMultistateOutput:
		return "MultistateOutput"
	case ClusterMultistateValue:
		return "MultistateValue"
	case ClusterOTAUpgrade:
		return "OTAUpgrade"
	case ClusterPollControl:
		return "PollControl"
	case ClusterElectricalMeas:
		return "ElectricalMeasurement"
	case ClusterPrice:
		return "Price"
	case ClusterDRLC:
		return "DemandResponseLoadControl"
	case ClusterMeteringSimple:
		return "SimpleMetering"
	case ClusterSEMessaging:
		return "SEMessaging"
	case ClusterTempMeasurement:
		return "TemperatureMeasurement"
	case ClusterPressureMeas:
		return "PressureMeasurement"
	case ClusterFlowMeasurement:
		return "FlowMeasurement"
	case ClusterHumidityMeas:
		return "RelativeHumidityMeasurement"
	case ClusterIlluminanceMeas:
		return "IlluminanceMeasurement"
	case ClusterCarbonMonoxide:
		return "CarbonMonoxideMeasurement"
	case ClusterCarbonDioxide:
		return "CarbonDioxideMeasurement"
	case ClusterPM25Measurement:
		return "PM25Measurement"
	case ClusterFormaldehydeMeasurement:
		return "FormaldehydeMeasurement"
	case ClusterThermostat:
		return "Thermostat"
	case ClusterFan:
		return "Fan"
	case ClusterOccupancySensing:
		return "OccupancySensing"
	case ClusterIASZone:
		return "IASZone"
	case ClusterIASWD:
		return "IASWD"
	default:
		return fmt.Sprintf("Cluster(0x%04X)", uint16(c))
	}
}

// AttributeID type for attribute IDs.
type AttributeID uint16

// Basic Cluster Attributes (0x0000).
const (
	AttrBasicZCLVersion                 AttributeID = 0x0000
	AttrBasicAppVersion                 AttributeID = 0x0001
	AttrBasicStackVersion               AttributeID = 0x0002
	AttrBasicHWVersion                  AttributeID = 0x0003
	AttrBasicManufacturerName           AttributeID = 0x0004
	AttrBasicModelIdentifier            AttributeID = 0x0005
	AttrBasicDateCode                   AttributeID = 0x0006
	AttrBasicPowerSource                AttributeID = 0x0007
	AttrBasicProductCode                AttributeID = 0x000A
	AttrBasicProductURL                 AttributeID = 0x000B
	AttrBasicManufacturerVersionDetails AttributeID = 0x000C
	AttrBasicSerialNumber               AttributeID = 0x000D
	AttrBasicProductLabel               AttributeID = 0x000E
	AttrBasicLocationDescription        AttributeID = 0x0010
	AttrBasicPhysicalEnvironment        AttributeID = 0x0011
	AttrBasicDeviceEnabled              AttributeID = 0x0012
	AttrBasicAlarmMask                  AttributeID = 0x0013
	AttrBasicDisableLocalConfig         AttributeID = 0x0014
	AttrBasicSWBuildID                  AttributeID = 0x4000
	AttrBasicGenericDeviceClass         AttributeID = 0xFFFD
	AttrBasicGenericDeviceType          AttributeID = 0xFFFE
)

// Basic Cluster Commands (0x0000).
const (
	CmdBasicResetToFactoryDefaults uint8 = 0x00 // Reset all attribute values to factory defaults.
)

// Power Configuration Attributes (0x0001).
const (
	// Mains Information.
	AttrPowerMainsVoltage             AttributeID = 0x0000 // uint16 in 0.1V.
	AttrPowerMainsFrequency           AttributeID = 0x0001 // uint8 in 2Hz increments.
	AttrPowerMainsAlarmMask           AttributeID = 0x0010 // bitmap8.
	AttrPowerMainsVoltageMinThreshold AttributeID = 0x0011 // uint16.
	AttrPowerMainsVoltageMaxThreshold AttributeID = 0x0012 // uint16.
	AttrPowerMainsVoltageDwellTripPnt AttributeID = 0x0013 // uint16.

	// Battery Information Set 1.
	AttrPowerBatteryVoltage                AttributeID = 0x0020 // uint8 in 0.1V.
	AttrPowerBatteryPercentage             AttributeID = 0x0021 // uint8 in 0.5% (0-200).
	AttrPowerBatteryManufacturer           AttributeID = 0x0030 // string.
	AttrPowerBatterySize                   AttributeID = 0x0031 // enum8.
	AttrPowerBatteryAHrRating              AttributeID = 0x0032 // uint16.
	AttrPowerBatteryQuantity               AttributeID = 0x0033 // uint8.
	AttrPowerBatteryRatedVoltage           AttributeID = 0x0034 // uint8.
	AttrPowerBatteryAlarmMask              AttributeID = 0x0035 // bitmap8.
	AttrPowerBatteryVoltageMinThreshold    AttributeID = 0x0036 // uint8.
	AttrPowerBatteryVoltageThreshold1      AttributeID = 0x0037 // uint8.
	AttrPowerBatteryVoltageThreshold2      AttributeID = 0x0038 // uint8.
	AttrPowerBatteryVoltageThreshold3      AttributeID = 0x0039 // uint8.
	AttrPowerBatteryPercentageMinThreshold AttributeID = 0x003A // uint8.
	AttrPowerBatteryPercentageThreshold1   AttributeID = 0x003B // uint8.
	AttrPowerBatteryPercentageThreshold2   AttributeID = 0x003C // uint8.
	AttrPowerBatteryPercentageThreshold3   AttributeID = 0x003D // uint8.
	AttrPowerBatteryAlarmState             AttributeID = 0x003E // bitmap32.

	// Battery Information Set 2.
	AttrPowerBattery2Voltage                AttributeID = 0x0040 // uint8 in 0.1V.
	AttrPowerBattery2Percentage             AttributeID = 0x0041 // uint8 in 0.5% (0-200).
	AttrPowerBattery2Manufacturer           AttributeID = 0x0050 // string.
	AttrPowerBattery2Size                   AttributeID = 0x0051 // enum8.
	AttrPowerBattery2AHrRating              AttributeID = 0x0052 // uint16.
	AttrPowerBattery2Quantity               AttributeID = 0x0053 // uint8.
	AttrPowerBattery2RatedVoltage           AttributeID = 0x0054 // uint8.
	AttrPowerBattery2AlarmMask              AttributeID = 0x0055 // bitmap8.
	AttrPowerBattery2VoltageMinThreshold    AttributeID = 0x0056 // uint8.
	AttrPowerBattery2VoltageThreshold1      AttributeID = 0x0057 // uint8.
	AttrPowerBattery2VoltageThreshold2      AttributeID = 0x0058 // uint8.
	AttrPowerBattery2VoltageThreshold3      AttributeID = 0x0059 // uint8.
	AttrPowerBattery2PercentageMinThreshold AttributeID = 0x005A // uint8.
	AttrPowerBattery2PercentageThreshold1   AttributeID = 0x005B // uint8.
	AttrPowerBattery2PercentageThreshold2   AttributeID = 0x005C // uint8.
	AttrPowerBattery2PercentageThreshold3   AttributeID = 0x005D // uint8.
	AttrPowerBattery2AlarmState             AttributeID = 0x005E // bitmap32.

	// Battery Information Set 3.
	AttrPowerBattery3Voltage                AttributeID = 0x0060 // uint8 in 0.1V.
	AttrPowerBattery3Percentage             AttributeID = 0x0061 // uint8 in 0.5% (0-200).
	AttrPowerBattery3Manufacturer           AttributeID = 0x0070 // string.
	AttrPowerBattery3Size                   AttributeID = 0x0071 // enum8.
	AttrPowerBattery3AHrRating              AttributeID = 0x0072 // uint16.
	AttrPowerBattery3Quantity               AttributeID = 0x0073 // uint8.
	AttrPowerBattery3RatedVoltage           AttributeID = 0x0074 // uint8.
	AttrPowerBattery3AlarmMask              AttributeID = 0x0075 // bitmap8.
	AttrPowerBattery3VoltageMinThreshold    AttributeID = 0x0076 // uint8.
	AttrPowerBattery3VoltageThreshold1      AttributeID = 0x0077 // uint8.
	AttrPowerBattery3VoltageThreshold2      AttributeID = 0x0078 // uint8.
	AttrPowerBattery3VoltageThreshold3      AttributeID = 0x0079 // uint8.
	AttrPowerBattery3PercentageMinThreshold AttributeID = 0x007A // uint8.
	AttrPowerBattery3PercentageThreshold1   AttributeID = 0x007B // uint8.
	AttrPowerBattery3PercentageThreshold2   AttributeID = 0x007C // uint8.
	AttrPowerBattery3PercentageThreshold3   AttributeID = 0x007D // uint8.
	AttrPowerBattery3AlarmState             AttributeID = 0x007E // bitmap32.
)

// Identify Cluster Attributes (0x0003).
const (
	AttrIdentifyTime AttributeID = 0x0000 // Time remaining in identify mode (seconds).
)

// Identify Cluster Commands (0x0003).
const (
	CmdIdentify      uint8 = 0x00 // Start identifying for N seconds.
	CmdIdentifyQuery uint8 = 0x01 // Query if device is identifying.
	CmdTriggerEffect uint8 = 0x40 // Trigger a specific effect (ZCL 6+).
)

// Identify Effect IDs for TriggerEffect command.
const (
	IdentifyEffectBlink        uint8 = 0x00 // Light flashes on/off once.
	IdentifyEffectBreathe      uint8 = 0x01 // Light fades in and out.
	IdentifyEffectOkay         uint8 = 0x02 // Brief green flash (color lights).
	IdentifyEffectChannelChg   uint8 = 0x0B // Brief orange flash (color lights).
	IdentifyEffectFinishEffect uint8 = 0xFE // Complete current effect gracefully.
	IdentifyEffectStopEffect   uint8 = 0xFF // Stop immediately.
)

// Groups Cluster Commands (0x0004).
const (
	CmdGroupsAdd              uint8 = 0x00 // Add group membership.
	CmdGroupsView             uint8 = 0x01 // View group info.
	CmdGroupsGetMembership    uint8 = 0x02 // Get list of groups device belongs to.
	CmdGroupsRemove           uint8 = 0x03 // Remove from a group.
	CmdGroupsRemoveAll        uint8 = 0x04 // Remove from all groups.
	CmdGroupsAddIfIdentifying uint8 = 0x05 // Add to group only if device is identifying.
)

// Groups Cluster Response Commands (0x0004).
const (
	CmdGroupsAddResponse           uint8 = 0x00
	CmdGroupsViewResponse          uint8 = 0x01
	CmdGroupsGetMembershipResponse uint8 = 0x02
	CmdGroupsRemoveResponse        uint8 = 0x03
)

// Scenes Cluster Attributes (0x0005).
const (
	AttrScenesSceneCount   AttributeID = 0x0000 // Number of scenes stored.
	AttrScenesCurrentScene AttributeID = 0x0001 // Currently active scene ID.
	AttrScenesCurrentGroup AttributeID = 0x0002 // Currently active group ID.
	AttrScenesSceneValid   AttributeID = 0x0003 // Whether current scene is valid.
	AttrScenesNameSupport  AttributeID = 0x0004 // Whether scene names are supported.
)

// Scenes Cluster Commands (0x0005).
const (
	CmdScenesAdd           uint8 = 0x00 // Add a scene with explicit attribute values.
	CmdScenesView          uint8 = 0x01 // View scene details.
	CmdScenesRemove        uint8 = 0x02 // Remove a scene.
	CmdScenesRemoveAll     uint8 = 0x03 // Remove all scenes for a group.
	CmdScenesStore         uint8 = 0x04 // Store current state as a scene.
	CmdScenesRecall        uint8 = 0x05 // Recall/activate a scene.
	CmdScenesGetMembership uint8 = 0x06 // Get list of scenes for a group.
	CmdScenesCopy          uint8 = 0x42 // Copy scenes between groups (ZCL 7+).
)

// Scenes Cluster Response Commands (0x0005).
const (
	CmdScenesAddResponse           uint8 = 0x00
	CmdScenesViewResponse          uint8 = 0x01
	CmdScenesRemoveResponse        uint8 = 0x02
	CmdScenesRemoveAllResponse     uint8 = 0x03
	CmdScenesStoreResponse         uint8 = 0x04
	CmdScenesGetMembershipResponse uint8 = 0x06
)

// PollControl Cluster Attributes (0x0020).
const (
	AttrPollControlCheckInInterval     AttributeID = 0x0000 // uint32: check-in interval in quarter seconds.
	AttrPollControlLongPollInterval    AttributeID = 0x0001 // uint32: long poll interval in quarter seconds.
	AttrPollControlShortPollInterval   AttributeID = 0x0002 // uint16: short poll interval in quarter seconds.
	AttrPollControlFastPollTimeout     AttributeID = 0x0003 // uint16: fast poll timeout in quarter seconds.
	AttrPollControlCheckInIntervalMin  AttributeID = 0x0004 // uint32: minimum check-in interval.
	AttrPollControlLongPollIntervalMin AttributeID = 0x0005 // uint32: minimum long poll interval.
	AttrPollControlFastPollTimeoutMax  AttributeID = 0x0006 // uint16: maximum fast poll timeout.
)

// PollControl Cluster Commands - Server to Client (from sleepy device).
const (
	CmdPollControlCheckIn uint8 = 0x00 // Device check-in notification.
)

// PollControl Cluster Commands - Client to Server (to sleepy device).
const (
	CmdPollControlCheckInResponse      uint8 = 0x00 // Response to check-in.
	CmdPollControlFastPollStop         uint8 = 0x01 // Stop fast polling.
	CmdPollControlSetLongPollInterval  uint8 = 0x02 // Set long poll interval.
	CmdPollControlSetShortPollInterval uint8 = 0x03 // Set short poll interval.
)

// On/Off Cluster Attributes (0x0006).
const (
	AttrOnOff           AttributeID = 0x0000
	AttrGlobalSceneCtrl AttributeID = 0x4000
	AttrOnTime          AttributeID = 0x4001
	AttrOffWaitTime     AttributeID = 0x4002
	AttrStartUpOnOff    AttributeID = 0x4003
)

// Level Control Cluster Attributes (0x0008).
const (
	AttrLevelCurrentLevel     AttributeID = 0x0000 // Current brightness level (0-254).
	AttrLevelRemainingTime    AttributeID = 0x0001 // Time remaining in transition (1/10s).
	AttrLevelMinLevel         AttributeID = 0x0002 // Minimum allowed level.
	AttrLevelMaxLevel         AttributeID = 0x0003 // Maximum allowed level.
	AttrLevelCurrentFrequency AttributeID = 0x0004 // Current frequency (Hz).
	AttrLevelMinFrequency     AttributeID = 0x0005 // Minimum frequency (Hz).
	AttrLevelMaxFrequency     AttributeID = 0x0006 // Maximum frequency (Hz).
	AttrLevelOptions          AttributeID = 0x000F // Behavior options bitmap.
	AttrLevelOnOffTransTime   AttributeID = 0x0010 // Transition time for On/Off (1/10s).
	AttrLevelOnLevel          AttributeID = 0x0011 // Level when turned on (0xFF=restore).
	AttrLevelOnTransTime      AttributeID = 0x0012 // Transition time for on (1/10s).
	AttrLevelOffTransTime     AttributeID = 0x0013 // Transition time for off (1/10s).
	AttrLevelDefaultMoveRate  AttributeID = 0x0014 // Default rate for Move command.
	AttrLevelStartupLevel     AttributeID = 0x4000 // Level on power-up (0xFF=previous).
)

// MoveMode defines the direction for Move and Step commands.
type MoveMode uint8

const (
	MoveModeUp   MoveMode = 0x00 // Increase level.
	MoveModeDown MoveMode = 0x01 // Decrease level.
)

// String returns human-readable move mode name.
func (m MoveMode) String() string {
	switch m {
	case MoveModeUp:
		return "Up"
	case MoveModeDown:
		return "Down"
	default:
		return fmt.Sprintf("MoveMode(0x%02X)", uint8(m))
	}
}

// StepMode defines the direction for Step commands (same as MoveMode).
type StepMode uint8

const (
	StepModeUp   StepMode = 0x00 // Increase level by step.
	StepModeDown StepMode = 0x01 // Decrease level by step.
)

// String returns human-readable step mode name.
func (s StepMode) String() string {
	switch s {
	case StepModeUp:
		return "Up"
	case StepModeDown:
		return "Down"
	default:
		return fmt.Sprintf("StepMode(0x%02X)", uint8(s))
	}
}

// Alarms Cluster Attributes (0x0009).
const (
	AttrAlarmCount AttributeID = 0x0000 // Number of alarm entries currently in the alarm log (uint16).
)

// Alarms Cluster Commands - Client to Server (0x0009).
const (
	CmdAlarmsResetAlarm     uint8 = 0x00 // Reset a specific alarm.
	CmdAlarmsResetAllAlarms uint8 = 0x01 // Reset all alarms.
	CmdAlarmsGetAlarm       uint8 = 0x02 // Get an alarm entry from the log.
	CmdAlarmsResetAlarmLog  uint8 = 0x03 // Reset the alarm log.
)

// Alarms Cluster Commands - Server to Client (0x0009).
const (
	CmdAlarmsAlarm            uint8 = 0x00 // Alarm notification from device.
	CmdAlarmsGetAlarmResponse uint8 = 0x01 // Response to GetAlarm command.
)

// DoorLock Cluster Attributes (0x0101).
const (
	// Lock Information.
	AttrDoorLockState            AttributeID = 0x0000 // enum8 (locked, unlocked, etc.).
	AttrDoorLockType             AttributeID = 0x0001 // enum8.
	AttrDoorLockActuatorEnabled  AttributeID = 0x0002 // boolean.
	AttrDoorLockDoorState        AttributeID = 0x0003 // enum8 (open, closed).
	AttrDoorLockDoorOpenEvents   AttributeID = 0x0004 // uint32.
	AttrDoorLockDoorClosedEvents AttributeID = 0x0005 // uint32.
	AttrDoorLockOpenPeriod       AttributeID = 0x0006 // uint16.

	// User Settings.
	AttrDoorLockNumLogRecords AttributeID = 0x0010 // uint16.
	AttrDoorLockNumTotalUsers AttributeID = 0x0011 // uint16.
	AttrDoorLockNumPINUsers   AttributeID = 0x0012 // uint16.
	AttrDoorLockNumRFIDUsers  AttributeID = 0x0013 // uint16.
	AttrDoorLockMaxPINLen     AttributeID = 0x0017 // uint8.
	AttrDoorLockMinPINLen     AttributeID = 0x0018 // uint8.
	AttrDoorLockMaxRFIDLen    AttributeID = 0x0019 // uint8.
	AttrDoorLockMinRFIDLen    AttributeID = 0x001A // uint8.

	// Operating Settings.
	AttrDoorLockEnableLogging         AttributeID = 0x0020 // boolean.
	AttrDoorLockLanguage              AttributeID = 0x0021 // string.
	AttrDoorLockLEDSettings           AttributeID = 0x0022 // uint8.
	AttrDoorLockAutoRelockTime        AttributeID = 0x0023 // uint32.
	AttrDoorLockSoundVolume           AttributeID = 0x0024 // uint8.
	AttrDoorLockOperatingMode         AttributeID = 0x0025 // enum8.
	AttrDoorLockSupportedModes        AttributeID = 0x0026 // bitmap16.
	AttrDoorLockDefaultConfigReg      AttributeID = 0x0027 // bitmap16.
	AttrDoorLockEnableLocalProg       AttributeID = 0x0028 // boolean.
	AttrDoorLockEnableOneTouchLocking AttributeID = 0x0029 // boolean.
	AttrDoorLockEnablePrivacyButton   AttributeID = 0x002B // boolean.
	AttrDoorLockWrongCodeEntryLimit   AttributeID = 0x0030 // uint8.
	AttrDoorLockUserCodeTempDisable   AttributeID = 0x0031 // uint8.
	AttrDoorLockSendPINOverTheAir     AttributeID = 0x0032 // boolean
	AttrDoorLockRequirePINForRF       AttributeID = 0x0033 // boolean
)

// DoorLockState represents the current state of the door lock.
type DoorLockState uint8

const (
	DoorLockStateNotFullyLocked DoorLockState = 0x00 // Lock is not fully locked
	DoorLockStateLocked         DoorLockState = 0x01 // Lock is locked
	DoorLockStateUnlocked       DoorLockState = 0x02 // Lock is unlocked
	DoorLockStateUndefined      DoorLockState = 0xFF // Lock state is undefined
)

// String returns human-readable door lock state name.
func (d DoorLockState) String() string {
	switch d {
	case DoorLockStateNotFullyLocked:
		return "NotFullyLocked"
	case DoorLockStateLocked:
		return "Locked"
	case DoorLockStateUnlocked:
		return "Unlocked"
	case DoorLockStateUndefined:
		return "Undefined"
	default:
		return fmt.Sprintf("DoorLockState(0x%02X)", uint8(d))
	}
}

// DoorState represents the current state of the door.
type DoorState uint8

const (
	DoorStateOpen        DoorState = 0x00 // Door is open
	DoorStateClosed      DoorState = 0x01 // Door is closed
	DoorStateJammed      DoorState = 0x02 // Door is jammed
	DoorStateForcedOpen  DoorState = 0x03 // Door was forced open
	DoorStateUnspecified DoorState = 0x04 // Door state is unspecified
)

// String returns human-readable door state name.
func (d DoorState) String() string {
	switch d {
	case DoorStateOpen:
		return "Open"
	case DoorStateClosed:
		return "Closed"
	case DoorStateJammed:
		return "Jammed"
	case DoorStateForcedOpen:
		return "ForcedOpen"
	case DoorStateUnspecified:
		return "Unspecified"
	default:
		return fmt.Sprintf("DoorState(0x%02X)", uint8(d))
	}
}

// DoorLock Cluster Commands (0x0101)
const (
	CmdDoorLockLock              uint8 = 0x00 // Lock the door
	CmdDoorLockUnlock            uint8 = 0x01 // Unlock the door
	CmdDoorLockToggle            uint8 = 0x02 // Toggle lock state
	CmdDoorLockUnlockWithTimeout uint8 = 0x03 // Unlock with automatic relock
	CmdDoorLockSetPINCode        uint8 = 0x05 // Set user PIN code
	CmdDoorLockGetPINCode        uint8 = 0x06 // Get user PIN code
	CmdDoorLockClearPINCode      uint8 = 0x07 // Clear user PIN code
	CmdDoorLockClearAllPINCodes  uint8 = 0x08 // Clear all PIN codes
	CmdDoorLockSetUserStatus     uint8 = 0x09 // Set user status
)

// WindowCovering Cluster Attributes (0x0102)
const (
	// Information Attributes
	AttrWindowCoveringType                       AttributeID = 0x0000 // enum8.
	AttrWindowCoveringPhysicalClosedLimitLift    AttributeID = 0x0001 // uint16.
	AttrWindowCoveringPhysicalClosedLimitTilt    AttributeID = 0x0002 // uint16.
	AttrWindowCoveringCurrentPositionLift        AttributeID = 0x0003 // uint16.
	AttrWindowCoveringCurrentPositionTilt        AttributeID = 0x0004 // uint16.
	AttrWindowCoveringNumActuationsLift          AttributeID = 0x0005 // uint16.
	AttrWindowCoveringNumActuationsTilt          AttributeID = 0x0006 // uint16.
	AttrWindowCoveringConfigStatus               AttributeID = 0x0007 // bitmap8: Configuration status
	AttrWindowCoveringCurrentPositionLiftPercent AttributeID = 0x0008 // uint8 (0-100%)
	AttrWindowCoveringCurrentPositionTiltPercent AttributeID = 0x0009 // uint8 (0-100%)
	AttrWindowCoveringOperationalStatus          AttributeID = 0x000A // bitmap8: Current operation status
	AttrWindowCoveringTargetPositionLiftPct      AttributeID = 0x000B // uint8: Target lift position %
	AttrWindowCoveringTargetPositionTiltPct      AttributeID = 0x000C // uint8: Target tilt position %

	// Settings Attributes
	AttrWindowCoveringInstalledOpenLimitLift    AttributeID = 0x0010 // uint16.
	AttrWindowCoveringInstalledClosedLimitLift  AttributeID = 0x0011 // uint16.
	AttrWindowCoveringInstalledOpenLimitTilt    AttributeID = 0x0012 // uint16.
	AttrWindowCoveringInstalledClosedLimitTilt  AttributeID = 0x0013 // uint16.
	AttrWindowCoveringVelocityLift              AttributeID = 0x0014 // uint16.
	AttrWindowCoveringAccelerationTimeLift      AttributeID = 0x0015 // uint16.
	AttrWindowCoveringDecelerationTimeLift      AttributeID = 0x0016 // uint16.
	AttrWindowCoveringMode                      AttributeID = 0x0017 // bitmap8: Mode
	AttrWindowCoveringIntermediateSetpointsLift AttributeID = 0x0018 // octet string
	AttrWindowCoveringIntermediateSetpointsTilt AttributeID = 0x0019 // octet string
)

// WindowCovering Cluster Commands (0x0102)
const (
	CmdWindowCoveringUpOpen          uint8 = 0x00 // Open/raise
	CmdWindowCoveringDownClose       uint8 = 0x01 // Close/lower
	CmdWindowCoveringStop            uint8 = 0x02 // Stop movement
	CmdWindowCoveringGoToLiftValue   uint8 = 0x04 // Move to lift value (uint16)
	CmdWindowCoveringGoToLiftPercent uint8 = 0x05 // Move to lift percentage (uint8)
	CmdWindowCoveringGoToTiltValue   uint8 = 0x07 // Move to tilt value (uint16)
	CmdWindowCoveringGoToTiltPercent uint8 = 0x08 // Move to tilt percentage (uint8)
)

// Color Control Cluster Attributes (0x0300)
const (
	AttrColorCurrentHue        AttributeID = 0x0000
	AttrColorCurrentSaturation AttributeID = 0x0001
	AttrColorRemainingTime     AttributeID = 0x0002
	AttrColorCurrentX          AttributeID = 0x0003
	AttrColorCurrentY          AttributeID = 0x0004
	AttrColorTempMireds        AttributeID = 0x0007 // Color temperature in mireds
	AttrColorMode              AttributeID = 0x0008 // 0=HS, 1=XY, 2=ColorTemp
	AttrColorTempMin           AttributeID = 0x400B // Min color temp in mireds
	AttrColorTempMax           AttributeID = 0x400C // Max color temp in mireds

	// Enhanced Color Control attributes (ZCL 6+)
	AttrColorCapabilities               AttributeID = 0x400A // bitmap16: Supported color features (MANDATORY)
	AttrColorEnhancedCurrentHue         AttributeID = 0x4000 // uint16: Enhanced hue (0-65535)
	AttrColorEnhancedColorMode          AttributeID = 0x4001 // enum8: Enhanced color mode
	AttrColorColorLoopActive            AttributeID = 0x4002 // uint8: Color loop active
	AttrColorColorLoopDirection         AttributeID = 0x4003 // uint8: Color loop direction
	AttrColorColorLoopTime              AttributeID = 0x4004 // uint16: Color loop time
	AttrColorColorLoopStartEnhancedHue  AttributeID = 0x4005 // uint16.
	AttrColorColorLoopStoredEnhancedHue AttributeID = 0x4006 // uint16.
)

// Color Control Cluster Commands (0x0300)
const (
	CmdColorMoveToHue           uint8 = 0x00 // Move to hue
	CmdColorMoveHue             uint8 = 0x01 // Move hue
	CmdColorStepHue             uint8 = 0x02 // Step hue
	CmdColorMoveToSaturation    uint8 = 0x03 // Move to saturation
	CmdColorMoveSaturation      uint8 = 0x04 // Move saturation
	CmdColorStepSaturation      uint8 = 0x05 // Step saturation
	CmdColorMoveToHueSaturation uint8 = 0x06 // Move to hue and saturation
	CmdColorMoveToColor         uint8 = 0x07 // Move to XY color
	CmdColorMoveColor           uint8 = 0x08 // Move color
	CmdColorStepColor           uint8 = 0x09 // Step color
	CmdColorMoveToColorTemp     uint8 = 0x0A // Move to color temperature

	// Enhanced Color Control commands (ZCL 6+)
	CmdColorEnhancedMoveToHue              uint8 = 0x40
	CmdColorEnhancedMoveHue                uint8 = 0x41
	CmdColorEnhancedStepHue                uint8 = 0x42
	CmdColorEnhancedMoveToHueAndSaturation uint8 = 0x43
	CmdColorColorLoopSet                   uint8 = 0x44
	CmdColorStopMoveStep                   uint8 = 0x47 // Stop move/step
	CmdColorMoveColorTemp                  uint8 = 0x4B // Move color temperature
	CmdColorStepColorTemp                  uint8 = 0x4C // Step color temperature
)

// ColorCapabilities bitmap values (attribute 0x400A)
const (
	ColorCapabilityHueSaturation uint16 = 1 << 0 // Supports Hue/Saturation
	ColorCapabilityEnhancedHue   uint16 = 1 << 1 // Supports Enhanced Hue
	ColorCapabilityColorLoop     uint16 = 1 << 2 // Supports Color Loop
	ColorCapabilityXY            uint16 = 1 << 3 // Supports XY attributes
	ColorCapabilityColorTemp     uint16 = 1 << 4 // Supports Color Temperature
)

// Thermostat Cluster Attributes (0x0201)
const (
	// Thermostat Information Set
	AttrThermostatLocalTemp          AttributeID = 0x0000 // int16 in 0.01°C
	AttrThermostatOutdoorTemp        AttributeID = 0x0001 // int16 in 0.01°C
	AttrThermostatOccupancy          AttributeID = 0x0002 // bitmap8.
	AttrThermostatAbsMinHeatSetpoint AttributeID = 0x0003 // int16 in 0.01°C
	AttrThermostatAbsMaxHeatSetpoint AttributeID = 0x0004 // int16 in 0.01°C
	AttrThermostatAbsMinCoolSetpoint AttributeID = 0x0005 // int16 in 0.01°C
	AttrThermostatAbsMaxCoolSetpoint AttributeID = 0x0006 // int16 in 0.01°C
	AttrThermostatPICoolingDemand    AttributeID = 0x0007 // uint8 (0-100%)
	AttrThermostatPIHeatingDemand    AttributeID = 0x0008 // uint8 (0-100%)

	// Thermostat Settings Set
	AttrThermostatLocalTempCalibration      AttributeID = 0x0010 // int8 in 0.1°C
	AttrThermostatOccupiedCoolingSetpoint   AttributeID = 0x0011 // int16 in 0.01°C
	AttrThermostatOccupiedHeatingSetpoint   AttributeID = 0x0012 // int16 in 0.01°C
	AttrThermostatUnoccupiedCoolingSetpoint AttributeID = 0x0013 // int16 in 0.01°C
	AttrThermostatUnoccupiedHeatingSetpoint AttributeID = 0x0014 // int16 in 0.01°C
	AttrThermostatMinHeatSetpointLimit      AttributeID = 0x0015 // int16 in 0.01°C
	AttrThermostatMaxHeatSetpointLimit      AttributeID = 0x0016 // int16 in 0.01°C
	AttrThermostatMinCoolSetpointLimit      AttributeID = 0x0017 // int16 in 0.01°C
	AttrThermostatMaxCoolSetpointLimit      AttributeID = 0x0018 // int16 in 0.01°C
	AttrThermostatMinSetpointDeadBand       AttributeID = 0x0019 // int8
	AttrThermostatControlSeqOfOper          AttributeID = 0x001B // enum8.
	AttrThermostatSystemMode                AttributeID = 0x001C // enum8.
	AttrThermostatRunningMode               AttributeID = 0x001E // enum8.
	AttrThermostatRunningState              AttributeID = 0x0029 // bitmap16
)

// ThermostatSystemMode represents the operating mode of a thermostat.
type ThermostatSystemMode uint8

const (
	ThermostatModeOff           ThermostatSystemMode = 0x00 // Thermostat is off
	ThermostatModeAuto          ThermostatSystemMode = 0x01 // Auto mode (heat/cool as needed)
	ThermostatModeCool          ThermostatSystemMode = 0x03 // Cooling mode
	ThermostatModeHeat          ThermostatSystemMode = 0x04 // Heating mode
	ThermostatModeEmergencyHeat ThermostatSystemMode = 0x05 // Emergency heating mode
	ThermostatModePrecooling    ThermostatSystemMode = 0x06 // Precooling mode
	ThermostatModeFanOnly       ThermostatSystemMode = 0x07 // Fan only mode
	ThermostatModeDry           ThermostatSystemMode = 0x08 // Dry/dehumidify mode
	ThermostatModeSleep         ThermostatSystemMode = 0x09 // Sleep mode
)

// String returns human-readable thermostat system mode name.
func (m ThermostatSystemMode) String() string {
	switch m {
	case ThermostatModeOff:
		return "Off"
	case ThermostatModeAuto:
		return "Auto"
	case ThermostatModeCool:
		return "Cool"
	case ThermostatModeHeat:
		return "Heat"
	case ThermostatModeEmergencyHeat:
		return "EmergencyHeat"
	case ThermostatModePrecooling:
		return "Precooling"
	case ThermostatModeFanOnly:
		return "FanOnly"
	case ThermostatModeDry:
		return "Dry"
	case ThermostatModeSleep:
		return "Sleep"
	default:
		return fmt.Sprintf("ThermostatSystemMode(0x%02X)", uint8(m))
	}
}

// Thermostat Cluster Commands (0x0201)
const (
	CmdThermostatSetpointRaiseLower  uint8 = 0x00 // Raise/lower setpoint
	CmdThermostatSetWeeklySchedule   uint8 = 0x01 // Set weekly schedule
	CmdThermostatGetWeeklySchedule   uint8 = 0x02 // Get weekly schedule
	CmdThermostatClearWeeklySchedule uint8 = 0x03 // Clear weekly schedule
)

// Fan Cluster Attributes (0x0202)
const (
	AttrFanMode           AttributeID = 0x0000 // enum8: current fan mode
	AttrFanModeSequence   AttributeID = 0x0001 // enum8: supported fan mode sequence
	AttrFanPercentSetting AttributeID = 0x0002 // uint8: fan speed as percentage (0-100%)
	AttrFanPercentCurrent AttributeID = 0x0003 // uint8: current fan speed percentage (0-100%)
	AttrFanSpeedMax       AttributeID = 0x0004 // uint8: maximum fan speed
	AttrFanSpeedSetting   AttributeID = 0x0005 // uint8: fan speed setting
	AttrFanSpeedCurrent   AttributeID = 0x0006 // uint8: current fan speed
)

// FanMode represents the operating mode of a fan.
type FanMode uint8

const (
	FanModeOff    FanMode = 0x00 // Fan is off
	FanModeLow    FanMode = 0x01 // Low speed
	FanModeMedium FanMode = 0x02 // Medium speed
	FanModeHigh   FanMode = 0x03 // High speed
	FanModeOn     FanMode = 0x04 // On (device determines speed)
	FanModeAuto   FanMode = 0x05 // Automatic mode
	FanModeSmart  FanMode = 0x06 // Smart mode
)

// String returns human-readable fan mode name.
func (f FanMode) String() string {
	switch f {
	case FanModeOff:
		return "Off"
	case FanModeLow:
		return "Low"
	case FanModeMedium:
		return "Medium"
	case FanModeHigh:
		return "High"
	case FanModeOn:
		return "On"
	case FanModeAuto:
		return "Auto"
	case FanModeSmart:
		return "Smart"
	default:
		return fmt.Sprintf("FanMode(0x%02X)", uint8(f))
	}
}

// FanModeSequence represents the sequence of fan modes available.
type FanModeSequence uint8

const (
	FanSequenceLowMedHigh     FanModeSequence = 0x00 // Low, Medium, High
	FanSequenceLowHigh        FanModeSequence = 0x01 // Low, High
	FanSequenceLowMedHighAuto FanModeSequence = 0x02 // Low, Medium, High, Auto
	FanSequenceLowHighAuto    FanModeSequence = 0x03 // Low, High, Auto
	FanSequenceOnAuto         FanModeSequence = 0x04 // On, Auto
)

// String returns human-readable fan mode sequence name.
func (f FanModeSequence) String() string {
	switch f {
	case FanSequenceLowMedHigh:
		return "LowMedHigh"
	case FanSequenceLowHigh:
		return "LowHigh"
	case FanSequenceLowMedHighAuto:
		return "LowMedHighAuto"
	case FanSequenceLowHighAuto:
		return "LowHighAuto"
	case FanSequenceOnAuto:
		return "OnAuto"
	default:
		return fmt.Sprintf("FanModeSequence(0x%02X)", uint8(f))
	}
}

// Time Cluster Attributes (0x000A)
const (
	AttrTimeTime           AttributeID = 0x0000 // uint32: UTC time in seconds since Jan 1, 2000
	AttrTimeStatus         AttributeID = 0x0001 // bitmap8: status flags
	AttrTimeZone           AttributeID = 0x0002 // int32: offset from UTC in seconds
	AttrTimeDstStart       AttributeID = 0x0003 // uint32: DST start time
	AttrTimeDstEnd         AttributeID = 0x0004 // uint32: DST end time
	AttrTimeDstShift       AttributeID = 0x0005 // int32: DST shift in seconds
	AttrTimeStandardTime   AttributeID = 0x0006 // uint32: standard time
	AttrTimeLocalTime      AttributeID = 0x0007 // uint32: local time
	AttrTimeLastSetTime    AttributeID = 0x0008 // uint32: last time Time was set
	AttrTimeValidUntilTime AttributeID = 0x0009 // uint32: time until which Time is valid
)

// TimeStatus bitmap flags.
const (
	TimeStatusMaster        uint8 = 1 << 0 // Device is time master
	TimeStatusSynchronized  uint8 = 1 << 1 // Time is synchronized
	TimeStatusMasterZoneDst uint8 = 1 << 2 // Master for timezone/DST
	TimeStatusSuperseding   uint8 = 1 << 3 // Can supersede other masters
)

// Binary Input Cluster Attributes (0x000F)
const (
	AttrBinaryInputActiveText   AttributeID = 0x0004 // string: text for active state
	AttrBinaryInputDescription  AttributeID = 0x001C // string: description of input
	AttrBinaryInputInactiveText AttributeID = 0x002E // string: text for inactive state
	AttrBinaryInputOutOfService AttributeID = 0x0051 // boolean: out of service flag
	AttrBinaryInputPolarity     AttributeID = 0x0054 // enum8: 0=normal, 1=reversed
	AttrBinaryInputPresentValue AttributeID = 0x0055 // boolean: current input value
	AttrBinaryInputReliability  AttributeID = 0x0067 // enum8: reliability state
	AttrBinaryInputStatusFlags  AttributeID = 0x006F // bitmap8: status flags
)

// Binary Input StatusFlags bitmap.
const (
	BinaryInputStatusFlagInAlarm      uint8 = 1 << 0 // In alarm state
	BinaryInputStatusFlagFault        uint8 = 1 << 1 // Fault detected
	BinaryInputStatusFlagOverridden   uint8 = 1 << 2 // Value overridden
	BinaryInputStatusFlagOutOfService uint8 = 1 << 3 // Out of service
)

// IOReliability represents the reliability status of input/output values.
// This enum is used by Binary Input, Analog Input, and other I/O clusters.
type IOReliability uint8

const (
	IOReliabilityNoFaultDetected IOReliability = 0x00 // No fault detected
	IOReliabilityNoSensor        IOReliability = 0x01 // No sensor present
	IOReliabilityOverRange       IOReliability = 0x02 // Value over range
	IOReliabilityUnderRange      IOReliability = 0x03 // Value under range
	IOReliabilityOpenLoop        IOReliability = 0x04 // Open loop detected
	IOReliabilityShortedLoop     IOReliability = 0x05 // Shorted loop detected
	IOReliabilityNoOutput        IOReliability = 0x06 // No output
	IOReliabilityUnreliableOther IOReliability = 0x07 // Unreliable other reason
)

// String returns human-readable reliability name.
func (r IOReliability) String() string {
	switch r {
	case IOReliabilityNoFaultDetected:
		return "NoFaultDetected"
	case IOReliabilityNoSensor:
		return "NoSensor"
	case IOReliabilityOverRange:
		return "OverRange"
	case IOReliabilityUnderRange:
		return "UnderRange"
	case IOReliabilityOpenLoop:
		return "OpenLoop"
	case IOReliabilityShortedLoop:
		return "ShortedLoop"
	case IOReliabilityNoOutput:
		return "NoOutput"
	case IOReliabilityUnreliableOther:
		return "UnreliableOther"
	default:
		return fmt.Sprintf("IOReliability(0x%02X)", uint8(r))
	}
}

// AnalogInput Cluster Attributes (0x000C)
const (
	AttrAnalogInputDescription      AttributeID = 0x001C // string: description of input
	AttrAnalogInputMaxPresentValue  AttributeID = 0x0041 // single (float32): max value
	AttrAnalogInputMinPresentValue  AttributeID = 0x0045 // single (float32): min value
	AttrAnalogInputOutOfService     AttributeID = 0x0051 // boolean: out of service
	AttrAnalogInputPresentValue     AttributeID = 0x0055 // single (float32): current analog value
	AttrAnalogInputReliability      AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrAnalogInputResolution       AttributeID = 0x006A // single (float32): resolution
	AttrAnalogInputStatusFlags      AttributeID = 0x006F // bitmap8: status flags
	AttrAnalogInputEngineeringUnits AttributeID = 0x0075 // enum16: engineering units
)

// EngineeringUnits represents the units of measurement for analog values.

// AnalogOutput Cluster Attributes (0x000D)
const (
	AttrAnalogOutputDescription       AttributeID = 0x001C // string: description of output
	AttrAnalogOutputMaxPresentValue   AttributeID = 0x0041 // single (float32): max value
	AttrAnalogOutputMinPresentValue   AttributeID = 0x0045 // single (float32): min value
	AttrAnalogOutputOutOfService      AttributeID = 0x0051 // boolean: out of service
	AttrAnalogOutputPresentValue      AttributeID = 0x0055 // single (float32): current output value (writable)
	AttrAnalogOutputReliability       AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrAnalogOutputRelinquishDefault AttributeID = 0x0068 // single (float32): default when released
	AttrAnalogOutputResolution        AttributeID = 0x006A // single (float32): resolution
	AttrAnalogOutputStatusFlags       AttributeID = 0x006F // bitmap8: status flags
	AttrAnalogOutputEngineeringUnits  AttributeID = 0x0075 // enum16: engineering units
)

type EngineeringUnits uint16

// EngineeringUnits values for common measurements.
const (
	EngineeringUnitsSquareMeters      EngineeringUnits = 0  // Area (m²)
	EngineeringUnitsCubicMeters       EngineeringUnits = 2  // Volume (m³)
	EngineeringUnitsAmperes           EngineeringUnits = 3  // Electric current (A)
	EngineeringUnitsVolts             EngineeringUnits = 5  // Electric potential (V)
	EngineeringUnitsWatts             EngineeringUnits = 47 // Power (W)
	EngineeringUnitsKilowatts         EngineeringUnits = 48 // Power (kW)
	EngineeringUnitsDegreesCelsius    EngineeringUnits = 62 // Temperature (°C)
	EngineeringUnitsDegreesFahrenheit EngineeringUnits = 64 // Temperature (°F)
	EngineeringUnitsLiters            EngineeringUnits = 75 // Volume (L)
	EngineeringUnitsNoUnits           EngineeringUnits = 95 // Dimensionless
	EngineeringUnitsPercent           EngineeringUnits = 98 // Percent (%)
)

// String returns human-readable engineering units name.
func (e EngineeringUnits) String() string {
	switch e {
	case EngineeringUnitsSquareMeters:
		return "m²"
	case EngineeringUnitsCubicMeters:
		return "m³"
	case EngineeringUnitsAmperes:
		return "A"
	case EngineeringUnitsVolts:
		return "V"
	case EngineeringUnitsWatts:
		return "W"
	case EngineeringUnitsKilowatts:
		return "kW"
	case EngineeringUnitsDegreesCelsius:
		return "°C"
	case EngineeringUnitsDegreesFahrenheit:
		return "°F"
	case EngineeringUnitsLiters:
		return "L"
	case EngineeringUnitsNoUnits:
		return "NoUnits"
	case EngineeringUnitsPercent:
		return "%"
	default:
		return fmt.Sprintf("EngineeringUnits(0x%04X)", uint16(e))
	}
}

// AnalogValue Cluster Attributes (0x000E)
const (
	AttrAnalogValueDescription       AttributeID = 0x001C // string: description
	AttrAnalogValueOutOfService      AttributeID = 0x0051 // boolean: out of service
	AttrAnalogValuePresentValue      AttributeID = 0x0055 // single (float32): current analog value (read/write)
	AttrAnalogValueReliability       AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrAnalogValueRelinquishDefault AttributeID = 0x0068 // single (float32): default when released
	AttrAnalogValueStatusFlags       AttributeID = 0x006F // bitmap8: status flags
	AttrAnalogValueEngineeringUnits  AttributeID = 0x0075 // enum16: engineering units
)

// BinaryOutput Cluster Attributes (0x0010)
const (
	AttrBinaryOutputActiveText        AttributeID = 0x0004 // string: text for active state
	AttrBinaryOutputDescription       AttributeID = 0x001C // string: description
	AttrBinaryOutputInactiveText      AttributeID = 0x002E // string: text for inactive state
	AttrBinaryOutputMinimumOffTime    AttributeID = 0x0042 // uint32: minimum off time in ms
	AttrBinaryOutputMinimumOnTime     AttributeID = 0x0043 // uint32: minimum on time in ms
	AttrBinaryOutputOutOfService      AttributeID = 0x0051 // boolean: out of service
	AttrBinaryOutputPolarity          AttributeID = 0x0054 // enum8: 0=normal, 1=reversed
	AttrBinaryOutputPresentValue      AttributeID = 0x0055 // boolean: current output value (writable)
	AttrBinaryOutputReliability       AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrBinaryOutputRelinquishDefault AttributeID = 0x0068 // boolean: default when released
	AttrBinaryOutputStatusFlags       AttributeID = 0x006F // bitmap8: status flags
)

// BinaryOutput StatusFlags bitmap.
const (
	BinaryOutputStatusFlagInAlarm      uint8 = 1 << 0 // In alarm state
	BinaryOutputStatusFlagFault        uint8 = 1 << 1 // Fault detected
	BinaryOutputStatusFlagOverridden   uint8 = 1 << 2 // Value overridden
	BinaryOutputStatusFlagOutOfService uint8 = 1 << 3 // Out of service
)

// BinaryValue Cluster Attributes (0x0011)
const (
	AttrBinaryValueActiveText     AttributeID = 0x0004 // string: text for active state
	AttrBinaryValueDescription    AttributeID = 0x001C // string: description
	AttrBinaryValueInactiveText   AttributeID = 0x002E // string: text for inactive state
	AttrBinaryValueMinimumOffTime AttributeID = 0x0042 // uint32: minimum off time in milliseconds
	AttrBinaryValueMinimumOnTime  AttributeID = 0x0043 // uint32: minimum on time in milliseconds
	AttrBinaryValueOutOfService   AttributeID = 0x0051 // boolean: out of service flag
	AttrBinaryValuePresentValue   AttributeID = 0x0055 // boolean: current value (read/write)
	AttrBinaryValueReliability    AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrBinaryValueRelinquishDef  AttributeID = 0x0068 // boolean: default value when released
	AttrBinaryValueStatusFlags    AttributeID = 0x006F // bitmap8: status flags
)

// BinaryValue StatusFlags bitmap.
const (
	BinaryValueStatusFlagInAlarm      uint8 = 1 << 0 // In alarm state
	BinaryValueStatusFlagFault        uint8 = 1 << 1 // Fault detected
	BinaryValueStatusFlagOverridden   uint8 = 1 << 2 // Value overridden
	BinaryValueStatusFlagOutOfService uint8 = 1 << 3 // Out of service
)

// MultistateInput Cluster Attributes (0x0012)
const (
	AttrMultistateInputStateText      AttributeID = 0x000E // array of strings: text for each state
	AttrMultistateInputDescription    AttributeID = 0x001C // string: description
	AttrMultistateInputNumberOfStates AttributeID = 0x004A // uint16: number of possible states
	AttrMultistateInputOutOfService   AttributeID = 0x0051 // boolean: out of service flag
	AttrMultistateInputPresentValue   AttributeID = 0x0055 // uint16: current state value (1-based)
	AttrMultistateInputReliability    AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrMultistateInputStatusFlags    AttributeID = 0x006F // bitmap8: status flags
)

// MultistateInput StatusFlags bitmap.
const (
	MultistateInputStatusFlagInAlarm      uint8 = 1 << 0 // In alarm state
	MultistateInputStatusFlagFault        uint8 = 1 << 1 // Fault detected
	MultistateInputStatusFlagOverridden   uint8 = 1 << 2 // Value overridden
	MultistateInputStatusFlagOutOfService uint8 = 1 << 3 // Out of service
)

// MultistateOutput Cluster Attributes (0x0013)
const (
	AttrMultistateOutputDescription       AttributeID = 0x001C // string: description
	AttrMultistateOutputNumberOfStates    AttributeID = 0x004A // uint16: number of states
	AttrMultistateOutputOutOfService      AttributeID = 0x0051 // boolean: out of service
	AttrMultistateOutputPresentValue      AttributeID = 0x0055 // uint16: current state (writable)
	AttrMultistateOutputReliability       AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrMultistateOutputRelinquishDefault AttributeID = 0x0068 // uint16: default when released
	AttrMultistateOutputStatusFlags       AttributeID = 0x006F // bitmap8: status flags
)

// MultistateValue Cluster Attributes (0x0014)
const (
	AttrMultistateValueDescription       AttributeID = 0x001C // string: description
	AttrMultistateValueNumberOfStates    AttributeID = 0x004A // uint16: number of states
	AttrMultistateValueOutOfService      AttributeID = 0x0051 // boolean: out of service
	AttrMultistateValuePresentValue      AttributeID = 0x0055 // uint16: current value (read/write)
	AttrMultistateValueReliability       AttributeID = 0x0067 // enum8: reliability (IOReliability)
	AttrMultistateValueRelinquishDefault AttributeID = 0x0068 // uint16: default when released
	AttrMultistateValueStatusFlags       AttributeID = 0x006F // bitmap8: status flags
)

// MultistateOutput StatusFlags bitmap.
const (
	MultistateOutputStatusFlagInAlarm      uint8 = 1 << 0 // In alarm state
	MultistateOutputStatusFlagFault        uint8 = 1 << 1 // Fault detected
	MultistateOutputStatusFlagOverridden   uint8 = 1 << 2 // Value overridden
	MultistateOutputStatusFlagOutOfService uint8 = 1 << 3 // Out of service
)

// MultistateValue StatusFlags bitmap.
const (
	MultistateValueStatusFlagInAlarm      uint8 = 1 << 0 // In alarm state
	MultistateValueStatusFlagFault        uint8 = 1 << 1 // Fault detected
	MultistateValueStatusFlagOverridden   uint8 = 1 << 2 // Value overridden
	MultistateValueStatusFlagOutOfService uint8 = 1 << 3 // Out of service
)

// Temperature Measurement Attributes (0x0402)
const (
	AttrTempMeasuredValue    AttributeID = 0x0000
	AttrTempMinMeasuredValue AttributeID = 0x0001
	AttrTempMaxMeasuredValue AttributeID = 0x0002
	AttrTempTolerance        AttributeID = 0x0003
)

// Humidity Measurement Attributes (0x0405)
const (
	AttrHumidityMeasuredValue    AttributeID = 0x0000
	AttrHumidityMinMeasuredValue AttributeID = 0x0001
	AttrHumidityMaxMeasuredValue AttributeID = 0x0002
	AttrHumidityTolerance        AttributeID = 0x0003
)

// Flow Measurement Cluster Attributes (0x0404)
const (
	AttrFlowMeasuredValue    AttributeID = 0x0000 // uint16: Measured flow in 0.1 m³/h
	AttrFlowMinMeasuredValue AttributeID = 0x0001 // uint16: Min measurable value
	AttrFlowMaxMeasuredValue AttributeID = 0x0002 // uint16: Max measurable value
	AttrFlowTolerance        AttributeID = 0x0003 // uint16: Tolerance
)

// Pressure Measurement Attributes (0x0403)
const (
	AttrPressureMeasuredValue    AttributeID = 0x0000 // int16 in 10 Pa (0.1 hPa)
	AttrPressureMinMeasuredValue AttributeID = 0x0001 // int16 in 10 Pa (0.1 hPa)
	AttrPressureMaxMeasuredValue AttributeID = 0x0002 // int16 in 10 Pa (0.1 hPa)
	AttrPressureTolerance        AttributeID = 0x0003 // uint16 in 10 Pa (0.1 hPa)
	AttrPressureScaledValue      AttributeID = 0x0010 // int16 scaled value
	AttrPressureMinScaledValue   AttributeID = 0x0011 // int16 scaled value
	AttrPressureMaxScaledValue   AttributeID = 0x0012 // int16 scaled value
	AttrPressureScaledTolerance  AttributeID = 0x0013 // uint16 scaled tolerance
	AttrPressureScale            AttributeID = 0x0014 // int8 (10^scale multiplier)
)

// Illuminance Measurement Attributes (0x0400)
const (
	AttrIlluminanceMeasuredValue    AttributeID = 0x0000
	AttrIlluminanceMinMeasuredValue AttributeID = 0x0001
	AttrIlluminanceMaxMeasuredValue AttributeID = 0x0002
	AttrIlluminanceTolerance        AttributeID = 0x0003
)

// Carbon Monoxide Concentration Measurement Attributes (0x040C)
const (
	AttrCOMeasuredValue    AttributeID = 0x0000 // single (float32) - ppm
	AttrCOMinMeasuredValue AttributeID = 0x0001 // single
	AttrCOMaxMeasuredValue AttributeID = 0x0002 // single
	AttrCOTolerance        AttributeID = 0x0003 // single
)

// Carbon Dioxide Concentration Measurement Attributes (0x040D)
const (
	AttrCO2MeasuredValue    AttributeID = 0x0000 // single (float32) - ppm
	AttrCO2MinMeasuredValue AttributeID = 0x0001 // single
	AttrCO2MaxMeasuredValue AttributeID = 0x0002 // single
	AttrCO2Tolerance        AttributeID = 0x0003 // single
)

// PM2.5 Measurement Attributes (0x042A)
const (
	AttrPM25MeasuredValue    AttributeID = 0x0000 // single (float32) - µg/m³
	AttrPM25MinMeasuredValue AttributeID = 0x0001 // single
	AttrPM25MaxMeasuredValue AttributeID = 0x0002 // single
	AttrPM25Tolerance        AttributeID = 0x0003 // single
)

// Formaldehyde Measurement Attributes (0x042B)
const (
	AttrFormaldehydeMeasuredValue    AttributeID = 0x0000 // single (float32) - ppm
	AttrFormaldehydeMinMeasuredValue AttributeID = 0x0001 // single
	AttrFormaldehydeMaxMeasuredValue AttributeID = 0x0002 // single
	AttrFormaldehydeTolerance        AttributeID = 0x0003 // single
)

// Occupancy Sensing Attributes (0x0406)
const (
	AttrOccupancyValue                            AttributeID = 0x0000
	AttrOccupancyType                             AttributeID = 0x0001
	AttrOccupancyPIROccupiedToUnoccupiedDelay     AttributeID = 0x0010
	AttrOccupancyPIRUnoccupiedToOccupiedDelay     AttributeID = 0x0011
	AttrOccupancyPIRUnoccupiedToOccupiedThreshold AttributeID = 0x0012
)

// IAS Zone Attributes (0x0500)
const (
	AttrIASZoneState                       AttributeID = 0x0000 // enum8: 0=not enrolled, 1=enrolled
	AttrIASZoneType                        AttributeID = 0x0001 // enum16: type of zone
	AttrIASZoneStatus                      AttributeID = 0x0002 // bitmap16: current zone status
	AttrIASZoneCIEAddr                     AttributeID = 0x0010 // IEEE address of CIE (coordinator)
	AttrIASZoneID                          AttributeID = 0x0011 // uint8: zone ID (0-254)
	AttrIASZoneNumZoneSensitivityLevels    AttributeID = 0x0012 // uint8: number of sensitivity levels
	AttrIASZoneCurrentZoneSensitivityLevel AttributeID = 0x0013 // uint8: current sensitivity level
)

// IASZoneType represents the type of IAS Zone device.
type IASZoneType uint16

const (
	IASZoneTypeStandardCIE       IASZoneType = 0x0000 // Standard CIE
	IASZoneTypeMotionSensor      IASZoneType = 0x000D // Motion sensor
	IASZoneTypeContactSwitch     IASZoneType = 0x0015 // Contact switch
	IASZoneTypeDoorWindow        IASZoneType = 0x0016 // Door/Window sensor (same as contact)
	IASZoneTypeFireSensor        IASZoneType = 0x0028 // Fire sensor
	IASZoneTypeWaterSensor       IASZoneType = 0x002A // Water sensor
	IASZoneTypeCOSensor          IASZoneType = 0x002B // Carbon monoxide sensor
	IASZoneTypePersonalEmergency IASZoneType = 0x002C // Personal emergency device
	IASZoneTypeVibrationMovement IASZoneType = 0x002D // Vibration/Movement sensor
	IASZoneTypeRemoteControl     IASZoneType = 0x010F // Remote control
	IASZoneTypeKeyFob            IASZoneType = 0x0115 // Key fob
	IASZoneTypeKeypad            IASZoneType = 0x021D // Keypad
	IASZoneTypeStandardWarning   IASZoneType = 0x0225 // Standard warning device
	IASZoneTypeGlassBreak        IASZoneType = 0x0226 // Glass break sensor
	IASZoneTypeSecurityRepeater  IASZoneType = 0x0229 // Security repeater
	IASZoneTypeInvalidZone       IASZoneType = 0xFFFF // Invalid zone type
)

// String returns human-readable zone type name.
func (z IASZoneType) String() string {
	switch z {
	case IASZoneTypeStandardCIE:
		return "StandardCIE"
	case IASZoneTypeMotionSensor:
		return "MotionSensor"
	case IASZoneTypeContactSwitch:
		return "ContactSwitch"
	case IASZoneTypeDoorWindow:
		return "DoorWindow"
	case IASZoneTypeFireSensor:
		return "FireSensor"
	case IASZoneTypeWaterSensor:
		return "WaterSensor"
	case IASZoneTypeCOSensor:
		return "COSensor"
	case IASZoneTypePersonalEmergency:
		return "PersonalEmergency"
	case IASZoneTypeVibrationMovement:
		return "VibrationMovement"
	case IASZoneTypeRemoteControl:
		return "RemoteControl"
	case IASZoneTypeKeyFob:
		return "KeyFob"
	case IASZoneTypeKeypad:
		return "Keypad"
	case IASZoneTypeStandardWarning:
		return "StandardWarning"
	case IASZoneTypeGlassBreak:
		return "GlassBreak"
	case IASZoneTypeSecurityRepeater:
		return "SecurityRepeater"
	case IASZoneTypeInvalidZone:
		return "InvalidZone"
	default:
		return fmt.Sprintf("IASZoneType(0x%04X)", uint16(z))
	}
}

// IAS Zone Status bitmap flags.
const (
	IASZoneStatusAlarm1         uint16 = 1 << 0 // Zone alarm 1
	IASZoneStatusAlarm2         uint16 = 1 << 1 // Zone alarm 2
	IASZoneStatusTamper         uint16 = 1 << 2 // Tamper detected
	IASZoneStatusBattery        uint16 = 1 << 3 // Low battery
	IASZoneStatusSupervision    uint16 = 1 << 4 // Supervision reports enabled
	IASZoneStatusRestoreReports uint16 = 1 << 5 // Restore reports enabled
	IASZoneStatusTrouble        uint16 = 1 << 6 // Trouble/failure detected
	IASZoneStatusACMains        uint16 = 1 << 7 // AC mains fault
	IASZoneStatusTest           uint16 = 1 << 8 // Sensor in test mode
	IASZoneStatusBatteryDefect  uint16 = 1 << 9 // Battery defect
)

// IAS Zone Cluster Commands (0x0500)
const (
	// Server to Client (notifications from device)
	CmdIASZoneStatusChangeNotification uint8 = 0x00 // Zone status change notification
	CmdIASZoneEnrollRequest            uint8 = 0x01 // Enroll request from device

	// Client to Server (commands to device)
	CmdIASZoneEnrollResponse uint8 = 0x00 // Enroll response to device
)

// Electrical Measurement Cluster Attributes (0x0B04)
// These provide real-time instantaneous electrical measurements.
const (
	// Basic Information
	AttrElecMeasType AttributeID = 0x0000 // Bitmap of measurement types supported

	// AC (Non-phase specific) Measurements
	AttrElecACFrequency        AttributeID = 0x0300 // Frequency in 0.01 Hz
	AttrElecACFrequencyMin     AttributeID = 0x0301
	AttrElecACFrequencyMax     AttributeID = 0x0302
	AttrElecACFrequencyDivisor AttributeID = 0x0400 // Divisor for frequency
	AttrElecPowerMultiplier    AttributeID = 0x0402 // Multiplier for power
	AttrElecPowerDivisor       AttributeID = 0x0403 // Divisor for power

	// AC Single Phase Measurements
	AttrElecRMSVoltage       AttributeID = 0x0505 // RMS voltage in volts (with divisor)
	AttrElecRMSVoltageMin    AttributeID = 0x0506
	AttrElecRMSVoltageMax    AttributeID = 0x0507
	AttrElecRMSCurrent       AttributeID = 0x0508 // RMS current in amps (with divisor)
	AttrElecRMSCurrentMin    AttributeID = 0x0509
	AttrElecRMSCurrentMax    AttributeID = 0x050A
	AttrElecActivePower      AttributeID = 0x050B // Active power in watts (with divisor)
	AttrElecActivePowerMin   AttributeID = 0x050C
	AttrElecActivePowerMax   AttributeID = 0x050D
	AttrElecReactivePower    AttributeID = 0x050E // Reactive power in VAR
	AttrElecApparentPower    AttributeID = 0x050F // Apparent power in VA
	AttrElecPowerFactor      AttributeID = 0x0510 // Power factor (-100 to 100, in %)
	AttrElecACVoltageMulti   AttributeID = 0x0600 // Multiplier for voltage
	AttrElecACVoltageDivisor AttributeID = 0x0601 // Divisor for voltage
	AttrElecACCurrentMulti   AttributeID = 0x0602 // Multiplier for current
	AttrElecACCurrentDivisor AttributeID = 0x0603 // Divisor for current
	AttrElecACPowerMulti     AttributeID = 0x0604 // Multiplier for power
	AttrElecACPowerDivisor   AttributeID = 0x0605 // Divisor for power
)

// Simple Metering Cluster Attributes (0x0702)
// These provide cumulative energy consumption readings.
const (
	// Reading Information Set
	AttrMeterCurrentSummation AttributeID = 0x0000 // Total energy consumed (with formatting)
	AttrMeterCurrentTier1     AttributeID = 0x0100 // Tier 1 consumption
	AttrMeterCurrentTier2     AttributeID = 0x0102 // Tier 2 consumption

	// Meter Status
	AttrMeterStatus AttributeID = 0x0200 // Metering device status

	// Formatting
	AttrMeterUnitOfMeasure    AttributeID = 0x0300 // Unit (0=kWh, 1=m3, etc.)
	AttrMeterMultiplier       AttributeID = 0x0301 // Multiplier for summation
	AttrMeterDivisor          AttributeID = 0x0302 // Divisor for summation
	AttrMeterSummationFormat  AttributeID = 0x0303 // Format of summation (decimal places)
	AttrMeterDemandFormat     AttributeID = 0x0304 // Format of demand
	AttrMeterHistoricalFormat AttributeID = 0x0305 // Format of historical data

	// Historical Consumption
	AttrMeterInstantDemand     AttributeID = 0x0400 // Instantaneous demand (watts)
	AttrMeterCurrentDayConsume AttributeID = 0x0401 // Today's consumption
	AttrMeterPrevDayConsume    AttributeID = 0x0403 // Yesterday's consumption
)

// DRLC (Demand Response and Load Control) Cluster Attributes (0x0701)
const (
	AttrDRLCUtilityEnrolmentGroup    AttributeID = 0x0000 // uint8: utility enrolment group
	AttrDRLCStartRandomizeMinutes    AttributeID = 0x0001 // uint8: randomize start time
	AttrDRLCDurationRandomizeMinutes AttributeID = 0x0002 // uint8: randomize duration
	AttrDRLCDeviceClassValue         AttributeID = 0x0003 // uint16: device class bitmap
)

// DRLC Device Class bitmap values.
// These identify the type of load control device.
const (
	DRLCDeviceClassHVAC             uint16 = 1 << 0  // HVAC compressor or furnace
	DRLCDeviceClassStrip            uint16 = 1 << 1  // Strip/baseboard heater
	DRLCDeviceClassWaterHeater      uint16 = 1 << 2  // Water heater
	DRLCDeviceClassPoolPump         uint16 = 1 << 3  // Pool pump/spa/jacuzzi
	DRLCDeviceClassSmartAppliance   uint16 = 1 << 4  // Smart appliance
	DRLCDeviceClassIrrigation       uint16 = 1 << 5  // Irrigation pump
	DRLCDeviceClassManagedCI        uint16 = 1 << 6  // Managed commercial & industrial loads
	DRLCDeviceClassSimpleMiscLoads  uint16 = 1 << 7  // Simple misc loads
	DRLCDeviceClassExteriorLighting uint16 = 1 << 8  // Exterior lighting
	DRLCDeviceClassInteriorLighting uint16 = 1 << 9  // Interior lighting
	DRLCDeviceClassEV               uint16 = 1 << 10 // Electric vehicle
	DRLCDeviceClassGeneration       uint16 = 1 << 11 // Generation systems
)

// DRLC Cluster Commands - Server to Client (0x0701)
const (
	CmdDRLCLoadControlEvent       uint8 = 0x00 // Load control event notification
	CmdDRLCCancelLoadControlEvent uint8 = 0x01 // Cancel a load control event
	CmdDRLCCancelAllLoadControl   uint8 = 0x02 // Cancel all load control events
)

// DRLC Cluster Commands - Client to Server (0x0701)
const (
	CmdDRLCReportEventStatus  uint8 = 0x00 // Report event status
	CmdDRLCGetScheduledEvents uint8 = 0x01 // Get scheduled events
)

// Price Cluster Attributes (0x0700)
// Part of the Smart Energy profile for pricing information.
const (
	// Tier Label Attributes - names for pricing tiers
	AttrPriceTierLabel1  AttributeID = 0x0000 // string: tier 1 label
	AttrPriceTierLabel2  AttributeID = 0x0001 // string: tier 2 label
	AttrPriceTierLabel3  AttributeID = 0x0002 // string: tier 3 label
	AttrPriceTierLabel4  AttributeID = 0x0003 // string: tier 4 label
	AttrPriceTierLabel5  AttributeID = 0x0004 // string: tier 5 label
	AttrPriceTierLabel6  AttributeID = 0x0005 // string: tier 6 label
	AttrPriceTierLabel7  AttributeID = 0x0006 // string: tier 7 label
	AttrPriceTierLabel8  AttributeID = 0x0007 // string: tier 8 label
	AttrPriceTierLabel9  AttributeID = 0x0008 // string: tier 9 label
	AttrPriceTierLabel10 AttributeID = 0x0009 // string: tier 10 label
	AttrPriceTierLabel11 AttributeID = 0x000A // string: tier 11 label
	AttrPriceTierLabel12 AttributeID = 0x000B // string: tier 12 label
	AttrPriceTierLabel13 AttributeID = 0x000C // string: tier 13 label
	AttrPriceTierLabel14 AttributeID = 0x000D // string: tier 14 label
	AttrPriceTierLabel15 AttributeID = 0x000E // string: tier 15 label
	AttrPriceTierLabel16 AttributeID = 0x000F // string: tier 16 label

	// Block Threshold Attributes - consumption thresholds for each block
	AttrPriceBlock1Threshold  AttributeID = 0x0100 // uint48: threshold for block 1
	AttrPriceBlock2Threshold  AttributeID = 0x0101 // uint48: threshold for block 2
	AttrPriceBlock3Threshold  AttributeID = 0x0102 // uint48: threshold for block 3
	AttrPriceBlock4Threshold  AttributeID = 0x0103 // uint48: threshold for block 4
	AttrPriceBlock5Threshold  AttributeID = 0x0104 // uint48: threshold for block 5
	AttrPriceBlock6Threshold  AttributeID = 0x0105 // uint48: threshold for block 6
	AttrPriceBlock7Threshold  AttributeID = 0x0106 // uint48: threshold for block 7
	AttrPriceBlock8Threshold  AttributeID = 0x0107 // uint48: threshold for block 8
	AttrPriceBlock9Threshold  AttributeID = 0x0108 // uint48: threshold for block 9
	AttrPriceBlock10Threshold AttributeID = 0x0109 // uint48: threshold for block 10
	AttrPriceBlock11Threshold AttributeID = 0x010A // uint48: threshold for block 11
	AttrPriceBlock12Threshold AttributeID = 0x010B // uint48: threshold for block 12
	AttrPriceBlock13Threshold AttributeID = 0x010C // uint48: threshold for block 13
	AttrPriceBlock14Threshold AttributeID = 0x010D // uint48: threshold for block 14
	AttrPriceBlock15Threshold AttributeID = 0x010E // uint48: threshold for block 15
	AttrPriceBlock16Threshold AttributeID = 0x010F // uint48: threshold for block 16

	// Block Period Attributes
	AttrPriceStartOfBlockPeriod      AttributeID = 0x0200 // uint32: UTC time of block period start
	AttrPriceBlockPeriodDuration     AttributeID = 0x0201 // uint24: duration in minutes
	AttrPriceThresholdMultiplier     AttributeID = 0x0202 // uint24: multiplier for thresholds
	AttrPriceThresholdDivisor        AttributeID = 0x0203 // uint24: divisor for thresholds
	AttrPriceBlockPeriodDurationType AttributeID = 0x0204 // enum8: duration type

	// Current Tier Summation Attributes (most commonly used)
	AttrPriceCurrentTier1SummationDelivered AttributeID = 0x0100 // uint48: tier 1 delivered
	AttrPriceCurrentTier2SummationDelivered AttributeID = 0x0102 // uint48: tier 2 delivered
	AttrPriceCurrentTier3SummationDelivered AttributeID = 0x0104 // uint48: tier 3 delivered
	AttrPriceCurrentTier4SummationDelivered AttributeID = 0x0106 // uint48: tier 4 delivered
	AttrPriceCurrentTier5SummationDelivered AttributeID = 0x0108 // uint48: tier 5 delivered
	AttrPriceCurrentTier6SummationDelivered AttributeID = 0x010A // uint48: tier 6 delivered
)

// Price Cluster Commands - Server to Client (0x0700)
const (
	CmdPricePublishPrice       uint8 = 0x00 // Publish price information
	CmdPricePublishBlockPeriod uint8 = 0x01 // Publish block period info
)

// Price Cluster Commands - Client to Server (0x0700)
const (
	CmdPriceGetCurrentPrice    uint8 = 0x00 // Request current price
	CmdPriceGetScheduledPrices uint8 = 0x01 // Request scheduled prices
)

// OTA Upgrade Cluster Attributes (0x0019) - Server side
const (
	AttrOTAUpgradeServerID              AttributeID = 0x0000 // IEEE address of OTA server
	AttrOTAFileOffset                   AttributeID = 0x0001 // uint32: current file offset
	AttrOTACurrentFileVersion           AttributeID = 0x0002 // uint32: current firmware version
	AttrOTACurrentZigBeeStackVersion    AttributeID = 0x0003 // uint16: current stack version
	AttrOTADownloadedFileVersion        AttributeID = 0x0004 // uint32: downloaded file version
	AttrOTADownloadedZigBeeStackVersion AttributeID = 0x0005 // uint16: downloaded stack version
	AttrOTAImageUpgradeStatus           AttributeID = 0x0006 // enum8: upgrade status
	AttrOTAManufacturerID               AttributeID = 0x0007 // uint16: manufacturer ID
	AttrOTAImageTypeID                  AttributeID = 0x0008 // uint16: image type ID
	AttrOTAMinBlockRequestDelay         AttributeID = 0x0009 // uint16: min delay between blocks (ms)
	AttrOTAImageStamp                   AttributeID = 0x000A // uint32: image stamp
)

// OTAUpgradeStatus represents the current status of an OTA upgrade.
type OTAUpgradeStatus uint8

const (
	OTAStatusNormal             OTAUpgradeStatus = 0x00 // Normal operation
	OTAStatusDownloadInProgress OTAUpgradeStatus = 0x01 // Download in progress
	OTAStatusDownloadComplete   OTAUpgradeStatus = 0x02 // Download complete
	OTAStatusWaitingToUpgrade   OTAUpgradeStatus = 0x03 // Waiting to upgrade
	OTAStatusCountDown          OTAUpgradeStatus = 0x04 // Count down to upgrade
	OTAStatusWaitForMore        OTAUpgradeStatus = 0x05 // Wait for more data
)

// String returns human-readable OTA upgrade status name.
func (o OTAUpgradeStatus) String() string {
	switch o {
	case OTAStatusNormal:
		return "Normal"
	case OTAStatusDownloadInProgress:
		return "DownloadInProgress"
	case OTAStatusDownloadComplete:
		return "DownloadComplete"
	case OTAStatusWaitingToUpgrade:
		return "WaitingToUpgrade"
	case OTAStatusCountDown:
		return "CountDown"
	case OTAStatusWaitForMore:
		return "WaitForMore"
	default:
		return fmt.Sprintf("OTAUpgradeStatus(0x%02X)", uint8(o))
	}
}

// OTA Upgrade Cluster Commands (0x0019) - Client to Server
const (
	CmdOTAQueryNextImageRequest uint8 = 0x01 // Request info about available upgrade
	CmdOTAImageBlockRequest     uint8 = 0x03 // Request a block of image data
	CmdOTAImagePageRequest      uint8 = 0x04 // Request a page of image data
	CmdOTAUpgradeEndRequest     uint8 = 0x06 // Notify server of upgrade completion
)

// OTA Upgrade Cluster Commands (0x0019) - Server to Client
const (
	CmdOTAImageNotify            uint8 = 0x00 // Notify client of available upgrade
	CmdOTAQueryNextImageResponse uint8 = 0x02 // Response to query
	CmdOTAImageBlockResponse     uint8 = 0x05 // Response with image block
	CmdOTAUpgradeEndResponse     uint8 = 0x07 // Response to upgrade end
)

// Tuya-specific cluster IDs
const (
	ClusterTuyaSpecific ClusterID = 0xE001 // Tuya manufacturer-specific cluster
	ClusterTuyaEF00     ClusterID = 0xEF00 // Tuya MCU cluster
)

// Tuya-specific attributes for energy reset (cluster 0xE001)
const (
	AttrTuyaEnergyReset AttributeID = 0xD004 // Some Tuya plugs use this to reset energy
)

// PowerSource values for Basic cluster.
type PowerSource uint8

const (
	PowerSourceUnknown        PowerSource = 0x00
	PowerSourceMains          PowerSource = 0x01
	PowerSourceBattery        PowerSource = 0x03
	PowerSourceDC             PowerSource = 0x04
	PowerSourceEmergencyMains PowerSource = 0x05
	PowerSourceEmergencyDC    PowerSource = 0x06
)

// String returns human-readable power source name.
func (p PowerSource) String() string {
	switch p {
	case PowerSourceUnknown:
		return "Unknown"
	case PowerSourceMains:
		return "MainsSinglePhase"
	case PowerSourceBattery:
		return "Battery"
	case PowerSourceDC:
		return "DCSource"
	case PowerSourceEmergencyMains:
		return "EmergencyMainsAlwaysOn"
	case PowerSourceEmergencyDC:
		return "EmergencyMainsTransferSwitch"
	default:
		return fmt.Sprintf("PowerSource(0x%02X)", uint8(p))
	}
}

// BatterySize represents the size/type of battery installed.
type BatterySize uint8

const (
	BatterySizeNoBattery BatterySize = 0x00 // No battery
	BatterySizeBuiltIn   BatterySize = 0x01 // Built-in, non-replaceable
	BatterySizeOther     BatterySize = 0x02 // Other
	BatterySizeAA        BatterySize = 0x03 // AA battery
	BatterySizeAAA       BatterySize = 0x04 // AAA battery
	BatterySizeC         BatterySize = 0x05 // C battery
	BatterySizeD         BatterySize = 0x06 // D battery
	BatterySizeCR2       BatterySize = 0x07 // CR2 battery
	BatterySizeCR123A    BatterySize = 0x08 // CR123A battery
	BatterySizeUnknown   BatterySize = 0xFF // Unknown
)

// String returns human-readable battery size name.
func (b BatterySize) String() string {
	switch b {
	case BatterySizeNoBattery:
		return "NoBattery"
	case BatterySizeBuiltIn:
		return "BuiltIn"
	case BatterySizeOther:
		return "Other"
	case BatterySizeAA:
		return "AA"
	case BatterySizeAAA:
		return "AAA"
	case BatterySizeC:
		return "C"
	case BatterySizeD:
		return "D"
	case BatterySizeCR2:
		return "CR2"
	case BatterySizeCR123A:
		return "CR123A"
	case BatterySizeUnknown:
		return "Unknown"
	default:
		return fmt.Sprintf("BatterySize(0x%02X)", uint8(b))
	}
}

// StartUpOnOff defines the startup behavior for OnOff cluster devices.
type StartUpOnOff uint8

const (
	StartUpOnOffOff      StartUpOnOff = 0x00 // Turn off when powered on
	StartUpOnOffOn       StartUpOnOff = 0x01 // Turn on when powered on
	StartUpOnOffToggle   StartUpOnOff = 0x02 // Toggle state when powered on
	StartUpOnOffPrevious StartUpOnOff = 0xFF // Restore previous state when powered on
)

// String returns human-readable startup behavior name.
func (s StartUpOnOff) String() string {
	switch s {
	case StartUpOnOffOff:
		return "Off"
	case StartUpOnOffOn:
		return "On"
	case StartUpOnOffToggle:
		return "Toggle"
	case StartUpOnOffPrevious:
		return "Previous"
	default:
		return fmt.Sprintf("StartUpOnOff(0x%02X)", uint8(s))
	}
}

// WindowCoveringType defines the type of window covering device.
type WindowCoveringType uint8

const (
	WindowCoveringTypeRollerShade               WindowCoveringType = 0x00
	WindowCoveringTypeRollerShade2Motor         WindowCoveringType = 0x01
	WindowCoveringTypeRollerShadeExterior       WindowCoveringType = 0x02
	WindowCoveringTypeRollerShadeExterior2Motor WindowCoveringType = 0x03
	WindowCoveringTypeDrapery                   WindowCoveringType = 0x04
	WindowCoveringTypeAwning                    WindowCoveringType = 0x05
	WindowCoveringTypeShutter                   WindowCoveringType = 0x06
	WindowCoveringTypeTiltBlindTiltOnly         WindowCoveringType = 0x07
	WindowCoveringTypeTiltBlindLiftAndTilt      WindowCoveringType = 0x08
	WindowCoveringTypeProjectorScreen           WindowCoveringType = 0x09
	WindowCoveringTypeUnknown                   WindowCoveringType = 0xFF
)

// String returns human-readable window covering type name.
func (w WindowCoveringType) String() string {
	switch w {
	case WindowCoveringTypeRollerShade:
		return "RollerShade"
	case WindowCoveringTypeRollerShade2Motor:
		return "RollerShade2Motor"
	case WindowCoveringTypeRollerShadeExterior:
		return "RollerShadeExterior"
	case WindowCoveringTypeRollerShadeExterior2Motor:
		return "RollerShadeExterior2Motor"
	case WindowCoveringTypeDrapery:
		return "Drapery"
	case WindowCoveringTypeAwning:
		return "Awning"
	case WindowCoveringTypeShutter:
		return "Shutter"
	case WindowCoveringTypeTiltBlindTiltOnly:
		return "TiltBlindTiltOnly"
	case WindowCoveringTypeTiltBlindLiftAndTilt:
		return "TiltBlindLiftAndTilt"
	case WindowCoveringTypeProjectorScreen:
		return "ProjectorScreen"
	case WindowCoveringTypeUnknown:
		return "Unknown"
	default:
		return fmt.Sprintf("WindowCoveringType(0x%02X)", uint8(w))
	}
}

// Messaging Cluster (0x0703) - Smart Energy
// This cluster provides an interface for passing text messages between devices.
// The cluster is command-based with no mandatory attributes.

// MessageControl bitmap for Smart Energy Messaging cluster.
const (
	SEMsgCtrlTransmissionNormal    uint8 = 0x00      // Normal transmission
	SEMsgCtrlTransmissionAnonymous uint8 = 0x01 << 0 // Anonymous/inter-PAN transmission
	SEMsgCtrlImportanceLow         uint8 = 0x00      // Low importance
	SEMsgCtrlImportanceMedium      uint8 = 0x01 << 2 // Medium importance
	SEMsgCtrlImportanceHigh        uint8 = 0x02 << 2 // High importance
	SEMsgCtrlImportanceCritical    uint8 = 0x03 << 2 // Critical importance
	SEMsgCtrlConfirmationRequired  uint8 = 0x01 << 7 // Confirmation required
)

// Messaging Cluster Commands - Server to Client (0x0703)
const (
	CmdSEMessagingDisplayMessage          uint8 = 0x00 // Display a message on device
	CmdSEMessagingCancelMessage           uint8 = 0x01 // Cancel a displayed message
	CmdSEMessagingDisplayProtectedMessage uint8 = 0x02 // Display protected message
	CmdSEMessagingCancelAllMessages       uint8 = 0x03 // Cancel all messages
)

// Messaging Cluster Commands - Client to Server (0x0703)
const (
	CmdSEMessagingGetLastMessage      uint8 = 0x00 // Request last message
	CmdSEMessagingMessageConfirmation uint8 = 0x01 // Confirm message receipt
)

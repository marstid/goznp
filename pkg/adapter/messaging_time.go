package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
)

// Time Cluster (0x000A)

// Zigbee epoch is January 1, 2000 00:00:00 UTC (946684800 Unix seconds).
const zigbeeEpoch = 946684800

// ZigbeeTimeToUnix converts Zigbee time (seconds since 2000-01-01) to Unix time.
// Zigbee time is uint32 seconds since January 1, 2000 00:00:00 UTC.
// Unix time is int64 seconds since January 1, 1970 00:00:00 UTC.
func ZigbeeTimeToUnix(zigbeeTime uint32) time.Time {
	unixSeconds := int64(zigbeeTime) + zigbeeEpoch
	return time.Unix(unixSeconds, 0)
}

// UnixToZigbeeTime converts Unix time to Zigbee time (seconds since 2000-01-01).
// Returns uint32 seconds since January 1, 2000 00:00:00 UTC.
func UnixToZigbeeTime(t time.Time) uint32 {
	unixSeconds := t.Unix()
	if unixSeconds < zigbeeEpoch {
		return 0 // Time before Zigbee epoch
	}
	return uint32(unixSeconds - zigbeeEpoch)
}

// TimeInfo contains time information from a Time cluster device.
type TimeInfo struct {
	Time       uint32 // UTC time since 2000-01-01 (Zigbee epoch)
	TimeStatus uint8  // Status bitmap
	TimeZone   int32  // Timezone offset in seconds
	LocalTime  uint32 // Local time
}

// GetTime reads the current time and status from a Time cluster device.
// Returns time information including UTC time, status flags, timezone offset, and local time.
// Use ZigbeeTimeToUnix() to convert the Time field to a standard time.Time value.
func (a *Adapter) GetTime(ctx context.Context, nwkAddr uint16, endpoint uint8) (*TimeInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrTimeTime,
		zcl.AttrTimeStatus,
		zcl.AttrTimeZone,
		zcl.AttrTimeLocalTime,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read time attributes: %w", err)
	}

	info := &TimeInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrTimeTime:
			if val, ok := toUint32(r.Value); ok {
				info.Time = val
			}
		case zcl.AttrTimeStatus:
			if val, ok := r.Value.(uint8); ok {
				info.TimeStatus = val
			}
		case zcl.AttrTimeZone:
			if val, ok := toInt32(r.Value); ok {
				info.TimeZone = val
			}
		case zcl.AttrTimeLocalTime:
			if val, ok := toUint32(r.Value); ok {
				info.LocalTime = val
			}
		}
	}

	return info, nil
}

// SetTime sets the UTC time on a Time cluster device.
// zigbeeTime is seconds since January 1, 2000 00:00:00 UTC.
// Use UnixToZigbeeTime() to convert from time.Time to Zigbee time.
//
// Example: Set device to current time:
//
//	zigbeeTime := adapter.UnixToZigbeeTime(time.Now())
//	err := adapter.SetTime(ctx, nwkAddr, endpoint, zigbeeTime)
func (a *Adapter) SetTime(ctx context.Context, nwkAddr uint16, endpoint uint8, zigbeeTime uint32) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTimeTime: {
			Type:  zcl.TypeUint32,
			Value: zigbeeTime,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, values)
}

// GetTimeZone reads the timezone offset from a Time cluster device.
// Returns the timezone offset in seconds from UTC.
// For example: UTC+2 = 7200 seconds, UTC-5 = -18000 seconds.
func (a *Adapter) GetTimeZone(ctx context.Context, nwkAddr uint16, endpoint uint8) (int32, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, zcl.AttrTimeZone)
	if err != nil {
		return 0, fmt.Errorf("failed to read timezone: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read timezone returned status 0x%02X", result.Status)
	}

	if offset, ok := toInt32(result.Value); ok {
		return offset, nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// SetTimeZone sets the timezone offset on a Time cluster device.
// offsetSeconds is the timezone offset in seconds from UTC.
// For example: UTC+2 = 7200 seconds, UTC-5 = -18000 seconds.
//
// Example: Set timezone to UTC+1 (Central European Time):
//
//	err := adapter.SetTimeZone(ctx, nwkAddr, endpoint, 3600)
func (a *Adapter) SetTimeZone(ctx context.Context, nwkAddr uint16, endpoint uint8, offsetSeconds int32) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrTimeZone: {
			Type:  zcl.TypeInt32,
			Value: offsetSeconds,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTime, values)
}

// DRLC (Demand Response and Load Control) Cluster (0x0701)

// DRLCEventInfo contains information about a demand response load control event.
// This represents a request from the utility to reduce or shift energy consumption.
type DRLCEventInfo struct {
	IssuerEventID    uint32 // Unique event ID from utility
	DeviceClass      uint16 // Device class bitmap (which devices should respond)
	UtilityGroupID   uint8  // Utility enrolment group
	StartTime        uint32 // Event start time (Zigbee time, seconds since 2000-01-01)
	Duration         uint16 // Event duration in minutes
	CriticalityLevel uint8  // Criticality level (0-15, higher = more critical)
	CoolingOffset    uint8  // Temperature offset for cooling (°C)
	HeatingOffset    uint8  // Temperature offset for heating (°C)
}

// GetDRLCInfo reads the current DRLC configuration from a device.
// This retrieves the device's demand response settings including which utility group
// it belongs to, the device class, and randomization settings.
//
// The device class bitmap indicates which types of loads this device can control.
// Use the zcl.DRLCDeviceClass* constants to check which classes are supported.
//
// Returns the utility enrolment group, device class, and randomization settings.
func (a *Adapter) GetDRLCInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DRLCEventInfo, error) {
	// Read DRLC attributes
	attrs := []zcl.AttributeID{
		zcl.AttrDRLCUtilityEnrolmentGroup,
		zcl.AttrDRLCStartRandomizeMinutes,
		zcl.AttrDRLCDurationRandomizeMinutes,
		zcl.AttrDRLCDeviceClassValue,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterDRLC, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read DRLC attributes: %w", err)
	}

	info := &DRLCEventInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrDRLCUtilityEnrolmentGroup:
			if v, ok := r.Value.(uint8); ok {
				info.UtilityGroupID = v
			}
		case zcl.AttrDRLCDeviceClassValue:
			if v, ok := toUint16(r.Value); ok {
				info.DeviceClass = v
			}
		}
	}

	return info, nil
}

// OTA Upgrade Cluster (0x0019)

// OTAInfo contains OTA upgrade status information.
type OTAInfo struct {
	CurrentFileVersion uint32               // Current firmware version
	UpgradeStatus      zcl.OTAUpgradeStatus // Current upgrade status
	ManufacturerID     uint16               // Manufacturer ID
	ImageTypeID        uint16               // Image type ID
	FileOffset         uint32               // Current file offset
}

// GetOTAInfo reads OTA upgrade status information from a device.
// Returns current firmware version, upgrade status, manufacturer/image IDs, and file offset.
// This provides a comprehensive view of the device's OTA upgrade state.
func (a *Adapter) GetOTAInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*OTAInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrOTACurrentFileVersion,
		zcl.AttrOTAImageUpgradeStatus,
		zcl.AttrOTAManufacturerID,
		zcl.AttrOTAImageTypeID,
		zcl.AttrOTAFileOffset,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOTAUpgrade, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read OTA attributes: %w", err)
	}

	info := &OTAInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrOTACurrentFileVersion:
			if val, ok := toUint32(r.Value); ok {
				info.CurrentFileVersion = val
			}
		case zcl.AttrOTAImageUpgradeStatus:
			if val, ok := r.Value.(uint8); ok {
				info.UpgradeStatus = zcl.OTAUpgradeStatus(val)
			}
		case zcl.AttrOTAManufacturerID:
			if val, ok := toUint16(r.Value); ok {
				info.ManufacturerID = val
			}
		case zcl.AttrOTAImageTypeID:
			if val, ok := toUint16(r.Value); ok {
				info.ImageTypeID = val
			}
		case zcl.AttrOTAFileOffset:
			if val, ok := toUint32(r.Value); ok {
				info.FileOffset = val
			}
		}
	}

	return info, nil
}

// GetOTAUpgradeStatus reads the current OTA upgrade status from a device.
// Returns the upgrade status enum indicating the device's current state
// (Normal, DownloadInProgress, DownloadComplete, etc.).
func (a *Adapter) GetOTAUpgradeStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (zcl.OTAUpgradeStatus, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOTAUpgrade, zcl.AttrOTAImageUpgradeStatus)
	if err != nil {
		return 0, fmt.Errorf("failed to read OTA upgrade status: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return 0, fmt.Errorf("read OTA upgrade status returned status 0x%02X", result.Status)
	}

	if status, ok := result.Value.(uint8); ok {
		return zcl.OTAUpgradeStatus(status), nil
	}
	return 0, fmt.Errorf("unexpected value type: %T", result.Value)
}

// PollControl Cluster (0x0020)

// PollControlInfo contains poll control configuration for a sleepy end device.
type PollControlInfo struct {
	CheckInInterval   uint32 // In quarter seconds
	LongPollInterval  uint32 // In quarter seconds
	ShortPollInterval uint16 // In quarter seconds
	FastPollTimeout   uint16 // In quarter seconds
}

// GetPollControlInfo reads poll control configuration from a device.
// Returns the check-in interval, long poll interval, short poll interval,
// and fast poll timeout. All intervals are in quarter seconds.
//
// To convert to seconds, use QuarterSecondsToSeconds():
//
//	info, err := adapter.GetPollControlInfo(ctx, nwkAddr, endpoint)
//	if err != nil {
//		return err
//	}
//	checkInSecs := QuarterSecondsToSeconds(info.CheckInInterval)
func (a *Adapter) GetPollControlInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*PollControlInfo, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrPollControlCheckInInterval,
		zcl.AttrPollControlLongPollInterval,
		zcl.AttrPollControlShortPollInterval,
		zcl.AttrPollControlFastPollTimeout,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read poll control attributes: %w", err)
	}

	info := &PollControlInfo{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrPollControlCheckInInterval:
			if val, ok := toUint32(r.Value); ok {
				info.CheckInInterval = val
			}
		case zcl.AttrPollControlLongPollInterval:
			if val, ok := toUint32(r.Value); ok {
				info.LongPollInterval = val
			}
		case zcl.AttrPollControlShortPollInterval:
			if val, ok := toUint16(r.Value); ok {
				info.ShortPollInterval = val
			}
		case zcl.AttrPollControlFastPollTimeout:
			if val, ok := toUint16(r.Value); ok {
				info.FastPollTimeout = val
			}
		}
	}

	return info, nil
}

// SendCheckInResponse responds to a check-in notification from a sleepy device.
// This tells the device whether to enter fast polling mode and for how long.
//
// Parameters:
//   - startFastPolling: Whether the device should enter fast polling mode
//   - fastPollTimeout: Duration of fast polling in quarter seconds (0 = use device default)
//
// The device will poll more frequently during fast polling, allowing the coordinator
// to send pending commands. After the timeout, the device returns to normal polling.
//
// Example: Enable fast polling for 10 seconds (40 quarter seconds):
//
//	err := adapter.SendCheckInResponse(ctx, nwkAddr, endpoint, true, 40)
func (a *Adapter) SendCheckInResponse(ctx context.Context, nwkAddr uint16, endpoint uint8, startFastPolling bool, fastPollTimeout uint16) error {
	// CheckInResponse payload: startFastPolling (1 byte bool) + fastPollTimeout (2 bytes LE)
	var startFastPoll uint8
	if startFastPolling {
		startFastPoll = 0x01
	} else {
		startFastPoll = 0x00
	}

	payload := []byte{
		startFastPoll,
		byte(fastPollTimeout & 0xFF),
		byte(fastPollTimeout >> 8),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlCheckInResponse, payload)
}

// StopFastPolling instructs a sleepy device to stop fast polling immediately.
// The device will return to its normal long poll interval.
//
// This is useful to conserve battery when you no longer have pending commands
// for the device.
func (a *Adapter) StopFastPolling(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlFastPollStop, nil)
}

// SetLongPollInterval sets the long poll interval on a sleepy device.
// The long poll interval is how often the device wakes up to check for messages
// when it's in normal (non-fast-polling) mode.
//
// intervalQuarterSecs is the interval in quarter seconds (4 = 1 second).
//
// Example: Set long poll interval to 5 minutes (1200 quarter seconds):
//
//	err := adapter.SetLongPollInterval(ctx, nwkAddr, endpoint, 1200)
func (a *Adapter) SetLongPollInterval(ctx context.Context, nwkAddr uint16, endpoint uint8, intervalQuarterSecs uint32) error {
	// SetLongPollInterval payload: newLongPollInterval (4 bytes LE)
	payload := []byte{
		byte(intervalQuarterSecs & 0xFF),
		byte((intervalQuarterSecs >> 8) & 0xFF),
		byte((intervalQuarterSecs >> 16) & 0xFF),
		byte((intervalQuarterSecs >> 24) & 0xFF),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlSetLongPollInterval, payload)
}

// SetShortPollInterval sets the short poll interval (fast poll interval) on a sleepy device.
// The short poll interval is how often the device polls when in fast polling mode.
//
// intervalQuarterSecs is the interval in quarter seconds (4 = 1 second).
//
// Example: Set short poll interval to 250ms (1 quarter second):
//
//	err := adapter.SetShortPollInterval(ctx, nwkAddr, endpoint, 1)
func (a *Adapter) SetShortPollInterval(ctx context.Context, nwkAddr uint16, endpoint uint8, intervalQuarterSecs uint16) error {
	// SetShortPollInterval payload: newShortPollInterval (2 bytes LE)
	payload := []byte{
		byte(intervalQuarterSecs & 0xFF),
		byte(intervalQuarterSecs >> 8),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterPollControl, zcl.CmdPollControlSetShortPollInterval, payload)
}

// QuarterSecondsToSeconds converts quarter seconds to seconds.
// Quarter seconds are commonly used in Zigbee for timing intervals.
// 4 quarter seconds = 1 second.
func QuarterSecondsToSeconds(quarterSecs uint32) float64 {
	return float64(quarterSecs) / 4.0
}

// SecondsToQuarterSeconds converts seconds to quarter seconds.
// Quarter seconds are commonly used in Zigbee for timing intervals.
// 1 second = 4 quarter seconds.
func SecondsToQuarterSeconds(secs float64) uint32 {
	return uint32(secs * 4.0)
}

package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
)

// IASZone Cluster (0x0500) - Security Sensors

// IASZoneStatus contains current zone sensor state.
type IASZoneStatus struct {
	ZoneState  uint8           // 0=not enrolled, 1=enrolled
	ZoneType   zcl.IASZoneType // Type of sensor
	ZoneStatus uint16          // Bitmap of current alarms and states
	ZoneID     *uint8          // Zone ID (0-254), nil if not available
	CIEAddress *[8]byte        // IEEE address of CIE, nil if not available

	// Parsed status flags for convenience
	Alarm1        bool // Alarm 1 active (e.g., motion detected, door open)
	Alarm2        bool // Alarm 2 active (secondary alarm)
	Tamper        bool // Tamper detected
	LowBattery    bool // Low battery warning
	Trouble       bool // Sensor trouble/failure
	ACMains       bool // AC mains fault
	Test          bool // Sensor in test mode
	BatteryDefect bool // Battery defect detected
}

// IASZoneNotification represents a zone status change notification from a device.
// These are sent automatically by enrolled IAS Zone devices when their status changes
// (e.g., motion detected, door opened, tamper alert).
type IASZoneNotification struct {
	SrcAddr    uint16 // Source device network address
	Endpoint   uint8  // Source endpoint
	ZoneStatus uint16 // Status bitmap
	ExtStatus  uint8  // Extended status
	ZoneID     uint8  // Zone ID
	Delay      uint16 // Delay in quarter-seconds

	// Parsed status flags for convenience
	Alarm1        bool // Alarm 1 active
	Alarm2        bool // Alarm 2 active
	Tamper        bool // Tamper detected
	LowBattery    bool // Low battery
	Trouble       bool // Trouble/failure
	ACMains       bool // AC mains fault
	Test          bool // Test mode
	BatteryDefect bool // Battery defect
}

// GetIASZoneStatus reads the current status from an IAS Zone device.
// This queries all relevant attributes and parses the zone status bitmap.
func (a *Adapter) GetIASZoneStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*IASZoneStatus, error) {
	// Read IAS Zone attributes
	attrs := []zcl.AttributeID{
		zcl.AttrIASZoneState,
		zcl.AttrIASZoneType,
		zcl.AttrIASZoneStatus,
		zcl.AttrIASZoneID,
		zcl.AttrIASZoneCIEAddr,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read IAS Zone attributes: %w", err)
	}

	status := &IASZoneStatus{}

	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrIASZoneState:
			if v, ok := r.Value.(uint8); ok {
				status.ZoneState = v
			}
		case zcl.AttrIASZoneType:
			if v, ok := toUint16(r.Value); ok {
				status.ZoneType = zcl.IASZoneType(v)
			}
		case zcl.AttrIASZoneStatus:
			if v, ok := toUint16(r.Value); ok {
				status.ZoneStatus = v
				// Parse status bitmap
				status.Alarm1 = (v & zcl.IASZoneStatusAlarm1) != 0
				status.Alarm2 = (v & zcl.IASZoneStatusAlarm2) != 0
				status.Tamper = (v & zcl.IASZoneStatusTamper) != 0
				status.LowBattery = (v & zcl.IASZoneStatusBattery) != 0
				status.Trouble = (v & zcl.IASZoneStatusTrouble) != 0
				status.ACMains = (v & zcl.IASZoneStatusACMains) != 0
				status.Test = (v & zcl.IASZoneStatusTest) != 0
				status.BatteryDefect = (v & zcl.IASZoneStatusBatteryDefect) != 0
			}
		case zcl.AttrIASZoneID:
			if v, ok := r.Value.(uint8); ok {
				status.ZoneID = &v
			}
		case zcl.AttrIASZoneCIEAddr:
			// IEEE address is 8 bytes
			if v, ok := r.Value.([8]byte); ok {
				status.CIEAddress = &v
			}
		}
	}

	return status, nil
}

// EnrollIASZone sends an enroll response to a zone device.
// This must be called after receiving an enroll request from the device.
//
// Parameters:
//   - responseCode: Enrollment response code
//     0x00 = Success
//     0x01 = Not supported
//     0x02 = No enroll permit
//     0x03 = Too many zones
//   - zoneID: Assigned zone ID (0-254)
//
// The device will not send zone status change notifications until it is enrolled.
func (a *Adapter) EnrollIASZone(ctx context.Context, nwkAddr uint16, endpoint uint8, responseCode uint8, zoneID uint8) error {
	// EnrollResponse payload: responseCode (1 byte) + zoneID (1 byte)
	payload := []byte{responseCode, zoneID}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, zcl.CmdIASZoneEnrollResponse, payload)
}

// WriteIASZoneCIEAddress writes the coordinator's IEEE address to the zone device.
// This is required before a device can be enrolled. The device needs to know
// which coordinator (CIE - Control and Indicating Equipment) to send notifications to.
//
// The cieAddress should be the coordinator's IEEE address, which can be obtained
// from the network configuration or device information.
func (a *Adapter) WriteIASZoneCIEAddress(ctx context.Context, nwkAddr uint16, endpoint uint8, cieAddress [8]byte) error {
	values := map[zcl.AttributeID]zcl.AttributeValue{
		zcl.AttrIASZoneCIEAddr: {
			Type:  zcl.TypeIEEEAddr,
			Value: cieAddress,
		},
	}
	return a.WriteAttributes(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, values)
}

// WaitForIASZoneNotification waits for an incoming IAS Zone status change notification.
// Returns when a zone notification is received or timeout occurs.
//
// Note: The device must be enrolled before it will send notifications.
// Use EnrollIASZone to enroll the device first.
func (a *Adapter) WaitForIASZoneNotification(ctx context.Context, timeout time.Duration) (*IASZoneNotification, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	dedupe := a.dedupe
	a.mu.Unlock()

	deadline := time.Now().Add(timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("wait for IAS Zone notification: timeout")
		}

		// Wait for incoming message on IASZone cluster
		msg, err := znpClient.WaitForIncomingMsg(ctx, 0, uint16(zcl.ClusterIASZone), remaining)
		if err != nil {
			return nil, err
		}

		// Parse ZCL frame
		frame, err := zcl.ParseFrame(msg.Data)
		if err != nil {
			continue
		}

		// Check for duplicates
		if dedupe != nil && dedupe.isDuplicate(msg.SrcAddr, msg.ClusterID, frame.TransSeqNum) {
			continue
		}

		// Check if this is a status change notification
		if frame.CommandID != zcl.CmdIASZoneStatusChangeNotification {
			continue
		}

		// Parse notification payload
		// Payload format: zoneStatus (2 bytes LE) + extendedStatus (1 byte) + zoneID (1 byte) + delay (2 bytes LE)
		if len(frame.Payload) < 6 {
			continue
		}

		notification := &IASZoneNotification{
			SrcAddr:    msg.SrcAddr,
			Endpoint:   msg.SrcEndpoint,
			ZoneStatus: uint16(frame.Payload[0]) | uint16(frame.Payload[1])<<8,
			ExtStatus:  frame.Payload[2],
			ZoneID:     frame.Payload[3],
			Delay:      uint16(frame.Payload[4]) | uint16(frame.Payload[5])<<8,
		}

		// Parse status flags
		notification.Alarm1 = (notification.ZoneStatus & zcl.IASZoneStatusAlarm1) != 0
		notification.Alarm2 = (notification.ZoneStatus & zcl.IASZoneStatusAlarm2) != 0
		notification.Tamper = (notification.ZoneStatus & zcl.IASZoneStatusTamper) != 0
		notification.LowBattery = (notification.ZoneStatus & zcl.IASZoneStatusBattery) != 0
		notification.Trouble = (notification.ZoneStatus & zcl.IASZoneStatusTrouble) != 0
		notification.ACMains = (notification.ZoneStatus & zcl.IASZoneStatusACMains) != 0
		notification.Test = (notification.ZoneStatus & zcl.IASZoneStatusTest) != 0
		notification.BatteryDefect = (notification.ZoneStatus & zcl.IASZoneStatusBatteryDefect) != 0

		return notification, nil
	}
}

// DoorLock Cluster (0x0101)

// DoorLockStatus contains current lock state.
type DoorLockStatus struct {
	LockState       zcl.DoorLockState
	DoorState       *zcl.DoorState
	ActuatorEnabled bool
	AutoRelockTime  *uint32
}

// GetDoorLockStatus reads the current door lock status from a device.
// Returns the lock state, door state (if available), actuator status,
// and auto-relock time (if configured).
func (a *Adapter) GetDoorLockStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DoorLockStatus, error) {
	attrs := []zcl.AttributeID{
		zcl.AttrDoorLockState,
		zcl.AttrDoorLockDoorState,
		zcl.AttrDoorLockActuatorEnabled,
		zcl.AttrDoorLockAutoRelockTime,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read door lock attributes: %w", err)
	}

	status := &DoorLockStatus{}
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue
		}

		switch r.AttributeID {
		case zcl.AttrDoorLockState:
			if v, ok := r.Value.(uint8); ok {
				status.LockState = zcl.DoorLockState(v)
			}
		case zcl.AttrDoorLockDoorState:
			if v, ok := r.Value.(uint8); ok {
				doorState := zcl.DoorState(v)
				status.DoorState = &doorState
			}
		case zcl.AttrDoorLockActuatorEnabled:
			if v, ok := r.Value.(bool); ok {
				status.ActuatorEnabled = v
			} else if v, ok := r.Value.(uint8); ok {
				status.ActuatorEnabled = v != 0
			}
		case zcl.AttrDoorLockAutoRelockTime:
			if v, ok := toUint32(r.Value); ok {
				status.AutoRelockTime = &v
			}
		}
	}

	return status, nil
}

// LockDoor sends a lock command to the door lock device.
func (a *Adapter) LockDoor(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockLock, nil)
}

// UnlockDoor sends an unlock command to the door lock device.
func (a *Adapter) UnlockDoor(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockUnlock, nil)
}

// UnlockDoorWithTimeout unlocks the door and automatically relocks after the specified timeout.
// timeoutSeconds is the number of seconds before the door automatically relocks.
func (a *Adapter) UnlockDoorWithTimeout(ctx context.Context, nwkAddr uint16, endpoint uint8, timeoutSeconds uint16) error {
	// UnlockWithTimeout payload: timeout (2 bytes LE)
	payload := []byte{
		byte(timeoutSeconds & 0xFF),
		byte(timeoutSeconds >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockUnlockWithTimeout, payload)
}

// ToggleDoorLock toggles the lock state (locked <-> unlocked).
func (a *Adapter) ToggleDoorLock(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockToggle, nil)
}

// SetDoorLockPIN sets a PIN code for a user.
// userID is the user identifier (0-65535).
// pin is the PIN code string.
// The userStatus and userType are set to default values (enabled user with unrestricted access).
func (a *Adapter) SetDoorLockPIN(ctx context.Context, nwkAddr uint16, endpoint uint8, userID uint16, pin string) error {
	// SetPINCode payload: userID (2 bytes LE) + userStatus (1 byte) + userType (1 byte) + pin (string with length prefix)
	pinBytes := []byte(pin)
	payload := make([]byte, 5+len(pinBytes))
	payload[0] = byte(userID & 0xFF)
	payload[1] = byte(userID >> 8)
	payload[2] = 0x01 // userStatus: enabled (0x01)
	payload[3] = 0x00 // userType: unrestricted (0x00)
	payload[4] = byte(len(pinBytes))
	copy(payload[5:], pinBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockSetPINCode, payload)
}

// ClearDoorLockPIN removes a user's PIN code.
// userID is the user identifier to clear.
func (a *Adapter) ClearDoorLockPIN(ctx context.Context, nwkAddr uint16, endpoint uint8, userID uint16) error {
	// ClearPINCode payload: userID (2 bytes LE)
	payload := []byte{
		byte(userID & 0xFF),
		byte(userID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterDoorLock, zcl.CmdDoorLockClearPINCode, payload)
}

// IAS Warning Device Cluster (0x0502)

// StartWarning activates the warning device (siren/strobe).
// This sends a command to the device to start sounding an alarm with the specified parameters.
//
// Parameters:
//   - mode: Type of warning (Burglar, Fire, Emergency, etc.)
//   - useStrobe: Whether to activate the strobe light
//   - sirenLevel: Loudness of the siren (Low, Medium, High, VeryHigh)
//   - durationSecs: How long to sound the alarm (max depends on device, typically 240 seconds)
//   - strobeDutyCycle: Duty cycle percentage for strobe (0-100), where 100 = always on
//   - strobeLevel: Brightness of the strobe (Low, Medium, High, VeryHigh)
//
// Example: Sound burglar alarm for 30 seconds with high siren and medium strobe:
//
//	err := adapter.StartWarning(ctx, nwkAddr, endpoint,
//	    zcl.WarningModeBurglar, true, zcl.SirenLevelHigh,
//	    30, 50, zcl.StrobeLevelMedium)
func (a *Adapter) StartWarning(ctx context.Context, nwkAddr uint16, endpoint uint8, mode zcl.WarningMode, useStrobe bool, sirenLevel zcl.SirenLevel, durationSecs uint16, strobeDutyCycle uint8, strobeLevel zcl.StrobeLevel) error {
	// Build the first byte: (mode & 0x0F) | ((strobe & 0x03) << 4) | ((sirenLevel & 0x03) << 6)
	var strobeBits uint8
	if useStrobe {
		strobeBits = 0x02 // Use strobe
	} else {
		strobeBits = 0x00 // No strobe
	}

	firstByte := (uint8(mode) & 0x0F) | ((strobeBits & 0x03) << 4) | ((uint8(sirenLevel) & 0x03) << 6)

	// Build payload: firstByte (1 byte) + warningDuration (2 bytes LE) + strobeDutyCycle (1 byte) + strobeLevel (1 byte)
	payload := []byte{
		firstByte,
		byte(durationSecs & 0xFF),
		byte(durationSecs >> 8),
		strobeDutyCycle,
		uint8(strobeLevel),
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASWD, zcl.CmdIASWDStartWarning, payload)
}

// StopWarning stops any active warning on the device.
// This immediately silences the alarm and turns off any strobe.
//
// This is equivalent to calling StartWarning with WarningModeStop.
func (a *Adapter) StopWarning(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	// To stop warning, send StartWarning command with mode=Stop and duration=0
	payload := []byte{
		uint8(zcl.WarningModeStop), // mode=Stop, no strobe, siren level=Low
		0x00,                       // duration = 0 (LE)
		0x00,
		0x00, // duty cycle = 0
		0x00, // strobe level = Low
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASWD, zcl.CmdIASWDStartWarning, payload)
}

// Squawk makes a brief sound on the device (for arming/disarming feedback).
// This is typically used to provide audio feedback when arming or disarming a security system.
//
// Parameters:
//   - mode: Squawk mode (0=armed, 1=disarmed)
//   - useStrobe: Whether to flash the strobe
//   - level: Volume level (Low, Medium, High, VeryHigh)
//
// Example: Play disarmed squawk with medium volume and strobe:
//
//	err := adapter.Squawk(ctx, nwkAddr, endpoint, 1, true, zcl.SirenLevelMedium)
func (a *Adapter) Squawk(ctx context.Context, nwkAddr uint16, endpoint uint8, mode uint8, useStrobe bool, level zcl.SirenLevel) error {
	// Build the byte: (mode & 0x0F) | ((strobe & 0x01) << 4) | ((level & 0x03) << 6)
	var strobeBit uint8
	if useStrobe {
		strobeBit = 0x01
	}

	squawkByte := (mode & 0x0F) | ((strobeBit & 0x01) << 4) | ((uint8(level) & 0x03) << 6)

	payload := []byte{squawkByte}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterIASWD, zcl.CmdIASWDSquawk, payload)
}

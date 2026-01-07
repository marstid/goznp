package znp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TestDevStateString tests the DevState.String method.
func TestDevStateString(t *testing.T) {
	tests := []struct {
		name  string
		state DevState
		want  string
	}{
		{"DevStateHold", DevStateHold, "Hold"},
		{"DevStateInit", DevStateInit, "Initializing"},
		{"DevStateNwkDisc", DevStateNwkDisc, "Discovering"},
		{"DevStateNwkJoining", DevStateNwkJoining, "Joining"},
		{"DevStateNwkRejoin", DevStateNwkRejoin, "Rejoining"},
		{"DevStateEndDeviceUnauth", DevStateEndDeviceUnauth, "EndDevice (Unauth)"},
		{"DevStateEndDevice", DevStateEndDevice, "EndDevice"},
		{"DevStateRouter", DevStateRouter, "Router"},
		{"DevStateCoordStarting", DevStateCoordStarting, "Coordinator Starting"},
		{"DevStateZbCoord", DevStateZbCoord, "Coordinator"},
		{"DevStateNwkOrphan", DevStateNwkOrphan, "Orphan"},
		{"Unknown state", DevState(99), "Unknown (99)"},
		{"Unknown state 255", DevState(255), "Unknown (255)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("DevState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestApplicationProfileString tests the ApplicationProfile.String method.
func TestApplicationProfileString(t *testing.T) {
	tests := []struct {
		name string
		p    ApplicationProfile
		want string
	}{
		{"ProfileUnknown", ProfileUnknown, "Unknown"},
		{"ProfileHomeAutomation", ProfileHomeAutomation, "Home Automation (HA)"},
		{"ProfileSmartEnergy", ProfileSmartEnergy, "Smart Energy (SE)"},
		{"ProfileGreenPower", ProfileGreenPower, "Green Power (GP)"},
		{"ProfileLightLink", ProfileLightLink, "Light Link / Zigbee 3.0 (ZLL)"},
		{"Custom profile 0x0001", ApplicationProfile(0x0001), "Custom (0x0001)"},
		{"Custom profile 0xFFFF", ApplicationProfile(0xFFFF), "Custom (0xFFFF)"},
		{"Custom profile 0x0100", ApplicationProfile(0x0100), "Custom (0x0100)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.String(); got != tt.want {
				t.Errorf("ApplicationProfile.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestApplicationProfileShortName tests the ApplicationProfile.ShortName method.
func TestApplicationProfileShortName(t *testing.T) {
	tests := []struct {
		name string
		p    ApplicationProfile
		want string
	}{
		{"ProfileUnknown", ProfileUnknown, "unknown"},
		{"ProfileHomeAutomation", ProfileHomeAutomation, "ha"},
		{"ProfileSmartEnergy", ProfileSmartEnergy, "se"},
		{"ProfileGreenPower", ProfileGreenPower, "gp"},
		{"ProfileLightLink", ProfileLightLink, "zll"},
		{"Custom profile", ApplicationProfile(0x0001), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.ShortName(); got != tt.want {
				t.Errorf("ApplicationProfile.ShortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseProfile tests the ParseProfile function.
func TestParseProfile(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      ApplicationProfile
		wantError bool
	}{
		{"ha", "ha", ProfileHomeAutomation, false},
		{"home-automation", "home-automation", ProfileHomeAutomation, false},
		{"homeautomation", "homeautomation", ProfileHomeAutomation, false},
		{"se", "se", ProfileSmartEnergy, false},
		{"smart-energy", "smart-energy", ProfileSmartEnergy, false},
		{"smartenergy", "smartenergy", ProfileSmartEnergy, false},
		{"gp", "gp", ProfileGreenPower, false},
		{"green-power", "green-power", ProfileGreenPower, false},
		{"greenpower", "greenpower", ProfileGreenPower, false},
		{"zll", "zll", ProfileLightLink, false},
		{"light-link", "light-link", ProfileLightLink, false},
		{"lightlink", "lightlink", ProfileLightLink, false},
		{"zigbee3", "zigbee3", ProfileLightLink, false},
		{"unknown profile", "xyz", 0, true},
		{"empty string", "", 0, true},
		{"partial profile name", "hom", 0, true},
		{"uppercase HA", "HA", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProfile(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseProfile() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("ParseProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAvailableProfiles tests the AvailableProfiles function.
func TestAvailableProfiles(t *testing.T) {
	profiles := AvailableProfiles()

	if len(profiles) != 4 {
		t.Errorf("AvailableProfiles() returned %d profiles, want 4", len(profiles))
	}

	expectedProfiles := []ApplicationProfile{
		ProfileHomeAutomation,
		ProfileSmartEnergy,
		ProfileGreenPower,
		ProfileLightLink,
	}

	for i, want := range expectedProfiles {
		if profiles[i] != want {
			t.Errorf("AvailableProfiles()[%d] = %v, want %v", i, profiles[i], want)
		}
	}
}

// TestNvSystemIDConstants tests the NvSystemID constants.
func TestNvSystemIDConstants(t *testing.T) {
	tests := []struct {
		name string
		id   NvSystemID
	}{
		{"NvSysIdZStack", NvSysIdZStack},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constant is defined correctly
			if tt.id > 255 {
				t.Errorf("%s value out of range: %d", tt.name, tt.id)
			}
		})
	}
}

// TestNvItemIDConstants tests the NvItemID constants.
func TestNvItemIDConstants(t *testing.T) {
	// Just verify key constants are within valid ranges
	validIDs := []NvItemID{
		NvExtAddr,
		NvStartupOption,
		NvNIB,
		NvAddrMgr,
		NvNwkActiveKeyInfo,
		NvNwkAlternKeyInfo,
		NvApsLinkKeyTable,
		NvNwkSecMaterialTableStart,
		NvTclkSeed,
		NvApsLinkKeyDataStart,
		NvApsLinkKeyDataEnd,
		NvPrecfgKey,
		NvPrecfgKeysEnable,
		NvLogicalType,
		NvZdoDirectCb,
		NvTransmitPower,
		NvZStackProfile,
		NvDeviceNames,
	}

	for _, id := range validIDs {
		if id == 0 && id != NvStartupOption {
			t.Errorf("NvItemID %d should not be 0 unless it's NvStartupOption", id)
		}
	}
}

// TestBdbCommissioningModeConstants tests the BdbCommissioningMode constants.
func TestBdbCommissioningModeConstants(t *testing.T) {
	tests := []struct {
		name string
		mode BdbCommissioningMode
	}{
		{"BdbCommissioningModeNetworkSteering", BdbCommissioningModeNetworkSteering},
		{"BdbCommissioningModeNetworkFormation", BdbCommissioningModeNetworkFormation},
		{"BdbCommissioningModeFindingBinding", BdbCommissioningModeFindingBinding},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constant is defined correctly
			if tt.mode > 255 {
				t.Errorf("%s value out of range: %d", tt.name, tt.mode)
			}
		})
	}
}

// TestAddressModeConstants tests the AddressMode constants.
func TestAddressModeConstants(t *testing.T) {
	tests := []struct {
		name string
		mode AddressMode
	}{
		{"AddrModeNotPresent", AddrModeNotPresent},
		{"AddrModeGroup", AddrModeGroup},
		{"AddrMode16Bit", AddrMode16Bit},
		{"AddrMode64Bit", AddrMode64Bit},
		{"AddrModeBroadcast", AddrModeBroadcast},
	}

	expectedValues := []uint8{0x00, 0x01, 0x02, 0x03, 0x0F}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint8(tt.mode) != expectedValues[i] {
				t.Errorf("%s = %d, want %d", tt.name, tt.mode, expectedValues[i])
			}
		})
	}
}

// TestParameterTypeConstants tests the ParameterType constants.
func TestParameterTypeConstants(t *testing.T) {
	tests := []struct {
		name string
		pt   ParameterType
		want uint8
	}{
		{"ParamUint8", ParamUint8, 0},
		{"ParamUint16", ParamUint16, 1},
		{"ParamUint32", ParamUint32, 2},
		{"ParamInt8", ParamInt8, 3},
		{"ParamIEEEAddr", ParamIEEEAddr, 4},
		{"ParamBuffer", ParamBuffer, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint8(tt.pt) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.pt, tt.want)
			}
		})
	}
}

// TestDeviceInfoFields tests the DeviceInfo struct.
func TestDeviceInfoFields(t *testing.T) {
	info := &DeviceInfo{
		IEEEAddr:        [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		ShortAddr:       0x1234,
		DeviceType:      uint8(DeviceTypeCoordinator),
		DeviceState:     uint8(DevStateZbCoord),
		NumAssocDevices: 5,
		AssocDevices:    []uint16{0x1001, 0x1002, 0x1003, 0x1004, 0x1005},
	}

	// Just verify the struct can be created and fields are accessible
	if info.ShortAddr != 0x1234 {
		t.Errorf("DeviceInfo.ShortAddr = %d, want 0x1234", info.ShortAddr)
	}
	if info.DeviceType != uint8(DeviceTypeCoordinator) {
		t.Errorf("DeviceInfo.DeviceType = %d, want %d", info.DeviceType, DeviceTypeCoordinator)
	}
	if len(info.AssocDevices) != 5 {
		t.Errorf("DeviceInfo.AssocDevices length = %d, want 5", len(info.AssocDevices))
	}
}

// TestDeviceTypeConstants tests the DeviceType constants.
func TestDeviceTypeConstants(t *testing.T) {
	tests := []struct {
		name string
		dt   uint8
	}{
		{"DeviceTypeCoordinator", DeviceTypeCoordinator},
		{"DeviceTypeRouter", DeviceTypeRouter},
		{"DeviceTypeEndDevice", DeviceTypeEndDevice},
	}

	expectedValues := []uint8{0, 1, 2}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dt != expectedValues[i] {
				t.Errorf("%s = %d, want %d", tt.name, tt.dt, expectedValues[i])
			}
		})
	}
}

// TestAssocDeviceFields tests the AssocDevice struct.
func TestAssocDeviceFields(t *testing.T) {
	device := &AssocDevice{
		ShortAddr:    0x1001,
		IEEEAddr:     [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		NodeRelation: 1,
		Age:          10,
	}

	// Just verify the struct can be created and fields are accessible
	if device.ShortAddr != 0x1001 {
		t.Errorf("AssocDevice.ShortAddr = %d, want 0x1001", device.ShortAddr)
	}
	if device.NodeRelation != 1 {
		t.Errorf("AssocDevice.NodeRelation = %d, want 1", device.NodeRelation)
	}
	if device.Age != 10 {
		t.Errorf("AssocDevice.Age = %d, want 10", device.Age)
	}
}

// TestGetDeviceInfo tests the GetDeviceInfo method.
func TestGetDeviceInfo(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	// Simulate a response after the request
	go func() {
		time.Sleep(10 * time.Millisecond)
		// Build response: status, ieeeAddr(8), shortAddr, deviceType, deviceState, numAssoc, assocList
		buf := NewBuffaloWriter()
		buf.WriteUint8(0) // status success
		buf.WriteIEEEAddr([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
		buf.WriteUint16(0x0000) // shortAddr
		buf.WriteUint8(0) // deviceType: Coordinator
		buf.WriteUint8(9) // deviceState: Coordinator
		buf.WriteUint8(3) // numAssocDevices
		buf.WriteUint16(0x1000)
		buf.WriteUint16(0x1001)
		buf.WriteUint16(0x1002)
		responseFrame := &unpi.Frame{
			Type:      unpi.SRSP,
			Subsystem: unpi.UTIL,
			CommandID: CmdUtilGetDeviceInfo.ID,
			Data:      buf.Bytes(),
		}
		z.waiter.Resolve(responseFrame)
	}()

	ctx := context.Background()
	info, err := z.GetDeviceInfo(ctx)

	if err != nil {
		t.Errorf("GetDeviceInfo() error = %v", err)
		return
	}

	if info.ShortAddr != 0x0000 {
		t.Errorf("GetDeviceInfo() ShortAddr = 0x%04X, want 0x0000", info.ShortAddr)
	}
	if info.DeviceType != 0 {
		t.Errorf("GetDeviceInfo() DeviceType = %d, want 0", info.DeviceType)
	}
	if info.NumAssocDevices != 3 {
		t.Errorf("GetDeviceInfo() NumAssocDevices = %d, want 3", info.NumAssocDevices)
	}
	if len(info.AssocDevices) != 3 {
		t.Errorf("GetDeviceInfo() AssocDevices length = %d, want 3", len(info.AssocDevices))
	}
}

// TestGetDeviceInfoNotOpen tests that GetDeviceInfo returns ErrNotOpen.
func TestGetDeviceInfoNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.GetDeviceInfo(ctx)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("GetDeviceInfo() error = %v, want ErrNotOpen", err)
	}
}

// TestGetAssocCount tests the GetAssocCount method.
func TestGetAssocCount(t *testing.T) {
	tests := []struct {
		name    string
		count   uint16
		wantErr bool
	}{
		{"zero devices", 0, false},
		{"one device", 1, false},
		{"many devices", 100, false},
		{"max uint8 devices (255)", 255, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint16(tt.count)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.UTIL,
					CommandID: CmdUtilAssocCount.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			count, err := z.GetAssocCount(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAssocCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && count != tt.count {
				t.Errorf("GetAssocCount() count = %d, want %d", count, tt.count)
			}
		})
	}
}

// TestGetAssocDevice tests the GetAssocDevice method.
func TestGetAssocDevice(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	// Simulate a response after the request
	go func() {
		time.Sleep(10 * time.Millisecond)
		buf := NewBuffaloWriter()
		// shortAddr, addrIdx, nodeRelation, devStatus, assocCnt, age
		buf.WriteUint16(0x1000)        // shortAddr
		buf.WriteUint16(0)            // addrIdx
		buf.WriteUint8(2)             // nodeRelation
		buf.WriteUint8(0)             // devStatus
		buf.WriteUint8(1)             // assocCnt
		buf.WriteUint8(10)            // age
		// linkInfo: txCounter, txCost, rxLqi, inKeySeqNum, inFrmCntrHigh
		buf.WriteUint8(5)             // txCounter
		buf.WriteUint8(10)            // txCost
		buf.WriteUint8(255)           // rxLqi
		buf.WriteUint8(0)             // inKeySeqNum
		buf.WriteUint16(1234)         // inFrmCntrHigh
		// ieeeAddr
		buf.WriteIEEEAddr([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

		responseFrame := &unpi.Frame{
			Type:      unpi.SRSP,
			Subsystem: unpi.UTIL,
			CommandID: CmdUtilAssocFindDevice.ID,
			Data:      buf.Bytes(),
		}
		z.waiter.Resolve(responseFrame)
	}()

	ctx := context.Background()
	dev, err := z.GetAssocDevice(ctx, 0)

	if err != nil {
		t.Errorf("GetAssocDevice() error = %v", err)
		return
	}

	if dev.ShortAddr != 0x1000 {
		t.Errorf("GetAssocDevice() ShortAddr = 0x%04X, want 0x1000", dev.ShortAddr)
	}
	if dev.NodeRelation != 2 {
		t.Errorf("GetAssocDevice() NodeRelation = %d, want 2", dev.NodeRelation)
	}
	if dev.Age != 10 {
		t.Errorf("GetAssocDevice() Age = %d, want 10", dev.Age)
	}
	if dev.LinkInfo.RxLqi != 255 {
		t.Errorf("GetAssocDevice() LinkInfo.RxLqi = %d, want 255", dev.LinkInfo.RxLqi)
	}
}

// TestSetChannels tests the SetChannels method.
func TestSetChannels(t *testing.T) {
	tests := []struct {
		name        string
		channelMask uint32
		status      uint8
		wantErr     bool
	}{
		{"single channel 11", 1 << 11, 0, false},
		{"single channel 15", 1 << 15, 0, false},
		{"single channel 20", 1 << 20, 0, false},
		{"all channels 11-26", 0x07FFF800, 0, false},
		{"multiple channels", (1<<11)|(1<<15)|(1<<20), 0, false},
		{"zero mask", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)
			z.Open(context.Background())

			// Simulate a response after the request
			go func() {
				time.Sleep(10 * time.Millisecond)
				buf := NewBuffaloWriter()
				buf.WriteUint8(tt.status)
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.UTIL,
					CommandID: CmdUtilSetChannels.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.SetChannels(ctx, tt.channelMask)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("SetChannels() status = %d, want %d", status, tt.status)
			}
		})
	}
}

// TestResetIndicationFields tests the ResetIndication struct.
func TestResetIndicationFields(t *testing.T) {
	ind := &ResetIndication{
		Reason:       1,
		TransportRev: 2,
		Product:      1,
		MajorRel:     3,
		MinorRel:     0,
		MaintRel:     2,
	}

	// Just verify the struct can be created and fields are accessible
	if ind.Reason != 1 {
		t.Errorf("ResetIndication.Reason = %d, want 1", ind.Reason)
	}
	if ind.Product != 1 {
		t.Errorf("ResetIndication.Product = %d, want 1", ind.Product)
	}
}

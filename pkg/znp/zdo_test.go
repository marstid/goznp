package znp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TestExtNwkInfo tests the ExtNwkInfo method.
func TestExtNwkInfo(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)
	z.Open(context.Background())

	// Simulate a response after the request
	go func() {
		time.Sleep(10 * time.Millisecond)
		// Build response: shortAddr, deviceState, panId, parentAddr, extendedPanId, parentExtAddr, channel
		buf := NewBuffaloWriter()
		buf.WriteUint16(0x0000)  // shortAddr
		buf.WriteUint8(9)          // deviceState: Coordinator
		buf.WriteUint16(0x1234)    // panId
		buf.WriteUint16(0xFFFF)    // parentAddr (none for coordinator)
		buf.WriteIEEEAddr([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}) // extendedPanId
		buf.WriteIEEEAddr([8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // parentExtAddr
		buf.WriteUint8(15)         // channel

		responseFrame := &unpi.Frame{
			Type:      unpi.SRSP,
			Subsystem: unpi.ZDO,
			CommandID: CmdZdoExtNwkInfo.ID,
			Data:      buf.Bytes(),
		}
		z.waiter.Resolve(responseFrame)
	}()

	ctx := context.Background()
	info, err := z.ExtNwkInfo(ctx)

	if err != nil {
		t.Errorf("ExtNwkInfo() error = %v", err)
		return
	}

	if info.ShortAddr != 0x0000 {
		t.Errorf("ExtNwkInfo() ShortAddr = 0x%04X, want 0x0000", info.ShortAddr)
	}
	if info.DevState != 9 {
		t.Errorf("ExtNwkInfo() DevState = %d, want 9", info.DevState)
	}
	if info.PanID != 0x1234 {
		t.Errorf("ExtNwkInfo() PanID = 0x%04X, want 0x1234", info.PanID)
	}
	if info.Channel != 15 {
		t.Errorf("ExtNwkInfo() Channel = %d, want 15", info.Channel)
	}
}

// TestExtNwkInfoNotOpen tests that ExtNwkInfo returns ErrNotOpen.
func TestExtNwkInfoNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.ExtNwkInfo(ctx)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("ExtNwkInfo() error = %v, want ErrNotOpen", err)
	}
}

// TestStartupFromApp tests the StartupFromApp method.
func TestStartupFromApp(t *testing.T) {
	tests := []struct {
		name       string
		startDelay uint16
		status     uint8
		wantErr    bool
	}{
		{
			name:       "immediate start",
			startDelay: 0,
			status:     0,
			wantErr:    false,
		},
		{
			name:       "delayed start",
			startDelay: 100,
			status:     0,
			wantErr:    false,
		},
		{
			name:       "max delay",
			startDelay: 65535,
			status:     0,
			wantErr:    false,
		},
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
					Subsystem: unpi.ZDO,
					CommandID: CmdZdoStartupFromApp.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.StartupFromApp(ctx, tt.startDelay)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartupFromApp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("StartupFromApp() status = %d, want %d", status, tt.status)
			}
		})
	}
}

// TestStartupFromAppNotOpen tests that StartupFromApp returns ErrNotOpen.
func TestStartupFromAppNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.StartupFromApp(ctx, 0)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("StartupFromApp() error = %v, want ErrNotOpen", err)
	}
}

// TestMgmtNwkUpdateReq tests the MgmtNwkUpdateReq method.
func TestMgmtNwkUpdateReq(t *testing.T) {
	tests := []struct {
		name           string
		dstAddr        uint16
		dstAddrMode    uint8
		channelMask    uint32
		scanDuration   uint8
		scanCount      uint8
		nwkManagerAddr uint16
		status         uint8
		wantErr        bool
	}{
		{
			name:           "channel change broadcast",
			dstAddr:        0xFFFF,
			dstAddrMode:    0x0F,
			channelMask:    1 << 15,
			scanDuration:   0xFE,
			scanCount:      0,
			nwkManagerAddr: 0x0000,
			status:         0,
			wantErr:        false,
		},
		{
			name:           "energy scan",
			dstAddr:        0x0000,
			dstAddrMode:    0x02,
			channelMask:    0x07FFF800,
			scanDuration:   0x04,
			scanCount:      1,
			nwkManagerAddr: 0x0000,
			status:         0,
			wantErr:        false,
		},
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
					Subsystem: unpi.ZDO,
					CommandID: CmdZdoMgmtNwkUpdateReq.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.MgmtNwkUpdateReq(ctx, tt.dstAddr, tt.dstAddrMode, tt.channelMask, tt.scanDuration, tt.scanCount, tt.nwkManagerAddr)

			if (err != nil) != tt.wantErr {
				t.Errorf("MgmtNwkUpdateReq() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("MgmtNwkUpdateReq() status = %d, want %d", status, tt.status)
			}
		})
	}
}

// TestMgmtNwkUpdateReqNotOpen tests that MgmtNwkUpdateReq returns ErrNotOpen.
func TestMgmtNwkUpdateReqNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.MgmtNwkUpdateReq(ctx, 0xFFFF, 0x0F, 1<<15, 0xFE, 0, 0x0000)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("MgmtNwkUpdateReq() error = %v, want ErrNotOpen", err)
	}
}

// TestExtNetworkInfo tests the ExtNetworkInfo struct.
func TestExtNetworkInfo(t *testing.T) {
	info := &ExtNetworkInfo{
		ShortAddr:     0x0000,
		DevState:      9,
		PanID:         0x1234,
		ParentAddr:    0xFFFF,
		ExtendedPanID: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		ParentExtAddr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Channel:       15,
	}

	if info.ShortAddr != 0x0000 {
		t.Errorf("ExtNetworkInfo.ShortAddr = 0x%04X, want 0x0000", info.ShortAddr)
	}
	if info.DevState != 9 {
		t.Errorf("ExtNetworkInfo.DevState = %d, want 9", info.DevState)
	}
	if info.PanID != 0x1234 {
		t.Errorf("ExtNetworkInfo.PanID = 0x%04X, want 0x1234", info.PanID)
	}
	if info.Channel != 15 {
		t.Errorf("ExtNetworkInfo.Channel = %d, want 15", info.Channel)
	}
}

package znp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TestBdbStartCommissioning tests the BdbStartCommissioning method.
func TestBdbStartCommissioning(t *testing.T) {
	tests := []struct {
		name      string
		mode      BdbCommissioningMode
		status    uint8
		wantErr   bool
	}{
		{
			name:    "network steering",
			mode:    BdbCommissioningModeNetworkSteering,
			status:  0,
			wantErr: false,
		},
		{
			name:    "network formation",
			mode:    BdbCommissioningModeNetworkFormation,
			status:  0,
			wantErr: false,
		},
		{
			name:    "finding binding",
			mode:    BdbCommissioningModeFindingBinding,
			status:  0,
			wantErr: false,
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
					Subsystem: unpi.APPCNF,
					CommandID: CmdAppCnfBdbStartCommissioning.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.BdbStartCommissioning(ctx, tt.mode)

			if (err != nil) != tt.wantErr {
				t.Errorf("BdbStartCommissioning() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("BdbStartCommissioning() status = %d, want %d", status, tt.status)
			}
		})
	}
}

// TestBdbStartCommissioningNotOpen tests that BdbStartCommissioning returns ErrNotOpen.
func TestBdbStartCommissioningNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.BdbStartCommissioning(ctx, BdbCommissioningModeNetworkSteering)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("BdbStartCommissioning() error = %v, want ErrNotOpen", err)
	}
}

// TestBdbSetChannel tests the BdbSetChannel method.
func TestBdbSetChannel(t *testing.T) {
	tests := []struct {
		name        string
		isPrimary   bool
		channel     uint32
		status      uint8
		wantErr     bool
	}{
		{
			name:      "primary channel 11",
			isPrimary: true,
			channel:   1 << 11,
			status:    0,
			wantErr:   false,
		},
		{
			name:      "secondary channel 15",
			isPrimary: false,
			channel:   1 << 15,
			status:    0,
			wantErr:   false,
		},
		{
			name:      "primary all channels",
			isPrimary: true,
			channel:   0x07FFF800,
			status:    0,
			wantErr:   false,
		},
		{
			name:      "secondary all channels",
			isPrimary: false,
			channel:   0x07FFF800,
			status:    0,
			wantErr:   false,
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
					Subsystem: unpi.APPCNF,
					CommandID: CmdAppCnfBdbSetChannel.ID,
					Data:      buf.Bytes(),
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.BdbSetChannel(ctx, tt.isPrimary, tt.channel)

			if (err != nil) != tt.wantErr {
				t.Errorf("BdbSetChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("BdbSetChannel() status = %d, want %d", status, tt.status)
			}
		})
	}
}

// TestBdbSetChannelNotOpen tests that BdbSetChannel returns ErrNotOpen.
func TestBdbSetChannelNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.BdbSetChannel(ctx, true, 1<<15)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("BdbSetChannel() error = %v, want ErrNotOpen", err)
	}
}

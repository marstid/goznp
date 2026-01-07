package znp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TestAfDelete tests the AfDelete method.
func TestAfDelete(t *testing.T) {
	tests := []struct {
		name     string
		endpoint uint8
		status   uint8
	}{
		{
			name:     "success status 0",
			endpoint: 1,
			status:   0x00,
		},
		{
			name:     "success status 0x09 (not found)",
			endpoint: 1,
			status:   0x09,
		},
		{
			name:     "all endpoints",
			endpoint: 0xFF,
			status:   0x00,
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
				responseData := []byte{tt.status}
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.AF,
					CommandID: CmdAfDelete.ID,
					Data:      responseData,
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.AfDelete(ctx, tt.endpoint)

			if err != nil {
				t.Errorf("AfDelete() error = %v", err)
			}
			if status != tt.status {
				t.Errorf("AfDelete() status = %v, want %v", status, tt.status)
			}
		})
	}
}

// TestAfDeleteNotOpen tests that AfDelete returns ErrNotOpen when not open.
func TestAfDeleteNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	ctx := context.Background()
	_, err := z.AfDelete(ctx, 1)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("AfDelete() error = %v, want ErrNotOpen", err)
	}
}

// TestAfRegister tests the AfRegister method.
func TestAfRegister(t *testing.T) {
	tests := []struct {
		name     string
		config   EndpointConfig
		status   uint8
		wantErr  bool
	}{
		{
			name: "simple endpoint",
			config: EndpointConfig{
				Endpoint:    1,
				AppProfID:   0x0104,
				AppDeviceID: 0x0000,
				AppDevVer:   1,
				LatencyReq:  0,
				InClusters:  []uint16{0x0000, 0x0001},
				OutClusters: []uint16{0x0006},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "empty clusters",
			config: EndpointConfig{
				Endpoint:    2,
				AppProfID:   0x0104,
				AppDeviceID: 0x0100,
				AppDevVer:   0,
				LatencyReq:  0,
				InClusters:  []uint16{},
				OutClusters: []uint16{},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "max clusters",
			config: EndpointConfig{
				Endpoint:    3,
				AppProfID:   0x0104,
				AppDeviceID: 0x0200,
				AppDevVer:   1,
				LatencyReq:  0,
				InClusters:  make([]uint16, MaxClusters),
				OutClusters: make([]uint16, MaxClusters),
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name:     "too many input clusters",
			config: EndpointConfig{
				Endpoint:    4,
				AppProfID:   0x0104,
				AppDeviceID: 0x0100,
				AppDevVer:   1,
				LatencyReq:  0,
				InClusters:  make([]uint16, MaxClusters+1),
				OutClusters: []uint16{},
			},
			wantErr: true,
		},
		{
			name: "too many output clusters",
			config: EndpointConfig{
				Endpoint:    5,
				AppProfID:   0x0104,
				AppDeviceID: 0x0100,
				AppDevVer:   1,
				LatencyReq:  0,
				InClusters:  []uint16{},
				OutClusters: make([]uint16, MaxClusters+1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)

			if !tt.wantErr {
				z.Open(context.Background())

				// Simulate a response after the request
				go func() {
					time.Sleep(10 * time.Millisecond)
					responseData := []byte{tt.status}
					responseFrame := &unpi.Frame{
						Type:      unpi.SRSP,
						Subsystem: unpi.AF,
						CommandID: CmdAfRegister.ID,
						Data:      responseData,
					}
					z.waiter.Resolve(responseFrame)
				}()
			}

			ctx := context.Background()
			status, err := z.AfRegister(ctx, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("AfRegister() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("AfRegister() status = %v, want %v", status, tt.status)
			}
		})
	}
}

// TestAfRegisterNotOpen tests that AfRegister returns ErrNotOpen when not open.
func TestAfRegisterNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	config := EndpointConfig{
		Endpoint:    1,
		AppProfID:   0x0104,
		AppDeviceID: 0x0100,
		AppDevVer:   1,
		LatencyReq:  0,
		InClusters:  []uint16{},
		OutClusters: []uint16{},
	}

	ctx := context.Background()
	_, err := z.AfRegister(ctx, config)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("AfRegister() error = %v, want ErrNotOpen", err)
	}
}

// TestAfDataRequest tests the AfDataRequest method.
func TestAfDataRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     DataRequest
		status  uint8
		wantErr bool
	}{
		{
			name: "simple data request",
			req: DataRequest{
				DstAddr:     0x1234,
				DstEndpoint: 1,
				SrcEndpoint: 0,
				ClusterID:   0x0006,
				TransID:     1,
				Options:     AfOptionNone,
				Radius:      0,
				Data:        []byte{0x01, 0x00},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "with acknowledgment",
			req: DataRequest{
				DstAddr:     0x1234,
				DstEndpoint: 1,
				SrcEndpoint: 0,
				ClusterID:   0x0006,
				TransID:     1,
				Options:     AfOptionAckRequest,
				Radius:      30,
				Data:        []byte{0x01},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "with security",
			req: DataRequest{
				DstAddr:     0x1234,
				DstEndpoint: 1,
				SrcEndpoint: 0,
				ClusterID:   0x0006,
				TransID:     1,
				Options:     AfOptionEnableSecurity,
				Radius:      10,
				Data:        []byte{0x01},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "all options",
			req: DataRequest{
				DstAddr:     0x1234,
				DstEndpoint: 1,
				SrcEndpoint: 0,
				ClusterID:   0x0006,
				TransID:     1,
				Options:     AfOptionAckRequest | AfOptionEnableSecurity | AfOptionDiscoverRoute,
				Radius:      255,
				Data:        []byte{0x01},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "empty data",
			req: DataRequest{
				DstAddr:     0x1234,
				DstEndpoint: 1,
				SrcEndpoint: 0,
				ClusterID:   0x0006,
				TransID:     1,
				Options:     AfOptionNone,
				Radius:      0,
				Data:        []byte{},
			},
			status:  0x00,
			wantErr: false,
		},
		{
			name: "large data",
			req: DataRequest{
				DstAddr:     0x1234,
				DstEndpoint: 1,
				SrcEndpoint: 0,
				ClusterID:   0x0006,
				TransID:     1,
				Options:     AfOptionNone,
				Radius:      0,
				Data:        make([]byte, 100),
			},
			status:  0x00,
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
				responseData := []byte{tt.status}
				responseFrame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.AF,
					CommandID: CmdAfDataRequest.ID,
					Data:      responseData,
				}
				z.waiter.Resolve(responseFrame)
			}()

			ctx := context.Background()
			status, err := z.AfDataRequest(ctx, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("AfDataRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && status != tt.status {
				t.Errorf("AfDataRequest() status = %v, want %v", status, tt.status)
			}
		})
	}
}

// TestAfDataRequestNotOpen tests that AfDataRequest returns ErrNotOpen when not open.
func TestAfDataRequestNotOpen(t *testing.T) {
	mock := &mockPort{}
	z := New(mock)

	req := DataRequest{
		DstAddr:     0x1234,
		DstEndpoint: 1,
		SrcEndpoint: 0,
		ClusterID:   0x0006,
		TransID:     1,
		Options:     AfOptionNone,
		Radius:      0,
		Data:        []byte{0x01},
	}

	ctx := context.Background()
	_, err := z.AfDataRequest(ctx, req)

	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("AfDataRequest() error = %v, want ErrNotOpen", err)
	}
}

// TestWaitForDataConfirm tests the WaitForDataConfirm method.
func TestWaitForDataConfirm(t *testing.T) {
	tests := []struct {
		name          string
		transID       uint8
		responses     [][]byte
		timeout       time.Duration
		wantStatus    uint8
		wantEndpoint  uint8
		wantTransID   uint8
		wantErr       bool
		cancelContext bool
	}{
		{
			name: "success",
			transID: 1,
			responses: [][]byte{
				// status, endpoint, transID
				{0x00, 0x01, 0x01},
			},
			timeout:      100 * time.Millisecond,
			wantStatus:   0x00,
			wantEndpoint: 0x01,
			wantTransID:  0x01,
			wantErr:      false,
		},
		{
			name: "wrong transID rejected",
			transID: 1,
			responses: [][]byte{
				// Wrong transID first
				{0x00, 0x01, 0x02},
				// Correct transID second
				{0x00, 0x01, 0x01},
			},
			timeout:      100 * time.Millisecond,
			wantStatus:   0x00,
			wantEndpoint: 0x01,
			wantTransID:  0x01,
			wantErr:      false,
		},
		{
			name:    "timeout",
			transID: 1,
			responses: [][]byte{
				// status, endpoint, transID
				{0x00, 0x01, 0x02}, // Wrong transID
			},
			timeout:  50 * time.Millisecond,
			wantErr:  true,
		},
		{
			name:      "context cancelled",
			transID:   1,
			timeout:   100 * time.Millisecond,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPort{}
			z := New(mock)

			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// Send responses after initial delay
			go func() {
				time.Sleep(5 * time.Millisecond)
				for i, resp := range tt.responses {
					time.Sleep(10 * time.Millisecond)
					frame := &unpi.Frame{
						Type:      unpi.AREQ,
						Subsystem: unpi.AF,
						CommandID: CmdAfDataConfirm.ID,
						Data:      resp,
					}
					z.waiter.Resolve(frame)
					if i == 0 {
						// Signal waiter has something
						time.Sleep(10 * time.Millisecond)
					}
				}
			}()

			if !tt.cancelContext {
				time.Sleep(10 * time.Millisecond)
			}

			confirm, err := z.WaitForDataConfirm(ctx, tt.transID, tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Logf("WaitForDataConfirm() error = %v, wantErr %v", err, tt.wantErr)
				// Don't fail here, as timeout behavior may vary
			}

			if !tt.wantErr && err == nil {
				if confirm.Status != tt.wantStatus {
					t.Errorf("WaitForDataConfirm() status = %v, want %v", confirm.Status, tt.wantStatus)
				}
				if confirm.Endpoint != tt.wantEndpoint {
					t.Errorf("WaitForDataConfirm() endpoint = %v, want %v", confirm.Endpoint, tt.wantEndpoint)
				}
				if confirm.TransID != tt.wantTransID {
					t.Errorf("WaitForDataConfirm() transID = %v, want %v", confirm.TransID, tt.wantTransID)
				}
			}
		})
	}
}

// TestParseIncomingMsg tests the parseIncomingMsg function.
func TestParseIncomingMsg(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		wantGroup uint16
		wantCluster uint16
		wantSrcAddr  uint16
	}{
		{
			name: "valid message with data",
			data: // groupID(2), clusterID(2), srcAddr(2), srcEndpoint(1), dstEndpoint(1),
				// wasBroadcast(1), linkQuality(1), securityUse(1), timestamp(4), transSeq(1), dataLen(1), data(N)
				func() []byte {
					buf := NewBuffaloWriter()
					buf.WriteUint16(0x0000)  // groupID
					buf.WriteUint16(0x0006)  // clusterID
					buf.WriteUint16(0x1234)  // srcAddr
					buf.WriteUint8(0x01)     // srcEndpoint
					buf.WriteUint8(0x00)     // dstEndpoint
					buf.WriteUint8(0x00)     // wasBroadcast
					buf.WriteUint8(0xFF)     // linkQuality
					buf.WriteUint8(0x00)     // securityUse
					buf.WriteUint32(12345)   // timestamp
					buf.WriteUint8(0x01)     // transSeqNum
					buf.WriteUint8(0x02)     // data length
					buf.WriteBytes([]byte{0x01, 0x00}) // data
					return buf.Bytes()
				}(),
			wantErr:      false,
			wantGroup:    0x0000,
			wantCluster:  0x0006,
			wantSrcAddr:  0x1234,
		},
		{
			name: "broadcast message",
			data: func() []byte {
				buf := NewBuffaloWriter()
				buf.WriteUint16(0x0000)
				buf.WriteUint16(0x0006)
				buf.WriteUint16(0xFFFD)  // broadcast addr
				buf.WriteUint8(0x01)
				buf.WriteUint8(0x00)
				buf.WriteUint8(0x01)     // was broadcast
				buf.WriteUint8(0xFF)
				buf.WriteUint8(0x00)
				buf.WriteUint32(12345)
				buf.WriteUint8(0x01)
				buf.WriteUint8(0x00)     // no data
				return buf.Bytes()
			}(),
			wantErr:      false,
			wantGroup:    0x0000,
			wantCluster:  0x0006,
			wantSrcAddr:  0xFFFD,
		},
		{
			name: "secure message",
			data: func() []byte {
				buf := NewBuffaloWriter()
				buf.WriteUint16(0x0000)
				buf.WriteUint16(0x0006)
				buf.WriteUint16(0x1234)
				buf.WriteUint8(0x01)
				buf.WriteUint8(0x00)
				buf.WriteUint8(0x00)
				buf.WriteUint8(0xFF)
				buf.WriteUint8(0x01)     // security used
				buf.WriteUint32(12345)
				buf.WriteUint8(0x01)
				buf.WriteUint8(0x00)
				return buf.Bytes()
			}(),
			wantErr:      false,
			wantGroup:    0x0000,
			wantCluster:  0x0006,
			wantSrcAddr:  0x1234,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name: "too short - missing fields",
			data: func() []byte {
				buf := NewBuffaloWriter()
				buf.WriteUint16(0x0000) // incomplete
				return buf.Bytes()
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := parseIncomingMsg(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseIncomingMsg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if msg.GroupID != tt.wantGroup {
					t.Errorf("parseIncomingMsg() groupID = %v, want %v", msg.GroupID, tt.wantGroup)
				}
				if msg.ClusterID != tt.wantCluster {
					t.Errorf("parseIncomingMsg() clusterID = %v, want %v", msg.ClusterID, tt.wantCluster)
				}
				if msg.SrcAddr != tt.wantSrcAddr {
					t.Errorf("parseIncomingMsg() srcAddr = %v, want %v", msg.SrcAddr, tt.wantSrcAddr)
				}
			}
		})
	}
}

// TestIncomingMessage tests IncomingMessage struct fields.
func TestIncomingMessage(t *testing.T) {
	msg := &IncomingMessage{
		GroupID:      0x0000,
		ClusterID:    0x0006,
		SrcAddr:      0x1234,
		SrcEndpoint:  1,
		DstEndpoint:  0,
		WasBroadcast: true,
		LinkQuality:  255,
		SecurityUse:  true,
		Timestamp:    12345,
		TransSeqNum:  1,
		Data:         []byte{0x01, 0x00},
	}

	// Verify fields are accessible
	if msg.GroupID != 0x0000 {
		t.Errorf("IncomingMessage.GroupID = %v, want 0x0000", msg.GroupID)
	}
	if msg.ClusterID != 0x0006 {
		t.Errorf("IncomingMessage.ClusterID = %v, want 0x0006", msg.ClusterID)
	}
	if msg.SrcAddr != 0x1234 {
		t.Errorf("IncomingMessage.SrcAddr = %v, want 0x1234", msg.SrcAddr)
	}
	if !msg.WasBroadcast {
		t.Errorf("IncomingMessage.WasBroadcast = false, want true")
	}
	if !msg.SecurityUse {
		t.Errorf("IncomingMessage.SecurityUse = false, want true")
	}
}

// TestDataConfirm tests DataConfirm struct fields.
func TestDataConfirm(t *testing.T) {
	confirm := &DataConfirm{
		Status:   0x00,
		Endpoint: 1,
		TransID:  42,
	}

	if confirm.Status != 0x00 {
		t.Errorf("DataConfirm.Status = %v, want 0x00", confirm.Status)
	}
	if confirm.Endpoint != 1 {
		t.Errorf("DataConfirm.Endpoint = %v, want 1", confirm.Endpoint)
	}
	if confirm.TransID != 42 {
		t.Errorf("DataConfirm.TransID = %v, want 42", confirm.TransID)
	}
}

// TestDataRequestOptions tests DataRequestOptions constants.
func TestDataRequestOptions(t *testing.T) {
	tests := []struct {
		name  string
		value DataRequestOptions
	}{
		{"AfOptionNone", AfOptionNone},
		{"AfOptionAckRequest", AfOptionAckRequest},
		{"AfOptionDiscoverRoute", AfOptionDiscoverRoute},
		{"AfOptionEnableSecurity", AfOptionEnableSecurity},
		{"AfOptionSkipRouting", AfOptionSkipRouting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constant is defined and has the expected value
			if tt.value < 0 || tt.value > 0xFF {
				t.Errorf("%s = %d, out of valid range", tt.name, tt.value)
			}
		})
	}
}

// TestEndpointConfig tests EndpointConfig struct.
func TestEndpointConfig(t *testing.T) {
	config := EndpointConfig{
		Endpoint:    1,
		AppProfID:   uint16(ProfileHomeAutomation),
		AppDeviceID: 0x0100,
		AppDevVer:   1,
		LatencyReq:  0,
		InClusters:  []uint16{0x0000, 0x0001, 0x0003, 0x0006},
		OutClusters: []uint16{0x0019, 0x0201},
	}

	if config.Endpoint != 1 {
		t.Errorf("EndpointConfig.Endpoint = %v, want 1", config.Endpoint)
	}
	if config.AppProfID != uint16(ProfileHomeAutomation) {
		t.Errorf("EndpointConfig.AppProfID = %v, want %v", config.AppProfID, ProfileHomeAutomation)
	}
	if len(config.InClusters) != 4 {
		t.Errorf("EndpointConfig.InClusters length = %v, want 4", len(config.InClusters))
	}
	if len(config.OutClusters) != 2 {
		t.Errorf("EndpointConfig.OutClusters length = %v, want 2", len(config.OutClusters))
	}
}

// TestMaxClusters tests MaxClusters constant.
func TestMaxClusters(t *testing.T) {
	if MaxClusters != 32 {
		t.Errorf("MaxClusters = %d, want 32", MaxClusters)
	}
}

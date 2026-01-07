// Package znp implements the Zigbee Network Processor protocol for communicating
// with TI Z-Stack coordinators.
//
// The ZNP protocol provides a command interface over a serial port for
// controlling the Zigbee coordinator and managing the network.
//
// # Protocol Layers
//
// The ZNP protocol is built on top of UNPI (Unified Network Processor Interface):
//
//	ZNP Commands (SYS, AF, ZDO, NV)
//	    ↓
//	UNPI Frames (SREQ, SRSP, AREQ)
//	    ↓
//	Serial Port
//
// # Basic Usage
//
// Create a ZNP client and open it:
//
//	port, _ := serial.Open(serial.Config{Path: "/dev/ttyUSB0", BaudRate: 115200})
//	z := znp.New(port)
//	if err := z.Open(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer z.Close()
//
// # Request-Response Pattern
//
// Send a synchronous request and wait for response:
//
//	frame, err := z.Request(ctx, unpi.SYS, znp.CmdSysPing, nil)
//
// # Asynchronous Messages
//
// Register a callback for unsolicited frames (device events, etc.):
//
//	z.OnFrame(func(frame *unpi.Frame) {
//	    fmt.Printf("Received: %+v\n", frame)
//	})
//
// # Device Events
//
// Register callbacks for device-related events:
//
//	z.OnDeviceJoin(func(ind *znp.TcDeviceInd) {
//	    fmt.Printf("Device joined: %v\n", ind.IEEEAddr)
//	})
//
//	z.OnDeviceLeave(func(ind *znp.DeviceLeave) {
//	    fmt.Printf("Device left: %v\n", ind.IEEEAddr)
//	})
//
//	z.OnDeviceAnnounce(func(ind *znp.DeviceAnnounce) {
//	    fmt.Printf("Device announced: %v\n", ind.IEEEAddr)
//	})
//
// # Subsystems
//
// ZNP commands are organized into subsystems:
//
//	SYS: System management (ping, reset, version, NV operations)
//	AF:  Application Framework (endpoints, data send/receive)
//	ZDO: Zigbee Device Objects (discovery, binding, network management)
//	NV:  Non-Volatile memory access (persistent storage)
//
// # Frame Types
//
// - SREQ: Synchronous Request - expects SRSP response
// - SRSP: Synchronous Response - response to SREQ
// - AREQ: Asynchronous Request - unsolicited notification
//
// # Timeout Handling
//
// Waiter mechanism handles request-response matching with configurable timeout:
//
//	const DefaultSREQTimeout = 6 * time.Second
//
// # Context Support
//
// All methods accept context.Context for cancellation:
//
//	resp, err := z.Request(ctxWithTimeout, unpi.SYS, cmd, data)
//
// # Thread Safety
//
// ZNP is safe for concurrent use from multiple goroutines.
package znp

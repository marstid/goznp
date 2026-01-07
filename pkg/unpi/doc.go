// Package unpi provides parsing and serialization for the Unified Network Processor
// Interface (UNPI) protocol used by Texas Instruments Z-Stack firmware.
//
// UNPI is a binary framing protocol that wraps ZNP (Zigbee Network Processor) commands
// for serial communication with TI CC2652 and similar Zigbee coordinators.
//
// # Frame Structure
//
// A UNPI frame has the following structure:
//
//	SOF (1 byte):     0xFE - Start of Frame delimiter
//	Len (2 bytes):    Big-endian length of data payload
//	Cmd0 (1 byte):    Header byte combining type and subsystem
//	Cmd1 (1 byte):    Command ID
//	Data (N bytes):   Command-specific payload
//	FCS (1 byte):     Frame Check Sequence (checksum)
//
// Total size: 6 + N bytes minimum
//
// # Frame Types
//
//	0x00: SREQ - Synchronous Request (expects SRSP)
//	0x01: SRSP - Synchronous Response
//	0x02: AREQ - Asynchronous Request (unsolicited notification)
//
// # Subsystems
//
//	SYS (0x00): System management commands
//	AF  (0x01): Application Framework
//	ZDO (0x02): Zigbee Device Objects
//	UTL (0x04): Utilities
//	NV  (0x05): Non-Volatile memory
//
// # Creating Frames
//
// Create a frame for a ping request:
//
//	frame, err := unpi.NewFrame(
//	    unpi.SREQ,         // Frame type
//	    unpi.SYS,          // Subsystem
//	    0x01,             // Command ID (ping)
//	    []byte{0x30},     // Data payload
//	)
//
// # Serializing Frames
//
// Convert frame to wire bytes:
//
//	data := frame.ToBytes()
//	// Returns: [0xFE, 0x00, 0x01, 0x00, 0x01, 0x30, checksum]
//
// # Parsing Frames
//
// Parse raw bytes into a frame:
//
//	frame, err := unpi.ParseFrame(data)
//	if err != nil {
//	    // Invalid frame (bad checksum, length, etc.)
//	}
//
// # Streaming Parser
//
// For reading from serial ports, use the streaming parser:
//
//	parser := unpi.NewParser()
//
//	// Feed incoming bytes
//	parser.Frame() <- frame
//
//	// Receive parsed frames
//	for frame := range parser.Frames() {
//	    handleFrame(frame)
//	}
//
//	// Handle parse errors
//	errors := parser.Errors()
//	for err := range errors {
//	    log.Printf("Parse error: %v", err)
//	}
//
// # Checksum Validation
//
// Frames are automatically validated on parse. The checksum calculation:
//
//	checksum = 0xFF - (sum of all bytes from SOF to end of data) & 0xFF
//
// # Maximum Frame Size
//
//	data := unpi.NewFrame(...)
//	if len(data) > unpi.MaxDataSize {
//	    // Data payload too large
//	}
//
// # Common Errors
//
//   - ErrInvalidSOF: Frame doesn't start with 0xFE
//   - ErrInvalidChecksum: Checksum doesn't match calculated value
//   - ErrDataTooLarge: Data payload exceeds MaxDataSize
//   - ErrFrameTooShort: Frame shorter than minimum length
//
// # Usage Pattern
//
// Typical serial communication pattern:
//
//	// Create parser for incoming data
//	parser := NewParser()
//
//	go func() {
//	    buf := make([]byte, 512)
//	    for {
//	        n, _ := serialPort.Read(buf)
//	        parser.Feed(buf[:n])
//	    }
//	}()
//
//	// Handle incoming frames
//	for frame := range parser.Frames() {
//	    // Dispatch based on type/subsystem
//	    handleIncomingFrame(frame)
//	}
//
//	// Send commands
//	reqFrame := NewFrame(SREQ, SYS, CmdPing, nil)
//	serialPort.Write(reqFrame.ToBytes())
//
// # Thread Safety
//
// Frame creation and parsing are stateless and safe for concurrent use.
// The Parser is not thread-safe and should only be used from a single goroutine.
package unpi

package unpi_test

import (
	"fmt"

	"github.com/marstid/goznp/pkg/unpi"
)

// Example demonstrating basic frame creation and serialization.
func ExampleNewFrame() {
	// Create a SREQ frame to the SYS subsystem.
	frame, err := unpi.NewFrame(
		unpi.SREQ,
		unpi.SYS,
		0x01,
		[]byte{0xAA, 0xBB, 0xCC},
	)
	if err != nil {
		panic(err)
	}

	// Serialize to bytes for transmission.
	frameBytes := frame.ToBytes()

	fmt.Printf("Frame length: %d bytes\n", len(frameBytes))
	fmt.Printf("SOF: 0x%02X\n", frameBytes[0])
	fmt.Printf("Type: %s\n", frame.Type)
	fmt.Printf("Subsystem: %s\n", frame.Subsystem)
	// Output:
	// Frame length: 8 bytes
	// SOF: 0xFE
	// Type: SREQ
	// Subsystem: SYS
}

// Example demonstrating frame parsing from bytes.
func ExampleParseFrame() {
	// Raw bytes received from serial port (example SREQ to SYS).
	rawBytes := []byte{0xFE, 0x03, 0x21, 0x01, 0xAA, 0xBB, 0xCC, 0xFE}

	// Parse the frame.
	frame, err := unpi.ParseFrame(rawBytes)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Type: %s\n", frame.Type)
	fmt.Printf("Subsystem: %s\n", frame.Subsystem)
	fmt.Printf("CommandID: 0x%02X\n", frame.CommandID)
	fmt.Printf("Data length: %d\n", len(frame.Data))
	// Output:
	// Type: SREQ
	// Subsystem: SYS
	// CommandID: 0x01
	// Data length: 3
}

// Example demonstrating streaming parser usage.
func ExampleParser() {
	parser := unpi.NewParser()

	// Simulate receiving data from serial port in chunks.
	chunk1 := []byte{0xFE, 0x01, 0x21}
	chunk2 := []byte{0x01, 0xAA, 0x8B}

	parser.Feed(chunk1)
	parser.Feed(chunk2)

	// Read parsed frame.
	select {
	case frame := <-parser.Frames():
		fmt.Printf("Type: %s\n", frame.Type)
		fmt.Printf("Subsystem: %s\n", frame.Subsystem)
		fmt.Printf("CommandID: 0x%02X\n", frame.CommandID)
	case err := <-parser.Errors():
		fmt.Printf("Error: %v\n", err)
	}
	// Output:
	// Type: SREQ
	// Subsystem: SYS
	// CommandID: 0x01
}

// Example demonstrating Type enum usage.
func ExampleType_String() {
	types := []unpi.Type{unpi.POLL, unpi.SREQ, unpi.AREQ, unpi.SRSP}

	for _, t := range types {
		fmt.Printf("%s (0x%02X)\n", t, uint8(t))
	}
	// Output:
	// POLL (0x00)
	// SREQ (0x01)
	// AREQ (0x02)
	// SRSP (0x03)
}

// Example demonstrating Subsystem enum usage.
func ExampleSubsystem_String() {
	subsystems := []unpi.Subsystem{
		unpi.SYS,
		unpi.ZDO,
		unpi.AF,
		unpi.UTIL,
	}

	for _, s := range subsystems {
		fmt.Printf("%s (0x%02X)\n", s, uint8(s))
	}
	// Output:
	// SYS (0x01)
	// ZDO (0x05)
	// AF (0x04)
	// UTIL (0x07)
}

package unpi

import (
	"testing"
	"time"
)

func TestParser_SingleFrame(t *testing.T) {
	parser := NewParser()

	// Create a test frame
	frame, err := NewFrame(SREQ, SYS, 0x01, []byte{0xAA, 0xBB})
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	frameBytes := frame.ToBytes()

	// Feed the frame
	parser.Feed(frameBytes)

	// Read the parsed frame
	select {
	case parsed := <-parser.Frames():
		if parsed.Type != frame.Type {
			t.Errorf("Type = %v, want %v", parsed.Type, frame.Type)
		}
		if parsed.Subsystem != frame.Subsystem {
			t.Errorf("Subsystem = %v, want %v", parsed.Subsystem, frame.Subsystem)
		}
		if parsed.CommandID != frame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, frame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for parsed frame")
	}
}

func TestParser_MultipleFrames(t *testing.T) {
	parser := NewParser()

	// Create multiple test frames
	frame1, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA})
	frame2, _ := NewFrame(AREQ, ZDO, 0x02, []byte{0xBB, 0xCC})
	frame3, _ := NewFrame(SRSP, AF, 0x03, []byte{})

	// Concatenate all frames
	allBytes := append(frame1.ToBytes(), frame2.ToBytes()...)
	allBytes = append(allBytes, frame3.ToBytes()...)

	// Feed all frames at once
	parser.Feed(allBytes)

	// Should receive 3 frames
	for i := 0; i < 3; i++ {
		select {
		case <-parser.Frames():
			// Successfully received frame
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for frame %d", i+1)
		}
	}
}

func TestParser_PartialFrames(t *testing.T) {
	parser := NewParser()

	// Create a test frame
	frame, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA, 0xBB, 0xCC})
	frameBytes := frame.ToBytes()

	// Feed frame in chunks
	chunkSize := 3
	for i := 0; i < len(frameBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(frameBytes) {
			end = len(frameBytes)
		}
		parser.Feed(frameBytes[i:end])
	}

	// Should eventually receive the complete frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != frame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, frame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for parsed frame")
	}
}

func TestParser_SOFSynchronization(t *testing.T) {
	parser := NewParser()

	// Create a valid frame
	frame, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA})
	frameBytes := frame.ToBytes()

	// Add garbage before the frame
	garbageBytes := []byte{0x00, 0x11, 0x22, 0x33}
	dataWithGarbage := append(garbageBytes, frameBytes...)

	// Feed data with garbage
	parser.Feed(dataWithGarbage)

	// Should skip garbage and parse the valid frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != frame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, frame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for parsed frame")
	}

	// Should also receive error about skipped bytes
	select {
	case err := <-parser.Errors():
		if err == nil {
			t.Error("expected error about skipped bytes, got nil")
		}
	case <-time.After(100 * time.Millisecond):
		// No error is also acceptable if implementation doesn't report it
	}
}

func TestParser_InvalidChecksum(t *testing.T) {
	parser := NewParser()

	// Create a frame with invalid checksum
	invalidFrame := []byte{0xFE, 0x01, 0x21, 0x01, 0xAA, 0xFF} // Wrong checksum

	parser.Feed(invalidFrame)

	// Should receive an error
	select {
	case err := <-parser.Errors():
		if err == nil {
			t.Error("expected checksum error, got nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for error")
	}
}

func TestParser_EmptyBuffer(t *testing.T) {
	parser := NewParser()

	// Feed empty data
	parser.Feed([]byte{})

	// Should not receive any frames or errors
	select {
	case <-parser.Frames():
		t.Error("unexpected frame from empty buffer")
	case <-parser.Errors():
		// Some implementations may report this, others may not
	case <-time.After(50 * time.Millisecond):
		// Expected: timeout means nothing was parsed
	}
}

func TestParser_IncompleteFrame(t *testing.T) {
	parser := NewParser()

	// Create an incomplete frame (missing FCS byte)
	frame, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA, 0xBB})
	frameBytes := frame.ToBytes()
	incompleteFrame := frameBytes[:len(frameBytes)-1] // Remove last byte (FCS)

	parser.Feed(incompleteFrame)

	// Should not receive a frame yet (waiting for more data)
	select {
	case <-parser.Frames():
		t.Error("unexpected frame from incomplete data")
	case <-time.After(50 * time.Millisecond):
		// Expected: timeout means frame is still buffered
	}

	// Now send the missing byte
	parser.Feed(frameBytes[len(frameBytes)-1:])

	// Should now receive the complete frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != frame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, frame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for parsed frame")
	}
}

func TestParser_MaxDataSize(t *testing.T) {
	parser := NewParser()

	// Create a frame with maximum data size
	maxData := make([]byte, MaxDataSize)
	for i := range maxData {
		maxData[i] = byte(i % 256)
	}

	frame, err := NewFrame(AREQ, AF, 0x10, maxData)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	frameBytes := frame.ToBytes()
	parser.Feed(frameBytes)

	// Should parse successfully
	select {
	case parsed := <-parser.Frames():
		if len(parsed.Data) != MaxDataSize {
			t.Errorf("Data length = %d, want %d", len(parsed.Data), MaxDataSize)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for parsed frame")
	}
}

// TestParserPartialData feeds frame data byte-by-byte to test state machine.
func TestParserPartialData(t *testing.T) {
	parser := NewParser()

	// Create a test frame
	frame, err := NewFrame(SREQ, SYS, 0x01, []byte{0xAA, 0xBB, 0xCC})
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	fullFrame := frame.ToBytes()

	// Feed data byte by byte
	for i, b := range fullFrame {
		parser.Feed([]byte{b})

		// Should not receive frame until all bytes are fed
		if i < len(fullFrame)-1 {
			select {
			case <-parser.Frames():
				t.Fatalf("unexpected frame after byte %d/%d", i+1, len(fullFrame))
			case <-time.After(10 * time.Millisecond):
				// Expected: no frame yet
			}
		}
	}

	// After feeding all bytes, should receive complete frame
	select {
	case parsed := <-parser.Frames():
		if parsed.Type != frame.Type {
			t.Errorf("Type = %v, want %v", parsed.Type, frame.Type)
		}
		if parsed.CommandID != frame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, frame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for complete frame")
	}
}

// TestParserMultiplePartialFeeds tests feeding data in various chunk sizes.
func TestParserMultiplePartialFeeds(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int
	}{
		{"single byte", 1},
		{"two bytes", 2},
		{"three bytes", 3},
		{"five bytes", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()

			// Create multiple frames
			frame1, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA})
			frame2, _ := NewFrame(AREQ, ZDO, 0x02, []byte{0xBB, 0xCC})

			allBytes := append(frame1.ToBytes(), frame2.ToBytes()...)

			// Feed in chunks
			for i := 0; i < len(allBytes); i += tt.chunkSize {
				end := i + tt.chunkSize
				if end > len(allBytes) {
					end = len(allBytes)
				}
				parser.Feed(allBytes[i:end])
			}

			// Should eventually receive both frames
			receivedCount := 0
			timeout := time.After(100 * time.Millisecond)
			for receivedCount < 2 {
				select {
				case <-parser.Frames():
					receivedCount++
				case <-timeout:
					t.Fatalf("received only %d/2 frames", receivedCount)
				}
			}
		})
	}
}

// TestParserErrorRecovery verifies parser can recover from bad data.
func TestParserErrorRecovery(t *testing.T) {
	parser := NewParser()

	// Create a valid frame
	goodFrame, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA})
	goodBytes := goodFrame.ToBytes()

	// Feed bad frame with invalid checksum
	badFrame := []byte{0xFE, 0x01, 0x21, 0x01, 0xAA, 0xFF}
	parser.Feed(badFrame)

	// Should receive error
	select {
	case err := <-parser.Errors():
		if err == nil {
			t.Error("expected error for bad checksum")
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("timeout waiting for error")
	}

	// Feed good frame - parser should recover
	parser.Feed(goodBytes)

	// Should receive good frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != goodFrame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, goodFrame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("parser did not recover - timeout waiting for good frame")
	}
}

// TestParserRecoveryFromGarbageStream tests recovery from continuous garbage.
func TestParserRecoveryFromGarbageStream(t *testing.T) {
	parser := NewParser()

	// Feed continuous garbage without SOF
	garbage := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}
	parser.Feed(garbage)

	// Should receive error about no SOF
	select {
	case err := <-parser.Errors():
		if err == nil {
			t.Error("expected error for garbage data")
		}
	case <-time.After(50 * time.Millisecond):
		// Some implementations might not report this
	}

	// Now feed a valid frame
	goodFrame, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xBB})
	parser.Feed(goodFrame.ToBytes())

	// Should successfully parse the good frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != goodFrame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, goodFrame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("parser did not recover from garbage")
	}
}

// TestParserBufferOverflow tests buffer overflow handling.
func TestParserBufferOverflow(t *testing.T) {
	parser := NewParser()

	// Feed data larger than MaxBufferSize without valid frames
	// This should trigger buffer clearing
	largeGarbage := make([]byte, MaxBufferSize+1000)
	for i := range largeGarbage {
		largeGarbage[i] = 0x00 // No SOF bytes
	}

	parser.Feed(largeGarbage)

	// Should receive buffer overflow error
	select {
	case err := <-parser.Errors():
		if err == nil {
			t.Error("expected buffer overflow error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for buffer overflow error")
	}

	// Parser should still work after overflow - feed valid frame
	goodFrame, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xCC})
	parser.Feed(goodFrame.ToBytes())

	// Should parse successfully
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != goodFrame.CommandID {
			t.Errorf("CommandID = 0x%02X, want 0x%02X", parsed.CommandID, goodFrame.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("parser did not recover after buffer overflow")
	}
}

// TestParserInterleavedGoodAndBadFrames tests mixed valid and invalid frames.
func TestParserInterleavedGoodAndBadFrames(t *testing.T) {
	parser := NewParser()

	// Create frames
	good1, _ := NewFrame(SREQ, SYS, 0x01, []byte{0xAA})
	good2, _ := NewFrame(AREQ, ZDO, 0x02, []byte{0xBB})

	// Bad frame with invalid checksum
	bad := []byte{0xFE, 0x01, 0x21, 0x03, 0xCC, 0xFF}

	// Feed: good1, bad, good2
	parser.Feed(good1.ToBytes())
	parser.Feed(bad)
	parser.Feed(good2.ToBytes())

	// Should receive first good frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != 0x01 {
			t.Errorf("first frame CommandID = 0x%02X, want 0x01", parsed.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for first good frame")
	}

	// Should receive error for bad frame
	select {
	case err := <-parser.Errors():
		if err == nil {
			t.Error("expected error for bad frame")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for error")
	}

	// Should receive second good frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != 0x02 {
			t.Errorf("second frame CommandID = 0x%02X, want 0x02", parsed.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for second good frame")
	}
}

// TestParserFrameWithSOFInPayload tests frames where payload contains SOF byte.
func TestParserFrameWithSOFInPayload(t *testing.T) {
	parser := NewParser()

	// Create frame with SOF byte in payload
	payloadWithSOF := []byte{0x00, SOF, 0xFF, SOF, 0xAA}
	frame, err := NewFrame(SREQ, SYS, 0x42, payloadWithSOF)
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}

	parser.Feed(frame.ToBytes())

	// Should parse correctly despite SOF in payload
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != 0x42 {
			t.Errorf("CommandID = 0x%02X, want 0x42", parsed.CommandID)
		}
		if len(parsed.Data) != len(payloadWithSOF) {
			t.Errorf("data length = %d, want %d", len(parsed.Data), len(payloadWithSOF))
		}
		// Verify payload content
		for i, b := range payloadWithSOF {
			if parsed.Data[i] != b {
				t.Errorf("data[%d] = 0x%02X, want 0x%02X", i, parsed.Data[i], b)
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for frame with SOF in payload")
	}
}

// TestParserIncompleteThenComplete tests feeding incomplete data then completing it.
func TestParserIncompleteThenComplete(t *testing.T) {
	parser := NewParser()

	frame, _ := NewFrame(SREQ, SYS, 0x10, []byte{0xAA, 0xBB, 0xCC, 0xDD})
	frameBytes := frame.ToBytes()

	// Feed only SOF and length
	parser.Feed(frameBytes[:2])

	// Should not receive frame yet
	select {
	case <-parser.Frames():
		t.Error("unexpected frame from incomplete data (SOF+Length only)")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}

	// Feed Cmd0 and Cmd1
	parser.Feed(frameBytes[2:4])

	// Still should not receive frame
	select {
	case <-parser.Frames():
		t.Error("unexpected frame from incomplete data (no payload)")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}

	// Feed partial payload
	parser.Feed(frameBytes[4:6])

	// Still should not receive frame
	select {
	case <-parser.Frames():
		t.Error("unexpected frame from incomplete data (partial payload)")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}

	// Feed rest of payload and checksum
	parser.Feed(frameBytes[6:])

	// Now should receive complete frame
	select {
	case parsed := <-parser.Frames():
		if parsed.CommandID != 0x10 {
			t.Errorf("CommandID = 0x%02X, want 0x10", parsed.CommandID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for complete frame")
	}
}

// TestParserZeroLengthFrame tests parsing of frames with zero data length.
func TestParserZeroLengthFrame(t *testing.T) {
	parser := NewParser()

	// Create frame with zero length
	frame, _ := NewFrame(POLL, SYS, 0x00, []byte{})
	parser.Feed(frame.ToBytes())

	// Should parse successfully
	select {
	case parsed := <-parser.Frames():
		if len(parsed.Data) != 0 {
			t.Errorf("data length = %d, want 0", len(parsed.Data))
		}
		if parsed.Type != POLL {
			t.Errorf("Type = %v, want POLL", parsed.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for zero-length frame")
	}
}

// TestParserConcurrentFrames tests parsing multiple frames fed rapidly.
func TestParserConcurrentFrames(t *testing.T) {
	parser := NewParser()

	// Create 10 frames
	frames := make([]*Frame, 10)
	for i := 0; i < 10; i++ {
		frames[i], _ = NewFrame(SREQ, SYS, uint8(i), []byte{uint8(i * 10)})
	}

	// Feed all frames concatenated
	var allBytes []byte
	for _, f := range frames {
		allBytes = append(allBytes, f.ToBytes()...)
	}
	parser.Feed(allBytes)

	// Should receive all 10 frames
	for i := 0; i < 10; i++ {
		select {
		case <-parser.Frames():
			// Successfully received frame
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for frame %d/10", i+1)
		}
	}

	// Should not have any more frames
	select {
	case <-parser.Frames():
		t.Error("unexpected extra frame")
	case <-time.After(10 * time.Millisecond):
		// Expected: no more frames
	}
}

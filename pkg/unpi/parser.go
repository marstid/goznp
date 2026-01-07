package unpi

import (
	"fmt"
)

// MaxBufferSize is the maximum allowed buffer size before discarding data.
// This prevents memory exhaustion from malformed data streams.
const MaxBufferSize = 8192

// Parser implements a streaming frame parser for UNPI protocol.
// It maintains an internal buffer and extracts complete frames as they arrive.
type Parser struct {
	buffer []byte
	frames chan *Frame
	errors chan error
}

// NewParser creates a new Parser instance with buffered channels.
// The channels are buffered to prevent blocking during frame extraction.
func NewParser() *Parser {
	return &Parser{
		buffer: make([]byte, 0, 512),
		frames: make(chan *Frame, 10),
		errors: make(chan error, 10),
	}
}

// Feed adds incoming bytes to the parser's buffer and attempts to parse complete frames.
// It continuously tries to extract frames until no more complete frames are available.
// If the buffer exceeds MaxBufferSize, it is cleared to prevent memory exhaustion.
func (p *Parser) Feed(data []byte) {
	if len(p.buffer)+len(data) > MaxBufferSize {
		p.errors <- fmt.Errorf("buffer overflow: size %d exceeds limit %d, searching for next frame",
			len(p.buffer)+len(data), MaxBufferSize)

		// Search for next SOF to preserve potential valid frame.
		foundSOF := false
		for i := 1; i < len(p.buffer); i++ {
			if p.buffer[i] == SOF {
				p.buffer = p.buffer[i:]
				foundSOF = true
				break
			}
		}

		// If no SOF found or still too large, clear completely.
		if !foundSOF || len(p.buffer) > MaxBufferSize/2 {
			p.buffer = p.buffer[:0]
		}
	}
	p.buffer = append(p.buffer, data...)
	p.parseNext()
}

// Frames returns a read-only channel that delivers successfully parsed frames.
func (p *Parser) Frames() <-chan *Frame {
	return p.frames
}

// Errors returns a read-only channel that delivers parsing errors.
func (p *Parser) Errors() <-chan error {
	return p.errors
}

// parseNext attempts to extract and parse the next complete frame from the buffer.
// It handles SOF synchronization by skipping invalid bytes until a valid SOF is found.
// This method is called recursively to process multiple frames in the buffer.
func (p *Parser) parseNext() {
	for {
		// Need at least MinMessageLength bytes to attempt parsing.
		if len(p.buffer) < MinMessageLength {
			return
		}

		// Find SOF byte.
		sofIndex := p.findSOF()
		if sofIndex == -1 {
			// No SOF found, discard entire buffer.
			if len(p.buffer) > 0 {
				p.errors <- fmt.Errorf("no SOF found in %d bytes, discarding buffer", len(p.buffer))
				p.buffer = p.buffer[:0]
			}
			return
		}

		// Discard bytes before SOF if any.
		if sofIndex > 0 {
			p.errors <- fmt.Errorf("skipping %d bytes before SOF", sofIndex)
			p.buffer = p.buffer[sofIndex:]
		}

		// Check if we have enough data to read the length field.
		if len(p.buffer) < DataStart {
			return
		}

		// Calculate expected frame size.
		dataLen := int(p.buffer[PositionDataLength])
		expectedFrameSize := MinMessageLength + dataLen

		// Wait for complete frame.
		if len(p.buffer) < expectedFrameSize {
			return
		}

		// Extract frame bytes.
		frameBytes := p.buffer[:expectedFrameSize]

		// Attempt to parse the frame.
		frame, err := ParseFrame(frameBytes)
		if err != nil {
			// Parsing failed, skip this SOF and look for the next one.
			p.errors <- fmt.Errorf("frame parse error: %w", err)
			p.buffer = p.buffer[1:] // Skip the invalid SOF.
			continue
		}

		// Successfully parsed frame.
		p.frames <- frame
		p.buffer = p.buffer[expectedFrameSize:]

		// Continue parsing if more data is available.
		if len(p.buffer) < MinMessageLength {
			return
		}
	}
}

// findSOF searches for the SOF byte in the buffer and returns its index.
// Returns -1 if SOF is not found.
func (p *Parser) findSOF() int {
	for i, b := range p.buffer {
		if b == SOF {
			return i
		}
	}
	return -1
}

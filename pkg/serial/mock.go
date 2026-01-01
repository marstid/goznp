package serial

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// MockPort is a mock implementation of Port for testing.
// It is thread-safe and can simulate various serial port behaviors.
type MockPort struct {
	mu            sync.Mutex
	readBuf       []byte        // Buffer for data to be read
	writeBuf      []byte        // Buffer storing all written data
	closed        bool          // Whether the port is closed
	readTimeout   time.Duration // Read timeout setting
	dtr           bool          // DTR signal state
	rts           bool          // RTS signal state
	onWrite       func([]byte)  // Callback invoked when data is written
	readDeadline  time.Time     // Read deadline for timeout simulation
	writeDeadline time.Time     // Write deadline for timeout simulation
}

// NewMockPort creates a new MockPort for testing.
func NewMockPort() *MockPort {
	return &MockPort{
		readBuf:     make([]byte, 0),
		writeBuf:    make([]byte, 0),
		readTimeout: 100 * time.Millisecond,
		dtr:         true,
		rts:         true,
	}
}

// Read reads data from the mock port's read buffer.
// Implements io.Reader interface.
func (m *MockPort) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.EOF
	}

	if len(m.readBuf) == 0 {
		// Simulate timeout behavior
		if m.readTimeout > 0 {
			// In a real implementation, this would block until timeout
			// For testing, we just return immediately with no data
			return 0, nil
		}
		return 0, io.EOF
	}

	// Copy available data to the caller's buffer
	n := copy(p, m.readBuf)
	m.readBuf = m.readBuf[n:]
	return n, nil
}

// Write writes data to the mock port and invokes the OnWrite callback if set.
// Implements io.Writer interface.
func (m *MockPort) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, fmt.Errorf("mock serial: port is closed")
	}

	// Store written data
	m.writeBuf = append(m.writeBuf, p...)
	n := len(p)

	// Invoke callback if set (outside lock to avoid deadlock)
	if m.onWrite != nil {
		callback := m.onWrite
		dataCopy := make([]byte, len(p))
		copy(dataCopy, p)
		m.mu.Unlock()
		callback(dataCopy)
		m.mu.Lock()
	}

	return n, nil
}

// Close closes the mock port.
// Implements io.Closer interface.
func (m *MockPort) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock serial: port already closed")
	}

	m.closed = true
	return nil
}

// SetReadTimeout sets the read timeout for the mock port.
func (m *MockPort) SetReadTimeout(timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock serial: port is closed")
	}

	m.readTimeout = timeout
	return nil
}

// SetDTR sets the Data Terminal Ready signal state.
func (m *MockPort) SetDTR(dtr bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock serial: port is closed")
	}

	m.dtr = dtr
	return nil
}

// SetRTS sets the Request To Send signal state.
func (m *MockPort) SetRTS(rts bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock serial: port is closed")
	}

	m.rts = rts
	return nil
}

// ResetInputBuffer clears the read buffer.
func (m *MockPort) ResetInputBuffer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock serial: port is closed")
	}

	m.readBuf = make([]byte, 0)
	return nil
}

// SimulateResponse injects bytes into the read buffer for testing.
// This simulates data arriving from a real serial device.
func (m *MockPort) SimulateResponse(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readBuf = append(m.readBuf, data...)
}

// WrittenData returns all data that has been written to the mock port.
// This is useful for verifying what was sent in tests.
func (m *MockPort) WrittenData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent external modification
	result := make([]byte, len(m.writeBuf))
	copy(result, m.writeBuf)
	return result
}

// ClearWrittenData clears the write buffer.
// Useful for resetting state between test cases.
func (m *MockPort) ClearWrittenData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writeBuf = make([]byte, 0)
}

// OnWrite sets a callback function that is invoked whenever data is written.
// This is useful for simulating device responses in tests.
// The callback receives a copy of the written data.
//
// Example:
//
//	mock.OnWrite(func(data []byte) {
//	    if isCommandFrame(data) {
//	        mock.SimulateResponse(createResponse(data))
//	    }
//	})
func (m *MockPort) OnWrite(callback func([]byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onWrite = callback
}

// SetClosed simulates a disconnection by marking the port as closed.
// Subsequent operations will fail with appropriate errors.
func (m *MockPort) SetClosed() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
}

// IsClosed returns whether the mock port is closed.
func (m *MockPort) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closed
}

// GetDTR returns the current DTR signal state.
func (m *MockPort) GetDTR() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.dtr
}

// GetRTS returns the current RTS signal state.
func (m *MockPort) GetRTS() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.rts
}

// GetReadTimeout returns the current read timeout setting.
func (m *MockPort) GetReadTimeout() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.readTimeout
}

// ReadBufLen returns the current length of the read buffer.
// Useful for testing buffer management.
func (m *MockPort) ReadBufLen() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.readBuf)
}

// WriteBufLen returns the current length of the write buffer.
// Useful for testing write operations.
func (m *MockPort) WriteBufLen() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.writeBuf)
}

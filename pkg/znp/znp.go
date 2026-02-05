package znp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// Constants for ZNP operation timeouts and retry logic.
const (
	// DefaultSREQTimeout is the default timeout for synchronous requests.
	DefaultSREQTimeout = 6 * time.Second
	// DefaultResetTimeout is the default timeout for waiting for reset indication.
	DefaultResetTimeout = 30 * time.Second
	// PingRetries is the number of ping attempts during handshake.
	PingRetries = 3
)

// Sentinel errors for ZNP operations.
var (
	// ErrTimeout indicates a request timed out waiting for a response.
	ErrTimeout = errors.New("request timeout")
	// ErrNotOpen indicates the ZNP client is not open.
	ErrNotOpen = errors.New("port not open")
	// ErrPingFailed indicates the ping handshake failed.
	ErrPingFailed = errors.New("ping handshake failed")
)

// ZNP represents the Zigbee Network Processor client.
// It manages communication with the ZNP device over a serial port.
type ZNP struct {
	// port is the underlying I/O connection to the ZNP device.
	port io.ReadWriteCloser
	// parser processes incoming bytes and extracts UNPI frames.
	parser *unpi.Parser
	// waiter manages pending request-response matching.
	waiter *Waiter

	// mu protects the isOpen flag and callbacks.
	mu     sync.Mutex
	isOpen bool
	ctx    context.Context // Stores the context from Open()

	// onFrame is an optional callback for frames not matched to pending requests.
	onFrame func(*unpi.Frame)

	// onError is an optional callback for error events.
	onError func(error)

	// Device event callbacks
	onDeviceJoin     func(*TcDeviceInd)
	onDeviceLeave    func(*DeviceLeave)
	onDeviceAnnounce func(*DeviceAnnounce)

	// stopReader signals the read/frame loops to stop.
	stopReader chan struct{}
	// wg tracks active goroutines for clean shutdown.
	wg sync.WaitGroup
}

// New creates a new ZNP client instance with the given port.
// The port should be an open serial port or other ReadWriteCloser.
// Call Open to start communication.
func New(port io.ReadWriteCloser) *ZNP {
	return &ZNP{
		port:       port,
		parser:     unpi.NewParser(),
		waiter:     NewWaiter(),
		stopReader: make(chan struct{}),
	}
}

// Open starts the ZNP client's read and frame processing loops.
// It launches background goroutines to handle incoming data.
func (z *ZNP) Open(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	z.mu.Lock()
	if z.isOpen {
		z.mu.Unlock()
		return nil
	}
	z.isOpen = true
	z.ctx = ctx
	z.mu.Unlock()

	z.wg.Add(1)
	go z.readLoop()

	z.wg.Add(1)
	go z.frameLoop()

	return nil
}

// Close stops the ZNP client and closes the underlying port.
// It waits for all background goroutines to complete.
func (z *ZNP) Close() error {
	z.mu.Lock()
	if !z.isOpen {
		z.mu.Unlock()
		return nil
	}
	z.isOpen = false
	z.ctx = nil
	z.mu.Unlock()

	close(z.stopReader)
	z.wg.Wait()

	if z.port != nil {
		return z.port.Close()
	}

	return nil
}

// Request sends a synchronous request (SREQ) and waits for the corresponding response (SRSP).
// It automatically creates a frame matcher for the expected SRSP response.
// Returns the response frame or an error if the request times out or the context is canceled.
func (z *ZNP) Request(ctx context.Context, subsystem unpi.Subsystem, cmd Command, data []byte) (*unpi.Frame, error) {
	z.mu.Lock()
	if !z.isOpen {
		z.mu.Unlock()
		return nil, ErrNotOpen
	}
	z.mu.Unlock()

	// Create SREQ frame
	frame, err := unpi.NewFrame(unpi.SREQ, subsystem, cmd.ID, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create frame: %w", err)
	}

	// Create matcher for expected SRSP response
	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: subsystem,
		CommandID: cmd.ID,
	}

	// Send the frame FIRST to avoid goroutine leak if write fails
	frameBytes := frame.ToBytes()
	if _, err := z.port.Write(frameBytes); err != nil {
		return nil, fmt.Errorf("failed to write frame: %w", err)
	}

	// Now start waiting for response (only after successful write)
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	responseChan := make(chan *unpi.Frame, 1)
	errorChan := make(chan error, 1)

	go func() {
		resp, err := z.waiter.WaitFor(waitCtx, matcher, DefaultSREQTimeout)
		if err != nil {
			select {
			case errorChan <- err:
			case <-waitCtx.Done():
			}
		} else {
			select {
			case responseChan <- resp:
			case <-waitCtx.Done():
			}
		}
	}()

	// Wait for response or error
	select {
	case resp := <-responseChan:
		return resp, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Send sends an asynchronous request (AREQ) without waiting for a response.
func (z *ZNP) Send(subsystem unpi.Subsystem, cmd Command, data []byte) error {
	z.mu.Lock()
	if !z.isOpen {
		z.mu.Unlock()
		return ErrNotOpen
	}
	z.mu.Unlock()

	frame, err := unpi.NewFrame(unpi.AREQ, subsystem, cmd.ID, data)
	if err != nil {
		return fmt.Errorf("failed to create frame: %w", err)
	}

	if _, err := z.port.Write(frame.ToBytes()); err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}

	return nil
}

// OnFrame sets a callback function to handle frames that are not matched to pending requests.
// This is useful for handling asynchronous notifications from the ZNP device.
func (z *ZNP) OnFrame(handler func(*unpi.Frame)) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.onFrame = handler
}

// OnDeviceJoin sets a callback for device join events (tcDeviceInd).
func (z *ZNP) OnDeviceJoin(handler func(*TcDeviceInd)) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.onDeviceJoin = handler
}

// OnDeviceLeave sets a callback for device leave events (leaveInd).
func (z *ZNP) OnDeviceLeave(handler func(*DeviceLeave)) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.onDeviceLeave = handler
}

// OnDeviceAnnounce sets a callback for device announcement events (endDeviceAnnceInd).
// This is triggered when a device announces itself on the network (join or rejoin).
func (z *ZNP) OnDeviceAnnounce(handler func(*DeviceAnnounce)) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.onDeviceAnnounce = handler
}

// OnError sets a callback for error events from the parser and read operations.
// This is useful for logging and monitoring errors that would otherwise be lost.
func (z *ZNP) OnError(handler func(error)) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.onError = handler
}

// reportError reports an error to the error handler if set.
func (z *ZNP) reportError(err error) {
	z.mu.Lock()
	handler := z.onError
	z.mu.Unlock()

	if handler != nil {
		handler(err)
	}
}

// readLoop continuously reads from the port and feeds data to the parser.
// It runs in a background goroutine until the ZNP client is closed.
func (z *ZNP) readLoop() {
	defer z.wg.Done()

	buf := make([]byte, 512)
	for {
		select {
		case <-z.stopReader:
			return
		default:
			// Read from port with timeout to allow checking stop signal
			n, err := z.port.Read(buf)
			if err != nil {
				if err != io.EOF {
					z.reportError(fmt.Errorf("read error: %w", err))
				}
				select {
				case <-z.stopReader:
					return
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}

			if n > 0 {
				z.parser.Feed(buf[:n])
			}
		}
	}
}

// frameLoop processes parsed frames and either resolves pending waiters or calls the frame handler.
// It runs in a background goroutine until the ZNP client is closed.
func (z *ZNP) frameLoop() {
	defer z.wg.Done()

	for {
		select {
		case <-z.stopReader:
			return
		case frame := <-z.parser.Frames():
			// Try to resolve pending waiter
			if !z.waiter.Resolve(frame) {
				// Check for device events
				z.handleDeviceEvents(frame)

				// Also call generic frame handler if set
				z.mu.Lock()
				handler := z.onFrame
				z.mu.Unlock()

				if handler != nil {
					handler(frame)
				}
			}
		case err := <-z.parser.Errors():
			z.reportError(fmt.Errorf("parser error: %w", err))
		}
	}
}

// handleDeviceEvents checks for and dispatches device join/leave/announce events.
func (z *ZNP) handleDeviceEvents(frame *unpi.Frame) {
	// Only handle ZDO AREQ frames
	if frame.Type != unpi.AREQ || frame.Subsystem != unpi.ZDO {
		return
	}

	z.mu.Lock()
	ctx := z.ctx
	joinHandler := z.onDeviceJoin
	leaveHandler := z.onDeviceLeave
	announceHandler := z.onDeviceAnnounce
	z.mu.Unlock()

	switch frame.CommandID {
	case CmdZdoTcDevInd.ID:
		if joinHandler != nil {
			if ind, err := ParseTcDeviceInd(frame.Data); err == nil {
				z.wg.Add(1)
				go func(ind *TcDeviceInd) {
					defer z.wg.Done()
					if ctx != nil {
						select {
						case <-ctx.Done():
							return
						default:
						}
					}
					joinHandler(ind)
				}(ind)
			}
		}
	case CmdZdoLeaveInd.ID:
		if leaveHandler != nil {
			if ind, err := ParseLeaveInd(frame.Data); err == nil {
				z.wg.Add(1)
				go func(ind *DeviceLeave) {
					defer z.wg.Done()
					if ctx != nil {
						select {
						case <-ctx.Done():
							return
						default:
						}
					}
					leaveHandler(ind)
				}(ind)
			}
		}
	case CmdZdoEndDeviceAnnceInd.ID:
		if announceHandler != nil {
			if ind, err := ParseEndDeviceAnnceInd(frame.Data); err == nil {
				z.wg.Add(1)
				go func(ind *DeviceAnnounce) {
					defer z.wg.Done()
					if ctx != nil {
						select {
						case <-ctx.Done():
							return
						default:
						}
					}
					announceHandler(ind)
				}(ind)
			}
		}
	}
}

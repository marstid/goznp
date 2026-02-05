package znp

import (
	"context"
	"sync"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// FrameMatcher defines criteria for matching frames.
// It is used to identify which incoming frame should resolve a pending request.
type FrameMatcher struct {
	// Type is the expected frame type (e.g., SRSP).
	Type unpi.Type
	// Subsystem is the expected subsystem.
	Subsystem unpi.Subsystem
	// CommandID is the expected command ID.
	CommandID uint8
}

// Matches returns true if the given frame satisfies this matcher's criteria.
func (m FrameMatcher) Matches(frame *unpi.Frame) bool {
	return frame.Type == m.Type &&
		frame.Subsystem == m.Subsystem &&
		frame.CommandID == m.CommandID
}

// pendingRequest represents a request waiting for a matching response frame.
type pendingRequest struct {
	// matcher defines the criteria for the expected response.
	matcher FrameMatcher
	// response is the channel where the matching frame will be delivered.
	response chan *unpi.Frame
	// done is closed when the request is completed or canceled.
	done chan struct{}
}

// Waiter manages pending request waiters and matches incoming frames to them.
// It is thread-safe and can be used concurrently from multiple goroutines.
// It uses an indexed structure for O(1) frame matching instead of O(n) linear search.
type Waiter struct {
	mu sync.Mutex
	// waiters is indexed by FrameMatcher for O(1) lookups, then by ID for multiple
	// concurrent requests waiting for the same frame type.
	waiters map[FrameMatcher]map[uint64]*pendingRequest
	nextID  uint64
}

// NewWaiter creates a new Waiter instance.
func NewWaiter() *Waiter {
	return &Waiter{
		waiters: make(map[FrameMatcher]map[uint64]*pendingRequest),
		nextID:  1,
	}
}

// WaitFor registers a waiter for a frame matching the given criteria.
// It blocks until a matching frame arrives, the timeout expires, or the context is canceled.
// Returns the matching frame or an error if the wait was interrupted.
func (w *Waiter) WaitFor(ctx context.Context, matcher FrameMatcher, timeout time.Duration) (*unpi.Frame, error) {
	// Create pending request
	req := &pendingRequest{
		matcher:  matcher,
		response: make(chan *unpi.Frame, 1),
		done:     make(chan struct{}),
	}

	// Register the waiter and get its ID
	w.mu.Lock()
	id := w.nextID
	w.nextID++
	// Initialize the inner map if needed
	if w.waiters[matcher] == nil {
		w.waiters[matcher] = make(map[uint64]*pendingRequest)
	}
	w.waiters[matcher][id] = req
	w.mu.Unlock()

	// Ensure cleanup on exit
	defer func() {
		close(req.done)
		w.removeWaiter(matcher, id)
	}()

	// Create timeout timer
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Wait for response, timeout, or cancellation
	select {
	case frame := <-req.response:
		return frame, nil
	case <-timer.C:
		return nil, ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Resolve tries to match the incoming frame to a pending waiter.
// Returns true if the frame was matched and delivered to at least one waiter.
// Returns false if no matching waiter was found.
// Uses O(1) lookup by FrameMatcher index.
func (w *Waiter) Resolve(frame *unpi.Frame) bool {
	// Build matcher from frame for O(1) lookup
	matcher := FrameMatcher{
		Type:      frame.Type,
		Subsystem: frame.Subsystem,
		CommandID: frame.CommandID,
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// O(1) lookup by matcher
	requests, ok := w.waiters[matcher]
	if !ok || len(requests) == 0 {
		return false
	}

	// Try to deliver to one of the matching waiters (first available)
	delivered := false
	for id, req := range requests {
		select {
		case req.response <- frame:
			// Successfully delivered
			delete(requests, id)
			delivered = true
			// Clean up empty matcher entry
			if len(requests) == 0 {
				delete(w.waiters, matcher)
			}
			goto done // Only deliver to one waiter per frame
		case <-req.done:
			// Waiter already completed, remove it
			delete(requests, id)
		}
	}
done:

	// Clean up empty matcher entry
	if len(requests) == 0 {
		delete(w.waiters, matcher)
	}

	return delivered
}

// removeWaiter removes a waiter by its matcher and ID.
// This is called internally to clean up after a waiter completes.
func (w *Waiter) removeWaiter(matcher FrameMatcher, id uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if requests, ok := w.waiters[matcher]; ok {
		delete(requests, id)
		// Clean up empty matcher entry
		if len(requests) == 0 {
			delete(w.waiters, matcher)
		}
	}
}

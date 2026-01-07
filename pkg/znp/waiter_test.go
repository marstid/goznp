package znp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/unpi"
)

// TestWaiterBasicResolve tests that a matching frame resolves a pending waiter.
func TestWaiterBasicResolve(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	// Start waiting in goroutine
	var result *unpi.Frame
	var err error
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result, err = w.WaitFor(ctx, matcher, time.Second)
		close(done)
	}()

	// Give waiter time to register
	time.Sleep(10 * time.Millisecond)

	// Resolve with matching frame
	frame := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
		Data:      []byte{0x00},
	}
	resolved := w.Resolve(frame)

	<-done

	if !resolved {
		t.Error("expected frame to resolve waiter")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected result frame")
	}
	if result != nil && result.CommandID != 0x01 {
		t.Errorf("expected CommandID 0x01, got 0x%02X", result.CommandID)
	}
}

// TestWaiterTimeout tests that WaitFor times out correctly.
func TestWaiterTimeout(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0xFF, // Use unlikely command ID
	}

	start := time.Now()
	ctx := context.Background()
	result, err := w.WaitFor(ctx, matcher, 50*time.Millisecond)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("timeout duration unexpected: %v (expected ~50ms)", elapsed)
	}
}

// TestWaiterContextCancellation tests that canceling context stops waiting.
func TestWaiterContextCancellation(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0xFF,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start waiting in goroutine
	var err error
	done := make(chan struct{})
	go func() {
		_, err = w.WaitFor(ctx, matcher, time.Second)
		close(done)
	}()

	// Give waiter time to register
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	start := time.Now()
	cancel()
	<-done
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// Should be nearly instant
	if elapsed > 100*time.Millisecond {
		t.Errorf("cancellation took too long: %v", elapsed)
	}
}

// TestWaiterMultiplePending tests multiple concurrent waiters for different frames.
func TestWaiterMultiplePending(t *testing.T) {
	w := NewWaiter()

	// Create multiple matchers
	matcher1 := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}
	matcher2 := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x02,
	}
	matcher3 := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.UTIL,
		CommandID: 0x01,
	}

	var wg sync.WaitGroup
	results := make([]*unpi.Frame, 3)
	errors := make([]error, 3)

	// Start three waiters
	for i := 0; i < 3; i++ {
		wg.Add(1)
		idx := i
		var matcher FrameMatcher
		switch idx {
		case 0:
			matcher = matcher1
		case 1:
			matcher = matcher2
		case 2:
			matcher = matcher3
		}

		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			results[idx], errors[idx] = w.WaitFor(ctx, matcher, time.Second)
		}()
	}

	// Give waiters time to register (increased for parallel test runs).
	time.Sleep(100 * time.Millisecond)

	// Resolve each frame in reverse order
	frame3 := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.UTIL,
		CommandID: 0x01,
		Data:      []byte{0x03},
	}
	frame2 := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x02,
		Data:      []byte{0x02},
	}
	frame1 := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
		Data:      []byte{0x01},
	}

	w.Resolve(frame3)
	time.Sleep(5 * time.Millisecond)
	w.Resolve(frame2)
	time.Sleep(5 * time.Millisecond)
	w.Resolve(frame1)

	wg.Wait()

	// Check all waiters got their frames
	for i := 0; i < 3; i++ {
		if errors[i] != nil {
			t.Errorf("waiter %d: unexpected error: %v", i, errors[i])
		}
		if results[i] == nil {
			t.Errorf("waiter %d: expected result frame", i)
			continue
		}
		expectedData := []byte{byte(i + 1)}
		if len(results[i].Data) != 1 || results[i].Data[0] != expectedData[0] {
			t.Errorf("waiter %d: expected data %v, got %v", i, expectedData, results[i].Data)
		}
	}
}

// TestWaiterNoMatch tests that non-matching frames don't resolve waiters.
func TestWaiterNoMatch(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	// Start waiting in goroutine
	var err error
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err = w.WaitFor(ctx, matcher, 100*time.Millisecond)
		close(done)
	}()

	// Give waiter time to register
	time.Sleep(10 * time.Millisecond)

	// Try to resolve with non-matching frames
	nonMatchingFrames := []*unpi.Frame{
		// Wrong type
		{
			Type:      unpi.AREQ,
			Subsystem: unpi.SYS,
			CommandID: 0x01,
			Data:      []byte{0x00},
		},
		// Wrong subsystem
		{
			Type:      unpi.SRSP,
			Subsystem: unpi.UTIL,
			CommandID: 0x01,
			Data:      []byte{0x00},
		},
		// Wrong command ID
		{
			Type:      unpi.SRSP,
			Subsystem: unpi.SYS,
			CommandID: 0x02,
			Data:      []byte{0x00},
		},
	}

	for i, frame := range nonMatchingFrames {
		resolved := w.Resolve(frame)
		if resolved {
			t.Errorf("frame %d should not have resolved waiter", i)
		}
	}

	<-done

	// Should have timed out since no matching frame was sent
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
}

// TestWaiterMultipleSameMatch tests multiple concurrent waiters for the same frame type.
func TestWaiterMultipleSameMatch(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	var wg sync.WaitGroup
	results := make([]*unpi.Frame, 3)
	errors := make([]error, 3)

	// Start three waiters for the same frame type
	for i := 0; i < 3; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			results[idx], errors[idx] = w.WaitFor(ctx, matcher, time.Second)
		}()
	}

	// Give waiters time to register (increased for parallel test runs).
	time.Sleep(100 * time.Millisecond)

	// Send three matching frames (one for each waiter)
	for i := 0; i < 3; i++ {
		frame := &unpi.Frame{
			Type:      unpi.SRSP,
			Subsystem: unpi.SYS,
			CommandID: 0x01,
			Data:      []byte{byte(i)},
		}
		resolved := w.Resolve(frame)
		if !resolved {
			t.Errorf("frame %d should have resolved a waiter", i)
		}
		time.Sleep(5 * time.Millisecond)
	}

	wg.Wait()

	// Check all waiters got a frame (order may vary)
	for i := 0; i < 3; i++ {
		if errors[i] != nil {
			t.Errorf("waiter %d: unexpected error: %v", i, errors[i])
		}
		if results[i] == nil {
			t.Errorf("waiter %d: expected result frame", i)
		}
	}
}

// TestWaiterResolveBeforeWait tests that Resolve returns false when called before any waiters.
func TestWaiterResolveBeforeWait(t *testing.T) {
	w := NewWaiter()

	frame := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
		Data:      []byte{0x00},
	}

	resolved := w.Resolve(frame)
	if resolved {
		t.Error("Resolve should return false when no waiters are pending")
	}
}

// TestFrameMatcher tests the FrameMatcher.Matches method.
func TestFrameMatcher(t *testing.T) {
	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	tests := []struct {
		name    string
		frame   *unpi.Frame
		matches bool
	}{
		{
			name: "exact match",
			frame: &unpi.Frame{
				Type:      unpi.SRSP,
				Subsystem: unpi.SYS,
				CommandID: 0x01,
				Data:      []byte{0x00},
			},
			matches: true,
		},
		{
			name: "wrong type",
			frame: &unpi.Frame{
				Type:      unpi.AREQ,
				Subsystem: unpi.SYS,
				CommandID: 0x01,
				Data:      []byte{0x00},
			},
			matches: false,
		},
		{
			name: "wrong subsystem",
			frame: &unpi.Frame{
				Type:      unpi.SRSP,
				Subsystem: unpi.UTIL,
				CommandID: 0x01,
				Data:      []byte{0x00},
			},
			matches: false,
		},
		{
			name: "wrong command ID",
			frame: &unpi.Frame{
				Type:      unpi.SRSP,
				Subsystem: unpi.SYS,
				CommandID: 0x02,
				Data:      []byte{0x00},
			},
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Matches(tt.frame)
			if result != tt.matches {
				t.Errorf("Matches() = %v, want %v", result, tt.matches)
			}
		})
	}
}

// TestWaiterCleanup tests that waiters are cleaned up after completion.
func TestWaiterCleanup(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	// Start and resolve a waiter
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		//nolint:errcheck // Error intentionally ignored in test
		w.WaitFor(ctx, matcher, time.Second)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)

	frame := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
		Data:      []byte{0x00},
	}
	w.Resolve(frame)
	<-done

	// Verify waiter was cleaned up by checking that another resolve returns false
	resolved := w.Resolve(frame)
	if resolved {
		t.Error("waiter should have been cleaned up after resolution")
	}

	// Verify internal state is clean
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.waiters) != 0 {
		t.Errorf("expected empty waiters map, got %d entries", len(w.waiters))
	}
}

// TestWaiterConcurrentAccess tests concurrent access to the Waiter (race detector test).
func TestWaiterConcurrentAccess(t *testing.T) {
	w := NewWaiter()

	var wg sync.WaitGroup
	numWaiters := 20
	numResolvers := 10

	// Start multiple waiters
	for i := 0; i < numWaiters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			matcher := FrameMatcher{
				Type:      unpi.SRSP,
				Subsystem: unpi.SYS,
				CommandID: uint8(id % 5), // Use 5 different command IDs
			}
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			//nolint:errcheck // Error intentionally ignored in test
			w.WaitFor(ctx, matcher, 200*time.Millisecond)
		}(i)
	}

	// Start multiple resolvers
	for i := 0; i < numResolvers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			for j := 0; j < 5; j++ {
				frame := &unpi.Frame{
					Type:      unpi.SRSP,
					Subsystem: unpi.SYS,
					CommandID: uint8(j),
					Data:      []byte{byte(id)},
				}
				w.Resolve(frame)
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify cleanup
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.waiters) != 0 {
		t.Errorf("expected empty waiters map after all operations, got %d entries", len(w.waiters))
	}
}

// TestWaiterWithZeroTimeout tests WaitFor with zero timeout.
func TestWaiterWithZeroTimeout(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	ctx := context.Background()
	start := time.Now()
	_, err := w.WaitFor(ctx, matcher, 0)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
	// Zero timeout should return immediately
	if elapsed > 50*time.Millisecond {
		t.Errorf("zero timeout took too long: %v", elapsed)
	}
}

// TestWaiterResolveAfterTimeout tests that a waiter is cleaned up after timeout.
func TestWaiterResolveAfterTimeout(t *testing.T) {
	w := NewWaiter()

	matcher := FrameMatcher{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
	}

	ctx := context.Background()
	_, err := w.WaitFor(ctx, matcher, 50*time.Millisecond)

	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}

	// Try to resolve after timeout
	frame := &unpi.Frame{
		Type:      unpi.SRSP,
		Subsystem: unpi.SYS,
		CommandID: 0x01,
		Data:      []byte{0x00},
	}
	resolved := w.Resolve(frame)
	if resolved {
		t.Error("should not resolve waiter after timeout")
	}
}

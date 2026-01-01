package adapter

import (
	"testing"
	"time"
)

// TestDedupeCache verifies basic deduplication functionality.
func TestDedupeCache(t *testing.T) {
	cache := newDedupeCache(100 * time.Millisecond)

	// First message should not be duplicate
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("first message should not be duplicate")
	}

	// Same message immediately should be duplicate
	if !cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("same message should be duplicate")
	}

	// Different seqnum should not be duplicate
	if cache.isDuplicate(0x1234, 0x0006, 2) {
		t.Error("different seqnum should not be duplicate")
	}

	// After window expires, should not be duplicate
	time.Sleep(150 * time.Millisecond)
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("expired message should not be duplicate")
	}
}

// TestDedupeCacheDifferentAddresses verifies messages from different sources are tracked separately.
func TestDedupeCacheDifferentAddresses(t *testing.T) {
	cache := newDedupeCache(100 * time.Millisecond)

	// Message from addr 0x1234
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("first message from 0x1234 should not be duplicate")
	}

	// Same seqnum but different address should not be duplicate
	if cache.isDuplicate(0x5678, 0x0006, 1) {
		t.Error("message from different address should not be duplicate")
	}

	// Original message should still be duplicate
	if !cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("repeated message from 0x1234 should be duplicate")
	}

	// Second address message should now be duplicate
	if !cache.isDuplicate(0x5678, 0x0006, 1) {
		t.Error("repeated message from 0x5678 should be duplicate")
	}
}

// TestDedupeCacheDifferentClusters verifies messages from different clusters are tracked separately.
func TestDedupeCacheDifferentClusters(t *testing.T) {
	cache := newDedupeCache(100 * time.Millisecond)

	// Message with cluster 0x0006
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("first message with cluster 0x0006 should not be duplicate")
	}

	// Same address and seqnum but different cluster should not be duplicate
	if cache.isDuplicate(0x1234, 0x0008, 1) {
		t.Error("message with different cluster should not be duplicate")
	}

	// Original cluster message should still be duplicate
	if !cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("repeated message with cluster 0x0006 should be duplicate")
	}

	// Second cluster message should now be duplicate
	if !cache.isDuplicate(0x1234, 0x0008, 1) {
		t.Error("repeated message with cluster 0x0008 should be duplicate")
	}
}

// TestDedupeCacheSequenceNumbers verifies that different sequence numbers are tracked separately.
func TestDedupeCacheSequenceNumbers(t *testing.T) {
	cache := newDedupeCache(100 * time.Millisecond)

	// Record multiple sequence numbers
	for i := uint8(0); i < 5; i++ {
		if cache.isDuplicate(0x1234, 0x0006, i) {
			t.Errorf("first occurrence of seqnum %d should not be duplicate", i)
		}
	}

	// All should now be duplicates
	for i := uint8(0); i < 5; i++ {
		if !cache.isDuplicate(0x1234, 0x0006, i) {
			t.Errorf("second occurrence of seqnum %d should be duplicate", i)
		}
	}
}

// TestDedupeCacheExpiration verifies that old entries are removed.
func TestDedupeCacheExpiration(t *testing.T) {
	window := 50 * time.Millisecond
	cache := newDedupeCache(window)

	// Record a message
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("first message should not be duplicate")
	}

	// Should be duplicate immediately
	if !cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("immediate repeat should be duplicate")
	}

	// Wait for expiration
	time.Sleep(window + 10*time.Millisecond)

	// Should not be duplicate after expiration
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("message should not be duplicate after window expires")
	}
}

// TestDedupeCacheCleanup verifies that expired entries are cleaned up.
func TestDedupeCacheCleanup(t *testing.T) {
	window := 50 * time.Millisecond
	cache := newDedupeCache(window)

	// Add multiple messages
	for i := uint8(0); i < 10; i++ {
		cache.isDuplicate(0x1234, 0x0006, i)
	}

	// Cache should have entries
	cache.mu.Lock()
	initialCount := len(cache.entries)
	cache.mu.Unlock()

	if initialCount != 10 {
		t.Errorf("expected 10 entries, got %d", initialCount)
	}

	// Wait for expiration
	time.Sleep(window + 10*time.Millisecond)

	// Trigger cleanup by checking any message
	cache.isDuplicate(0x5678, 0x0008, 99)

	// Cache should have only the new entry
	cache.mu.Lock()
	finalCount := len(cache.entries)
	cache.mu.Unlock()

	if finalCount != 1 {
		t.Errorf("expected 1 entry after cleanup, got %d", finalCount)
	}
}

// TestDedupeCacheZeroWindow verifies behavior with zero window (no deduplication).
func TestDedupeCacheZeroWindow(t *testing.T) {
	cache := newDedupeCache(0)

	// First message
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("first message should not be duplicate")
	}

	// With zero window, immediate repeat should also not be duplicate
	// because the entry expires immediately
	if cache.isDuplicate(0x1234, 0x0006, 1) {
		t.Error("with zero window, message should not be duplicate")
	}
}

// TestDedupeCacheConcurrency verifies basic thread safety.
func TestDedupeCacheConcurrency(t *testing.T) {
	cache := newDedupeCache(100 * time.Millisecond)

	// Launch multiple goroutines checking the same message
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			cache.isDuplicate(0x1234, 0x0006, 1)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and cache should be consistent
	cache.mu.Lock()
	count := len(cache.entries)
	cache.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 entry, got %d", count)
	}
}

// TestDedupeCacheMultipleMessages verifies tracking multiple distinct messages.
func TestDedupeCacheMultipleMessages(t *testing.T) {
	cache := newDedupeCache(200 * time.Millisecond)

	// Create a mix of unique and duplicate messages
	testCases := []struct {
		addr      uint16
		cluster   uint16
		seqnum    uint8
		duplicate bool
	}{
		{0x1234, 0x0006, 1, false}, // First message
		{0x1234, 0x0006, 1, true},  // Duplicate
		{0x1234, 0x0006, 2, false}, // Different seqnum
		{0x5678, 0x0006, 1, false}, // Different addr
		{0x1234, 0x0008, 1, false}, // Different cluster
		{0x5678, 0x0006, 1, true},  // Duplicate of addr 0x5678
		{0x1234, 0x0008, 1, true},  // Duplicate of cluster 0x0008
	}

	for i, tc := range testCases {
		result := cache.isDuplicate(tc.addr, tc.cluster, tc.seqnum)
		if result != tc.duplicate {
			t.Errorf("case %d (addr=%04X, cluster=%04X, seq=%d): expected duplicate=%v, got %v",
				i, tc.addr, tc.cluster, tc.seqnum, tc.duplicate, result)
		}
	}
}

// TestNewDedupeCache verifies cache initialization.
func TestNewDedupeCache(t *testing.T) {
	window := 500 * time.Millisecond
	cache := newDedupeCache(window)

	if cache == nil {
		t.Fatal("newDedupeCache returned nil")
	}

	if cache.window != window {
		t.Errorf("expected window %v, got %v", window, cache.window)
	}

	if cache.entries == nil {
		t.Error("cache entries map should be initialized")
	}

	cache.mu.Lock()
	count := len(cache.entries)
	cache.mu.Unlock()

	if count != 0 {
		t.Errorf("new cache should be empty, got %d entries", count)
	}
}

// TestDedupeKeyUniqueness verifies that dedupeKey correctly identifies unique messages.
func TestDedupeKeyUniqueness(t *testing.T) {
	cache := newDedupeCache(100 * time.Millisecond)

	// These should all be treated as different messages
	keys := []struct {
		addr    uint16
		cluster uint16
		seqnum  uint8
	}{
		{0x0000, 0x0000, 0},
		{0x0000, 0x0000, 1},
		{0x0000, 0x0001, 0},
		{0x0001, 0x0000, 0},
		{0xFFFF, 0xFFFF, 0xFF},
	}

	for _, k := range keys {
		if cache.isDuplicate(k.addr, k.cluster, k.seqnum) {
			t.Errorf("key (addr=%04X, cluster=%04X, seq=%d) should be unique on first call",
				k.addr, k.cluster, k.seqnum)
		}
	}

	// All should now be duplicates
	for _, k := range keys {
		if !cache.isDuplicate(k.addr, k.cluster, k.seqnum) {
			t.Errorf("key (addr=%04X, cluster=%04X, seq=%d) should be duplicate on second call",
				k.addr, k.cluster, k.seqnum)
		}
	}
}

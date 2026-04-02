package oauth

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiterBasic(t *testing.T) {
	// 10 requests per 100ms window
	rl := NewRateLimiter(100*time.Millisecond, 10)

	// First 10 requests should be allowed
	for i := 0; i < 10; i++ {
		if !rl.Allow("key1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	if rl.Allow("key1") {
		t.Error("11th request should be denied")
	}

	// Different key should still be allowed
	if !rl.Allow("key2") {
		t.Error("Request from different key should be allowed")
	}
}

func TestRateLimiterWindowReset(t *testing.T) {
	// 5 requests per 50ms window
	rl := NewRateLimiter(50*time.Millisecond, 5)

	// Use up the quota
	for i := 0; i < 5; i++ {
		if !rl.Allow("key1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if rl.Allow("key1") {
		t.Error("6th request should be denied immediately")
	}

	// Wait for window to pass
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("key1") {
		t.Error("Request should be allowed after window expires")
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 5)

	// Use up the quota
	for i := 0; i < 5; i++ {
		rl.Allow("key1")
	}

	// Should be at limit
	if rl.Allow("key1") {
		t.Error("Should be at limit")
	}

	// Reset
	rl.Reset()

	// Should be allowed again
	if !rl.Allow("key1") {
		t.Error("Request should be allowed after reset")
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	rl := NewRateLimiter(time.Minute, 100)

	var wg sync.WaitGroup
	numGoroutines := 10
	requestsPerGoroutine := 20

	// Spawn multiple goroutines making requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "concurrent-key"
			for j := 0; j < requestsPerGoroutine; j++ {
				rl.Allow(key)
			}
		}(i)
	}

	wg.Wait()

	// Total requests = 10 * 20 = 200, but limit is 100
	// So count should be at most 100
	count := rl.GetCount("concurrent-key")
	if count > 100 {
		t.Errorf("Count %d exceeds limit of 100", count)
	}

	// Further requests should be denied
	if rl.Allow("concurrent-key") {
		t.Error("Further requests should be denied after exceeding limit")
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	// Short window for testing
	rl := NewRateLimiter(10*time.Millisecond, 5)

	// Make some requests
	rl.Allow("key1")
	rl.Allow("key2")

	// Wait for windows to expire
	time.Sleep(25 * time.Millisecond)

	// Trigger cleanup
	rl.cleanupExpiredEntries()

	// Keys should be cleaned up
	if rl.GetCount("key1") != 0 {
		t.Error("key1 should have been cleaned up")
	}
	if rl.GetCount("key2") != 0 {
		t.Error("key2 should have been cleaned up")
	}

	// New request should start fresh
	if !rl.Allow("key1") {
		t.Error("New request should be allowed after cleanup")
	}
}

func TestRateLimiterCleanupBackground(t *testing.T) {
	rl := NewRateLimiter(10*time.Millisecond, 5)

	// Make some requests
	rl.Allow("key1")
	rl.Allow("key2")

	// Start background cleanup with short interval
	stop := rl.StartCleanup(10 * time.Millisecond)
	defer stop()

	// Wait for cleanup to run
	time.Sleep(30 * time.Millisecond)

	// Keys should be cleaned up
	if rl.GetCount("key1") != 0 {
		t.Error("key1 should have been cleaned up by background task")
	}
	if rl.GetCount("key2") != 0 {
		t.Error("key2 should have been cleaned up by background task")
	}
}

func TestRateLimiterGetters(t *testing.T) {
	window := 50 * time.Millisecond
	maxReqs := 10
	rl := NewRateLimiter(window, maxReqs)

	if rl.GetWindow() != window {
		t.Errorf("GetWindow() = %v, want %v", rl.GetWindow(), window)
	}

	if rl.GetMaxReqs() != maxReqs {
		t.Errorf("GetMaxReqs() = %d, want %d", rl.GetMaxReqs(), maxReqs)
	}

	rl.Allow("key1")
	if rl.GetCount("key1") != 1 {
		t.Errorf("GetCount() = %d, want 1", rl.GetCount("key1"))
	}
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	// Test that the sliding window works correctly
	// 5 requests per 100ms window
	rl := NewRateLimiter(100*time.Millisecond, 5)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		if !rl.Allow("key1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Wait 60ms (more than half the window)
	time.Sleep(60 * time.Millisecond)

	// The window hasn't fully reset, so we should have some remaining quota
	// In our simple implementation, we use a fixed window, so the 60ms wait
	// won't give us more quota until the full 100ms passes.
	// This is a known limitation of the simple implementation.

	// Make 2 more requests (total 5)
	for i := 0; i < 2; i++ {
		if !rl.Allow("key1") {
			t.Errorf("Request %d should be allowed", i+4)
		}
	}

	// 6th request should be denied
	if rl.Allow("key1") {
		t.Error("6th request should be denied")
	}

	// Wait for window to fully reset
	time.Sleep(50 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("key1") {
		t.Error("Request should be allowed after window expires")
	}
}

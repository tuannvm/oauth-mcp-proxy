package oauth

import (
	"sync"
	"time"
)

// RateLimiter provides simple token-based rate limiting to prevent abuse.
// Uses a fixed-window algorithm to track request counts per key.
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*rateLimiterEntry
	window   time.Duration
	maxReqs  int
}

// rateLimiterEntry tracks requests for a single key
type rateLimiterEntry struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter.
// window: the time window for rate limiting (e.g., 1 minute)
// maxReqs: maximum number of requests allowed per window
// Panics if window <= 0 or maxReqs <= 0 (invalid configuration)
func NewRateLimiter(window time.Duration, maxReqs int) *RateLimiter {
	if window <= 0 {
		panic("NewRateLimiter: window must be positive")
	}
	if maxReqs <= 0 {
		panic("NewRateLimiter: maxReqs must be positive")
	}
	return &RateLimiter{
		requests: make(map[string]*rateLimiterEntry),
		window:   window,
		maxReqs:  maxReqs,
	}
}

// Allow checks if a request from the given key should be allowed.
// Returns true if the request is within the rate limit, false otherwise.
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.requests[key]
	if !exists {
		// First request from this key
		rl.requests[key] = &rateLimiterEntry{
			count:       1,
			windowStart: now,
		}
		return true
	}

	// Check if the window has expired
	if now.Sub(entry.windowStart) >= rl.window {
		// Start a new window
		entry.count = 1
		entry.windowStart = now
		return true
	}

	// Check if we've exceeded the limit
	if entry.count >= rl.maxReqs {
		return false
	}

	// Increment counter
	entry.count++
	return true
}

// Reset clears all rate limit data. Useful for testing or manual resets.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests = make(map[string]*rateLimiterEntry)
}

// cleanupExpiredEntries removes entries that have expired.
// This should be called periodically to prevent memory leaks.
// For production use, run as a background goroutine.
func (rl *RateLimiter) cleanupExpiredEntries() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, entry := range rl.requests {
		if now.Sub(entry.windowStart) >= rl.window*2 {
			// Remove entries that are 2x past their window
			delete(rl.requests, key)
		}
	}
}

// StartCleanup starts a background goroutine that periodically cleans up expired entries.
// Returns a function that stops the cleanup when called.
// If interval <= 0, returns a no-op stop function without starting a goroutine.
func (rl *RateLimiter) StartCleanup(interval time.Duration) func() {
	if interval <= 0 {
		return func() {}
	}

	stopCh := make(chan struct{})
	stopOnce := &sync.Once{}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanupExpiredEntries()
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			close(stopCh)
		})
	}
}

// GetCount returns the current request count for a key (for monitoring/debugging)
func (rl *RateLimiter) GetCount(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	if entry, exists := rl.requests[key]; exists {
		return entry.count
	}
	return 0
}

// GetWindow returns the rate limit window duration
func (rl *RateLimiter) GetWindow() time.Duration {
	return rl.window
}

// GetMaxReqs returns the maximum requests per window
func (rl *RateLimiter) GetMaxReqs() int {
	return rl.maxReqs
}

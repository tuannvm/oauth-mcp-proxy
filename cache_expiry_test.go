package oauth

import (
	"testing"
	"time"
)

func TestTokenCacheExpiry(t *testing.T) {
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Subject:  "123",
	}

	// Add a token with very short expiry
	tokenHash := "test-token-hash"
	expiresAt := time.Now().Add(10 * time.Millisecond)
	cache.setCachedToken(tokenHash, user, expiresAt)

	// Should be cached immediately
	cached, exists := cache.getCachedToken(tokenHash)
	if !exists {
		t.Error("Token should be cached immediately")
	}
	if cached.User.Username != "testuser" {
		t.Errorf("Cached user username = %s, want testuser", cached.User.Username)
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Should be expired now
	cached, exists = cache.getCachedToken(tokenHash)
	if exists {
		t.Error("Token should be expired after expiry time")
	}
	if cached != nil {
		t.Error("Expired token should return nil for cached entry")
	}
}

func TestTokenCacheConcurrentExpiry(t *testing.T) {
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Subject:  "123",
	}

	// Add multiple tokens with short expiry
	for i := 0; i < 10; i++ {
		tokenHash := "test-token-hash-" + string(rune('0'+i))
		expiresAt := time.Now().Add(10 * time.Millisecond)
		cache.setCachedToken(tokenHash, user, expiresAt)
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// All should be expired
	for i := 0; i < 10; i++ {
		tokenHash := "test-token-hash-" + string(rune('0'+i))
		_, exists := cache.getCachedToken(tokenHash)
		if exists {
			t.Errorf("Token %s should be expired", tokenHash)
		}
	}
}

func TestTokenCacheNoExpiry(t *testing.T) {
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Subject:  "123",
	}

	// Add a token with long expiry
	tokenHash := "test-token-hash"
	expiresAt := time.Now().Add(1 * time.Hour)
	cache.setCachedToken(tokenHash, user, expiresAt)

	// Wait a bit but not past expiry
	time.Sleep(10 * time.Millisecond)

	// Should still be cached
	cached, exists := cache.getCachedToken(tokenHash)
	if !exists {
		t.Error("Token should still be cached")
	}
	if cached.User.Username != "testuser" {
		t.Errorf("Cached user username = %s, want testuser", cached.User.Username)
	}
}

func TestTokenCacheOverwrite(t *testing.T) {
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	user1 := &User{
		Username: "user1",
		Email:    "user1@example.com",
		Subject:  "1",
	}

	user2 := &User{
		Username: "user2",
		Email:    "user2@example.com",
		Subject:  "2",
	}

	// Add first user
	tokenHash := "test-token-hash"
	expiresAt := time.Now().Add(1 * time.Hour)
	cache.setCachedToken(tokenHash, user1, expiresAt)

	// Verify first user
	cached, exists := cache.getCachedToken(tokenHash)
	if !exists || cached.User.Username != "user1" {
		t.Error("First user should be cached")
	}

	// Overwrite with second user
	cache.setCachedToken(tokenHash, user2, expiresAt)

	// Verify second user
	cached, exists = cache.getCachedToken(tokenHash)
	if !exists || cached.User.Username != "user2" {
		t.Error("Second user should be cached after overwrite")
	}
}

func TestTokenCacheEmpty(t *testing.T) {
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	// Non-existent token should return not exists
	_, exists := cache.getCachedToken("non-existent")
	if exists {
		t.Error("Non-existent token should not exist in cache")
	}
}

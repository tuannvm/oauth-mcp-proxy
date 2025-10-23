package oauth

import (
	"sync"
	"time"

	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

// Re-export User from provider for backwards compatibility
type User = provider.User

// TokenCache stores validated tokens to avoid re-validation
type TokenCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedToken
}

// CachedToken represents a cached token validation result
type CachedToken struct {
	User      *User
	ExpiresAt time.Time
}

// getCachedToken retrieves a cached token validation result
func (tc *TokenCache) getCachedToken(tokenHash string) (*CachedToken, bool) {
	tc.mu.RLock()

	cached, exists := tc.cache[tokenHash]
	if !exists {
		tc.mu.RUnlock()
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		tc.mu.RUnlock()
		go tc.deleteExpiredToken(tokenHash)
		return nil, false
	}

	tc.mu.RUnlock()
	return cached, true
}

// deleteExpiredToken safely deletes an expired token from the cache
func (tc *TokenCache) deleteExpiredToken(tokenHash string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if cached, exists := tc.cache[tokenHash]; exists && time.Now().After(cached.ExpiresAt) {
		delete(tc.cache, tokenHash)
	}
}

// setCachedToken stores a token validation result
func (tc *TokenCache) setCachedToken(tokenHash string, user *User, expiresAt time.Time) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache[tokenHash] = &CachedToken{
		User:      user,
		ExpiresAt: expiresAt,
	}
}

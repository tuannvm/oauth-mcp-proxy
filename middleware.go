package oauth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

// Re-export User from provider for backwards compatibility
type User = provider.User

// Context keys
type contextKey string

const (
	oauthTokenKey  contextKey = "oauth_token"
	userContextKey contextKey = "user"
)

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

// WithOAuthToken adds an OAuth token to the context
func WithOAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, oauthTokenKey, token)
}

// GetOAuthToken extracts an OAuth token from the context
func GetOAuthToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(oauthTokenKey).(string)
	return token, ok
}

// getCachedToken retrieves a cached token validation result
func (tc *TokenCache) getCachedToken(tokenHash string) (*CachedToken, bool) {
	tc.mu.RLock()

	cached, exists := tc.cache[tokenHash]
	if !exists {
		tc.mu.RUnlock()
		return nil, false
	}

	// Check if token is expired
	if time.Now().After(cached.ExpiresAt) {
		tc.mu.RUnlock()
		// Schedule expired token deletion in a separate operation
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

	// Double-check if token is still expired before deleting
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

// Middleware returns an authentication middleware for MCP tools
func (s *Server) Middleware() func(server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Extract token from context (set by HTTP middleware)
			tokenString, ok := GetOAuthToken(ctx)
			if !ok {
				s.logger.Info("No token found in context for tool: %s", req.Params.Name)
				return nil, fmt.Errorf("authentication required: missing OAuth token")
			}

			// Create token hash for caching
			tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))

			// Check cache first
			if cached, exists := s.cache.getCachedToken(tokenHash); exists {
				s.logger.Info("Using cached authentication for tool: %s (user: %s)", req.Params.Name, cached.User.Username)
				ctx = context.WithValue(ctx, userContextKey, cached.User)
				return next(ctx, req)
			}

			// Log token hash for debugging (prevents sensitive data exposure)
			tokenHashFull := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
			tokenHashPreview := tokenHashFull[:16] + "..."
			s.logger.Info("Validating token for tool %s (hash: %s)", req.Params.Name, tokenHashPreview)

			// Validate token using configured provider (with request context for timeout/cancellation)
			user, err := s.validator.ValidateToken(ctx, tokenString)
			if err != nil {
				s.logger.Error("Token validation failed for tool %s: %v", req.Params.Name, err)
				return nil, fmt.Errorf("authentication failed: %w", err)
			}

			// Cache the validation result (expire in 5 minutes)
			expiresAt := time.Now().Add(5 * time.Minute)
			s.cache.setCachedToken(tokenHash, user, expiresAt)

			// Add user to context for downstream handlers
			ctx = context.WithValue(ctx, userContextKey, user)
			s.logger.Info("Authenticated user %s for tool: %s (cached for 5 minutes)", user.Username, req.Params.Name)

			return next(ctx, req)
		}
	}
}

// OAuthMiddleware creates an authentication middleware (legacy function for compatibility)
// Deprecated: Use NewServer() and Server.Middleware() instead
func OAuthMiddleware(validator provider.TokenValidator, enabled bool) func(server.ToolHandlerFunc) server.ToolHandlerFunc {
	// Create a temporary server for legacy compatibility
	cache := &TokenCache{cache: make(map[string]*CachedToken)}
	s := &Server{
		validator: validator,
		cache:     cache,
		logger:    &defaultLogger{},
	}

	if !enabled {
		// Return passthrough middleware
		return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
			return next
		}
	}

	return s.Middleware()
}

// validateJWT is deprecated - use provider-based validation instead

// GetUserFromContext extracts user from context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

// CreateHTTPContextFunc creates the HTTP context function for token extraction
func CreateHTTPContextFunc() func(context.Context, *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		// Extract Bearer token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// Clean any whitespace
			token = strings.TrimSpace(token)
			ctx = WithOAuthToken(ctx, token)
			log.Printf("OAuth: Token extracted from request (length: %d)", len(token))
		} else if authHeader != "" {
			preview := authHeader
			if len(authHeader) > 30 {
				preview = authHeader[:30] + "..."
			}
			log.Printf("OAuth: Invalid Authorization header format: %s", preview)
		}
		return ctx
	}
}

// CreateRequestAuthHook creates a server-level authentication hook for all MCP requests
// Note: This function is deprecated and should not be used as it cannot propagate context.
// Use OAuthMiddleware at the tool level instead, which properly handles context propagation.
func CreateRequestAuthHook(validator provider.TokenValidator) func(context.Context, interface{}, interface{}) error {
	return func(ctx context.Context, id interface{}, message interface{}) error {
		// This hook cannot propagate context changes due to its signature limitation.
		// Authentication is handled by tool-level middleware instead.
		log.Printf("OAuth: Server-level auth hook called for request ID: %v (using tool-level middleware)", id)
		return nil // Always succeed - actual auth is done at tool level
	}
}

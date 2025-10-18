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

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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

// Global token cache
var tokenCache = &TokenCache{
	cache: make(map[string]*CachedToken),
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

// OAuthMiddleware creates an authentication middleware for MCP tools
func OAuthMiddleware(validator TokenValidator, enabled bool) func(server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if !enabled {
				// OAuth disabled, allow all requests
				log.Printf("OAuth: Authentication disabled - allowing tool: %s", req.Params.Name)
				return next(ctx, req)
			}

			// Extract token from context (set by HTTP middleware)
			tokenString, ok := GetOAuthToken(ctx)
			if !ok {
				log.Printf("OAuth: No token found in context for tool: %s", req.Params.Name)
				return nil, fmt.Errorf("authentication required: missing OAuth token")
			}

			// Create token hash for caching
			tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))

			// Check cache first
			if cached, exists := tokenCache.getCachedToken(tokenHash); exists {
				log.Printf("OAuth: Using cached authentication for tool: %s (user: %s)", req.Params.Name, cached.User.Username)
				ctx = context.WithValue(ctx, userContextKey, cached.User)
				return next(ctx, req)
			}

			// Log token hash for debugging (prevents sensitive data exposure)
			tokenHashFull := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
			tokenHashPreview := tokenHashFull[:16] + "..."
			log.Printf("OAuth: Validating token for tool %s (hash: %s)", req.Params.Name, tokenHashPreview)

			// Validate token using configured provider
			user, err := validator.ValidateToken(tokenString)
			if err != nil {
				log.Printf("OAuth: Token validation failed for tool %s: %v", req.Params.Name, err)
				return nil, fmt.Errorf("authentication failed: %w", err)
			}

			// Cache the validation result (expire in 5 minutes)
			expiresAt := time.Now().Add(5 * time.Minute)
			tokenCache.setCachedToken(tokenHash, user, expiresAt)

			// Add user to context for downstream handlers
			ctx = context.WithValue(ctx, userContextKey, user)
			log.Printf("OAuth: Authenticated user %s for tool: %s (cached for 5 minutes)", user.Username, req.Params.Name)

			return next(ctx, req)
		}
	}
}

// User represents an authenticated user
type User struct {
	Username string
	Email    string
	Subject  string
}

// validateJWT is deprecated - use provider-based validation instead

// validateTokenClaims validates standard JWT claims
func validateTokenClaims(claims jwt.MapClaims) error {
	// Validate expiration
	if exp, ok := claims["exp"]; ok {
		if expTime, ok := exp.(float64); ok {
			if time.Now().Unix() > int64(expTime) {
				return fmt.Errorf("token expired")
			}
		}
	}

	// Validate not before
	if nbf, ok := claims["nbf"]; ok {
		if nbfTime, ok := nbf.(float64); ok {
			if time.Now().Unix() < int64(nbfTime) {
				return fmt.Errorf("token not yet valid")
			}
		}
	}

	// Validate issued at (should not be in the future)
	if iat, ok := claims["iat"]; ok {
		if iatTime, ok := iat.(float64); ok {
			if time.Now().Unix() < int64(iatTime) {
				return fmt.Errorf("token issued in the future")
			}
		}
	}

	return nil
}

// getStringClaim safely extracts a string claim
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

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
func CreateRequestAuthHook(validator TokenValidator) func(context.Context, interface{}, interface{}) error {
	return func(ctx context.Context, id interface{}, message interface{}) error {
		// This hook cannot propagate context changes due to its signature limitation.
		// Authentication is handled by tool-level middleware instead.
		log.Printf("OAuth: Server-level auth hook called for request ID: %v (using tool-level middleware)", id)
		return nil // Always succeed - actual auth is done at tool level
	}
}

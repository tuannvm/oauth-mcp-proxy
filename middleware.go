package oauth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

// Middleware returns an authentication middleware for MCP tools.
// Validates OAuth tokens, caches results, and adds authenticated user to context.
//
// The middleware:
//  1. Extracts OAuth token from context (set by CreateHTTPContextFunc)
//  2. Checks token cache (5-minute TTL)
//  3. Validates token using configured provider if not cached
//  4. Adds User to context via userContextKey
//  5. Passes request to tool handler with authenticated context
//
// Use GetUserFromContext(ctx) in tool handlers to access authenticated user.
//
// Note: WithOAuth() returns this middleware wrapped as mcpserver.ServerOption.
// Only call directly if using NewServer() for advanced use cases.
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

// OAuthMiddleware creates an authentication middleware (legacy function for compatibility).
//
// Deprecated: Use WithOAuth() for new code. This function creates a temporary
// Server instance for each call and doesn't support custom logging. Kept for
// backward compatibility only.
//
// Modern usage:
//
//	oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{...})
//	mcpServer := server.NewMCPServer("name", "1.0.0", oauthOption)
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

// CreateHTTPContextFunc creates an HTTP context function that extracts OAuth tokens
// from Authorization headers. Use with mcpserver.WithHTTPContextFunc() to enable
// token extraction from HTTP requests.
//
// Example:
//
//	streamableServer := mcpserver.NewStreamableHTTPServer(
//	    mcpServer,
//	    mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
//	)
//
// This extracts "Bearer <token>" from Authorization header and adds it to context
// via WithOAuthToken(). The OAuth middleware then retrieves it via GetOAuthToken().
func CreateHTTPContextFunc() func(context.Context, *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		// Extract Bearer token from Authorization header (case-insensitive per OAuth 2.0 spec)
		authHeader := r.Header.Get("Authorization")
		authLower := strings.ToLower(authHeader)

		if strings.HasPrefix(authLower, "bearer ") {
			// Extract token (skip "Bearer " or "bearer " prefix)
			token := authHeader[7:]
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

// CreateRequestAuthHook creates a server-level authentication hook for all MCP requests.
//
// Deprecated: This function cannot propagate context changes due to its signature limitation.
// Use WithOAuth() instead, which properly handles context propagation via tool-level middleware.
//
// This function is a no-op that always returns nil. Authentication happens at the tool level
// via Server.Middleware() which can properly propagate the authenticated user in context.
func CreateRequestAuthHook(validator provider.TokenValidator) func(context.Context, interface{}, interface{}) error {
	return func(ctx context.Context, id interface{}, message interface{}) error {
		// This hook cannot propagate context changes due to its signature limitation.
		// Authentication is handled by tool-level middleware instead.
		log.Printf("OAuth: Server-level auth hook called for request ID: %v (using tool-level middleware)", id)
		return nil // Always succeed - actual auth is done at tool level
	}
}

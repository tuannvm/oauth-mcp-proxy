package mcp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
)

// WithOAuth returns an OAuth-protected HTTP handler for the official
// modelcontextprotocol/go-sdk.
//
// Usage:
//
//	import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
//
//	mux := http.NewServeMux()
//	mcpServer := mcp.NewServer(&mcp.Implementation{
//	    Name:    "time-server",
//	    Version: "1.0.0",
//	}, nil)
//
//	oauthServer, handler, err := mcpoauth.WithOAuth(mux, &oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://my-server",
//	}, mcpServer)
//
//	http.ListenAndServe(":8080", handler)
//
// This function:
// - Creates OAuth server instance
// - Registers OAuth HTTP endpoints on mux
// - Wraps MCP StreamableHTTPHandler with automatic 401 handling
// - Returns OAuth server and protected HTTP handler
//
// The returned handler automatically:
// - Returns 401 with WWW-Authenticate headers if Bearer token missing
// - Passes through OPTIONS requests (CORS pre-flight)
// - Rejects non-Bearer auth schemes (OAuth-only endpoint)
//
// The returned oauth.Server instance provides access to:
// - LogStartup() - Log OAuth endpoint information
// - Discovery URL helpers (GetCallbackURL, GetMetadataURL, etc.)
//
// Tool handlers can access the authenticated user via oauth.GetUserFromContext(ctx).
func WithOAuth(mux *http.ServeMux, cfg *oauth.Config, mcpServer *mcp.Server) (*oauth.Server, http.Handler, error) {
	oauthServer, err := oauth.NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	oauthServer.RegisterHandlers(mux)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pass through OPTIONS requests (CORS pre-flight)
		if r.Method == http.MethodOptions {
			mcpHandler.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		authLower := strings.ToLower(authHeader)

		// Return 401 if Bearer token missing
		if authHeader == "" {
			oauthServer.Return401(w)
			return
		}

		// Check if it's a Bearer token (case-insensitive per OAuth 2.0 spec)
		if !strings.HasPrefix(authLower, "bearer") {
			// Reject non-Bearer schemes (OAuth endpoints require Bearer tokens only)
			oauthServer.Return401(w)
			return
		}

		// Malformed Bearer token (no space after "Bearer")
		if !strings.HasPrefix(authLower, "bearer ") {
			oauthServer.Return401InvalidToken(w)
			return
		}

		// Extract and validate token (safe slice operation)
		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix)+1 {
			oauthServer.Return401InvalidToken(w)
			return
		}
		token := authHeader[len(bearerPrefix):]

		// Clean any whitespace (e.g., "Bearer token ")
		token = strings.TrimSpace(token)

		// Validate token is not empty
		if token == "" {
			oauthServer.Return401InvalidToken(w)
			return
		}

		user, err := oauthServer.ValidateTokenCached(r.Context(), token)
		if err != nil {
			oauthServer.Return401InvalidToken(w)
			return
		}

		// Add token and user to context
		ctx := oauth.WithOAuthToken(r.Context(), token)
		ctx = oauth.WithUser(ctx, user)
		r = r.WithContext(ctx)

		// Pass to wrapped handler
		mcpHandler.ServeHTTP(w, r)
	})

	return oauthServer, wrappedHandler, nil
}

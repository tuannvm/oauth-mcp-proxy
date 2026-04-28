package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
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
// - Returns 401 with WWW-Authenticate headers if Bearer token missing/invalid
// - Populates both oauth-mcp-proxy context AND go-sdk auth context
// - Passes through OPTIONS requests (CORS pre-flight)
//
// The returned oauth.Server instance provides access to:
// - LogStartup() - Log OAuth endpoint information
// - Discovery URL helpers (GetCallbackURL, GetMetadataURL, etc.)
//
// Tool handlers can access the authenticated user via:
// - oauth.GetUserFromContext(ctx) - from oauth-mcp-proxy
// - auth.TokenInfoFromContext(ctx) - from go-sdk
func WithOAuth(mux *http.ServeMux, cfg *oauth.Config, mcpServer *mcp.Server) (*oauth.Server, http.Handler, error) {
	oauthServer, err := oauth.NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	oauthServer.RegisterHandlers(mux)

	// Create MCP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	// Create a token verifier that uses our OAuth server
	verifier := auth.TokenVerifier(func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		user, err := oauthServer.ValidateTokenCached(ctx, token)
		if err != nil {
			// Return auth.ErrInvalidToken directly for proper 401 response
			return nil, auth.ErrInvalidToken
		}

		// Build TokenInfo for go-sdk session management
		// If token has no expiry, set a reasonable default (1 hour from now)
		expiration := user.Expiry
		if expiration.IsZero() {
			expiration = time.Now().Add(time.Hour)
		}

		tokenInfo := &auth.TokenInfo{
			UserID:     user.Subject, // For session hijacking prevention
			Expiration: expiration,
		}

		// Also store in oauth-mcp-proxy context for backward compatibility
		ctx = oauth.WithOAuthToken(ctx, token)
		ctx = oauth.WithUser(ctx, user)
		*req = *req.WithContext(ctx)

		return tokenInfo, nil
	})

	// Create auth middleware using go-sdk's RequireBearerToken
	// Use protected resource metadata URL (RFC 9728) not authorization server metadata
	authMiddleware := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: oauthServer.GetProtectedResourceMetadataURL(),
	})

	// Wrap handler with CORS preflight support then auth
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pass through OPTIONS requests (CORS pre-flight)
		if r.Method == http.MethodOptions {
			mcpHandler.ServeHTTP(w, r)
			return
		}
		// Apply auth middleware for all other methods
		authMiddleware(mcpHandler).ServeHTTP(w, r)
	})

	return oauthServer, wrappedHandler, nil
}

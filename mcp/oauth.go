package mcp

import (
	"fmt"
	"net/http"

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
// - Wraps MCP StreamableHTTPHandler with OAuth token validation
// - Returns OAuth server and protected HTTP handler
//
// The returned oauth.Server instance provides access to:
// - LogStartup() - Log OAuth endpoint information
// - Discovery URL helpers (GetCallbackURL, GetMetadataURL, etc.)
//
// The HTTP handler validates OAuth tokens before delegating to the MCP server.
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
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		token := authHeader[7:]

		user, err := oauthServer.ValidateTokenCached(r.Context(), token)
		if err != nil {
			http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
			return
		}

		ctx := oauth.WithOAuthToken(r.Context(), token)
		ctx = oauth.WithUser(ctx, user)
		r = r.WithContext(ctx)

		mcpHandler.ServeHTTP(w, r)
	})

	return oauthServer, wrappedHandler, nil
}

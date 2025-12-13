package mark3labs

import (
	"fmt"
	"net/http"

	mcpserver "github.com/mark3labs/mcp-go/server"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
)

// WithOAuth returns a server option that enables OAuth authentication
// for mark3labs/mcp-go SDK.
//
// Usage:
//
//	import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
//
//	mux := http.NewServeMux()
//	oauthServer, oauthOption, err := mark3labs.WithOAuth(mux, &oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://my-server",
//	})
//	mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
//
//	streamableServer := server.NewStreamableHTTPServer(mcpServer, ...)
//	mux.HandleFunc("/mcp", oauthServer.WrapMCPEndpoint(streamableServer))
//
// This function:
// - Creates OAuth server instance
// - Registers OAuth HTTP endpoints on mux
// - Returns server instance and middleware as server option
//
// The returned Server instance provides access to:
// - WrapMCPEndpoint() - Wrap /mcp endpoint with automatic 401 handling
// - WrapHandler() - Wrap HTTP handlers with OAuth token validation
// - GetHTTPServerOptions() - Get StreamableHTTPServer options
// - LogStartup() - Log OAuth endpoint information
// - Discovery URL helpers (GetCallbackURL, GetMetadataURL, etc.)
//
// Note: You must also configure HTTPContextFunc to extract the OAuth token
// from HTTP headers. Use GetHTTPServerOptions() or CreateHTTPContextFunc().
func WithOAuth(mux *http.ServeMux, cfg *oauth.Config) (*oauth.Server, mcpserver.ServerOption, error) {
	oauthServer, err := oauth.NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	oauthServer.RegisterHandlers(mux)

	return oauthServer, mcpserver.WithToolHandlerMiddleware(NewMiddleware(oauthServer)), nil
}

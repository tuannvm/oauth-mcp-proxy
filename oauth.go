package oauth

import (
	"fmt"
	"net/http"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

// Server represents an OAuth authentication server instance.
// Each Server maintains its own token cache and validator, allowing
// multiple independent OAuth configurations in the same application.
//
// Create using NewServer(). Access middleware via Middleware() and
// register HTTP endpoints via RegisterHandlers().
type Server struct {
	config    *Config
	validator provider.TokenValidator
	cache     *TokenCache
	handler   *OAuth2Handler
	logger    Logger
}

// NewServer creates a new OAuth server with the given configuration.
// Validates configuration, initializes provider-specific token validator,
// and creates instance-scoped token cache.
//
// Example:
//
//	server, err := oauth.NewServer(&oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://my-server",
//	})
//
// Most users should use WithOAuth() instead, which wraps NewServer()
// and automatically registers handlers and middleware.
func NewServer(cfg *Config) (*Server, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Use default logger if not provided
	logger := cfg.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Create validator with logger
	validator, err := createValidator(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	// Create instance-scoped cache
	cache := &TokenCache{
		cache: make(map[string]*CachedToken),
	}

	// Create OAuth handler with logger
	handler := CreateOAuth2Handler(cfg, "1.0.0", logger)

	return &Server{
		config:    cfg,
		validator: validator,
		cache:     cache,
		handler:   handler,
		logger:    logger,
	}, nil
}

// RegisterHandlers registers OAuth HTTP endpoints on the provided mux.
// Endpoints registered:
//   - /.well-known/oauth-authorization-server - OAuth 2.0 metadata (RFC 8414)
//   - /.well-known/oauth-protected-resource - Resource metadata
//   - /.well-known/jwks.json - JWKS keys
//   - /.well-known/openid-configuration - OIDC discovery
//   - /oauth/authorize - Authorization endpoint (proxy mode)
//   - /oauth/callback - Callback handler (proxy mode)
//   - /oauth/token - Token exchange (proxy mode)
//
// Note: WithOAuth() calls this automatically. Only call directly if using
// NewServer() for advanced use cases.
func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handler.HandleAuthorizationServerMetadata)
	mux.HandleFunc("/.well-known/oauth-protected-resource", s.handler.HandleProtectedResourceMetadata)
	mux.HandleFunc("/.well-known/jwks.json", s.handler.HandleJWKS)
	mux.HandleFunc("/oauth/authorize", s.handler.HandleAuthorize)
	mux.HandleFunc("/oauth/callback", s.handler.HandleCallback)
	mux.HandleFunc("/oauth/token", s.handler.HandleToken)
	mux.HandleFunc("/.well-known/openid-configuration", s.handler.HandleOIDCDiscovery)
}

// WithOAuth returns a server option that enables OAuth authentication
// This is the composable API for mcp-go v0.41.1
//
// Usage:
//
//	mux := http.NewServeMux()
//	oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{...})
//	mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
//
// This function:
// - Creates OAuth server internally
// - Registers OAuth HTTP endpoints on mux
// - Returns middleware as server option
//
// Note: You must also configure HTTPContextFunc to extract the OAuth token
// from HTTP headers. Use CreateHTTPContextFunc() helper.
func WithOAuth(mux *http.ServeMux, cfg *Config) (mcpserver.ServerOption, error) {
	// Create OAuth server
	oauthServer, err := NewServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	// Register HTTP handlers
	oauthServer.RegisterHandlers(mux)

	// Return middleware as server option
	return mcpserver.WithToolHandlerMiddleware(oauthServer.Middleware()), nil
}

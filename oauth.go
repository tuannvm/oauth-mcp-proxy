package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
//   - /oauth/register - Dynamic client registration
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
	mux.HandleFunc("/oauth/register", s.handler.HandleRegister)
	mux.HandleFunc("/.well-known/openid-configuration", s.handler.HandleOIDCDiscovery)
}

// ValidateTokenCached validates a token with caching support.
// This is the core validation method that SDK adapters can use.
//
// The method:
//  1. Checks token cache (5-minute TTL)
//  2. Validates token using configured provider if not cached
//  3. Caches validation result for future requests
//  4. Returns authenticated User or error
//
// This method is used internally by both WrapHandler and adapter middleware.
func (s *Server) ValidateTokenCached(ctx context.Context, token string) (*User, error) {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))

	if cached, exists := s.cache.getCachedToken(tokenHash); exists {
		s.logger.Info("Using cached authentication (hash: %s...)", tokenHash[:16])
		return cached.User, nil
	}

	s.logger.Info("Validating token (hash: %s...)", tokenHash[:16])

	user, err := s.validator.ValidateToken(ctx, token)
	if err != nil {
		s.logger.Error("Token validation failed: %v", err)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	s.cache.setCachedToken(tokenHash, user, expiresAt)

	s.logger.Info("Authenticated user %s (cached for 5 minutes)", user.Username)
	return user, nil
}

// GetAuthorizationServerMetadataURL returns the OAuth 2.0 authorization server metadata URL
func (s *Server) GetAuthorizationServerMetadataURL() string {
	return fmt.Sprintf("%s/.well-known/oauth-authorization-server", s.config.ServerURL)
}

// GetProtectedResourceMetadataURL returns the protected resource metadata URL
func (s *Server) GetProtectedResourceMetadataURL() string {
	return fmt.Sprintf("%s/.well-known/oauth-protected-resource", s.config.ServerURL)
}

// GetOIDCDiscoveryURL returns the OIDC discovery URL
func (s *Server) GetOIDCDiscoveryURL() string {
	return fmt.Sprintf("%s/.well-known/openid-configuration", s.config.ServerURL)
}

// GetCallbackURL returns the OAuth callback URL
func (s *Server) GetCallbackURL() string {
	return fmt.Sprintf("%s/oauth/callback", s.config.ServerURL)
}

// GetAuthorizeURL returns the OAuth authorization URL
func (s *Server) GetAuthorizeURL() string {
	return fmt.Sprintf("%s/oauth/authorize", s.config.ServerURL)
}

// GetTokenURL returns the OAuth token URL
func (s *Server) GetTokenURL() string {
	return fmt.Sprintf("%s/oauth/token", s.config.ServerURL)
}

// GetRegisterURL returns the dynamic client registration URL
func (s *Server) GetRegisterURL() string {
	return fmt.Sprintf("%s/oauth/register", s.config.ServerURL)
}

// Endpoint represents an OAuth endpoint with its path and description
type Endpoint struct {
	Path        string
	Description string
}

// GetAllEndpoints returns all OAuth endpoints with descriptions
func (s *Server) GetAllEndpoints() []Endpoint {
	endpoints := []Endpoint{
		{Path: s.GetAuthorizationServerMetadataURL(), Description: "OAuth metadata"},
		{Path: s.GetProtectedResourceMetadataURL(), Description: "Resource metadata"},
		{Path: s.GetOIDCDiscoveryURL(), Description: "OIDC discovery"},
	}

	if s.config.Mode == "proxy" {
		endpoints = append(endpoints,
			Endpoint{Path: s.GetAuthorizeURL(), Description: "Authorization endpoint"},
			Endpoint{Path: s.GetCallbackURL(), Description: "OAuth callback"},
			Endpoint{Path: s.GetTokenURL(), Description: "Token endpoint"},
			Endpoint{Path: s.GetRegisterURL(), Description: "Client registration"},
		)
	}

	return endpoints
}

// LogStartup logs OAuth startup information including endpoints and configuration.
// Set useTLS to true if using HTTPS, false for HTTP (will add warning).
func (s *Server) LogStartup(useTLS bool) {
	warning := ""
	if !useTLS {
		warning = " - WARNING: HTTPS recommended for production"
	}

	s.logger.Info("OAuth enabled - mode: %s, provider: %s%s", s.config.Mode, s.config.Provider, warning)
	s.logger.Info("OAuth endpoints:")
	s.logger.Info("  - Authorization server metadata: %s", s.GetAuthorizationServerMetadataURL())
	s.logger.Info("  - Protected resource metadata: %s", s.GetProtectedResourceMetadataURL())
	s.logger.Info("  - OIDC discovery: %s", s.GetOIDCDiscoveryURL())

	if s.config.Mode == "proxy" {
		s.logger.Info("  - Authorization endpoint: %s", s.GetAuthorizeURL())
		s.logger.Info("  - OAuth callback: %s", s.GetCallbackURL())
		s.logger.Info("  - Token endpoint: %s", s.GetTokenURL())
		s.logger.Info("  - Client registration: %s", s.GetRegisterURL())
	}
}

// GetStatusString returns a human-readable OAuth status string
func (s *Server) GetStatusString(useTLS bool) string {
	if !useTLS {
		return fmt.Sprintf("OAuth enabled (mode: %s, provider: %s - WARNING: HTTPS recommended for production)", s.config.Mode, s.config.Provider)
	}
	return fmt.Sprintf("OAuth enabled (mode: %s, provider: %s)", s.config.Mode, s.config.Provider)
}

// GetHTTPServerOptions returns StreamableHTTPServer options needed for OAuth.
// Returns WithHTTPContextFunc option to extract OAuth tokens from HTTP headers.
// Consumers should append these to their own options when creating StreamableHTTPServer.
//
// Example:
//
//	oauthOpts := oauthServer.GetHTTPServerOptions()
//	allOpts := append(oauthOpts,
//	    mcpserver.WithEndpointPath("/mcp"),
//	    mcpserver.WithStateLess(false),
//	)
//	server := mcpserver.NewStreamableHTTPServer(mcpServer, allOpts...)
func (s *Server) GetHTTPServerOptions() []mcpserver.StreamableHTTPOption {
	return []mcpserver.StreamableHTTPOption{
		mcpserver.WithHTTPContextFunc(CreateHTTPContextFunc()),
	}
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// WrapHandler wraps an http.Handler with OAuth Bearer token validation.
// It checks for a valid Authorization header before delegating to the wrapped handler.
// If the token is missing or invalid, returns 401 with WWW-Authenticate headers
// and proper OAuth error response per RFC 6750.
//
// This eliminates the need for consumers to manually check Bearer tokens in
// their HTTP handlers. Use this to wrap MCP endpoints or any protected resource.
//
// Example:
//
//	wrappedHandler := oauthServer.WrapHandler(mcpHandler)
//	mux.HandleFunc("/mcp", wrappedHandler)
func (s *Server) WrapHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			s.logger.Info("OAuth: No bearer token provided, returning 401 with discovery info")

			metadataURL := s.GetProtectedResourceMetadataURL()
			w.Header().Add("WWW-Authenticate", `Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token"`)
			w.Header().Add("WWW-Authenticate", fmt.Sprintf(`resource_metadata="%s"`, metadataURL))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)

			if err := json.NewEncoder(w).Encode(oauthErrorResponse{
				Error:            "invalid_token",
				ErrorDescription: "Missing or invalid access token",
			}); err != nil {
				s.logger.Error("Error encoding OAuth error response: %v", err)
			}
			return
		}

		contextFunc := CreateHTTPContextFunc()
		ctx := contextFunc(r.Context(), r)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// WrapHandlerFunc wraps an http.HandlerFunc with OAuth Bearer token validation.
// Convenience wrapper around WrapHandler for HandlerFunc types.
func (s *Server) WrapHandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return s.WrapHandler(next).ServeHTTP
}

// WithOAuth returns a server option that enables OAuth authentication
// This is the composable API for mcp-go v0.41.1
//
// Usage:
//
//	mux := http.NewServeMux()
//	oauthServer, oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{...})
//	mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
//
// This function:
// - Creates OAuth server instance
// - Registers OAuth HTTP endpoints on mux
// - Returns server instance and middleware as server option
//
// The returned Server instance provides access to:
// - WrapHandler() - Wrap HTTP handlers with OAuth token validation
// - GetHTTPServerOptions() - Get StreamableHTTPServer options
// - LogStartup() - Log OAuth endpoint information
// - Discovery URL helpers (GetCallbackURL, GetMetadataURL, etc.)
//
// Note: You must also configure HTTPContextFunc to extract the OAuth token
// from HTTP headers. Use GetHTTPServerOptions() or CreateHTTPContextFunc().
func WithOAuth(mux *http.ServeMux, cfg *Config) (*Server, mcpserver.ServerOption, error) {
	oauthServer, err := NewServer(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	oauthServer.RegisterHandlers(mux)

	return oauthServer, mcpserver.WithToolHandlerMiddleware(oauthServer.Middleware()), nil
}

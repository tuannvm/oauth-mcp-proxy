package oauth

import (
	"fmt"
	"net/http"
)

// Server represents an OAuth authentication server instance
type Server struct {
	config    *Config
	validator TokenValidator
	cache     *TokenCache
	handler   *OAuth2Handler
	logger    Logger
}

// NewServer creates a new OAuth server with the given configuration
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

	if err := validator.Initialize(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
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

// RegisterHandlers registers OAuth HTTP endpoints on the provided mux
func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handler.HandleAuthorizationServerMetadata)
	mux.HandleFunc("/.well-known/oauth-protected-resource", s.handler.HandleProtectedResourceMetadata)
	mux.HandleFunc("/.well-known/jwks.json", s.handler.HandleJWKS)
	mux.HandleFunc("/oauth/authorize", s.handler.HandleAuthorize)
	mux.HandleFunc("/oauth/callback", s.handler.HandleCallback)
	mux.HandleFunc("/oauth/token", s.handler.HandleToken)
	mux.HandleFunc("/.well-known/openid-configuration", s.handler.HandleOIDCDiscovery)
}

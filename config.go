package oauth

import (
	"fmt"
	"log"
)

// Config holds OAuth configuration
type Config struct {
	// OAuth settings
	Mode         string // "native" or "proxy"
	Provider     string // "hmac", "okta", "google", "azure"
	RedirectURIs string // Redirect URIs (single or comma-separated)

	// OIDC configuration
	Issuer       string
	Audience     string
	ClientID     string
	ClientSecret string

	// Server configuration
	ServerURL string // Full URL of the MCP server

	// Security
	JWTSecret []byte // For HMAC provider and state signing
}

// SetupOAuth initializes OAuth validation and sets up OAuth configuration
func SetupOAuth(cfg *Config) (TokenValidator, error) {
	// Initialize OAuth provider based on configuration
	validator, err := createValidator(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth validator: %w", err)
	}

	if err := validator.Initialize(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize OAuth validator: %w", err)
	}

	log.Printf("OAuth authentication enabled with provider: %s", cfg.Provider)
	return validator, nil
}

// createValidator creates the appropriate token validator based on configuration
func createValidator(cfg *Config) (TokenValidator, error) {
	switch cfg.Provider {
	case "hmac":
		return &HMACValidator{}, nil
	case "okta", "google", "azure":
		return &OIDCValidator{}, nil
	default:
		return nil, fmt.Errorf("unknown OAuth provider: %s", cfg.Provider)
	}
}

// CreateOAuth2Handler creates a new OAuth2 handler for HTTP endpoints
func CreateOAuth2Handler(cfg *Config, version string) *OAuth2Handler {
	oauth2Config := NewOAuth2ConfigFromConfig(cfg, version)
	return NewOAuth2Handler(oauth2Config)
}

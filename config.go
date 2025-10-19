package oauth

import (
	"fmt"
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

	// Optional
	Logger Logger // Pluggable logger (defaults to standard logger)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Auto-detect mode if not specified
	if c.Mode == "" {
		if c.ClientID != "" {
			c.Mode = "proxy"
		} else {
			c.Mode = "native"
		}
	}

	// Validate mode
	if c.Mode != "native" && c.Mode != "proxy" {
		return fmt.Errorf("mode must be 'native' or 'proxy', got: %s", c.Mode)
	}

	// Validate provider
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	// Validate provider-specific requirements
	switch c.Provider {
	case "hmac":
		if len(c.JWTSecret) == 0 {
			return fmt.Errorf("JWTSecret is required for HMAC provider")
		}
	case "okta", "google", "azure":
		if c.Issuer == "" {
			return fmt.Errorf("Issuer is required for OIDC provider")
		}
	default:
		return fmt.Errorf("unknown provider: %s (supported: hmac, okta, google, azure)", c.Provider)
	}

	// Validate audience
	if c.Audience == "" {
		return fmt.Errorf("Audience is required")
	}

	// Validate proxy mode requirements
	if c.Mode == "proxy" {
		if c.ClientID == "" {
			return fmt.Errorf("proxy mode requires ClientID")
		}
		if c.ServerURL == "" {
			return fmt.Errorf("proxy mode requires ServerURL")
		}
		if c.RedirectURIs == "" {
			return fmt.Errorf("proxy mode requires RedirectURIs")
		}
	}

	return nil
}

// SetupOAuth initializes OAuth validation and sets up OAuth configuration
// Deprecated: Use NewServer() instead
func SetupOAuth(cfg *Config) (TokenValidator, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Initialize OAuth provider based on configuration
	validator, err := createValidator(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth validator: %w", err)
	}

	if err := validator.Initialize(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize OAuth validator: %w", err)
	}

	logger.Info("OAuth authentication enabled with provider: %s", cfg.Provider)
	return validator, nil
}

// createValidator creates the appropriate token validator based on configuration
func createValidator(cfg *Config, logger Logger) (TokenValidator, error) {
	switch cfg.Provider {
	case "hmac":
		return &HMACValidator{logger: logger}, nil
	case "okta", "google", "azure":
		return &OIDCValidator{logger: logger}, nil
	default:
		return nil, fmt.Errorf("unknown OAuth provider: %s", cfg.Provider)
	}
}

// CreateOAuth2Handler creates a new OAuth2 handler for HTTP endpoints
func CreateOAuth2Handler(cfg *Config, version string, logger Logger) *OAuth2Handler {
	if logger == nil {
		logger = &defaultLogger{}
	}
	oauth2Config := NewOAuth2ConfigFromConfig(cfg, version)
	return NewOAuth2Handler(oauth2Config, logger)
}

package oauth

import (
	"fmt"

	"github.com/tuannvm/oauth-mcp-proxy/provider"
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

	// Optional - Logging
	// Logger allows custom logging implementation. If nil, uses default logger
	// that outputs to log.Printf with level prefixes ([INFO], [ERROR], etc.).
	// Implement the Logger interface (Debug, Info, Warn, Error methods) to
	// integrate with your application's logging system (e.g., zap, logrus).
	Logger Logger
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
func SetupOAuth(cfg *Config) (provider.TokenValidator, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Initialize OAuth provider based on configuration
	validator, err := createValidator(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth validator: %w", err)
	}

	logger.Info("OAuth authentication enabled with provider: %s", cfg.Provider)
	return validator, nil
}

// createValidator creates the appropriate token validator based on configuration
func createValidator(cfg *Config, logger Logger) (provider.TokenValidator, error) {
	// Convert root Config to provider.Config
	providerCfg := &provider.Config{
		Provider:  cfg.Provider,
		Issuer:    cfg.Issuer,
		Audience:  cfg.Audience,
		JWTSecret: cfg.JWTSecret,
		Logger:    logger,
	}

	var validator provider.TokenValidator
	switch cfg.Provider {
	case "hmac":
		validator = &provider.HMACValidator{}
	case "okta", "google", "azure":
		validator = &provider.OIDCValidator{}
	default:
		return nil, fmt.Errorf("unknown OAuth provider: %s", cfg.Provider)
	}

	if err := validator.Initialize(providerCfg); err != nil {
		return nil, err
	}

	return validator, nil
}

// CreateOAuth2Handler creates a new OAuth2 handler for HTTP endpoints
func CreateOAuth2Handler(cfg *Config, version string, logger Logger) *OAuth2Handler {
	if logger == nil {
		logger = &defaultLogger{}
	}
	oauth2Config := NewOAuth2ConfigFromConfig(cfg, version)
	return NewOAuth2Handler(oauth2Config, logger)
}

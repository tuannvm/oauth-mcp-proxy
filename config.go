package oauth

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

// Config holds OAuth configuration
type Config struct {
	// OAuth settings
	Mode             string // "native" or "proxy"
	Provider         string // "hmac", "okta", "google", "azure"
	RedirectURIs     string // Redirect URIs allowlist (single or comma-separated)
	FixedRedirectURI string // Optional fixed redirect URI used for proxying callbacks

	// OIDC configuration
	Issuer       string
	Audience     string
	ClientID     string
	ClientSecret string
	Scopes       []string // OIDC scopes

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

	SkipAudienceCheck bool // whether to skip audience validation
	// The issuer URL to use for issuer validation.
	// This should only be set if the issuer in the token differs from the standard issuer URL.
	ValidatorIssuer string
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
			return fmt.Errorf("issuer is required for OIDC provider")
		}
	default:
		return fmt.Errorf("unknown provider: %s (supported: hmac, okta, google, azure)", c.Provider)
	}

	// Validate audience
	if c.Audience == "" {
		return fmt.Errorf("audience is required")
	}

	// Validate proxy mode requirements
	if c.Mode == "proxy" {
		if c.ClientID == "" {
			return fmt.Errorf("proxy mode requires ClientID")
		}
		if c.ServerURL == "" {
			return fmt.Errorf("proxy mode requires ServerURL")
		}
		if c.RedirectURIs == "" && c.FixedRedirectURI == "" {
			return fmt.Errorf("proxy mode requires RedirectURIs or FixedRedirectURI")
		}
	}

	return nil
}

// SetupOAuth initializes OAuth validation and sets up OAuth configuration.
//
// Deprecated: Use WithOAuth() for new code, which provides complete OAuth setup
// including middleware and HTTP handlers. This function only creates a validator
// and requires manual wiring.
//
// Modern usage:
//
//	oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{...})
//	mcpServer := server.NewMCPServer("name", "1.0.0", oauthOption)
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
		Provider:          cfg.Provider,
		Issuer:            cfg.Issuer,
		Audience:          cfg.Audience,
		JWTSecret:         cfg.JWTSecret,
		Logger:            logger,
		SkipAudienceCheck: cfg.SkipAudienceCheck,
		ValidatorIssuer:   cfg.ValidatorIssuer,
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

// ConfigBuilder provides a fluent API for constructing OAuth Config
type ConfigBuilder struct {
	config *Config
	host   string
	port   string
	useTLS bool
}

// NewConfigBuilder creates a new ConfigBuilder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{},
		host:   "localhost",
		port:   "8080",
	}
}

// WithMode sets the OAuth mode ("native" or "proxy")
func (b *ConfigBuilder) WithMode(mode string) *ConfigBuilder {
	b.config.Mode = mode
	return b
}

// WithProvider sets the OAuth provider ("hmac", "okta", "google", "azure")
func (b *ConfigBuilder) WithProvider(provider string) *ConfigBuilder {
	b.config.Provider = provider
	return b
}

// WithRedirectURIs sets the redirect URIs
func (b *ConfigBuilder) WithRedirectURIs(uris string) *ConfigBuilder {
	b.config.RedirectURIs = uris
	return b
}

// WithFixedRedirectURI sets the fixed redirect URI used for proxying callbacks
func (b *ConfigBuilder) WithFixedRedirectURI(uri string) *ConfigBuilder {
	b.config.FixedRedirectURI = uri
	return b
}

// WithIssuer sets the OIDC issuer
func (b *ConfigBuilder) WithIssuer(issuer string) *ConfigBuilder {
	b.config.Issuer = issuer
	return b
}

// WithAudience sets the audience
func (b *ConfigBuilder) WithAudience(audience string) *ConfigBuilder {
	b.config.Audience = audience
	return b
}

// WithClientID sets the client ID
func (b *ConfigBuilder) WithClientID(clientID string) *ConfigBuilder {
	b.config.ClientID = clientID
	return b
}

// WithClientSecret sets the client secret
func (b *ConfigBuilder) WithClientSecret(secret string) *ConfigBuilder {
	b.config.ClientSecret = secret
	return b
}

// WithJWTSecret sets the JWT secret
func (b *ConfigBuilder) WithJWTSecret(secret []byte) *ConfigBuilder {
	b.config.JWTSecret = secret
	return b
}

// WithScopes sets the OIDC scopes
func (b *ConfigBuilder) WithScopes(scopes []string) *ConfigBuilder {
	b.config.Scopes = scopes
	return b
}

// WithLogger sets the logger
func (b *ConfigBuilder) WithLogger(logger Logger) *ConfigBuilder {
	b.config.Logger = logger
	return b
}

// WithSkipAudienceCheck sets audience check toggle
func (b *ConfigBuilder) WithSkipAudienceCheck(skipAudienceCheck bool) *ConfigBuilder {
	b.config.SkipAudienceCheck = skipAudienceCheck
	return b
}

// WithValidatorIssuer sets the validator issuer URL
func (b *ConfigBuilder) WithValidatorIssuer(validatorIssuer string) *ConfigBuilder {
	b.config.ValidatorIssuer = validatorIssuer
	return b
}

// WithServerURL sets the full server URL directly
func (b *ConfigBuilder) WithServerURL(url string) *ConfigBuilder {
	b.config.ServerURL = url
	return b
}

// WithHost sets the server host (used to construct ServerURL if not set)
func (b *ConfigBuilder) WithHost(host string) *ConfigBuilder {
	b.host = host
	return b
}

// WithPort sets the server port (used to construct ServerURL if not set)
func (b *ConfigBuilder) WithPort(port string) *ConfigBuilder {
	b.port = port
	return b
}

// WithTLS enables HTTPS scheme (used to construct ServerURL if not set)
func (b *ConfigBuilder) WithTLS(useTLS bool) *ConfigBuilder {
	b.useTLS = useTLS
	return b
}

// Build constructs and validates the Config
func (b *ConfigBuilder) Build() (*Config, error) {
	if b.config.ServerURL == "" {
		b.config.ServerURL = AutoDetectServerURL(b.host, b.port, b.useTLS)
	}
	if err := b.config.Validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}

// AutoDetectServerURL constructs a server URL from components
func AutoDetectServerURL(host, port string, useTLS bool) string {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

// FromEnv creates a Config from environment variables
func FromEnv() (*Config, error) {
	serverURL := getEnv("MCP_URL", "")

	host := getEnv("MCP_HOST", "localhost")
	port := getEnv("MCP_PORT", "8080")
	useTLS := getEnv("HTTPS_CERT_FILE", "") != "" && getEnv("HTTPS_KEY_FILE", "") != ""

	if serverURL == "" {
		serverURL = AutoDetectServerURL(host, port, useTLS)
	}

	jwtSecret := getEnv("JWT_SECRET", "")

	scopes := []string{}
	scopesEnv := getEnv("OIDC_SCOPES", "")
	if scopesEnv != "" {
		scopes = strings.Split(scopesEnv, " ")
	}

	return NewConfigBuilder().
		WithMode(getEnv("OAUTH_MODE", "")).
		WithProvider(getEnv("OAUTH_PROVIDER", "")).
		WithRedirectURIs(getEnv("OAUTH_REDIRECT_URIS", "")).
		WithFixedRedirectURI(getEnv("OAUTH_FIXED_REDIRECT_URI", "")).
		WithIssuer(getEnv("OIDC_ISSUER", "")).
		WithAudience(getEnv("OIDC_AUDIENCE", "")).
		WithClientID(getEnv("OIDC_CLIENT_ID", "")).
		WithClientSecret(getEnv("OIDC_CLIENT_SECRET", "")).
		WithScopes(scopes).
		WithSkipAudienceCheck(parseBoolEnv("OIDC_SKIP_AUDIENCE_CHECK", false)).
		WithValidatorIssuer(getEnv("OIDC_VALIDATOR_ISSUER", "")).
		WithServerURL(serverURL).
		WithJWTSecret([]byte(jwtSecret)).
		Build()
}

// parseBoolEnv parses a boolean environment variable
func parseBoolEnv(key string, defaultVal bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

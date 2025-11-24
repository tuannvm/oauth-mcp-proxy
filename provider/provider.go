package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user
type User struct {
	Username string
	Email    string
	Subject  string
}

// Logger interface for pluggable logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Config holds OAuth configuration (subset needed by provider)
type Config struct {
	Provider          string
	Issuer            string
	Audience          string
	JWTSecret         []byte
	Logger            Logger
	SkipAudienceCheck bool
	ValidatorIssuer   string
}

// TokenValidator interface for OAuth token validation
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*User, error)
	Initialize(cfg *Config) error
}

// HMACValidator validates JWT tokens using HMAC-SHA256 (backward compatibility)
type HMACValidator struct {
	secret     string
	audience   string
	secretOnce sync.Once
}

// OIDCValidator validates JWT tokens using OIDC/JWKS (Okta, Google, Azure)
type OIDCValidator struct {
	verifier        *oidc.IDTokenVerifier
	provider        *oidc.Provider
	audience        string
	TokenValidators []func(claims jwt.MapClaims) error
	logger          Logger
}

// Initialize sets up the HMAC validator with JWT secret and audience
func (v *HMACValidator) Initialize(cfg *Config) error {
	v.secretOnce.Do(func() {
		v.secret = string(cfg.JWTSecret)
		v.audience = cfg.Audience
	})

	if v.secret == "" {
		return fmt.Errorf("JWT_SECRET is required for HMAC provider")
	}

	if v.audience == "" {
		return fmt.Errorf("JWT audience is required for HMAC provider")
	}

	return nil
}

// ValidateToken validates JWT token using HMAC-SHA256
func (v *HMACValidator) ValidateToken(ctx context.Context, tokenString string) (*User, error) {
	// Note: ctx parameter accepted for interface compliance, but HMAC validation is local-only (no I/O)
	// Remove Bearer prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse and validate JWT with signature verification
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(v.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse and validate token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate required claims including audience
	if err := validateTokenClaims(claims); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Validate audience claim for security
	if err := v.validateAudience(claims); err != nil {
		return nil, fmt.Errorf("audience validation failed: %w", err)
	}

	// Extract user information
	user := &User{
		Subject:  getStringClaim(claims, "sub"),
		Username: getStringClaim(claims, "preferred_username"),
		Email:    getStringClaim(claims, "email"),
	}

	if user.Subject == "" {
		return nil, fmt.Errorf("missing subject in token")
	}

	return user, nil
}

// validateAudience validates the audience claim matches the expected value
func (v *HMACValidator) validateAudience(claims jwt.MapClaims) error {
	// Extract audience claim (can be string or []string)
	audClaim, exists := claims["aud"]
	if !exists {
		return fmt.Errorf("missing audience claim")
	}

	// Handle string audience
	if audStr, ok := audClaim.(string); ok {
		if audStr != v.audience {
			return fmt.Errorf("invalid audience: expected %s, got %s", v.audience, audStr)
		}
		return nil
	}

	// Handle array of audiences
	if audArray, ok := audClaim.([]interface{}); ok {
		for _, aud := range audArray {
			if audStr, ok := aud.(string); ok && audStr == v.audience {
				return nil
			}
		}
		return fmt.Errorf("invalid audience: expected %s not found in audience list", v.audience)
	}

	return fmt.Errorf("invalid audience claim type")
}

// Initialize sets up the OIDC validator with provider discovery
func (v *OIDCValidator) Initialize(cfg *Config) error {
	if cfg.Issuer == "" {
		return fmt.Errorf("OIDC issuer is required for OIDC provider")
	}
	if cfg.Audience == "" {
		return fmt.Errorf("OIDC audience is required for OIDC provider")
	}

	v.logger = cfg.Logger
	if v.logger == nil {
		v.logger = &noOpLogger{}
	}
	v.audience = cfg.Audience

	// Use standard library context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure HTTP client with appropriate timeouts and TLS settings
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Verify TLS certificates
				MinVersion:         tls.VersionTLS12,
			},
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}
	if cfg.ValidatorIssuer != "" {
		ctx = oidc.InsecureIssuerURLContext(ctx, cfg.ValidatorIssuer)
	}
	// Create OIDC provider with custom HTTP client
	provider, err := oidc.NewProvider(
		oidc.ClientContext(ctx, httpClient),
		cfg.Issuer,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	// Configure token verifier with required validation settings
	verifier := provider.Verifier(&oidc.Config{
		ClientID:             cfg.Audience, // Note: go-oidc uses ClientID field for audience validation - see https://github.com/coreos/go-oidc/blob/v3/oidc/verify.go#L85
		SupportedSigningAlgs: []string{oidc.RS256, oidc.ES256},
		SkipClientIDCheck:    cfg.SkipAudienceCheck,
		SkipExpiryCheck:      false,
		SkipIssuerCheck:      false,
	})

	v.provider = provider
	v.verifier = verifier
	if !cfg.SkipAudienceCheck {
		v.logger.Info("OAuth: OIDC validator initialized with audience validation: %s", cfg.Audience)
		v.TokenValidators = append(v.TokenValidators, v.validateAudience)
	}
	return nil
}

// ValidateToken validates JWT token using OIDC/JWKS
func (v *OIDCValidator) ValidateToken(ctx context.Context, tokenString string) (*User, error) {
	// Remove Bearer prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Use incoming context with timeout for OIDC provider call
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// go-oidc handles RSA signature validation, JWKS fetching, and key rotation
	idToken, err := v.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	// Extract claims from verified token
	var claims struct {
		Subject           string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified,omitempty"`
		Name              string `json:"name,omitempty"`
		// Standard OIDC claims are validated by go-oidc:
		// - iss (issuer)
		// - aud (audience)
		// - exp (expiration)
		// - iat (issued at)
		// - nbf (not before)
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Extract raw claims for audience validation
	var rawClaims jwt.MapClaims
	if err := idToken.Claims(&rawClaims); err != nil {
		return nil, fmt.Errorf("failed to extract raw claims: %w", err)
	}

	// Run extra validation functions
	for i, fn := range v.TokenValidators {
		err := fn(rawClaims)
		if err != nil {
			return nil, fmt.Errorf("validation function %d failed with error: %w", i, err)
		}
	}

	return &User{
		Subject:  claims.Subject,
		Username: claims.PreferredUsername,
		Email:    claims.Email,
	}, nil
}

// validateAudience validates the audience claim matches the expected value for OIDC tokens
func (v *OIDCValidator) validateAudience(claims jwt.MapClaims) error {
	// Extract audience claim (can be string or []string)
	audClaim, exists := claims["aud"]
	if !exists {
		return fmt.Errorf("missing audience claim")
	}

	// Handle string audience
	if audStr, ok := audClaim.(string); ok {
		if audStr != v.audience {
			return fmt.Errorf("invalid audience: expected %s, got %s", v.audience, audStr)
		}
		return nil
	}

	// Handle array of audiences
	if audArray, ok := audClaim.([]interface{}); ok {
		for _, aud := range audArray {
			if audStr, ok := aud.(string); ok && audStr == v.audience {
				return nil
			}
		}
		return fmt.Errorf("invalid audience: expected %s not found in audience list", v.audience)
	}

	return fmt.Errorf("invalid audience claim type")
}

// validateTokenClaims validates standard JWT claims
func validateTokenClaims(claims jwt.MapClaims) error {
	// Validate expiration
	if exp, ok := claims["exp"]; ok {
		if expTime, ok := exp.(float64); ok {
			if time.Now().Unix() > int64(expTime) {
				return fmt.Errorf("token expired")
			}
		}
	}

	// Validate not before
	if nbf, ok := claims["nbf"]; ok {
		if nbfTime, ok := nbf.(float64); ok {
			if time.Now().Unix() < int64(nbfTime) {
				return fmt.Errorf("token not yet valid")
			}
		}
	}

	// Validate issued at (should not be in the future)
	if iat, ok := claims["iat"]; ok {
		if iatTime, ok := iat.(float64); ok {
			if time.Now().Unix() < int64(iatTime) {
				return fmt.Errorf("token issued in the future")
			}
		}
	}

	return nil
}

// getStringClaim safely extracts a string claim
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

// noOpLogger is a no-op logger used when cfg.Logger is nil
type noOpLogger struct{}

func (l *noOpLogger) Debug(msg string, args ...interface{}) {}
func (l *noOpLogger) Info(msg string, args ...interface{})  {}
func (l *noOpLogger) Warn(msg string, args ...interface{})  {}
func (l *noOpLogger) Error(msg string, args ...interface{}) {}

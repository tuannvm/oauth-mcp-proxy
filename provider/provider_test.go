package provider

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestHMACValidator_AudienceValidation tests JWT audience validation
func TestHMACValidator_AudienceValidation(t *testing.T) {
	// Test configuration
	cfg := &Config{
		JWTSecret: []byte("test-secret-key-for-hmac-validation"),
		Audience:  "test-service-audience",
	}

	validator := &HMACValidator{}
	err := validator.Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	t.Run("ValidAudience", func(t *testing.T) {
		// Create token with correct audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "test-user",
			"aud":   "test-service-audience",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"email": "test@example.com",
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		user, err := validator.ValidateToken(context.Background(), tokenString)
		if err != nil {
			t.Errorf("Expected valid token to pass, got error: %v", err)
		}

		if user == nil || user.Subject != "test-user" {
			t.Errorf("Expected valid user, got: %+v", user)
		}
	})

	t.Run("InvalidAudience", func(t *testing.T) {
		// Create token with wrong audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"aud": "wrong.audience.com", // Wrong audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("Expected token with wrong audience to fail validation")
		}

		if err != nil && err.Error() != "audience validation failed: invalid audience: expected test-service-audience, got wrong.audience.com" {
			t.Errorf("Expected specific audience error, got: %v", err)
		}
	})

	t.Run("MissingAudience", func(t *testing.T) {
		// Create token without audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("Expected token without audience to fail validation")
		}

		if err != nil && err.Error() != "audience validation failed: missing audience claim" {
			t.Errorf("Expected missing audience error, got: %v", err)
		}
	})

	t.Run("AudienceArray", func(t *testing.T) {
		// Create token with audience as array (valid)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"aud": []string{"test-service-audience", "other.service.com"}, // Array with correct audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		user, err := validator.ValidateToken(context.Background(), tokenString)
		if err != nil {
			t.Errorf("Expected token with correct audience in array to pass, got error: %v", err)
		}

		if user == nil || user.Subject != "test-user" {
			t.Errorf("Expected valid user, got: %+v", user)
		}
	})

	t.Run("AudienceArrayInvalid", func(t *testing.T) {
		// Create token with audience array not containing expected audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"aud": []string{"wrong.service.com", "other.service.com"}, // Array without correct audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("Expected token with wrong audience array to fail validation")
		}

		if err != nil && err.Error() != "audience validation failed: invalid audience: expected test-service-audience not found in audience list" {
			t.Errorf("Expected specific audience array error, got: %v", err)
		}
	})
}

// TestHMACValidator_InitializationValidation tests validator initialization
func TestHMACValidator_InitializationValidation(t *testing.T) {
	t.Run("MissingSecret", func(t *testing.T) {
		cfg := &Config{
			JWTSecret: []byte(""), // Missing secret
			Audience:  "test-service-audience",
		}

		validator := &HMACValidator{}
		err := validator.Initialize(cfg)

		if err == nil {
			t.Error("Expected initialization to fail with missing secret")
		}

		if err != nil && err.Error() != "JWT_SECRET is required for HMAC provider" {
			t.Errorf("Expected specific secret error, got: %v", err)
		}
	})

	t.Run("MissingAudience", func(t *testing.T) {
		cfg := &Config{
			JWTSecret: []byte("test-secret"),
			Audience:  "", // Missing audience
		}

		validator := &HMACValidator{}
		err := validator.Initialize(cfg)

		if err == nil {
			t.Error("Expected initialization to fail with missing audience")
		}

		if err != nil && err.Error() != "JWT audience is required for HMAC provider" {
			t.Errorf("Expected specific audience error, got: %v", err)
		}
	})

	t.Run("ValidConfiguration", func(t *testing.T) {
		cfg := &Config{
			JWTSecret: []byte("test-secret"),
			Audience:  "test-service-audience",
		}

		validator := &HMACValidator{}
		err := validator.Initialize(cfg)

		if err != nil {
			t.Errorf("Expected valid configuration to succeed, got error: %v", err)
		}

		if validator.secret != "test-secret" {
			t.Errorf("Expected secret to be set correctly")
		}

		if validator.audience != "test-service-audience" {
			t.Errorf("Expected audience to be set correctly")
		}
	})
}

// TestHMACValidator_SecurityValidation tests that the vulnerability is fixed
func TestHMACValidator_SecurityValidation(t *testing.T) {
	// This test specifically validates that the vulnerability described in PE-7429 is fixed

	t.Run("RejectCrossServiceToken", func(t *testing.T) {
		cfg := &Config{
			JWTSecret: []byte("test-secret"),
			Audience:  "test-service-audience",
		}

		validator := &HMACValidator{}
		err := validator.Initialize(cfg)
		if err != nil {
			t.Fatalf("Failed to initialize validator: %v", err)
		}

		// Simulate a token from another service (different audience)
		crossServiceToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "cross-service-user",
			"aud": "other.service.com", // Different service audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
			"iss": "company.okta.com", // Same issuer
		})

		tokenString, err := crossServiceToken.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign cross-service token: %v", err)
		}

		// This should FAIL - the vulnerability would allow this to pass
		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("SECURITY VULNERABILITY: Cross-service token was accepted! This should fail.")
		}

		// Verify it fails for the correct reason (audience validation)
		if err != nil && !strings.Contains(err.Error(), "audience validation failed") {
			t.Errorf("Token failed for wrong reason. Expected audience validation failure, got: %v", err)
		}
	})
}

// TestOIDCValidator_AudienceValidation tests OIDC JWT audience validation
func TestOIDCValidator_AudienceValidation(t *testing.T) {
	// Test the validateAudience method directly since OIDC provider setup requires external services
	validator := &OIDCValidator{
		audience: "test-service-audience",
	}

	tests := []struct {
		name      string
		claims    jwt.MapClaims
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid audience string",
			claims: jwt.MapClaims{
				"aud": "test-service-audience",
				"sub": "user123",
			},
			expectErr: false,
		},
		{
			name: "invalid audience string",
			claims: jwt.MapClaims{
				"aud": "wrong.audience.com",
				"sub": "user123",
			},
			expectErr: true,
			errMsg:    "invalid audience: expected test-service-audience, got wrong.audience.com",
		},
		{
			name: "missing audience claim",
			claims: jwt.MapClaims{
				"sub": "user123",
			},
			expectErr: true,
			errMsg:    "missing audience claim",
		},
		{
			name: "valid audience array",
			claims: jwt.MapClaims{
				"aud": []interface{}{"test-service-audience", "other.service.com"},
				"sub": "user123",
			},
			expectErr: false,
		},
		{
			name: "invalid audience array",
			claims: jwt.MapClaims{
				"aud": []interface{}{"wrong.service.com", "other.service.com"},
				"sub": "user123",
			},
			expectErr: true,
			errMsg:    "invalid audience: expected test-service-audience not found in audience list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateAudience(tt.claims)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOIDCValidator_InitializationValidation tests OIDC initialization validation
func TestOIDCValidator_InitializationValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing issuer",
			config: &Config{
				Issuer:   "",
				Audience: "test-audience",
			},
			expectError: true,
			errorMsg:    "OIDC issuer is required for OIDC provider",
		},
		{
			name: "missing audience",
			config: &Config{
				Issuer:   "https://example.com",
				Audience: "",
			},
			expectError: true,
			errorMsg:    "OIDC audience is required for OIDC provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &OIDCValidator{}
			err := validator.Initialize(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

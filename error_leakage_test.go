package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

func TestErrorMessagesDoNotLeakSensitiveInfo(t *testing.T) {
	// Test that error messages don't leak redirect URIs or OAuth provider details

	// Test 1: Invalid redirect URI error doesn't leak the URI
	t.Run("InvalidRedirectURIErrorDoesNotLeakURI", func(t *testing.T) {
		handler := &OAuth2Handler{
			config: &OAuth2Config{
				Mode:                         "proxy",
				FixedRedirectURI:             "https://server.com/callback",
				AllowedClientRedirectDomains: "example.com",
			},
			logger: &defaultLogger{},
		}

		// Try to use an evil redirect URI
		params := url.Values{}
		params.Set("redirect_uri", "https://evil.com/steal-tokens")
		params.Set("state", "test-state")

		req := httptest.NewRequest("GET", "/oauth/authorize?"+params.Encode(), nil)
		recorder := httptest.NewRecorder()

		handler.HandleAuthorize(recorder, req)

		body := recorder.Body.String()
		// Error should not contain the evil redirect URI
		if strings.Contains(body, "evil.com") {
			t.Errorf("Error message leaks redirect URI: %s", body)
		}
		if strings.Contains(body, "steal-tokens") {
			t.Errorf("Error message leaks redirect URI path: %s", body)
		}
	})

	// Test 2: Authorization error doesn't leak OAuth provider details
	t.Run("AuthorizationErrorDoesNotLeakProviderDetails", func(t *testing.T) {
		handler := &OAuth2Handler{
			config: &OAuth2Config{
				Mode:             "proxy",
				FixedRedirectURI: "https://server.com/callback",
				Issuer:           "https://secret-internal-idp.company.com",
				ClientID:         "secret-client-id",
			},
			logger: &defaultLogger{},
		}

		params := url.Values{}
		params.Set("redirect_uri", "http://localhost:3000/callback")
		params.Set("state", "test-state")
		params.Set("error", "access_denied")
		params.Set("error_description", "The user denied access to scope: profile email internal_admin_role")

		req := httptest.NewRequest("GET", "/oauth/callback?"+params.Encode(), nil)
		recorder := httptest.NewRecorder()

		handler.HandleCallback(recorder, req)

		body := recorder.Body.String()
		// Error should not leak the detailed error description from the OAuth provider
		if strings.Contains(body, "internal_admin_role") {
			t.Errorf("Error message leaks OAuth provider details: %s", body)
		}
		if strings.Contains(body, "secret-internal-idp") {
			t.Errorf("Error message leaks internal issuer: %s", body)
		}
	})
}

func TestGenericErrorMessagesForSecurity(t *testing.T) {
	// Test that sensitive operations return generic error messages

	t.Run("InvalidStateReturnsGenericError", func(t *testing.T) {
		handler := &OAuth2Handler{
			config: &OAuth2Config{
				Mode:             "proxy",
				FixedRedirectURI: "https://server.com/callback",
				stateSigningKey:  []byte("test-key-32-bytes-long!"),
			},
			logger:     &defaultLogger{},
			seenNonces: make(map[string]time.Time),
		}

		// Use a tampered state
		params := url.Values{}
		params.Set("code", "valid-code")
		params.Set("state", "tampered-state-without-signature")

		req := httptest.NewRequest("GET", "/oauth/callback?"+params.Encode(), nil)
		recorder := httptest.NewRecorder()

		handler.HandleCallback(recorder, req)

		// Should return generic error, not reveal anything about state format
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", recorder.Code)
		}

		body := recorder.Body.String()
		// Error should be generic
		if !strings.Contains(body, "Invalid state") {
			t.Errorf("Expected generic 'Invalid state' error, got: %s", body)
		}
		// Should not contain internal implementation details
		if strings.Contains(body, "HMAC") || strings.Contains(body, "signature") {
			t.Errorf("Error message reveals implementation details: %s", body)
		}
	})
}

func TestTokenValidationErrorMessages(t *testing.T) {
	// Test that token validation errors don't leak token details
	t.Run("TokenErrorsDoNotLeakTokenContent", func(t *testing.T) {
		validator := &provider.HMACValidator{}
		err := validator.Initialize(&provider.Config{
			Audience:  "test-audience",
			JWTSecret: []byte("test-secret-key-32-bytes-long"),
		})
		if err != nil {
			t.Fatalf("Failed to initialize validator: %v", err)
		}

		// Try to validate an invalid token
		_, err = validator.ValidateToken(context.TODO(), "invalid.token.here")

		if err == nil {
			t.Error("Expected error for invalid token")
		} else {
			// Error should not contain the actual token content
			if strings.Contains(err.Error(), "invalid.token.here") {
				t.Errorf("Error message leaks token content: %v", err)
			}
		}
	})
}

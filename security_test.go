package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRedirectURIValidation(t *testing.T) {
	tests := []struct {
		name             string
		allowedRedirects string
		testURI          string
		expected         bool
	}{
		{
			name:             "No allowlist configured - reject all",
			allowedRedirects: "",
			testURI:          "https://client.example.com/callback",
			expected:         false,
		},
		{
			name:             "Single URI match",
			allowedRedirects: "https://client.example.com/callback",
			testURI:          "https://client.example.com/callback",
			expected:         true,
		},
		{
			name:             "Multiple URIs - first match",
			allowedRedirects: "https://client1.example.com/callback,https://client2.example.com/callback",
			testURI:          "https://client1.example.com/callback",
			expected:         true,
		},
		{
			name:             "Multiple URIs - second match",
			allowedRedirects: "https://client1.example.com/callback,https://client2.example.com/callback",
			testURI:          "https://client2.example.com/callback",
			expected:         true,
		},
		{
			name:             "No match",
			allowedRedirects: "https://client1.example.com/callback",
			testURI:          "https://malicious.example.com/callback",
			expected:         false,
		},
		{
			name:             "Partial match rejected",
			allowedRedirects: "https://client.example.com/callback",
			testURI:          "https://client.example.com/callback/evil",
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuth2Config{
				RedirectURIs: tt.allowedRedirects,
			}
			handler := &OAuth2Handler{config: config, logger: &defaultLogger{}}

			result := handler.isValidRedirectURI(tt.testURI)
			if result != tt.expected {
				t.Errorf("isValidRedirectURI(%q) = %v, expected %v", tt.testURI, result, tt.expected)
			}
		})
	}
}

func TestOAuthParameterValidation(t *testing.T) {
	handler := &OAuth2Handler{logger: &defaultLogger{}}

	tests := []struct {
		name        string
		params      map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid parameters",
			params: map[string]string{
				"code":           "valid_code_123",
				"state":          "valid_state",
				"code_challenge": "valid_challenge",
			},
			expectError: false,
		},
		{
			name: "Code too long",
			params: map[string]string{
				"code": strings.Repeat("a", 513), // 513 characters
			},
			expectError: true,
			errorMsg:    "invalid code parameter length",
		},
		{
			name: "State too long",
			params: map[string]string{
				"state": strings.Repeat("a", 257), // 257 characters
			},
			expectError: true,
			errorMsg:    "invalid state parameter length",
		},
		{
			name: "Code challenge too long",
			params: map[string]string{
				"code_challenge": strings.Repeat("a", 257), // 257 characters
			},
			expectError: true,
			errorMsg:    "invalid code_challenge parameter length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a form request with the test parameters
			values := make([]string, 0, len(tt.params)*2)
			for key, value := range tt.params {
				values = append(values, key, value)
			}

			req := httptest.NewRequest("POST", "/test", strings.NewReader(""))
			req.Form = make(map[string][]string)
			for i := 0; i < len(values); i += 2 {
				req.Form[values[i]] = []string{values[i+1]}
			}

			err := handler.validateOAuthParams(req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := &OAuth2Handler{logger: &defaultLogger{}}
	recorder := httptest.NewRecorder()

	handler.addSecurityHeaders(recorder)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Cache-Control":          "no-store, no-cache, max-age=0",
		"Pragma":                 "no-cache",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := recorder.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Header %s = %q, expected %q", header, actualValue, expectedValue)
		}
	}
}

func TestHTTPSEnforcementInHandlers(t *testing.T) {
	config := &OAuth2Config{
		Mode: "proxy",
	}
	handler := &OAuth2Handler{config: config, logger: &defaultLogger{}}

	endpoints := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"HandleAuthorize", handler.HandleAuthorize},
		{"HandleCallback", handler.HandleCallback},
		{"HandleToken", handler.HandleToken},
	}

	for _, endpoint := range endpoints {
		t.Run("Native mode blocks "+endpoint.name, func(t *testing.T) {
			// Test native mode blocking
			nativeConfig := &OAuth2Config{Mode: "native"}
			nativeHandler := &OAuth2Handler{config: nativeConfig}

			var testHandler func(http.ResponseWriter, *http.Request)
			switch endpoint.name {
			case "HandleAuthorize":
				testHandler = nativeHandler.HandleAuthorize
			case "HandleCallback":
				testHandler = nativeHandler.HandleCallback
			case "HandleToken":
				testHandler = nativeHandler.HandleToken
			}

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)

			testHandler(recorder, req)

			if recorder.Code != http.StatusNotFound {
				t.Errorf("%s in native mode should return 404, got %d", endpoint.name, recorder.Code)
			}

			body := recorder.Body.String()
			if !strings.Contains(body, "OAuth proxy disabled in native mode") {
				t.Errorf("%s should return OAuth proxy disabled message", endpoint.name)
			}
		})
	}
}

func TestJWKSProxyMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		provider string
		expected int
	}{
		{
			name:     "Native mode blocks JWKS",
			mode:     "native",
			provider: "okta",
			expected: http.StatusNotFound,
		},
		{
			name:     "HMAC provider returns empty JWKS",
			mode:     "proxy",
			provider: "hmac",
			expected: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuth2Config{
				Mode:     tt.mode,
				Provider: tt.provider,
			}
			handler := &OAuth2Handler{config: config, logger: &defaultLogger{}}

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)

			handler.HandleJWKS(recorder, req)

			if recorder.Code != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, recorder.Code)
			}

			if tt.mode == "native" {
				body := recorder.Body.String()
				if !strings.Contains(body, "JWKS endpoint disabled in native mode") {
					t.Errorf("Should return JWKS disabled message in native mode")
				}
			}

			if tt.provider == "hmac" && tt.mode == "proxy" {
				body := recorder.Body.String()
				if body != `{"keys":[]}` {
					t.Errorf("HMAC provider should return empty JWKS, got %s", body)
				}
			}
		})
	}
}

func TestAttackScenarios(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	handler := &OAuth2Handler{
		config: &OAuth2Config{
			stateSigningKey: key,
			RedirectURIs:    "https://mcp-server.com/oauth/callback",
		},
	}

	t.Run("StateTampering", func(t *testing.T) {
		stateData := map[string]string{
			"state":    "legitimate-state",
			"redirect": "https://legitimate-client.com/callback",
		}

		signedState, err := handler.signState(stateData)
		if err != nil {
			t.Fatalf("Failed to sign state: %v", err)
		}

		decoded, _ := base64.URLEncoding.DecodeString(signedState)
		var tamperedData map[string]string
		_ = json.Unmarshal(decoded, &tamperedData)

		tamperedData["redirect"] = "https://evil.com/steal-codes"

		tamperedJSON, _ := json.Marshal(tamperedData)
		tamperedState := base64.URLEncoding.EncodeToString(tamperedJSON)

		_, err = handler.verifyState(tamperedState)

		if err == nil {
			t.Error("SECURITY FAILURE: Tampered state was accepted!")
		}
	})

	t.Run("UnsignedState", func(t *testing.T) {
		unsignedData := map[string]string{
			"state":    "some-state",
			"redirect": "https://evil.com/callback",
		}

		unsignedJSON, _ := json.Marshal(unsignedData)
		unsignedState := base64.URLEncoding.EncodeToString(unsignedJSON)

		_, err := handler.verifyState(unsignedState)

		if err == nil {
			t.Error("SECURITY FAILURE: Unsigned state was accepted!")
		}
	})

	t.Run("EvilRedirectAtAuthorization", func(t *testing.T) {
		clientRedirectURI := "https://evil.com/steal-codes"
		isLocalhost := isLocalhostURI(clientRedirectURI)

		if isLocalhost {
			t.Error("SECURITY FAILURE: evil.com detected as localhost!")
		}
	})

	t.Run("LeakedKeyForgedState", func(t *testing.T) {
		maliciousState := map[string]string{
			"state":    "attack",
			"redirect": "https://evil.com/steal",
		}

		forgedState, err := handler.signState(maliciousState)
		if err != nil {
			t.Fatalf("Failed to sign forged state: %v", err)
		}

		verified, err := handler.verifyState(forgedState)
		if err != nil {
			t.Fatalf("Signature verification should succeed: %v", err)
		}

		redirectURI := verified["redirect"]
		isLocalhost := isLocalhostURI(redirectURI)

		if isLocalhost {
			t.Error("SECURITY FAILURE: evil.com detected as localhost!")
		}
	})

	t.Run("SubdomainAttack", func(t *testing.T) {
		attackURI := "https://localhost.evil.com/callback"
		isLocalhost := isLocalhostURI(attackURI)

		if isLocalhost {
			t.Error("SECURITY FAILURE: Subdomain attack succeeded!")
		}
	})

	t.Run("LegitimateLocalhost", func(t *testing.T) {
		clientRedirectURI := "http://localhost:6274/oauth/callback/debug"
		isLocalhost := isLocalhostURI(clientRedirectURI)

		if !isLocalhost {
			t.Error("SECURITY FAILURE: localhost not detected!")
		}

		stateData := map[string]string{
			"state":    "inspector-session",
			"redirect": clientRedirectURI,
		}

		signedState, err := handler.signState(stateData)
		if err != nil {
			t.Fatalf("Failed to sign state: %v", err)
		}

		verified, err := handler.verifyState(signedState)
		if err != nil {
			t.Fatalf("State verification failed: %v", err)
		}

		redirectURI := verified["redirect"]
		if !isLocalhostURI(redirectURI) {
			t.Error("SECURITY FAILURE: localhost redirect rejected!")
		}
	})
}

func TestHTTPSEnforcementForNonLocalhost(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		shouldFail bool
	}{
		{"HTTP localhost allowed", "http://localhost:8080/callback", false},
		{"HTTP 127.0.0.1 allowed", "http://127.0.0.1:3000/callback", false},
		{"HTTPS production allowed", "https://example.com/callback", false},
		{"HTTP production rejected", "http://example.com/callback", true},
		{"HTTP subdomain rejected", "http://app.example.com/callback", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isLocalhost := isLocalhostURI(tt.uri)
			parsed, _ := url.Parse(tt.uri)

			requiresHTTPS := !isLocalhost && parsed.Scheme == "http"

			if requiresHTTPS && !tt.shouldFail {
				t.Errorf("HTTP non-localhost should be rejected but test expects pass: %s", tt.uri)
			}
			if !requiresHTTPS && tt.shouldFail {
				t.Errorf("URI should be allowed but test expects fail: %s", tt.uri)
			}
		})
	}
}

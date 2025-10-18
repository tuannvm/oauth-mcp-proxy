package oauth

import (
	"net/http"
	"net/http/httptest"
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
			handler := &OAuth2Handler{config: config}

			result := handler.isValidRedirectURI(tt.testURI)
			if result != tt.expected {
				t.Errorf("isValidRedirectURI(%q) = %v, expected %v", tt.testURI, result, tt.expected)
			}
		})
	}
}

func TestOAuthParameterValidation(t *testing.T) {
	handler := &OAuth2Handler{}

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
	handler := &OAuth2Handler{}
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
	handler := &OAuth2Handler{config: config}

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
			handler := &OAuth2Handler{config: config}

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

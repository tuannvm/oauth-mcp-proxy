package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAuthorizationServerMetadata(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		provider    string
		issuer      string
		mcpURL      string
		checkFields []string
	}{
		{
			name:        "Native mode with Okta",
			mode:        "native",
			provider:    "okta",
			issuer:      "https://dev.okta.com",
			mcpURL:      "https://mcp.example.com",
			checkFields: []string{"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri"},
		},
		{
			name:        "Native mode with Google",
			mode:        "native",
			provider:    "google",
			issuer:      "https://accounts.google.com",
			mcpURL:      "https://mcp.example.com",
			checkFields: []string{"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri"},
		},
		{
			name:        "Proxy mode",
			mode:        "proxy",
			provider:    "okta",
			issuer:      "https://dev.okta.com",
			mcpURL:      "https://mcp.example.com",
			checkFields: []string{"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuth2Config{
				Mode:     tt.mode,
				Provider: tt.provider,
				Issuer:   tt.issuer,
				MCPURL:   tt.mcpURL,
			}
			handler := &OAuth2Handler{config: config, logger: &defaultLogger{}}

			metadata := handler.GetAuthorizationServerMetadata()

			// Check that required fields are present
			for _, field := range tt.checkFields {
				if _, exists := metadata[field]; !exists {
					t.Errorf("Missing required field: %s", field)
				}
			}

			// Verify mode-specific behavior
			issuer := metadata["issuer"].(string)
			authEndpoint := metadata["authorization_endpoint"].(string)

			if tt.mode == "native" {
				// Native mode should point to OAuth provider
				if issuer != tt.issuer {
					t.Errorf("Native mode issuer = %s, expected %s", issuer, tt.issuer)
				}
				if tt.provider == "okta" {
					expectedAuth := tt.issuer + "/oauth2/v1/authorize"
					if authEndpoint != expectedAuth {
						t.Errorf("Native mode auth endpoint = %s, expected %s", authEndpoint, expectedAuth)
					}
				}
			} else {
				// Proxy mode should point to MCP server
				if issuer != tt.mcpURL {
					t.Errorf("Proxy mode issuer = %s, expected %s", issuer, tt.mcpURL)
				}
				expectedAuth := tt.mcpURL + "/oauth/authorize"
				if authEndpoint != expectedAuth {
					t.Errorf("Proxy mode auth endpoint = %s, expected %s", authEndpoint, expectedAuth)
				}
			}
		})
	}
}

func TestHandleAuthorizationServerMetadata(t *testing.T) {
	config := &OAuth2Config{
		Mode:     "native",
		Provider: "okta",
		Issuer:   "https://dev.okta.com",
		MCPURL:   "https://mcp.example.com",
	}
	handler := &OAuth2Handler{config: config, logger: &defaultLogger{}}

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{"GET request", "GET", http.StatusOK},
		{"HEAD request", "HEAD", http.StatusOK},
		{"OPTIONS request", "OPTIONS", http.StatusOK},
		{"POST request", "POST", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/.well-known/oauth-authorization-server", nil)

			handler.HandleAuthorizationServerMetadata(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("Status = %d, expected %d", recorder.Code, tt.expectedStatus)
			}

			// Check CORS headers are present
			if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
				t.Errorf("CORS Allow-Origin = %s, expected *", origin)
			}

			if tt.method == "GET" {
				// Verify JSON response
				var metadata map[string]interface{}
				if err := json.Unmarshal(recorder.Body.Bytes(), &metadata); err != nil {
					t.Errorf("Failed to parse JSON response: %v", err)
				}

				if issuer, exists := metadata["issuer"]; !exists {
					t.Errorf("Missing issuer field in metadata")
				} else if issuer != config.Issuer {
					t.Errorf("Issuer = %s, expected %s", issuer, config.Issuer)
				}
			}
		})
	}
}

func TestHandleProtectedResourceMetadata(t *testing.T) {
	config := &OAuth2Config{
		Issuer: "https://dev.okta.com",
		MCPURL: "https://mcp.example.com",
	}
	handler := &OAuth2Handler{config: config, logger: &defaultLogger{}}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)

	handler.HandleProtectedResourceMetadata(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Status = %d, expected %d", recorder.Code, http.StatusOK)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &metadata); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check required fields
	if resource := metadata["resource"]; resource != config.MCPURL {
		t.Errorf("Resource = %s, expected %s", resource, config.MCPURL)
	}

	authServers, exists := metadata["authorization_servers"]
	if !exists {
		t.Errorf("Missing authorization_servers field")
	} else {
		servers := authServers.([]interface{})
		if len(servers) != 1 || servers[0] != config.Issuer {
			t.Errorf("Authorization servers = %v, expected [%s]", servers, config.Issuer)
		}
	}
}

func TestHandleOIDCDiscovery(t *testing.T) {
	config := &OAuth2Config{
		MCPURL:   "https://mcp.example.com",
		Provider: "okta",
		Audience: "https://api.example.com",
	}
	handler := &OAuth2Handler{
		config: config,
		logger: &defaultLogger{},
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/openid_configuration", nil)

	handler.HandleOIDCDiscovery(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Status = %d, expected %d", recorder.Code, http.StatusOK)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &metadata); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check required OIDC fields
	requiredFields := []string{
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"response_types_supported",
		"subject_types_supported",
		"id_token_signing_alg_values_supported",
	}

	for _, field := range requiredFields {
		if _, exists := metadata[field]; !exists {
			t.Errorf("Missing required OIDC field: %s", field)
		}
	}

	if issuer := metadata["issuer"]; issuer != config.MCPURL {
		t.Errorf("OIDC issuer = %s, expected %s", issuer, config.MCPURL)
	}

	if audience := metadata["audience"]; audience != config.Audience {
		t.Errorf("OIDC audience = %s, expected %s", audience, config.Audience)
	}
}

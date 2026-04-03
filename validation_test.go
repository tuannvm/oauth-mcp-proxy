package oauth

import (
	"testing"
)

func TestValidateIssuerURL(t *testing.T) {
	tests := []struct {
		name      string
		issuer    string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Valid HTTPS issuer",
			issuer:    "https://accounts.example.com",
			expectErr: false,
		},
		{
			name:      "Valid HTTPS issuer with path",
			issuer:    "https://accounts.example.com/oidc/",
			expectErr: false,
		},
		{
			name:      "Valid HTTP localhost",
			issuer:    "http://localhost:8080",
			expectErr: false,
		},
		{
			name:      "Valid HTTP 127.0.0.1",
			issuer:    "http://127.0.0.1:9000",
			expectErr: false,
		},
		{
			name:      "Empty issuer",
			issuer:    "",
			expectErr: true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "Invalid URL format",
			issuer:    "not-a-valid-url",
			expectErr: true,
			errMsg:    "must use http or https",
		},
		{
			name:      "Non-HTTPS non-localhost",
			issuer:    "http://example.com",
			expectErr: true,
			errMsg:    "must use HTTPS",
		},
		{
			name:      "Invalid scheme",
			issuer:    "ftp://example.com",
			expectErr: true,
			errMsg:    "must use http or https",
		},
		{
			name:      "Missing host",
			issuer:    "https://",
			expectErr: true,
			errMsg:    "must have a host",
		},
		{
			name:      "Raw IP address",
			issuer:    "https://192.168.1.1",
			expectErr: true,
			errMsg:    "should not be a raw IP",
		},
		{
			name:      "Path without trailing slash",
			issuer:    "https://example.com/oidc",
			expectErr: false,
		},
		{
			name:      "Suspicious pattern with dots",
			issuer:    "https://..evil.com",
			expectErr: true,
			errMsg:    "invalid patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIssuerURL(tt.issuer)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		expectErr   bool
		errMsg      string
	}{
		{
			name:        "Valid HTTPS redirect",
			redirectURI: "https://client.example.com/callback",
			expectErr:   false,
		},
		{
			name:        "Valid HTTP localhost",
			redirectURI: "http://localhost:3000/oauth/callback",
			expectErr:   false,
		},
		{
			name:        "Valid HTTP 127.0.0.1",
			redirectURI: "http://127.0.0.1:8080/callback",
			expectErr:   false,
		},
		{
			name:        "Empty redirect URI",
			redirectURI: "",
			expectErr:   true,
			errMsg:      "cannot be empty",
		},
		{
			name:        "Invalid URL format",
			redirectURI: "not-a-url",
			expectErr:   true,
			errMsg:      "must use http or https",
		},
		{
			name:        "Non-HTTPS non-localhost",
			redirectURI: "http://example.com/callback",
			expectErr:   true,
			errMsg:      "must use HTTPS",
		},
		{
			name:        "Contains fragment",
			redirectURI: "https://example.com/callback#fragment",
			expectErr:   true,
			errMsg:      "must not contain a fragment",
		},
		{
			name:        "Suspicious hostname pattern",
			redirectURI: "https://..evil.com/callback",
			expectErr:   true,
			errMsg:      "invalid patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRedirectURI(tt.redirectURI)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

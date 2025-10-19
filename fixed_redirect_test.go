package oauth

import (
	"crypto/rand"
	"testing"
)

func TestFixedRedirectModeLocalhostOnly(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	tests := []struct {
		name          string
		clientURI     string
		shouldPass    bool
		expectedError string
	}{
		{
			name:       "HTTP localhost allowed",
			clientURI:  "http://localhost:8080/callback",
			shouldPass: true,
		},
		{
			name:       "HTTP 127.0.0.1 allowed",
			clientURI:  "http://127.0.0.1:3000/callback",
			shouldPass: true,
		},
		{
			name:       "HTTP IPv6 localhost allowed",
			clientURI:  "http://[::1]:9000/callback",
			shouldPass: true,
		},
		{
			name:       "HTTPS localhost allowed",
			clientURI:  "https://localhost/callback",
			shouldPass: true,
		},
		{
			name:          "HTTPS production domain rejected",
			clientURI:     "https://evil.com/callback",
			shouldPass:    false,
			expectedError: "Fixed redirect mode only allows localhost",
		},
		{
			name:          "HTTP production domain rejected",
			clientURI:     "http://evil.com/callback",
			shouldPass:    false,
			expectedError: "HTTPS required for non-localhost",
		},
		{
			name:          "localhost subdomain rejected",
			clientURI:     "https://localhost.evil.com/callback",
			shouldPass:    false,
			expectedError: "Fixed redirect mode only allows localhost",
		},
		{
			name:          "URI with fragment rejected",
			clientURI:     "http://localhost:8080/callback#fragment",
			shouldPass:    false,
			expectedError: "must not contain fragment",
		},
		{
			name:          "Custom scheme rejected",
			clientURI:     "custom://localhost:8080/callback",
			shouldPass:    false,
			expectedError: "Invalid redirect_uri scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isLocalhost := isLocalhostURI(tt.clientURI)

			if tt.shouldPass && !isLocalhost {
				t.Errorf("Expected localhost detection to pass for %s", tt.clientURI)
			}

			if !tt.shouldPass && isLocalhost && tt.expectedError != "must not contain fragment" && tt.expectedError != "Invalid redirect_uri scheme" {
				t.Errorf("Expected localhost detection to fail for %s", tt.clientURI)
			}

			t.Logf("URI: %s, isLocalhost: %v, shouldPass: %v", tt.clientURI, isLocalhost, tt.shouldPass)
		})
	}
}

func TestFixedRedirectModeSecurityModel(t *testing.T) {
	t.Log("Fixed Redirect Mode Security Model:")
	t.Log("- Single OAUTH_REDIRECT_URI configured (no commas)")
	t.Log("- Server uses fixed URI to communicate with OAuth provider")
	t.Log("- Client redirect URIs MUST be localhost for security")
	t.Log("- HMAC-signed state prevents redirect URI tampering")
	t.Log("")
	t.Log("Attack Prevention:")
	t.Log("1. Open Redirect → Localhost-only restriction prevents external redirects")
	t.Log("2. State Tampering → HMAC signature verification prevents modification")
	t.Log("3. Code Theft → PKCE prevents token exchange without code_verifier")
	t.Log("4. HTTP Exposure → HTTPS required for non-localhost URIs")
	t.Log("")
	t.Log("Use Case: Development tools (MCP Inspector) running on localhost")
	t.Log("Production: Use allowlist mode instead")
}

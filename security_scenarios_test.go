package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"
)

func TestSecurityScenarios(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	handler := &OAuth2Handler{
		config: &OAuth2Config{
			stateSigningKey: key,
			RedirectURIs:    "https://mcp-server.com/oauth/callback",
		},
	}

	t.Run("Attack: State tampering to redirect to attacker site", func(t *testing.T) {
		// Attacker obtains valid signed state
		stateData := map[string]string{
			"state":    "legitimate-state",
			"redirect": "https://legitimate-client.com/callback",
		}

		signedState, err := handler.signState(stateData)
		if err != nil {
			t.Fatalf("Failed to sign state: %v", err)
		}

		// Attacker tries to decode and modify the redirect URI
		decoded, _ := base64.URLEncoding.DecodeString(signedState)
		var tamperedData map[string]string
		_ = json.Unmarshal(decoded, &tamperedData)

		// Change redirect to evil site
		tamperedData["redirect"] = "https://evil.com/steal-codes"

		// Re-encode (but signature is now invalid)
		tamperedJSON, _ := json.Marshal(tamperedData)
		tamperedState := base64.URLEncoding.EncodeToString(tamperedJSON)

		// Try to verify tampered state
		_, err = handler.verifyState(tamperedState)

		// Should fail due to invalid signature
		if err == nil {
			t.Error("SECURITY FAILURE: Tampered state was accepted!")
		} else {
			t.Logf("✓ Security working: Tampered state rejected: %v", err)
		}
	})

	t.Run("Attack: Remove signature from state", func(t *testing.T) {
		// Create unsigned state without signature
		unsignedData := map[string]string{
			"state":    "some-state",
			"redirect": "https://evil.com/callback",
		}

		unsignedJSON, _ := json.Marshal(unsignedData)
		unsignedState := base64.URLEncoding.EncodeToString(unsignedJSON)

		// Try to verify unsigned state
		_, err := handler.verifyState(unsignedState)

		if err == nil {
			t.Error("SECURITY FAILURE: Unsigned state was accepted!")
		} else {
			t.Logf("✓ Security working: Unsigned state rejected: %v", err)
		}
	})

	t.Run("Attack: Replay state from different session", func(t *testing.T) {
		// Sign state with one handler
		stateData := map[string]string{
			"state":    "session-1",
			"redirect": "https://client.com/callback",
		}
		signedState, _ := handler.signState(stateData)

		// Create new handler with different key (simulates different server/restart)
		newKey := make([]byte, 32)
		_, _ = rand.Read(newKey)

		newHandler := &OAuth2Handler{
			config: &OAuth2Config{
				stateSigningKey: newKey,
			},
		}

		// Try to use old state with new handler
		_, err := newHandler.verifyState(signedState)

		if err == nil {
			t.Error("SECURITY FAILURE: State from different key was accepted!")
		} else {
			t.Logf("✓ Security working: Cross-session state rejected: %v", err)
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

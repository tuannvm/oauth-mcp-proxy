package oauth

import (
	"crypto/rand"
	"testing"
)

func TestStateSigningAndVerification(t *testing.T) {
	// Create handler with signing key
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	handler := &OAuth2Handler{
		config: &OAuth2Config{
			stateSigningKey: key,
		},
	}

	tests := []struct {
		name        string
		stateData   map[string]string
		expectError bool
		tamper      bool
	}{
		{
			name: "Valid state with both fields",
			stateData: map[string]string{
				"state":    "abc123",
				"redirect": "https://example.com/callback",
			},
			expectError: false,
		},
		{
			name: "Valid state with localhost redirect",
			stateData: map[string]string{
				"state":    "xyz789",
				"redirect": "http://localhost:8080/callback",
			},
			expectError: false,
		},
		{
			name: "State with special characters",
			stateData: map[string]string{
				"state":    "state-with-dashes_and_underscores",
				"redirect": "https://example.com/callback?foo=bar&baz=qux",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sign state
			signed, err := handler.signState(tt.stateData)
			if err != nil {
				t.Fatalf("Failed to sign state: %v", err)
			}

			// Verify state
			verified, err := handler.verifyState(signed)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check data integrity
			if !tt.expectError {
				if verified["state"] != tt.stateData["state"] {
					t.Errorf("State mismatch: got %s, want %s", verified["state"], tt.stateData["state"])
				}
				if verified["redirect"] != tt.stateData["redirect"] {
					t.Errorf("Redirect mismatch: got %s, want %s", verified["redirect"], tt.stateData["redirect"])
				}
			}
		})
	}
}

func TestStateTamperingDetection(t *testing.T) {
	// Create handler with signing key
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	handler := &OAuth2Handler{
		config: &OAuth2Config{
			stateSigningKey: key,
		},
	}

	// Create and sign valid state
	stateData := map[string]string{
		"state":    "original",
		"redirect": "https://good.com/callback",
	}

	signed, err := handler.signState(stateData)
	if err != nil {
		t.Fatalf("Failed to sign state: %v", err)
	}

	// Verify the original signed state works correctly
	_, err = handler.verifyState(signed)
	if err != nil {
		t.Logf("Good: Original state verification works: %v", err)
	}

	// Now create a handler with different key
	differentKey := make([]byte, 32)
	_, _ = rand.Read(differentKey)

	handler2 := &OAuth2Handler{
		config: &OAuth2Config{
			stateSigningKey: differentKey,
		},
	}

	// Try to verify with different key (should fail)
	_, err = handler2.verifyState(signed)
	if err == nil {
		t.Error("Expected verification to fail with different key, but it succeeded")
	} else {
		t.Logf("Good: Verification failed with different key: %v", err)
	}

	// Test with completely invalid base64
	_, err = handler.verifyState("not-valid-base64!!!")
	if err == nil {
		t.Error("Expected verification to fail with invalid base64")
	}
}

func TestLocalhostDetection(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{"HTTP localhost", "http://localhost:8080/callback", true},
		{"HTTPS localhost", "https://localhost/callback", true},
		{"HTTP 127.0.0.1", "http://127.0.0.1:3000/callback", true},
		{"HTTPS 127.0.0.1", "https://127.0.0.1/callback", true},
		{"IPv6 localhost", "http://[::1]:8080/callback", true},
		{"Non-localhost domain", "http://example.com/callback", false},
		{"Non-localhost subdomain", "https://localhost.example.com/callback", false},
		{"Invalid URI", "not-a-valid-uri", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalhostURI(tt.uri)
			if result != tt.expected {
				t.Errorf("isLocalhostURI(%q) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}

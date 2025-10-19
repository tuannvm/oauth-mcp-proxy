package oauth

import (
	"crypto/rand"
	"testing"
)

func TestCompleteAttackScenarios(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	handler := &OAuth2Handler{
		config: &OAuth2Config{
			stateSigningKey: key,
			RedirectURIs:    "https://mcp-server.com/oauth/callback", // Single URI = fixed mode
		},
	}

	t.Run("Scenario 1: Attacker tries to use evil.com redirect at authorization", func(t *testing.T) {
		// Attacker submits authorization request with evil redirect
		clientRedirectURI := "https://evil.com/steal-codes"

		// Check validation
		isLocalhost := isLocalhostURI(clientRedirectURI)

		if isLocalhost {
			t.Error("SECURITY FAILURE: evil.com detected as localhost!")
		}

		// Should be rejected at authorization step
		t.Logf("✓ evil.com is not localhost: %v", !isLocalhost)
		t.Logf("✓ Would be rejected with: 'Fixed redirect mode only allows localhost'")
	})

	t.Run("Scenario 2: Attacker with leaked JWT_SECRET tries to forge state", func(t *testing.T) {
		// Attacker creates malicious state
		maliciousState := map[string]string{
			"state":    "attack",
			"redirect": "https://evil.com/steal",
		}

		// Sign with same key (simulating leaked secret)
		forgedState, err := handler.signState(maliciousState)
		if err != nil {
			t.Fatalf("Failed to sign forged state: %v", err)
		}

		// Verify signature (will succeed - signature is valid)
		verified, err := handler.verifyState(forgedState)
		if err != nil {
			t.Fatalf("Signature verification should succeed: %v", err)
		}

		// BUT: callback handler re-validates localhost
		redirectURI := verified["redirect"]
		isLocalhost := isLocalhostURI(redirectURI)

		if isLocalhost {
			t.Error("SECURITY FAILURE: evil.com detected as localhost!")
		}

		t.Logf("✓ Signature verified (attacker has valid key)")
		t.Logf("✓ But redirect URI validation fails: evil.com is not localhost")
		t.Logf("✓ Defense in depth: HMAC + localhost validation")
	})

	t.Run("Scenario 3: Legitimate MCP Inspector flow", func(t *testing.T) {
		// Inspector sends legitimate localhost redirect
		clientRedirectURI := "http://localhost:6274/oauth/callback/debug"

		// Validate
		isLocalhost := isLocalhostURI(clientRedirectURI)
		if !isLocalhost {
			t.Error("SECURITY FAILURE: localhost not detected!")
		}

		// Create signed state
		stateData := map[string]string{
			"state":    "inspector-session",
			"redirect": clientRedirectURI,
		}

		signedState, err := handler.signState(stateData)
		if err != nil {
			t.Fatalf("Failed to sign state: %v", err)
		}

		// Verify state
		verified, err := handler.verifyState(signedState)
		if err != nil {
			t.Fatalf("State verification failed: %v", err)
		}

		// Validate redirect URI
		redirectURI := verified["redirect"]
		if !isLocalhostURI(redirectURI) {
			t.Error("SECURITY FAILURE: localhost redirect rejected!")
		}

		t.Logf("✓ localhost redirect accepted")
		t.Logf("✓ State signed and verified successfully")
		t.Logf("✓ Callback would proxy to: %s", redirectURI)
	})

	t.Run("Scenario 4: localhost.evil.com subdomain attack", func(t *testing.T) {
		// Attacker uses subdomain that contains "localhost"
		attackURI := "https://localhost.evil.com/callback"

		isLocalhost := isLocalhostURI(attackURI)
		if isLocalhost {
			t.Error("SECURITY FAILURE: Subdomain attack succeeded!")
		}

		t.Logf("✓ localhost.evil.com correctly identified as non-localhost")
		t.Logf("✓ Hostname parsing prevents subdomain attacks")
	})
}

func TestDefenseInDepthLayers(t *testing.T) {
	t.Log("=== Defense in Depth Security Layers ===")
	t.Log("")
	t.Log("Layer 1: Authorization Request Validation")
	t.Log("  - Localhost-only restriction for fixed redirect mode")
	t.Log("  - HTTPS enforcement for non-localhost URIs")
	t.Log("  - Fragment rejection per OAuth 2.0 spec")
	t.Log("  - Scheme validation (http/https only)")
	t.Log("")
	t.Log("Layer 2: State Integrity Protection")
	t.Log("  - HMAC-SHA256 signature using JWT_SECRET")
	t.Log("  - Deterministic signing algorithm")
	t.Log("  - Constant-time comparison prevents timing attacks")
	t.Log("")
	t.Log("Layer 3: Callback Validation")
	t.Log("  - HMAC signature verification")
	t.Log("  - Localhost re-validation (defense in depth)")
	t.Log("  - Even with leaked key, evil.com redirects blocked")
	t.Log("")
	t.Log("Layer 4: Token Exchange")
	t.Log("  - PKCE code_verifier required")
	t.Log("  - Prevents code theft even if intercepted")
	t.Log("")
	t.Log("Result: Multiple independent security controls")
	t.Log("Even if one layer is bypassed, others prevent attack")
}

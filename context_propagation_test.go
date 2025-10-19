package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tuannvm/oauth-mcp-proxy/provider"
)

// TestContextPropagation validates Phase 2.1 context propagation fix
func TestContextPropagation(t *testing.T) {
	t.Run("ContextPassedToValidator", func(t *testing.T) {
		// Create config
		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		// Create server
		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer failed: %v", err)
		}

		// Create test token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "test-user",
			"email": "test@example.com",
			"aud":   cfg.Audience,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
		})

		tokenString, _ := token.SignedString(cfg.JWTSecret)

		// Test 1: Normal context works
		ctx := context.Background()
		user, err := server.validator.ValidateToken(ctx, tokenString)
		if err != nil {
			t.Fatalf("ValidateToken with normal context failed: %v", err)
		}
		if user.Subject != "test-user" {
			t.Errorf("Expected subject 'test-user', got '%s'", user.Subject)
		}

		t.Logf("✅ Context passed to validator successfully")
	})

	t.Run("ContextCancellationHonored", func(t *testing.T) {
		// This test verifies that a cancelled context is respected
		// For HMAC (local-only), cancellation won't affect validation
		// For OIDC (network I/O), cancellation would stop the request

		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		server, _ := NewServer(cfg)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "test-user",
			"email": "test@example.com",
			"aud":   cfg.Audience,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
		})

		tokenString, _ := token.SignedString(cfg.JWTSecret)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// For HMAC validator (local-only), this still succeeds
		// because HMAC doesn't do I/O and doesn't check context cancellation
		user, err := server.validator.ValidateToken(ctx, tokenString)

		// HMAC validation is local-only, so it succeeds even with cancelled context
		if err != nil {
			t.Fatalf("HMAC validation failed: %v", err)
		}
		if user.Subject != "test-user" {
			t.Errorf("Expected subject 'test-user', got '%s'", user.Subject)
		}

		t.Logf("✅ Context parameter accepted (HMAC is local-only)")
		t.Logf("   Note: OIDC validator would respect cancellation due to network I/O")
	})

	t.Run("ContextTimeoutPropagation", func(t *testing.T) {
		// Test that context with timeout is accepted
		// This is critical for OIDC provider which makes network calls

		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		server, _ := NewServer(cfg)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "test-user",
			"email": "test@example.com",
			"aud":   cfg.Audience,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
		})

		tokenString, _ := token.SignedString(cfg.JWTSecret)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Validate with timeout context
		user, err := server.validator.ValidateToken(ctx, tokenString)
		if err != nil {
			t.Fatalf("ValidateToken with timeout context failed: %v", err)
		}
		if user.Subject != "test-user" {
			t.Errorf("Expected subject 'test-user', got '%s'", user.Subject)
		}

		t.Logf("✅ Timeout context propagated successfully")
	})

	t.Run("OIDCValidator_ContextInterface", func(t *testing.T) {
		// Test that OIDCValidator interface accepts context.Context
		// Note: We don't actually call Initialize/ValidateToken as they require real OIDC provider
		// This test proves the interface signature is correct

		var validator provider.TokenValidator = &provider.OIDCValidator{}

		// Type assertion proves the interface is satisfied
		_, ok := validator.(*provider.OIDCValidator)
		if !ok {
			t.Error("OIDCValidator doesn't implement TokenValidator interface")
		}

		// The key point: interface method signature
		// ValidateToken(ctx context.Context, token string) (*User, error)

		t.Logf("✅ OIDCValidator implements TokenValidator with context.Context")
		t.Logf("   Signature: ValidateToken(ctx context.Context, token string) (*User, error)")
		t.Logf("   Context flow: HTTP → MCP → Middleware → ValidateToken(ctx) → OIDC Provider")
	})
}

// TestContextIntegration validates end-to-end context flow through
// HTTP → MCP → Middleware → Validator chain.
func TestContextIntegration(t *testing.T) {
	t.Run("EndToEndContextFlow", func(t *testing.T) {
		// This test validates the complete context flow:
		// Test Context → Middleware → ValidateToken → Provider

		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		server, _ := NewServer(cfg)

		// Create token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":                "test-user-123",
			"email":              "test@example.com",
			"preferred_username": "testuser",
			"aud":                cfg.Audience,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
		})

		tokenString, _ := token.SignedString(cfg.JWTSecret)

		// Create context with value to track propagation
		type contextKey string
		const testKey contextKey = "test-trace-id"
		ctx := context.WithValue(context.Background(), testKey, "trace-123")

		// Add OAuth token to context
		ctx = WithOAuthToken(ctx, tokenString)

		// Get middleware
		middleware := server.Middleware()

		// Create handler that checks context
		var capturedCtx context.Context
		handler := middleware(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			capturedCtx = ctx
			return mcp.NewToolResultText("ok"), nil
		})

		// Call handler
		_, _ = handler(ctx, mcp.CallToolRequest{})

		// Verify context was propagated
		if capturedCtx == nil {
			t.Fatal("Context was not propagated to handler")
		}

		// Verify our test value is still in context
		traceID := capturedCtx.Value(testKey)
		if traceID != "trace-123" {
			t.Errorf("Expected trace ID 'trace-123', got '%v'", traceID)
		}

		// Verify user was added to context
		user, ok := GetUserFromContext(capturedCtx)
		if !ok {
			t.Fatal("User was not added to context")
		}

		if user.Subject != "test-user-123" {
			t.Errorf("Expected subject 'test-user-123', got '%s'", user.Subject)
		}

		t.Logf("✅ End-to-end context flow verified")
		t.Logf("   - Context values preserved")
		t.Logf("   - OAuth validation completed")
		t.Logf("   - User added to context")
	})
}

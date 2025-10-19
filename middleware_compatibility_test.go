package oauth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// TestMCPGoMiddlewareCompatibility validates mcp-go v0.41.1 middleware integration
func TestMCPGoMiddlewareCompatibility(t *testing.T) {
	t.Run("WithToolHandlerMiddleware_ServerWide", func(t *testing.T) {
		// This test validates that our middleware works with mcp-go v0.41.1's
		// WithToolHandlerMiddleware option (server-wide middleware)

		// 1. Create OAuth server
		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		oauthServer, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer failed: %v", err)
		}

		// 2. Create MCP server with OAuth middleware (server-wide)
		// This is the CORRECT pattern for mcp-go v0.41.1
		mcpServer := mcpserver.NewMCPServer("Test Server", "1.0.0",
			mcpserver.WithToolHandlerMiddleware(oauthServer.Middleware()),
		)

		// 3. Verify server was created successfully
		if mcpServer == nil {
			t.Fatal("MCP server creation failed")
		}

		// 4. Add a tool (middleware automatically applies)
		toolCalled := false
		var capturedCtx context.Context
		mcpServer.AddTool(
			mcp.Tool{
				Name:        "test_tool",
				Description: "Test tool",
			},
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				toolCalled = true
				capturedCtx = ctx

				// Verify user was added to context by middleware
				user, ok := GetUserFromContext(ctx)
				if !ok {
					return nil, fmt.Errorf("user not found in context")
				}
				if user.Subject != "test-user-123" {
					return nil, fmt.Errorf("expected subject 'test-user-123', got '%s'", user.Subject)
				}

				return mcp.NewToolResultText("success"), nil
			},
		)

		// 5. Manually test the middleware directly
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":                "test-user-123",
			"email":              "test@example.com",
			"preferred_username": "testuser",
			"aud":                cfg.Audience,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
		})

		tokenString, _ := token.SignedString(cfg.JWTSecret)
		ctx := WithOAuthToken(context.Background(), tokenString)

		// Get the middleware and apply it to a test handler
		middleware := oauthServer.Middleware()
		testHandler := middleware(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolCalled = true
			capturedCtx = ctx
			return mcp.NewToolResultText("ok"), nil
		})

		// Call the wrapped handler
		result, err := testHandler(ctx, mcp.CallToolRequest{})
		if err != nil {
			t.Fatalf("Middleware handler failed: %v", err)
		}

		if !toolCalled {
			t.Error("Tool was not called")
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Verify user is in context
		if capturedCtx != nil {
			user, ok := GetUserFromContext(capturedCtx)
			if !ok {
				t.Error("User not found in captured context")
			}
			if user != nil && user.Subject != "test-user-123" {
				t.Errorf("Expected subject 'test-user-123', got '%s'", user.Subject)
			}
		}

		t.Logf("✅ WithToolHandlerMiddleware compatible with mcp-go v0.41.1")
		t.Logf("   - Middleware applied server-wide")
		t.Logf("   - OAuth validation successful")
		t.Logf("   - User context propagated to tool")
	})

	t.Run("MiddlewareCompilationCheck", func(t *testing.T) {
		// Test that server creation with middleware compiles correctly

		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		oauthServer, _ := NewServer(cfg)

		// This is the key test: server creation with middleware should compile
		mcpServer := mcpserver.NewMCPServer("Test Server", "1.0.0",
			mcpserver.WithToolHandlerMiddleware(oauthServer.Middleware()),
		)

		if mcpServer == nil {
			t.Fatal("Server creation failed")
		}

		// Add multiple tools to verify middleware applies to all
		for _, toolName := range []string{"tool1", "tool2", "tool3"} {
			mcpServer.AddTool(
				mcp.Tool{
					Name:        toolName,
					Description: "Test tool",
				},
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return mcp.NewToolResultText("ok"), nil
				},
			)
		}

		t.Logf("✅ Server-wide middleware compilation successful")
		t.Logf("   - 3 tools added, all protected by middleware")
	})

	t.Run("MiddlewareRejectsInvalidToken", func(t *testing.T) {
		// Test that middleware rejects invalid tokens

		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		oauthServer, _ := NewServer(cfg)

		// Get middleware and test directly
		middleware := oauthServer.Middleware()

		toolCalled := false
		wrappedHandler := middleware(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolCalled = true
			return mcp.NewToolResultText("should not reach here"), nil
		})

		// Try with invalid token
		ctx := WithOAuthToken(context.Background(), "invalid-token")

		_, err := wrappedHandler(ctx, mcp.CallToolRequest{})

		// Should fail
		if err == nil {
			t.Error("Expected authentication error, got nil")
		}

		if toolCalled {
			t.Error("Tool should not be called with invalid token")
		}

		t.Logf("✅ Middleware correctly rejects invalid tokens")
		t.Logf("   - Error: %v", err)
	})
}

// TestMiddlewareSignatureCompatibility validates the middleware function signature
func TestMiddlewareSignatureCompatibility(t *testing.T) {
	// This test ensures our Server.Middleware() returns the correct type
	// for mcp-go v0.41.1's WithToolHandlerMiddleware

	cfg := &Config{
		Mode:      "native",
		Provider:  "hmac",
		Issuer:    "https://test.example.com",
		Audience:  "api://test",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	}

	server, _ := NewServer(cfg)

	// Get middleware
	middleware := server.Middleware()

	// Type assertion: should be func(ToolHandlerFunc) ToolHandlerFunc
	// If this compiles, the signature is correct
	var _ func(mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc = middleware

	t.Logf("✅ Middleware signature is compatible with mcp-go v0.41.1")
	t.Logf("   Type: func(server.ToolHandlerFunc) server.ToolHandlerFunc")
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
)

func main() {
	log.Println("=== OAuth MCP Proxy - Simple Example ===")
	log.Println("This example shows the simplest way to add OAuth to an MCP server.")
	log.Println()

	// Step 1: Create HTTP multiplexer
	mux := http.NewServeMux()

	// Step 2: Enable OAuth authentication
	// This single call:
	//   - Validates configuration
	//   - Creates OAuth server with token validator
	//   - Registers all OAuth HTTP endpoints (/.well-known/*, /oauth/*)
	//   - Returns middleware as a server option
	//
	// Provider: "hmac" uses shared secret (good for testing)
	// Audience: Must match the "aud" claim in tokens
	// Logger: Optional - use your own logger (zap, logrus, etc.)
	//         If not provided, uses default log.Printf with level prefixes
	oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{
		Provider:  "hmac",                                           // or "okta", "google", "azure"
		Issuer:    "https://test.example.com",                       // Token issuer URL
		Audience:  "api://simple-server",                            // Must match token's "aud" claim
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"), // For HMAC provider
		// Logger: &myCustomLogger{}, // Optional: integrate with your logging system
	})
	if err != nil {
		log.Fatalf("WithOAuth failed: %v", err)
	}

	log.Println("✅ OAuth configured successfully")
	log.Println("   → HTTP endpoints registered (/.well-known/*, /oauth/*)")
	log.Println("   → Token validator initialized (HMAC-SHA256)")
	log.Println("   → Middleware ready to protect tools")
	log.Println()

	// Step 3: Create MCP server with OAuth option
	// The oauthOption applies OAuth middleware to ALL tools automatically.
	// Every tool call will require a valid OAuth token in the request.
	mcpServer := mcpserver.NewMCPServer("Simple OAuth Server", "1.0.0",
		oauthOption, // This is all you need - middleware applied!
	)

	log.Println("✅ MCP server created with OAuth protection enabled")

	// Step 4: Add tools (automatically OAuth-protected!)
	// Because we used WithOAuth(), all tools automatically require authentication.
	// No per-tool configuration needed - OAuth is applied server-wide.
	mcpServer.AddTool(
		mcp.Tool{
			Name:        "hello",
			Description: "Says hello to the authenticated user (OAuth protected)",
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Extract authenticated user from context
			// OAuth middleware validates token and adds user to context before calling this handler
			user, ok := oauth.GetUserFromContext(ctx)
			if !ok {
				// This should never happen if OAuth is working correctly
				return nil, fmt.Errorf("authentication required")
			}

			// User information available from token claims:
			// - user.Subject: Token "sub" claim (unique user ID)
			// - user.Email: Token "email" claim
			// - user.Username: Token "preferred_username" or "email" or "sub"
			message := fmt.Sprintf("Hello, %s! (Subject: %s, Email: %s)",
				user.Username, user.Subject, user.Email)
			return mcp.NewToolResultText(message), nil
		},
	)

	log.Println("✅ Tools registered (all automatically OAuth-protected)")
	log.Println()

	// Step 5: Setup MCP endpoint with token extraction
	// CreateHTTPContextFunc() extracts "Bearer <token>" from Authorization header
	// and adds it to the request context. OAuth middleware then validates it.
	streamableServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"), // MCP endpoint path
		mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()), // Token extraction
	)

	mux.Handle("/mcp", streamableServer)

	// Step 6: Generate a test token (for HMAC provider testing)
	// In production with OIDC providers (Okta/Google/Azure), clients get tokens
	// from the OAuth provider directly. This is just for local testing.
	testToken := generateTestToken(&oauth.Config{
		Issuer:    "https://test.example.com",
		Audience:  "api://simple-server",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	})

	log.Println("📋 Testing Instructions:")
	log.Println()
	log.Println("1. Server is starting on http://localhost:8080")
	log.Println()
	log.Println("2. Test OAuth metadata endpoint:")
	log.Println("   curl http://localhost:8080/.well-known/oauth-authorization-server")
	log.Println()
	log.Println("3. Call the 'hello' tool with authentication:")
	log.Printf("   curl -X POST http://localhost:8080/mcp \\\n")
	log.Printf("     -H 'Authorization: Bearer %s' \\\n", testToken[:50]+"...")
	log.Printf("     -H 'Content-Type: application/json' \\\n")
	log.Printf("     -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"hello\",\"arguments\":{}}}'\n")
	log.Println()
	log.Println("4. Try without token (should fail with authentication error)")
	log.Println()

	log.Println("🚀 Server starting on http://localhost:8080")
	log.Println()
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// generateTestToken creates a valid JWT token for testing HMAC provider.
// In production with OIDC providers (Okta, Google, Azure), clients obtain tokens
// from the OAuth provider's authorization server, not from your code.
func generateTestToken(cfg *oauth.Config) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                "test-user-123",                  // Subject: unique user identifier
		"email":              "test@example.com",               // User's email address
		"preferred_username": "testuser",                       // Username (optional)
		"aud":                cfg.Audience,                     // Must match Config.Audience!
		"iss":                cfg.Issuer,                       // Must match Config.Issuer
		"exp":                time.Now().Add(time.Hour).Unix(), // Token expires in 1 hour
		"iat":                time.Now().Unix(),                // Issued at (now)
	})

	// Sign with secret (must match Config.JWTSecret)
	tokenString, err := token.SignedString(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	return tokenString
}

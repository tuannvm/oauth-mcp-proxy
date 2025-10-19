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
	log.Println("=== OAuth MCP Proxy - Embedded Mode Example ===")
	log.Println("Phase 2: Package structure + Context propagation")
	log.Println()

	// 1. Configure OAuth (HMAC mode for simplicity)
	cfg := &oauth.Config{
		Mode:      "native",
		Provider:  "hmac",
		Issuer:    "https://test.example.com",
		Audience:  "api://test-mcp-server",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	}

	// 2. Create OAuth server
	// Phase 2 features demonstrated:
	// - provider/ package isolation (HMACValidator from provider/)
	// - Context propagation (ValidateToken accepts context.Context)
	// - Instance-scoped state (Server has own cache)
	oauthServer, err := oauth.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create OAuth server: %v", err)
	}
	log.Println("✅ OAuth server created (provider/ package)")
	log.Println("   - HMACValidator from provider/ subpackage")
	log.Println("   - Instance-scoped cache (no globals)")
	log.Println("   - Context propagation enabled")

	// 3. Create MCP server with OAuth middleware applied to ALL tools
	// Using mcp-go v0.41.1's WithToolHandlerMiddleware option
	mcpServer := mcpserver.NewMCPServer("Hello World MCP Server", "1.0.0",
		mcpserver.WithToolHandlerMiddleware(oauthServer.Middleware()),
	)

	// 4. Define tool handler
	// Context flow: HTTP Request → MCP → OAuth Middleware → ValidateToken(ctx) → Tool Handler
	helloHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get authenticated user from context (set by OAuth middleware)
		// The ctx here has traveled through: HTTP → MCP → OAuth validation chain
		user, ok := oauth.GetUserFromContext(ctx)
		if !ok {
			return mcp.NewToolResultError("Authentication required"), nil
		}

		message := fmt.Sprintf("Hello, %s! Your email is %s (Subject: %s)",
			user.Username, user.Email, user.Subject)

		return mcp.NewToolResultText(message), nil
	}

	// 5. Add tool to MCP server
	// OAuth middleware is automatically applied (server-wide)
	mcpServer.AddTool(
		mcp.Tool{
			Name:        "hello",
			Description: "Says hello to the authenticated user (OAuth protected)",
			InputSchema: mcp.ToolInputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		helloHandler, // OAuth middleware applied automatically by server!
	)

	log.Println("✅ MCP server created with OAuth middleware")
	log.Println("   - All tools protected by OAuth (server-wide)")

	// 6. Setup HTTP server
	mux := http.NewServeMux()

	// Register OAuth endpoints
	oauthServer.RegisterHandlers(mux)
	log.Println("✅ OAuth handlers registered")

	// Setup MCP endpoint with OAuth context extraction
	oauthContextFunc := func(ctx context.Context, r *http.Request) context.Context {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := authHeader
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
			ctx = oauth.WithOAuthToken(ctx, token)
		}
		return ctx
	}

	streamableServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithHTTPContextFunc(oauthContextFunc),
	)

	mux.Handle("/mcp", streamableServer)
	log.Println("✅ MCP endpoint configured at /mcp")

	// Generate test token
	testToken := generateTestToken(cfg)
	log.Println()
	log.Println("📋 Testing Instructions:")
	log.Println()
	log.Println("1. Start the server:")
	log.Println("   go run examples/embedded.go")
	log.Println()
	log.Println("2. Test OAuth metadata:")
	log.Println("   curl http://localhost:8080/.well-known/oauth-authorization-server")
	log.Println()
	log.Println("3. Call MCP tools with token:")
	log.Printf("   curl -X POST http://localhost:8080/mcp \\\n")
	log.Printf("     -H 'Authorization: Bearer %s' \\\n", testToken[:50]+"...")
	log.Printf("     -H 'Content-Type: application/json' \\\n")
	log.Printf("     -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"hello\",\"arguments\":{}}}'\n")
	log.Println()

	// Start server
	log.Println("🚀 Server starting on http://localhost:8080")
	log.Println()

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// generateTestToken creates a valid HMAC token for testing
func generateTestToken(cfg *oauth.Config) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "test-user-123",
		"email": "test@example.com",
		"name":  "Test User",
		"aud":   cfg.Audience,
		"iss":   cfg.Issuer,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})

	tokenString, err := token.SignedString(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	return tokenString
}

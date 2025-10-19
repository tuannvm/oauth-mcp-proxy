package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	oauth "github.com/tuannvm/oauth-mcp-proxy"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("=== OAuth MCP Proxy - Embedded Mode Example ===")
	log.Println("Creating a simple MCP server with OAuth authentication")
	log.Println()

	// 1. Configure OAuth (HMAC mode for simplicity)
	cfg := &oauth.Config{
		Mode:      "native",
		Provider:  "hmac",
		Issuer:    "https://test.example.com",
		Audience:  "api://test-mcp-server",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	}

	// 2. Create OAuth server (Phase 1.5 API)
	oauthServer, err := oauth.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create OAuth server: %v", err)
	}
	log.Println("✅ OAuth server created (HMAC, instance-scoped cache)")

	// 3. Create MCP server with a simple tool
	mcpServer := mcpserver.NewMCPServer("Hello World MCP Server", "1.0.0")

	// Get OAuth middleware
	middleware := oauthServer.Middleware()

	// Define tool handler
	helloHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get authenticated user from context (set by middleware)
		user, ok := oauth.GetUserFromContext(ctx)
		if !ok {
			return mcp.NewToolResultError("Authentication required"), nil
		}

		message := fmt.Sprintf("Hello, %s! Your email is %s (Subject: %s)",
			user.Username, user.Email, user.Subject)

		return mcp.NewToolResultText(message), nil
	}

	// Wrap handler with OAuth middleware
	protectedHandler := middleware(helloHandler)

	// Add protected tool to MCP server
	mcpServer.AddTool(
		mcp.Tool{
			Name:        "hello",
			Description: "Says hello to the authenticated user (OAuth protected)",
			InputSchema: mcp.ToolInputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		protectedHandler, // OAuth middleware applied!
	)

	log.Println("✅ MCP server created with OAuth-protected 'hello' tool")

	// 5. Setup HTTP server
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

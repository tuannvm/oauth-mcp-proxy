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
	log.Println("=== OAuth MCP Proxy - Simple API (Phase 3) ===")
	log.Println()

	// Setup HTTP mux
	mux := http.NewServeMux()

	// Line 1: Get OAuth server option (registers HTTP handlers)
	oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{
		Provider:  "hmac",
		Issuer:    "https://test.example.com",
		Audience:  "api://simple-server",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	})
	if err != nil {
		log.Fatalf("WithOAuth failed: %v", err)
	}

	log.Println("✅ OAuth configured")
	log.Println("   - HTTP handlers registered")
	log.Println("   - Middleware ready")

	// Line 2: Create MCP server with OAuth option
	mcpServer := mcpserver.NewMCPServer("Simple OAuth Server", "1.0.0",
		oauthOption, // OAuth middleware applied to ALL tools!
	)

	log.Println("✅ MCP server created with OAuth middleware")

	// Add tools (automatically protected by OAuth)
	mcpServer.AddTool(
		mcp.Tool{
			Name:        "hello",
			Description: "Says hello to authenticated user",
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			user, ok := oauth.GetUserFromContext(ctx)
			if !ok {
				return nil, fmt.Errorf("authentication required")
			}

			message := fmt.Sprintf("Hello, %s! (Subject: %s)", user.Username, user.Subject)
			return mcp.NewToolResultText(message), nil
		},
	)

	log.Println("✅ Tools added (automatically OAuth-protected)")

	// Setup MCP endpoint with OAuth context extraction
	streamableServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
	)

	mux.Handle("/mcp", streamableServer)

	// Generate test token
	testToken := generateTestToken(&oauth.Config{
		Issuer:    "https://test.example.com",
		Audience:  "api://simple-server",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	})

	log.Println()
	log.Println("📋 Test Command:")
	log.Printf("curl -X POST http://localhost:8080/mcp \\\n")
	log.Printf("  -H 'Authorization: Bearer %s' \\\n", testToken[:50]+"...")
	log.Printf("  -H 'Content-Type: application/json' \\\n")
	log.Printf("  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"hello\",\"arguments\":{}}}'\n")
	log.Println()

	log.Println("🚀 Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func generateTestToken(cfg *oauth.Config) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                "test-user",
		"email":              "test@example.com",
		"preferred_username": "testuser",
		"aud":                cfg.Audience,
		"iss":                cfg.Issuer,
		"exp":                time.Now().Add(time.Hour).Unix(),
		"iat":                time.Now().Unix(),
	})

	tokenString, _ := token.SignedString(cfg.JWTSecret)
	return tokenString
}

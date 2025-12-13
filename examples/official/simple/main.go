package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
	mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

func main() {
	// 1. Create HTTP multiplexer
	mux := http.NewServeMux()

	// 2. Create MCP server using official SDK
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "Official SDK OAuth Example",
		Version: "1.0.0",
	}, nil)

	// 3. Add a simple greeting tool (all tools will be OAuth-protected)
	type GreetParams struct{}

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "greet",
		Description: "Greets the authenticated user by their username",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *GreetParams) (*mcp.CallToolResult, any, error) {
		// Access the authenticated user from context
		user, ok := oauth.GetUserFromContext(ctx)
		if !ok {
			return nil, nil, fmt.Errorf("authentication required")
		}

		message := fmt.Sprintf("Hello, %s! Your email is: %s", user.Username, user.Email)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: message},
			},
		}, nil, nil
	})

	// 4. Configure OAuth with Okta
	// Set these environment variables:
	//   export OKTA_DOMAIN="dev-12345.okta.com"
	//   export OKTA_AUDIENCE="api://my-mcp-server"
	//   export SERVER_URL="http://localhost:8080"
	cfg := &oauth.Config{
		Provider:  "okta",
		Issuer:    fmt.Sprintf("https://%s", getEnv("OKTA_DOMAIN", "dev-12345.okta.com")),
		Audience:  getEnv("OKTA_AUDIENCE", "api://my-mcp-server"),
		ServerURL: getEnv("SERVER_URL", "http://localhost:8080"),
	}

	// 5. Enable OAuth protection - returns an http.Handler
	oauthServer, handler, err := mcpoauth.WithOAuth(mux, cfg, mcpServer)
	if err != nil {
		log.Fatalf("OAuth setup failed: %v", err)
	}

	// 6. Log OAuth information
	oauthServer.LogStartup(false) // false = HTTP (set true if using HTTPS)

	// 7. Start server
	// Note: PORT is the local bind port. If you change SERVER_URL port
	// (e.g., http://localhost:9000), also set PORT=9000 to match.
	// For production with reverse proxy, PORT is your local port while
	// SERVER_URL is the public URL (e.g., SERVER_URL=https://api.example.com, PORT=8080)
	port := getEnv("PORT", "8080")
	addr := ":" + port

	log.Printf("Starting MCP server on %s", addr)
	log.Printf("OAuth Provider: Okta")
	log.Printf("Issuer: https://%s", getEnv("OKTA_DOMAIN", "dev-12345.okta.com"))
	log.Printf("Audience: %s", getEnv("OKTA_AUDIENCE", "api://my-mcp-server"))
	log.Println("\nMake sure to set your Okta environment variables:")
	log.Println("  export OKTA_DOMAIN=dev-12345.okta.com")
	log.Println("  export OKTA_AUDIENCE=api://my-mcp-server")
	log.Println("  export SERVER_URL=http://localhost:8080")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

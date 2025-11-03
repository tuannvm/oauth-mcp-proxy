package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
	"github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

func main() {
	// 1. Create HTTP multiplexer
	mux := http.NewServeMux()

	// 2. Configure OAuth with Okta
	// Set these environment variables:
	//   export OKTA_DOMAIN="dev-12345.okta.com"              (your Okta domain)
	//   export OKTA_AUDIENCE="api://my-mcp-server"          (your API identifier)
	//   export SERVER_URL="https://mcp.example.com"         (your server URL)
	oauthServer, oauthOption, err := mark3labs.WithOAuth(mux, &oauth.Config{
		Provider:  "okta",
		Issuer:    fmt.Sprintf("https://%s", getEnv("OKTA_DOMAIN", "dev-12345.okta.com")),
		Audience:  getEnv("OKTA_AUDIENCE", "api://my-mcp-server"),
		ServerURL: getEnv("SERVER_URL", "http://localhost:8080"),
	})
	if err != nil {
		log.Fatalf("OAuth setup failed: %v", err)
	}

	// 3. Create MCP server with OAuth
	mcpServer := mcpserver.NewMCPServer("My MCP Server", "1.0.0", oauthOption)

	// 4. Add your tools (all automatically OAuth-protected)
	mcpServer.AddTool(
		mcp.Tool{
			Name:        "greet",
			Description: "Greets the authenticated user",
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			user, ok := oauth.GetUserFromContext(ctx)
			if !ok {
				return nil, fmt.Errorf("authentication required")
			}
			message := fmt.Sprintf("Hello, %s!", user.Username)
			return mcp.NewToolResultText(message), nil
		},
	)

	// 5. Setup MCP endpoint with automatic 401 handling
	streamableServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
	)
	mux.HandleFunc("/mcp", oauthServer.WrapMCPEndpoint(streamableServer))

	// 6. Start server
	// Note: PORT is the local bind port. If you change SERVER_URL port
	// (e.g., http://localhost:9000), also set PORT=9000 to match.
	// For production with reverse proxy, PORT is your local port while
	// SERVER_URL is the public URL (e.g., SERVER_URL=https://api.example.com, PORT=8080)
	port := getEnv("PORT", "8080")
	log.Printf("Starting MCP server on :%s", port)
	log.Printf("OAuth Provider: Okta")
	log.Printf("Issuer: https://%s", getEnv("OKTA_DOMAIN", "dev-12345.okta.com"))
	log.Printf("Audience: %s", getEnv("OKTA_AUDIENCE", "api://my-mcp-server"))
	log.Println("\nMake sure to set your Okta environment variables:")
	log.Println("  export OKTA_DOMAIN=dev-12345.okta.com")
	log.Println("  export OKTA_AUDIENCE=api://my-mcp-server")
	log.Println("  export SERVER_URL=http://localhost:8080")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

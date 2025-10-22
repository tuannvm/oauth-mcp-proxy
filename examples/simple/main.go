package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
	"github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

func main() {
	// 1. Create HTTP multiplexer
	mux := http.NewServeMux()

	// 2. Enable OAuth authentication
	_, oauthOption, err := mark3labs.WithOAuth(mux, &oauth.Config{
		Provider:  "okta", // or "hmac", "google", "azure"
		Issuer:    "https://your-company.okta.com",
		Audience:  "api://your-mcp-server",
		ServerURL: "https://your-server.com",
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

	// 5. Setup MCP endpoint
	streamableServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
	)
	mux.Handle("/mcp", streamableServer)

	// 6. Start server
	log.Println("Starting MCP server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

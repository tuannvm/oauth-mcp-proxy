package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
	mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

func main() {
	// Feature 1: ConfigBuilder - Auto-generates ServerURL from host/port/TLS
	// Set these environment variables:
	//   export OKTA_DOMAIN="dev-12345.okta.com"
	//   export OKTA_AUDIENCE="api://my-mcp-server"
	cfg, err := oauth.NewConfigBuilder().
		WithProvider("okta").
		WithIssuer(fmt.Sprintf("https://%s", getEnv("OKTA_DOMAIN", "dev-12345.okta.com"))).
		WithAudience(getEnv("OKTA_AUDIENCE", "api://my-mcp-server")).
		WithHost(getEnv("MCP_HOST", "localhost")).
		WithPort(getEnv("MCP_PORT", "8080")).
		WithTLS(getEnv("HTTPS_CERT_FILE", "") != "").
		Build()

	if err != nil {
		log.Fatalf("Config setup failed: %v", err)
	}

	// Feature 2: Create MCP server with multiple tools
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "Advanced Official SDK Example",
		Version: "1.0.0",
	}, nil)

	// Tool 1: Greet user
	type GreetParams struct{}
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "greet",
		Description: "Greets the authenticated user",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *GreetParams) (*mcp.CallToolResult, any, error) {
		user, ok := oauth.GetUserFromContext(ctx)
		if !ok {
			return nil, nil, fmt.Errorf("authentication required")
		}

		message := fmt.Sprintf("Hello, %s! Your email is: %s", user.Username, user.Email)
		log.Printf("[greet] Called by user: %s", user.Username)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: message},
			},
		}, nil, nil
	})

	// Tool 2: Get user info
	type UserInfoParams struct{}
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "whoami",
		Description: "Returns information about the authenticated user",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *UserInfoParams) (*mcp.CallToolResult, any, error) {
		user, ok := oauth.GetUserFromContext(ctx)
		if !ok {
			return nil, nil, fmt.Errorf("authentication required")
		}

		info := fmt.Sprintf(`User Information:
- Subject: %s
- Username: %s
- Email: %s`, user.Subject, user.Username, user.Email)

		log.Printf("[whoami] Called by user: %s", user.Username)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: info},
			},
		}, nil, nil
	})

	// Tool 3: Server time
	type TimeParams struct{}
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "server_time",
		Description: "Returns the current server time",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *TimeParams) (*mcp.CallToolResult, any, error) {
		user, ok := oauth.GetUserFromContext(ctx)
		if !ok {
			return nil, nil, fmt.Errorf("authentication required")
		}

		now := time.Now().Format(time.RFC3339)
		message := fmt.Sprintf("Server time: %s (requested by %s)", now, user.Username)

		log.Printf("[server_time] Called by user: %s", user.Username)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: message},
			},
		}, nil, nil
	})

	// Feature 3: Enable OAuth with official SDK
	mux := http.NewServeMux()
	oauthServer, handler, err := mcpoauth.WithOAuth(mux, cfg, mcpServer)
	if err != nil {
		log.Fatalf("OAuth setup failed: %v", err)
	}

	// Feature 4: LogStartup - Displays OAuth endpoint information
	useTLS := getEnv("HTTPS_CERT_FILE", "") != ""
	oauthServer.LogStartup(useTLS)

	// Additional OAuth endpoints available via helper methods
	log.Printf("\nOAuth Discovery URLs:")
	log.Printf("  - Metadata: %s", oauthServer.GetAuthorizationServerMetadataURL())
	log.Printf("  - OIDC Discovery: %s", oauthServer.GetOIDCDiscoveryURL())

	// Feature 5: Server status
	log.Printf("\n%s", oauthServer.GetStatusString(useTLS))
	log.Printf("Tools registered: greet, whoami, server_time")

	// Start server
	port := getEnv("MCP_PORT", "8080")
	addr := ":" + port

	log.Printf("\nStarting MCP server on %s", addr)
	log.Println("\nMake sure to set your Okta environment variables:")
	log.Println("  export OKTA_DOMAIN=dev-12345.okta.com")
	log.Println("  export OKTA_AUDIENCE=api://my-mcp-server")
	log.Println("\nTo test, obtain an access token from Okta and use:")
	log.Printf("  curl -H 'Authorization: Bearer <okta-token>' http://localhost:%s", port)

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

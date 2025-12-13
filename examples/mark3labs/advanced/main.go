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

	// Alternative: cfg, _ := oauth.FromEnv()  // Reads from environment variables

	mux := http.NewServeMux()

	// Feature 2: WithOAuth returns Server instance for helper methods
	oauthServer, oauthOption, err := mark3labs.WithOAuth(mux, cfg)
	if err != nil {
		log.Fatalf("OAuth setup failed: %v", err)
	}

	// Feature 3: GetHTTPServerOptions - Returns OAuth-required options
	mcpServer := mcpserver.NewMCPServer("Advanced MCP Server", "1.0.0", oauthOption)

	oauthOpts := oauthServer.GetHTTPServerOptions()
	httpOpts := append(oauthOpts,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithStateLess(false),
	)
	streamableServer := mcpserver.NewStreamableHTTPServer(mcpServer, httpOpts...)

	// Feature 4: WrapMCPEndpoint - Automatic 401 handling with CORS support
	mcpHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		streamableServer.ServeHTTP(w, r)
	}

	mux.HandleFunc("/mcp", oauthServer.WrapMCPEndpoint(http.HandlerFunc(mcpHandler)))

	// Add status endpoint (not OAuth protected)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Feature 5: GetStatusString - Human-readable OAuth status
		useTLS := getEnv("HTTPS_CERT_FILE", "") != ""
		_, _ = fmt.Fprintf(w, `{"status":"ok","oauth":"%s"}`, oauthServer.GetStatusString(useTLS))
	})

	// Add OAuth-protected tools
	mcpServer.AddTool(
		mcp.Tool{
			Name:        "get_user_info",
			Description: "Returns authenticated user information",
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			user, ok := oauth.GetUserFromContext(ctx)
			if !ok {
				return nil, fmt.Errorf("authentication required")
			}
			info := fmt.Sprintf("User: %s\nEmail: %s\nSubject: %s",
				user.Username, user.Email, user.Subject)
			return mcp.NewToolResultText(info), nil
		},
	)

	// Feature 6: LogStartup - Logs all OAuth endpoints with warnings
	useTLS := getEnv("HTTPS_CERT_FILE", "") != "" && getEnv("HTTPS_KEY_FILE", "") != ""
	oauthServer.LogStartup(useTLS)

	// Feature 7: Discovery URL helpers - Programmatic endpoint access
	log.Printf("Server starting on %s", cfg.ServerURL)
	log.Printf("OAuth callback: %s", oauthServer.GetCallbackURL())
	log.Printf("Total endpoints: %d", len(oauthServer.GetAllEndpoints()))

	// Start server
	port := getEnv("MCP_PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	if useTLS {
		certFile := getEnv("HTTPS_CERT_FILE", "")
		keyFile := getEnv("HTTPS_KEY_FILE", "")
		if err := http.ListenAndServeTLS(addr, certFile, keyFile, mux); err != nil {
			log.Fatalf("HTTPS server failed: %v", err)
		}
	} else {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

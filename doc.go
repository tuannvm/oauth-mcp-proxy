// Package oauth provides OAuth 2.1 authentication for Go MCP servers.
//
// This library supports both major MCP SDKs:
//   - github.com/mark3labs/mcp-go (mark3labs adapter)
//   - github.com/modelcontextprotocol/go-sdk (official mcp adapter)
//
// Quick Start with mark3labs/mcp-go:
//
//	import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
//
//	oauthServer, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://your-mcp-server",
//	})
//	mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
//	streamableServer := server.NewStreamableHTTPServer(mcpServer, ...)
//	mux.HandleFunc("/mcp", oauthServer.WrapMCPEndpoint(streamableServer))
//
// Quick Start with Official SDK:
//
//	import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
//
//	mcpServer := mcp.NewServer(&mcp.Implementation{...}, nil)
//	_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://your-mcp-server",
//	}, mcpServer)
//	http.ListenAndServe(":8080", handler)
//
// Features:
//
//   - Simple 3-line integration via WithOAuth()
//   - Automatic 401 handling with RFC 6750 compliance
//   - Fast token caching (5-minute TTL, JWT expiry-aware)
//   - Multiple providers: HMAC, Okta, Google, Azure AD
//   - Security hardening: state replay protection, DoS prevention, input validation
//   - Built-in rate limiter
//   - CORS support (OPTIONS pass-through)
//
// Configuration:
//
// Configure via Config struct, ConfigBuilder, or environment variables:
//
//	cfg, _ := oauth.NewConfigBuilder().
//	    WithProvider("okta").
//	    WithIssuer("https://company.okta.com").
//	    WithAudience("api://my-server").
//	    Build()
//
// OAuth Modes:
//
// Native Mode (default, recommended): Client handles OAuth flow directly.
// Server only validates tokens. Use with OAuth-capable clients like Claude Desktop.
//
// Proxy Mode: Server proxies OAuth flow for simple clients (CLI tools, legacy apps).
// Requires ClientID, ClientSecret, ServerURL, and RedirectURIs configuration.
//
// Accessing Authenticated User:
//
// In tool handlers, use GetUserFromContext to access the authenticated user:
//
//	func toolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//	    user, ok := oauth.GetUserFromContext(ctx)
//	    if !ok {
//	        return nil, fmt.Errorf("authentication required")
//	    }
//	    // Use user.Subject, user.Email, user.Username
//	}
//
// Security Features:
//
//   - State replay protection (timestamp + nonce with HMAC-SHA256)
//   - Token cache with JWT expiry awareness (uses min(token.expiry, now+5min))
//   - Constant-time HMAC comparison (timing attack prevention)
//   - Secure nonce generation (panics on crypto/rand failure)
//   - Issuer URL validation (HTTPS enforced for OIDC providers)
//   - Input validation (parameter length limits, request body size limits)
//   - Rate limiting (fixed-window token bucket algorithm)
//   - auth.TokenInfo population for go-sdk session management
//
// Supported Providers:
//
//   - HMAC: Testing and development (use JWTSecret)
//   - Okta: Enterprise SSO (use Issuer + Audience)
//   - Google: Google Workspace (use Issuer + Client ID as Audience)
//   - Azure AD: Microsoft 365 (use Issuer + Application ID)
//
// Documentation:
//
// Full documentation: https://github.com/tuannvm/oauth-mcp-proxy
//
//   - docs/CONFIGURATION.md - All configuration options
//   - docs/SECURITY.md - Production best practices
//   - docs/providers/ - Provider-specific setup guides
//   - examples/ - Working code examples
package oauth

# oauth-mcp-proxy

**Add OAuth 2.1 authentication to your Go MCP server in 3 lines of code.**

```go
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{Provider: "okta", ...})
mcpServer := server.NewMCPServer("My Server", "1.0.0", oauthOption)
// Done! All tools OAuth-protected.
```

---

## Why oauth-mcp-proxy?

**Without this library:** 100+ lines of OAuth boilerplate, token validation, PKCE, security concerns.

**With this library:** 3 lines. Production-ready OAuth with best practices built-in.

---

## Features

- **3-Line Integration** - `WithOAuth()` does everything
- **4 Providers** - HMAC, Okta, Google, Azure AD
- **OAuth 2.1** - Native + Proxy modes, PKCE, token caching
- **Pluggable Logging** - Integrate with zap, logrus, slog
- **Production Ready** - Instance-scoped, no globals, security hardened
- **MCP-Native** - Built specifically for `mcp-go` servers

---

## Quick Start (5 Minutes)

### 1. Install

```bash
go get github.com/tuannvm/oauth-mcp-proxy
```

### 2. Add OAuth to Your MCP Server

```go
package main

import (
    "net/http"
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
    mux := http.NewServeMux()

    // Enable OAuth (auto-registers endpoints, applies middleware)
    oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{
        Provider:  "okta",                        // or "hmac", "google", "azure"
        Issuer:    "https://yourcompany.okta.com",
        Audience:  "api://your-mcp-server",
        // Optional for proxy mode:
        // ClientID: "...", ClientSecret: "...", ServerURL: "...", RedirectURIs: "..."
    })
    if err != nil {
        panic(err)
    }

    // Create MCP server with OAuth
    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

    // Add your tools - automatically OAuth-protected!
    mcpServer.AddTool(myTool, myToolHandler)

    // Setup MCP endpoint
    mux.Handle("/mcp", mcpserver.NewStreamableHTTPServer(
        mcpServer,
        mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
    ))

    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

### 3. Access Authenticated User in Tools

```go
func myToolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    user, ok := oauth.GetUserFromContext(ctx)
    if !ok {
        return nil, fmt.Errorf("not authenticated")
    }

    // user.Subject, user.Email, user.Username available
    return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", user.Username)), nil
}
```

**That's it!** Your MCP server now requires OAuth authentication.

---

## Configuration Options

```go
type Config struct {
    // Required
    Provider string // "hmac", "okta", "google", "azure"
    Audience string // Your API audience (e.g., "api://my-server")

    // Provider-specific
    Issuer    string // OIDC issuer URL (Okta/Google/Azure)
    JWTSecret []byte // Secret key (HMAC provider only)

    // Optional - OAuth Mode
    Mode string // "native" (default) or "proxy" - auto-detected

    // Optional - Proxy Mode (client can't do OAuth)
    ClientID     string // OAuth client ID
    ClientSecret string // OAuth client secret
    ServerURL    string // Your server's public URL
    RedirectURIs string // Allowed redirect URIs

    // Optional - Custom Logging
    Logger Logger // Your logger (zap, logrus, etc.) - uses default if nil
}
```

### OAuth Modes

**Native Mode** (recommended): Client handles OAuth directly with provider
- Use when: Claude Desktop, browser-based clients
- Config: Just Provider, Issuer, Audience

**Proxy Mode**: Server proxies OAuth flow for client
- Use when: Simple CLI tools, legacy clients
- Config: All fields including ClientID, ServerURL, RedirectURIs

Mode is auto-detected based on whether `ClientID` is provided.

---

## Provider Setup

Choose your OAuth provider:

### HMAC (Shared Secret)

**Use when:** Testing, simple deployments, service-to-service

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-server",
    JWTSecret: []byte("your-32-byte-secret-key-here"),
})
```

Generate tokens with `jwt.NewWithClaims()`. See [HMAC Guide](docs/providers/HMAC.md).

### Okta

**Use when:** Enterprise SSO, user management

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://yourcompany.okta.com",
    Audience: "api://your-server",
})
```

Setup: [Okta Guide](docs/providers/OKTA.md)

### Google

**Use when:** Google Workspace integration

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "google",
    Issuer:   "https://accounts.google.com",
    Audience: "your-client-id.apps.googleusercontent.com",
})
```

Setup: [Google Guide](docs/providers/GOOGLE.md)

### Azure AD

**Use when:** Microsoft 365 integration

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "azure",
    Issuer:   "https://login.microsoftonline.com/{tenant}/v2.0",
    Audience: "api://your-app-id",
})
```

Setup: [Azure AD Guide](docs/providers/AZURE.md)

---

## Custom Logger

Integrate with your logging system:

```go
// Implement Logger interface
type MyLogger struct{ logger *zap.Logger }

func (l *MyLogger) Debug(msg string, args ...interface{}) { l.logger.Sugar().Debugf(msg, args...) }
func (l *MyLogger) Info(msg string, args ...interface{})  { l.logger.Sugar().Infof(msg, args...) }
func (l *MyLogger) Warn(msg string, args ...interface{})  { l.logger.Sugar().Warnf(msg, args...) }
func (l *MyLogger) Error(msg string, args ...interface{}) { l.logger.Sugar().Errorf(msg, args...) }

// Use in config
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Audience: "api://my-server",
    Logger:   &MyLogger{logger: zapLogger},
})
```

Default logger uses `log.Printf` with level prefixes.

---

## Security Best Practices

🔒 **See [SECURITY.md](docs/SECURITY.md) for complete guide**

**Quick checklist:**
- ✅ Use HTTPS in production
- ✅ Never commit secrets (use environment variables)
- ✅ Validate audience matches your server
- ✅ Use strong JWTSecret (32+ bytes) for HMAC
- ✅ Enable PKCE for public clients (auto-enabled)
- ✅ Regularly rotate secrets

---

## Troubleshooting

### "Authentication required: missing OAuth token"
- Check: Is `Authorization: Bearer <token>` header present?
- Check: Did you call `WithHTTPContextFunc(oauth.CreateHTTPContextFunc())`?

### "Authentication failed: invalid token"
- Check: Token issuer matches config Issuer
- Check: Token audience matches config Audience
- Check: Token not expired
- Check: For HMAC, secret matches

### "Invalid redirect URI"
- Native mode: Client's redirect must be localhost for security
- Proxy mode: Redirect must be in RedirectURIs allowlist

### Token caching not working
- Tokens cached 5 minutes by default
- Cache is instance-scoped (per Server)

---

## Examples

- **[Simple Example](examples/simple/)** - Production-ready 3-line integration
- **[Advanced Example](examples/embedded/)** - Lower-level API for customization

---

## Migration from mcp-trino

Using `mcp-trino`'s OAuth? See [MIGRATION.md](docs/MIGRATION.md) for upgrade guide.

---

## Development Status

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 0-4 | ✅ Complete | Core library + tests |
| **Phase 5** | 🔄 Current | Documentation + examples |
| Phase 6 | ⏳ Next | mcp-trino migration validation |

**Status:** Ready for v0.1.0 after Phase 5 completion.

---

## Dependencies

4 well-maintained dependencies:
- **mcp-go** (v0.41.1) - MCP protocol
- **go-oidc** (v3.16.0) - OIDC validation
- **jwt** (v5.3.0) - JWT validation
- **oauth2** (v0.32.0) - OAuth flows

---

## Contributing

Not accepting contributions during extraction phase. After v0.1.0 release, contributions welcome!

---

## License

MIT License - See [LICENSE](LICENSE)

---

## Links

- **[Implementation Plan](docs/plan.md)** - v0.1.0 roadmap
- **[Provider Guides](docs/providers/)** - Setup instructions
- **[Security Guide](docs/SECURITY.md)** - Best practices
- **[Migration Guide](docs/MIGRATION.md)** - From mcp-trino
- **[Examples](examples/)** - Working code samples

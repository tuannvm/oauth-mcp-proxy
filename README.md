# oauth-mcp-proxy

OAuth 2.1 authentication library for MCP servers

## Status

🚧 **In Development** - Extracting from [mcp-trino](https://github.com/tuannvm/mcp-trino)

**Current Phase:** Phase 4 - Complete (Ready for v0.1.0)

## Quick Links

- **[v0.1.0 Plan (Embedded)](docs/plan.md)** ← Embedded mode plan (current)
- **[v0.2.0 Plan (Standalone)](docs/plan-standalone.md)** ← Proxy service plan (future)
- **[Implementation Log](docs/implementation.md)** ← Progress tracking
- [OAuth Architecture](docs/oauth.md) - Original OAuth design from mcp-trino
- [Archived Plans](docs/archive/) - Historical planning documents

## Overview

`oauth-mcp-proxy` is a standalone OAuth 2.1 authentication library extracted from mcp-trino. It enables any Go-based MCP server to add OAuth authentication with minimal integration.

### Features (v0.1.0)

- **Embedded Mode:** Library for MCP servers
- **4 Providers:** HMAC, Okta, Google, Azure AD
- **OAuth 2.1:** Native + Proxy modes, PKCE support
- **Production Ready:** Token caching, graceful shutdown

**Note:** Standalone mode (proxy service) deferred to v0.2.0 - see [plan-standalone.md](docs/plan-standalone.md)

## Installation

```bash
# Coming soon - library not yet published
go get github.com/tuannvm/oauth-mcp-proxy
```

## Dependencies

This library requires 4 external dependencies:

- **`github.com/mark3labs/mcp-go`** (v0.41.1) - MCP protocol and server
- **`github.com/coreos/go-oidc/v3`** (v3.16.0) - OIDC/JWKS validation
- **`github.com/golang-jwt/jwt/v5`** (v5.3.0) - JWT/HMAC validation
- **`golang.org/x/oauth2`** (v0.32.0) - OAuth 2.0 flows

All dependencies are well-maintained, industry-standard Go libraries.

## Features

- **Pluggable Logging** - Provide your own logger or use the default
- **4 Providers** - HMAC, Okta, Google, Azure AD
- **OAuth 2.1** - Native + Proxy modes with PKCE support
- **Instance-scoped** - No global state, supports multiple instances
- **Production Ready** - Token caching, validation, security best practices

## Usage

### Embedded Mode (Library)

```go
import oauth "github.com/tuannvm/oauth-mcp-proxy"

// One function call - OAuth enabled!
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
    ClientID: "client-id",      // Optional: triggers proxy mode
    ClientSecret: "secret",      // Optional: for proxy mode
})

mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

// All tools automatically OAuth-protected!
```

### Custom Logger

Provide your own logger to control OAuth logging:

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...interface{}) { /* custom */ }
func (l *MyLogger) Info(msg string, args ...interface{})  { /* custom */ }
func (l *MyLogger) Warn(msg string, args ...interface{})  { /* custom */ }
func (l *MyLogger) Error(msg string, args ...interface{}) { /* custom */ }

oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
    Provider: "hmac",
    Audience: "api://my-server",
    JWTSecret: []byte("secret"),
    Logger: &MyLogger{}, // Use custom logger
})
```

If no logger provided, uses default (`log.Printf` with `[INFO]`, `[ERROR]`, etc. prefixes).

### Standalone Mode (v0.2.0+)

Deferred to v0.2.0 - See [plan-standalone.md](docs/plan-standalone.md) for details

## Development Status

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 0 | ✅ Complete | Repository setup, copy code |
| Phase 1 | ✅ Complete | Make it compile |
| Phase 1.5 | ✅ Complete | Fix critical architecture (global state, logging, validation) |
| Phase 2 | ✅ Complete | Package structure (provider/ only) |
| Phase 3 | ✅ Complete | Implement WithOAuth() API |
| Phase 4 | ✅ Complete | OAuth-only tests (validate it works!) |
| Phase 5 | ⏳ Next | Documentation + examples |
| Phase 6 | ⏳ Planned | mcp-trino migration |

## Development

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linters
make lint

# Format code
make fmt
```

## Contributing

Not accepting contributions yet - extraction in progress.

## License

MIT License - See [LICENSE](LICENSE) for details

## Related Projects

- [mcp-trino](https://github.com/tuannvm/mcp-trino) - Source of OAuth implementation
- [mcp-go](https://github.com/mark3labs/mcp-go) - MCP protocol library

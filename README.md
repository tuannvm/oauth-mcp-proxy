# oauth-mcp-proxy

OAuth 2.1 authentication library for MCP servers

## Status

🚧 **In Development** - Extracting from [mcp-trino](https://github.com/tuannvm/mcp-trino)

**Current Phase:** Phase 2 - Package Structure

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

## Usage

### Embedded Mode (Library)

```go
import oauth "github.com/tuannvm/oauth-mcp-proxy"

// One function call - OAuth enabled!
oauth.EnableOAuth(mcpServer, mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
    ClientID: "client-id",      // Optional: triggers proxy mode
    ClientSecret: "secret",      // Optional: for proxy mode
})

// See docs/plan.md for native vs proxy mode examples
```

### Standalone Mode (v0.2.0+)

Deferred to v0.2.0 - See [plan-standalone.md](docs/plan-standalone.md) for details

## Development Status

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 0 | ✅ Complete | Repository setup, copy code |
| Phase 1 | ✅ Complete | Make it compile |
| Phase 1.5 | ✅ Complete | Fix critical architecture (global state, logging, validation) |
| Phase 2 | 🔄 Current | Package structure (provider/ only) |
| Phase 3 | ⏳ Planned | Implement EnableOAuth() API |
| Phase 4 | ⏳ Planned | OAuth-only tests (validate it works!) |
| Phase 5 | ⏳ Planned | Documentation + examples |
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

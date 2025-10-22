# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**oauth-mcp-proxy** is an OAuth 2.1 authentication library for Go MCP servers. It provides server-side OAuth integration with minimal code (3-line integration via `WithOAuth()`), supporting multiple providers (HMAC, Okta, Google, Azure AD).

**Version**: v2.0.0 (Supports both `mark3labs/mcp-go` and official `modelcontextprotocol/go-sdk`)

## Build Commands

```bash
# Run tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run linters (same as CI - checks go.mod tidy + golangci-lint)
make lint

# Format code
make fmt

# Clean build artifacts and caches
make clean

# Install/download dependencies
make install

# Check for security vulnerabilities
make vuln
```

## Architecture

### Package Structure (v2.0.0)

```
oauth-mcp-proxy/
├── [core package - SDK-agnostic]
│   ├── oauth.go         - Server type, NewServer, ValidateTokenCached
│   ├── config.go        - Configuration validation and provider setup
│   ├── cache.go         - Token cache with 5-minute TTL
│   ├── context.go       - Context utilities (WithOAuthToken, GetUserFromContext, etc.)
│   ├── handlers.go      - OAuth HTTP endpoints (/.well-known/*, /oauth/*)
│   ├── middleware.go    - CreateHTTPContextFunc for token extraction
│   ├── logger.go        - Logger interface
│   ├── metadata.go      - OAuth metadata structures
│   └── provider/        - Token validators (HMAC, OIDC)
│
├── mark3labs/          - Adapter for mark3labs/mcp-go SDK
│   ├── oauth.go        - WithOAuth → ServerOption
│   └── middleware.go   - Middleware for mark3labs types
│
└── mcp/                - Adapter for official modelcontextprotocol/go-sdk
    └── oauth.go        - WithOAuth → http.Handler
```

### Core Components

**Core Package** (SDK-agnostic):
1. **oauth.go** - `Server` type, `NewServer()`, `ValidateTokenCached()` (used by adapters)
2. **config.go** - Configuration validation and provider setup
3. **cache.go** - Token caching logic (`TokenCache`, `CachedToken`)
4. **context.go** - Context utilities (`WithOAuthToken`, `GetOAuthToken`, `WithUser`, `GetUserFromContext`)
5. **handlers.go** - OAuth HTTP endpoints
6. **provider/provider.go** - Token validators (HMACValidator, OIDCValidator)

**Adapters** (SDK-specific):
- **mark3labs/** - Middleware adapter for `mark3labs/mcp-go`
- **mcp/** - HTTP handler wrapper for official SDK

### Key Design Patterns

- **OpenTelemetry Pattern**: Core logic is SDK-agnostic; adapters provide SDK-specific integration
- **Instance-scoped**: Each `Server` instance has its own token cache and validator (no globals)
- **Provider abstraction**: `TokenValidator` interface supports multiple OAuth providers
- **Caching strategy**: Tokens cached for 5 minutes using SHA-256 hash as key
- **Context propagation**: OAuth token extracted from HTTP header → stored in context → validated → user added to context

### Integration Flow

**mark3labs SDK:**
```text
1. HTTP request with "Authorization: Bearer <token>" header
2. CreateHTTPContextFunc() extracts token → adds to context via WithOAuthToken()
3. mark3labs middleware validates token:
   - Calls Server.ValidateTokenCached() (checks cache first)
   - If not cached, validates via provider (HMAC or OIDC)
   - Caches result (5-minute TTL)
4. Adds authenticated User to context via WithUser()
5. Tool handler accesses user via GetUserFromContext(ctx)
```

**Official SDK:**
```text
1. HTTP request with "Authorization: Bearer <token>" header
2. mcp adapter's HTTP handler intercepts request
3. Validates token via Server.ValidateTokenCached():
   - Checks cache first (5-minute TTL)
   - If not cached, validates via provider
   - Caches result
4. Adds token and user to context (WithOAuthToken, WithUser)
5. Passes request to official SDK's StreamableHTTPHandler
6. Tool handler accesses user via GetUserFromContext(ctx)
```

### Provider System

- **HMAC**: Validates JWT tokens with shared secret (testing/dev)
- **OIDC**: Validates tokens via JWKS/OIDC discovery (Okta/Google/Azure)
- All validation happens in `provider/provider.go`
- Validators implement `TokenValidator` interface

## Testing

The codebase has extensive test coverage across multiple scenarios:

- **api_test.go** - Core API functionality tests
- **integration_test.go** - End-to-end integration tests
- **security_test.go** - Security validation tests
- **attack_scenarios_test.go** - Security attack scenario tests
- **middleware_compatibility_test.go** - Middleware compatibility tests
- **provider/provider_test.go** - Token validator tests

Run single test:

```bash
go test -v -run TestName ./...
```


## Important Notes

1. **User Context**: Always use `GetUserFromContext(ctx)` in tool handlers to access authenticated user
2. **Token Caching**: Tokens cached for 5 minutes - design for this TTL in testing
3. **Logging**: Config.Logger is optional. If nil, uses default logger (log.Printf with level prefixes)
4. **Modes**: Library supports "native" (token validation only) and "proxy" (OAuth flow proxy) modes
5. **Security**: All redirect URIs validated, state parameters HMAC-signed, tokens never logged (only hash previews)
6. **v2.0.0 Breaking Change**: `WithOAuth()` moved to adapter packages (`mark3labs.WithOAuth()` or `mcp.WithOAuth()`). See `MIGRATION-V2.md`.

## Using the Library

### With mark3labs/mcp-go
```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{...})
mcpServer := server.NewMCPServer("name", "1.0.0", oauthOption)
```

### With Official SDK
```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

mcpServer := mcp.NewServer(&mcp.Implementation{...}, nil)
_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{...}, mcpServer)
http.ListenAndServe(":8080", handler)
```

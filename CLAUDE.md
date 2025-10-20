# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**oauth-mcp-proxy** is an OAuth 2.1 authentication library for Go MCP servers. It provides server-side OAuth integration with minimal code (3-line integration via `WithOAuth()`), supporting multiple providers (HMAC, Okta, Google, Azure AD).

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

### Core Components

1. **oauth.go** - Main entry point, provides `WithOAuth()` function that creates OAuth server and returns MCP server option
2. **config.go** - Configuration validation and provider setup
3. **middleware.go** - Token validation middleware with 5-minute caching
4. **handlers.go** - OAuth HTTP endpoints (/.well-known/*, /oauth/*)
5. **provider/provider.go** - Token validators (HMACValidator, OIDCValidator)

### Key Design Patterns

- **Instance-scoped**: Each `Server` instance has its own token cache and validator (no globals)
- **Provider abstraction**: `TokenValidator` interface supports multiple OAuth providers
- **Caching strategy**: Tokens cached for 5 minutes using SHA-256 hash as key
- **Context propagation**: OAuth token extracted from HTTP header → stored in context → validated by middleware → user added to context

### Integration Flow

```
1. HTTP request with "Authorization: Bearer <token>" header
2. CreateHTTPContextFunc() extracts token → adds to context via WithOAuthToken()
3. OAuth middleware (Server.Middleware()) validates token:
   - Checks cache first (5-minute TTL)
   - If not cached, validates via provider (HMAC or OIDC)
   - Caches result
4. Adds authenticated User to context via userContextKey
5. Tool handler accesses user via GetUserFromContext(ctx)
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

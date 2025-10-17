# OAuth MCP Proxy - Extraction Plan

> **⚠️ IMPORTANT:** This plan has been reviewed and updated. See `oauth-extraction-review.md` for critical issues and confirmed decisions.

## Executive Summary

This document outlines the plan to extract the OAuth authentication implementation from `mcp-trino` into a standalone, reusable library called `oauth-mcp-proxy`. This library will enable any Go-based MCP server to add OAuth 2.1 authentication with minimal integration effort.

**Repository:** `github.com/tuannvm/oauth-mcp-proxy` ✅ (Confirmed)
**Go Module:** `github.com/tuannvm/oauth-mcp-proxy`

**Target:** Reusable OAuth 2.1 proxy/validator for Go MCP servers

**Status:** 18 critical issues identified → Phase 0 required before extraction

## Current State Analysis

### Existing Implementation

The current `internal/oauth/` package in mcp-trino provides:

**Core Components:**
- `config.go` - OAuth setup and validator factory (154 lines)
- `middleware.go` - Token validation, caching, MCP context propagation (240 lines)
- `handlers.go` - OAuth 2.0 flows with PKCE support (804 lines)
- `metadata.go` - RFC 8414/9728 metadata endpoints (325 lines)
- `providers.go` - Token validators: HMAC-SHA256, OIDC/JWKS (269 lines)

**Test Coverage:**
- `providers_test.go` - Provider validation tests
- `attack_scenarios_test.go` - Security attack scenarios
- `fixed_redirect_test.go` - Redirect URI validation
- `security_scenarios_test.go` - Security edge cases
- `security_test.go` - Core security tests
- `state_test.go` - State parameter handling
- `metadata_test.go` - Metadata endpoint tests

**Total Implementation:** ~1,800 lines of production code + comprehensive tests

### Dependencies

**External Dependencies (to keep):**
```go
github.com/coreos/go-oidc/v3 v3.15.0          // OIDC discovery/verification
github.com/golang-jwt/jwt/v5 v5.3.0           // JWT parsing
golang.org/x/oauth2 v0.30.0                   // OAuth2 flows
```

**Internal Dependencies (to remove/abstract):**
```go
github.com/tuannvm/mcp-trino/internal/config  // Trino-specific config
github.com/mark3labs/mcp-go/mcp               // MCP protocol types
github.com/mark3labs/mcp-go/server            // MCP server types
```

## Target Architecture

### Package Structure

```
github.com/tuannvm/oauth-mcp-proxy
├── oauth.go              // Core Server, Config, interfaces
├── auth.go               // Authentication logic
├── cache.go              // Token cache
├── errors.go             // Error types
├── logger.go             // Logger interface + default
├── context.go            // Context helpers
├── provider.go           // TokenValidator interface
├── provider_hmac.go      // HMAC validator
├── provider_oidc.go      // OIDC validator
├── handler_authorize.go  // OAuth handlers (proxy mode)
├── handler_callback.go
├── handler_token.go
├── handler_metadata.go
├── handler_validate.go   // Token validation API (standalone mode)
├── adapter/
│   └── mcp/             // MCP-specific adapter
│       ├── adapter.go
│       └── metadata.go  // MCP metadata endpoints
├── client/              // Client library for standalone mode
│   ├── client.go        // HTTP client for /validate endpoint
│   ├── batch.go         // Batch validation support
│   └── circuit.go       // Circuit breaker integration
├── cmd/
│   └── oauth-mcp-proxy/ // Standalone service binary
│       └── main.go
├── internal/            // Private utilities
│   ├── pkce.go
│   ├── state.go
│   └── redirect.go
├── examples/
│   ├── embedded/        // MCP server with embedded mcpoauth
│   ├── standalone/      // Standalone mcpoauth service
│   ├── client/          // MCP server using standalone mcpoauth
│   └── custom-provider/
└── testutil/            // Testing helpers
    └── testutil.go
```

> **Note:** Supports both **embedded** (library) and **standalone** (service) deployment modes with dedicated client library.

### Deployment Modes

#### Mode 1: Embedded Library (In-Process)
```
┌─────────────────────────────────────┐
│   MCP Server Process                │
│                                     │
│  ┌──────────┐      ┌─────────────┐ │
│  │   MCP    │◄────►│  mcpoauth   │ │
│  │  Tools   │      │  (library)  │ │
│  └──────────┘      └─────────────┘ │
│                           ▲         │
└───────────────────────────┼─────────┘
                            │
                            ▼
                     OAuth Provider
                     (Okta/Google)
```

**Use when:**
- Simple deployment (single service)
- Full control over configuration
- Minimal latency requirements

#### Mode 2: Standalone Service (Out-of-Process)
```
┌────────────┐      ┌────────────────┐      ┌──────────────┐
│  MCP       │      │   mcpoauth     │      │    OAuth     │
│  Server A  │──────►   Service      │◄─────►   Provider   │
└────────────┘      │  (standalone)  │      │  (Okta/etc)  │
                    └────────────────┘      └──────────────┘
┌────────────┐             ▲
│  MCP       │             │
│  Server B  │─────────────┘
└────────────┘

Multiple MCP servers share one OAuth service
```

**Use when:**
- Multiple MCP servers need OAuth
- Centralized OAuth configuration
- Service mesh / microservices architecture
- Easier OAuth config updates

### Core Interfaces

```go
// Config is the main configuration for OAuth proxy
type Config struct {
    // Mode: "native" (direct OAuth) or "proxy" (server-mediated)
    Mode string

    // Provider: "hmac", "okta", "google", "azure", or custom
    Provider string

    // Server configuration
    ServerURL string // Full URL of the MCP server

    // OIDC configuration
    Issuer       string
    Audience     string
    ClientID     string
    ClientSecret string

    // Security
    JWTSecret        []byte // For HMAC provider and state signing
    RedirectURIs     string // Single URI or comma-separated list
    EnforceHTTPS     bool   // Require HTTPS for OAuth endpoints

    // Optional
    Scopes []string
}

// TokenValidator validates OAuth tokens
type TokenValidator interface {
    Initialize(cfg *Config) error
    ValidateToken(token string) (*User, error)
}

// User represents an authenticated user
type User struct {
    Subject  string
    Username string
    Email    string
}

// Server provides OAuth proxy functionality
type Server struct {
    config    *Config
    validator TokenValidator
    handlers  map[string]http.Handler
}

// NewServer creates a new OAuth proxy server
func NewServer(cfg *Config) (*Server, error)

// HTTPMiddleware returns standard http.Handler middleware
func (s *Server) HTTPMiddleware(next http.Handler) http.Handler

// MCPMiddleware returns MCP ToolHandlerFunc middleware
// Requires: github.com/mark3labs/mcp-go/server
func (s *Server) MCPMiddleware(next server.ToolHandlerFunc) server.ToolHandlerFunc

// RegisterHandlers registers OAuth endpoints on a mux
func (s *Server) RegisterHandlers(mux *http.ServeMux)

// ValidateToken is the API endpoint for standalone mode
// POST /validate with Bearer token returns User info
func (s *Server) HandleValidateToken(w http.ResponseWriter, r *http.Request)
```

## Extraction Strategy

### Phase 1: Repository Setup (Week 1)

**Objectives:**
- Create new repository structure
- Set up CI/CD pipeline
- Copy and adapt existing code

**Tasks:**
1. Create `github.com/tuannvm/mcpoauth` repository
2. Initialize Go module: `go mod init github.com/tuannvm/mcpoauth`
3. Set up GitHub Actions (lint, test, security scanning)
4. Copy test files and security scenarios
5. Set up examples directory

**Deliverables:**
- Working repository with basic structure
- CI/CD pipeline functional
- License file (MIT recommended)

### Phase 2: Core Extraction (Week 1-2)

**Objectives:**
- Extract and refactor core OAuth logic
- Remove Trino-specific dependencies
- Implement generic configuration

**Tasks:**

#### 2.1 Configuration Layer
```go
// Before (mcp-trino specific)
func SetupOAuth(cfg *config.TrinoConfig) (TokenValidator, error)

// After (generic oauth-mcp-proxy)
func NewServer(cfg *Config) (*Server, error)
func (s *Server) Validator() TokenValidator
```

#### 2.2 Provider Layer
- Move `providers.go` → `providers/`
- Split into `hmac.go` and `oidc.go`
- Extract `cache.go` for token caching
- Make `TokenValidator` interface public

#### 2.3 Middleware Layer
- Extract HTTP middleware (no MCP dependencies)
- Create MCP adapter (optional dependency)
- Separate context helpers

#### 2.4 Handlers Layer
- Move handlers to `handlers/` package
- Remove MCP-specific metadata endpoints
- Keep standard OAuth 2.1 endpoints
- **Add `/validate` endpoint for standalone mode**

#### 2.5 Standalone Service Binary
- Create `cmd/mcpoauth/main.go`
- HTTP server with all OAuth endpoints
- Configuration from env/file
- Docker image support
- Health checks, metrics

**Deliverables:**
- Generic `Config` struct
- Public `TokenValidator` interface
- Standalone provider implementations
- HTTP middleware without MCP dependencies
- **Working standalone binary**

### Phase 3: MCP Integration Adapter (Week 2)

**Objectives:**
- Create optional MCP adapter package
- Maintain backward compatibility
- Provide seamless integration

**Tasks:**

#### 3.1 MCP Adapter Package
```go
// middleware/mcp/adapter.go
package mcp

import (
    "github.com/mark3labs/mcp-go/server"
    "github.com/tuannvm/oauth-mcp-proxy"
)

// Adapter wraps mcpoauth for MCP servers
type Adapter struct {
    server *oauth.Server
}

// NewAdapter creates MCP-compatible middleware
func NewAdapter(cfg *oauth.Config) (*Adapter, error)

// Middleware returns MCP ToolHandlerFunc middleware
func (a *Adapter) Middleware() func(server.ToolHandlerFunc) server.ToolHandlerFunc

// RegisterMetadata registers MCP-specific metadata endpoints
func (a *Adapter) RegisterMetadata(mux *http.ServeMux)
```

#### 3.2 MCP Metadata Endpoints
- Move MCP-specific metadata to adapter
- Keep RFC-compliant endpoints in core

**Deliverables:**
- `middleware/mcp/` package
- MCP server integration guide
- Example MCP server

### Phase 4: Documentation & Examples (Week 2-3)

**Objectives:**
- Comprehensive documentation
- Multiple integration examples
- Migration guide for mcp-trino

**Tasks:**

#### 4.1 Core Documentation
- README.md with quickstart
- Architecture documentation
- Security best practices
- Provider comparison guide

#### 4.2 Examples
```
examples/
├── embedded/           # MCP server with embedded mcpoauth
├── standalone/         # Standalone mcpoauth service (binary)
├── client/             # MCP server using standalone mcpoauth
├── custom-provider/    # Custom provider implementation
└── migration/          # Migrating from mcp-trino
```

**Example Details:**
- **embedded/**: Complete MCP server with mcpoauth as library (in-process)
- **standalone/**: Run mcpoauth as independent OAuth service
- **client/**: MCP server calling standalone mcpoauth for validation
- **custom-provider/**: Implementing TokenValidator for custom auth
- **migration/**: Step-by-step guide from mcp-trino internal/oauth

#### 4.3 API Documentation
- GoDoc comments for all public APIs
- Configuration examples
- Provider setup guides

**Deliverables:**
- Complete README.md
- 3+ working examples
- Migration guide
- API documentation

### Phase 5: Migration & Testing (Week 3-4)

**Objectives:**
- Migrate mcp-trino to use oauth-mcp-proxy
- Validate all functionality
- Performance benchmarking

**Tasks:**

#### 5.1 mcp-trino Migration
```go
// Before
import "github.com/tuannvm/mcp-trino/internal/oauth"

// After
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/adapter/mcp"
)

// Setup
oauthServer, err := oauth.NewServer(&oauth.Config{
    Mode:         cfg.OAuthMode,
    Provider:     cfg.OAuthProvider,
    ServerURL:    cfg.MCPURL,
    Issuer:       cfg.OIDCIssuer,
    Audience:     cfg.OIDCAudience,
    ClientID:     cfg.OIDCClientID,
    ClientSecret: cfg.OIDCClientSecret,
    JWTSecret:    []byte(cfg.JWTSecret),
    RedirectURIs: cfg.OAuthRedirectURIs,
    EnforceHTTPS: true,
})

// MCP middleware
adapter, err := mcp.NewAdapter(oauthServer)
mcpServer.UseMiddleware(adapter.Middleware())

// HTTP handlers
oauthServer.RegisterHandlers(mux)
adapter.RegisterMetadata(mux)
```

#### 5.2 Testing
- Run all existing security tests
- Add integration tests
- Performance benchmarks
- Load testing (token cache effectiveness)

#### 5.3 Validation
- Test against Okta, Google, Azure
- Verify PKCE flows
- Validate metadata endpoints
- Test with Claude Code, mcp-remote

**Deliverables:**
- mcp-trino using mcpoauth
- All tests passing
- Performance benchmarks
- Validation report

### Phase 6: Release & Documentation (Week 4)

**Objectives:**
- Public release
- Community announcement
- Documentation site

**Tasks:**

#### 6.1 Release Preparation
- Tag v0.1.0 (initial release)
- Generate CHANGELOG.md
- Security audit review
- Dependency audit (govulncheck)

#### 6.2 Community
- Announcement post
- Add to MCP ecosystem list
- Create GitHub discussions
- Add security policy

**Deliverables:**
- v0.1.0 release
- Public announcement
- Community channels established

## Integration Guide (Preview)

### Mode 1: Embedded Library (Recommended for Single MCP Server)

```go
package main

import (
    "net/http"
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpadapter "github.com/tuannvm/oauth-mcp-proxy/adapter/mcp"
    mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
    // 1. Configure OAuth
    oauthCfg := &oauth.Config{
        Mode:         "proxy",
        Provider:     "okta",
        ServerURL:    "https://mcp.example.com",
        Issuer:       "https://dev-12345.okta.com",
        Audience:     "api://mcp-server",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        RedirectURIs: "https://mcp.example.com/oauth/callback",
        EnforceHTTPS: true,
    }

    // 2. Create OAuth server
    oauthServer, err := oauth.NewServer(oauthCfg)
    if err != nil {
        log.Fatal(err)
    }

    // 3. Create MCP server
    mcpServer := mcpserver.NewMCPServer("My MCP Server", "1.0.0")

    // 4. Apply OAuth middleware
    adapter, _ := mcpadapter.NewAdapter(oauthServer)
    mcpServer.UseMiddleware(adapter.Middleware())

    // 5. Register handlers
    mux := http.NewServeMux()
    oauthServer.RegisterHandlers(mux)
    adapter.RegisterMetadata(mux)

    // 6. Add MCP endpoint
    streamableServer := mcpserver.NewStreamableHTTPServer(
        mcpServer,
        mcpserver.WithHTTPContextFunc(oauthServer.HTTPContextFunc()),
    )
    mux.Handle("/mcp", streamableServer)

    // 7. Start server
    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

### Mode 2: Standalone Service (Recommended for Multiple MCP Servers)

#### Step 1: Run Standalone mcpoauth Service

```bash
# Using binary
./oauth-mcp-proxy --config config.yaml

# Using Docker
docker run -p 9000:9000 \
  -e OAUTH_PROVIDER=okta \
  -e OIDC_ISSUER=https://dev-12345.okta.com \
  -e OIDC_AUDIENCE=api://mcp-server \
  -e OIDC_CLIENT_ID=your-client-id \
  -e OIDC_CLIENT_SECRET=your-secret \
  ghcr.io/tuannvm/oauth-mcp-proxy:latest
```

**Standalone service exposes:**
- `POST /validate` - Validate token and return user info
- `GET /.well-known/oauth-authorization-server` - OAuth metadata
- `/oauth/authorize`, `/oauth/callback`, `/oauth/token` - OAuth flows (proxy mode)

#### Step 2: MCP Server Uses Standalone Service

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    mcpserver "github.com/mark3labs/mcp-go/server"
)

// Call standalone oauth-mcp-proxy service for validation
func validateWithStandalone(token string) (*User, error) {
    req, _ := http.NewRequest("POST", "http://oauth-mcp-proxy:9000/validate", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("validation failed: %d", resp.StatusCode)
    }

    var user User
    json.NewDecoder(resp.Body).Decode(&user)
    return &user, nil
}

// Custom middleware calling standalone service
func oauthMiddleware(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        token := getTokenFromContext(ctx)
        user, err := validateWithStandalone(token)
        if err != nil {
            return nil, fmt.Errorf("authentication failed: %w", err)
        }

        ctx = context.WithValue(ctx, "user", user)
        return next(ctx, req)
    }
}

func main() {
    mcpServer := mcpserver.NewMCPServer("My MCP Server", "1.0.0")
    mcpServer.UseMiddleware(oauthMiddleware)

    // ... rest of MCP server setup
}
```

**Benefits of Standalone Mode:**
- ✅ Single OAuth configuration for multiple MCP servers
- ✅ Update OAuth config without redeploying MCP servers
- ✅ Centralized audit logs
- ✅ Easier to scale OAuth capacity independently

### Configuration Examples

#### Native Mode (Direct OAuth)
```bash
export OAUTH_MODE=native
export OAUTH_PROVIDER=okta
export OIDC_ISSUER=https://dev-12345.okta.com
export OIDC_AUDIENCE=api://my-mcp-server
```

#### Proxy Mode (OAuth Proxy)
```bash
export OAUTH_MODE=proxy
export OAUTH_PROVIDER=okta
export OIDC_ISSUER=https://dev-12345.okta.com
export OIDC_AUDIENCE=api://my-mcp-server
export OIDC_CLIENT_ID=0oa...xyz
export OIDC_CLIENT_SECRET=secret...
export OAUTH_REDIRECT_URI=https://mcp.example.com/oauth/callback
export JWT_SECRET=your-32-byte-secret
```

#### HMAC Mode (Development)
```bash
export OAUTH_MODE=native
export OAUTH_PROVIDER=hmac
export JWT_SECRET=your-32-byte-secret
export OIDC_AUDIENCE=api://my-mcp-server
```

## Key Features to Preserve

### Security Features
- ✅ HMAC-SHA256 state signing
- ✅ Localhost-only redirects in fixed mode
- ✅ PKCE support (OAuth 2.1 standard)
- ✅ Token caching (5min TTL)
- ✅ HTTPS enforcement
- ✅ Defense-in-depth validation

### OAuth 2.1 Compliance
- ✅ RFC 8414 - Authorization Server Metadata
- ✅ RFC 9728 - Protected Resource Metadata
- ✅ RFC 7636 - PKCE
- ✅ OIDC Discovery

### Provider Support
- ✅ HMAC-SHA256 (development/testing)
- ✅ Okta (OIDC/JWKS)
- ✅ Google (OIDC/JWKS)
- ✅ Azure AD (OIDC/JWKS)
- ✅ Custom providers (extensible)

### Operational Modes
- ✅ Native mode (direct OAuth)
- ✅ Proxy mode (OAuth proxy)
- ✅ Fixed redirect URI (single URI)
- ✅ Allowlist mode (multiple URIs)

## Success Criteria

### Functional Requirements
- ✅ Generic configuration (no MCP-specific types in core)
- ✅ Standard HTTP middleware (works with any Go HTTP server)
- ✅ MCP adapter (optional, seamless integration)
- ✅ All existing tests pass
- ✅ 3+ working examples
- ✅ Migration guide for mcp-trino

### Non-Functional Requirements
- ✅ Zero breaking changes for mcp-trino
- ✅ Performance: <5ms overhead per request (cached tokens)
- ✅ Security: All existing security tests pass
- ✅ Documentation: 100% public API documented
- ✅ Test coverage: >80% for core packages

### Community Requirements
- ✅ Open source (MIT license)
- ✅ Comprehensive README
- ✅ Contributing guidelines
- ✅ Security policy
- ✅ GitHub Discussions enabled

## Timeline (Revised)

> **Updated based on Gemini 2.0 reviews:** Extended for standalone mode + client library

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 0: Critical Fixes** | **7 days** | **Fix 18 critical issues (8 original + 10 new)** |
| Phase 1: Repository Setup | 3 days | Working repo with CI/CD |
| Phase 2: Core Extraction | 7 days | Generic config, providers, auth, standalone binary |
| Phase 3: MCP Adapter | 4 days | Clean adapter pattern |
| Phase 4: Testing | 5 days | Unit, integration, security tests |
| Phase 5: Documentation | 5 days | Complete docs, examples (both modes) |
| Phase 6: Migration | 5 days | mcp-trino migration, validation |
| Phase 7: Client Library | 3 days | Build client lib for standalone mode |
| Phase 8: Security Audit | 4 days | Security review (standalone mode focus) |
| Phase 9: Release | 3 days | v0.1.0 release, announcement |
| **Total** | **~7 weeks (48 days)** | Production-ready library + service |

See `oauth-extraction-review.md` for:
- Detailed Phase 0 task breakdown (18 critical issues)
- Standalone service security considerations
- Client library API design

## Risk Assessment

### Technical Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Breaking changes in MCP dependencies | High | Low | Version pinning, adapter pattern |
| Performance regression | Medium | Low | Benchmarking, token caching |
| Security vulnerabilities | High | Low | Security tests, audit |
| Provider compatibility | Medium | Medium | Integration tests with real providers |

### Operational Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Migration complexity | Medium | Medium | Comprehensive migration guide |
| Adoption challenges | Low | Medium | Examples, documentation |
| Maintenance burden | Medium | Low | Good test coverage, CI/CD |

## Next Steps

1. **Review & Approval** - Review this plan with stakeholders
2. **Repository Creation** - Create `mcpoauth` repository
3. **Phase 1 Kickoff** - Begin repository setup and CI/CD
4. **Weekly Updates** - Progress updates and blocker resolution

## References

- [RFC 8414 - OAuth 2.0 Authorization Server Metadata](https://datatracker.ietf.org/doc/html/rfc8414)
- [RFC 9728 - OAuth 2.0 Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728)
- [RFC 7636 - PKCE](https://datatracker.ietf.org/doc/html/rfc7636)
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-10)
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Author:** OAuth MCP Proxy Team

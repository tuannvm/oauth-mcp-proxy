# OAuth MCP Proxy - Extraction Plan

> **Canonical Reference:** This is the single source of truth for the extraction project.

## Executive Summary

**Project:** Extract OAuth authentication from `mcp-trino` into standalone `oauth-mcp-proxy` library
**Repository:** `github.com/tuannvm/oauth-mcp-proxy`
**Source:** `../mcp-trino/internal/oauth/` (~3000 LOC)

**Strategy:** ✅ Extract then Fix (confirmed)
- Copy code to this repo → Fix here → No changes to mcp-trino during development
- No releases until extraction complete
- Build OAuth-only tests (remove all Trino dependencies)

---

## Package Structure (Confirmed)

```
oauth-mcp-proxy/
├── oauth.go              // Public API: Server, Config, NewServer()
├── cache.go              // Token cache (instance-scoped, internal access)
├── errors.go             // Sentinel error types (public)
├── logger.go             // Logger interface + default impl (public)
├── context.go            // Context helpers (public)
├── metrics.go            // Metrics interface + default impl (public)
├── provider/
│   ├── provider.go       // TokenValidator interface (public)
│   ├── hmac.go           // HMAC validator implementation
│   ├── oidc.go           // OIDC/JWKS validator implementation
│   └── provider_test.go  // Provider tests
├── handler/
│   ├── authorize.go      // GET /oauth/authorize
│   ├── callback.go       // GET /oauth/callback
│   ├── token.go          // POST /oauth/token
│   ├── metadata.go       // GET /.well-known/*
│   ├── validate.go       // POST /validate (standalone mode)
│   └── handler_test.go   // Handler tests
├── adapter/
│   └── mcp/             // MCP adapter (optional import)
│       ├── adapter.go    // MCP middleware wrapper
│       └── metadata.go   // MCP-specific metadata
├── cmd/
│   └── oauth-mcp-proxy/ // Standalone service binary
│       └── main.go
├── internal/            // Private utilities
│   ├── pkce.go          // PKCE code generation
│   ├── state.go         // State parameter handling
│   └── redirect.go      // Redirect URI validation
├── examples/
│   ├── embedded/        // MCP server with embedded library
│   │   └── main.go
│   ├── standalone/      // Run oauth-mcp-proxy as service
│   │   ├── main.go
│   │   └── config.yaml
│   └── client/          // MCP server calling standalone service
│       └── main.go
└── testutil/            // Testing helpers for library consumers
    └── testutil.go      // Mock providers, test servers
```

---

## How Other Projects Import This Library

### Basic Import (Embedded Mode)

```go
package main

import (
    "net/http"

    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/provider"
    "github.com/tuannvm/oauth-mcp-proxy/adapter/mcp"
)

func main() {
    // 1. Create OAuth config
    cfg := &oauth.Config{
        Mode:         oauth.ModeProxy,
        Provider:     "okta",
        ServerURL:    "https://mcp.example.com",
        Issuer:       "https://company.okta.com",
        Audience:     "api://mcp-server",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        RedirectURIs: "https://mcp.example.com/callback",
    }

    // 2. Create OAuth server
    oauthServer, err := oauth.NewServer(cfg)
    if err != nil {
        panic(err)
    }
    defer oauthServer.Stop()

    // 3. Use with MCP server
    mcpAdapter := mcp.NewAdapter(oauthServer)

    // Apply to MCP server middleware
    mcpServer.UseMiddleware(mcpAdapter.Middleware())

    // Register OAuth HTTP handlers
    mux := http.NewServeMux()
    oauthServer.RegisterHandlers(mux)

    http.ListenAndServe(":8080", mux)
}
```

### Custom Provider

```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/provider"
)

// Implement custom provider
type MyProvider struct {}

func (p *MyProvider) Initialize(cfg *oauth.Config) error {
    return nil
}

func (p *MyProvider) ValidateToken(token string) (*oauth.User, error) {
    // Custom validation logic
    return &oauth.User{
        Subject:  "user-123",
        Email:    "user@example.com",
        Username: "user",
    }, nil
}

// Register and use
func main() {
    provider.Register("custom", func() provider.TokenValidator {
        return &MyProvider{}
    })

    cfg := &oauth.Config{
        Provider: "custom",
        // ...
    }

    oauthServer, _ := oauth.NewServer(cfg)
    // ...
}
```

### Standalone Mode (Client)

```go
import (
    "context"
    "net/http"

    oauth "github.com/tuannvm/oauth-mcp-proxy"
)

func main() {
    // Call standalone oauth-mcp-proxy service
    req, _ := http.NewRequest("POST", "http://oauth-mcp-proxy:9000/validate", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("X-API-Key", "your-api-key")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        // Handle auth failure
        return
    }

    var user oauth.User
    json.NewDecoder(resp.Body).Decode(&user)

    // Use authenticated user
    fmt.Printf("Authenticated: %s\n", user.Email)
}
```

### Testing with Testutil

```go
import (
    "testing"

    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/testutil"
)

func TestMyMCPServer(t *testing.T) {
    // Use mock OAuth server for testing
    mockOAuth := testutil.NewMockOAuthServer(t)
    defer mockOAuth.Close()

    // Configure your MCP server to use mock
    cfg := &oauth.Config{
        Mode:     oauth.ModeNative,
        Provider: "mock",
        Issuer:   mockOAuth.URL,
    }

    // Test your server
    // ...
}
```

### Import Paths Summary

| Import Path | Usage |
|-------------|-------|
| `github.com/tuannvm/oauth-mcp-proxy` | Main package (Server, Config) |
| `github.com/tuannvm/oauth-mcp-proxy/provider` | Custom providers (TokenValidator) |
| `github.com/tuannvm/oauth-mcp-proxy/adapter/mcp` | MCP integration |
| `github.com/tuannvm/oauth-mcp-proxy/testutil` | Testing helpers |

---

## Implementation Phases

### Phase 0: Repository Setup

**Goal:** Initialize repo, copy source code

**Tasks:**
- [ ] Initialize go.mod (`go mod init github.com/tuannvm/oauth-mcp-proxy`)
- [ ] Copy all `.go` files from `../mcp-trino/internal/oauth/`
- [ ] Set up .gitignore, LICENSE (MIT)
- [ ] Update imports from `internal/oauth` → root package
- [ ] Commit: "Initial extraction from mcp-trino"

**Success Criteria:**
- go.mod initialized with correct dependencies
- All source code copied
- Compiles (even if tests fail)

---

### Phase 1: Architecture Fixes + Interface Design

**Goal:** Fix 11 P0 issues + design MCP adapter interface

**Critical Change (from Gemini feedback):** Design MCP adapter interface FIRST, then build core against it.

#### P0 Architecture Issues

| # | Issue | Impact |
|---|-------|--------|
| 1 | Global Token Cache → Instance-scoped | High |
| 2 | Global Middleware Registry → Remove | High |
| 3 | Private Context Keys → Public accessors | Medium |
| 4 | Hardcoded Logging → Logger interface | Medium |
| 5 | Configuration Validation → Validate() method | High |
| 6 | Error Handling → Sentinel errors | Medium |
| 7 | MCP Adapter Coupling → Separate adapter | High |
| 8 | Missing Graceful Shutdown → Start/Stop | Medium |
| 10 | External Call Timeouts → Configurable | Medium |
| 13 | Context Cancellation → Audit paths | Low |
| 15 | Standalone Security → API key auth | High |
| **NEW** | **Basic Metrics for Standalone** | **High** |

#### New P0 Issue: Basic Metrics (from Gemini feedback)

**Problem:** Standalone service needs operational visibility

**Solution:**
```go
// metrics.go
type Metrics interface {
    RecordValidation(duration time.Duration, success bool, provider string)
    RecordCacheHit(hit bool)
}

// Default implementation
type defaultMetrics struct {
    validationTotal   map[string]int64  // provider -> count
    validationErrors  map[string]int64
    cacheHits         int64
    cacheMisses       int64
}

// Expose metrics on /metrics endpoint (Prometheus format)
```

**Tasks:**
- [ ] Design core interfaces (TokenValidator, Logger, Config, Metrics)
- [ ] **Design MCP adapter interface** (define API contract for adapter/mcp/)
- [ ] Fix Issue #1: Move token cache to Server struct
- [ ] Fix Issue #2: Remove global middleware registry
- [ ] Fix Issue #3: Public context key accessors
- [ ] Fix Issue #4: Add Logger interface
- [ ] Fix Issue #5: Add Config.Validate() method
- [ ] Fix Issue #6: Define sentinel error types
- [ ] Fix Issue #7: Create adapter/mcp/ package structure
- [ ] Fix Issue #8: Add Start()/Stop() methods
- [ ] Fix Issue #10: Add configurable timeouts
- [ ] Fix Issue #13: Audit context cancellation
- [ ] Fix Issue #15: Add API key auth for /validate
- [ ] **NEW: Fix Issue #12: Add basic Metrics interface**

**Success Criteria:**
- No global variables
- All interfaces defined
- MCP adapter interface designed and documented
- Code follows Go best practices
- Basic metrics exposed on /metrics endpoint

---

### Phase 2: Core OAuth Library

**Goal:** Generic OAuth functionality

**Tasks:**
- [ ] Generic Config (remove Trino dependencies)
- [ ] Create provider/ subpackage structure
  - [ ] provider/provider.go (TokenValidator interface)
  - [ ] provider/hmac.go (HMAC implementation)
  - [ ] provider/oidc.go (OIDC implementation)
- [ ] Create handler/ subpackage structure
  - [ ] handler/authorize.go
  - [ ] handler/callback.go
  - [ ] handler/token.go
  - [ ] handler/metadata.go
  - [ ] handler/validate.go
- [ ] Token validation logic
- [ ] Token caching with TTL (cache.go in root)

**Success Criteria:**
- Core library compiles
- No Trino imports in core packages
- All providers implement provider.TokenValidator interface
- Clean package boundaries (root, provider/, handler/)

---

### Phase 3: Dual Mode Support

**Goal:** Embedded + standalone deployment

**Tasks:**
- [ ] Embedded mode: Library usage patterns
- [ ] Standalone mode: HTTP server + /validate endpoint
- [ ] /health endpoint (simple HTTP 200)
- [ ] /metrics endpoint (basic metrics)
- [ ] API key authentication for /validate
- [ ] cmd/oauth-mcp-proxy/main.go (binary entry point)
- [ ] Configuration from env vars

**Success Criteria:**
- Embedded mode: Can import as library
- Standalone mode: Binary runs, /validate works
- /health and /metrics endpoints functional

---

### Phase 4: MCP Adapter Implementation

**Goal:** Build adapter against interface from Phase 1

**Tasks:**
- [ ] Implement adapter/mcp/adapter.go
- [ ] MCP-specific metadata endpoints
- [ ] Middleware wrapper for MCP servers
- [ ] Context propagation
- [ ] Error translation (OAuth errors → MCP errors)

**Success Criteria:**
- Adapter implements interface from Phase 1
- MCP servers can use adapter seamlessly
- No MCP imports in core packages

---

### Phase 5: OAuth-Only Tests

**Goal:** Comprehensive test suite without Trino dependencies

#### Provider Tests
- [ ] HMAC token validation
- [ ] OIDC token validation (mock JWKS)
- [ ] Okta provider integration (sandbox)
- [ ] Google provider integration (sandbox)
- [ ] Azure AD provider integration (sandbox)
- [ ] Token cache behavior
- [ ] Token expiration handling

#### Security Tests
- [ ] State parameter signing/verification
- [ ] PKCE flow
- [ ] Redirect URI validation (fixed mode)
- [ ] Redirect URI validation (allowlist mode)
- [ ] Attack scenario tests (from mcp-trino)
- [ ] Token tampering prevention
- [ ] Open redirect prevention

#### Integration Tests
- [ ] Embedded mode (library usage)
- [ ] Standalone mode (/validate endpoint)
- [ ] API key authentication
- [ ] OAuth proxy flow (full flow)
- [ ] Metrics collection
- [ ] Graceful shutdown

#### Remove from Tests
- ❌ All Trino-specific config tests
- ❌ Tests importing `internal/config`
- ❌ Tests requiring Trino database
- ❌ Tests with Trino query execution

**Success Criteria:**
- >80% test coverage
- All tests pass
- No external dependencies (use mocks)
- Security tests from mcp-trino preserved

---

### Phase 6: Documentation

**Goal:** Complete documentation for library users

**Tasks:**
- [ ] README.md (quickstart, architecture)
- [ ] API documentation (GoDoc comments)
- [ ] examples/embedded/ (MCP server with library)
- [ ] examples/standalone/ (run standalone service)
- [ ] examples/client/ (MCP server calling standalone)
- [ ] Security best practices guide
- [ ] Provider setup guides (Okta, Google, Azure)

**Success Criteria:**
- All public APIs documented
- 3 working examples
- README explains both deployment modes

---

### Phase 7: Migration Support

**Goal:** Enable mcp-trino to use new library

**Tasks:**
- [ ] Create migration guide
- [ ] Document breaking changes
- [ ] Create mcp-trino integration example
- [ ] Test with real mcp-trino instance
- [ ] Validate all 4 providers (HMAC, Okta, Google, Azure)
- [ ] Performance comparison (before/after)

**Success Criteria:**
- Migration guide complete
- mcp-trino can import oauth-mcp-proxy
- All existing functionality preserved
- No regressions

---

## P0 Features (MVP v0.1.0)

### Core Functionality
- ✅ OAuth token validation (HMAC, OIDC/JWKS)
- ✅ Support 4 providers: HMAC, Okta, Google, Azure AD
- ✅ Token caching with TTL (5min default)
- ✅ Dual OAuth modes: native (direct) + proxy (server-mediated)
- ✅ PKCE support (RFC 7636)
- ✅ OAuth 2.1 metadata endpoints (RFC 8414, RFC 9728)

### Deployment Modes
- ✅ Embedded mode (library)
- ✅ Standalone mode (service)
- ✅ /validate endpoint
- ✅ API key authentication
- ✅ /health endpoint
- ✅ **NEW: /metrics endpoint (basic metrics)**

### Architecture
- ✅ Instance-scoped state (no globals)
- ✅ Generic configuration
- ✅ Logger interface
- ✅ Sentinel error types
- ✅ Graceful shutdown
- ✅ Configurable timeouts
- ✅ Context cancellation

---

## Deferred Features (v0.2.0+)

### P1 Features (v0.2.0)
- Client library for standalone mode
- Circuit breaker pattern
- Batch validation endpoint
- Enhanced error metrics (granular)
- Service discovery documentation
- Rate limiting (internal)
- mTLS support for /validate
- IP allowlist for /validate
- Distributed tracing (OpenTelemetry)
- Advanced metrics (Prometheus integration)

### P2 Features (v0.3.0+)
- Multi-tenancy support
- Custom provider plugin system
- Token introspection endpoint
- Advanced cache eviction (LRU)
- HTTP/2 or gRPC support

---

## Success Criteria

### Functional
- ✅ Embedded mode works (MCP server imports library)
- ✅ Standalone mode works (binary runs, /validate endpoint)
- ✅ All 4 providers work (HMAC, Okta, Google, Azure)
- ✅ Token caching reduces load
- ✅ mcp-trino migrates successfully (zero breaking changes)

### Non-Functional
- ✅ <5ms validation with cache hit
- ✅ <50ms validation with cache miss (OIDC)
- ✅ No memory leaks (no global state)
- ✅ No goroutine leaks (graceful shutdown)
- ✅ >80% test coverage

### Documentation
- ✅ Clear README for both modes
- ✅ Working examples (embedded + standalone)
- ✅ Migration guide from mcp-trino
- ✅ Security best practices documented

---

## Risk Management

### Upstream Dependency Volatility

**Risk:** `mcp-go` releases breaking changes in security patch

**Mitigation:**
- Pin exact version in go.mod (`github.com/mark3labs/mcp-go v0.38.0`)
- Monitor mcp-go releases
- Test against new versions in separate branch
- Adapter pattern isolates breaking changes to adapter/mcp/ package

**Strategy:**
```go
// go.mod
require github.com/mark3labs/mcp-go v0.38.0 // pinned

// If v0.39.0 has breaking changes:
// 1. Create adapter/mcp_v039/ package
// 2. Keep adapter/mcp/ for v0.38.0 users
// 3. Document migration path
```

### Documentation Drift

**Mitigation:** This document is the single source of truth
- Archive old planning documents in docs/archive/
- All changes update THIS document only
- Link from README.md to this plan

---

## Key Principles

1. **Isolation:** mcp-trino unchanged during development
2. **Focus:** OAuth functionality only, no Trino coupling
3. **Quality:** Complete each phase before moving forward
4. **Testing:** Comprehensive OAuth test suite from scratch
5. **Metrics:** Basic operational visibility in MVP (standalone mode)
6. **Interface-First:** Design adapter interface before core implementation

---

## Document History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-10-17 | Initial consolidated plan (post-Gemini review) |

**Status:** ✅ Ready to execute Phase 0
**Strategy:** Extract then Fix (confirmed)
**Next Action:** Initialize go.mod and copy source code

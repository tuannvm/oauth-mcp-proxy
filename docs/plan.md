# OAuth MCP Proxy - v0.1.0 Plan (Embedded Mode Only)

> **Canonical Reference:** This is the plan for v0.1.0 - Embedded library mode only

## Executive Summary

**Project:** Extract OAuth authentication from `mcp-trino` into reusable `oauth-mcp-proxy` library
**Repository:** `github.com/tuannvm/oauth-mcp-proxy`
**Source:** `../mcp-trino/internal/oauth/` (~3000 LOC)
**Version:** v0.1.0 - Embedded Mode (Library) Only

**Focus:**
- MCP servers import oauth-mcp-proxy as a library
- Add OAuth authentication to their own tools
- Original use case from mcp-trino

**Strategy:** Extract then Fix
- Copy code to this repo → Fix here → No changes to mcp-trino
- No releases until extraction complete
- Build OAuth-only tests (remove all Trino dependencies)

**Standalone Mode:** Deferred to v0.2.0+ (see [plan-standalone.md](plan-standalone.md))

**Dependencies:** This library requires 4 external dependencies (carried over from mcp-trino)

---

## Dependencies

### Required (Direct)

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/mark3labs/mcp-go` | v0.41.1 | MCP protocol types and server |
| `github.com/coreos/go-oidc/v3` | v3.16.0 | OIDC discovery and JWT verification |
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | HMAC-SHA256 token validation |
| `golang.org/x/oauth2` | v0.32.0 | OAuth 2.0 client flows (proxy mode) |

### Transitive (Indirect)

- `github.com/go-jose/go-jose/v4` - JWKS/JWE handling (via go-oidc)
- `golang.org/x/crypto` - Cryptographic primitives
- `golang.org/x/net` - HTTP/2 support

### To Remove (Trino-specific)

- ❌ `github.com/tuannvm/mcp-trino/internal/config` - Removed in Phase 1

### Note on Dependencies

**All 4 dependencies are necessary:**
- **mcp-go:** Core MCP integration (library is MCP-only)
- **go-oidc:** Industry-standard OIDC library (no good alternatives)
- **jwt:** Standard Go JWT library (minimal, well-maintained)
- **oauth2:** Official Go OAuth2 library (maintained by Go team)

**No optional dependencies** - All required for core functionality

---

## Package Structure (Simplified for MCP-Only)

```
oauth-mcp-proxy/
├── oauth.go              // Server, Config, NewServer(), WithOAuth()
├── middleware.go         // OAuth middleware for MCP (uses Server)
├── token.go              // Token validation logic
├── user.go               // User type
├── handler_authorize.go  // OAuth handlers in ROOT (need Server internals)
├── handler_callback.go
├── handler_token.go
├── handler_metadata.go
├── errors.go             // Sentinel error types
├── logger.go             // Logger interface
├── context.go            // Context helpers
├── provider/
│   ├── provider.go       // TokenValidator interface
│   ├── hmac.go           // HMAC validator
│   ├── oidc.go           // OIDC/JWKS validator
│   └── provider_test.go
├── internal/
│   ├── cache/            // Token cache (instance-scoped, fixed in Phase 1.5)
│   │   └── cache.go
│   ├── pkce.go
│   ├── state.go
│   └── redirect.go
├── examples/
│   └── embedded/
│       └── main.go
└── testutil/
    └── testutil.go
```

**Key Decisions:**
- ✅ Handlers in ROOT (need access to Server internals)
- ✅ Middleware in ROOT (needs Server, creates MCP middleware)
- ✅ Providers in provider/ (self-contained)
- ✅ NO adapter/ (library is MCP-only, no abstraction needed)
- ✅ Cache in internal/cache/ (moved in Phase 1.5, not public API)

---

## Simplest API Design (Option A)

### One Function Call Integration

```go
package main

import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
    // 1. Create your MCP server as usual
    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0")
    mux := http.NewServeMux()

    // 2. Enable OAuth with ONE function call
    oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
        Provider:     "okta",
        Issuer:       "https://company.okta.com",
        Audience:     "api://my-server",
        ClientID:     "client-id",
        ClientSecret: "client-secret",
        ServerURL:    "https://my-server.com",
        RedirectURIs: "https://my-server.com/callback",
    })

    // 3. Create MCP server with OAuth option
    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

    // Done! OAuth is now enabled:
    // ✅ Middleware applied to mcpServer
    // ✅ HTTP handlers registered on mux
    // ✅ Context extraction configured

    // 4. Continue with normal MCP setup
    mux.Handle("/mcp", mcpserver.NewStreamableHTTPServer(mcpServer))
    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

---

## Native vs Proxy Mode

### Mode Comparison

| Aspect | Native Mode | Proxy Mode |
|--------|-------------|------------|
| **Config Required** | Issuer, Audience | + ClientID, ClientSecret, ServerURL, RedirectURIs |
| **Client Setup** | Client configures OAuth | No client OAuth config needed |
| **OAuth Flow** | Client ↔ Provider directly | Client ↔ MCP Server ↔ Provider |
| **HTTP Endpoints** | Return 404 | Fully functional |
| **Metadata** | Points to Provider | Points to MCP Server |
| **Use Case** | OAuth-capable clients (Claude.ai) | Any MCP client |

### Native Mode Example

**When to use:** Client can handle OAuth directly (e.g., Claude.ai, browser-based clients)

```go
import oauth "github.com/tuannvm/oauth-mcp-proxy"

func main() {
    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0")
    mux := http.NewServeMux()

    // Native mode - minimal config
    oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
        Mode:     "native",  // Explicit (or auto-detected if omitted)
        Provider: "okta",
        Issuer:   "https://company.okta.com",
        Audience: "api://my-server",
    })

    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

    // What happens:
    // ✅ Middleware validates Bearer tokens
    // ✅ HTTP endpoints return 404 with helpful message
    // ✅ Metadata points client to Okta directly
    // ✅ Client authenticates with Okta → Gets token → Calls MCP server

    mux.Handle("/mcp", mcpserver.NewStreamableHTTPServer(mcpServer))
    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

### Proxy Mode Example

**When to use:** Client cannot handle OAuth (e.g., simple CLI tools, legacy clients)

```go
import oauth "github.com/tuannvm/oauth-mcp-proxy"

func main() {
    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0")
    mux := http.NewServeMux()

    // Proxy mode - full config
    oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
        Mode:         "proxy",  // Explicit
        Provider:     "okta",
        Issuer:       "https://company.okta.com",
        Audience:     "api://my-server",
        ClientID:     "client-id",
        ClientSecret: "client-secret",
        ServerURL:    "https://my-server.com",
        RedirectURIs: "https://my-server.com/callback",
    })

    mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

    // What happens:
    // ✅ Middleware validates Bearer tokens
    // ✅ HTTP endpoints fully functional (/oauth/authorize, /callback, /token)
    // ✅ Metadata points client to MCP server
    // ✅ MCP server proxies OAuth flow to Okta
    // ✅ Client authenticates through MCP server

    mux.Handle("/mcp", mcpserver.NewStreamableHTTPServer(mcpServer))
    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

### Mode Auto-Detection + Validation

```go
// Inside WithOAuth():
if cfg.Mode == "" {
    if cfg.ClientID != "" {
        cfg.Mode = "proxy"
    } else {
        cfg.Mode = "native"
    }
}

// Validate mode requirements
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}

// In Config.Validate():
if c.Mode == "proxy" {
    if c.ClientID == "" {
        return errors.New("proxy mode requires ClientID")
    }
    if c.ServerURL == "" {
        return errors.New("proxy mode requires ServerURL")
    }
}
```

---

## Implementation Phases

### Phase 0: Repository Setup

**Goal:** Copy code as-is

**Tasks:**
- [ ] Initialize go.mod (`go mod init github.com/tuannvm/oauth-mcp-proxy`)
- [ ] Add required dependencies to go.mod (latest stable):
  - [ ] `github.com/mark3labs/mcp-go@latest` (v0.41.1)
  - [ ] `github.com/coreos/go-oidc/v3@latest` (v3.16.0)
  - [ ] `github.com/golang-jwt/jwt/v5@latest` (v5.3.0)
  - [ ] `golang.org/x/oauth2@latest` (v0.32.0)
- [ ] Copy all `.go` files from `../mcp-trino/internal/oauth/`
- [ ] Set up .gitignore, LICENSE (MIT), Makefile
- [ ] First commit: "Initial extraction from mcp-trino"

**Success:** Code copied, go.mod with dependencies, `make test` available

---

### Phase 1: Make It Compile

**Goal:** Minimal changes to compile standalone

**Tasks:**
- [ ] Remove Trino-specific imports (`internal/config`)
- [ ] Update imports from `internal/oauth` → root package
- [ ] Replace Trino config types with generic ones
- [ ] Fix compilation errors (minimal changes only)

**Success:** `go build ./...` works (tests can fail, that's ok)

---

### Phase 1.5: Critical Architecture Fixes

**Goal:** Fix fundamental issues before structuring

**Critical Fixes (from Gemini 2.5 Pro review):**
- [ ] **Fix ALL global state**
  - [ ] Global token cache → Instance-scoped in Server struct
  - [ ] Move cache implementation to internal/cache/
  - [ ] Global middleware registry → Remove (if exists)
- [ ] **Add Logger interface** → Replace all log.Printf() calls
- [ ] **Add Config.Validate()** → Validate mode, provider, required fields

**Why now, not v0.2.0:**
- Prevents breaking changes in v0.2.0
- These are fundamental for library usability
- Global state blocks multi-instance usage
- Hardcoded logging unusable in production

**Success:** Zero global variables, logger interface works, config validates on NewServer()

---

### Phase 2: Package Structure

**Goal:** Organize providers into subpackage

**Tasks:**
- [ ] Move provider code to provider/ package
  - [ ] provider/provider.go (TokenValidator interface)
  - [ ] provider/hmac.go
  - [ ] provider/oidc.go
- [ ] **Handlers stay in ROOT** (they need Server internals)
- [ ] **Middleware stays in ROOT** (needs Server, mcp-go types)
- [ ] Cache already in internal/cache/ (done in Phase 1.5)
- [ ] Update imports across codebase

**Success:** Clean package structure, only providers moved, still compiles

---

### Phase 3: Simple API Implementation

**Goal:** Implement WithOAuth() convenience function

**Tasks:**
- [ ] **Implement `oauth.WithOAuth()` in ROOT package**
  - [ ] Create Server internally (calls NewServer with validation)
  - [ ] Apply middleware to mcpServer (using existing middleware.go)
  - [ ] Register HTTP handlers on mux (using Server.RegisterHandlers)
  - [ ] Set up HTTPContextFunc for token extraction
  - [ ] Auto-detect mode if not specified (with validation)
- [ ] Test both native and proxy modes work
- [ ] Test error handling for invalid configs

**Success:** WithOAuth() works for both modes, clear error messages

**Note:** This wraps existing Server/middleware/handler code into one convenient call

---

### Phase 4: OAuth-Only Tests

**Goal:** Make sure it works before shipping

**Tasks:**
- [ ] Copy tests from mcp-trino
- [ ] Remove Trino-specific tests
- [ ] Fix failing tests (OAuth-only)
- [ ] Add integration test (embedded mode)
- [ ] Test all 4 providers work

**Success:** Tests pass, library works end-to-end

---

### Phase 5: Documentation

**Goal:** Complete documentation

**Tasks:**
- [ ] README.md (embedded mode focus)
- [ ] GoDoc comments (all public APIs)
- [ ] examples/embedded/ (working example)
- [ ] Security best practices
- [ ] Provider setup guides
- [ ] Migration guide from mcp-trino

**Success:** Clear README, working example, all APIs documented

---

### Phase 6: Migration Validation

**Goal:** Validate with mcp-trino

**Tasks:**
- [ ] Update mcp-trino to use oauth-mcp-proxy
- [ ] Test with real Trino instance
- [ ] Validate all 4 providers
- [ ] Performance comparison (before/after)
- [ ] Fix any regressions

**Success:** mcp-trino migrates successfully, no regressions

---

## P0 Features (v0.1.0)

### Core
- OAuth token validation (HMAC, OIDC/JWKS)
- 4 providers: HMAC, Okta, Google, Azure AD
- Token caching with TTL (5min default)
- **OAuth modes: native + proxy (auto-detected)**
- PKCE support (RFC 7636)
- OAuth 2.1 metadata endpoints

### Simple API
- **`oauth.WithOAuth()` - One function call integration (composable option)**
- Auto-detection of native vs proxy mode
- Automatic middleware application
- Automatic HTTP handler registration
- Context extraction configured automatically

### Deployment
- Embedded mode (library) ONLY
- Works with any MCP server using mcp-go

### Architecture
- **Instance-scoped state (no globals) - FIXED in Phase 1.5**
- **Logger interface (no hardcoded logging) - FIXED in Phase 1.5**
- **Config validation (fail fast) - FIXED in Phase 1.5**
- Generic configuration (no Trino dependencies)
- Clean package structure (provider/)

---

## Deferred to v0.2.0

### Standalone Mode
See [plan-standalone.md](plan-standalone.md):
- Proxy service binary
- Request routing to downstream MCP servers
- User context propagation
- /validate endpoint
- /health and /metrics
- Service discovery

### Architecture Cleanup (Remaining Issues)
**Rationale:** Fixed critical 3 in v0.1.0 (Phase 1.5), defer others to v0.2.0

**Fixed in v0.1.0 (Phase 1.5):**
- ✅ All Global State → Instance-scoped
  - Global Token Cache → Server.cache
  - Global Middleware Registry → Removed/instance-scoped
- ✅ Hardcoded Logging → Logger interface
- ✅ Configuration Validation → Validate() method

**Deferred to v0.2.0:**
1. Private Context Keys → Public accessors
2. Error Handling → Sentinel errors (use standard errors for now)
3. Graceful Shutdown → Start/Stop methods
4. External Call Timeouts → Configurable
5. Context Cancellation → Comprehensive audit

**Note:** Critical issues (globals, logging, validation) fixed in v0.1.0 to prevent breaking v0.2.0

---

## Success Criteria

### Functional
- Embedded mode works (WithOAuth() function)
- All 4 providers work (HMAC, Okta, Google, Azure)
- Both modes work (native + proxy)
- Token caching reduces load
- mcp-trino migrates (zero breaking changes)

### Non-Functional
- <5ms validation (cache hit)
- <50ms validation (cache miss, OIDC)
- No memory leaks
- No goroutine leaks
- >80% test coverage

### Documentation
- Clear README
- Working embedded example
- Migration guide
- Security practices documented
- All public APIs have GoDoc

---

## Key Principles

1. **Simplest API:** One function call (`WithOAuth`) for MCP developers
2. **MCP-Only:** Library exclusively for MCP servers (no generic abstraction)
3. **Embedded First:** v0.1.0 focuses on library mode only
4. **Quality First:** Fix critical architecture (globals, logging, validation) in v0.1.0
5. **Ship Smart:** Fix fundamentals now, defer nice-to-haves to v0.2.0
6. **Isolation:** mcp-trino unchanged during development
7. **Focus:** OAuth only, no Trino coupling
8. **Minimal Changes:** Copy → Compile → Fix Critical → Structure → Test → Ship
9. **Defer Complexity:** Standalone mode and advanced features in v0.2.0

---

**Status:** ✅ Ready for Phase 0
**Next:** Initialize go.mod and copy source code

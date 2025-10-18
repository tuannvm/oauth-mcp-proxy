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

---

## Package Structure (Simplified for MCP-Only)

```
oauth-mcp-proxy/
├── oauth.go              // Server, Config, NewServer(), EnableOAuth()
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
│   ├── cache/            // Token cache (instance-scoped)
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
- ✅ Providers in provider/ (self-contained)
- ✅ NO adapter/ (library is MCP-only, no abstraction needed)
- ✅ Cache in internal/ (not public API)

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
    oauth.EnableOAuth(mcpServer, mux, &oauth.Config{
        Provider:     "okta",
        Issuer:       "https://company.okta.com",
        Audience:     "api://my-server",
        ClientID:     "client-id",
        ClientSecret: "client-secret",
        ServerURL:    "https://my-server.com",
        RedirectURIs: "https://my-server.com/callback",
    })

    // Done! OAuth is now enabled:
    // ✅ Middleware applied to mcpServer
    // ✅ HTTP handlers registered on mux
    // ✅ Context extraction configured

    // 3. Continue with normal MCP setup
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
    oauth.EnableOAuth(mcpServer, mux, &oauth.Config{
        Mode:     "native",  // Explicit (or auto-detected if omitted)
        Provider: "okta",
        Issuer:   "https://company.okta.com",
        Audience: "api://my-server",
    })

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
    oauth.EnableOAuth(mcpServer, mux, &oauth.Config{
        Mode:         "proxy",  // Explicit
        Provider:     "okta",
        Issuer:       "https://company.okta.com",
        Audience:     "api://my-server",
        ClientID:     "client-id",
        ClientSecret: "client-secret",
        ServerURL:    "https://my-server.com",
        RedirectURIs: "https://my-server.com/callback",
    })

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
// Inside EnableOAuth():
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
- [ ] Initialize go.mod
- [ ] Copy all `.go` files from `../mcp-trino/internal/oauth/`
- [ ] Set up .gitignore, LICENSE (MIT)
- [ ] First commit: "Initial extraction from mcp-trino"

**Success:** Code copied, go.mod exists

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
- [ ] **Fix global token cache** → Instance-scoped (in internal/cache/)
- [ ] **Add Logger interface** → No hardcoded log.Printf()
- [ ] **Add Config.Validate()** → Fail fast on invalid config

**Why now, not v0.2.0:**
- Prevents breaking changes in v0.2.0
- These are fundamental for library usability
- Global state blocks multi-instance usage
- Hardcoded logging unusable in production

**Success:** No global state, logger interface works, config validates

---

### Phase 2: Package Structure

**Goal:** Organize providers into subpackage

**Tasks:**
- [ ] Move provider code to provider/ package
  - [ ] provider/provider.go (TokenValidator interface)
  - [ ] provider/hmac.go
  - [ ] provider/oidc.go
- [ ] **Keep handlers in ROOT** (they need Server internals)
- [ ] Move cache to internal/cache/
- [ ] Update imports across codebase

**Success:** Clean package structure, handlers in root, still compiles

---

### Phase 3: Simple API Implementation

**Goal:** Implement EnableOAuth() function

**Tasks:**
- [ ] **Implement `oauth.EnableOAuth()` in ROOT package**
  - [ ] Auto-detect native vs proxy mode (with validation)
  - [ ] Apply middleware to mcpServer
  - [ ] Register HTTP handlers on mux
  - [ ] Return HTTPContextFunc for streamable server
- [ ] Add mode validation (fail fast on misconfiguration)
- [ ] Test both native and proxy modes work

**Success:** EnableOAuth() works for both modes, clear error messages

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
- **`oauth.EnableOAuth()` - One function call integration (Option A)**
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

**Fixed in v0.1.0:**
- ✅ Global Token Cache → Instance-scoped (Phase 1.5)
- ✅ Hardcoded Logging → Logger interface (Phase 1.5)
- ✅ Configuration Validation → Validate() method (Phase 1.5)

**Deferred to v0.2.0:**
1. Global Middleware Registry → Remove
2. Private Context Keys → Public accessors
3. Error Handling → Sentinel errors
4. Missing Graceful Shutdown → Start/Stop
5. External Call Timeouts → Configurable
6. Context Cancellation → Audit paths

**Note:** Critical issues fixed in v0.1.0, quality improvements in v0.2.0

---

## Success Criteria

### Functional
- Embedded mode works
- All 4 providers work
- Token caching works
- MCP adapter integrates seamlessly
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

1. **Simplest API:** One function call (`EnableOAuth`) for MCP developers
2. **Embedded First:** v0.1.0 focuses exclusively on library
3. **Auto-Magic:** Auto-detect native vs proxy mode from config
4. **Working First:** Make it work (v0.1.0), make it perfect (v0.2.0)
5. **Ship Fast:** Tests pass = ship, don't wait for perfect architecture
6. **Isolation:** mcp-trino unchanged during development
7. **Focus:** OAuth only, no Trino coupling
8. **Minimal Changes:** Copy → Compile → Structure → Test → Document → Ship
9. **Defer Perfection:** Architecture cleanup and standalone mode in v0.2.0

---

**Status:** ✅ Ready for Phase 0
**Next:** Initialize go.mod and copy source code

# oauth-mcp-proxy - MVP Scope (v0.1.0)

> **Goal:** Dual-mode OAuth library (embedded + standalone) that works from day 1

## Executive Summary

**Package Name:** `github.com/tuannvm/oauth-mcp-proxy` ✅ (Confirmed)
**Go Module:** `github.com/tuannvm/oauth-mcp-proxy`
**Delivery:** Working dual-mode OAuth with essential features
**Deferred:** Enterprise features to v0.2.0+

**Key Principles:**
- Copy code from mcp-trino to this repo
- No releases during migration
- No new features until extraction complete
- Focus on OAuth-only tests (remove Trino dependencies)

---

## P0 Features (MUST HAVE - v0.1.0)

### Core Functionality
- ✅ OAuth token validation (HMAC, OIDC/JWKS)
- ✅ Support 4 providers: HMAC, Okta, Google, Azure AD
- ✅ Token caching with TTL (5min default, configurable)
- ✅ Dual OAuth modes: native (direct) + proxy (server-mediated)
- ✅ PKCE support (RFC 7636)
- ✅ OAuth 2.1 metadata endpoints (RFC 8414, RFC 9728)

### Deployment Modes
- ✅ **Embedded mode**: Library imported into MCP server (in-process)
- ✅ **Standalone mode**: Independent OAuth service (out-of-process)
- ✅ `/validate` endpoint for standalone (POST with Bearer token)
- ✅ Basic security for /validate (API key authentication)
- ✅ Standalone binary (`cmd/oauth-mcp-proxy/main.go`)
- ✅ `/health` endpoint (simple HTTP 200 OK)

### Core Architecture
- ✅ Instance-scoped state (no globals)
- ✅ Generic configuration (not Trino-specific)
- ✅ MCP adapter pattern (clean separation)
- ✅ Graceful shutdown
- ✅ Logger interface (pluggable logging)
- ✅ Configuration validation
- ✅ Standard error types
- ✅ Context cancellation support
- ✅ External call timeouts

### Testing & Documentation
- ✅ Unit tests for providers
- ✅ Integration tests (embedded + standalone)
- ✅ Security tests (from current suite)
- ✅ README with quickstart
- ✅ Examples for both modes
- ✅ Migration guide from mcp-trino

---

## P1 Features (SHOULD HAVE - v0.2.0)

### Operational Enhancements
- 🔄 Client library for standalone mode
- 🔄 Circuit breaker pattern (client + server)
- 🔄 Enhanced error metrics (granular)
- 🔄 Service discovery documentation (K8s, Consul)
- 🔄 Dynamic configuration loading
- 🔄 Rate limiting (internal)

### Security Enhancements
- 🔄 mTLS support for /validate
- 🔄 IP allowlist for /validate
- 🔄 Secrets management guide
- 🔄 Audit logging

---

## P2 Features (NICE TO HAVE - v0.3.0+)

### Performance & Scale
- 🔄 Batch validation (`/validate/batch`)
- 🔄 HTTP/2 or gRPC support
- 🔄 Advanced cache eviction (LRU)

### Observability
- 🔄 Distributed tracing (OpenTelemetry)
- 🔄 Prometheus metrics integration
- 🔄 Health check endpoint

### Advanced Features
- 🔄 Multi-tenancy support
- 🔄 Custom provider plugin system
- 🔄 Token introspection endpoint

---

## P0 Issues to Fix (Phase 1)

### Architecture Issues

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

### Deferred Issues (v0.2.0+)

| # | Issue | Priority |
|---|-------|----------|
| 9 | Configuration Loading Strategy | P1 |
| 11 | Rate Limiting Guidance | P1 |
| 12 | Secrets Management Docs | P1 |
| 14 | Error Metrics Granularity | P1 |
| 16 | Client Library | P1 |
| 17 | Circuit Breaker Pattern | P1 |
| 18 | Service Discovery Support | P1 |

---

## Implementation Phases

| Phase | Focus | Deliverables |
|-------|-------|--------------|
| **Phase 0** | **Repository Setup** | Copy code, initialize go.mod, basic structure |
| **Phase 1** | **Fix P0 Issues** | Remove globals, add interfaces, fix architecture |
| **Phase 2** | **Core OAuth Library** | Generic config, providers, handlers, /validate |
| **Phase 3** | **Dual Mode Support** | Embedded + standalone binary + /health endpoint |
| **Phase 4** | **MCP Adapter** | Separate adapter package (no MCP in core) |
| **Phase 5** | **OAuth-Only Tests** | Unit + integration tests without Trino dependencies |
| **Phase 6** | **Documentation** | README, examples, API docs |
| **Phase 7** | **Migration Support** | mcp-trino integration guide, validation |

**Note:** No timeline/deadlines. Complete each phase before moving to next.

---

## MVP Package Structure

```
github.com/tuannvm/oauth-mcp-proxy
├── oauth.go              // Core Server, Config
├── auth.go               // Authentication logic
├── cache.go              // Token cache (instance-scoped)
├── errors.go             // Sentinel error types
├── logger.go             // Logger interface
├── context.go            // Context helpers (public)
├── provider.go           // TokenValidator interface
├── provider_hmac.go      // HMAC validator
├── provider_oidc.go      // OIDC validator
├── handler_authorize.go  // OAuth proxy handlers
├── handler_callback.go
├── handler_token.go
├── handler_metadata.go
├── handler_validate.go   // /validate endpoint (standalone)
├── adapter/
│   └── mcp/             // MCP adapter (optional import)
│       ├── adapter.go
│       └── metadata.go
├── cmd/
│   └── oauth-mcp-proxy/ // Standalone service binary
│       └── main.go
├── internal/            // Private utilities
│   ├── pkce.go
│   ├── state.go
│   └── redirect.go
├── examples/
│   ├── embedded/        // MCP with embedded oauth-mcp-proxy
│   ├── standalone/      // Run oauth-mcp-proxy service
│   └── client/          // MCP calling standalone
└── testutil/
    └── testutil.go
```

**Deferred to v0.2.0:**
- `client/` package (client library)
- Circuit breaker integration
- Batch validation endpoint
- Multi-tenancy support

---

## Success Criteria (v0.1.0)

### Functional
- ✅ Embedded mode: MCP server imports oauth-mcp-proxy, OAuth works
- ✅ Standalone mode: oauth-mcp-proxy binary runs, MCP calls /validate
- ✅ All 4 providers work (HMAC, Okta, Google, Azure)
- ✅ Token caching reduces load (5min TTL)
- ✅ mcp-trino migrated successfully (zero breaking changes)

### Non-Functional
- ✅ <5ms validation with cache hit
- ✅ <50ms validation with cache miss (OIDC)
- ✅ No memory leaks (no global state)
- ✅ No goroutine leaks (graceful shutdown)
- ✅ >80% test coverage

### Documentation
- ✅ Clear README with both modes
- ✅ Working examples for embedded + standalone
- ✅ Migration guide from mcp-trino

---

## What's NOT in v0.1.0

### Explicitly Deferred
❌ Client library (`client/` package) → v0.2.0
❌ Circuit breaker pattern → v0.2.0
❌ Batch validation endpoint → v0.2.0
❌ Distributed tracing → v0.2.0
❌ Multi-tenancy → v0.3.0
❌ Advanced cache eviction (LRU) → v0.2.0
❌ gRPC support → Future
❌ Token introspection endpoint → Future

### Why Defer These?
- **Client library**: Users can write simple HTTP client (30 lines)
- **Circuit breaker**: Can be added by consumers using existing libs
- **Batch validation**: Optimization, not core functionality
- **Tracing/metrics**: Can be added without breaking changes
- **Multi-tenancy**: Advanced use case, need real user feedback first

---

## Standalone Mode: MVP Implementation

### Basic Security (v0.1.0)

```go
// API Key authentication only for v0.1.0
type Config struct {
    ValidateAPIKey string // Single API key for /validate
}

func (s *Server) HandleValidate(w http.ResponseWriter, r *http.Request) {
    // Simple API key check
    if r.Header.Get("X-API-Key") != s.config.ValidateAPIKey {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Validate token
    token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
    user, err := s.validator.ValidateToken(token)
    if err != nil {
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }

    json.NewEncoder(w).Encode(user)
}
```

**Deferred to v0.2.0:**
- Multiple API keys
- mTLS authentication
- IP allowlisting
- Request signing

---

## Migration Path

### v0.1.0 → v0.2.0
- Add client library (non-breaking)
- Add circuit breaker support (non-breaking)
- Add batch endpoint (non-breaking)
- Add observability hooks (non-breaking)

### v0.2.0 → v0.3.0
- Add multi-tenancy (may require config changes)
- Add advanced features based on user feedback

---

## Risk Mitigation

### Top 3 Risks

**1. Dual Mode Complexity**
- **Mitigation**: Clear separation, both modes tested in Phase 5
- **Fallback**: If standalone issues, can ship embedded-only

**2. Timeline Slip**
- **Mitigation**: 30-day timeline has 5-day buffer
- **Fallback**: Can defer documentation polish

**3. Security Issues in Standalone**
- **Mitigation**: Security tests, simple API key auth for v0.1.0
- **Fallback**: Document standalone as "beta" in v0.1.0

---

## Decision Log

### ✅ Confirmed Decisions
1. **Package name:** `oauth-mcp-proxy` (go module: `github.com/tuannvm/oauth-mcp-proxy`)
2. **Dual mode required from day 1** (embedded + standalone)
3. **All P0 features in Phase 0** (11 critical issues fixed before extraction)
4. **4-5 week timeline for MVP** (30 working days)
5. **Defer P1/P2 to v0.2.0+** (ship working MVP first)
6. **Standalone security:** API key only for v0.1.0 (mTLS, IP allowlist → v0.2.0)
7. **No client library in v0.1.0** (users write simple HTTP client, add client lib in v0.2.0)

### 📋 Open Questions
1. ~~Should standalone mode be marked "beta" in v0.1.0?~~ → **NO**, mark as stable if tests pass
2. ~~Do we need /health endpoint in v0.1.0 or v0.2.0?~~ → **YES v0.1.0**, simple HTTP 200 OK endpoint
3. ~~Should we support both `OAUTH_API_KEY` env var and config file?~~ → **YES**, env var priority for v0.1.0

---

---

## Extraction Strategy

**Confirmed Approach:**
1. Copy code from `../mcp-trino/internal/oauth/` to this repo
2. Fix issues here (no changes to mcp-trino during development)
3. No releases until extraction complete
4. Create OAuth-only tests (remove all Trino dependencies)

---

## Phase 0: Repository Setup

**Goal:** Initialize repo, copy source code

**Tasks:**
- [ ] Initialize go.mod (`go mod init github.com/tuannvm/oauth-mcp-proxy`)
- [ ] Copy all `.go` files from `../mcp-trino/internal/oauth/`
- [ ] Set up .gitignore, LICENSE (MIT)
- [ ] Create basic package structure
- [ ] Update imports from `internal/oauth` → root package
- [ ] First commit: "Initial extraction from mcp-trino"

**Success Criteria:**
- go.mod initialized
- All source code copied
- Compiles (even if tests fail)

---

## Phase 1: Fix P0 Architecture Issues

**Goal:** Fix 11 P0 issues

**Core Issues:**
- [ ] Issue #1: Move token cache to Server struct (remove global)
- [ ] Issue #2: Remove global middleware registry
- [ ] Issue #3: Public context key accessors
- [ ] Issue #4: Add Logger interface
- [ ] Issue #5: Add Config.Validate() method
- [ ] Issue #6: Define sentinel error types
- [ ] Issue #7: Separate MCP adapter package
- [ ] Issue #8: Add Start()/Stop() methods (graceful shutdown)
- [ ] Issue #10: Add configurable timeouts
- [ ] Issue #13: Audit context cancellation
- [ ] Issue #15: Add API key auth for /validate endpoint

**Success Criteria:**
- No global variables
- All interfaces defined
- Code follows Go best practices

---

## Phase 5: OAuth-Only Tests

**Goal:** Create tests without Trino dependencies

**Test Categories:**

**Provider Tests:**
- [ ] HMAC token validation
- [ ] OIDC token validation (mock JWKS)
- [ ] Okta provider integration
- [ ] Google provider integration
- [ ] Azure AD provider integration

**Security Tests:**
- [ ] Token cache behavior
- [ ] State parameter signing/verification
- [ ] PKCE flow
- [ ] Redirect URI validation
- [ ] Attack scenario tests (from mcp-trino)

**Integration Tests:**
- [ ] Embedded mode (library usage)
- [ ] Standalone mode (/validate endpoint)
- [ ] API key authentication
- [ ] OAuth proxy flow
- [ ] Token expiration handling

**Remove:**
- ❌ All Trino-specific config tests
- ❌ Tests that import `internal/config`
- ❌ Tests that require Trino database

**Success Criteria:**
- >80% test coverage
- All tests pass
- No external dependencies in tests (use mocks)

---

---

**Document Version:** 1.2
**Last Updated:** 2025-10-17
**Status:** ✅ Ready to start Phase 0
**Project:** `oauth-mcp-proxy`
**Strategy:** Copy code → Fix here → No releases during migration

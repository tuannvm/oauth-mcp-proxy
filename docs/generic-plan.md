# OpenTelemetry Pattern Refactoring Plan

## Implementation Status

**Status**: ✅ **IMPLEMENTED** (2025-10-22)

**Completion**: 82% (Core implementation complete, documentation pending)

**See**: `docs/generic-implementation.md` for detailed checkpoint tracking.

---

## Overview

This document outlines the plan to refactor `oauth-mcp-proxy` to support both mark3labs/mcp-go and the official modelcontextprotocol/go-sdk using the OpenTelemetry pattern approach.

## Original State

The library originally supported only `github.com/mark3labs/mcp-go` (v0.41.1) with a single `WithOAuth()` function that returns `mcpserver.ServerOption`.

## Proposed Structure

Following the OpenTelemetry instrumentation pattern, we'll organize the codebase as:

```
oauth-mcp-proxy/
├── [core package - SDK-agnostic]
│   ├── server.go         (Server, NewServer, RegisterHandlers, WrapHandler)
│   ├── config.go         (Config, validation)
│   ├── cache.go          (TokenCache, token caching logic)
│   ├── context.go        (WithOAuthToken, GetOAuthToken, GetUserFromContext)
│   ├── handlers.go       (OAuth HTTP endpoints)
│   ├── logger.go         (Logger interface, defaultLogger)
│   ├── metadata.go       (OAuth metadata structures)
│   └── provider/         (TokenValidator interface, HMAC/OIDC validators)
│       ├── provider.go
│       └── provider_test.go
│
├── mark3labs/           [SDK-specific adapter]
│   ├── oauth.go         (WithOAuth → ServerOption)
│   └── middleware.go    (Middleware adapter for mark3labs types)
│
└── mcp/                 [SDK-specific adapter]
    └── oauth.go         (WithOAuth → http.Handler)
```

## Package Naming Convention

Following OpenTelemetry's pattern:

```
github.com/tuannvm/oauth-mcp-proxy              (core, SDK-agnostic)
github.com/tuannvm/oauth-mcp-proxy/mark3labs    (mark3labs/mcp-go adapter)
github.com/tuannvm/oauth-mcp-proxy/mcp          (official SDK adapter)
```

## API Examples

### mark3labs (existing SDK)

```go
import (
    "github.com/mark3labs/mcp-go/server"
    "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

mux := http.NewServeMux()
oauthServer, oauthOption, err := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
})

mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
```

### Official SDK (new support)

```go
import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

mux := http.NewServeMux()
mcpServer := mcp.NewServer(&mcp.Implementation{
    Name:    "time-server",
    Version: "1.0.0",
}, nil)

oauthServer, handler, err := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
}, mcpServer)

http.ListenAndServe(":8080", handler)
```

## Pros

### 1. Clean Separation of Concerns

90% of the OAuth logic (validation, caching, config, providers) remains SDK-agnostic. Only adapters are SDK-specific.

### 2. Easier Maintenance

Bug fixes and new features in the core benefit both SDKs automatically. No need to duplicate logic.

### 3. Clear API Contracts

Users explicitly import the SDK-specific package they need. The import path makes intent clear:

- `oauth-mcp-proxy/mark3labs` → I'm using mark3labs SDK
- `oauth-mcp-proxy/mcp` → I'm using official SDK

### 4. Discoverability

Package structure clearly communicates "this library supports multiple SDKs" and makes it easy to find the right integration.

### 5. Future Extensibility

Adding support for SDK v3 or another MCP implementation = create new adapter package. Core remains unchanged.

### 6. Follows Go Ecosystem Patterns

Same approach used by:

- OpenTelemetry (`go.opentelemetry.io/contrib/instrumentation/.../otel{package}`)
- Sentry (`github.com/getsentry/sentry-go/{framework}`)
- Datadog, NewRelic, and other observability libraries

## Cons

### 1. Import Path Changes (Breaking Change)

Existing users must update imports:

**Before:**

```go
import "github.com/tuannvm/oauth-mcp-proxy"
oauth.WithOAuth(mux, cfg)
```

**After:**

```go
import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
mark3labs.WithOAuth(mux, cfg)
```

### 2. Requires Major Version Bump

This is a breaking change requiring v1 → v2 semver bump.

### 3. More Files

Adds 2-3 new files vs current monolithic approach. Slightly more complex directory structure.

### 4. Documentation Updates Required

All examples, README.md, CLAUDE.md, and tutorials need updates to reflect new import paths.

## Refactoring Complexity Assessment

**Overall Complexity: MEDIUM**

### Code Distribution

| Component | Lines | Location | Effort |
|-----------|-------|----------|--------|
| Core OAuth logic | ~800 | Root package | Move/rename (2 hours) |
| mark3labs adapter | ~40 | New: mark3labs/ | Extract (30 min) |
| Official SDK adapter | ~30 | New: mcp/ | Write new (30 min) |
| Tests | ~1000 | Update imports | Update (1 hour) |
| Documentation | Multiple files | Update all | Update (1 hour) |

### What Moves Where

**Core Package (stays at root):**

- ✅ `Server` type (remove SDK-specific methods)
- ✅ `Config`, validation, providers
- ✅ `TokenCache`, `CachedToken`
- ✅ Context functions (`WithOAuthToken`, `GetOAuthToken`, `GetUserFromContext`)
- ✅ HTTP handlers (OAuth endpoints)
- ✅ `WrapHandler` (already SDK-agnostic)
- ✅ Logger interface and implementation

**mark3labs/ (extract ~40 lines):**

- `WithOAuth()` → returns `(*oauth.Server, mcpserver.ServerOption, error)`
- `Middleware()` → wraps mark3labs-specific types
- `GetHTTPServerOptions()` → returns `[]mcpserver.StreamableHTTPOption`

**mcp/ (write ~30 new lines):**

- `WithOAuth()` → returns `(*oauth.Server, http.Handler, error)`
- Handler wrapper using `mcp.NewStreamableHTTPHandler()`
- Integration with official SDK's HTTP model

### Migration Effort Breakdown

| Phase | Tasks | Time Estimate |
|-------|-------|---------------|
| **Code Refactoring** | Extract core, create adapters, update imports | 2-3 hours |
| **Testing** | Verify both integrations, update tests | 1-2 hours |
| **Documentation** | Update README, examples, migration guide | 1 hour |
| **Validation** | Run full test suite, manual testing | 30 min |

**Total Estimated Effort: 1 day of focused work**

## Migration Strategy

### For Library Maintainers

1. **Phase 1**: Core extraction (keep existing API working)
2. **Phase 2**: Create mark3labs adapter
3. **Phase 3**: Create mcp adapter
4. **Phase 4**: Update tests
5. **Phase 5**: Update documentation
6. **Phase 6**: Release v2.0.0 with migration guide

### For Library Users

Users have two migration paths:

**Option 1: Quick Update (mark3labs users)**

```diff
- import "github.com/tuannvm/oauth-mcp-proxy"
+ import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"

- oauth.WithOAuth(mux, cfg)
+ mark3labs.WithOAuth(mux, cfg)
```

**Option 2: Migrate to Official SDK**
Follow the official SDK migration guide in the new documentation.

---

## Implementation Results

### What Was Implemented

**Date**: 2025-10-22

**Files Created:**
- `cache.go` (68 lines) - Token cache logic
- `context.go` (46 lines) - Context utilities (WithOAuthToken, GetOAuthToken, WithUser, GetUserFromContext)
- `mark3labs/oauth.go` (45 lines) - mark3labs SDK adapter
- `mark3labs/middleware.go` (38 lines) - mark3labs middleware implementation
- `mcp/oauth.go` (76 lines) - Official SDK adapter
- `verify_context_test.go` - Context propagation verification test

**Files Modified:**
- `oauth.go` - Added ValidateTokenCached() method
- `middleware.go` - Removed extracted code to new files
- `examples/simple/main.go` - Updated to use mark3labs package
- `examples/advanced/main.go` - Updated to use mark3labs package
- `go.mod` - Added official SDK v1.0.0

**Verification:**
- ✅ All existing tests pass
- ✅ Both example apps build successfully
- ✅ Official SDK context propagation verified
- ✅ Core API contract implemented as designed

### Implementation Time

**Actual Time**: ~3 hours (vs 1 day estimated)

Faster than estimated due to:
- Clear verification phase eliminated uncertainty
- Well-defined core API contract
- Minimal changes needed to existing tests

### Deviations from Plan

1. **Checkpoint 3.4 Skipped**: Official SDK example not created (can be added later)
2. **Checkpoint 4.2 & 4.3 Pending**: Adapter-specific integration tests deferred to follow-up PR
3. **mcp/oauth.go**: Implemented custom HTTP handler wrapper instead of using WrapHandler for more explicit control

### Outstanding Work

- **Phase 5**: README.md updates (show both SDKs, migration guide)
- **Phase 6**: Release preparation (CHANGELOG, version bump, PR)
- **Future**: Comprehensive adapter integration tests

---

## Open Questions

### Answered (Based on Gemini 2.5 Pro Review)

1. **Should we maintain v1 branch for bug fixes during transition period?**
   - ✅ Yes. Create `v1` branch from last commit before refactor. Support critical security fixes for 3-6 months.

2. **How long should we support v1 before deprecating?**
   - ✅ 3-6 months for critical security fixes only. No new features.

3. **Should we add compatibility shims in v2 to ease migration?**
   - ❌ No. Major version bump is the time for clean break. Shims add complexity and confusion. Use clear migration guide instead.

## References

- OpenTelemetry Go Contrib: <https://github.com/open-telemetry/opentelemetry-go-contrib>
- Sentry Go SDK: <https://github.com/getsentry/sentry-go>
- Official MCP Go SDK: <https://github.com/modelcontextprotocol/go-sdk>

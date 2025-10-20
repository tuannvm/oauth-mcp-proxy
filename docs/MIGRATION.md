# Migration Guide: mcp-trino → oauth-mcp-proxy

This guide helps mcp-trino users migrate to the standalone oauth-mcp-proxy library.

---

## Why Migrate?

**Benefits:**

- ✅ Latest OAuth improvements and security fixes
- ✅ Reusable across any MCP server (not Trino-specific)
- ✅ Better API (`WithOAuth()` vs manual setup)
- ✅ Pluggable logging support
- ✅ Active maintenance in dedicated repo
- ✅ No Trino dependencies

**Timeline:** mcp-trino will migrate to oauth-mcp-proxy in a future release.

---

## Breaking Changes

### Import Path

**Before (mcp-trino):**

```go
import "github.com/tuannvm/mcp-trino/internal/oauth"
```

**After (oauth-mcp-proxy):**

```go
import oauth "github.com/tuannvm/oauth-mcp-proxy"
```

### API Changes

| Old (mcp-trino) | New (oauth-mcp-proxy) | Notes |
|---|---|---|
| `oauth.SetupOAuth()` | `oauth.WithOAuth()` | New API is simpler |
| `oauth.OAuthMiddleware()` | `oauth.WithOAuth()` | Returns server option |
| `internal/oauth` package | Root `oauth` package | Now public API |

---

## Migration Steps

### Step 1: Add Dependency

```bash
go get github.com/tuannvm/oauth-mcp-proxy
```

### Step 2: Update Imports

```diff
- import "github.com/tuannvm/mcp-trino/internal/oauth"
+ import oauth "github.com/tuannvm/oauth-mcp-proxy"
```

### Step 3: Migrate Configuration

**Before (mcp-trino):**

```go
// Old internal API
validator, err := oauth.SetupOAuth(&oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://trino-server",
})

middleware := oauth.OAuthMiddleware(validator, true)

mcpServer := server.NewMCPServer("Trino", "1.0.0",
    server.WithToolHandlerMiddleware(middleware),
)
```

**After (oauth-mcp-proxy):**

```go
// New simple API
mux := http.NewServeMux()

_, oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://trino-server",
})

mcpServer := server.NewMCPServer("Trino", "1.0.0", oauthOption)
```

**Differences:**

- ✅ Simpler: 1 function call vs 3
- ✅ `mux` passed to WithOAuth (auto-registers endpoints)
- ✅ Returns `mcpserver.ServerOption` directly
- ✅ No manual middleware wrapping needed

### Step 4: Update HTTP Context Setup

**Before (mcp-trino):**

```go
// Manual token extraction
oauthContextFunc := func(ctx context.Context, r *http.Request) context.Context {
    authHeader := r.Header.Get("Authorization")
    if strings.HasPrefix(authHeader, "Bearer ") {
        token := strings.TrimPrefix(authHeader, "Bearer ")
        ctx = oauth.WithOAuthToken(ctx, token)
    }
    return ctx
}

streamableServer := mcpserver.NewStreamableHTTPServer(
    mcpServer,
    mcpserver.WithHTTPContextFunc(oauthContextFunc),
)
```

**After (oauth-mcp-proxy):**

```go
// Use helper function
streamableServer := mcpserver.NewStreamableHTTPServer(
    mcpServer,
    mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
)
```

**Difference:** Helper function provided for convenience.

### Step 5: Update User Context Access

**Before & After (same):**

```go
func toolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    user, ok := oauth.GetUserFromContext(ctx)
    if !ok {
        return nil, fmt.Errorf("authentication required")
    }
    // Use user.Subject, user.Email, user.Username
}
```

No changes needed! ✅

---

## Complete Example

### Before (mcp-trino internal OAuth)

```go
package main

import (
    "github.com/tuannvm/mcp-trino/internal/oauth"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    // Step 1: Setup OAuth
    validator, err := oauth.SetupOAuth(&oauth.Config{
        Provider: "okta",
        Issuer:   "https://company.okta.com",
        Audience: "api://trino",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Step 2: Create middleware
    middleware := oauth.OAuthMiddleware(validator, true)

    // Step 3: Create server with middleware
    mcpServer := server.NewMCPServer("Trino", "1.0.0",
        server.WithToolHandlerMiddleware(middleware),
    )

    // Step 4: Manual HTTP setup
    mux := http.NewServeMux()
    // ... register OAuth handlers manually ...

    // Step 5: Create context func manually
    contextFunc := func(ctx context.Context, r *http.Request) context.Context {
        authHeader := r.Header.Get("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            token := strings.TrimPrefix(authHeader, "Bearer ")
            ctx = oauth.WithOAuthToken(ctx, token)
        }
        return ctx
    }

    streamable := server.NewStreamableHTTPServer(mcpServer,
        server.WithHTTPContextFunc(contextFunc),
    )
    mux.Handle("/mcp", streamable)

    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

### After (oauth-mcp-proxy)

```go
package main

import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    mux := http.NewServeMux()

    // Step 1: Enable OAuth (one call!)
    _, oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{
        Provider: "okta",
        Issuer:   "https://company.okta.com",
        Audience: "api://trino",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Step 2: Create server with OAuth
    mcpServer := server.NewMCPServer("Trino", "1.0.0", oauthOption)

    // Step 3: Setup MCP endpoint (use helper)
    streamable := server.NewStreamableHTTPServer(mcpServer,
        server.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
    )
    mux.Handle("/mcp", streamable)

    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}
```

**From ~40 lines → ~20 lines** ✅

---

## Configuration Mapping

| mcp-trino Config | oauth-mcp-proxy Config | Notes |
|---|---|---|
| `Provider` | `Provider` | Same |
| `Issuer` | `Issuer` | Same |
| `Audience` | `Audience` | Same |
| `ClientID` | `ClientID` | Same |
| `ClientSecret` | `ClientSecret` | Same |
| `MCPHost + MCPPort` | `ServerURL` | Simplified to one field |
| `RedirectURIs` | `RedirectURIs` | Same |
| `JWTSecret` | `JWTSecret` | Same |
| N/A | `Logger` | **New:** Pluggable logging |
| N/A | `Mode` | **New:** Auto-detected |

---

## New Features

### Pluggable Logging

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://trino",
    Logger:   &myCustomLogger{},  // NEW!
})
```

### Auto-Mode Detection

```go
// Native mode auto-detected (no ClientID)
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "...",
    Audience: "...",
})

// Proxy mode auto-detected (has ClientID)
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    ClientID: "...",  // Triggers proxy mode
    ServerURL: "...",
})
```

No need to set `Mode` explicitly unless you want to.

---

## Testing Migration

### 1. Keep Old Code Commented

```go
// Old mcp-trino OAuth
// validator, err := trinoOAuth.SetupOAuth(...)

// New oauth-mcp-proxy
_, oauthOption, err := oauth.WithOAuth(mux, &oauth.Config{...})
```

### 2. Test Locally

```bash
go run main.go
# Verify OAuth endpoints work
curl http://localhost:8080/.well-known/oauth-authorization-server
```

### 3. Test Authentication

Use same test tokens as before - token validation logic unchanged.

### 4. Deploy & Monitor

- Watch logs for OAuth errors
- Verify users can authenticate
- Check token caching works (look for cache hit logs)

---

## Rollback Plan

If issues occur:

```go
// Comment out new code
// _, oauthOption, err := oauth.WithOAuth(...)

// Uncomment old code
validator, err := trinoOAuth.SetupOAuth(...)
middleware := trinoOAuth.OAuthMiddleware(validator, true)
mcpServer := server.NewMCPServer("Trino", "1.0.0",
    server.WithToolHandlerMiddleware(middleware),
)
```

Redeploy. OAuth logic is identical, just packaged differently.

---

## Support

Questions? Check:

- [README.md](../README.md) - Quick start
- [Provider Guides](./providers/) - Provider-specific setup
- [SECURITY.md](./SECURITY.md) - Security best practices
- [GitHub Issues](https://github.com/tuannvm/oauth-mcp-proxy/issues)

---

## Timeline

- **Now:** oauth-mcp-proxy v0.1.0 available
- **Future:** mcp-trino will update to use oauth-mcp-proxy library
- **Support:** Both approaches work, new approach recommended

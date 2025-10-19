# OAuth MCP Proxy Examples

## 1. Simple API (Phase 3) - `simple/`

**Simplest OAuth setup using `WithOAuth()` API:**

```go
mux := http.NewServeMux()

// Line 1: Get OAuth option (registers HTTP handlers)
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-server",
    JWTSecret: []byte("secret"),
})

// Line 2: Create MCP server with OAuth
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)

// Add tools - automatically OAuth-protected!
mcpServer.AddTool(tool, handler)

// Setup MCP endpoint with token extraction
streamable := server.NewStreamableHTTPServer(mcpServer,
    server.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
)
```

**Features:**
- **2-line OAuth setup** (Phase 3 goal achieved!)
- Server-wide middleware (all tools protected automatically)
- Composable with other server options
- Uses mcp-go v0.41.1 `WithToolHandlerMiddleware` pattern

**Run:** `go run examples/simple/main.go`

**Test:**
```bash
curl -X POST http://localhost:8080/mcp \
  -H 'Authorization: Bearer <token-from-output>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

---

## 2. Embedded Mode (Phase 2) - `embedded/`

**Detailed implementation showing internal architecture:**

`embedded/main.go` - Complete MCP server with OAuth authentication

**Features:**
- Full MCP server implementation (using mark3labs/mcp-go v0.41.1)
- Shows internal `NewServer()` API
- Demonstrates `Server.Middleware()` usage
- Shows manual middleware application
- provider/ package architecture visible
- Context propagation explained
- Instance-scoped state (no globals)
- HMAC token validation with caching
- OAuth metadata endpoints
- Auto-generates test token

**Run:** `go run examples/embedded/main.go`

**What It Demonstrates (Phase 2 internals):**
1. Creating OAuth server (`oauth.NewServer()`)
2. Getting middleware (`server.Middleware()`)
3. Server-wide middleware with `WithToolHandlerMiddleware`
4. provider/ package isolation (HMACValidator from provider/)
5. Context propagation (`ValidateToken(ctx, token)`)
6. Instance-scoped cache (Server.cache, no globals)
7. OAuth context extraction from HTTP headers
8. User context available in MCP tools

---

## Comparison

| Feature | `simple/` (Phase 3) | `embedded/` (Phase 2) |
|---------|---------------------|----------------------|
| **Setup Lines** | 2 lines | ~10 lines |
| **API** | `WithOAuth()` convenience | `NewServer()` + manual setup |
| **Recommended** | ✅ For production | For learning internals |
| **Shows** | Simplest usage | Architecture details |

**Recommendation:** Use `simple/` for real projects, read `embedded/` to understand how it works.

---

## OAuth Endpoints

Both examples expose:
- `POST /mcp` - MCP protocol endpoint (OAuth protected)
- `GET /.well-known/oauth-authorization-server` - OAuth metadata
- `GET /.well-known/oauth-protected-resource` - Resource metadata
- `GET /.well-known/jwks.json` - JWKS keys (HMAC mode)
- `GET /.well-known/openid-configuration` - OIDC discovery

# OAuth MCP Proxy Examples

## Quick Start: 3 Lines of Code

```go
// 1. Get OAuth option
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{Provider: "hmac", Audience: "api://my-server", JWTSecret: []byte("secret")})

// 2. Create MCP server with OAuth
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)

// 3. Add tools - automatically OAuth-protected!
mcpServer.AddTool(tool, handler)
```

That's it! All your MCP tools are now protected by OAuth authentication.

---

## 1. Simple API - `simple/`

**Recommended for all production usage. Complete working example:**

```go
mux := http.NewServeMux()

// Get OAuth option (registers HTTP handlers automatically)
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-server",
    JWTSecret: []byte("secret"),
})

// Create MCP server with OAuth middleware
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)

// Add tools - all automatically OAuth-protected
mcpServer.AddTool(tool, handler)

// Setup MCP endpoint with token extraction
streamable := server.NewStreamableHTTPServer(mcpServer,
    server.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
)
mux.Handle("/mcp", streamable)
```

**What you get:**
- All tools OAuth-protected automatically
- OAuth HTTP endpoints registered
- Token validation with caching
- User context in tool handlers
- Production-ready security
- Pluggable logging (optional custom logger)

**Run:** `cd examples/simple && go run main.go`

**Test:**
```bash
curl -X POST http://localhost:8080/mcp \
  -H 'Authorization: Bearer <token-from-output>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

---

## 2. Advanced: Internal Architecture - `embedded/`

**For understanding how the library works internally. Not recommended for production.**

Shows lower-level APIs:
- `oauth.NewServer()` - Manual server creation
- `server.Middleware()` - Manual middleware application
- `server.RegisterHandlers()` - Manual endpoint registration
- Custom context extraction
- Provider package isolation

**Run:** `cd examples/embedded && go run main.go`

---

## Comparison

| | `simple/` | `embedded/` |
|---|---|---|
| **Lines of code** | 3 core lines | ~15 lines |
| **Use case** | Production | Learning internals |
| **API** | `WithOAuth()` | `NewServer()` + manual |
| **Recommended** | ✅ Yes | Only for learning |

Use `simple/` for real projects. Read `embedded/` to understand internals.

---

## Supported Providers

```go
// HMAC (shared secret)
oauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    JWTSecret: []byte("your-secret-key"),
    Audience:  "api://my-server",
})

// Okta
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
})

// Google
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "google",
    Issuer:   "https://accounts.google.com",
    Audience: "your-client-id.apps.googleusercontent.com",
})

// Azure AD
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "azure",
    Issuer:   "https://login.microsoftonline.com/{tenant}/v2.0",
    Audience: "api://your-app-id",
})
```

All providers support both native mode (client handles OAuth) and proxy mode (server proxies OAuth flow).

---

## Custom Logging

Control OAuth logging by providing your own logger:

```go
// Implement the Logger interface
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...interface{}) { /* custom implementation */ }
func (l *MyLogger) Info(msg string, args ...interface{})  { /* custom implementation */ }
func (l *MyLogger) Warn(msg string, args ...interface{})  { /* custom implementation */ }
func (l *MyLogger) Error(msg string, args ...interface{}) { /* custom implementation */ }

// Use it in your config
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
    Provider: "hmac",
    Audience: "api://my-server",
    JWTSecret: []byte("secret"),
    Logger: &MyLogger{}, // Your custom logger
})
```

**Default behavior:** If no logger provided, uses `log.Printf` with level prefixes (`[INFO]`, `[ERROR]`, `[WARN]`, `[DEBUG]`).

**What gets logged:**
- Authorization requests and callbacks
- Token validation (with token hash for security)
- Security violations (invalid redirects, state verification failures)
- OAuth flow errors
- HTTP endpoint access

# OAuth MCP Proxy Examples

## Embedded Mode Example

`embedded.go` - Complete MCP server with OAuth authentication

**Features:**
- Full MCP server implementation (using mark3labs/mcp-go v0.41.1)
- **Server-wide OAuth middleware** (WithToolHandlerMiddleware)
- OAuth-protected MCP tool ("hello")
- Context propagation (HTTP → MCP → OAuth → Tool)
- HMAC token validation with caching
- OAuth metadata endpoints
- Auto-generates test token for easy testing

**Run:**
```bash
go run examples/embedded.go
```

**Test:**
```bash
# Server will print the test command on startup

# Test MCP tool call with OAuth:
curl -X POST http://localhost:8080/mcp \
  -H 'Authorization: Bearer <token-from-output>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

**What It Demonstrates (Phase 2 Features):**
1. Creating OAuth server (`oauth.NewServer()`)
2. **Server-wide middleware** (`WithToolHandlerMiddleware`) - mcp-go v0.41.1
3. provider/ package isolation (HMACValidator from provider/)
4. Context propagation (`ValidateToken(ctx, token)`)
5. Instance-scoped state (Server.cache, no globals)
6. OAuth context extraction from HTTP headers
7. User context available in MCP tools
8. Complete end-to-end OAuth flow

**Endpoints:**
- `POST /mcp` - MCP protocol endpoint (OAuth protected)
- `GET /.well-known/oauth-authorization-server` - OAuth metadata
- `GET /.well-known/oauth-protected-resource` - Resource metadata

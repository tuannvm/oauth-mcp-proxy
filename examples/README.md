# OAuth MCP Proxy Examples

## Embedded Mode Example

`embedded.go` - Complete MCP server with OAuth authentication

**Features:**
- Full MCP server implementation (using mark3labs/mcp-go)
- OAuth-protected MCP tool ("hello")
- Demonstrates middleware application to tools
- HMAC token validation
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

**What It Demonstrates:**
1. Creating OAuth server (`oauth.NewServer()`)
2. Creating MCP server with tools
3. **Applying OAuth middleware to tools** (wraps tool handler)
4. OAuth context extraction from HTTP headers
5. User context available in MCP tools
6. Complete end-to-end OAuth flow

**Endpoints:**
- `POST /mcp` - MCP protocol endpoint (OAuth protected)
- `GET /.well-known/oauth-authorization-server` - OAuth metadata
- `GET /.well-known/oauth-protected-resource` - Resource metadata

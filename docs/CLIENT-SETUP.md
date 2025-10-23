# Client Configuration Guide

How MCP clients discover and use OAuth authentication with your server.

---

## Overview

When you enable OAuth on your MCP server, clients need to know:

1. **How to authenticate** - OAuth provider details
2. **Where to get tokens** - Authorization endpoints
3. **How to send tokens** - Authorization header format

This library provides **automatic discovery** via OAuth 2.0 metadata endpoints.

---

## Client Auto-Discovery (Recommended)

### How It Works

```mermaid
sequenceDiagram
    participant C as MCP Client
    participant S as Your MCP Server
    participant P as OAuth Provider

    Note over C: User adds server to client

    C->>S: GET /.well-known/oauth-authorization-server
    S->>C: OAuth metadata (issuer, endpoints, etc.)

    Note over C: Client auto-configures OAuth

    C->>P: OAuth flow (authorization code)
    P->>C: Access token

    C->>S: POST /mcp + Bearer token
    S->>C: Authenticated tool response
```

**Clients that support auto-discovery:**

- Claude Desktop (native OAuth)
- Claude Code (native OAuth)
- MCP Inspector (browser OAuth)

### Client Configuration

**Claude Desktop** (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp"
    }
  }
}
```

That's it! Claude Desktop will:

1. Fetch `https://your-server.com/.well-known/oauth-authorization-server`
2. Discover OAuth issuer and endpoints
3. Guide user through OAuth flow
4. Store and manage tokens automatically

---

## Manual Client Configuration

For clients without auto-discovery:

### With Bearer Token (Pre-obtained)

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_TOKEN_HERE"
      }
    }
  }
}
```

**How to get token:**

- HMAC: Generate using `jwt.NewWithClaims()` (see [HMAC Guide](providers/HMAC.md))
- OIDC: Use OAuth provider's token endpoint or admin tools

### Proxy Mode (Server Handles OAuth)

For simple clients that can't do OAuth:

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp",
      "oauth": {
        "authorizationUrl": "https://your-server.com/oauth/authorize",
        "tokenUrl": "https://your-server.com/oauth/token"
      }
    }
  }
}
```

Client can now use your server's OAuth endpoints instead of going directly to the provider.

---

## OAuth Metadata Endpoints

Your server automatically exposes (when using `WithOAuth()`):

### OAuth 2.0 Authorization Server Metadata (RFC 8414)

```bash
GET https://your-server.com/.well-known/oauth-authorization-server
```

**Returns:**

```json
{
  "issuer": "https://your-server.com",
  "authorization_endpoint": "https://your-server.com/oauth/authorize",
  "token_endpoint": "https://your-server.com/oauth/token",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code"],
  "code_challenge_methods_supported": ["plain", "S256"]
}
```

### OIDC Discovery

```bash
GET https://your-server.com/.well-known/openid-configuration
```

Returns similar metadata with OIDC-specific fields.

### Protected Resource Metadata

```bash
GET https://your-server.com/.well-known/oauth-protected-resource
```

Tells clients this is an OAuth-protected resource.

---

## Configuration By Mode

### Native Mode

**Server config (mark3labs SDK):**

```go
import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
})
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
```

**Server config (official SDK):**

```go
import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"

mcpServer := mcp.NewServer(&mcp.Implementation{...}, nil)
_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
}, mcpServer)
```

**Client discovers:**

- Metadata endpoints return Okta URLs
- Client authenticates directly with Okta
- Client sends Okta token to your server
- Your server validates token against Okta

**Client config (auto-discovery):**

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp"
    }
  }
}
```

Client fetches metadata, sees Okta issuer, handles OAuth with Okta directly.

### Proxy Mode

**Server config (mark3labs SDK):**

```go
import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider:     "okta",
    ClientID:     "...",
    ClientSecret: "...",
    ServerURL:    "https://your-server.com",
    RedirectURIs: "https://your-server.com/oauth/callback",
})
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
```

**Server config (official SDK):**

```go
import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"

mcpServer := mcp.NewServer(&mcp.Implementation{...}, nil)
_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider:     "okta",
    ClientID:     "...",
    ClientSecret: "...",
    ServerURL:    "https://your-server.com",
    RedirectURIs: "https://your-server.com/oauth/callback",
}, mcpServer)
```

**Client discovers:**

- Metadata endpoints return YOUR server URLs (not Okta)
- Client authenticates through your server
- Your server proxies to Okta
- Client sends token from your server

**Client config (auto-discovery):**

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp"
    }
  }
}
```

Client fetches metadata, sees your server as issuer, does OAuth flow through your server.

---

## Deployment Configuration

### Environment Variables (Recommended)

```bash
# OAuth provider
export OAUTH_PROVIDER=okta
export OAUTH_ISSUER=https://company.okta.com
export OAUTH_AUDIENCE=api://my-server

# Proxy mode (if needed)
export OAUTH_CLIENT_ID=your-client-id
export OAUTH_CLIENT_SECRET=your-client-secret
export OAUTH_SERVER_URL=https://your-server.com
export OAUTH_REDIRECT_URIS=https://your-server.com/oauth/callback

# HMAC (if using)
export JWT_SECRET=your-32-byte-secret
```

### Kubernetes (Helm)

```yaml
# values.yaml
oauth:
  enabled: true
  mode: native  # or proxy
  provider: okta
  redirectURIs: ""  # For proxy mode

  oidc:
    issuer: https://company.okta.com
    audience: api://my-server
    clientId: ""        # For proxy mode
    clientSecret: ""    # For proxy mode (stored in Secret)
```

### Docker Compose

```yaml
services:
  mcp-server:
    image: your-mcp-server:latest
    environment:
      OAUTH_PROVIDER: okta
      OAUTH_ISSUER: https://company.okta.com
      OAUTH_AUDIENCE: api://my-server
    env_file:
      - .env.secrets  # Contains OAUTH_CLIENT_SECRET, JWT_SECRET
```

---

## Testing Client Setup

### 1. Verify Metadata Endpoints

```bash
# Check OAuth discovery
curl https://your-server.com/.well-known/oauth-authorization-server | jq

# Verify issuer matches expected provider
jq '.issuer' # Should be your provider (native) or your server (proxy)
```

### 2. Test Manual Authentication

```bash
# For HMAC - generate test token
# See examples/mark3labs/simple/ or examples/official/simple/ for token generation

# For OIDC - get token from provider (see examples/README.md for Okta setup)
# Test with curl
curl -X POST https://your-server.com/mcp \
  -H "Authorization: Bearer <token>" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### 3. Test Client Auto-Discovery

Add server to Claude Desktop and verify:

- OAuth flow initiates automatically
- No manual token configuration needed
- Authentication works end-to-end

---

## Troubleshooting Client Issues

### Client Can't Discover OAuth

**Check:**

```bash
curl https://your-server.com/.well-known/oauth-authorization-server
# Should return 200 with JSON metadata
```

If 404, verify `WithOAuth()` was called and server is running.

### Client Shows "Authentication Required"

**Check:**

1. Client is sending `Authorization: Bearer <token>` header
2. Token is valid (not expired)
3. Token's `iss` and `aud` match server config

**Debug:**
Enable verbose logging in client if available.

### OAuth Flow Fails

**Native mode:**

- Check client can reach OAuth provider directly
- Verify provider's redirect URIs include client's callback

**Proxy mode:**

- Check client can reach your server's /oauth endpoints
- Verify your server's redirect URIs configured in provider

---

## Client Configuration Examples

### Claude Desktop

**Location:**

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

**Config:**

```json
{
  "mcpServers": {
    "my-oauth-server": {
      "url": "https://mcp-server.example.com/mcp"
    }
  }
}
```

Claude Desktop auto-discovers OAuth via metadata endpoints.

### Cursor / Other MCP Clients

**With auto-discovery:**

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp"
    }
  }
}
```

**With manual token:**

```json
{
  "mcpServers": {
    "my-server": {
      "url": "https://your-server.com/mcp",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
      }
    }
  }
}
```

---

## See Also

- [Configuration Guide](CONFIGURATION.md) - Server-side OAuth configuration
- [Provider Guides](providers/) - OAuth provider setup
- [Troubleshooting](TROUBLESHOOTING.md) - Common client issues

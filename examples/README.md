# OAuth MCP Proxy Examples

This directory contains example MCP servers demonstrating OAuth integration with both supported SDKs.

## Directory Structure

```
examples/
├── mark3labs/              (mark3labs/mcp-go SDK examples)
│   ├── simple/            - Basic OAuth integration
│   └── advanced/          - ConfigBuilder, env vars, multiple tools
│
└── official/               (modelcontextprotocol/go-sdk examples)
    ├── simple/            - Basic OAuth integration
    └── advanced/          - Multiple tools, env vars, logging
```

## Examples Overview

| SDK | Example | Tools | Provider | Features |
|-----|---------|-------|----------|----------|
| **mark3labs** | simple | 1 (greet) | Okta | Basic OAuth, env vars |
| **mark3labs** | advanced | 3 (greet, echo, time) | Okta | ConfigBuilder, env vars, logging |
| **official** | simple | 1 (greet) | Okta | Basic OAuth, env vars |
| **official** | advanced | 3 (greet, whoami, server_time) | Okta | ConfigBuilder, env vars, logging |

---

## Quick Start

### mark3labs SDK

**Simple:**
```bash
cd examples/mark3labs/simple
go run main.go
```

**Advanced:**
```bash
cd examples/mark3labs/advanced
go run main.go
```

### Official SDK

**Simple:**
```bash
cd examples/official/simple
go run main.go
```

**Advanced:**
```bash
cd examples/official/advanced
go run main.go
```

All examples start a server on `http://localhost:8080` with OAuth protection.

---

## Okta Setup

All examples use **Okta** as the OAuth provider. Before running, you need to set up Okta:

### 1. Create Okta Account

Sign up at https://developer.okta.com (free developer account)

### 2. Create API in Okta

1. Go to **Security > API** in Okta Admin Console
2. Click **Add Authorization Server** or use the default
3. Note your **Issuer URI** (e.g., `https://dev-12345.okta.com`)
4. Create an **Audience** identifier (e.g., `api://my-mcp-server`)

### 3. Set Environment Variables

```bash
export OKTA_DOMAIN="dev-12345.okta.com"        # Your Okta domain
export OKTA_AUDIENCE="api://my-mcp-server"     # Your API identifier
export SERVER_URL="http://localhost:8080"      # Your server URL
```

### 4. Get a Test Token

**Option A: Using Okta CLI**
```bash
# Install Okta CLI
brew install --cask oktacli  # macOS

# Login and get token
okta login
okta get token --audience api://my-mcp-server
```

**Option B: Using Okta Dashboard**
1. Go to **Security > API > Authorization Servers**
2. Click your authorization server
3. Go to **Token Preview** tab
4. Generate a token with your audience

### 5. Test the Server

```bash
# Save your Okta token
TOKEN="<your-okta-access-token>"

# Test with curl
curl -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -H "Accept: application/json, text/event-stream" \
     -X POST \
     http://localhost:8080 \
     -d '{
       "jsonrpc": "2.0",
       "method": "tools/list",
       "id": 1
     }'
```

---

## Configuration Options

### Environment Variables

All examples support these environment variables:

```bash
# Required Okta Configuration
export OKTA_DOMAIN="dev-12345.okta.com"         # Your Okta domain
export OKTA_AUDIENCE="api://my-mcp-server"      # Your API identifier

# Server Configuration
export SERVER_URL="http://localhost:8080"       # Your server URL
export PORT="8080"                              # Server port (default: 8080)
export MCP_HOST="localhost"                     # Server host (default: localhost)
export MCP_PORT="8080"                          # Server port for ConfigBuilder

# Optional: For HTTPS
export HTTPS_CERT_FILE="/path/to/cert.pem"      # If set, enables HTTPS
```

### Using Other Providers

To use Google or Azure AD instead of Okta, modify the config:

**Google:**
```go
&oauth.Config{
    Provider: "google",
    Issuer:   "https://accounts.google.com",
    Audience: "your-google-client-id.apps.googleusercontent.com",
}
```

**Azure AD:**
```go
&oauth.Config{
    Provider: "azure",
    Issuer:   "https://login.microsoftonline.com/YOUR-TENANT-ID/v2.0",
    Audience: "api://your-app-id",
}
```

---

## Example Comparison

### mark3labs/simple

**What it shows:**
- Basic OAuth integration with `mark3labs.WithOAuth()`
- Single tool with user context access
- Okta provider configuration
- Environment variable support

**Use when:** You want the simplest possible OAuth setup with mark3labs SDK.

### mark3labs/advanced

**What it shows:**
- `ConfigBuilder` for flexible configuration
- Environment variable support (Okta domain, audience, server URL)
- Multiple tools with different functionality
- Custom logging
- OAuth endpoint discovery logging
- Production-ready patterns

**Use when:** You need production-ready configuration with mark3labs SDK.

### official/simple

**What it shows:**
- Basic OAuth integration with `mcpoauth.WithOAuth()`
- Single tool with user context access
- Official SDK tool definition patterns
- Okta provider configuration
- Environment variable support

**Use when:** You want the simplest possible OAuth setup with official SDK.

### official/advanced

**What it shows:**
- `ConfigBuilder` for flexible configuration
- Multiple tools (greet, whoami, server_time)
- Environment variable support (Okta domain, audience)
- OAuth endpoint discovery logging
- Production-ready patterns
- Official SDK patterns

**Use when:** You need production-ready configuration with official SDK.

---

## Code Patterns Comparison

### mark3labs SDK

**Setup:**
```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{...})
mcpServer := mcpserver.NewMCPServer("name", "1.0.0", oauthOption)
```

**Adding Tools:**
```go
mcpServer.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    user, _ := oauth.GetUserFromContext(ctx)
    return mcp.NewToolResultText("Hello, " + user.Username), nil
})
```

### Official SDK

**Setup:**
```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

mcpServer := mcp.NewServer(&mcp.Implementation{...}, nil)
_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{...}, mcpServer)
http.ListenAndServe(":8080", handler)
```

**Adding Tools:**
```go
mcp.AddTool(mcpServer, &mcp.Tool{...},
    func(ctx context.Context, req *mcp.CallToolRequest, params *P) (*mcp.CallToolResult, any, error) {
        user, _ := oauth.GetUserFromContext(ctx)
        return &mcp.CallToolResult{
            Content: []mcp.Content{&mcp.TextContent{Text: "Hello, " + user.Username}},
        }, nil, nil
    })
```

**Key Difference**: mark3labs uses ServerOption before server creation, official SDK wraps the server with http.Handler after creation.

---

## Accessing User Information

All examples show how to access authenticated user information:

```go
user, ok := oauth.GetUserFromContext(ctx)
if !ok {
    return nil, fmt.Errorf("authentication required")
}

// Available fields:
user.Subject   // OAuth "sub" claim (user ID)
user.Username  // "preferred_username" or "sub"
user.Email     // "email" claim
```

---

## Common Issues

### "authentication required: missing OAuth token"

**Cause:** No Authorization header or invalid format.

**Solution:**
```bash
# Make sure to include Bearer token
curl -H "Authorization: Bearer YOUR_TOKEN" ...
```

### "authentication failed: token validation failed"

**Cause:** Invalid token or wrong secret.

**Solution:**
- For HMAC: Ensure `HMAC_SECRET` matches the secret used to sign the token
- For OIDC: Verify issuer, audience, and that the token is from the correct provider

### "Accept must contain both 'application/json' and 'text/event-stream'"

**Cause:** Missing Accept header (official SDK only).

**Solution:**
```bash
curl -H "Accept: application/json, text/event-stream" ...
```

---

## Building for Production

### Dockerfile Example

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./examples/advanced

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .

# Set production environment variables
ENV OAUTH_PROVIDER=okta
ENV SERVER_URL=https://your-server.com

CMD ["./server"]
```

### Production Checklist

- [ ] Use OIDC provider (Okta/Google/Azure), not HMAC
- [ ] Set `SERVER_URL` to your actual domain (HTTPS)
- [ ] Store secrets in environment variables or secret manager
- [ ] Enable HTTPS (use reverse proxy like nginx or Caddy)
- [ ] Configure proper CORS if needed
- [ ] Set up monitoring and logging
- [ ] Review OAuth scopes and permissions

---

## Further Reading

- **Migration Guide**: [../MIGRATION-V2.md](../MIGRATION-V2.md)
- **Main README**: [../README.md](../README.md)
- **Project Documentation**: [../CLAUDE.md](../CLAUDE.md)
- **Implementation Details**: [../docs/generic-implementation.md](../docs/generic-implementation.md)

---

## Need Help?

- **Issues**: https://github.com/tuannvm/oauth-mcp-proxy/issues
- **Discussions**: https://github.com/tuannvm/oauth-mcp-proxy/discussions
- **Documentation**: See files in `/docs` directory

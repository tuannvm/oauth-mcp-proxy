# Okta Provider Guide

> **📢 v2.0.0:** This guide shows examples for both `mark3labs/mcp-go` and official `modelcontextprotocol/go-sdk`.
> See [examples/README.md](../../examples/README.md) for complete Okta setup guide.

## Overview

Okta provider uses OIDC/JWKS for JWT validation. Ideal for enterprise SSO, user management, and production deployments.

## When to Use

✅ **Good for:**

- Enterprise SSO integration
- User authentication with existing Okta org
- Production applications
- Multi-tenant applications
- MFA requirements

---

## Setup in Okta

### 1. Create OAuth Application

1. Log in to Okta Admin Console
2. Navigate to **Applications** → **Applications**
3. Click **Create App Integration**
4. Select:
   - **Sign-in method:** OIDC - OpenID Connect
   - **Application type:** Web Application (for proxy mode) or Native Application (for native mode)
5. Click **Next**

### 2. Configure Application

**General Settings:**

- **App integration name:** Your MCP Server
- **Grant type:**
  - ✅ Authorization Code
  - ✅ Refresh Token (optional)

**Sign-in redirect URIs:**

- Native mode: Managed by client (e.g., Claude Desktop)
- Proxy mode: `https://your-mcp-server.com/oauth/callback`

**Sign-out redirect URIs:** (optional)

- Add if you support logout

**Controlled access:**

- Select who can use this application

**Save** the application.

### 3. Get Configuration Values

After saving, note these values:

- **Client ID:** Copy from the application page
- **Client Secret:** Copy from the Client Secrets section (proxy mode only)
- **Okta Domain:** Your Okta org URL (e.g., `https://yourcompany.okta.com`)

### 4. Configure Authorization Server

By default, Okta uses the org authorization server. For custom authorization server:

1. Navigate to **Security** → **API** → **Authorization Servers**
2. Use `default` or create custom
3. Note the **Issuer URI**

---

## Configuration (Native Mode)

**When:** Client handles OAuth (Claude Desktop, browser clients)

**mark3labs SDK:**
```go
import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://yourcompany.okta.com",
    Audience: "api://your-mcp-server",
})
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
```

**Official SDK:**
```go
import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"

_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://yourcompany.okta.com",
    Audience: "api://your-mcp-server",
}, mcpServer)
http.ListenAndServe(":8080", handler)
```

Client configures OAuth directly with Okta. Server only validates tokens.

---

## Configuration (Proxy Mode)

**When:** Client cannot do OAuth (simple CLI tools)

**mark3labs SDK:**
```go
import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider:     "okta",
    Issuer:       "https://yourcompany.okta.com",
    Audience:     "api://your-mcp-server",
    ClientID:     "0oa...",                           // From Okta app
    ClientSecret: "secret-from-okta",                 // From Okta app
    ServerURL:    "https://your-mcp-server.com",     // Your public URL
    RedirectURIs: "https://your-mcp-server.com/oauth/callback",
})
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
```

**Official SDK:**
```go
import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"

_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider:     "okta",
    Issuer:       "https://yourcompany.okta.com",
    Audience:     "api://your-mcp-server",
    ClientID:     "0oa...",                           // From Okta app
    ClientSecret: "secret-from-okta",                 // From Okta app
    ServerURL:    "https://your-mcp-server.com",     // Your public URL
    RedirectURIs: "https://your-mcp-server.com/oauth/callback",
}, mcpServer)
http.ListenAndServe(":8080", handler)
```

Server proxies OAuth flow. Client gets tokens from your server.

---

## Audience Configuration

Okta tokens include `aud` (audience) claim. Configure it:

### Option 1: Use Client ID as Audience

Simplest approach:

```go
// mark3labs or official SDK - same config
mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://yourcompany.okta.com",
    Audience: "0oa...",  // Same as ClientID
})
```

Okta tokens automatically include Client ID in `aud`.

### Option 2: Custom Audience

For custom audience (e.g., `api://my-server`):

1. In Okta, navigate to **Security** → **API** → **Authorization Servers**
2. Select your auth server → **Claims** tab
3. Add custom claim:
   - **Name:** `aud`
   - **Include in:** ID Token, Always
   - **Value type:** Expression
   - **Value:** `"api://my-server"`

Then configure:

```go
// mark3labs or official SDK - same config
mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://yourcompany.okta.com",
    Audience: "api://my-server",  // Your custom audience
})
```

---

## Testing

### 1. Start Your MCP Server

```bash
go run main.go
```

### 2. Test OAuth Flow (Proxy Mode)

```bash
# Get OAuth metadata
curl https://your-server.com/.well-known/oauth-authorization-server

# Follow authorization flow in browser
open "https://your-server.com/oauth/authorize?client_id=...&redirect_uri=...&response_type=code&code_challenge=..."
```

### 3. Verify Token Validation (Native Mode)

Get token from Okta (using client), then test:

```bash
curl -X POST https://your-server.com/mcp \
  -H "Authorization: Bearer <okta-token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

---

## Scopes

Okta tokens include scopes. Recommended scopes for MCP:

- `openid` - Required for OIDC
- `profile` - User profile information
- `email` - User email address

These are automatically requested when using proxy mode.

---

## Troubleshooting

### "Failed to initialize OIDC provider"

- Check: Issuer URL is correct (no trailing slash)
- Check: Server can reach Okta (network/firewall)
- Check: Issuer serves `.well-known/openid-configuration`

### "Invalid audience"

- Check: Token `aud` claim matches `Config.Audience`
- Check: Okta app/auth server configured to include correct audience

### "Token verification failed"

- Check: Token not expired
- Check: Token signed by Okta (check `iss` claim)
- Check: Issuer URL matches exactly

---

## Production Checklist

- [ ] Use HTTPS for all endpoints
- [ ] Store ClientSecret in environment variables
- [ ] Configure appropriate token expiration in Okta
- [ ] Enable MFA in Okta for user accounts
- [ ] Set up Okta rate limiting
- [ ] Monitor Okta auth logs
- [ ] Configure CORS if needed for browser clients

---

## References

- [Okta Developer Docs](https://developer.okta.com/docs/)
- [OIDC Overview](https://developer.okta.com/docs/concepts/oauth-openid/)
- [Create Web App](https://developer.okta.com/docs/guides/sign-into-web-app/go/main/)

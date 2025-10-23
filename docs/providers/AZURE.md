> **📢 v2.0.0:** This guide shows examples for both `mark3labs/mcp-go` and official `modelcontextprotocol/go-sdk`.
> See [examples/README.md](../../examples/README.md) for complete setup guide.

# Azure AD Provider Guide

## Overview

Azure AD (Microsoft Entra ID) provider uses OIDC/JWKS for JWT validation. Ideal for Microsoft 365 integration and enterprise authentication.

## When to Use

✅ **Good for:**

- Microsoft 365 / Azure integration
- Enterprise SSO with Azure AD
- Applications for corporate Microsoft users
- Multi-tenant SaaS applications

---

## Setup in Azure Portal

### 1. Register Application

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to **Microsoft Entra ID** (formerly Azure Active Directory)
3. Select **App registrations** → **New registration**
4. Configure:
   - **Name:** Your MCP Server
   - **Supported account types:**
     - Single tenant (your org only)
     - Multi-tenant (any Azure AD)
     - Multi-tenant + personal Microsoft accounts
   - **Redirect URI:** (for proxy mode)
     - Type: Web
     - URI: `https://your-server.com/oauth/callback`
5. Click **Register**

### 2. Get Application (client) ID

After registration, copy:

- **Application (client) ID** - This is your Client ID
- **Directory (tenant) ID** - Used in issuer URL

### 3. Create Client Secret (Proxy Mode Only)

1. In your app, go to **Certificates & secrets**
2. Click **New client secret**
3. Add description: "MCP Server OAuth"
4. Choose expiration (recommend: 6-12 months)
5. Click **Add**
6. **Copy the secret value immediately** (shown only once!)

### 4. Configure API Permissions

1. Go to **API permissions**
2. Click **Add a permission**
3. Select **Microsoft Graph**
4. Choose **Delegated permissions**
5. Add permissions:
   - `openid` (sign users in)
   - `profile` (user profile)
   - `email` (user email)
6. Click **Grant admin consent** (if you're admin)

### 5. Configure Token Claims (Optional)

For custom audience claim:

1. Go to **Token configuration**
2. Click **Add optional claim**
3. Select **ID** token type
4. Add claims as needed

---

## Configuration (Native Mode)

**When:** Client handles OAuth with Azure AD directly

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "azure",
    Issuer:   "https://login.microsoftonline.com/{tenant-id}/v2.0",
    Audience: "api://your-app-id",  // Or Application ID
})
```

Replace `{tenant-id}` with:

- Your Directory (tenant) ID, OR
- `common` for multi-tenant apps
- `organizations` for any Azure AD user
- `consumers` for personal Microsoft accounts only

---

## Configuration (Proxy Mode)

**When:** Server proxies OAuth flow

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:     "azure",
    Issuer:       "https://login.microsoftonline.com/{tenant-id}/v2.0",
    Audience:     "api://your-app-id",
    ClientID:     "12345678-1234-1234-1234-123456789012",  // Application ID
    ClientSecret: "secret~from~azure",                      // Client secret
    ServerURL:    "https://your-server.com",
    RedirectURIs: "https://your-server.com/oauth/callback",
})
```

---

## Audience Options

Azure AD is flexible with audience:

### Option 1: Application ID (Simplest)

```go
Audience: "12345678-1234-1234-1234-123456789012"  // Your Application ID
```

Azure tokens automatically include Application ID in `aud` claim.

### Option 2: Custom App ID URI

1. In Azure portal, go to **App registrations** → Your app
2. Navigate to **Expose an API**
3. Set **Application ID URI:** `api://your-server`
4. Click **Save**

Then configure:

```go
Audience: "api://your-server"  // Matches Application ID URI
```

---

## Testing

### 1. Environment Setup

```bash
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-app-id"
export AZURE_CLIENT_SECRET="your-secret"

# Build issuer URL
export AZURE_ISSUER="https://login.microsoftonline.com/${AZURE_TENANT_ID}/v2.0"
```

### 2. Start Server

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:     "azure",
    Issuer:       os.Getenv("AZURE_ISSUER"),
    Audience:     os.Getenv("AZURE_CLIENT_ID"),
    ClientID:     os.Getenv("AZURE_CLIENT_ID"),
    ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
    ServerURL:    "https://your-server.com",
    RedirectURIs: "https://your-server.com/oauth/callback",
})
```

### 3. Test Authentication

```bash
# Test OAuth flow
curl https://your-server.com/.well-known/oauth-authorization-server

# Test with token
curl -X POST https://your-server.com/mcp \
  -H "Authorization: Bearer <azure-token>" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

---

## User Claims

Azure AD ID tokens include:

```json
{
  "sub": "AAAAAAAAAAAAAAAAAAAAAIkzqFVrSaSaFHy782bbtaQ",
  "name": "John Doe",
  "email": "john.doe@company.com",
  "preferred_username": "john.doe@company.com",
  "aud": "api://your-server",
  "iss": "https://login.microsoftonline.com/{tenant}/v2.0",
  "exp": 1234567890,
  "iat": 1234567890,
  "tid": "tenant-id"
}
```

oauth-mcp-proxy extracts:

- `sub` → User.Subject
- `email` → User.Email
- `preferred_username` or `email` → User.Username

---

## Multi-Tenant Applications

For SaaS applications serving multiple Azure AD tenants:

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "azure",
    Issuer:   "https://login.microsoftonline.com/common/v2.0",  // Note: "common"
    Audience: "api://your-server",
})
```

Validates tokens from any Azure AD tenant. Extract tenant from `tid` claim if needed.

---

## Troubleshooting

### "Failed to initialize OIDC provider"

- Check: Issuer URL format correct (ends with `/v2.0`)
- Check: Tenant ID is correct
- Check: Network can reach `login.microsoftonline.com`

### "Invalid audience"

- Check: `Config.Audience` matches token's `aud` claim
- Check: Application ID URI configured in Azure if using custom audience

### "AADSTS errors" from Azure

- `AADSTS50011`: Redirect URI mismatch - check Azure portal configuration
- `AADSTS700016`: Application not found - check Client ID
- `AADSTS7000215`: Invalid client secret - regenerate secret

---

## Production Checklist

- [ ] Use HTTPS for all endpoints
- [ ] Store ClientSecret in Azure Key Vault or environment
- [ ] Configure appropriate token lifetimes in Azure AD
- [ ] Enable Conditional Access policies
- [ ] Set up Azure AD monitoring and alerts
- [ ] Configure API permissions with least privilege
- [ ] Test token expiration and refresh flows
- [ ] Document tenant onboarding for multi-tenant apps

---

## References

- [Microsoft Identity Platform](https://learn.microsoft.com/en-us/entra/identity-platform/)
- [Register an Application](https://learn.microsoft.com/en-us/entra/identity-platform/quickstart-register-app)
- [ID Tokens](https://learn.microsoft.com/en-us/entra/identity-platform/id-tokens)
- [OAuth 2.0 and OpenID Connect](https://learn.microsoft.com/en-us/entra/identity-platform/v2-protocols-oidc)

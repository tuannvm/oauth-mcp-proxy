> **📢 v2.0.0:** This guide shows examples for both `mark3labs/mcp-go` and official `modelcontextprotocol/go-sdk`.
> See [examples/README.md](../../examples/README.md) for complete setup guide.

# Google Provider Guide

## Overview

Google provider uses OIDC/JWKS for JWT validation with Google's identity platform. Ideal for Google Workspace integration.

## When to Use

✅ **Good for:**

- Google Workspace integration
- Consumer applications with Google Sign-In
- Applications requiring Google account authentication
- Cross-platform user auth (Android, iOS, Web)

---

## Setup in Google Cloud Console

### 1. Create OAuth Client

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Select your project (or create new)
3. Navigate to **APIs & Services** → **Credentials**
4. Click **Create Credentials** → **OAuth client ID**
5. Configure OAuth consent screen if prompted (see below)
6. Select application type:
   - **Web application** (for proxy mode)
   - **Desktop app** or **iOS/Android** (for native mode)

### 2. Configure OAuth Consent Screen

Required before creating OAuth client:

1. Navigate to **APIs & Services** → **OAuth consent screen**
2. Choose **User Type:**
   - **Internal** - Google Workspace users only
   - **External** - Anyone with Google account
3. Fill in:
   - **App name:** Your MCP Server
   - **User support email:** Your email
   - **Developer contact:** Your email
4. Add scopes:
   - `openid`
   - `profile`
   - `email`
5. Save and Continue

### 3. Create OAuth Client ID

**For Web Application (Proxy Mode):**

- **Authorized JavaScript origins:** `https://your-server.com`
- **Authorized redirect URIs:** `https://your-server.com/oauth/callback`

**For Desktop App (Native Mode):**

- No redirect URIs needed (client handles it)

### 4. Get Configuration Values

After creation, note:

- **Client ID:** `<id>.apps.googleusercontent.com`
- **Client Secret:** (for proxy mode only)
- **Issuer:** Always `https://accounts.google.com`

---

## Configuration (Native Mode)

**When:** Client handles OAuth (Claude Desktop, mobile apps)

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "google",
    Issuer:   "https://accounts.google.com",
    Audience: "123456789.apps.googleusercontent.com",  // Your Client ID
})
```

**Important:** For Google, `Audience` must be your Client ID, not a custom value.

---

## Configuration (Proxy Mode)

**When:** Server proxies OAuth for simple clients

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:     "google",
    Issuer:       "https://accounts.google.com",
    Audience:     "123456789.apps.googleusercontent.com",  // Your Client ID
    ClientID:     "123456789.apps.googleusercontent.com",
    ClientSecret: "GOCSPX-...",                           // From Google Console
    ServerURL:    "https://your-server.com",
    RedirectURIs: "https://your-server.com/oauth/callback",
})
```

---

## Testing

### 1. Start MCP Server

```bash
export GOOGLE_CLIENT_ID="123456789.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="GOCSPX-..."
go run main.go
```

### 2. Test OAuth Flow (Browser)

```bash
# Get authorization URL
curl https://your-server.com/.well-known/oauth-authorization-server

# Open in browser to authenticate
open "https://your-server.com/oauth/authorize?..."
```

### 3. Test Token Validation

Get token from Google Sign-In, then:

```bash
curl -X POST https://your-server.com/mcp \
  -H "Authorization: Bearer <google-id-token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

---

## User Claims

Google ID tokens include:

```json
{
  "sub": "1234567890",
  "email": "user@gmail.com",
  "email_verified": true,
  "name": "John Doe",
  "picture": "https://...",
  "aud": "your-client-id.apps.googleusercontent.com",
  "iss": "https://accounts.google.com",
  "exp": 1234567890,
  "iat": 1234567890
}
```

oauth-mcp-proxy extracts:

- `sub` → User.Subject
- `email` → User.Email
- `name` or `email` → User.Username

---

## Troubleshooting

### "Failed to initialize OIDC provider"

- Check: Can reach `https://accounts.google.com/.well-known/openid-configuration`
- Check: No typo in issuer URL (must be exact)

### "Invalid audience"

- Google uses Client ID as audience
- Check: `Config.Audience` matches your Client ID exactly
- Example: `123456789.apps.googleusercontent.com`

### "redirect_uri_mismatch" error

- Check: Redirect URI in Google Console matches `Config.RedirectURIs`
- Must be exact match (including https://)
- No localhost in production

### "invalid_client" error

- Check: ClientID and ClientSecret correct
- Check: Client type matches mode (Web app for proxy mode)

---

## Production Checklist

- [ ] Use HTTPS for all endpoints
- [ ] Store ClientSecret in environment variables
- [ ] Configure OAuth consent screen properly
- [ ] Set appropriate token expiration
- [ ] Verify email domain restrictions if needed
- [ ] Enable Google Account security features
- [ ] Monitor Google API quotas

---

## References

- [Google Identity Platform](https://developers.google.com/identity)
- [OAuth 2.0 for Web Apps](https://developers.google.com/identity/protocols/oauth2/web-server)
- [ID Token Validation](https://developers.google.com/identity/protocols/oauth2/openid-connect#validatinganidtoken)

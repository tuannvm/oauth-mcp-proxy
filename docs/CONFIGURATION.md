# Configuration Guide

Complete reference for oauth-mcp-proxy configuration options.

---

## Config Struct

```go
type Config struct {
    // Required
    Provider string // "hmac", "okta", "google", "azure"
    Audience string // Your API audience

    // Provider-specific
    Issuer    string // OIDC issuer URL (Okta/Google/Azure)
    JWTSecret []byte // Secret key (HMAC only)

    // Optional - OAuth Mode
    Mode string // "native" or "proxy" - auto-detected

    // Optional - Proxy Mode
    ClientID     string // OAuth client ID
    ClientSecret string // OAuth client secret
    ServerURL    string // Your server's public URL
    RedirectURIs string // Allowed redirect URIs

    // Optional - Logging
    Logger Logger // Custom logger implementation
}
```

---

## Required Fields

### Provider

**Type:** `string`
**Required:** Yes
**Values:** `"hmac"`, `"okta"`, `"google"`, `"azure"`

Specifies which OAuth provider to use for token validation.

```go
Provider: "okta"  // Use Okta OIDC validation
```

**See:** [Provider Guides](providers/) for setup instructions

### Audience

**Type:** `string`
**Required:** Yes
**Purpose:** Validates the `aud` claim in JWT tokens

The audience must match exactly. This prevents token reuse across services.

**Examples:**
```go
// Custom audience
Audience: "api://my-mcp-server"

// Google (use Client ID)
Audience: "123456789.apps.googleusercontent.com"

// Azure (use Application ID or App ID URI)
Audience: "api://my-server"
// or
Audience: "12345678-1234-1234-1234-123456789012"
```

---

## Provider-Specific Fields

### Issuer

**Type:** `string`
**Required:** For OIDC providers (okta, google, azure)
**Not used:** HMAC provider

The OAuth provider's issuer URL. Must match token's `iss` claim exactly.

**Examples:**
```go
// Okta
Issuer: "https://yourcompany.okta.com"

// Google
Issuer: "https://accounts.google.com"

// Azure AD (single tenant)
Issuer: "https://login.microsoftonline.com/{tenant-id}/v2.0"

// Azure AD (multi-tenant)
Issuer: "https://login.microsoftonline.com/common/v2.0"
```

**Important:**
- No trailing slash
- Must serve `/.well-known/openid-configuration`
- HTTPS required

### JWTSecret

**Type:** `[]byte`
**Required:** For HMAC provider only
**Not used:** OIDC providers

Shared secret for HMAC-SHA256 token validation.

**Examples:**
```go
// From environment (recommended)
JWTSecret: []byte(os.Getenv("JWT_SECRET"))

// Minimum 32 bytes recommended
JWTSecret: []byte("your-very-long-secret-key-min-32-bytes")

// Generate securely
secret := make([]byte, 32)
rand.Read(secret)
JWTSecret: secret
```

**Security:** Never hardcode! Use environment variables. See [SECURITY.md](SECURITY.md).

---

## OAuth Mode

### Mode

**Type:** `string`
**Optional:** Auto-detected if not specified
**Values:** `"native"`, `"proxy"`

Determines whether client or server handles OAuth flow.

**Auto-detection:**
```go
// If ClientID is provided → proxy mode
// If ClientID is empty → native mode
Mode: ""  // Let library auto-detect
```

**Explicit:**
```go
Mode: "native"  // Client does OAuth
Mode: "proxy"   // Server proxies OAuth
```

### Native Mode

**When:** OAuth-capable clients (Claude Desktop, browser apps)

**Client:** Authenticates directly with provider → Gets token → Calls MCP server
**Server:** Only validates tokens (no OAuth endpoints used)

**Config:**
```go
oauth.WithOAuth(mux, &oauth.Config{
    Mode:     "native",  // Or omit (auto-detected)
    Provider: "okta",
    Issuer:   "https://company.okta.com",
    Audience: "api://my-server",
    // No ClientID/ServerURL/RedirectURIs needed
})
```

**OAuth endpoints:** Return 404 with helpful message (not needed by client)

### Proxy Mode

**When:** Simple clients that can't do OAuth (CLI tools, legacy clients)

**Client:** Calls MCP server → Server proxies to provider → Returns token to client
**Server:** Full OAuth authorization server functionality

**Config:**
```go
oauth.WithOAuth(mux, &oauth.Config{
    Mode:         "proxy",  // Or omit (auto-detected from ClientID)
    Provider:     "okta",
    Issuer:       "https://company.okta.com",
    Audience:     "api://my-server",
    ClientID:     "your-client-id",           // Required for proxy mode
    ClientSecret: "your-client-secret",       // Required for proxy mode
    ServerURL:    "https://your-server.com",  // Required for proxy mode
    RedirectURIs: "https://your-server.com/oauth/callback",  // Required
})
```

**OAuth endpoints:** Fully functional (`/oauth/authorize`, `/oauth/callback`, `/oauth/token`)

**Mode Comparison:**

| | Native | Proxy |
|---|---|---|
| **Client capability** | Can do OAuth | Cannot do OAuth |
| **OAuth flow** | Client ↔ Provider | Client ↔ Server ↔ Provider |
| **Config required** | Minimal | Full (ClientID, ServerURL, etc.) |
| **Endpoints active** | Metadata only | All endpoints |
| **Use case** | Production apps | Simple clients |

---

## Proxy Mode Fields

### ClientID

**Type:** `string`
**Required:** For proxy mode
**Purpose:** OAuth client identifier from provider

Obtained from your OAuth provider:
- Okta: Application → General → Client ID
- Google: Cloud Console → Credentials → OAuth 2.0 Client ID
- Azure: App registrations → Application (client) ID

```go
ClientID: "0oa..."  // Okta
ClientID: "123.apps.googleusercontent.com"  // Google
ClientID: "12345678-1234-1234-1234-123456789012"  // Azure
```

### ClientSecret

**Type:** `string`
**Required:** For proxy mode (confidential clients)
**Purpose:** OAuth client secret for token exchange

**Security:**
```go
// ✅ From environment
ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET")

// ❌ Never hardcode
ClientSecret: "abc123..."  // SECURITY VIOLATION
```

**See:** [SECURITY.md](SECURITY.md) for secret management best practices.

### ServerURL

**Type:** `string`
**Required:** For proxy mode
**Purpose:** Your MCP server's public URL

Used for:
- OAuth metadata endpoints (issuer URL)
- Redirect URI construction
- Endpoint URL generation

```go
ServerURL: "https://mcp-server.example.com"      // Production
ServerURL: "https://mcp-server.herokuapp.com"     // Cloud deployment
ServerURL: "http://localhost:8080"                // Local testing
```

**Requirements:**
- HTTPS in production
- No trailing slash
- Publicly accessible (for OAuth provider callbacks)

### RedirectURIs

**Type:** `string`
**Required:** For proxy mode
**Purpose:** OAuth redirect URI validation

**Single URI (Fixed Redirect):**
```go
RedirectURIs: "https://your-server.com/oauth/callback"
```

Server uses this URI with provider. For security, client redirects must be localhost only.

**Multiple URIs (Allowlist):**
```go
RedirectURIs: "https://app1.com/callback,https://app2.com/callback,https://app3.com/callback"
```

Comma-separated list. Client's redirect_uri must exactly match one of these.

**Security:**
- HTTPS required for non-localhost
- No wildcards allowed
- Exact string match
- See [SECURITY.md](SECURITY.md) for redirect URI security

---

## Optional Fields

### Logger

**Type:** `Logger` interface
**Default:** Uses `log.Printf` with level prefixes
**Purpose:** Custom logging integration

Implement Logger interface to integrate with your logging system:

```go
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}
```

**Examples:**

**Zap:**
```go
type ZapLogger struct{ logger *zap.Logger }

func (l *ZapLogger) Info(msg string, args ...interface{}) {
    l.logger.Sugar().Infof(msg, args...)
}
// ... implement Debug, Warn, Error

cfg := &oauth.Config{
    Provider: "okta",
    Logger:   &ZapLogger{logger: zapLogger},
}
```

**Logrus:**
```go
type LogrusLogger struct{ logger *logrus.Logger }

func (l *LogrusLogger) Info(msg string, args ...interface{}) {
    l.logger.Infof(msg, args...)
}
// ... implement Debug, Warn, Error

cfg := &oauth.Config{
    Logger: &LogrusLogger{logger: logrusLogger},
}
```

**Default behavior:**
```
[INFO] OAuth2: Authorization request - client_id: ...
[WARN] SECURITY: Invalid redirect URI ...
[ERROR] OAuth2: Token validation failed: ...
```

**What gets logged:** See [examples/README.md](../examples/README.md#custom-logging)

---

## Validation

Configuration is validated when calling `WithOAuth()` or `NewServer()`:

```go
oauthOption, err := oauth.WithOAuth(mux, cfg)
if err != nil {
    // err describes what's wrong:
    // - "provider is required"
    // - "JWTSecret is required for HMAC provider"
    // - "proxy mode requires ClientID"
    // - etc.
    log.Fatal(err)
}
```

### Validation Rules

**All modes:**
- Provider must be one of: hmac, okta, google, azure
- Audience is required
- Provider-specific fields validated (JWTSecret for HMAC, Issuer for OIDC)

**Proxy mode:**
- ClientID required
- ServerURL required
- RedirectURIs required

**Native mode:**
- ClientID, ServerURL, RedirectURIs optional (ignored if provided)

---

## Complete Examples

### HMAC (Testing)

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-server",
    JWTSecret: []byte(os.Getenv("JWT_SECRET")),
})
```

### Okta (Native - Recommended)

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   os.Getenv("OKTA_ISSUER"),
    Audience: "api://my-server",
})
```

### Okta (Proxy - For Simple Clients)

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:     "okta",
    Issuer:       os.Getenv("OKTA_ISSUER"),
    Audience:     "api://my-server",
    ClientID:     os.Getenv("OKTA_CLIENT_ID"),
    ClientSecret: os.Getenv("OKTA_CLIENT_SECRET"),
    ServerURL:    "https://mcp.example.com",
    RedirectURIs: "https://mcp.example.com/oauth/callback",
})
```

### Google

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "google",
    Issuer:   "https://accounts.google.com",
    Audience: os.Getenv("GOOGLE_CLIENT_ID"),  // Use Client ID as audience
})
```

### Azure AD

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "azure",
    Issuer:   fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0",
                          os.Getenv("AZURE_TENANT_ID")),
    Audience: os.Getenv("AZURE_CLIENT_ID"),
})
```

---

## Environment Variables Pattern

Recommended `.env` structure:

```bash
# OAuth Provider
OAUTH_PROVIDER=okta
OAUTH_ISSUER=https://yourcompany.okta.com
OAUTH_AUDIENCE=api://my-mcp-server

# HMAC (if using)
JWT_SECRET=your-32-byte-secret-key

# Proxy Mode (if using)
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret
OAUTH_SERVER_URL=https://your-server.com
OAUTH_REDIRECT_URIS=https://your-server.com/oauth/callback
```

Load in code:

```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load()

    oauth.WithOAuth(mux, &oauth.Config{
        Provider:     os.Getenv("OAUTH_PROVIDER"),
        Issuer:       os.Getenv("OAUTH_ISSUER"),
        Audience:     os.Getenv("OAUTH_AUDIENCE"),
        ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
        ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
        ServerURL:    os.Getenv("OAUTH_SERVER_URL"),
        RedirectURIs: os.Getenv("OAUTH_REDIRECT_URIS"),
        JWTSecret:    []byte(os.Getenv("JWT_SECRET")),
    })
}
```

---

## See Also

- [Provider Guides](providers/) - Provider-specific setup
- [SECURITY.md](SECURITY.md) - Security best practices
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common configuration issues

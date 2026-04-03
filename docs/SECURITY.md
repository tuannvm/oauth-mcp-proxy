# Security Best Practices

This guide outlines security best practices when using oauth-mcp-proxy in production.

---

## 🔒 Secrets Management

### Never Commit Secrets

**❌ BAD:**

```go
oauth.WithOAuth(mux, &oauth.Config{
    JWTSecret: []byte("my-secret-key"),  // Committed to git!
    ClientSecret: "hardcoded-secret",     // Committed to git!
})
```

**✅ GOOD:**

```go
oauth.WithOAuth(mux, &oauth.Config{
    JWTSecret:    []byte(os.Getenv("JWT_SECRET")),
    ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
})
```

### Environment Variables

```bash
# .env (add to .gitignore!)
JWT_SECRET=your-random-32-byte-secret-key-here
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret
OAUTH_ISSUER=https://yourcompany.okta.com
```

Load with library like `godotenv`:

```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load() // Load .env file

    oauth.WithOAuth(mux, &oauth.Config{
        Provider:     os.Getenv("OAUTH_PROVIDER"),
        Issuer:       os.Getenv("OAUTH_ISSUER"),
        JWTSecret:    []byte(os.Getenv("JWT_SECRET")),
        ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
    })
}
```

### .gitignore

```gitignore
# Secrets
.env
.env.local
.env.production

# Certificates
*.pem
*.key
*.crt

# OAuth tokens (testing)
*.token
```

---

## 🔐 JWT Secret Strength (HMAC Provider)

### Minimum Requirements

```go
// Generate cryptographically secure secret
secret := make([]byte, 32)  // 32 bytes = 256 bits
if _, err := rand.Read(secret); err != nil {
    log.Fatal(err)
}

// Store as base64 or hex
secretB64 := base64.StdEncoding.EncodeToString(secret)
fmt.Println("JWT_SECRET=" + secretB64)
```

### Validation

```go
secret := []byte(os.Getenv("JWT_SECRET"))
if len(secret) < 32 {
    log.Fatal("JWT_SECRET must be at least 32 bytes for security")
}
```

### Rotation

- **Rotate every:** 90 days recommended
- **Process:** Generate new secret → Update config → Deploy → Update token generators
- **Zero downtime:** Temporarily accept both old and new secrets during rotation

---

## 🌐 HTTPS in Production

### Always Use TLS

**❌ NEVER in production:**

```go
http.ListenAndServe(":80", mux)  // Unencrypted!
```

**✅ Production:**

```go
http.ListenAndServeTLS(":443", "server.crt", "server.key", mux)
```

### Get Certificates

**Development:**

- Use [mkcert](https://github.com/FiloSottile/mkcert) for local testing

**Production:**

- Use [Let's Encrypt](https://letsencrypt.org/) with [certbot](https://certbot.eff.org/)
- Or your cloud provider's certificate service (AWS ACM, GCP Certificate Manager)

### Certificate Management

```go
// Auto-reload certificates
certManager := &autocert.Manager{
    Prompt: autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("your-server.com"),
    Cache: autocert.DirCache("certs"),
}

server := &http.Server{
    Addr:      ":443",
    Handler:   mux,
    TLSConfig: certManager.TLSConfig(),
}

server.ListenAndServeTLS("", "")
```

---

## 🎯 Audience Validation

### Why Audience Matters

Prevents token reuse across services:

```
Service A: Audience = "api://service-a"
Service B: Audience = "api://service-b"
```

Token for Service A cannot be used on Service B (even with same issuer).

### Configuration

**HMAC Provider:**

```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "hmac",
    Audience: "api://my-specific-mcp-server",  // Unique per service
})
```

**OIDC Providers:**

- Okta: Configure custom audience in auth server claims
- Google: Use Client ID as audience
- Azure: Use Application ID or custom App ID URI

### Validation

```go
// Token must have matching audience
{
  "aud": "api://my-specific-mcp-server",  // Must match Config.Audience
  "iss": "https://issuer.com",
  "sub": "user-123"
}
```

---

## 🔄 Token Caching & Expiration

### Cache Behavior

- **Cache TTL:** 5 minutes (hardcoded in v0.1.0)
- **Cache scope:** Per Server instance
- **Cache key:** SHA-256 hash of token

### Token Expiration Recommendations

**User tokens:**

- Short-lived: 1 hour
- Refresh tokens: 7-30 days
- Reason: Limits damage if compromised

**Service tokens:**

- Medium-lived: 6-24 hours
- Reason: Balance between security and token refresh overhead

```go
// When generating tokens
token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "sub": "user-123",
    "aud": "api://my-server",
    "exp": time.Now().Add(1 * time.Hour).Unix(),  // Expire in 1 hour
    "iat": time.Now().Unix(),
})
```

---

## 🛡️ PKCE (Proof Key for Code Exchange)

### Automatic Protection

oauth-mcp-proxy automatically supports PKCE (RFC 7636):

- Prevents authorization code interception attacks
- Required for public clients (mobile, desktop, browser)
- Automatically validated when code_challenge provided

### No Configuration Needed

PKCE is automatically enabled when client provides:

- `code_challenge` parameter in /oauth/authorize
- `code_verifier` parameter in /oauth/token

---

## 🚪 Redirect URI Security

### Native Mode (Client OAuth)

**Localhost only for security:**

```
✅ http://localhost:8080/callback
✅ http://127.0.0.1:3000/callback
✅ http://[::1]:9000/callback
❌ http://evil.com/callback         (rejected)
❌ https://localhost.evil.com/...   (rejected - subdomain attack)
```

### Proxy Mode (Server OAuth)

**Allowlist configuration:**

```go
oauth.WithOAuth(mux, &oauth.Config{
    RedirectURIs: "https://app.example.com/callback",  // Single URI (fixed)
    // Or multiple:
    // RedirectURIs: "https://app1.com/cb,https://app2.com/cb",  // Allowlist
})
```

**Security checks:**

- HTTPS required for non-localhost
- No fragment allowed (per OAuth 2.0 spec)
- Exact match validation (no wildcards)

---

## 🎫 Token Security

### Token Storage (Client Side)

**Browser:**

- Use `httpOnly` cookies or sessionStorage (NOT localStorage)
- Clear on logout

**Mobile/Desktop:**

- Use OS keychain (macOS Keychain, Windows Credential Manager)
- Never store in plain text files

**CLI Tools:**

- Store in encrypted config files
- Use OS-specific secure storage when possible

### Token Transmission

**Always use Authorization header:**

```bash
curl -H "Authorization: Bearer <token>" https://server.com/mcp
```

**Never:**

- In URL query parameters (logged in web servers)
- In cookies without httpOnly flag
- In localStorage (XSS vulnerable)

---

## 🔍 Logging & Monitoring

### What Gets Logged

oauth-mcp-proxy logs (with custom logger or default):

**Info Level:**

- Authorization requests
- Successful authentications
- Token cache hits

**Warn Level:**

- Security violations (invalid redirects)
- Configuration issues

**Error Level:**

- Token validation failures
- OAuth provider errors

### What NOT to Log

✅ **Safe:** Token hash (SHA-256)

```
INFO: Validating token (hash: a7bc40a987f35871...)
```

❌ **NEVER log:** Full tokens

```
ERROR: Token xyz123... invalid  // SECURITY VIOLATION!
```

### Custom Logger for Production

```go
type ProductionLogger struct {
    logger *zap.Logger
}

func (l *ProductionLogger) Error(msg string, args ...interface{}) {
    // Sanitize before logging
    l.logger.Sugar().Errorf(msg, args...)
    // Send to error tracking (Sentry, etc.)
}

oauth.WithOAuth(mux, &oauth.Config{
    Logger: &ProductionLogger{logger: zapLogger},
})
```

---

## 🚨 Rate Limiting

### Built-in Rate Limiter

oauth-mcp-proxy includes a built-in rate limiter:

```go
import "github.com/tuannvm/oauth-mcp-proxy"

limiter := oauth.NewRateLimiter(time.Minute, 100) // 100 req/min
if !limiter.Allow("client-key") {
    http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
    return
}
```

**Features:**
- Fixed-window rate limiting
- Automatic cleanup of expired entries
- Thread-safe (uses sync.RWMutex)
- Background cleanup goroutine support

```go
// Start background cleanup (prevents memory leaks)
stopCleanup := limiter.StartCleanup(5 * time.Minute)
defer stopCleanup()
```

### Additional Protection

For OAuth endpoints, consider additional rate limiting:

```go
import "golang.org/x/time/rate"

globalLimiter := rate.NewLimiter(10, 20)  // 10 req/s, burst 20

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !globalLimiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply to OAuth endpoints
mux.Handle("/oauth/", rateLimitMiddleware(oauthHandler))
```

---

## 🔁 Security Headers

OAuth handler automatically adds security headers:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Cache-Control: no-store, no-cache, max-age=0
Pragma: no-cache
Content-Security-Policy: default-src 'none'; script-src 'none'; style-src 'none'; img-src 'none'; font-src 'none'; connect-src 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self';
```

---

## 🛡️ Built-in Security Features

oauth-mcp-proxy includes multiple security defenses:

### State Replay Protection

OAuth state parameters are protected against replay attacks:

- **Timestamp validation** - States expire after 10 minutes
- **Nonce uniqueness** - Each state uses a cryptographically random nonce
- **Replay detection** - Nonce tracked and rejected if reused
- **Automatic cleanup** - Expired nonces removed to prevent memory leaks
- **Rolling deploy compatible** - Accepts states from older versions during upgrades

### Token Cache Security

Token caching respects JWT expiration times:

```go
// Cache uses min(token.expiry, now + 5 minutes)
// This prevents cached tokens from being used past actual expiration
```

### Input Validation

Request parameters are validated to prevent abuse:

- **code parameter** - Max 512 characters
- **state parameter** - Max 256 characters  
- **code_challenge parameter** - Max 256 characters
- **Request body size** - Limited to prevent DoS (1MB for /oauth/token, 256KB for /oauth/register)

### Issuer URL Validation

OIDC provider issuer URLs are validated:

- **HTTPS required** for non-localhost URLs (prevents MITM attacks)
- **Valid URL format** - Must parse correctly
- **Not empty** - Issuer must be specified
- **No raw IP addresses** - Hostnames only (prevents misconfiguration)

### Constant-Time Cryptography

HMAC signatures verified using constant-time comparison:

```go
// Prevents timing attacks on signature validation
hmac.Equal([]byte(receivedSig), []byte(expectedSig))
```

### Secure Random Number Generation

Nonces generated using crypto/rand:

- **Panics on failure** - No fallback to weak timestamp-based nonces
- **Cryptographically secure** - Uses system CSPRNG

### Session Management (Official SDK)

The official SDK adapter populates the go-sdk auth context:

- **auth.TokenInfo populated** - User ID and expiration set for session binding
- **Session hijacking prevention** - Requests from different users rejected
- **CORS support** - OPTIONS requests pass through for browser clients

Add application-level headers:

```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
}

http.ListenAndServeTLS(":443", "cert.pem", "key.pem", securityHeaders(mux))
```

---

## 📋 Security Checklist

### Pre-Production

**Configuration:**
- [ ] All secrets in environment variables (not code)
- [ ] HTTPS enabled with valid certificates
- [ ] Audience configured and validated
- [ ] JWT secret 32+ bytes (HMAC) or provider-issued (OIDC)
- [ ] Issuer URL validated (OIDC providers)
- [ ] Redirect URIs properly configured

**Built-in Security (already enabled):**
- [x] State replay protection (timestamp + nonce)
- [x] Nonce cleanup (memory leak prevention)
- [x] Token cache with JWT expiry awareness
- [x] Input validation (parameter length limits)
- [x] Request body size limits (DoS prevention)
- [x] Constant-time HMAC comparison
- [x] Secure nonce generation (crypto/rand)
- [x] Security headers (CSP, X-Frame-Options, etc.)
- [x] CORS support (OPTIONS pass-through)

**Optional:**
- [ ] Custom logger configured (no sensitive data logged)
- [ ] Additional rate limiting on OAuth endpoints

### Regular Maintenance

- [ ] Rotate secrets every 90 days
- [ ] Review OAuth provider audit logs
- [ ] Monitor for unusual authentication patterns
- [ ] Update dependencies (`go get -u`)
- [ ] Review token expiration policies
- [ ] Test disaster recovery (secret compromise)

---

## 🚩 Security Incidents

### Token Compromise

**If JWT secret (HMAC) leaked:**

1. Generate new secret immediately
2. Update config and redeploy
3. All existing tokens invalidated (users must re-auth)
4. Review logs for suspicious activity

**If client secret (OIDC) leaked:**

1. Revoke in OAuth provider (Okta/Google/Azure)
2. Generate new secret
3. Update config and redeploy
4. Existing user tokens still valid (not affected)

### Suspicious Activity

- Multiple failed auth attempts → Consider IP blocking
- Unusual token usage patterns → Review logs
- Invalid redirect URI attempts → Security violation logged

---

## 📚 Additional Resources

- [OAuth 2.1 Security Best Practices](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)

---

## 🤝 Reporting Security Issues

Found a security vulnerability? Email security@[your-domain] or open a confidential GitHub Security Advisory.

Do NOT open public GitHub issues for security vulnerabilities.

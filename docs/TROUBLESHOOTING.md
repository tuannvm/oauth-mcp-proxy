# Troubleshooting Guide

Common issues and solutions when using oauth-mcp-proxy.

---

## Authentication Errors

### "Authentication required: missing OAuth token"

**Cause:** Token not extracted from HTTP request

**Solutions:**

1. **Check Authorization header present:**
```bash
# Make sure you're sending the header
curl -H "Authorization: Bearer <token>" https://server.com/mcp
```

2. **Verify CreateHTTPContextFunc configured:**
```go
streamable := mcpserver.NewStreamableHTTPServer(
    mcpServer,
    mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),  // Required!
)
```

3. **Check header format:**
```
✅ Authorization: Bearer eyJhbGc...
❌ Authorization: eyJhbGc...       (missing "Bearer ")
❌ authorization: Bearer ...       (lowercase - case-sensitive!)
```

---

### "Authentication failed: invalid token"

**Cause:** Token validation failed

**Check:**

1. **Token not expired:**
```bash
# Decode JWT (without validation) to check expiration
echo "<token>" | cut -d. -f2 | base64 -d 2>/dev/null | jq .exp
# Compare to current Unix timestamp
date +%s
```

2. **Issuer matches:**
```go
// Token's "iss" claim must match Config.Issuer exactly
Config.Issuer: "https://company.okta.com"
Token.iss:     "https://company.okta.com"  // Must match!
```

3. **Audience matches:**
```go
// Token's "aud" claim must match Config.Audience exactly
Config.Audience: "api://my-server"
Token.aud:       "api://my-server"  // Must match!
```

4. **Signature valid (HMAC):**
```go
// Secret must match the one used to sign token
Config.JWTSecret: []byte("secret-key-123")
// Token must be signed with same secret
```

5. **Provider reachable (OIDC):**
```bash
# Verify OIDC discovery works
curl https://yourcompany.okta.com/.well-known/openid-configuration
```

**Debug:**
```go
// Enable debug logging
type DebugLogger struct{}
func (l *DebugLogger) Debug(msg string, args ...interface{}) {
    log.Printf("[DEBUG] "+msg, args...)
}
// ... implement Info, Warn, Error

oauth.WithOAuth(mux, &oauth.Config{
    Logger: &DebugLogger{},  // See detailed validation logs
})
```

---

## Configuration Errors

### "invalid config: provider is required"

**Cause:** Missing or empty Provider field

**Solution:**
```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",  // Must be set!
    // ...
})
```

---

### "invalid config: JWTSecret is required for HMAC provider"

**Cause:** Using HMAC provider without JWTSecret

**Solution:**
```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    JWTSecret: []byte(os.Getenv("JWT_SECRET")),  // Required!
})
```

---

### "invalid config: Issuer is required for OIDC provider"

**Cause:** Using Okta/Google/Azure without Issuer

**Solution:**
```go
oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://yourcompany.okta.com",  // Required for OIDC!
})
```

---

### "invalid config: proxy mode requires ClientID"

**Cause:** Mode is "proxy" but ClientID not provided

**Solution:**
```go
oauth.WithOAuth(mux, &oauth.Config{
    Mode:     "proxy",
    ClientID: "your-client-id",  // Required for proxy mode
    ServerURL: "https://your-server.com",
    RedirectURIs: "...",
})
```

---

## Provider Errors

### "Failed to initialize OIDC provider"

**Cause:** Cannot connect to OAuth provider's discovery endpoint

**Check:**

1. **Issuer URL correct:**
```go
// ✅ Correct
Issuer: "https://company.okta.com"

// ❌ Common mistakes
Issuer: "https://company.okta.com/"   // Trailing slash
Issuer: "company.okta.com"             // Missing https://
Issuer: "http://company.okta.com"     // Must be HTTPS
```

2. **Network connectivity:**
```bash
# Verify server can reach provider
curl https://yourcompany.okta.com/.well-known/openid-configuration
```

3. **Firewall/proxy:**
- Check corporate firewall allows outbound HTTPS
- Check proxy settings if behind corporate proxy

**Debug:**
```bash
# Test OIDC discovery manually
curl -v https://yourcompany.okta.com/.well-known/openid-configuration
```

---

## Redirect URI Errors

### "Invalid redirect URI" (Native Mode)

**Cause:** Client redirect is not localhost (security protection)

**Fixed redirect mode only allows localhost:**

```
✅ http://localhost:8080/callback
✅ http://127.0.0.1:3000/callback
✅ http://[::1]:9000/callback
❌ http://app.example.com/callback    (not localhost)
❌ https://localhost.evil.com/...     (subdomain attack)
```

**Why:** Prevents open redirect attacks in fixed redirect mode.

**Solution:** Use allowlist mode if you need non-localhost redirects:
```go
RedirectURIs: "https://app1.com/cb,https://app2.com/cb"  // Allowlist
```

---

### "redirect_uri_mismatch" (Provider Error)

**Cause:** Redirect URI not configured in OAuth provider

**Solutions:**

**Okta:**
1. Go to Applications → Your App → General
2. Add to "Sign-in redirect URIs"
3. Must match exactly (including trailing slash if present)

**Google:**
1. Cloud Console → Credentials → OAuth 2.0 Client
2. Add to "Authorized redirect URIs"
3. Exact match required

**Azure:**
1. App registrations → Your App → Authentication
2. Add to "Redirect URIs"
3. Must match exactly

---

## Token Caching Issues

### Tokens Not Being Cached

**Expected:** Second request with same token should be faster (cache hit)

**Check:**

1. **Cache logs:**
```
[INFO] Using cached authentication for tool: hello (user: john)
```

2. **Cache TTL:** 5 minutes (hardcoded in v0.1.0)

3. **Cache scope:** Per Server instance

**Debug:**
- Different Server instances = different caches
- Token modified between requests = new cache entry
- Token expired = cache miss

**Metrics:**
```go
// Check if using cached validation
// Look for "Using cached authentication" in logs
```

---

## Runtime Errors

### Panic: "invalid memory address or nil pointer dereference"

**Cause:** Usually missing logger in test code or direct handler creation

**Solution:**
```go
// ✅ Always use WithOAuth() or NewServer()
oauthOption, _ := oauth.WithOAuth(mux, cfg)

// ❌ Don't create handlers directly (tests only)
handler := &OAuth2Handler{config: cfg}  // Missing logger!

// ✅ In tests, include logger
handler := &OAuth2Handler{
    config: cfg,
    logger: &oauth.defaultLogger{},  // Or use NewOAuth2Handler()
}
```

---

### "Token exchange failed"

**Cause:** OAuth provider rejected token exchange request

**Check:**

1. **Authorization code valid:**
- Code must be unused (single-use only)
- Code must not be expired (typically 10 minutes)

2. **PKCE parameters match:**
```go
// code_challenge in /authorize must match code_verifier in /token
// hash(code_verifier) == code_challenge
```

3. **Redirect URI matches:**
```go
// redirect_uri in /token must match the one used in /authorize
```

4. **Client credentials valid:**
```go
ClientID: "...",      // Must match OAuth provider
ClientSecret: "...",  // Must be current (not rotated)
```

**Debug:**
- Check OAuth provider logs (Okta/Google/Azure admin consoles)
- Look for specific error codes in provider response

---

## Performance Issues

### Slow Authentication

**Expected latency:**
- Cache hit: <5ms
- Cache miss (HMAC): <10ms
- Cache miss (OIDC): <100ms (network call to provider)

**If slower:**

1. **OIDC discovery slow:**
- First request does OIDC discovery (fetches `.well-known/openid-configuration`)
- Cached after first request
- Network latency to provider affects first request

2. **JWKS fetch slow:**
- OIDC validator fetches public keys on initialization
- Check network latency to OAuth provider

**Solutions:**
- Warm up on server start (make a test validation call)
- Check network connectivity to OAuth provider
- Consider caching OIDC discovery (future enhancement)

---

## Development vs Production

### Works Locally, Fails in Production

**Common causes:**

1. **HTTPS not configured:**
```go
// ❌ Development (http)
http.ListenAndServe(":8080", mux)

// ✅ Production (https)
http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
```

2. **Secrets not in environment:**
```bash
# Check environment variables are set
echo $OAUTH_CLIENT_SECRET
```

3. **Provider can't reach callback URL:**
- ServerURL must be publicly accessible
- Firewall must allow inbound HTTPS
- DNS must resolve correctly

4. **Redirect URI mismatch:**
- Localhost works in dev, but production URL different
- Update OAuth provider redirect URIs for production domain

---

## Debugging Tips

### Enable Verbose Logging

```go
type VerboseLogger struct{}

func (l *VerboseLogger) Debug(msg string, args ...interface{}) {
    log.Printf("[DEBUG] "+msg, args...)  // Enable debug
}
func (l *VerboseLogger) Info(msg string, args ...interface{}) {
    log.Printf("[INFO] "+msg, args...)
}
func (l *VerboseLogger) Warn(msg string, args ...interface{}) {
    log.Printf("[WARN] "+msg, args...)
}
func (l *VerboseLogger) Error(msg string, args ...interface{}) {
    log.Printf("[ERROR] "+msg, args...)
}

oauth.WithOAuth(mux, &oauth.Config{
    Logger: &VerboseLogger{},
})
```

### Check OAuth Metadata

```bash
# Verify OAuth configuration
curl https://your-server.com/.well-known/oauth-authorization-server | jq

# Check OIDC discovery
curl https://your-server.com/.well-known/openid-configuration | jq

# Verify JWKS endpoint (OIDC providers)
curl https://your-server.com/.well-known/jwks.json | jq
```

### Decode JWT Token

```bash
# Decode without verification (debugging only!)
echo "<token>" | cut -d. -f2 | base64 -d 2>/dev/null | jq

# Check claims:
# - iss matches Config.Issuer?
# - aud matches Config.Audience?
# - exp is in the future?
```

### Test Token Manually

```bash
# Generate test token (HMAC)
go run examples/simple/main.go
# Copy token from output, test with curl

# For OIDC providers, get token from provider:
# - Okta: Use Okta test tool or API call
# - Google: Use OAuth Playground
# - Azure: Use Azure portal token tool
```

---

## Still Having Issues?

1. **Check logs:** Look for ERROR and WARN level messages
2. **Verify configuration:** Review [CONFIGURATION.md](CONFIGURATION.md)
3. **Check provider setup:** Review provider-specific guide in [providers/](providers/)
4. **Security check:** Review [SECURITY.md](SECURITY.md)
5. **GitHub Issues:** Search or create issue at [github.com/tuannvm/oauth-mcp-proxy/issues](https://github.com/tuannvm/oauth-mcp-proxy/issues)

---

## Common Patterns

### Multiple OAuth Providers

```go
// Create separate Server instances
oktaOption, _ := oauth.WithOAuth(mux, &oauth.Config{Provider: "okta", ...})
googleOption, _ := oauth.WithOAuth(mux, &oauth.Config{Provider: "google", ...})

// Note: Can only use one per MCP server currently
// Use environment variables to select at runtime
```

### Custom Token Claims

Currently, oauth-mcp-proxy extracts:
- `sub` → User.Subject
- `email` → User.Email
- `preferred_username` → User.Username (fallback to email or sub)

For custom claims, access the raw token:
```go
// Get token string from context
token, _ := oauth.GetOAuthToken(ctx)
// Parse and extract custom claims as needed
```

---

## Getting Help

- 📖 **Documentation:** [docs/](.)
- 💬 **Discussions:** GitHub Discussions (coming soon)
- 🐛 **Bug Reports:** [GitHub Issues](https://github.com/tuannvm/oauth-mcp-proxy/issues)
- 🔒 **Security:** Email maintainer for confidential issues

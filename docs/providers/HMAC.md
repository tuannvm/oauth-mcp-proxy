# HMAC Provider Guide

> **📢 v1.0.0:** This guide shows examples for both `mark3labs/mcp-go` and official `modelcontextprotocol/go-sdk`.
> See [MIGRATION-V2.md](../../MIGRATION-V2.md) for upgrade details.

## Overview

HMAC provider uses shared secret JWT validation with HS256 algorithm. Best for testing, development, and service-to-service authentication.

## When to Use

✅ **Good for:**

- Local development and testing
- Service-to-service authentication
- Simple deployments without external OAuth provider
- Full control over token generation

❌ **Not ideal for:**

- User authentication (no SSO)
- Public-facing applications (secret distribution problem)
- Multi-tenant applications

---

## Configuration

### Using mark3labs/mcp-go

```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

mux := http.NewServeMux()

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-mcp-server",      // Your server's identifier
    JWTSecret: []byte("your-secret-key"),  // 32+ bytes recommended
})

mcpServer := server.NewMCPServer("My Server", "1.0.0", oauthOption)
```

### Using Official SDK

```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

mux := http.NewServeMux()
mcpServer := mcp.NewServer(&mcp.Implementation{
    Name:    "My Server",
    Version: "1.0.0",
}, nil)

_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-mcp-server",      // Your server's identifier
    JWTSecret: []byte("your-secret-key"),  // 32+ bytes recommended
}, mcpServer)

http.ListenAndServe(":8080", handler)
```

### Required Fields

- `Provider: "hmac"` - Use HMAC validator
- `Audience` - Must match the `aud` claim in tokens
- `JWTSecret` - Shared secret for signing/verifying tokens (32+ bytes recommended)

---

## Token Generation

Generate tokens using `github.com/golang-jwt/jwt/v5`:

```go
import "github.com/golang-jwt/jwt/v5"

func generateToken(secret []byte, audience string) string {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub":   "user-123",               // Subject (user ID)
        "email": "user@example.com",       // Email
        "preferred_username": "john.doe",  // Username
        "aud":   audience,                 // Must match Config.Audience
        "iss":   "https://your-server.com",// Issuer
        "exp":   time.Now().Add(time.Hour).Unix(),
        "iat":   time.Now().Unix(),
    })

    tokenString, _ := token.SignedString(secret)
    return tokenString
}
```

### Required JWT Claims

- `sub` - Subject (user identifier)
- `aud` - Audience (must match `Config.Audience`)
- `exp` - Expiration (Unix timestamp)
- `iat` - Issued at (Unix timestamp)

### Optional Claims (extracted if present)

- `email` - User's email
- `preferred_username` - Username (falls back to `email` or `sub`)

---

## Security Considerations

### Secret Management

```bash
# Store secret in environment variable
export JWT_SECRET="your-long-random-secret-key-min-32-bytes"
```

```go
// Load from environment
secret := []byte(os.Getenv("JWT_SECRET"))
if len(secret) < 32 {
    log.Fatal("JWT_SECRET must be at least 32 bytes")
}

// mark3labs SDK:
_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-server",
    JWTSecret: secret,
})
mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)

// OR official SDK:
_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider:  "hmac",
    Audience:  "api://my-server",
    JWTSecret: secret,
}, mcpServer)
```

### Secret Strength

- **Minimum:** 32 bytes (256 bits)
- **Recommended:** Generate with `crypto/rand`
- **Never:** Use passwords, dictionary words, or predictable values

```go
// Generate secure secret
secret := make([]byte, 32)
if _, err := rand.Read(secret); err != nil {
    log.Fatal(err)
}
fmt.Printf("Secret (base64): %s\n", base64.StdEncoding.EncodeToString(secret))
```

### Token Expiration

- **Recommended:** 1 hour for user tokens
- **Service tokens:** Up to 24 hours
- Always include `exp` claim

---

## Testing

### 1. Start Your MCP Server

```bash
export JWT_SECRET="test-secret-key-must-be-32-bytes-long!"
go run main.go
```

### 2. Generate Test Token

```go
token := generateToken(
    []byte("test-secret-key-must-be-32-bytes-long!"),
    "api://my-mcp-server",
)
fmt.Println("Token:", token)
```

### 3. Test Authentication

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}'
```

---

## Complete Examples

**mark3labs SDK:**
- [examples/mark3labs/simple/](../../examples/mark3labs/simple/) - Minimal HMAC setup
- [examples/mark3labs/advanced/](../../examples/mark3labs/advanced/) - Full featured

**Official SDK:**
- [examples/official/simple/](../../examples/official/simple/) - Minimal HMAC setup
- [examples/official/advanced/](../../examples/official/advanced/) - Full featured

See [examples/README.md](../../examples/README.md) for setup instructions.

---

## Limitations

- No built-in user management (you generate tokens)
- Secret must be shared with all token generators
- No automatic token refresh
- Not suitable for public clients (secret exposure risk)

For user authentication with SSO, consider Okta, Google, or Azure providers.

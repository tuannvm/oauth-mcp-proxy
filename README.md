# OAuth MCP proxy

OAuth 2.1 authentication library for Go MCP servers.

**Supports both MCP SDKs:**
- ✅ `mark3labs/mcp-go`
- ✅ `modelcontextprotocol/go-sdk` (official)

**One-time setup:** Configure provider + add `WithOAuth()` to your server.
**Result:** All tools automatically protected with token validation and caching.

### mark3labs/mcp-go
```go
import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"

_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://your-company.okta.com",
    Audience: "api://your-mcp-server",
})

mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
```

### Official SDK
```go
import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"

mcpServer := mcp.NewServer(&mcp.Implementation{...}, nil)
_, handler, _ := mcpoauth.WithOAuth(mux, cfg, mcpServer)
http.ListenAndServe(":8080", handler)
```


[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/tuannvm/oauth-mcp-proxy/test.yml?branch=main&label=Tests&logo=github)](https://github.com/tuannvm/oauth-mcp-proxy/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tuannvm/oauth-mcp-proxy?logo=go)](https://github.com/tuannvm/oauth-mcp-proxy/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/tuannvm/oauth-mcp-proxy)](https://goreportcard.com/report/github.com/tuannvm/oauth-mcp-proxy)
[![Go Reference](https://pkg.go.dev/badge/github.com/tuannvm/oauth-mcp-proxy.svg)](https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy)
[![GitHub Release](https://img.shields.io/github/v/release/tuannvm/oauth-mcp-proxy?sort=semver)](https://github.com/tuannvm/oauth-mcp-proxy/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Why Use This Library?

- **Dual SDK support** - Works with both mark3labs and official SDKs
- **Simple integration** - One `WithOAuth()` call protects all tools
- **Zero per-tool config** - All tools automatically protected
- **Fast token caching** - 5-min cache, <5ms validation
- **Production ready** - Security hardened, battle-tested
- **Multiple providers** - HMAC, Okta, Google, Azure AD

---

## How It Works

### Request Flow

```mermaid
sequenceDiagram
    participant Client
    participant MCP Server
    box lightyellow oauth-mcp-proxy Library
    participant Middleware
    participant Cache
    participant Provider
    end
    participant Your Tool Handler

    Client->>MCP Server: Request + Bearer token
    MCP Server->>Middleware: WithOAuth() intercepts

    alt Token in cache and fresh
        Middleware->>Cache: Check token hash
        Cache-->>Middleware: Return cached user
    else Token not cached or expired
        Middleware->>Provider: Validate token (HMAC/OIDC)
        Provider-->>Middleware: User claims
        Middleware->>Cache: Store user for 5 minutes
    end

    Middleware->>Your Tool Handler: Pass request with user in context
    Your Tool Handler->>Your Tool Handler: GetUserFromContext(ctx)
    Your Tool Handler-->>Client: Send response
```

### Token Validation Flow

```mermaid
flowchart TB
    Start([Your MCP Server receives request]) --> Extract[oauth-mcp-proxy: Extract Token]
    Extract --> Hash[oauth-mcp-proxy: SHA-256 Hash]
    Hash --> CheckCache{oauth-mcp-proxy: Token Cached?}

    CheckCache -->|Cache Hit| GetUser[oauth-mcp-proxy: Get Cached User]
    CheckCache -->|Cache Miss| Validate{oauth-mcp-proxy: Validate}

    Validate -->|Valid| Claims[oauth-mcp-proxy: Extract Claims]
    Validate -->|Invalid| Reject([Return 401])

    Claims --> Store[oauth-mcp-proxy: Cache]
    Store --> GetUser

    GetUser --> Context[oauth-mcp-proxy: Add User to Context]
    Context --> Tool[Your Tool Handler: GetUserFromContext]
    Tool --> Response([Your MCP Server: Return Response])

    style Start fill:#e8f5e9
    style Extract fill:#fff9c4
    style Hash fill:#fff9c4
    style CheckCache fill:#fff9c4
    style Validate fill:#fff9c4
    style Claims fill:#fff9c4
    style Store fill:#fff9c4
    style GetUser fill:#fff9c4
    style Context fill:#fff9c4
    style Tool fill:#e8f5e9
    style Response fill:#e8f5e9
    style Reject fill:#ffebee
```

**What oauth-mcp-proxy does:**

1. Extracts Bearer tokens from HTTP requests
2. Validates against your OAuth provider (with caching)
3. Adds authenticated user to request context
4. All your tools automatically protected

---

## Quick Start

### Using mark3labs/mcp-go

#### 1. Install

```bash
go get github.com/tuannvm/oauth-mcp-proxy
```

#### 2. Add to Your Server

```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
)

mux := http.NewServeMux()

// Enable OAuth (one time setup)
_, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
    Provider: "okta",                    // or "hmac", "google", "azure"
    Issuer:   "https://your-company.okta.com",
    Audience: "api://your-mcp-server",
    ServerURL: "https://your-server.com",
})

// Create MCP server with OAuth
mcpServer := mcpserver.NewMCPServer("Server", "1.0.0", oauthOption)

// Add tools - all automatically protected
mcpServer.AddTool(myTool, myHandler)

// Setup endpoint
streamable := mcpserver.NewStreamableHTTPServer(
    mcpServer,
    mcpserver.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
)
mux.Handle("/mcp", streamable)
```

#### 3. Access Authenticated User

```go
func myHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    user, ok := oauth.GetUserFromContext(ctx)
    if !ok {
        return nil, fmt.Errorf("authentication required")
    }
    // Use user.Username, user.Email, user.Subject
}
```

---

### Using Official SDK

#### 1. Install

```bash
go get github.com/modelcontextprotocol/go-sdk
go get github.com/tuannvm/oauth-mcp-proxy
```

#### 2. Add to Your Server

```go
import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

mux := http.NewServeMux()

// Create MCP server
mcpServer := mcp.NewServer(&mcp.Implementation{
    Name:    "my-server",
    Version: "1.0.0",
}, nil)

// Add tools
mcp.AddTool(mcpServer, &mcp.Tool{
    Name: "greet",
    Description: "Greet user",
}, func(ctx context.Context, req *mcp.CallToolRequest, params *struct{}) (*mcp.CallToolResult, any, error) {
    user, _ := oauth.GetUserFromContext(ctx)
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: "Hello, " + user.Username},
        },
    }, nil, nil
})

// Add OAuth protection
_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",
    Issuer:   "https://your-company.okta.com",
    Audience: "api://your-mcp-server",
}, mcpServer)

http.ListenAndServe(":8080", handler)
```

Your MCP server now requires OAuth authentication.

---

## Examples

See [examples/README.md](examples/README.md) for detailed setup guide including Okta configuration.

| SDK | Example | Description |
|-----|---------|-------------|
| **mark3labs** | [Simple](examples/mark3labs/simple/) | Minimal setup - copy/paste ready |
| **mark3labs** | [Advanced](examples/mark3labs/advanced/) | ConfigBuilder, multiple tools, logging |
| **Official** | [Simple](examples/official/simple/) | Minimal setup - copy/paste ready |
| **Official** | [Advanced](examples/official/advanced/) | ConfigBuilder, multiple tools, logging |

---

## Supported Providers

| Provider | Best For | Setup Guide |
|----------|----------|-------------|
| **HMAC** | Testing, development | [docs/providers/HMAC.md](docs/providers/HMAC.md) |
| **Okta** | Enterprise SSO | [docs/providers/OKTA.md](docs/providers/OKTA.md) |
| **Google** | Google Workspace | [docs/providers/GOOGLE.md](docs/providers/GOOGLE.md) |
| **Azure AD** | Microsoft 365 | [docs/providers/AZURE.md](docs/providers/AZURE.md) |

---

## Documentation

**Getting Started:**

- [Configuration Guide](docs/CONFIGURATION.md) - All config options
- [Client Setup](docs/CLIENT-SETUP.md) - Client configuration
- [Provider Setup](docs/providers/) - OAuth provider guides

**Advanced:**

- [Security Guide](docs/SECURITY.md) - Production best practices
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues

---

## License

MIT License - See [LICENSE](LICENSE)

# oauth-mcp-proxy

**OAuth 2.1 authentication library for Go MCP servers.**

Minimal server-side integration (3 lines of Go code) + deployment configuration.

```go
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{Provider: "okta", ...})
mcpServer := server.NewMCPServer("My Server", "1.0.0", oauthOption)
// Server-side OAuth complete. Also need: provider setup + deployment config + client config.
```

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/tuannvm/oauth-mcp-proxy/test.yml?branch=main&label=Tests&logo=github)](https://github.com/tuannvm/oauth-mcp-proxy/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tuannvm/oauth-mcp-proxy?logo=go)](https://github.com/tuannvm/oauth-mcp-proxy/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/tuannvm/oauth-mcp-proxy)](https://goreportcard.com/report/github.com/tuannvm/oauth-mcp-proxy)
[![Go Reference](https://pkg.go.dev/badge/github.com/tuannvm/oauth-mcp-proxy.svg)](https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy)
[![GitHub Release](https://img.shields.io/github/v/release/tuannvm/oauth-mcp-proxy?sort=semver)](https://github.com/tuannvm/oauth-mcp-proxy/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Complete Setup Overview

```mermaid
graph TD
    subgraph "1. OAuth Provider Setup"
        A[Create OAuth App<br/>Okta/Google/Azure]
        A --> B[Get ClientID<br/>Get ClientSecret]
    end

    subgraph "2. Server Integration"
        C[Add 3 Lines Go Code<br/>WithOAuth]
        D[Configure Deployment<br/>Helm/env vars]
        C --> D
    end

    subgraph "3. Client Configuration"
        E[Client discovers<br/>via .well-known endpoints]
        F[Or manual config<br/>claude_desktop_config.json]
        E -.->|Auto| G[Client Ready]
        F -.->|Manual| G
    end

    B --> C
    D --> E
    D --> F

    style A fill:#ffe5e5
    style C fill:#e1f5ff
    style G fill:#d4edda
```

**What you need:**
1. OAuth provider configured (one-time setup)
2. Server code updated (3 lines)
3. Deployment configured (environment variables / Helm)
4. Client configured (auto-discovery or manual)

---

## Architecture

```mermaid
graph LR
    Client[MCP Client] -->|HTTP + Bearer Token| Server[Your MCP Server]
    Server -->|1. Extract Token| OAuth[oauth-mcp-proxy]
    OAuth -->|2. Validate| Provider[OAuth Provider<br/>Okta/Google/Azure]
    OAuth -->|3. Add User to Context| Tools[Your MCP Tools]

    style OAuth fill:#e1f5ff
    style Tools fill:#d4edda
```

**What oauth-mcp-proxy does:**
1. Extracts tokens from HTTP requests
2. Validates against OAuth provider (with caching)
3. Adds authenticated user to context
4. Protects all your tools automatically

---

## Authentication Flow

```mermaid
sequenceDiagram
    participant C as MCP Client
    participant S as Your Server
    participant O as oauth-mcp-proxy
    participant P as OAuth Provider

    C->>S: POST /mcp<br/>Header: Bearer token
    S->>O: Extract token from context

    alt Token in cache
        O->>O: Return cached user (< 5ms)
    else Token not cached
        O->>P: Validate token (JWKS/OIDC)
        P->>O: Token valid + claims
        O->>O: Cache for 5 min
    end

    O->>S: Add User to context
    S->>C: Execute tool with auth context

    Note over O: Token caching saves<br/>~95ms per request
```

---

## Quick Start

**Prerequisites:** OAuth app created in your provider (Okta/Google/Azure). See [Provider Guides](docs/providers/).

### 1. Install Library

```bash
go get github.com/tuannvm/oauth-mcp-proxy
```

### 2. Add to Server Code (3 lines)

```go
import oauth "github.com/tuannvm/oauth-mcp-proxy"

mux := http.NewServeMux()
oauthOption, _ := oauth.WithOAuth(mux, &oauth.Config{
    Provider: "okta",                        // or "hmac", "google", "azure"
    Issuer:   os.Getenv("OAUTH_ISSUER"),     // From environment
    Audience: os.Getenv("OAUTH_AUDIENCE"),
})
mcpServer := mcpserver.NewMCPServer("Server", "1.0.0", oauthOption)
```

### 3. Configure Deployment

**Environment variables** (Kubernetes ConfigMap, docker-compose, etc.):
```bash
OAUTH_PROVIDER=okta
OAUTH_ISSUER=https://company.okta.com
OAUTH_AUDIENCE=api://my-server
# For proxy mode: OAUTH_CLIENT_ID, OAUTH_CLIENT_SECRET, etc.
```

**See:** [Configuration Guide](docs/CONFIGURATION.md#environment-variables-pattern)

### 4. Configure Client

**Auto-discovery** (Claude Desktop):
```json
{"mcpServers": {"my-server": {"url": "https://your-server.com/mcp"}}}
```

Client auto-discovers OAuth via `.well-known` endpoints.

**See:** [Client Setup Guide](docs/CLIENT-SETUP.md)

**Complete example:** [examples/simple/](examples/simple/)

---

## Providers

```mermaid
graph TD
    A[oauth-mcp-proxy] --> B[HMAC<br/>Shared Secret]
    A --> C[Okta<br/>Enterprise SSO]
    A --> D[Google<br/>Workspace]
    A --> E[Azure AD<br/>Microsoft 365]

    B -.->|Testing/Dev| F[Your Choice]
    C -.->|Enterprise| F
    D -.->|Google Users| F
    E -.->|MS Users| F

    style A fill:#e1f5ff
    style F fill:#d4edda
```

| Provider | Best For | Setup Guide |
|----------|----------|-------------|
| **HMAC** | Testing, development | [Setup](docs/providers/HMAC.md) |
| **Okta** | Enterprise SSO | [Setup](docs/providers/OKTA.md) |
| **Google** | Google Workspace | [Setup](docs/providers/GOOGLE.md) |
| **Azure AD** | Microsoft 365 | [Setup](docs/providers/AZURE.md) |

**Quick config examples:** See [Configuration Guide](docs/CONFIGURATION.md)

---

## Features

- ✅ **3-line integration** - `WithOAuth()` handles everything
- ✅ **Token caching** - 5-minute cache, <5ms validation
- ✅ **Security hardened** - PKCE, redirect validation, defense-in-depth
- ✅ **Pluggable logging** - Integrate with zap, logrus, slog
- ✅ **Instance-scoped** - No globals, thread-safe
- ✅ **OAuth 2.1** - Latest spec compliance

---

## Documentation

📖 **Setup Guides:**
- [Provider Setup](docs/providers/) - OAuth provider configuration (Okta/Google/Azure)
- [Configuration Reference](docs/CONFIGURATION.md) - All server config options
- [Client Setup](docs/CLIENT-SETUP.md) - Client configuration & auto-discovery
- [Deployment](docs/CONFIGURATION.md#environment-variables-pattern) - Helm/env vars

📚 **Reference:**
- [Security Best Practices](docs/SECURITY.md) - Production security guide
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues & solutions
- [Migration from mcp-trino](docs/MIGRATION.md) - Upgrade guide

🎯 **Examples:**
- [Simple Example](examples/simple/) - 3-line integration (recommended)
- [Advanced Example](examples/embedded/) - Lower-level API

📋 **Planning:**
- [v0.1.0 Plan](docs/plan.md) - Current release scope
- [v0.2.0 Plan](docs/plan-standalone.md) - Future standalone mode

---

## Status

**Current Release:** v0.0.1 (Preview)

| Phase | Status |
|-------|--------|
| 0-5 | ✅ **Complete** |
| 6 | ⏳ Next: mcp-trino migration |

**Stable Release (v0.1.0):** After Phase 6 validation complete

---

## Dependencies

4 well-maintained, industry-standard libraries:

- `github.com/mark3labs/mcp-go` v0.41.1 - MCP protocol
- `github.com/coreos/go-oidc/v3` v3.16.0 - OIDC validation
- `github.com/golang-jwt/jwt/v5` v5.3.0 - JWT validation
- `golang.org/x/oauth2` v0.32.0 - OAuth flows

All required for core functionality.

---

## Contributing

Not accepting contributions during extraction phase. After v0.1.0 release, contributions welcome!

**Report issues:** [GitHub Issues](https://github.com/tuannvm/oauth-mcp-proxy/issues)

---

## License

MIT License - See [LICENSE](LICENSE)

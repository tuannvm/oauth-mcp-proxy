# OAuth MCP Proxy - Standalone Mode Plan (v0.2.0)

> **Status:** 📋 Planning - Not started yet
> **Prerequisite:** v0.1.0 (Embedded mode) must be complete and stable

---

## Overview

**Standalone mode** runs oauth-mcp-proxy as a separate proxy service that:
1. Handles OAuth authentication
2. Routes authenticated requests to downstream MCP servers (no auth)
3. Propagates user context to downstream servers

## Architecture

```
┌──────────┐
│  Client  │
│ (OAuth)  │
└────┬─────┘
     │
     │ 1. OAuth authentication
     ↓
┌────────────────────────────┐
│   oauth-mcp-proxy          │
│   (Standalone Service)     │
│                            │
│  ┌──────────────────────┐  │
│  │ OAuth Handler        │  │
│  │ - Validate token     │  │
│  │ - Extract user info  │  │
│  └──────────────────────┘  │
│            │               │
│            ↓               │
│  ┌──────────────────────┐  │
│  │ Router               │  │
│  │ - Route by path      │  │
│  │ - Route by user      │  │
│  │ - Route by tenant    │  │
│  └──────────────────────┘  │
│            │               │
│            ↓               │
│  ┌──────────────────────┐  │
│  │ Context Injector     │  │
│  │ - Add user headers   │  │
│  │ - Transform request  │  │
│  └──────────────────────┘  │
└────────────┬───────────────┘
             │
             │ 2. Proxy with user context
             ↓
   ┌─────────┴────────────┐
   │                      │
┌──▼─────────┐   ┌────────▼──┐
│ MCP        │   │ MCP       │
│ Server A   │   │ Server B  │
│ (no auth)  │   │ (no auth) │
└────────────┘   └───────────┘
```

---

## Key Design Questions

### 1. Routing Strategy

**How to determine which downstream server to route to?**

**Option A: Path-based routing**
```yaml
routes:
  - path: /trino/*
    target: http://mcp-trino:8080
  - path: /postgres/*
    target: http://mcp-postgres:8080
```

**Option B: User-based routing**
```yaml
routes:
  - users: [alice@company.com, bob@company.com]
    target: http://mcp-team-a:8080
  - users: [charlie@company.com]
    target: http://mcp-team-b:8080
```

**Option C: Tenant-based routing**
```yaml
routes:
  - tenant: company-a
    target: http://mcp-company-a:8080
  - tenant: company-b
    target: http://mcp-company-b:8080
```

**Option D: Header-based routing**
```
X-MCP-Server: trino → http://mcp-trino:8080
X-MCP-Server: postgres → http://mcp-postgres:8080
```

**Recommendation:** Start with path-based (simplest), add others as needed

---

### 2. User Context Propagation

**How does downstream MCP server know who's authenticated?**

**Option A: HTTP Headers**
```http
GET /tools HTTP/1.1
X-User-Email: alice@company.com
X-User-Subject: user-123
X-User-Name: Alice
```

**Option B: JWT Token**
```http
GET /tools HTTP/1.1
X-User-JWT: eyJhbGc...  (signed by oauth-mcp-proxy)
```

**Option C: Custom Protocol Extension**
```json
{
  "mcp_version": "2024-11-05",
  "user": {
    "email": "alice@company.com",
    "subject": "user-123"
  },
  "method": "tools/list"
}
```

**Recommendation:** HTTP headers (simplest, works with existing MCP servers)

---

### 3. MCP Protocol Handling

**What protocols need to be proxied?**

- ✅ **HTTP/SSE** - Standard MCP over HTTP
- ❓ **stdio** - Not applicable (proxy can't intercept)
- ❓ **WebSocket** - Future consideration

**Initial Focus:** HTTP/SSE only

---

### 4. Configuration Format

```yaml
# config.yaml
server:
  port: 9000
  host: 0.0.0.0

oauth:
  mode: proxy
  provider: okta
  issuer: https://company.okta.com
  audience: api://mcp-proxy
  client_id: your-client-id
  client_secret: your-client-secret

routes:
  - name: trino
    path: /trino/*
    target: http://mcp-trino:8080
    strip_prefix: /trino

  - name: postgres
    path: /postgres/*
    target: http://mcp-postgres:8080
    strip_prefix: /postgres

security:
  validate_api_key: your-api-key  # For /validate endpoint

metrics:
  enabled: true
  path: /metrics

health:
  enabled: true
  path: /health
```

---

## Implementation Phases (v0.2.0)

### Phase 1: Basic Proxy

**Goal:** Forward requests to single downstream server

**Tasks:**
- [ ] Create `cmd/oauth-mcp-proxy/main.go`
- [ ] Configuration loading (YAML + env vars)
- [ ] HTTP proxy middleware
- [ ] Single route support (path-based)
- [ ] User header injection (X-User-*)

**Success Criteria:**
- Binary runs
- Authenticates user
- Forwards to downstream MCP server
- Downstream sees user headers

---

### Phase 2: Multi-Server Routing

**Goal:** Route to multiple downstream servers

**Tasks:**
- [ ] Path-based routing (strip prefix)
- [ ] Route matching logic
- [ ] Error handling (no route found)
- [ ] Health checks per route

**Success Criteria:**
- Multiple routes work
- Correct routing by path
- 404 for unknown paths

---

### Phase 3: Validation Endpoint

**Goal:** `/validate` endpoint for external callers

**Tasks:**
- [ ] `POST /validate` endpoint
- [ ] API key authentication
- [ ] Token validation logic
- [ ] Return user info as JSON

**Success Criteria:**
- External services can validate tokens
- API key protects endpoint

---

### Phase 4: Observability

**Goal:** Metrics and health endpoints

**Tasks:**
- [ ] `/health` endpoint
- [ ] `/metrics` endpoint (Prometheus format)
- [ ] Request metrics (per route)
- [ ] Error metrics
- [ ] Latency tracking

**Success Criteria:**
- Prometheus can scrape metrics
- Health checks work

---

### Phase 5: Advanced Routing

**Goal:** Additional routing strategies

**Tasks:**
- [ ] User-based routing
- [ ] Tenant-based routing (extract from token claims)
- [ ] Header-based routing
- [ ] Route priority/fallback

**Success Criteria:**
- Multiple routing strategies work
- Can combine strategies

---

## API Endpoints (Standalone Mode)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/oauth/authorize` | GET | OAuth authorization flow |
| `/oauth/callback` | GET | OAuth callback |
| `/oauth/token` | POST | Token exchange |
| `/.well-known/*` | GET | OAuth metadata |
| `/validate` | POST | Token validation API |
| `/health` | GET | Health check |
| `/metrics` | GET | Prometheus metrics |
| `/{route}/*` | ANY | Proxy to downstream MCP |

---

## Security Considerations

### 1. Downstream Trust Model

**Problem:** Downstream MCP servers trust proxy-provided headers

**Mitigation:**
- Run downstream MCP servers in private network
- Use mTLS between proxy and downstream
- Sign user JWT with proxy's private key

### 2. Token Replay

**Problem:** Token stolen from proxy could be replayed

**Mitigation:**
- Short token TTL (5 min)
- Token caching at proxy level
- Rate limiting per token

### 3. Route Authorization

**Problem:** User accesses unauthorized route

**Solution:**
```yaml
routes:
  - name: admin-tools
    path: /admin/*
    target: http://admin-mcp:8080
    required_claims:
      role: admin  # Check token claim
```

---

## Configuration Schema

```go
// Config for standalone mode
type StandaloneConfig struct {
    Server   ServerConfig   `yaml:"server"`
    OAuth    OAuthConfig    `yaml:"oauth"`
    Routes   []Route        `yaml:"routes"`
    Security SecurityConfig `yaml:"security"`
    Metrics  MetricsConfig  `yaml:"metrics"`
}

type Route struct {
    Name         string            `yaml:"name"`
    Path         string            `yaml:"path"`
    Target       string            `yaml:"target"`
    StripPrefix  string            `yaml:"strip_prefix"`
    RequiredClaims map[string]string `yaml:"required_claims"`
}
```

---

## Testing Strategy

### Unit Tests
- Route matching logic
- Header injection
- Token validation

### Integration Tests
- End-to-end proxy flow
- Multiple downstream servers
- Different routing strategies

### Load Tests
- Concurrent requests
- Route switching performance
- Token cache effectiveness

---

## Open Questions

1. **WebSocket Support:** Do we need to proxy WebSocket connections?
2. **Request Transformation:** Should proxy modify MCP requests?
3. **Response Caching:** Should proxy cache downstream responses?
4. **Circuit Breaker:** Add circuit breaker for downstream failures?
5. **Service Discovery:** Integrate with Consul/K8s service discovery?

---

## Success Criteria (v0.2.0)

- ✅ Standalone binary runs
- ✅ Routes to multiple downstream MCP servers
- ✅ Path-based routing works
- ✅ User headers injected correctly
- ✅ /validate endpoint works
- ✅ /health and /metrics work
- ✅ mcp-trino can be proxied successfully
- ✅ Documentation complete

---

**Document Version:** 1.0
**Date:** 2025-10-17
**Status:** 📋 Planning - Awaiting v0.1.0 completion

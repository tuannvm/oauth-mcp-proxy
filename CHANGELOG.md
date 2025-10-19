# Changelog

All notable changes to oauth-mcp-proxy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2025-10-19

**Preview Release** - Core functionality complete, pending mcp-trino migration validation (Phase 6).

### Added
- Initial extraction from mcp-trino
- OAuth 2.1 authentication for MCP servers
- Support for 4 providers: HMAC, Okta, Google, Azure AD
- Native and proxy OAuth modes
- `WithOAuth()` simple API for easy integration
- Token validation with 5-minute caching
- Pluggable logger interface
- Instance-scoped state (no globals)
- PKCE support (RFC 7636)
- Comprehensive documentation and examples
- Provider setup guides
- Security best practices guide
- Client configuration guide
- Migration guide from mcp-trino

### Fixed
- Global state → Instance-scoped (Phase 1.5)
- Hardcoded logging → Pluggable Logger interface
- Missing configuration validation

### Security
- Defense-in-depth redirect URI validation
- HMAC-signed state for proxy callbacks
- Localhost-only validation for fixed redirect mode
- Token hash logging (never log full tokens)

[Unreleased]: https://github.com/tuannvm/oauth-mcp-proxy/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/tuannvm/oauth-mcp-proxy/releases/tag/v0.0.1

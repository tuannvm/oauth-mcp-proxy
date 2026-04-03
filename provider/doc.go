// Package provider provides token validation interfaces and implementations
// for OAuth 2.0/OIDC providers.
//
// Supported Validators:
//
//   - HMACValidator: Validates JWT tokens using HMAC-SHA256 (shared secret)
//   - OIDCValidator: Validates JWT tokens using OIDC discovery and JWKS
//
// Usage:
//
// Validators implement the TokenValidator interface:
//
//	type TokenValidator interface {
//	    ValidateToken(ctx context.Context, token string) (*User, error)
//	}
//
// Create validators via Config.Validate() or use factory functions:
//
//	validator := NewHMACValidator(jwtSecret)
//	validator := NewOIDCValidator(issuer, audience, httpClient)
//
// User Type:
//
// The User type represents an authenticated user:
//
//	type User struct {
//	    Subject  string   // User ID (sub claim)
//	    Username string   // Username (preferred_username or name claim)
//	    Email    string   // Email (email claim)
//	    Expiry   time.Time // Token expiration (exp claim)
//	}
package provider

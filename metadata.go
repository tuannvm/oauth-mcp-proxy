package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HandleMetadata handles the legacy OAuth metadata endpoint for MCP compliance
func (h *OAuth2Handler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if r.Method != "GET" {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Return OAuth metadata based on configuration
	if !h.config.Enabled {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{
			"oauth_enabled": false,
			"authentication_methods": ["none"],
			"mcp_version": "1.0.0"
		}`)
		return
	}

	// Create provider-specific metadata
	metadata := map[string]interface{}{
		"oauth_enabled":          true,
		"authentication_methods": []string{"bearer_token"},
		"token_types":            []string{"JWT"},
		"token_validation":       "server_side",
		"supported_flows":        []string{"claude_code", "mcp_remote"},
		"mcp_version":            "1.0.0",
		"server_version":         h.config.Version,
		"provider":               h.config.Provider,

		// Add OIDC discovery fields for MCP client compatibility
		"issuer":                   h.config.MCPURL,
		"authorization_endpoint":   fmt.Sprintf("%s/oauth/authorize", h.config.MCPURL),
		"token_endpoint":           fmt.Sprintf("%s/oauth/token", h.config.MCPURL),
		"registration_endpoint":    fmt.Sprintf("%s/oauth/register", h.config.MCPURL),
		"response_types_supported": []string{"code"},
		"response_modes_supported": []string{"query"},
		"grant_types_supported":    []string{"authorization_code"},
	}

	// Add provider-specific metadata
	switch h.config.Provider {
	case "hmac":
		metadata["validation_method"] = "hmac_sha256"
		metadata["signature_algorithm"] = "HS256"
		metadata["requires_secret"] = true
	case "okta", "google", "azure":
		metadata["validation_method"] = "oidc_jwks"
		metadata["signature_algorithm"] = "RS256"
		metadata["requires_secret"] = false
		if h.config.Issuer != "" {
			metadata["issuer"] = h.config.Issuer
			metadata["jwks_uri"] = h.config.Issuer + "/.well-known/jwks.json"
		}
		if h.config.Audience != "" {
			metadata["audience"] = h.config.Audience
		}
	}

	// Encode and send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		h.logger.Error("OAuth2: Error encoding metadata: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleAuthorizationServerMetadata handles the standard OAuth 2.0 Authorization Server Metadata endpoint
func (h *OAuth2Handler) HandleAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	// Add CORS headers for browser-based MCP clients like MCP Inspector
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, *")
	w.Header().Set("Access-Control-Max-Age", "86400")

	switch r.Method {
	case "OPTIONS", "HEAD":
		w.WriteHeader(http.StatusOK)
		return
	case "GET":
		// Continue to metadata response
	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Return OAuth 2.0 Authorization Server Metadata (RFC 8414)
	metadata := h.GetAuthorizationServerMetadata()

	// Encode and send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		h.logger.Error("OAuth2: Error encoding Authorization Server metadata: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleProtectedResourceMetadata handles the OAuth 2.0 Protected Resource Metadata endpoint
func (h *OAuth2Handler) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if r.Method != "GET" {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Return OAuth 2.0 Protected Resource Metadata (RFC 9728)
	// Point to authorization server based on mode
	var authServer string
	if h.config.Mode == "proxy" {
		// Proxy mode: MCP server acts as authorization server
		authServer = h.config.MCPURL
	} else {
		// Native mode: Point directly to OAuth provider
		authServer = h.config.Issuer
	}

	metadata := map[string]interface{}{
		"resource":                              h.config.MCPURL,
		"authorization_servers":                 []string{authServer},
		"bearer_methods_supported":              []string{"header"},
		"resource_signing_alg_values_supported": []string{"RS256"},
		"resource_documentation":                fmt.Sprintf("%s/docs", h.config.MCPURL),
		"resource_policy_uri":                   fmt.Sprintf("%s/policy", h.config.MCPURL),
		"resource_tos_uri":                      fmt.Sprintf("%s/tos", h.config.MCPURL),
		"scopes_supported":                      h.config.Scopes,
	}

	// Encode and send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		h.logger.Error("OAuth2: Error encoding Protected Resource metadata: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleRegister handles OAuth dynamic client registration for mcp-remote
func (h *OAuth2Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers for browser-based MCP clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, *")
	w.Header().Set("Access-Control-Max-Age", "86400")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the registration request
	var regRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&regRequest); err != nil {
		h.logger.Error("OAuth2: Failed to parse registration request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("OAuth2: Registration request: %+v", regRequest)

	// Accept any client registration from mcp-remote
	// Return our pre-configured client_id
	response := map[string]interface{}{
		"client_id":                  h.config.ClientID,
		"client_secret":              "", // Public client, no secret
		"client_id_issued_at":        time.Now().Unix(),
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"application_type":           "native",
		"client_name":                regRequest["client_name"],
	}

	// Allow clients to register their own redirect URIs (needed for mcp-remote)
	if redirectUris, ok := regRequest["redirect_uris"]; ok {
		response["redirect_uris"] = redirectUris
		h.logger.Info("OAuth2: Registration allowing client redirect URIs: %v", redirectUris)
	} else if h.config.FixedRedirectURI != "" {
		// Fallback to fixed redirect URI if no client URIs provided
		trimmedURI := strings.TrimSpace(h.config.FixedRedirectURI)
		response["redirect_uris"] = []string{trimmedURI}
		h.logger.Info("OAuth2: Registration response using fixed redirect URI: %s", trimmedURI)
	} else if h.config.RedirectURIs != "" && !strings.Contains(h.config.RedirectURIs, ",") {
		// Backwards-compatible fallback: single RedirectURIs value
		trimmedURI := strings.TrimSpace(h.config.RedirectURIs)
		response["redirect_uris"] = []string{trimmedURI}
		h.logger.Info("OAuth2: Registration response using single redirect URI from RedirectURIs: %s", trimmedURI)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("OAuth2: Failed to encode registration response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleCallbackRedirect handles the /callback redirect for Claude Code compatibility
func (h *OAuth2Handler) HandleCallbackRedirect(w http.ResponseWriter, r *http.Request) {
	// Preserve all query parameters when redirecting
	redirectURL := "/oauth/callback"
	if r.URL.RawQuery != "" {
		redirectURL += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// HandleOIDCDiscovery handles the OIDC discovery endpoint for MCP client compatibility
func (h *OAuth2Handler) HandleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")

	if r.Method != "GET" {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("OAuth2: OIDC discovery request from %s", r.RemoteAddr)

	// Return OIDC Discovery metadata with existing /oauth/ endpoints
	metadata := map[string]interface{}{
		"issuer":                                h.config.MCPURL,
		"authorization_endpoint":                fmt.Sprintf("%s/oauth/authorize", h.config.MCPURL),
		"token_endpoint":                        fmt.Sprintf("%s/oauth/token", h.config.MCPURL),
		"registration_endpoint":                 fmt.Sprintf("%s/oauth/register", h.config.MCPURL),
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"code_challenge_methods_supported":      []string{"plain", "S256"},
		"subject_types_supported":               []string{"public"},
		"scopes_supported":                      h.config.Scopes,
	}

	// Add provider-specific fields
	if h.config.Audience != "" {
		metadata["audience"] = h.config.Audience
	}

	// Add provider-specific signing algorithm information
	switch h.config.Provider {
	case "hmac":
		metadata["id_token_signing_alg_values_supported"] = []string{"HS256"}
	case "okta", "google", "azure":
		metadata["id_token_signing_alg_values_supported"] = []string{"RS256"}
		metadata["jwks_uri"] = fmt.Sprintf("%s/.well-known/jwks.json", h.config.MCPURL)
	}

	h.logger.Info("OAuth2: Returning OIDC discovery metadata for issuer: %s", h.config.MCPURL)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		h.logger.Error("OAuth2: Error encoding OIDC discovery metadata: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetAuthorizationServerMetadata returns the OAuth 2.0 Authorization Server Metadata
// with conditional responses based on OAuth mode
func (h *OAuth2Handler) GetAuthorizationServerMetadata() map[string]interface{} {
	var metadata map[string]interface{}

	if h.config.Mode == "native" {
		// Native mode: Point to OAuth provider directly
		metadata = map[string]interface{}{
			"issuer":                                h.config.Issuer, // OAuth provider issuer
			"response_types_supported":              []string{"code"},
			"response_modes_supported":              []string{"query"},
			"grant_types_supported":                 []string{"authorization_code"},
			"token_endpoint_auth_methods_supported": []string{"none"},
			"code_challenge_methods_supported":      []string{"plain", "S256"},
			"scopes_supported":                      h.config.Scopes,
		}

		// Add provider-specific endpoints
		switch h.config.Provider {
		case "okta":
			metadata["authorization_endpoint"] = fmt.Sprintf("%s/oauth2/v1/authorize", h.config.Issuer)
			metadata["token_endpoint"] = fmt.Sprintf("%s/oauth2/v1/token", h.config.Issuer)
			metadata["registration_endpoint"] = fmt.Sprintf("%s/oauth2/v1/clients", h.config.Issuer)
			metadata["jwks_uri"] = fmt.Sprintf("%s/oauth2/v1/keys", h.config.Issuer)
		case "google":
			metadata["authorization_endpoint"] = "https://accounts.google.com/o/oauth2/v2/auth"
			metadata["token_endpoint"] = "https://oauth2.googleapis.com/token"
			metadata["jwks_uri"] = "https://www.googleapis.com/oauth2/v3/certs"
		case "azure":
			metadata["authorization_endpoint"] = fmt.Sprintf("%s/oauth2/v2.0/authorize", h.config.Issuer)
			metadata["token_endpoint"] = fmt.Sprintf("%s/oauth2/v2.0/token", h.config.Issuer)
			metadata["jwks_uri"] = fmt.Sprintf("%s/discovery/v2.0/keys", h.config.Issuer)
		}
	} else {
		// Proxy mode: Point to MCP server endpoints
		metadata = map[string]interface{}{
			"issuer":                                h.config.MCPURL,
			"authorization_endpoint":                fmt.Sprintf("%s/oauth/authorize", h.config.MCPURL),
			"token_endpoint":                        fmt.Sprintf("%s/oauth/token", h.config.MCPURL),
			"registration_endpoint":                 fmt.Sprintf("%s/oauth/register", h.config.MCPURL),
			"jwks_uri":                              fmt.Sprintf("%s/.well-known/jwks.json", h.config.MCPURL),
			"response_types_supported":              []string{"code"},
			"response_modes_supported":              []string{"query"},
			"grant_types_supported":                 []string{"authorization_code"},
			"token_endpoint_auth_methods_supported": []string{"none"},
			"code_challenge_methods_supported":      []string{"plain", "S256"},
			"scopes_supported":                      h.config.Scopes,
		}
	}

	return metadata
}

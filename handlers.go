package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OAuth2Handler handles OAuth2 flows using the standard library
type OAuth2Handler struct {
	config       *OAuth2Config
	oauth2Config *oauth2.Config
	logger       Logger
}

// GetConfig returns the OAuth2 configuration
func (h *OAuth2Handler) GetConfig() *OAuth2Config {
	return h.config
}

// OAuth2Config holds OAuth2 configuration
type OAuth2Config struct {
	Enabled      bool
	Mode         string // "native" or "proxy"
	Provider     string
	RedirectURIs string
	// FixedRedirectURI is an optional fixed redirect URI used when proxying callbacks
	FixedRedirectURI string

	// OIDC configuration
	Issuer       string
	Audience     string
	ClientID     string
	ClientSecret string
	Scopes       []string // OIDC scopes

	// Server configuration
	MCPHost string
	MCPPort string
	Scheme  string

	// MCPURL is the full URL of the MCP server, used for the resource endpoint in the OAuth 2.0 Protected Resource Metadata endpoint
	MCPURL string

	// Server version
	Version string

	// State signing key for integrity protection
	stateSigningKey []byte
}

// NewOAuth2Handler creates a new OAuth2 handler using the standard library
func NewOAuth2Handler(cfg *OAuth2Config, logger Logger) *OAuth2Handler {
	if logger == nil {
		logger = &defaultLogger{}
	}

	var endpoint oauth2.Endpoint

	// Use OIDC discovery for supported providers, fallback to hardcoded for others
	switch cfg.Provider {
	case "okta", "google", "azure":
		// Use OIDC discovery to get correct endpoints
		if discoveredEndpoint, err := discoverOIDCEndpoints(cfg.Issuer); err != nil {
			logger.Error("OIDC discovery failed for %s provider. Using Okta-style fallback endpoints which may not work for all providers: %v", cfg.Provider, err)
			// Fallback to Okta-style endpoints as they're most common
			endpoint = oauth2.Endpoint{
				AuthURL:  cfg.Issuer + "/oauth2/v1/authorize",
				TokenURL: cfg.Issuer + "/oauth2/v1/token",
			}
		} else {
			endpoint = discoveredEndpoint
		}
	default:
		// For HMAC and unknown providers, use hardcoded endpoints
		endpoint = oauth2.Endpoint{
			AuthURL:  cfg.Issuer + "/oauth2/v1/authorize",
			TokenURL: cfg.Issuer + "/oauth2/v1/token",
		}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		Scopes:       cfg.Scopes,
	}

	// Log client configuration type for debugging
	if cfg.ClientSecret == "" {
		logger.Info("Configuring public client (no client secret)")
	} else {
		logger.Info("Configuring confidential client (with client secret)")
	}

	// Initialize state signing key
	if len(cfg.stateSigningKey) == 0 {
		logger.Warn("No state signing key configured, generating random key (will not persist across restarts)")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			logger.Error("Failed to generate state signing key: %v", err)
			// Use a deterministic fallback (not ideal, but better than nothing)
			cfg.stateSigningKey = []byte("insecure-fallback-key-please-configure-JWT_SECRET")
			logger.Warn("Using insecure fallback key. Please configure JWT_SECRET environment variable.")
		} else {
			cfg.stateSigningKey = key
		}
	}

	return &OAuth2Handler{
		config:       cfg,
		oauth2Config: oauth2Config,
		logger:       logger,
	}
}

// discoverOIDCEndpoints uses OIDC discovery to get the correct authorization and token endpoints
func discoverOIDCEndpoints(issuer string) (oauth2.Endpoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure HTTP client with appropriate timeouts and TLS settings
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Verify TLS certificates
				MinVersion:         tls.VersionTLS12,
			},
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
		},
	}

	// Create OIDC provider with custom HTTP client
	provider, err := oidc.NewProvider(
		oidc.ClientContext(ctx, httpClient),
		issuer,
	)
	if err != nil {
		return oauth2.Endpoint{}, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Return the discovered endpoint
	return provider.Endpoint(), nil
}

// NewOAuth2ConfigFromConfig creates OAuth2 config from generic Config
func NewOAuth2ConfigFromConfig(cfg *Config, version string) *OAuth2Config {
	mcpHost := getEnv("MCP_HOST", "localhost")
	mcpPort := getEnv("MCP_PORT", "8080")

	// Determine scheme based on HTTPS configuration
	scheme := "http"
	if getEnv("HTTPS_CERT_FILE", "") != "" && getEnv("HTTPS_KEY_FILE", "") != "" {
		scheme = "https"
	}

	// Use ServerURL from config if provided, otherwise build from env vars
	mcpURL := cfg.ServerURL
	if mcpURL == "" {
		mcpURL = getEnv("MCP_URL", fmt.Sprintf("%s://%s:%s", scheme, mcpHost, mcpPort))
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	return &OAuth2Config{
		Enabled:          true,
		Mode:             cfg.Mode,
		Provider:         cfg.Provider,
		RedirectURIs:     cfg.RedirectURIs,
		FixedRedirectURI: cfg.FixedRedirectURI,
		Issuer:           cfg.Issuer,
		Audience:         cfg.Audience,
		ClientID:         cfg.ClientID,
		ClientSecret:     cfg.ClientSecret,
		Scopes:           scopes,
		MCPHost:          mcpHost,
		MCPPort:          mcpPort,
		MCPURL:           mcpURL,
		Scheme:           scheme,
		Version:          version,
		stateSigningKey:  cfg.JWTSecret,
	}
}

// HandleJWKS handles the JWKS endpoint for proxy mode
func (h *OAuth2Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	// Defense in depth: Check OAuth mode
	if h.config.Mode == "native" {
		http.Error(w, "JWKS endpoint disabled in native mode", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Proxy JWKS from upstream OAuth provider
	var jwksURL string
	switch h.config.Provider {
	case "okta":
		// Use Okta's standard JWKS path
		jwksURL = fmt.Sprintf("%s/oauth2/v1/keys", h.config.Issuer)
	case "google":
		jwksURL = "https://www.googleapis.com/oauth2/v3/certs"
	case "azure":
		jwksURL = fmt.Sprintf("%s/discovery/v2.0/keys", h.config.Issuer)
	case "hmac":
		// HMAC doesn't use JWKS, return empty key set
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"keys":[]}`))
		return
	default:
		http.Error(w, "JWKS not supported for this provider", http.StatusNotImplemented)
		return
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	// Fetch JWKS from upstream provider
	resp, err := client.Get(jwksURL)
	if err != nil {
		h.logger.Error("OAuth2: Failed to fetch JWKS from %s: %v", jwksURL, err)
		http.Error(w, "Failed to fetch JWKS", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("OAuth2: JWKS endpoint returned status %d", resp.StatusCode)
		http.Error(w, "JWKS endpoint error", http.StatusBadGateway)
		return
	}

	// Copy response headers
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(http.StatusOK)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Error("OAuth2: Failed to proxy JWKS response: %v", err)
	}
}

// HandleAuthorize handles OAuth2 authorization requests with PKCE
func (h *OAuth2Handler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Defense in depth: Check OAuth mode
	if h.config.Mode == "native" {
		http.Error(w, "OAuth proxy disabled in native mode", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract query parameters
	query := r.URL.Query()

	// PKCE parameters from client
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	clientRedirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	clientID := query.Get("client_id")

	h.logger.Info("OAuth2: Authorization request - client_id: %s, redirect_uri: %s, code_challenge: %s",
		clientID, clientRedirectURI, codeChallenge)

	// Determine redirect URI strategy based on configuration
	var redirectURI string
	actualState := state

	if h.config.RedirectURIs != "" {
		// Allowlist mode: Client's URI must be in allowlist, used directly (no proxy)
		if h.isValidRedirectURI(clientRedirectURI) {
			redirectURI = clientRedirectURI
			h.logger.Info("OAuth2: Allowlist mode - using client URI from allowlist: %s", redirectURI)
		} else {
			h.logger.Warn("SECURITY: Redirect URI not in allowlist: %s from %s", clientRedirectURI, r.RemoteAddr)
			h.logger.Info("SECURITY: Will try fixed redirect mode next")
		}
	}

	if redirectURI == "" && h.config.FixedRedirectURI != "" {
		// Fixed redirect mode: Use server's redirect URI to OAuth provider, proxy back to client
		h.logger.Info("OAuth2: Fixed redirect mode - using server URI: %s (will proxy to client: %s)", h.config.FixedRedirectURI, clientRedirectURI)

		// Validate client redirect URI format and security
		if clientRedirectURI == "" {
			h.logger.Warn("SECURITY: Missing client redirect URI")
			http.Error(w, "Missing redirect_uri", http.StatusBadRequest)
			return
		}

		parsedURI, err := url.Parse(clientRedirectURI)
		if err != nil {
			h.logger.Warn("SECURITY: Invalid client redirect URI format: %s", clientRedirectURI)
			http.Error(w, "Invalid redirect_uri format", http.StatusBadRequest)
			return
		}

		// Additional security checks for client redirect URI
		if parsedURI.Scheme != "http" && parsedURI.Scheme != "https" {
			h.logger.Warn("SECURITY: Invalid redirect URI scheme: %s (must be http or https)", parsedURI.Scheme)
			http.Error(w, "Invalid redirect_uri scheme", http.StatusBadRequest)
			return
		}

		// Enforce HTTPS for non-localhost URIs
		if parsedURI.Scheme == "http" && !isLocalhostURI(clientRedirectURI) {
			h.logger.Warn("SECURITY: HTTP redirect URI not allowed for non-localhost: %s", clientRedirectURI)
			http.Error(w, "HTTPS required for non-localhost redirect_uri", http.StatusBadRequest)
			return
		}

		// Prevent fragment in redirect URI (OAuth 2.0 spec)
		if parsedURI.Fragment != "" {
			h.logger.Warn("SECURITY: Redirect URI contains fragment: %s", clientRedirectURI)
			http.Error(w, "redirect_uri must not contain fragment", http.StatusBadRequest)
			return
		}

		// Security: For fixed redirect mode, only allow localhost or loopback addresses
		// This prevents open redirect attacks while still supporting development tools
		if !isLocalhostURI(clientRedirectURI) {
			h.logger.Warn("SECURITY: Fixed redirect mode only allows localhost URIs, rejecting: %s from %s", clientRedirectURI, r.RemoteAddr)
			http.Error(w, "Fixed redirect mode only allows localhost redirect URIs for security. Use allowlist mode for production.", http.StatusBadRequest)
			return
		}
		redirectURI = strings.TrimSpace(h.config.FixedRedirectURI)
		h.logger.Info("OAuth2: Validated localhost redirect URI for proxy: %s", clientRedirectURI)
		// For fixed redirect mode, create signed state with client redirect URI
		// Create state data with redirect URI
		stateData := map[string]string{
			"state":    state,
			"redirect": clientRedirectURI,
		}

		// Sign state for integrity protection
		signedState, err := h.signState(stateData)
		if err != nil {
			h.logger.Error("OAuth2: Failed to sign state: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		actualState = signedState
		h.logger.Info("OAuth2: Signed state for proxy callback (length: %d)", len(signedState))
	}

	if redirectURI == "" {
		// No configuration: Reject for security
		h.logger.Warn("SECURITY: No redirect URIs configured, rejecting: %s from %s", clientRedirectURI, r.RemoteAddr)
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return
	}

	// Update OAuth2 config with redirect URI
	h.oauth2Config.RedirectURL = redirectURI

	// Create authorization URL
	authURL := h.oauth2Config.AuthCodeURL(actualState, oauth2.AccessTypeOffline)

	// Add PKCE parameters to the URL if provided
	if codeChallenge != "" {
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			h.logger.Error("OAuth2: Failed to parse auth URL: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		query := parsedURL.Query()
		query.Set("code_challenge", codeChallenge)
		query.Set("code_challenge_method", codeChallengeMethod)

		parsedURL.RawQuery = query.Encode()
		authURL = parsedURL.String()
	}

	h.logger.Info("OAuth2: Redirecting to authorization URL: %s", authURL)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles OAuth2 callback
func (h *OAuth2Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Defense in depth: Check OAuth mode
	if h.config.Mode == "native" {
		http.Error(w, "OAuth proxy disabled in native mode", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	h.logger.Info("OAuth2: Callback received - code: %s, state: %s, error: %s",
		truncateString(code, 10), state, errorParam)

	// Handle OAuth errors
	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		h.logger.Error("OAuth2: Authorization error: %s - %s", errorParam, errorDesc)
		http.Error(w, fmt.Sprintf("Authorization failed: %s", errorDesc), http.StatusBadRequest)
		return
	}

	if code == "" {
		h.logger.Error("OAuth2: No authorization code received")
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// If using fixed redirect URI, handle proxy callback
	if h.config.FixedRedirectURI != "" {
		// Verify and decode signed state parameter
		stateData, err := h.verifyState(state)
		if err != nil {
			h.logger.Warn("SECURITY: State verification failed: %v", err)
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Extract original state and redirect URI
		originalState, hasState := stateData["state"]
		originalRedirectURI, hasRedirect := stateData["redirect"]

		if hasState && hasRedirect {
			// Re-validate redirect URI for defense in depth
			// Even though state is HMAC-signed, validate the redirect URI is localhost
			if !isLocalhostURI(originalRedirectURI) {
				h.logger.Warn("SECURITY: Callback redirect URI is not localhost (possible key compromise): %s", originalRedirectURI)
				http.Error(w, "Invalid redirect URI in state", http.StatusBadRequest)
				return
			}

			h.logger.Info("OAuth2: State verified, proxying callback to localhost client: %s", originalRedirectURI)

			// Build proxy callback URL
			proxyURL := fmt.Sprintf("%s?code=%s&state=%s", originalRedirectURI, code, originalState)
			http.Redirect(w, r, proxyURL, http.StatusFound)
			return
		}

		h.logger.Error("OAuth2: State missing required fields")
		http.Error(w, "Invalid state format", http.StatusBadRequest)
		return
	}

	// For non-fixed redirect mode or as fallback, show success page
	h.showSuccessPage(w, code, state)
}

// HandleToken handles OAuth2 token exchange
func (h *OAuth2Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	// Defense in depth: Check OAuth mode
	if h.config.Mode == "native" {
		http.Error(w, "OAuth proxy disabled in native mode", http.StatusNotFound)
		return
	}

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

	h.logger.Info("OAuth2: Token exchange request from %s", r.RemoteAddr)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("OAuth2: Failed to parse form: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Extract parameters
	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	refreshToken := r.FormValue("refresh_token")
	clientRedirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeVerifier := r.FormValue("code_verifier")

	h.logger.Info("OAuth2: Token request - grant_type: %s, client_id: %s, redirect_uri: %s, code: %s",
		grantType, clientID, clientRedirectURI, truncateString(code, 10))

	var token *oauth2.Token
	var err error

	switch grantType {
	case "authorization_code":
		// Validate parameters for authorization_code flow
		if code == "" {
			h.logger.Error("OAuth2: Missing authorization code")
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}

		// Set redirect URI for token exchange
		redirectURI := clientRedirectURI
		if h.config.FixedRedirectURI != "" && !h.isValidRedirectURI(clientRedirectURI) {
			redirectURI = strings.TrimSpace(h.config.FixedRedirectURI)
			h.logger.Info("OAuth2: Token exchange using fixed redirect URI: %s", redirectURI)
		}

		h.oauth2Config.RedirectURL = redirectURI

		// For PKCE, we need to manually add the code_verifier to the token exchange
		// Since oauth2 library doesn't support PKCE directly, we'll use a custom approach
		ctx := context.Background()

		// Create custom HTTP client for token exchange with PKCE
		if codeVerifier != "" {
			// Create a custom client that adds code_verifier to the token request
			customClient := &http.Client{
				Transport: &pkceTransport{
					base:         http.DefaultTransport,
					codeVerifier: codeVerifier,
				},
			}
			ctx = context.WithValue(ctx, oauth2.HTTPClient, customClient)
		}

		// Exchange code for tokens
		token, err = h.oauth2Config.Exchange(ctx, code)
		if err != nil {
			h.logger.Error("OAuth2: Token exchange failed: %v", err)
			http.Error(w, "Token exchange failed", http.StatusInternalServerError)
			return
		}
	case "refresh_token":
		// Validate parameters for refresh_token flow
		if refreshToken == "" {
			h.logger.Error("OAuth2: Missing refresh token")
			http.Error(w, "Missing refresh token", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		src := h.oauth2Config.TokenSource(ctx, &oauth2.Token{
			RefreshToken: refreshToken,
		})

		token, err = src.Token()
		if err != nil {
			h.logger.Error("OAuth2: Refresh token exchange failed: %v", err)
			http.Error(w, "Token refresh failed", http.StatusBadGateway)
			return
		}
	default:
		h.logger.Error("OAuth2: Unsupported grant type: %s", grantType)
		http.Error(w, "Unsupported grant type", http.StatusBadRequest)
		return
	}

	h.logger.Info("OAuth2: Token exchange successful")

	// Build response
	response := map[string]interface{}{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
		"expires_in":   int(time.Until(token.Expiry).Seconds()),
	}

	// Add optional fields
	if token.RefreshToken != "" {
		response["refresh_token"] = token.RefreshToken
	}

	// Add ID token if present
	if idToken, ok := token.Extra("id_token").(string); ok {
		response["id_token"] = idToken
	}

	// Add scope if present
	if scope, ok := token.Extra("scope").(string); ok {
		response["scope"] = scope
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("OAuth2: Failed to encode token response: %v", err)
	}
}

// showSuccessPage displays a success page after OAuth completion
func (h *OAuth2Handler) showSuccessPage(w http.ResponseWriter, code, state string) {
	// Log authorization details server-side (truncated for security)
	h.logger.Info("OAuth2: Authorization successful - code: %s, state: %s",
		truncateString(code, 10), truncateString(state, 10))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>OAuth2 Success</title>
		</head>
		<body>
			<h2>Authentication Successful!</h2>
			<p>You have been successfully authenticated.</p>
			<p>You can now close this window and return to your application.</p>
		</body>
		</html>`)
}

// truncateString safely truncates a string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// pkceTransport adds PKCE code_verifier to token exchange requests
type pkceTransport struct {
	base         http.RoundTripper
	codeVerifier string
}

// RoundTrip implements the RoundTripper interface
func (p *pkceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only modify POST requests to token endpoint
	if req.Method == "POST" && strings.Contains(req.URL.Path, "/token") {
		// Read the existing body
		defer func() {
			if closeErr := req.Body.Close(); closeErr != nil {
				// Note: pkceTransport doesn't have access to h.logger, using standard log
				log.Printf("Warning: failed to close request body: %v", closeErr)
			}
		}()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		// Parse the form data
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}

		// Add code_verifier if not already present
		if values.Get("code_verifier") == "" && p.codeVerifier != "" {
			values.Set("code_verifier", p.codeVerifier)
		}

		// Create new body with code_verifier
		newBody := strings.NewReader(values.Encode())
		req.Body = io.NopCloser(newBody)
		req.ContentLength = int64(len(values.Encode()))
	}

	return p.base.RoundTrip(req)
}

// getEnv gets environment variable with default value
func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

// signState signs state data with HMAC-SHA256 for integrity protection
func (h *OAuth2Handler) signState(stateData map[string]string) (string, error) {
	// Create deterministic string for signing
	dataToSign := ""
	if state, ok := stateData["state"]; ok {
		dataToSign += "state=" + state + "&"
	}
	if redirect, ok := stateData["redirect"]; ok {
		dataToSign += "redirect=" + redirect
	}

	// Create HMAC signature
	mac := hmac.New(sha256.New, h.config.stateSigningKey)
	mac.Write([]byte(dataToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Add signature to state data
	stateData["sig"] = signature
	signedData, err := json.Marshal(stateData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal signed state: %w", err)
	}

	// Base64 encode for URL safety
	return base64.URLEncoding.EncodeToString(signedData), nil
}

// verifyState verifies and decodes HMAC-signed state parameter
func (h *OAuth2Handler) verifyState(encodedState string) (map[string]string, error) {
	// Base64 decode
	decodedState, err := base64.URLEncoding.DecodeString(encodedState)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	// Unmarshal state data
	var stateData map[string]string
	if err := json.Unmarshal(decodedState, &stateData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Extract signature
	receivedSig, ok := stateData["sig"]
	if !ok {
		return nil, fmt.Errorf("state missing signature")
	}
	delete(stateData, "sig") // Remove for verification

	// Recalculate signature using same deterministic approach
	dataToSign := ""
	if state, ok := stateData["state"]; ok {
		dataToSign += "state=" + state + "&"
	}
	if redirect, ok := stateData["redirect"]; ok {
		dataToSign += "redirect=" + redirect
	}

	mac := hmac.New(sha256.New, h.config.stateSigningKey)
	mac.Write([]byte(dataToSign))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Verify signature using constant-time comparison
	if !hmac.Equal([]byte(receivedSig), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid state signature - possible tampering detected")
	}

	return stateData, nil
}

// isLocalhostURI checks if URI is localhost for development
func isLocalhostURI(uri string) bool {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return false
	}

	hostname := strings.ToLower(parsedURI.Hostname())
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

// isValidRedirectURI validates redirect URI against allowlist for security
func (h *OAuth2Handler) isValidRedirectURI(uri string) bool {
	if h.config.RedirectURIs == "" {
		return false
	}

	// Parse allowlist
	allowedURIs := strings.Split(h.config.RedirectURIs, ",")
	for _, allowed := range allowedURIs {
		allowed = strings.TrimSpace(allowed)
		if allowed != "" && uri == allowed {
			return true
		}
	}

	return false
}

// validateOAuthParams performs basic input validation to prevent abuse
func (h *OAuth2Handler) validateOAuthParams(r *http.Request) error {
	// Basic length validation to prevent abuse
	if code := r.FormValue("code"); len(code) > 512 {
		return fmt.Errorf("invalid code parameter length")
	}
	if state := r.FormValue("state"); len(state) > 256 {
		return fmt.Errorf("invalid state parameter length")
	}
	if challenge := r.FormValue("code_challenge"); len(challenge) > 256 {
		return fmt.Errorf("invalid code_challenge parameter length")
	}
	return nil
}

// addSecurityHeaders adds essential security headers for OAuth endpoints
func (h *OAuth2Handler) addSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store, no-cache, max-age=0")
	w.Header().Set("Pragma", "no-cache")
}

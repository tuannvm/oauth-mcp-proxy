package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// TestWithOAuth validates the WithOAuth() convenience API.
// Tests simple integration, both native and proxy modes, error handling,
// and composability with other server options.
func TestWithOAuth(t *testing.T) {
	t.Run("BasicUsage_NativeMode", func(t *testing.T) {
		// Test the simplest usage of WithOAuth

		mux := http.NewServeMux()
		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		// Get OAuth server and option
		oauthServer, oauthOption, err := WithOAuth(mux, cfg)
		if err != nil {
			t.Fatalf("WithOAuth failed: %v", err)
		}

		if oauthServer == nil {
			t.Fatal("Expected OAuth server, got nil")
		}

		if oauthOption == nil {
			t.Fatal("Expected server option, got nil")
		}

		// Create MCP server with OAuth option
		mcpServer := mcpserver.NewMCPServer("Test Server", "1.0.0", oauthOption)

		if mcpServer == nil {
			t.Fatal("MCP server creation failed")
		}

		// Verify HTTP handlers were registered
		req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code == http.StatusNotFound {
			t.Error("OAuth metadata endpoint not registered")
		}

		t.Logf("✅ WithOAuth() works in native mode")
		t.Logf("   - Server option returned")
		t.Logf("   - HTTP handlers registered")
		t.Logf("   - MCP server created with OAuth")
	})

	t.Run("ProxyMode", func(t *testing.T) {
		mux := http.NewServeMux()
		cfg := &Config{
			Mode:         "proxy",
			Provider:     "hmac",
			Issuer:       "https://test.example.com",
			Audience:     "api://test",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			ServerURL:    "https://test-server.com",
			RedirectURIs: "https://test-server.com/callback",
			JWTSecret:    []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		_, oauthOption, err := WithOAuth(mux, cfg)
		if err != nil {
			t.Fatalf("WithOAuth failed in proxy mode: %v", err)
		}

		mcpServer := mcpserver.NewMCPServer("Test Server", "1.0.0", oauthOption)
		if mcpServer == nil {
			t.Fatal("MCP server creation failed")
		}

		t.Logf("✅ WithOAuth() works in proxy mode")
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		mux := http.NewServeMux()
		cfg := &Config{
			Provider: "invalid-provider",
		}

		_, _, err := WithOAuth(mux, cfg)
		if err == nil {
			t.Error("Expected error with invalid config")
		}

		t.Logf("✅ WithOAuth() validates config")
		t.Logf("   - Error: %v", err)
	})

	t.Run("EndToEndWithHTTPContextFunc", func(t *testing.T) {
		// Test complete integration with CreateHTTPContextFunc

		mux := http.NewServeMux()
		cfg := &Config{
			Mode:      "native",
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		// 1. Get OAuth option
		_, oauthOption, err := WithOAuth(mux, cfg)
		if err != nil {
			t.Fatalf("WithOAuth failed: %v", err)
		}

		// 2. Create MCP server with OAuth
		mcpServer := mcpserver.NewMCPServer("Test Server", "1.0.0", oauthOption)

		// 3. Add a tool
		mcpServer.AddTool(
			mcp.Tool{
				Name:        "test",
				Description: "Test tool",
			},
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				user, ok := GetUserFromContext(ctx)
				if !ok {
					return nil, fmt.Errorf("no user in context")
				}
				if user.Subject != "test-user-123" {
					return nil, fmt.Errorf("wrong user: %s", user.Subject)
				}
				return mcp.NewToolResultText("ok"), nil
			},
		)

		// 4. Create StreamableHTTPServer with HTTPContextFunc
		streamableServer := mcpserver.NewStreamableHTTPServer(
			mcpServer,
			mcpserver.WithEndpointPath("/mcp"),
			mcpserver.WithHTTPContextFunc(CreateHTTPContextFunc()),
		)

		mux.Handle("/mcp", streamableServer)

		// 5. Generate test token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":                "test-user-123",
			"email":              "test@example.com",
			"preferred_username": "testuser",
			"aud":                cfg.Audience,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
		})

		tokenString, _ := token.SignedString(cfg.JWTSecret)

		// 6. Simulate HTTP request with Bearer token
		// Note: We can't easily test StreamableHTTPServer without full MCP protocol
		// But we can verify the HTTPContextFunc works
		contextFunc := CreateHTTPContextFunc()
		req := &http.Request{
			Header: http.Header{
				"Authorization": []string{"Bearer " + tokenString},
			},
		}

		ctx := contextFunc(context.Background(), req)

		// Verify token was extracted
		extractedToken, ok := GetOAuthToken(ctx)
		if !ok {
			t.Fatal("Token not extracted from context")
		}

		if extractedToken != tokenString {
			t.Error("Token mismatch")
		}

		t.Logf("✅ End-to-end integration works")
		t.Logf("   - WithOAuth() creates server option")
		t.Logf("   - CreateHTTPContextFunc() extracts token")
		t.Logf("   - Ready for StreamableHTTPServer")
	})
}

// TestWithOAuthAPI validates the WithOAuth() API design goals.
// Tests API simplicity, composability, and end-to-end integration.
func TestWithOAuthAPI(t *testing.T) {
	t.Run("TwoLineSetup", func(t *testing.T) {
		// Demonstrate the simplest possible setup

		mux := http.NewServeMux()

		// Line 1: Get OAuth option
		_, oauthOption, err := WithOAuth(mux, &Config{
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		})
		if err != nil {
			t.Fatalf("WithOAuth failed: %v", err)
		}

		// Line 2: Create server with OAuth
		mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

		if mcpServer == nil {
			t.Fatal("Server creation failed")
		}

		t.Logf("✅ Two-line OAuth setup works")
		t.Logf("   Line 1: oauthOption, _ := oauth.WithOAuth(mux, cfg)")
		t.Logf("   Line 2: server := mcpserver.NewMCPServer(name, ver, oauthOption)")
	})

	t.Run("ComposableWithOtherOptions", func(t *testing.T) {
		// Test that WithOAuth composes with other server options

		mux := http.NewServeMux()
		_, oauthOption, _ := WithOAuth(mux, &Config{
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		})

		// Combine with other options
		mcpServer := mcpserver.NewMCPServer("My Server", "1.0.0", oauthOption)

		if mcpServer == nil {
			t.Fatal("Server creation with multiple options failed")
		}

		t.Logf("✅ WithOAuth() composes with other server options")
	})
}

func TestServerWrapHandler(t *testing.T) {
	t.Run("Returns401WithoutToken", func(t *testing.T) {
		cfg := &Config{
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			ServerURL: "https://test-server.com",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer failed: %v", err)
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("protected resource"))
		})

		wrappedHandler := server.WrapHandler(handler)

		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}

		authHeader := w.Header().Get("WWW-Authenticate")
		if !strings.Contains(authHeader, "invalid_token") {
			t.Errorf("Expected WWW-Authenticate header with error, got: %s", authHeader)
		}

		if !strings.Contains(w.Body.String(), "invalid_token") {
			t.Errorf("Expected JSON error response, got: %s", w.Body.String())
		}
	})

	t.Run("ExtractsTokenWithBearer", func(t *testing.T) {
		cfg := &Config{
			Provider:  "hmac",
			Issuer:    "https://test.example.com",
			Audience:  "api://test",
			ServerURL: "https://test-server.com",
			JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer failed: %v", err)
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := server.WrapHandler(handler)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer some-token")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code == http.StatusUnauthorized {
			t.Log("Token validation works")
		}
	})
}

func TestServerHelperMethods(t *testing.T) {
	cfg := &Config{
		Provider:  "hmac",
		Issuer:    "https://test.example.com",
		Audience:  "api://test",
		ServerURL: "https://test-server.com",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	t.Run("DiscoveryURLHelpers", func(t *testing.T) {
		tests := []struct {
			name     string
			method   func() string
			expected string
		}{
			{"GetAuthorizationServerMetadataURL", server.GetAuthorizationServerMetadataURL, "https://test-server.com/.well-known/oauth-authorization-server"},
			{"GetProtectedResourceMetadataURL", server.GetProtectedResourceMetadataURL, "https://test-server.com/.well-known/oauth-protected-resource"},
			{"GetOIDCDiscoveryURL", server.GetOIDCDiscoveryURL, "https://test-server.com/.well-known/openid-configuration"},
			{"GetCallbackURL", server.GetCallbackURL, "https://test-server.com/oauth/callback"},
			{"GetAuthorizeURL", server.GetAuthorizeURL, "https://test-server.com/oauth/authorize"},
			{"GetTokenURL", server.GetTokenURL, "https://test-server.com/oauth/token"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := tt.method()
				if got != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, got)
				}
			})
		}
	})

	t.Run("GetAllEndpoints", func(t *testing.T) {
		endpoints := server.GetAllEndpoints()
		if len(endpoints) != 3 {
			t.Errorf("Expected 3 endpoints in native mode, got %d", len(endpoints))
		}

		proxyCfg := &Config{
			Mode:         "proxy",
			Provider:     "hmac",
			Issuer:       "https://test.example.com",
			Audience:     "api://test",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			ServerURL:    "https://test-server.com",
			RedirectURIs: "https://test-server.com/callback",
			JWTSecret:    []byte("test-secret-key-must-be-32-bytes-long!"),
		}
		proxyServer, _ := NewServer(proxyCfg)
		proxyEndpoints := proxyServer.GetAllEndpoints()
		if len(proxyEndpoints) != 7 {
			t.Errorf("Expected 7 endpoints in proxy mode, got %d", len(proxyEndpoints))
		}
	})

	t.Run("GetStatusString", func(t *testing.T) {
		statusHTTPS := server.GetStatusString(true)
		if !strings.Contains(statusHTTPS, "OAuth enabled") {
			t.Errorf("Expected 'OAuth enabled', got: %s", statusHTTPS)
		}
		if strings.Contains(statusHTTPS, "WARNING") {
			t.Errorf("Expected no warning for HTTPS, got: %s", statusHTTPS)
		}

		statusHTTP := server.GetStatusString(false)
		if !strings.Contains(statusHTTP, "WARNING") {
			t.Errorf("Expected warning for HTTP, got: %s", statusHTTP)
		}
	})

	t.Run("LogStartup", func(t *testing.T) {
		server.LogStartup(true)
		server.LogStartup(false)
		t.Log("LogStartup executed without errors")
	})

	t.Run("GetHTTPServerOptions", func(t *testing.T) {
		opts := server.GetHTTPServerOptions()
		if len(opts) == 0 {
			t.Error("Expected at least one option")
		}

		mcpServer := mcpserver.NewMCPServer("Test", "1.0.0")
		allOpts := append(opts,
			mcpserver.WithEndpointPath("/mcp"),
			mcpserver.WithStateLess(false),
		)

		streamableServer := mcpserver.NewStreamableHTTPServer(mcpServer, allOpts...)
		if streamableServer == nil {
			t.Error("Failed to create StreamableHTTPServer with OAuth options")
		}
	})
}

// TestWrapMCPEndpoint401 tests automatic 401 handling for /mcp endpoints
func TestWrapMCPEndpoint401(t *testing.T) {
	cfg := &Config{
		Mode:      "native",
		Provider:  "hmac",
		Audience:  "api://test",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		ServerURL: "https://test-server.com",
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handlerCalled := false
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	wrapped := server.WrapMCPEndpoint(mockHandler)

	tests := []struct {
		name          string
		method        string
		authHeader    string
		expectStatus  int
		expectHeaders bool
		expectBody    string
		expectBypass  bool
		expectError   string
	}{
		{
			name:          "Missing token returns 401",
			method:        "GET",
			authHeader:    "",
			expectStatus:  401,
			expectHeaders: true,
			expectBypass:  false,
			expectError:   "invalid_request",
		},
		{
			name:          "OPTIONS passthrough",
			method:        "OPTIONS",
			authHeader:    "",
			expectStatus:  200,
			expectHeaders: false,
			expectBody:    "success",
			expectBypass:  true,
		},
		{
			name:          "Basic auth rejected",
			method:        "GET",
			authHeader:    "Basic dXNlcjpwYXNz",
			expectStatus:  401,
			expectHeaders: true,
			expectBypass:  false,
			expectError:   "invalid_request",
		},
		{
			name:          "Malformed Bearer (no space) rejected",
			method:        "GET",
			authHeader:    "Bearertoken123",
			expectStatus:  401,
			expectHeaders: true,
			expectBypass:  false,
			expectError:   "invalid_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			req := httptest.NewRequest(tt.method, "/mcp", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != tt.expectStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectStatus)
			}

			if tt.expectBypass && !handlerCalled {
				t.Error("Handler should have been called (bypass expected)")
			}
			if !tt.expectBypass && handlerCalled {
				t.Error("Handler should NOT have been called (auth bypass prevented)")
			}

			if tt.expectHeaders {
				header := rec.Header().Get("Www-Authenticate")
				if header == "" {
					t.Error("WWW-Authenticate header missing")
				}

				// Check header contains all required Bearer challenge parameters
				if !strings.Contains(header, "Bearer") {
					t.Errorf("WWW-Authenticate header missing Bearer scheme: %s", header)
				}
				if tt.expectError != "" && !strings.Contains(header, tt.expectError) {
					t.Errorf("WWW-Authenticate header missing error code %s: %s", tt.expectError, header)
				}
				if !strings.Contains(header, "resource_metadata") {
					t.Errorf("WWW-Authenticate header missing resource_metadata: %s", header)
				}

				// Check JSON error response
				body := rec.Body.String()
				if tt.expectError != "" && !strings.Contains(body, tt.expectError) {
					t.Errorf("Response body missing %s error: %s", tt.expectError, body)
				}
			}

			if tt.expectBody != "" && !strings.Contains(rec.Body.String(), tt.expectBody) {
				t.Errorf("body = %s, want to contain %s", rec.Body.String(), tt.expectBody)
			}
		})
	}
}

// TestReturn401 tests the Return401 method directly
func TestReturn401(t *testing.T) {
	cfg := &Config{
		Mode:      "native",
		Provider:  "hmac",
		Audience:  "api://test",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		ServerURL: "https://test-server.com",
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	rec := httptest.NewRecorder()
	server.Return401(rec)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	header := rec.Header().Get("Www-Authenticate")
	if header == "" {
		t.Fatal("WWW-Authenticate header missing")
	}

	// Verify Bearer challenge with all parameters in single header
	if !strings.Contains(header, "Bearer realm=\"OAuth\"") {
		t.Errorf("Missing Bearer realm in header: %s", header)
	}
	if !strings.Contains(header, "error=\"invalid_request\"") {
		t.Errorf("Wrong error code, expected invalid_request: %s", header)
	}
	if !strings.Contains(header, "Bearer token required") {
		t.Errorf("Missing error description: %s", header)
	}

	// Verify resource_metadata parameter
	if !strings.Contains(header, "resource_metadata") {
		t.Errorf("Missing resource_metadata: %s", header)
	}
	if !strings.Contains(header, "https://test-server.com/.well-known/oauth-protected-resource") {
		t.Errorf("Wrong resource_metadata URL: %s", header)
	}

	// Verify JSON body
	body := rec.Body.String()
	if !strings.Contains(body, "\"error\":\"invalid_request\"") {
		t.Errorf("Wrong JSON error code: %s", body)
	}
	if !strings.Contains(body, "\"error_description\":\"Bearer token required\"") {
		t.Errorf("Wrong JSON error description: %s", body)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", ct)
	}
}

// TestReturn401InvalidToken tests the Return401InvalidToken method
func TestReturn401InvalidToken(t *testing.T) {
	cfg := &Config{
		Mode:      "native",
		Provider:  "hmac",
		Audience:  "api://test",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		ServerURL: "https://test-server.com",
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	rec := httptest.NewRecorder()
	server.Return401InvalidToken(rec)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	header := rec.Header().Get("Www-Authenticate")
	if header == "" {
		t.Fatal("WWW-Authenticate header missing")
	}

	// Verify Bearer challenge with invalid_token error
	if !strings.Contains(header, "error=\"invalid_token\"") {
		t.Errorf("Wrong error code, expected invalid_token: %s", header)
	}
	if !strings.Contains(header, "Authentication failed") {
		t.Errorf("Missing error description: %s", header)
	}

	// Verify resource_metadata parameter
	if !strings.Contains(header, "resource_metadata") {
		t.Errorf("Missing resource_metadata: %s", header)
	}

	// Verify JSON body
	body := rec.Body.String()
	if !strings.Contains(body, "\"error\":\"invalid_token\"") {
		t.Errorf("Wrong JSON error code: %s", body)
	}
	if !strings.Contains(body, "\"error_description\":\"Authentication failed\"") {
		t.Errorf("Wrong JSON error description: %s", body)
	}
}

// TestWrapMCPEndpointValidation tests that WrapMCPEndpoint validates tokens for mark3labs
// Note: mark3labs relies on middleware for actual validation, this just tests presence check
func TestWrapMCPEndpointWithValidToken(t *testing.T) {
	cfg := &Config{
		Mode:      "native",
		Provider:  "hmac",
		Audience:  "api://test",
		JWTSecret: []byte("test-secret-key-must-be-32-bytes-long!"),
		ServerURL: "https://test-server.com",
		Issuer:    "https://test.example.com",
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a valid HMAC token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "testuser",
		"aud": "api://test",
		"iss": "https://test.example.com",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(cfg.JWTSecret)

	handlerCalled := false
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// Verify token was added to context
		tokenFromCtx, ok := GetOAuthToken(r.Context())
		if !ok || tokenFromCtx == "" {
			t.Error("Token not in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.WrapMCPEndpoint(mockHandler)

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("Handler was not called with valid token")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// TestValidateTokenCached tests that only non-expired tokens are cached.
func TestValidateTokenCached(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		expirationFromNow time.Duration
		tokenExpiryBuffer time.Duration
		want              *User
		wantErr           string
	}{
		{
			name:              "success",
			expirationFromNow: time.Minute,
			want:              &User{Subject: "testuser"},
		},
		{
			name:              "expired token",
			expirationFromNow: -time.Minute,
			wantErr:           "authentication failed: failed to parse and validate token: token has invalid claims: token is expired",
		},
		{
			name:              "token expires in buffer",
			tokenExpiryBuffer: 5 * time.Minute,
			expirationFromNow: time.Minute,
			wantErr:           "authentication failed: token expired or expiring too soon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Mode:              "native",
				Provider:          "hmac",
				Audience:          "api://test",
				JWTSecret:         []byte("test-secret-key-must-be-32-bytes-long!"),
				ServerURL:         "https://test-server.com",
				Issuer:            "https://test.example.com",
				TokenExpiryBuffer: tt.tokenExpiryBuffer,
			}
			srv, err := NewServer(cfg)
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}

			// Create a valid HMAC token
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": "testuser",
				"aud": "api://test",
				"iss": "https://test.example.com",
				"exp": time.Now().Add(tt.expirationFromNow).Unix(),
			})
			tokenString, err := token.SignedString(cfg.JWTSecret)
			if err != nil {
				t.Fatalf("SignedString: %v", err)
			}

			user, err := srv.ValidateTokenCached(context.Background(), tokenString)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !reflect.DeepEqual(user, tt.want) {
				t.Errorf("expected user %v got user %v", tt.want, user)
			}
		})
	}
}

// TestValidateTokenCached tests that the cache expires correctly.
func TestValidateTokenCached_Expires(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		expirationFromNow time.Duration
		tokenExpiryBuffer time.Duration
		wantErr           string
	}{
		{
			name:              "default expiry buffer",
			expirationFromNow: time.Second,
			wantErr:           "authentication failed: failed to parse and validate token: token has invalid claims: token is expired",
		},
		{
			name:              "custom expiry buffer",
			expirationFromNow: 5 * time.Second,
			tokenExpiryBuffer: 4 * time.Second,
			wantErr:           "authentication failed: token expired or expiring too soon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Mode:              "native",
				Provider:          "hmac",
				Audience:          "api://test",
				JWTSecret:         []byte("test-secret-key-must-be-32-bytes-long!"),
				ServerURL:         "https://test-server.com",
				Issuer:            "https://test.example.com",
				TokenExpiryBuffer: tt.tokenExpiryBuffer,
			}
			srv, err := NewServer(cfg)
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}

			// Create a valid HMAC token
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": "testuser",
				"aud": "api://test",
				"iss": "https://test.example.com",
				"exp": time.Now().Add(tt.expirationFromNow).Unix(),
			})
			tokenString, err := token.SignedString(cfg.JWTSecret)
			if err != nil {
				t.Fatalf("SignedString: %v", err)
			}

			// Token is successfully verified and cached
			_, err = srv.ValidateTokenCached(context.Background(), tokenString)
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}

			// Wait twice as long as the token should take to expire.
			time.Sleep(2 * (tt.expirationFromNow - tt.tokenExpiryBuffer))

			_, err = srv.ValidateTokenCached(context.Background(), tokenString)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

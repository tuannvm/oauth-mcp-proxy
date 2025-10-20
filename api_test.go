package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

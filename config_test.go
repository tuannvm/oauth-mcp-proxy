package oauth

import (
	"os"
	"testing"
)

func TestConfigBuilder(t *testing.T) {
	tests := []struct {
		name        string
		buildFunc   func() (*Config, error)
		wantErr     bool
		wantURL     string
		wantMode    string
		wantProvider string
	}{
		{
			name: "basic HTTP config",
			buildFunc: func() (*Config, error) {
				return NewConfigBuilder().
					WithProvider("hmac").
					WithAudience("test-audience").
					WithJWTSecret([]byte("secret")).
					WithHost("example.com").
					WithPort("8080").
					Build()
			},
			wantURL:     "http://example.com:8080",
			wantMode:    "native",
			wantProvider: "hmac",
		},
		{
			name: "HTTPS config with TLS",
			buildFunc: func() (*Config, error) {
				return NewConfigBuilder().
					WithProvider("okta").
					WithIssuer("https://okta.example.com").
					WithAudience("test-audience").
					WithHost("secure.example.com").
					WithPort("443").
					WithTLS(true).
					Build()
			},
			wantURL:     "https://secure.example.com:443",
			wantMode:    "native",
			wantProvider: "okta",
		},
		{
			name: "explicit ServerURL overrides components",
			buildFunc: func() (*Config, error) {
				return NewConfigBuilder().
					WithProvider("hmac").
					WithAudience("test-audience").
					WithJWTSecret([]byte("secret")).
					WithHost("example.com").
					WithPort("8080").
					WithServerURL("https://override.example.com:9000").
					Build()
			},
			wantURL:     "https://override.example.com:9000",
			wantMode:    "native",
			wantProvider: "hmac",
		},
		{
			name: "proxy mode config",
			buildFunc: func() (*Config, error) {
				return NewConfigBuilder().
					WithMode("proxy").
					WithProvider("okta").
					WithIssuer("https://okta.example.com").
					WithAudience("test-audience").
					WithClientID("client-123").
					WithClientSecret("secret-456").
					WithRedirectURIs("http://localhost:8080/callback").
					WithHost("localhost").
					WithPort("8080").
					Build()
			},
			wantURL:     "http://localhost:8080",
			wantMode:    "proxy",
			wantProvider: "okta",
		},
		{
			name: "missing required fields returns error",
			buildFunc: func() (*Config, error) {
				return NewConfigBuilder().
					WithProvider("hmac").
					Build()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := tt.buildFunc()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cfg.ServerURL != tt.wantURL {
				t.Errorf("ServerURL = %v, want %v", cfg.ServerURL, tt.wantURL)
			}
			if cfg.Mode != tt.wantMode {
				t.Errorf("Mode = %v, want %v", cfg.Mode, tt.wantMode)
			}
			if cfg.Provider != tt.wantProvider {
				t.Errorf("Provider = %v, want %v", cfg.Provider, tt.wantProvider)
			}
		})
	}
}

func TestAutoDetectServerURL(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		port   string
		useTLS bool
		want   string
	}{
		{
			name:   "HTTP default",
			host:   "localhost",
			port:   "8080",
			useTLS: false,
			want:   "http://localhost:8080",
		},
		{
			name:   "HTTPS with TLS",
			host:   "example.com",
			port:   "443",
			useTLS: true,
			want:   "https://example.com:443",
		},
		{
			name:   "custom port with HTTPS",
			host:   "api.example.com",
			port:   "9000",
			useTLS: true,
			want:   "https://api.example.com:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoDetectServerURL(tt.host, tt.port, tt.useTLS)
			if got != tt.want {
				t.Errorf("AutoDetectServerURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantURL string
		wantErr bool
	}{
		{
			name: "basic env config with explicit MCP_URL",
			envVars: map[string]string{
				"MCP_URL":        "http://example.com:8080",
				"OAUTH_PROVIDER": "hmac",
				"OIDC_AUDIENCE":  "test-audience",
				"JWT_SECRET":     "test-secret",
			},
			wantURL: "http://example.com:8080",
			wantErr: false,
		},
		{
			name: "auto-detect URL from host/port (HTTP)",
			envVars: map[string]string{
				"MCP_HOST":       "testhost",
				"MCP_PORT":       "9000",
				"OAUTH_PROVIDER": "hmac",
				"OIDC_AUDIENCE":  "test-audience",
				"JWT_SECRET":     "test-secret",
			},
			wantURL: "http://testhost:9000",
			wantErr: false,
		},
		{
			name: "auto-detect HTTPS from cert files",
			envVars: map[string]string{
				"MCP_HOST":        "secure.example.com",
				"MCP_PORT":        "443",
				"HTTPS_CERT_FILE": "/path/to/cert.pem",
				"HTTPS_KEY_FILE":  "/path/to/key.pem",
				"OAUTH_PROVIDER":  "okta",
				"OIDC_ISSUER":     "https://okta.example.com",
				"OIDC_AUDIENCE":   "test-audience",
			},
			wantURL: "https://secure.example.com:443",
			wantErr: false,
		},
		{
			name: "defaults to localhost:8080",
			envVars: map[string]string{
				"OAUTH_PROVIDER": "hmac",
				"OIDC_AUDIENCE":  "test-audience",
				"JWT_SECRET":     "test-secret",
			},
			wantURL: "http://localhost:8080",
			wantErr: false,
		},
		{
			name: "missing required fields returns error",
			envVars: map[string]string{
				"OAUTH_PROVIDER": "hmac",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			cfg, err := FromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("FromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cfg.ServerURL != tt.wantURL {
				t.Errorf("ServerURL = %v, want %v", cfg.ServerURL, tt.wantURL)
			}
		})
	}
}

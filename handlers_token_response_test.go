package oauth

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func tokenHash(v interface{}) string {
	s, _ := v.(string)
	if s == "" {
		return "<empty>"
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))[:16]
}

func TestBuildTokenResponse(t *testing.T) {
	tests := []struct {
		name                 string
		provider             string
		accessToken          string
		idToken              string
		refreshToken         string
		scope                string
		expectedAccessToken  string
		expectedIDToken      string
		expectedRefreshToken string
	}{
		{
			name:                 "google opaque access token uses id token",
			provider:             "google",
			accessToken:          "ya29.a0ARW5m7Opaque",
			idToken:              "header.payload.signature",
			refreshToken:         "refresh-token",
			scope:                "openid profile email",
			expectedAccessToken:  "header.payload.signature",
			expectedIDToken:      "header.payload.signature",
			expectedRefreshToken: "refresh-token",
		},
		{
			name:                "google jwt access token preserved",
			provider:            "google",
			accessToken:         "jwt.access.token",
			idToken:             "id.token.value",
			expectedAccessToken: "jwt.access.token",
			expectedIDToken:     "id.token.value",
		},
		{
			name:                "non google access token preserved",
			provider:            "okta",
			accessToken:         "opaque-token-value",
			idToken:             "id.token.value",
			expectedAccessToken: "opaque-token-value",
			expectedIDToken:     "id.token.value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &OAuth2Handler{
				config: &OAuth2Config{Provider: tt.provider},
				logger: &defaultLogger{},
			}

			extra := map[string]interface{}{
				"id_token": tt.idToken,
			}
			if tt.scope != "" {
				extra["scope"] = tt.scope
			}

			token := (&oauth2.Token{
				AccessToken:  tt.accessToken,
				TokenType:    "Bearer",
				RefreshToken: tt.refreshToken,
				Expiry:       time.Now().Add(time.Hour),
			}).WithExtra(extra)

			response := handler.buildTokenResponse(token)

			if got := response["access_token"]; got != tt.expectedAccessToken {
				t.Fatalf("access_token hash = %s, want hash %s", tokenHash(got), tokenHash(tt.expectedAccessToken))
			}

			if got := response["id_token"]; got != tt.expectedIDToken {
				t.Fatalf("id_token hash = %s, want hash %s", tokenHash(got), tokenHash(tt.expectedIDToken))
			}

			if got := response["refresh_token"]; got != tt.expectedRefreshToken && !(got == nil && tt.expectedRefreshToken == "") {
				t.Fatalf("refresh_token hash = %s, want hash %s", tokenHash(got), tokenHash(tt.expectedRefreshToken))
			}
		})
	}
}

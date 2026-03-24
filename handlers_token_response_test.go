package oauth

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestBuildTokenResponseGoogleOpaqueAccessTokenUsesIDToken(t *testing.T) {
	handler := &OAuth2Handler{
		config: &OAuth2Config{Provider: "google"},
		logger: &defaultLogger{},
	}

	idToken := "header.payload.signature"
	token := (&oauth2.Token{
		AccessToken:  "ya29.a0ARW5m7Opaque",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}).WithExtra(map[string]interface{}{
		"id_token": idToken,
		"scope":    "openid profile email",
	})

	response := handler.buildTokenResponse(token)

	if response["access_token"] != idToken {
		t.Fatalf("expected access_token to be mapped to id_token for Google opaque token")
	}

	if response["id_token"] != idToken {
		t.Fatalf("expected id_token in response")
	}

	if response["refresh_token"] != "refresh-token" {
		t.Fatalf("expected refresh_token in response")
	}
}

func TestBuildTokenResponseGoogleJWTAccessTokenPreserved(t *testing.T) {
	handler := &OAuth2Handler{
		config: &OAuth2Config{Provider: "google"},
		logger: &defaultLogger{},
	}

	jwtAccessToken := "jwt.access.token"
	token := (&oauth2.Token{
		AccessToken: jwtAccessToken,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}).WithExtra(map[string]interface{}{
		"id_token": "id.token.value",
	})

	response := handler.buildTokenResponse(token)

	if response["access_token"] != jwtAccessToken {
		t.Fatalf("expected JWT access_token to remain unchanged")
	}
}

func TestBuildTokenResponseNonGoogleAccessTokenPreserved(t *testing.T) {
	handler := &OAuth2Handler{
		config: &OAuth2Config{Provider: "okta"},
		logger: &defaultLogger{},
	}

	token := (&oauth2.Token{
		AccessToken: "opaque-token-value",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}).WithExtra(map[string]interface{}{
		"id_token": "id.token.value",
	})

	response := handler.buildTokenResponse(token)

	if response["access_token"] != "opaque-token-value" {
		t.Fatalf("expected non-Google access_token to remain unchanged")
	}
}

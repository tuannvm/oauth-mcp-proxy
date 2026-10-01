package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestPublicClientSendsClientIDInBody reproduces an Entra ID behavior: the
// refresh grant accepts client_id via HTTP Basic auth, but the authorization
// code grant requires it in the body. With auth-style auto-detection, a refresh
// cached the header style and every later code exchange failed (AADSTS900144).
func TestPublicClientSendsClientIDInBody(t *testing.T) {
	var sawAuthHeader bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuthHeader = true
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse upstream form: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.PostForm.Get("grant_type") == "authorization_code" && r.PostForm.Get("client_id") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_request","error_description":"AADSTS900144: The request body must contain the following parameter: 'client_id'."}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "rt2",
		})
	}))
	defer upstream.Close()

	h := NewOAuth2Handler(&OAuth2Config{
		Mode:     "proxy",
		Provider: "hmac",
		Issuer:   upstream.URL,
		ClientID: "public-client",
	}, &defaultLogger{})

	for _, form := range []url.Values{
		{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
		{"grant_type": {"authorization_code"}, "code": {"c"}},
	} {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.HandleToken(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body %q", form.Get("grant_type"), rec.Code, rec.Body.String())
		}
	}
	if sawAuthHeader {
		t.Error("public client sent an Authorization header to the token endpoint")
	}
}

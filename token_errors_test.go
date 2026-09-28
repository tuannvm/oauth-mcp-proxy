package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func newTokenErrorTestHandler(t *testing.T, upstreamStatus int, upstreamBody string) *OAuth2Handler {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(upstreamStatus)
		_, _ = w.Write([]byte(upstreamBody))
	}))
	t.Cleanup(upstream.Close)

	return &OAuth2Handler{
		config: &OAuth2Config{Mode: "proxy"},
		oauth2Config: &oauth2.Config{
			ClientID: "client-id",
			Endpoint: oauth2.Endpoint{TokenURL: upstream.URL, AuthStyle: oauth2.AuthStyleInParams},
		},
		logger: &defaultLogger{},
	}
}

func postToken(t *testing.T, h *OAuth2Handler, form url.Values) (*httptest.ResponseRecorder, map[string]string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.HandleToken(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v (body %q)", err, rec.Body.String())
	}
	return rec, body
}

func TestHandleTokenErrorResponses(t *testing.T) {
	invalidGrant := `{"error":"invalid_grant","error_description":"AADSTS70043: The refresh token has expired"}`

	tests := []struct {
		name           string
		upstreamStatus int
		upstreamBody   string
		form           url.Values
		wantStatus     int
		wantError      string
		wantDesc       string
	}{
		{
			name:           "expired refresh token relays invalid_grant",
			upstreamStatus: http.StatusBadRequest,
			upstreamBody:   invalidGrant,
			form:           url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_grant",
			wantDesc:       "AADSTS70043: The refresh token has expired",
		},
		{
			name:           "rejected authorization code relays invalid_grant",
			upstreamStatus: http.StatusBadRequest,
			upstreamBody:   invalidGrant,
			form:           url.Values{"grant_type": {"authorization_code"}, "code": {"c"}},
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_grant",
		},
		{
			name:           "upstream invalid_request is relayed",
			upstreamStatus: http.StatusBadRequest,
			upstreamBody:   `{"error":"invalid_request"}`,
			form:           url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_request",
		},
		{
			name:           "upstream invalid_scope is relayed",
			upstreamStatus: http.StatusBadRequest,
			upstreamBody:   `{"error":"invalid_scope"}`,
			form:           url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_scope",
		},
		{
			name:           "upstream unauthorized_client is relayed",
			upstreamStatus: http.StatusBadRequest,
			upstreamBody:   `{"error":"unauthorized_client"}`,
			form:           url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
			wantStatus:     http.StatusBadRequest,
			wantError:      "unauthorized_client",
		},
		{
			name:           "upstream invalid_client is a server error",
			upstreamStatus: http.StatusUnauthorized,
			upstreamBody:   `{"error":"invalid_client"}`,
			form:           url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
			wantStatus:     http.StatusBadGateway,
			wantError:      "server_error",
		},
		{
			name:           "upstream 5xx with a grant error code stays a server error on refresh",
			upstreamStatus: http.StatusServiceUnavailable,
			upstreamBody:   invalidGrant,
			form:           url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"rt"}},
			wantStatus:     http.StatusBadGateway,
			wantError:      "server_error",
		},
		{
			name:           "upstream 5xx with a grant error code stays a server error on code exchange",
			upstreamStatus: http.StatusInternalServerError,
			upstreamBody:   invalidGrant,
			form:           url.Values{"grant_type": {"authorization_code"}, "code": {"c"}},
			wantStatus:     http.StatusInternalServerError,
			wantError:      "server_error",
		},
		{
			name:           "upstream outage is a server error",
			upstreamStatus: http.StatusServiceUnavailable,
			upstreamBody:   `unavailable`,
			form:           url.Values{"grant_type": {"authorization_code"}, "code": {"c"}},
			wantStatus:     http.StatusInternalServerError,
			wantError:      "server_error",
		},
		{
			name:       "missing authorization code",
			form:       url.Values{"grant_type": {"authorization_code"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "missing refresh token",
			form:       url.Values{"grant_type": {"refresh_token"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "unsupported grant type",
			form:       url.Values{"grant_type": {"password"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "unsupported_grant_type",
		},
		{
			name:       "oversized parameter",
			form:       url.Values{"grant_type": {"authorization_code"}, "code": {strings.Repeat("a", 100000)}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTokenErrorTestHandler(t, tt.upstreamStatus, tt.upstreamBody)
			rec, body := postToken(t, h, tt.form)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if body["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", body["error"], tt.wantError)
			}
			if tt.wantDesc != "" && body["error_description"] != tt.wantDesc {
				t.Errorf("error_description = %q, want %q", body["error_description"], tt.wantDesc)
			}
			if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
				t.Errorf("Cache-Control = %q, want no-store", cc)
			}
		})
	}
}

package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFMiddlewareDoubleSubmit(t *testing.T) {
	handler := CSRFMiddleware(CSRFConfig{
		Enabled:     true,
		ExemptPaths: []string{"/api/v1/auth/login"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))

	do := func(method, path, cookieToken, headerToken string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		if cookieToken != "" {
			r.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookieToken})
		}
		if headerToken != "" {
			r.Header.Set("X-CSRF-Token", headerToken)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}

	// Safe methods pass through and receive a fresh token cookie.
	w := do("GET", "/api/v1/companies", "", "")
	if w.Code != 204 {
		t.Fatalf("safe method blocked: %d", w.Code)
	}
	var token string
	for _, c := range w.Result().Cookies() {
		if c.Name == "csrf_token" {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("no csrf cookie issued")
	}

	// Exempt path works with no token at all.
	if w := do("POST", "/api/v1/auth/login", "", ""); w.Code != 204 {
		t.Fatalf("exempt login blocked: %d", w.Code)
	}
	// Bearer-authenticated requests are exempt (MCP API keys).
	r := httptest.NewRequest("POST", "http://example.com/api/v1/companies/create", nil)
	r.Header.Set("Authorization", "Bearer key")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r)
	if w2.Code != 204 {
		t.Fatalf("bearer request blocked: %d", w2.Code)
	}
	// Missing header -> 403.
	if w := do("POST", "/api/v1/companies/create", token, ""); w.Code != 403 {
		t.Fatalf("missing header accepted: %d", w.Code)
	}
	// Mismatched header -> 403.
	if w := do("POST", "/api/v1/companies/create", token, "forged"); w.Code != 403 {
		t.Fatalf("forged token accepted: %d", w.Code)
	}
	// Matching header + cookie -> allowed.
	if w := do("POST", "/api/v1/companies/create", token, token); w.Code != 204 {
		t.Fatalf("valid token rejected: %d", w.Code)
	}

	// Disabled middleware is a pass-through.
	off := CSRFMiddleware(CSRFConfig{Enabled: false})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	r3 := httptest.NewRequest("POST", "http://example.com/x", nil)
	w3 := httptest.NewRecorder()
	off.ServeHTTP(w3, r3)
	if w3.Code != 204 {
		t.Fatalf("disabled middleware blocked: %d", w3.Code)
	}
}

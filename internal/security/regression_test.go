package security

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRateLimitIgnoresSpoofedForwardingHeaders(t *testing.T) {
	cfg := DefaultSecurityConfig()
	cfg.RateLimitRPS = 0.0001
	cfg.RateLimitBurst = 1
	handler := RateLimitMiddleware(cfg, slog.Default())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for i, ip := range []string{"1.2.3.4", "2.3.4.5"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Forwarded-For", ip)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		want := 204
		if i > 0 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("got %d want %d", w.Code, want)
		}
	}
}

func TestSecureClientBlocksInitialPrivateRequest(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("blocked request reached private server") }))
	defer target.Close()
	client := SecureHTTPClient(DefaultSecureHTTPClientConfig())
	defer client.CloseIdleConnections()
	if response, err := client.Get(target.URL); err == nil {
		response.Body.Close()
		t.Fatal("private target was allowed")
	}
}

func TestSecureClientLimitsChunkedBody(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		io.WriteString(w, strings.Repeat("a", 100))
	}))
	defer target.Close()
	cfg := DefaultSecureHTTPClientConfig()
	cfg.AllowedCIDRs = []string{"127.0.0.0/8"}
	cfg.MaxResponseSize = 10
	client := SecureHTTPClient(cfg)
	defer client.CloseIdleConnections()
	response, err := client.Get(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err == nil || len(data) > 10 {
		t.Fatalf("response size limit failed: %d bytes, error %v", len(data), err)
	}
}

func TestCrossOriginWrites(t *testing.T) {
	handler := CrossOriginMiddleware([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, test := range []struct {
		origin string
		want   int
	}{{"https://evil.example", 403}, {"http://localhost:5173", 204}, {"http://example.com", 204}} {
		r := httptest.NewRequest("POST", "http://example.com/api/v1/auth/login", strings.NewReader("{}"))
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Errorf("%s: %d want %d", test.origin, w.Code, test.want)
		}
	}
}

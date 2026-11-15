package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistryExposition(t *testing.T) {
	reg := New()
	reg.SetJobStats(func() (uint64, uint64, uint64) { return 7, 5, 2 })

	handler := reg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
	}))

	for _, path := range []string{"/ok", "/ok", "/fail"} {
		r := httptest.NewRequest("GET", "http://example.com"+path, nil)
		r.Pattern = "/things/{id}"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}

	w := httptest.NewRecorder()
	reg.Handler(w, httptest.NewRequest("GET", "http://example.com/metrics", nil))
	if w.Code != 200 {
		t.Fatalf("metrics status: %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"crm_up 1",
		`crm_http_requests_total{method="GET",route="/things/{id}",status="204"} 2`,
		`crm_http_requests_total{method="GET",route="/things/{id}",status="500"} 1`,
		"crm_jobs_enqueued_total 7",
		"crm_jobs_completed_total 5",
		"crm_jobs_failed_total 2",
		"crm_http_requests_in_flight",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics missing %q", want)
		}
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content type: %q", ct)
	}

	// Non-GET is rejected.
	w2 := httptest.NewRecorder()
	reg.Handler(w2, httptest.NewRequest("POST", "http://example.com/metrics", nil))
	if w2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /metrics: %d", w2.Code)
	}
}

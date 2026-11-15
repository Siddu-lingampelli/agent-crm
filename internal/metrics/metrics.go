// Package metrics provides a tiny dependency-free Prometheus text exposition
// endpoint plus a middleware that counts HTTP requests. It exists so the
// k8s prometheus.io/* annotations and monitoring/prometheus.yml have a real
// /metrics target without pulling in a client library.
package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Registry collects process and HTTP counters in Prometheus text format.
type Registry struct {
	start time.Time

	mu           sync.Mutex
	requestCount map[string]*atomic.Uint64
	requestSum   map[string]*atomic.Uint64 // microseconds
	inFlight     atomic.Int64

	// jobStats, when set, supplies live job pool counters on each scrape.
	jobStats func() (enqueued, completed, failed uint64)

	// MCP per-tool counters: tool|status -> count, and duration sums (microseconds).
	mcpCount map[string]*atomic.Uint64
	mcpSum   map[string]*atomic.Uint64
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{
		start:        time.Now(),
		requestCount: map[string]*atomic.Uint64{},
		requestSum:   map[string]*atomic.Uint64{},
		mcpCount:     map[string]*atomic.Uint64{},
		mcpSum:       map[string]*atomic.Uint64{},
	}
}

// RecordMCPToolCall records one MCP tools/call: tool name, status ("ok"/"error")
// and duration. Safe for concurrent use.
func (r *Registry) RecordMCPToolCall(tool, status string, dur time.Duration) {
	key := tool + "|" + status
	r.mu.Lock()
	c, ok := r.mcpCount[key]
	if !ok {
		c = &atomic.Uint64{}
		r.mcpCount[key] = c
	}
	s, ok := r.mcpSum[key]
	if !ok {
		s = &atomic.Uint64{}
		r.mcpSum[key] = s
	}
	r.mu.Unlock()
	c.Add(1)
	s.Add(uint64(dur.Microseconds()))
}

// SetJobStats binds a live source of job pool counters (called on each scrape).
func (r *Registry) SetJobStats(fn func() (enqueued, completed, failed uint64)) {
	r.jobStats = fn
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Middleware records request counts and durations by method, route pattern
// and status code.
func (r *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.inFlight.Add(1)
		defer r.inFlight.Add(-1)
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, req)
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		route := req.Pattern
		if route == "" {
			route = req.URL.Path
		}
		key := fmt.Sprintf("%s|%s|%d", req.Method, route, status)
		r.mu.Lock()
		count, ok := r.requestCount[key]
		if !ok {
			count = &atomic.Uint64{}
			r.requestCount[key] = count
		}
		sum, ok := r.requestSum[key]
		if !ok {
			sum = &atomic.Uint64{}
			r.requestSum[key] = sum
		}
		r.mu.Unlock()
		count.Add(1)
		sum.Add(uint64(time.Since(start).Microseconds()))
	})
}

// Handler writes the current metrics in Prometheus text exposition format.
func (r *Registry) Handler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	var b strings.Builder
	b.WriteString("# HELP crm_up Whether the CRM process is up (always 1).\n# TYPE crm_up gauge\ncrm_up 1\n")
	fmt.Fprintf(&b, "# HELP crm_uptime_seconds Seconds since process start.\n# TYPE crm_uptime_seconds gauge\ncrm_uptime_seconds %.3f\n", time.Since(r.start).Seconds())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	writeGauge(&b, "crm_go_goroutines", "Number of goroutines.", float64(runtime.NumGoroutine()))
	writeGauge(&b, "crm_go_mem_alloc_bytes", "Heap bytes allocated and in use.", float64(m.HeapAlloc))
	writeGauge(&b, "crm_go_mem_sys_bytes", "Total bytes obtained from the OS.", float64(m.Sys))
	writeGauge(&b, "crm_go_gc_runs_total", "Number of completed GC cycles.", float64(m.NumGC))

	writeGauge(&b, "crm_http_requests_in_flight", "HTTP requests currently being served.", float64(r.inFlight.Load()))

	r.mu.Lock()
	keys := make([]string, 0, len(r.requestCount))
	for k := range r.requestCount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b.WriteString("# HELP crm_http_requests_total Total HTTP requests by method, route and status.\n# TYPE crm_http_requests_total counter\n")
	for _, k := range keys {
		parts := strings.SplitN(k, "|", 3)
		fmt.Fprintf(&b, "crm_http_requests_total{method=%q,route=%q,status=\"%s\"} %d\n", parts[0], parts[1], parts[2], r.requestCount[k].Load())
	}
	b.WriteString("# HELP crm_http_request_duration_microseconds_total Total request duration in microseconds.\n# TYPE crm_http_request_duration_microseconds_total counter\n")
	for _, k := range keys {
		parts := strings.SplitN(k, "|", 3)
		fmt.Fprintf(&b, "crm_http_request_duration_microseconds_total{method=%q,route=%q,status=\"%s\"} %d\n", parts[0], parts[1], parts[2], r.requestSum[k].Load())
	}
	r.mu.Unlock()

	var jobsEnqueued, jobsCompleted, jobsFailed uint64
	if r.jobStats != nil {
		jobsEnqueued, jobsCompleted, jobsFailed = r.jobStats()
	}
	writeCounter(&b, "crm_jobs_enqueued_total", "Jobs enqueued into the worker pool.", jobsEnqueued)
	writeCounter(&b, "crm_jobs_completed_total", "Jobs completed by the worker pool.", jobsCompleted)
	writeCounter(&b, "crm_jobs_failed_total", "Jobs that failed in the worker pool.", jobsFailed)

	r.mu.Lock()
	mcpKeys := make([]string, 0, len(r.mcpCount))
	for k := range r.mcpCount {
		mcpKeys = append(mcpKeys, k)
	}
	sort.Strings(mcpKeys)
	if len(mcpKeys) > 0 {
		b.WriteString("# HELP crm_mcp_tool_calls_total Total MCP tool calls by tool and status.\n# TYPE crm_mcp_tool_calls_total counter\n")
		for _, k := range mcpKeys {
			parts := strings.SplitN(k, "|", 2)
			fmt.Fprintf(&b, "crm_mcp_tool_calls_total{tool=%q,status=%q} %d\n", parts[0], parts[1], r.mcpCount[k].Load())
		}
		b.WriteString("# HELP crm_mcp_tool_call_duration_microseconds_total Total MCP tool call duration in microseconds.\n# TYPE crm_mcp_tool_call_duration_microseconds_total counter\n")
		for _, k := range mcpKeys {
			parts := strings.SplitN(k, "|", 2)
			fmt.Fprintf(&b, "crm_mcp_tool_call_duration_microseconds_total{tool=%q,status=%q} %d\n", parts[0], parts[1], r.mcpSum[k].Load())
		}
	}
	r.mu.Unlock()

	_, _ = w.Write([]byte(b.String()))
}

func writeGauge(b *strings.Builder, name, help string, v float64) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s gauge\n%s %.0f\n", name, help, name, name, v)
}

func writeCounter(b *strings.Builder, name, help string, v uint64) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, v)
}

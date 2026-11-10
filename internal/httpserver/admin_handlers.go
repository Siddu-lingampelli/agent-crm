// Package httpserver — REST handlers for admin endpoints.
package httpserver

import (
	"crm.local/internal/repository"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"crm.local/internal/db"
	"crm.local/internal/jobs"
	"crm.local/internal/jsonutil"
	"crm.local/internal/memtrack"
	"crm.local/internal/processmetrics"
)

// AdminHandler wraps admin endpoints.
type AdminHandler struct {
	db         *db.DB
	jobs       *jobs.Pool
	memTracker *memtrack.MemoryTracker
	start      time.Time
}

// NewAdminHandler returns a handler for admin endpoints.
func NewAdminHandler(database *db.DB, jobPool *jobs.Pool, memTracker *memtrack.MemoryTracker) *AdminHandler {
	return &AdminHandler{
		db:         database,
		jobs:       jobPool,
		memTracker: memTracker,
		start:      time.Now(),
	}
}

func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	started := time.Now()
	databaseOK := h.db != nil && h.db.Ready()
	databaseStatus, databaseMessage := "healthy", "Database connection OK"
	if !databaseOK {
		databaseStatus, databaseMessage = "down", "Database unavailable"
	}
	jobOK := h.jobs != nil
	jobDetails := map[string]any{}
	if h.jobs != nil {
		jobDetails = h.jobs.Status()
		if closed, ok := jobDetails["closed"].(bool); ok && closed {
			jobOK = false
		}
	}
	jobStatus, jobMessage := "healthy", "Job pool running"
	if !jobOK {
		jobStatus, jobMessage = "down", "Job pool unavailable"
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	checks := []map[string]any{{"id": "database", "name": "database", "status": databaseStatus, "message": databaseMessage, "latency_ms": float64(time.Since(started).Microseconds()) / 1000, "timestamp": timestamp, "details": map[string]any{}}, {"id": "jobs", "name": "jobs", "status": jobStatus, "message": jobMessage, "latency_ms": 0, "timestamp": timestamp, "details": jobDetails}}
	status := http.StatusOK
	if !databaseOK || !jobOK {
		status = http.StatusServiceUnavailable
	}
	jsonutil.Write(w, status, map[string]any{"checks": checks})
}

func (h *AdminHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := map[string]any{
		"cpu_usage":      0.0, // Would need OS-specific implementation
		"memory_usage":   m.HeapAlloc,
		"memory_total":   m.Sys,
		"disk_usage":     0, // Would need disk usage check
		"disk_total":     0,
		"network_in":     0,
		"network_out":    0,
		"goroutines":     runtime.NumGoroutine(),
		"db_connections": h.db.SQL().Stats().OpenConnections,
		"db_idle":        h.db.SQL().Stats().Idle,
		"uptime_seconds": time.Since(h.start).Seconds(),
	}
	metrics["process_memory"] = nil
	if processMemory, err := processmetrics.Memory(); err == nil {
		metrics["process_memory"] = processMemory
	}

	jsonutil.Write(w, http.StatusOK, metrics)
}

func (h *AdminHandler) Queues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	queueStats := []map[string]any{}
	if h.jobs != nil {
		stats := h.jobs.Status()
		stats["name"] = "default"
		elapsed := time.Since(h.start).Minutes()
		if completed, ok := stats["completed"].(int64); ok && elapsed > 0 {
			stats["throughput_per_min"] = float64(completed) / elapsed
		}
		queueStats = append(queueStats, stats)
	}

	jsonutil.Write(w, http.StatusOK, map[string]any{"queues": queueStats})
}

func (h *AdminHandler) SchemaTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	tables, err := repository.NewAdminRepository(h.db.SQL()).GetSchemaTables(r.Context())
	if err != nil {
		jsonutil.Error(w, 500, "schema_failed", "could not read schema")
		return
	}
	items := []map[string]any{}
	for _, table := range tables {
		if q := strings.ToLower(r.URL.Query().Get("q")); q != "" && !strings.Contains(strings.ToLower(table.Name), q) {
			continue
		}
		if kind := r.URL.Query().Get("type"); kind != "" && kind != "table" {
			continue
		}
		columns := []map[string]any{}
		for _, column := range table.Columns {
			columns = append(columns, map[string]any{"name": column.Name, "type": column.Type, "nullable": !column.NotNull, "default_value": column.Default, "primary_key": column.PK})
		}
		indexes := []map[string]any{}
		for _, index := range table.Indexes {
			indexes = append(indexes, map[string]any{"name": index, "columns": []string{}, "type": "index"})
		}
		items = append(items, map[string]any{"id": table.Name, "name": table.Name, "type": "table", "columns": columns, "indexes": indexes, "foreign_keys": []any{}})
	}
	total := len(items)
	limit, offset := pagination(r)
	start := min(offset, total)
	end := min(start+limit, total)
	jsonutil.Write(w, 200, map[string]any{"items": items[start:end], "total": total, "limit": limit, "offset": offset})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (h *AdminHandler) SchemaMigrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	migrations, err := repository.NewAdminRepository(h.db.SQL()).GetSchemaMigrations(r.Context())
	if err != nil {
		jsonutil.Error(w, 500, "migrations_failed", "could not read migrations")
		return
	}
	items := []map[string]any{}
	for _, migration := range migrations {
		items = append(items, map[string]any{"id": fmt.Sprint(migration.Version), "name": migration.Name, "applied_at": migration.AppliedAt})
	}
	jsonutil.Write(w, 200, map[string]any{"items": items})
}

func (h *AdminHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	limit, offset := pagination(r)
	page, err := repository.NewAdminRepository(h.db.SQL()).ListAuditLogs(r.Context(), repository.AuditLogListOptions{Query: r.URL.Query().Get("q"), WorkspaceID: getAuth(r).WorkspaceID, ActorType: r.URL.Query().Get("actor_type"), Action: r.URL.Query().Get("action"), ActorID: r.URL.Query().Get("actor_id"), ResourceType: r.URL.Query().Get("resource_type"), ResourceID: r.URL.Query().Get("resource_id"), Sort: r.URL.Query().Get("sort"), Order: r.URL.Query().Get("order"), StartFrom: r.URL.Query().Get("start_from"), StartTo: r.URL.Query().Get("start_to"), EndBefore: r.URL.Query().Get("end_before"), Limit: limit, Offset: offset})
	if err != nil {
		jsonutil.Error(w, 500, "audit_failed", "could not load audit records")
		return
	}
	jsonutil.Write(w, 200, page)
}

func (h *AdminHandler) Workflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	// Return empty workflows for now
	jsonutil.Write(w, http.StatusOK, map[string]any{
		"items": []any{},
		"total": 0,
	})
}

func (h *AdminHandler) Integrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	// Return empty integrations for now
	jsonutil.Write(w, http.StatusOK, map[string]any{
		"items": []any{},
		"total": 0,
	})
}

func (h *AdminHandler) MemoryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var memStats map[string]any
	if h.memTracker != nil {
		stats := h.memTracker.GetStats()
		budget := memtrack.DefaultMemoryBudget()
		memStats = map[string]any{
			"sample_count":         stats.SampleCount,
			"max_rss_bytes":        stats.MaxRSSBytes,
			"max_heap_alloc_bytes": stats.MaxHeapAllocBytes,
			"avg_rss_bytes":        stats.AvgRSSBytes,
			"avg_heap_alloc_bytes": stats.AvgHeapAllocBytes,
			"current_rss_bytes":    stats.CurrentRSSBytes,
			"current_heap_bytes":   stats.CurrentHeapBytes,
			"current_goroutines":   stats.CurrentGoroutines,
			"gc_count":             stats.GCCount,
			"status":               stats.String(budget),
			"budget": map[string]any{
				"normal_mib":  budget.NormalBudget / 1024 / 1024,
				"heavy_mib":   budget.HeavyBudget / 1024 / 1024,
				"ceiling_mib": budget.CeilingBudget / 1024 / 1024,
			},
		}
	} else {
		memStats = map[string]any{
			"current_rss_bytes":  m.HeapSys,
			"current_heap_bytes": m.HeapAlloc,
			"current_goroutines": runtime.NumGoroutine(),
			"gc_count":           m.NumGC,
			"status":             "unknown",
		}
	}

	jsonutil.Write(w, http.StatusOK, memStats)
}

func (h *AdminHandler) Agents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	// Query agents from database
	query := r.URL.Query().Get("q")
	enabledFilter := r.URL.Query().Get("enabled")
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			offset = parsed
		}
	}

	limit = repository.ClampLimit(limit)
	offset = max(offset, 0)

	// Get workspace from auth context (admin endpoints are protected)
	actor := getAuth(r)
	workspaceID := actor.WorkspaceID

	sqlQuery := `SELECT id, workspace_id, name, description, provider, enabled, scopes, rate_limit, last_used_at, created_at, updated_at FROM agents WHERE workspace_id = ?`
	args := []any{workspaceID}

	if query != "" {
		sqlQuery += ` AND (name LIKE ? OR description LIKE ?)`
		term := "%" + query + "%"
		args = append(args, term, term)
	}
	if enabledFilter != "" {
		sqlQuery += ` AND enabled = ?`
		if enabledFilter == "true" {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM agents WHERE workspace_id = ?`
	countArgs := []any{workspaceID}
	if query != "" {
		countQuery += ` AND (name LIKE ? OR description LIKE ?)`
		term := "%" + query + "%"
		countArgs = append(countArgs, term, term)
	}
	if enabledFilter != "" {
		countQuery += ` AND enabled = ?`
		if enabledFilter == "true" {
			countArgs = append(countArgs, 1)
		} else {
			countArgs = append(countArgs, 0)
		}
	}

	var total int
	if err := h.db.SQL().QueryRowContext(r.Context(), countQuery, countArgs...).Scan(&total); err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "failed to count agents")
		return
	}

	sqlQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.db.SQL().QueryContext(r.Context(), sqlQuery, args...)
	if err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "failed to query agents")
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, workspaceIDResult, name, description, provider, scopes string
		var enabled int
		var rateLimit, lastUsedAt, createdAt, updatedAt sql.NullString
		if err := rows.Scan(&id, &workspaceIDResult, &name, &description, &provider, &enabled, &scopes, &rateLimit, &lastUsedAt, &createdAt, &updatedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":           id,
			"workspace_id": workspaceIDResult,
			"name":         name,
			"description":  description,
			"provider":     provider,
			"enabled":      enabled == 1,
			"scopes":       scopes,
			"rate_limit":   nullStringToPtr(rateLimit),
			"last_used_at": nullStringToPtr(lastUsedAt),
			"created_at":   createdAt.String,
			"updated_at":   updatedAt.String,
		})
	}
	if err := rows.Err(); err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "failed to iterate agents")
		return
	}

	jsonutil.Write(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
	})
}

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func (h *AdminHandler) ForceGC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	if h.memTracker == nil {
		jsonutil.Error(w, http.StatusServiceUnavailable, "not_available", "memory tracker not initialized")
		return
	}

	before, after := h.memTracker.ForceGC()
	var freed uint64
	if before.RSSBytes > after.RSSBytes {
		freed = before.RSSBytes - after.RSSBytes
	}

	jsonutil.Write(w, http.StatusOK, map[string]any{
		"before": map[string]any{
			"rss_bytes":        before.RSSBytes,
			"heap_alloc_bytes": before.HeapAllocBytes,
			"heap_sys_bytes":   before.HeapSysBytes,
			"goroutines":       before.Goroutines,
			"gc_count":         before.GCCount,
		},
		"after": map[string]any{
			"rss_bytes":        after.RSSBytes,
			"heap_alloc_bytes": after.HeapAllocBytes,
			"heap_sys_bytes":   after.HeapSysBytes,
			"goroutines":       after.Goroutines,
			"gc_count":         after.GCCount,
		},
		"freed_bytes": freed,
	})
}

func pagination(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return repository.ClampLimit(limit), max(offset, 0)
}

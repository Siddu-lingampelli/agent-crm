package app

import (
	"context"
	"crm.local/internal/auth"
	"crm.local/internal/config"
	"crm.local/internal/permissions"
	"crm.local/internal/service/agents"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthenticatedHTTPAndMCPFlows(t *testing.T) {
	t.Setenv("CRM_OWNER_EMAIL", "owner@example.com")
	t.Setenv("CRM_OWNER_PASSWORD", "test-password-123")
	cfg := config.Defaults()
	cfg.Database.Path = filepath.Join(t.TempDir(), "crm.db")
	cfg.Security.SessionCookieName = "test_session"
	cfg.Security.SessionCookieSecure = true
	cfg.Security.MaxRequestBodySize = 2048
	app, err := New(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	handler := app.Handler()
	call := func(method, path, body string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		for key, value := range headers {
			r.Header.Set(key, value)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/mcp", "/api/v1/admin/memory", "/api/v1/status", "/api/v1/companies", "/api/v1/pipelines"} {
		w := call("GET", path, "", nil, nil)
		if w.Code != 401 {
			t.Fatalf("unprotected %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := call("GET", "/health", "", nil, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := call("OPTIONS", "/api/v1/companies", "", nil, map[string]string{"Origin": "http://localhost:5173", "Access-Control-Request-Method": "POST"}); w.Code != 204 {
		t.Fatalf("preflight requires auth: %d", w.Code)
	}
	if w := call("POST", "/api/v1/auth/login", `{}`, nil, map[string]string{"Origin": "https://evil.example"}); w.Code != 403 {
		t.Fatalf("cross-origin login allowed: %d", w.Code)
	}
	login := call("POST", "/api/v1/auth/login", `{"email":"owner@example.com","password":"test-password-123"}`, nil, nil)
	if login.Code != 200 {
		t.Fatalf("login: %d %s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing session cookie")
	}
	cookie := cookies[0]
	if cookie.Name != "test_session" || !cookie.HttpOnly || !cookie.Secure {
		t.Fatalf("insecure cookie: %+v", cookie)
	}
	var stored string
	if err := app.db.SQL().QueryRow("SELECT id FROM sessions LIMIT 1").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == cookie.Value {
		t.Fatal("raw session token stored in database")
	}
	if w := call("POST", "/api/v1/companies/create", strings.Repeat("x", 2049), cookie, nil); w.Code != 413 {
		t.Fatalf("protected body limit not enforced: %d", w.Code)
	}
	if w := call("POST", "/api/v1/companies/create", `{"name":"CSRF"}`, cookie, map[string]string{"Origin": "https://evil.example"}); w.Code != 403 {
		t.Fatalf("cross-origin write allowed: %d", w.Code)
	}
	for _, test := range []struct{ path, body string }{
		{"/api/v1/companies/create", `{"name":"Workspace Company"}`},
		{"/api/v1/emails/create", `{"to":"test@example.com","subject":"Test","body":"Draft"}`},
		{"/api/v1/calendar/create", `{"title":"Meeting","start_time":"2026-11-10T10:00:00Z","end_time":"2026-11-10T11:00:00Z"}`},
		{"/api/v1/integrations/create", `{"name":"Example","provider":"custom","config":{"test":true},"enabled":true}`},
		{"/api/v1/webhooks/create", `{"name":"Hook","url":"https://example.com/hook","events":["test"],"secret":"test-signing-secret"}`},
	} {
		w := call("POST", test.path, test.body, cookie, nil)
		if w.Code != 201 {
			t.Fatalf("%s: %d %s", test.path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/integrations", "/api/v1/webhooks", "/api/v1/email-templates", "/api/v1/pipelines"} {
		w := call("GET", path, "", cookie, nil)
		var page struct {
			Items []json.RawMessage `json:"items"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || page.Items == nil {
			t.Fatalf("invalid list envelope %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	for _, test := range []struct {
		query string
		count int
	}{
		{"q=exam", 1}, {"q=custom", 1}, {"q=missing", 0},
		{"q=Example&provider=custom", 1}, {"q=Example&status=connected", 0},
		{"q=%27%20OR%201%3D1%20--", 0},
	} {
		w := call("GET", "/api/v1/integrations?"+test.query, "", cookie, nil)
		var page struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != test.count || page.Total != test.count {
			t.Fatalf("integration search/count mismatch %s: %d %s", test.query, w.Code, w.Body.String())
		}
	}
	for _, body := range []string{
		`{"title":"Overnight","start_time":"2026-11-08T10:00:00Z","end_time":"2026-11-11T12:00:00Z","status":"confirmed"}`,
		`{"title":"Cancelled review","start_time":"2026-11-10T15:00:00Z","end_time":"2026-11-10T16:00:00Z","status":"cancelled"}`,
		`{"title":"Offset meeting","start_time":"2026-11-11T00:30:00+05:30","end_time":"2026-11-11T01:30:00+05:30","status":"tentative"}`,
	} {
		if w := call("POST", "/api/v1/calendar/create", body, cookie, nil); w.Code != 201 {
			t.Fatalf("calendar fixture: %d %s", w.Code, w.Body.String())
		}
	}
	for _, test := range []struct {
		start, end, extra string
		count             int
	}{
		{"2026-11-10T18:00:00Z", "2026-11-10T21:00:00Z", "", 2},
		{"2026-11-10T00:00:00Z", "2026-11-11T00:00:00Z", "&status=confirmed", 2},
		{"2026-11-10T00:00:00Z", "2026-11-11T00:00:00Z", "&status=cancelled", 1},
		{"2026-11-11T12:00:00Z", "2026-11-11T13:00:00Z", "", 0},
		{"2026-11-10T11:00:00Z", "2026-11-10T12:00:00Z", "", 1},
		{"2026-11-10T00:00:00Z", "2026-11-11T00:00:00Z", "&q=meeting", 2},
	} {
		w := call("GET", "/api/v1/calendar?range_start="+url.QueryEscape(test.start)+"&range_end="+url.QueryEscape(test.end)+test.extra, "", cookie, nil)
		var page struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != test.count || page.Total != test.count {
			t.Fatalf("calendar overlap/status mismatch %+v: %d %s", test, w.Code, w.Body.String())
		}
	}
	for _, test := range []struct {
		query string
		count int
	}{
		{"q=test%40example.com", 1}, {"q=Draft", 1}, {"q=Test&status=draft", 1}, {"q=Test&status=sent", 0}, {"q=nomatch", 0}, {"q=%27%20OR%201%3D1%20--", 0},
	} {
		w := call("GET", "/api/v1/emails?"+test.query, "", cookie, nil)
		var page struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != test.count || page.Total != test.count {
			t.Fatalf("email search mismatch %s: %d %s", test.query, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/dashboard/metrics", "/api/v1/admin/schema/tables", "/api/v1/admin/schema/migrations"} {
		if w := call("GET", path, "", cookie, nil); w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := call("GET", "/api/v1/auth/me", "", cookie, nil); w.Code != 200 || !strings.Contains(w.Body.String(), "owner@example.com") {
		t.Fatal("account endpoint failed")
	}
	if w := call("GET", "/api/v1/dashboard/metrics?day=2026-02-30", "", cookie, nil); w.Code != 400 {
		t.Fatalf("invalid dashboard date accepted: %d", w.Code)
	}
	if w := call("GET", "/api/v1/dashboard/metrics?day=2026-11-10", "", cookie, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"day":"2026-11-10"`) || !strings.Contains(w.Body.String(), `"currencies":`) {
		t.Fatalf("dashboard insight scope missing: %d %s", w.Code, w.Body.String())
	}
	projectIDs := []string{}
	for _, body := range []string{
		`{"name":"Alpha work","description":"Delivery migration","status":"active","priority":"high","progress":25}`,
		`{"name":"Beta work","description":"Portfolio migration","status":"planning","priority":"low","progress":0}`,
	} {
		w := call("POST", "/api/v1/projects/create", body, cookie, nil)
		var record struct {
			ID string `json:"id"`
		}
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &record) != nil || record.ID == "" {
			t.Fatalf("project fixture: %d %s", w.Code, w.Body.String())
		}
		projectIDs = append(projectIDs, record.ID)
	}
	for _, test := range []struct {
		query string
		count int
	}{{"priority=high", 1}, {"priority=low", 1}, {"q=migration", 2}, {"q=Portfolio&status=active", 0}, {"q=%27%20OR%201%3D1%20--", 0}} {
		w := call("GET", "/api/v1/projects?"+test.query, "", cookie, nil)
		var page struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != test.count || page.Total != test.count {
			t.Fatalf("project filters: %s %d %s", test.query, w.Code, w.Body.String())
		}
	}
	completedID := ""
	for index, input := range []map[string]any{
		{"title": "Past follow-up", "description": "Lookup phrase", "status": "open", "priority": "urgent", "due_date": "2026-11-09"},
		{"title": "Today follow-up", "status": "in_progress", "priority": "high", "due_date": "2026-11-10"},
		{"title": "Completed follow-up", "status": "completed", "priority": "high", "due_date": "2026-11-08"},
		{"title": "Undated task", "status": "open", "priority": "normal"},
		{"title": "Cancelled task", "status": "cancelled", "priority": "normal", "due_date": "2026-11-08"},
	} {
		if index < 3 {
			input["project_id"] = projectIDs[0]
		} else {
			input["project_id"] = projectIDs[1]
		}
		body, _ := json.Marshal(input)
		w := call("POST", "/api/v1/tasks/create", string(body), cookie, nil)
		var record struct {
			ID          string  `json:"id"`
			CompletedAt *string `json:"completed_at"`
		}
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &record) != nil {
			t.Fatalf("task fixture: %d %s", w.Code, w.Body.String())
		}
		if index == 2 {
			completedID = record.ID
			if record.CompletedAt == nil {
				t.Fatal("completed task missing timestamp")
			}
		}
	}
	for _, test := range []struct {
		query string
		count int
	}{
		{"due_before=2026-11-10", 1}, {"due_on=2026-11-10", 1}, {"due_before=2026-11-10&priority=high", 0},
		{"project_id=" + projectIDs[0], 3}, {"due_on=2026-11-10&status=completed", 0}, {"q=Lookup", 1}, {"q=%27%20OR%201%3D1%20--", 0},
		{"active_only=true&unscheduled=true", 1}, {"active_only=true&due_after=2026-11-09", 1}, {"active_only=true&due_after=2026-11-10", 0},
	} {
		w := call("GET", "/api/v1/tasks?"+test.query, "", cookie, nil)
		var page struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != test.count || page.Total != test.count {
			t.Fatalf("task filters: %s %d %s", test.query, w.Code, w.Body.String())
		}
	}
	for _, status := range []string{"open", "completed"} {
		w := call("PATCH", "/api/v1/tasks/"+completedID, `{"status":"`+status+`"}`, cookie, nil)
		var record struct {
			CompletedAt *string `json:"completed_at"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &record) != nil || (record.CompletedAt != nil) != (status == "completed") {
			t.Fatalf("completion transition %s: %d %s", status, w.Code, w.Body.String())
		}
	}
	metricsResponse := call("GET", "/api/v1/admin/metrics", "", cookie, nil)
	var metrics struct {
		MemoryUsage   uint64 `json:"memory_usage"`
		ProcessMemory *struct {
			PID      int    `json:"pid"`
			Resident uint64 `json:"resident_bytes"`
			Peak     uint64 `json:"peak_resident_bytes"`
		} `json:"process_memory"`
	}
	if metricsResponse.Code != 200 || json.Unmarshal(metricsResponse.Body.Bytes(), &metrics) != nil || metrics.MemoryUsage == 0 || !strings.Contains(metricsResponse.Body.String(), `"process_memory":`) || strings.Contains(metricsResponse.Body.String(), `"host_memory":`) {
		t.Fatalf("invalid process memory response: %d %s", metricsResponse.Code, metricsResponse.Body.String())
	}
	if memory := metrics.ProcessMemory; memory != nil && (memory.PID <= 0 || memory.Resident == 0 || memory.Peak < memory.Resident) {
		t.Fatalf("inconsistent process memory counters: %+v", memory)
	}
	var auditCount int
	if err := app.db.SQL().QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action='post'").Scan(&auditCount); err != nil || auditCount < 5 {
		t.Fatalf("successful REST writes were not audited: %d %v", auditCount, err)
	}
	var auditValues string
	if err := app.db.SQL().QueryRow("SELECT new_values FROM audit_logs WHERE resource_type='webhooks'").Scan(&auditValues); err != nil || strings.Contains(auditValues, "test-signing-secret") {
		t.Fatal("audit log contains webhook secret or was not written")
	}
	userID, err := auth.ValidateSession(context.Background(), app.db, cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	var workspace string
	if err := app.db.SQL().QueryRow("SELECT workspace_id FROM users WHERE id=?", userID).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	actor := permissions.AuthContext{ActorID: userID, ActorType: "user", WorkspaceID: workspace, Scopes: auth.OwnerScopes()}
	if _, err := app.db.SQL().Exec("INSERT INTO workspaces (id,name,created_at) VALUES ('other-pipeline-workspace','Other',datetime('now'))"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.SQL().Exec("INSERT INTO pipelines (id,workspace_id,name,is_default,created_at,updated_at) VALUES ('other-pipeline','other-pipeline-workspace','Private pipeline',0,datetime('now'),datetime('now'))"); err != nil {
		t.Fatal(err)
	}
	if w := call("GET", "/api/v1/pipelines", "", cookie, nil); w.Code != 200 || strings.Contains(w.Body.String(), "Private pipeline") {
		t.Fatalf("pipeline choices crossed workspace boundary: %d %s", w.Code, w.Body.String())
	}
	if w := call("POST", "/api/v1/pipelines", `{}`, cookie, nil); w.Code != 405 {
		t.Fatalf("pipeline choices are not read-only: %d", w.Code)
	}
	for _, record := range []struct{ id, workspace, actor, resource, at string }{
		{"ui-audit-old", workspace, userID, "old-resource", "2000-01-01T00:00:00Z"},
		{"ui-audit-new", workspace, "other-actor", "new-resource", "2099-01-01T00:00:00Z"},
		{"ui-audit-foreign", "other-pipeline-workspace", userID, "foreign-resource", "2199-01-01T00:00:00Z"},
	} {
		if _, err := app.db.SQL().Exec(`INSERT INTO audit_logs (id,workspace_id,actor_type,actor_id,action,resource_type,resource_id,old_values,new_values,ip_address,user_agent,created_at) VALUES (?,?,'user',?,'patch','admin-ui-test',?,'{}','{}','127.0.0.1','test',?)`, record.id, record.workspace, record.actor, record.resource, record.at); err != nil {
			t.Fatal(err)
		}
	}
	if w := call("GET", "/api/v1/admin/audit?resource_type=admin-ui-test&sort=created_at&order=desc&limit=1", "", cookie, nil); w.Code != 200 || !strings.Contains(w.Body.String(), "ui-audit-new") || strings.Contains(w.Body.String(), "ui-audit-foreign") {
		t.Fatalf("audit ordering or workspace boundary failed: %d %s", w.Code, w.Body.String())
	}
	if w := call("GET", "/api/v1/admin/audit?resource_type=admin-ui-test&actor_id="+userID+"&resource_id=old-resource", "", cookie, nil); w.Code != 200 || !strings.Contains(w.Body.String(), "ui-audit-old") || strings.Contains(w.Body.String(), "ui-audit-new") {
		t.Fatalf("audit actor/resource filters were ignored: %d %s", w.Code, w.Body.String())
	}
	// Search and UTC ranges work across old driver timestamps and API timestamps.
	for _, record := range []struct{ id, workspace, at string }{
		{"audit-date-legacy", workspace, "2050-11-10 10:15:30.123456789 +0000 UTC"},
		{"audit-date-offset", workspace, "2050-11-11T01:00:00+05:30"},
		{"audit-date-midnight", workspace, "2050-11-11T00:00:00Z"},
		{"audit-date-foreign", "other-pipeline-workspace", "2050-11-10T12:00:00Z"},
	} {
		if _, err := app.db.SQL().Exec("INSERT INTO audit_logs (id,workspace_id,actor_type,actor_id,action,resource_type,resource_id,old_values,new_values,ip_address,user_agent,created_at) VALUES (?,?,'agent','searchable-actor','patch','audit-date-test','searchable-record','{}','{}','192.0.2.10','test',?)", record.id, record.workspace, record.at); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"audit-date-legacy", "searchable-actor", "searchable-record", "192.0.2.10"} {
		w := call("GET", "/api/v1/admin/audit?resource_type=audit-date-test&q="+url.QueryEscape(query), "", cookie, nil)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "audit-date-legacy") || strings.Contains(w.Body.String(), "audit-date-foreign") || strings.Contains(w.Body.String(), "0001-01-01") {
			t.Fatalf("audit search/date decoding failed: %d %s", w.Code, w.Body.String())
		}
	}
	wDate := call("GET", "/api/v1/admin/audit?resource_type=audit-date-test&start_from=2050-11-10T00:00:00Z&end_before=2050-11-11T00:00:00Z&order=asc&limit=1", "", cookie, nil)
	var auditPage struct {
		Items []struct{ ID string }
		Total int
	}
	if err := json.Unmarshal(wDate.Body.Bytes(), &auditPage); err != nil {
		t.Fatal(err)
	}
	if wDate.Code != 200 || auditPage.Total != 2 || len(auditPage.Items) != 1 || auditPage.Items[0].ID != "audit-date-legacy" {
		t.Fatalf("UTC range/count/order failed: %d %s", wDate.Code, wDate.Body.String())
	}
	for _, query := range []string{"audit-date-foreign", "' OR 1=1 --"} {
		w := call("GET", "/api/v1/admin/audit?q="+url.QueryEscape(query), "", cookie, nil)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"total":0`) {
			t.Fatalf("audit search escaped workspace/query boundary: %d %s", w.Code, w.Body.String())
		}
	}
	var legacyTimestamp string
	if err := app.db.SQL().QueryRow("SELECT created_at FROM audit_logs WHERE id='audit-date-legacy'").Scan(&legacyTimestamp); err != nil || legacyTimestamp != "2050-11-10 10:15:30.123456789 +0000 UTC" {
		t.Fatalf("legacy history changed: %q %v", legacyTimestamp, err)
	}
	agent, err := app.agentService.CreateAgent(context.Background(), actor, agents.CreateAgentInput{Name: "Reader", Enabled: true, Scopes: []string{"companies:read"}})
	if err != nil {
		t.Fatal(err)
	}
	raw, key, err := app.agentService.CreateApiKey(context.Background(), actor, agents.CreateApiKeyInput{Name: "Read only", AgentID: agent.ID, Scopes: []string{"companies:read"}})
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{"Authorization": "Bearer " + raw, "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2025-03-26"}
	deniedActor := actor
	deniedActor.Scopes = []string{"companies:read"}
	deniedRequest := httptest.NewRequest("GET", "/api/v1/pipelines", nil)
	deniedRequest = deniedRequest.WithContext(auth.WithAuth(deniedRequest.Context(), deniedActor))
	deniedResponse := httptest.NewRecorder()
	app.handlePipelines(deniedResponse, deniedRequest)
	if deniedResponse.Code != 403 {
		t.Fatalf("pipeline choices ignored deal read scope: %d", deniedResponse.Code)
	}
	w := call("POST", "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"companies.list","arguments":{}}}`, nil, headers)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Workspace Company") {
		t.Fatalf("MCP lost identity or workspace: %d %s", w.Code, w.Body.String())
	}
	w = call("POST", "/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"contacts.list","arguments":{}}}`, nil, headers)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "scope denied") {
		t.Fatalf("MCP did not enforce scopes: %d %s", w.Code, w.Body.String())
	}
	if err := app.agentService.RevokeApiKey(context.Background(), actor, key.ID); err != nil {
		t.Fatal(err)
	}
	if w := call("POST", "/mcp", `{}`, nil, headers); w.Code != 401 {
		t.Fatalf("revoked key accepted: %d", w.Code)
	}
	if w := call("POST", "/api/v1/admin/settings/password", `{"current_password":"test-password-123","new_password":"changed-password-123"}`, cookie, nil); w.Code != 200 {
		t.Fatalf("password change endpoint: %d %s", w.Code, w.Body.String())
	}
	if w := call("POST", "/api/v1/auth/logout", `{}`, cookie, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := call("GET", "/api/v1/companies", "", cookie, nil); w.Code != 401 {
		t.Fatal("logout did not invalidate session")
	}
}

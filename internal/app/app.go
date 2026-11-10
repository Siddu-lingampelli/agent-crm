// Package app wires the CRM application: routing, services, database, jobs.
//
// It is the single entry point used by HTTP handlers, MCP handlers and
// background jobs so that UI, REST and MCP share one application service
// layer. See Tech.md section 2.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"crm.local/internal/auth"
	"crm.local/internal/config"
	"crm.local/internal/db"
	"crm.local/internal/httpserver"
	"crm.local/internal/jobs"
	"crm.local/internal/jsonutil"
	"crm.local/internal/mcp"
	"crm.local/internal/memtrack"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/security"
	"crm.local/internal/service/admin"
	"crm.local/internal/service/agents"
	"crm.local/internal/service/calendar"
	"crm.local/internal/service/companies"
	"crm.local/internal/service/contacts"
	"crm.local/internal/service/dashboard"
	"crm.local/internal/service/deals"
	"crm.local/internal/service/email"
	"crm.local/internal/service/integrations"
	"crm.local/internal/service/leads"
	"crm.local/internal/service/projects"
	"crm.local/internal/service/search"
	"crm.local/internal/service/tasks"
	"crm.local/internal/service/webhooks"
	"crm.local/internal/service/workflows"
	"crm.local/internal/uuid"
)

// App is the assembled CRM application.
type App struct {
	cfg        config.Config
	log        *slog.Logger
	db         *db.DB
	srv        *httpserver.Server
	jobs       *jobs.Pool
	memTracker *memtrack.MemoryTracker

	companyHandler     *httpserver.CompanyHandler
	contactHandler     *httpserver.ContactHandler
	leadHandler        *httpserver.LeadHandler
	dealHandler        *httpserver.DealHandler
	taskHandler        *httpserver.TaskHandler
	projectHandler     *httpserver.ProjectHandler
	searchHandler      *httpserver.SearchHandler
	authHandler        *httpserver.AuthHandler
	adminHandler       *httpserver.AdminHandler
	agentHandler       *httpserver.AgentHandler
	emailHandler       *httpserver.EmailHandler
	calendarHandler    *httpserver.CalendarHandler
	workflowHandler    *httpserver.WorkflowHandler
	integrationHandler *httpserver.IntegrationHandler
	webhookHandler     *httpserver.WebhookHandler
	dashboardHandler   *httpserver.DashboardHandler
	mcpServer          *mcp.Server
	agentService       *agents.Service
	cancel             context.CancelFunc
}

// New builds and initializes the application.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	smtpConfig, err := email.SMTPConfigFromEnvironment()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.Default()
	}
	ctx, cancel := context.WithCancel(ctx)
	initialized := false
	defer func() {
		if !initialized {
			cancel()
		}
	}()
	database, err := db.Open(ctx, db.Config{
		Path:              cfg.Database.Path,
		MaxOpenConns:      cfg.Database.MaxOpenConns,
		MaxIdleConns:      cfg.Database.MaxIdleConns,
		ConnMaxLifetime:   cfg.Database.ConnMaxLifetime,
		BusyTimeout:       cfg.Database.BusyTimeout,
		CacheSizeKiB:      cfg.Database.CacheSizeKiB,
		MmapSizeBytes:     cfg.Database.MmapSizeBytes,
		PRAGMAForeignKeys: cfg.Database.PRAGMAForeignKeys,
		PRAGMAJournalMode: cfg.Database.PRAGMAJournalMode,
		PRAGMASynchronous: cfg.Database.PRAGMASynchronous,
		PRAGMATempStore:   cfg.Database.PRAGMATempStore,
		PRAGMACacheSize:   cfg.Database.PRAGMACacheSize,
		PRAGMAPageSize:    cfg.Database.PRAGMAPageSize,
		PRAGMAMmapSize:    cfg.Database.PRAGMAMmapSize,
	}, log)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !initialized {
			_ = database.Close()
		}
	}()

	// Bootstrap owner user if not exists
	ownerEmail := os.Getenv("CRM_OWNER_EMAIL")
	ownerPassword := os.Getenv("CRM_OWNER_PASSWORD")
	if ownerEmail != "" && ownerPassword != "" {
		if err := auth.OwnerBootstrap(ctx, database, ownerEmail, ownerPassword); err != nil {
			log.Warn("owner bootstrap failed", "error", err)
		}
	}

	srv := httpserver.New(httpserver.Config{
		Addr:            cfg.HTTP.Addr(),
		ReadTimeout:     cfg.HTTP.ReadTimeout,
		WriteTimeout:    cfg.HTTP.WriteTimeout,
		IdleTimeout:     cfg.HTTP.IdleTimeout,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, log)

	jp := jobs.New(jobs.Config{
		Workers:   cfg.Jobs.Workers,
		QueueSize: cfg.Jobs.QueueSize,
	}, log)

	// Initialize memory tracker
	memTracker := memtrack.NewMemoryTracker(log, 1000)
	memTracker.Start(ctx, 5*time.Second)

	app := &App{
		cfg:        cfg,
		log:        log,
		db:         database,
		srv:        srv,
		jobs:       jp,
		memTracker: memTracker,
		cancel:     cancel,
	}

	// Initialize all services
	companySvc := companies.New(repository.NewCompanyRepository(database.SQL()), log)
	contactSvc := contacts.New(repository.NewContactRepository(database.SQL()), log)
	leadSvc := leads.New(repository.NewLeadRepository(database.SQL()), log)
	dealSvc := deals.New(repository.NewDealRepository(database.SQL()), log)
	taskSvc := tasks.New(repository.NewTaskRepository(database.SQL()), log)
	projectSvc := projects.New(repository.NewProjectRepository(database.SQL()), log)
	searchSvc := search.New(database.SQL())

	// Initialize dashboard service
	dashboardSvc := dashboard.New(
		repository.NewDealRepository(database.SQL()),
		repository.NewLeadRepository(database.SQL()),
		repository.NewTaskRepository(database.SQL()),
		repository.NewCompanyRepository(database.SQL()),
		repository.NewContactRepository(database.SQL()),
		log,
	)

	// Initialize Level 7 services
	emailRepo := repository.NewEmailRepository(database.SQL())
	emailSvc := email.New(emailRepo, log)
	emailSvc.SetJobPool(jp)
	if smtpConfig != nil {
		emailSvc.SetSMTPConfig(*smtpConfig)
	}

	calendarRepo := repository.NewCalendarRepository(database.SQL())
	calendarSvc := calendar.New(calendarRepo, log)

	workflowRepo := repository.NewWorkflowRepository(database.SQL())
	workflowSvc := workflows.New(workflowRepo, log)
	workflowSvc.SetJobPool(jp)

	integrationRepo := repository.NewIntegrationRepository(database.SQL())
	integrationSvc := integrations.New(integrationRepo, log)

	webhookRepo := repository.NewWebhookRepository(database.SQL())
	webhookSvc := webhooks.New(webhookRepo, log)

	adminRepo := repository.NewAdminRepository(database.SQL())
	adminSvc := admin.New(adminRepo, log)

	// Initialize agent services
	agentRepo := repository.NewAgentRepository(database.SQL())
	apiKeyRepo := repository.NewApiKeyRepository(database.SQL())
	agentSvc := agents.New(agentRepo, apiKeyRepo, log)
	app.agentService = agentSvc
	sessionOptions := auth.SessionOptions{CookieName: cfg.Security.SessionCookieName, Secure: cfg.Security.SessionCookieSecure, MaxAge: cfg.Security.SessionMaxAge, SameSite: http.SameSiteLaxMode}
	switch strings.ToLower(cfg.Security.SessionCookieSameSite) {
	case "strict":
		sessionOptions.SameSite = http.SameSiteStrictMode
	case "none":
		sessionOptions.SameSite = http.SameSiteNoneMode
	}

	// Initialize handlers
	app.companyHandler = httpserver.NewCompanyHandler(companySvc)
	app.contactHandler = httpserver.NewContactHandler(contactSvc, log)
	app.leadHandler = httpserver.NewLeadHandler(leadSvc, log)
	app.dealHandler = httpserver.NewDealHandler(dealSvc, log)
	app.taskHandler = httpserver.NewTaskHandler(taskSvc, log)
	app.projectHandler = httpserver.NewProjectHandler(projectSvc, log)
	app.searchHandler = httpserver.NewSearchHandler(searchSvc, log)
	app.authHandler = httpserver.NewAuthHandler(database, sessionOptions)
	app.adminHandler = httpserver.NewAdminHandler(database, jp, memTracker)
	app.agentHandler = httpserver.NewAgentHandler(agentSvc)
	app.emailHandler = httpserver.NewEmailHandler(emailSvc)
	app.calendarHandler = httpserver.NewCalendarHandler(calendarSvc)
	app.workflowHandler = httpserver.NewWorkflowHandler(workflowSvc)
	app.integrationHandler = httpserver.NewIntegrationHandler(integrationSvc)
	app.webhookHandler = httpserver.NewWebhookHandler(webhookSvc)
	app.dashboardHandler = httpserver.NewDashboardHandler(dashboardSvc)

	// Initialize MCP server with all services
	app.mcpServer = mcp.NewServer(
		mcp.NewCRMTools(
			companySvc, contactSvc, leadSvc, dealSvc, taskSvc, projectSvc, searchSvc,
			emailSvc, calendarSvc, workflowSvc, integrationSvc, webhookSvc, adminSvc, log,
		))

	// Add security middleware (applies to all routes including public)
	app.srv.MiddlewareNoAuth(security.CORSMiddleware(cfg.Security.ToSecurityConfig(), log))
	app.srv.MiddlewareNoAuth(security.SecurityHeadersMiddleware(log))
	app.srv.MiddlewareNoAuth(security.RequestSizeMiddleware(cfg.Security.ToSecurityConfig(), log))
	app.srv.MiddlewareNoAuth(security.RateLimitMiddleware(cfg.Security.ToSecurityConfig(), log))
	app.srv.MiddlewareNoAuth(security.TimeoutMiddleware(cfg.Security.ToSecurityConfig(), log))
	if cfg.Security.CSRFEnabled {
		app.srv.MiddlewareNoAuth(security.CrossOriginMiddleware(cfg.Security.CORSAllowedOrigins))
	}

	// Add auth middleware to protected routes
	app.srv.Middleware(auth.SessionMiddleware(app.db, log, sessionOptions))
	app.srv.Middleware(httpserver.AuditMiddleware(adminRepo, log))

	app.registerRoutes(srv)
	srv.ServeStatic("/", "web/dist")
	initialized = true
	return app, nil
}

// Handler returns the top-level HTTP handler.
func (a *App) Handler() http.Handler { return a.srv.Handler() }

// Shutdown performs graceful shutdown of jobs and the HTTP server.
func (a *App) Shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.jobs != nil {
		a.jobs.Shutdown(ctx)
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

// Close releases resources.
func (a *App) Close() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.jobs != nil {
		_ = a.jobs.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

// registerRoutes wires the initial route set. Routes are added as levels
// deliver features; nothing is registered that is not implemented.
func (a *App) registerRoutes(srv *httpserver.Server) {
	// Auth (public - no auth middleware)
	srv.HandleFuncNoAuth("/api/v1/auth/login", a.authHandler.Login)
	srv.HandleFuncNoAuth("/api/v1/auth/logout", a.authHandler.Logout)
	srv.HandleFunc("/api/v1/auth/me", a.authHandler.Me)
	srv.HandleFunc("/api/v1/admin/settings/password", a.authHandler.ChangePassword)

	// Public health/check endpoints
	srv.HandleFuncNoAuth("/health", a.handleHealth)
	srv.HandleFuncNoAuth("/ready", a.handleReady)
	srv.HandleFunc("/api/v1/status", a.handleStatus)

	// Search
	srv.HandleFunc("/api/v1/search", a.searchHandler.Search)

	// Dashboard
	srv.HandleFunc("/api/v1/dashboard/metrics", a.dashboardHandler.GetMetrics)

	// Companies
	srv.HandleFunc("/api/v1/companies", a.companyHandler.List)
	srv.HandleFunc("/api/v1/companies/create", a.companyHandler.Create)
	srv.HandleFunc("/api/v1/companies/{id}", a.handleCompany)

	// Contacts
	srv.HandleFunc("/api/v1/contacts", a.contactHandler.List)
	srv.HandleFunc("/api/v1/contacts/create", a.contactHandler.Create)
	srv.HandleFunc("/api/v1/contacts/{id}", a.handleContact)

	// Leads
	srv.HandleFunc("/api/v1/leads", a.leadHandler.List)
	srv.HandleFunc("/api/v1/leads/create", a.leadHandler.Create)
	srv.HandleFunc("/api/v1/leads/{id}", a.handleLead)
	srv.HandleFunc("/api/v1/leads/{id}/convert", a.leadHandler.Convert)

	// Deals
	srv.HandleFunc("/api/v1/pipelines", a.handlePipelines)
	srv.HandleFunc("/api/v1/deals", a.dealHandler.List)
	srv.HandleFunc("/api/v1/deals/create", a.dealHandler.Create)
	srv.HandleFunc("/api/v1/deals/{id}", a.handleDeal)

	// Tasks
	srv.HandleFunc("/api/v1/tasks", a.taskHandler.List)
	srv.HandleFunc("/api/v1/tasks/create", a.taskHandler.Create)
	srv.HandleFunc("/api/v1/tasks/{id}", a.handleTask)
	srv.HandleFunc("/api/v1/tasks/{id}/complete", a.taskHandler.Complete)

	// Projects
	srv.HandleFunc("/api/v1/projects", a.projectHandler.List)
	srv.HandleFunc("/api/v1/projects/create", a.projectHandler.Create)
	srv.HandleFunc("/api/v1/projects/{id}", a.handleProject)

	// Emails
	srv.HandleFunc("/api/v1/emails", a.emailHandler.List)
	srv.HandleFunc("/api/v1/emails/create", a.emailHandler.Create)
	srv.HandleFunc("/api/v1/emails/{id}", a.handleEmail)
	srv.HandleFunc("/api/v1/emails/{id}/approve", a.emailHandler.Approve)
	srv.HandleFunc("/api/v1/emails/{id}/send", a.emailHandler.Send)
	srv.HandleFunc("/api/v1/emails/retry", a.emailHandler.RetryFailed)
	srv.HandleFunc("/api/v1/email-templates", a.emailHandler.ListTemplates)
	srv.HandleFunc("/api/v1/email-templates/create", a.emailHandler.CreateTemplate)
	srv.HandleFunc("/api/v1/email-templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			a.emailHandler.UpdateTemplate(w, r)
		case http.MethodDelete:
			a.emailHandler.DeleteTemplate(w, r)
		default:
			a.emailHandler.GetTemplate(w, r)
		}
	})
	srv.HandleFunc("/api/v1/email-templates/{id}/apply", a.emailHandler.ApplyTemplate)

	// Calendar
	srv.HandleFunc("/api/v1/calendar", a.calendarHandler.List)
	srv.HandleFunc("/api/v1/calendar/create", a.calendarHandler.Create)
	srv.HandleFunc("/api/v1/calendar/{id}", a.handleCalendar)

	// Workflows
	srv.HandleFunc("/api/v1/workflows", a.workflowHandler.List)
	srv.HandleFunc("/api/v1/workflows/create", a.workflowHandler.Create)
	srv.HandleFunc("/api/v1/workflows/{id}", a.handleWorkflow)
	srv.HandleFunc("/api/v1/workflows/{id}/execute", a.workflowHandler.Execute)
	srv.HandleFunc("/api/v1/workflow-executions", a.workflowHandler.ListExecutions)
	srv.HandleFunc("/api/v1/workflow-executions/{id}", a.workflowHandler.GetExecution)

	// Integrations
	srv.HandleFunc("/api/v1/integrations", a.integrationHandler.List)
	srv.HandleFunc("/api/v1/integrations/create", a.integrationHandler.Create)
	srv.HandleFunc("/api/v1/integrations/{id}", a.handleIntegration)
	srv.HandleFunc("/api/v1/integrations/{id}/credentials", a.integrationHandler.UpdateCredentials)
	srv.HandleFunc("/api/v1/integrations/{id}/sync", a.integrationHandler.Sync)

	// Webhooks
	srv.HandleFunc("/api/v1/webhooks", a.webhookHandler.List)
	srv.HandleFunc("/api/v1/webhooks/create", a.webhookHandler.Create)
	srv.HandleFunc("/api/v1/webhooks/{id}", a.handleWebhook)
	srv.HandleFunc("/api/v1/webhooks/{id}/test", a.webhookHandler.Test)

	// MCP endpoint (public - handles its own auth via API keys)
	srv.HandleFuncNoAuth("/mcp", a.handleMCP)
	srv.HandleFunc("/api/v1/mcp", a.handleMCP)

	// Admin endpoints (public for monitoring)
	srv.HandleFunc("/api/v1/admin/health", a.adminHandler.Health)
	srv.HandleFunc("/api/v1/admin/metrics", a.adminHandler.Metrics)
	srv.HandleFunc("/api/v1/admin/queues", a.adminHandler.Queues)
	srv.HandleFunc("/api/v1/admin/memory", a.adminHandler.MemoryStatus)
	srv.HandleFunc("/api/v1/admin/memory/gc", a.adminHandler.ForceGC)
	srv.HandleFunc("/api/v1/admin/schema/tables", a.adminHandler.SchemaTables)
	srv.HandleFunc("/api/v1/admin/schema/migrations", a.adminHandler.SchemaMigrations)
	srv.HandleFunc("/api/v1/admin/audit", a.adminHandler.AuditLogs)
	srv.HandleFunc("/api/v1/admin/workflows", a.adminHandler.Workflows)
	srv.HandleFunc("/api/v1/admin/integrations", a.adminHandler.Integrations)
	srv.HandleFunc("/api/v1/admin/agents", a.adminHandler.Agents)

	// Agent endpoints
	srv.HandleFunc("/api/v1/agents", a.agentHandler.List)
	srv.HandleFunc("/api/v1/agents/create", a.agentHandler.Create)
	srv.HandleFunc("/api/v1/agents/{id}", a.handleAgent)
	srv.HandleFunc("/api/v1/agents/{id}/restore", a.agentHandler.Restore)

	// API Key endpoints
	srv.HandleFunc("/api/v1/api-keys", a.agentHandler.ListApiKeys)
	srv.HandleFunc("/api/v1/api-keys/create", a.agentHandler.CreateApiKey)
	srv.HandleFunc("/api/v1/api-keys/{id}", a.handleApiKey)
}

func (a *App) handleCompany(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.companyHandler.Get(w, r)
	case http.MethodPatch:
		a.companyHandler.Update(w, r)
	case http.MethodDelete:
		a.companyHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleContact(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.contactHandler.Get(w, r)
	case http.MethodPatch:
		a.contactHandler.Update(w, r)
	case http.MethodDelete:
		a.contactHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleLead(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.leadHandler.Get(w, r)
	case http.MethodPatch:
		a.leadHandler.Update(w, r)
	case http.MethodDelete:
		a.leadHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleDeal(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.dealHandler.Get(w, r)
	case http.MethodPatch:
		a.dealHandler.Update(w, r)
	case http.MethodDelete:
		a.dealHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.taskHandler.Get(w, r)
	case http.MethodPatch:
		a.taskHandler.Update(w, r)
	case http.MethodDelete:
		a.taskHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleProject(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.projectHandler.Get(w, r)
	case http.MethodPatch:
		a.projectHandler.Update(w, r)
	case http.MethodDelete:
		a.projectHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleAgent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.agentHandler.Get(w, r)
	case http.MethodPatch:
		a.agentHandler.Update(w, r)
	case http.MethodDelete:
		a.agentHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleApiKey(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.agentHandler.GetApiKey(w, r)
	case http.MethodPatch:
		a.agentHandler.UpdateApiKey(w, r)
	case http.MethodDelete:
		a.agentHandler.RevokeApiKey(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleEmail(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.emailHandler.Get(w, r)
	case http.MethodPatch:
		a.emailHandler.Update(w, r)
	case http.MethodDelete:
		a.emailHandler.Delete(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleCalendar(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.calendarHandler.Get(w, r)
	case http.MethodPatch:
		a.calendarHandler.Update(w, r)
	case http.MethodDelete:
		a.calendarHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleWorkflow(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.workflowHandler.Get(w, r)
	case http.MethodPatch:
		a.workflowHandler.Update(w, r)
	case http.MethodDelete:
		a.workflowHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleIntegration(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.integrationHandler.Get(w, r)
	case http.MethodPatch:
		a.integrationHandler.Update(w, r)
	case http.MethodDelete:
		a.integrationHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleWebhook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.webhookHandler.Get(w, r)
	case http.MethodPatch:
		a.webhookHandler.Update(w, r)
	case http.MethodDelete:
		a.webhookHandler.Archive(w, r)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *App) handleMCP(w http.ResponseWriter, r *http.Request) {
	actor := auth.GetAuth(r)
	if actor.ActorID == "" {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			jsonutil.Error(w, http.StatusUnauthorized, "unauthorized", "API key required")
			return
		}
		key, err := a.agentService.ValidateApiKey(r.Context(), uuid.HashAPIKey(parts[1]))
		if errors.Is(err, agents.ErrRateLimited) {
			w.Header().Set("Retry-After", "60")
			jsonutil.Error(w, 429, "rate_limited", err.Error())
			return
		}
		if err != nil {
			jsonutil.Error(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
			return
		}
		actor = permissions.AuthContext{ActorType: "agent", ActorID: key.AgentID, WorkspaceID: key.WorkspaceID, Scopes: key.Scopes}
	}
	r = r.WithContext(auth.WithAuth(r.Context(), actor))
	if a.mcpServer != nil {
		a.mcpServer.ServeHTTP(w, r)
	} else {
		jsonutil.Error(w, http.StatusNotImplemented, "not_implemented", "MCP not configured")
	}
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": a.cfg.Server.Version,
		"build":   a.cfg.Server.BuildID,
	})
}

func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if a.db == nil || !a.db.Ready() {
		jsonutil.Error(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"status": "ready"})
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	jsonutil.Write(w, http.StatusOK, a.statusPayload())
}

func (a *App) statusPayload() map[string]any {
	return map[string]any{
		"version":  a.cfg.Server.Version,
		"build":    a.cfg.Server.BuildID,
		"env":      a.cfg.Server.Env,
		"uptime":   a.srv.Uptime().String(),
		"database": a.db.Status(),
		"jobs":     a.jobs.Status(),
		"memory":   a.srv.MemoryStatus(),
	}
}

var _ = time.Now

// Package mcp implements the Model Context Protocol (MCP) server for CRM.
package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"crm.local/internal/auth"
	"crm.local/internal/metrics"
	"crm.local/internal/model"
	"crm.local/internal/repository"
	"crm.local/internal/service/admin"
	"crm.local/internal/service/calendar"
	"crm.local/internal/service/companies"
	"crm.local/internal/service/contacts"
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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CRMTools wraps all CRM services for MCP.
type CRMTools struct {
	companySvc     *companies.Service
	contactSvc     *contacts.Service
	leadSvc        *leads.Service
	dealSvc        *deals.Service
	taskSvc        *tasks.Service
	projectSvc     *projects.Service
	searchSvc      *search.Service
	emailSvc       *email.Service
	calendarSvc    *calendar.Service
	workflowSvc    *workflows.Service
	integrationSvc *integrations.Service
	webhookSvc     *webhooks.Service
	adminSvc       *admin.Service
	log            *slog.Logger
}

// NewCRMTools creates the CRM tools bundle.
func NewCRMTools(
	companySvc *companies.Service,
	contactSvc *contacts.Service,
	leadSvc *leads.Service,
	dealSvc *deals.Service,
	taskSvc *tasks.Service,
	projectSvc *projects.Service,
	searchSvc *search.Service,
	emailSvc *email.Service,
	calendarSvc *calendar.Service,
	workflowSvc *workflows.Service,
	integrationSvc *integrations.Service,
	webhookSvc *webhooks.Service,
	adminSvc *admin.Service,
	log *slog.Logger,
) *CRMTools {
	return &CRMTools{
		companySvc:     companySvc,
		contactSvc:     contactSvc,
		leadSvc:        leadSvc,
		dealSvc:        dealSvc,
		taskSvc:        taskSvc,
		projectSvc:     projectSvc,
		searchSvc:      searchSvc,
		emailSvc:       emailSvc,
		calendarSvc:    calendarSvc,
		workflowSvc:    workflowSvc,
		integrationSvc: integrationSvc,
		webhookSvc:     webhookSvc,
		adminSvc:       adminSvc,
		log:            log,
	}
}

// Server wraps the MCP server.
type Server struct {
	handler http.Handler
	tools   *CRMTools
}

// Instrumentation bundles the optional observability hooks for MCP tool calls.
type Instrumentation struct {
	Metrics *metrics.Registry        // Prometheus counters, may be nil
	Repo    *repository.OAuthRepository // audit persistence, may be nil
	Log     *slog.Logger
}

// NewServer creates and configures the MCP server with all CRM tools.
// inst may be nil to disable per-tool observability.
func NewServer(tools *CRMTools, inst *Instrumentation) *Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: "crm", Version: "0.1.0"},
		nil,
	)

	crmTools := tools

	// Register all tools
	registerCompanyTools(s, crmTools)
	registerContactTools(s, crmTools)
	registerLeadTools(s, crmTools)
	registerDealTools(s, crmTools)
	registerTaskTools(s, crmTools)
	registerProjectTools(s, crmTools)
	registerSearchTools(s, crmTools)
	registerEmailTools(s, crmTools)
	registerCalendarTools(s, crmTools)
	registerWorkflowTools(s, crmTools)
	registerIntegrationTools(s, crmTools)
	registerWebhookTools(s, crmTools)
	registerAdminTools(s, crmTools)

	// Per-tool-call observability: count, time and audit every tools/call.
	if inst != nil {
		s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
			return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
				if method != "tools/call" {
					return next(ctx, method, req)
				}
				tool := ""
				if p, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
					tool = p.Name
				} else if raw, err := json.Marshal(req.GetParams()); err == nil {
					var probe struct {
						Name string `json:"name"`
					}
					if json.Unmarshal(raw, &probe) == nil {
						tool = probe.Name
					}
				}
				start := time.Now()
				res, err := next(ctx, method, req)
				dur := time.Since(start)
				status := "ok"
				errMsg := ""
				if err != nil {
					status = "error"
					errMsg = err.Error()
				} else if r, ok := res.(*mcp.CallToolResult); ok && r.IsError {
					status = "error"
					if len(r.Content) > 0 {
						if tc, ok := r.Content[0].(*mcp.TextContent); ok {
							errMsg = tc.Text
						}
					}
				}
				if inst.Metrics != nil {
					inst.Metrics.RecordMCPToolCall(tool, status, dur)
				}
				if inst.Repo != nil {
					actor := auth.FromContext(ctx)
					ip := ""
					if req.GetExtra() != nil && req.GetExtra().Header != nil {
						ip = req.GetExtra().Header.Get("X-Forwarded-For")
					}
					rec := &model.MCPToolCall{
						ID: uuid.New(), WorkspaceID: actor.WorkspaceID,
						ActorType: actor.ActorType, ActorID: actor.ActorID,
						Tool: tool, Status: status, DurationUS: dur.Microseconds(),
						IPAddress: ip, CreatedAt: time.Now().UTC(),
					}
					if errMsg != "" {
						rec.Error = &errMsg
					}
					bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
					defer cancel()
					if rerr := inst.Repo.RecordMCPToolCall(bg, rec); rerr != nil && inst.Log != nil {
						inst.Log.Error("mcp tool call audit write failed", "error", rerr, "tool", tool)
					}
				}
				if inst.Log != nil {
					inst.Log.Info("mcp tool call", "tool", tool, "status", status, "duration_us", dur.Microseconds())
				}
				return res, err
			}
		})
	}

	// Create streamable HTTP handler
	streamableHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)

	return &Server{handler: streamableHandler, tools: tools}
}

// ServeHTTP handles MCP HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	actor := auth.FromContext(r.Context())
	if actor.ActorID == "" || actor.WorkspaceID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.handler.ServeHTTP(w, r)
}

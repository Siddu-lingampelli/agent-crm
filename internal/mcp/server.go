// Package mcp implements the Model Context Protocol (MCP) server for CRM.
package mcp

import (
	"crm.local/internal/auth"
	"log/slog"
	"net/http"

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

// NewServer creates and configures the MCP server with all CRM tools.
func NewServer(tools *CRMTools) *Server {
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

// Package mcp — all CRM tool registrations using typed handlers.
package mcp

import (
	"context"
	"crm.local/internal/auth"

	"crm.local/internal/permissions"
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

// Company tools
func registerCompanyTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "companies.list", Description: "List companies with pagination and filters"}, t.listCompanies)
	mcp.AddTool(s, &mcp.Tool{Name: "companies.get", Description: "Get a company by ID"}, t.getCompany)
	mcp.AddTool(s, &mcp.Tool{Name: "companies.create", Description: "Create a new company"}, t.createCompany)
	mcp.AddTool(s, &mcp.Tool{Name: "companies.update", Description: "Update a company"}, t.updateCompany)
	mcp.AddTool(s, &mcp.Tool{Name: "companies.archive", Description: "Archive a company"}, t.archiveCompany)
}

// Contact tools
func registerContactTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "contacts.list", Description: "List contacts with pagination and filters"}, t.listContacts)
	mcp.AddTool(s, &mcp.Tool{Name: "contacts.get", Description: "Get a contact by ID"}, t.getContact)
	mcp.AddTool(s, &mcp.Tool{Name: "contacts.create", Description: "Create a new contact"}, t.createContact)
	mcp.AddTool(s, &mcp.Tool{Name: "contacts.update", Description: "Update a contact"}, t.updateContact)
	mcp.AddTool(s, &mcp.Tool{Name: "contacts.archive", Description: "Archive a contact"}, t.archiveContact)
}

// Lead tools
func registerLeadTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "leads.list", Description: "List leads with pagination and filters"}, t.listLeads)
	mcp.AddTool(s, &mcp.Tool{Name: "leads.get", Description: "Get a lead by ID"}, t.getLead)
	mcp.AddTool(s, &mcp.Tool{Name: "leads.create", Description: "Create a new lead"}, t.createLead)
	mcp.AddTool(s, &mcp.Tool{Name: "leads.update", Description: "Update a lead"}, t.updateLead)
	mcp.AddTool(s, &mcp.Tool{Name: "leads.archive", Description: "Archive a lead"}, t.archiveLead)
	mcp.AddTool(s, &mcp.Tool{Name: "leads.convert", Description: "Convert a lead to contact/company/deal"}, t.convertLead)
}

// Deal tools
func registerDealTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "deals.list", Description: "List deals with pagination and filters"}, t.listDeals)
	mcp.AddTool(s, &mcp.Tool{Name: "deals.get", Description: "Get a deal by ID"}, t.getDeal)
	mcp.AddTool(s, &mcp.Tool{Name: "deals.create", Description: "Create a new deal"}, t.createDeal)
	mcp.AddTool(s, &mcp.Tool{Name: "deals.update", Description: "Update a deal"}, t.updateDeal)
	mcp.AddTool(s, &mcp.Tool{Name: "deals.archive", Description: "Archive a deal"}, t.archiveDeal)
}

// Task tools
func registerTaskTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "tasks.list", Description: "List tasks with pagination and filters"}, t.listTasks)
	mcp.AddTool(s, &mcp.Tool{Name: "tasks.get", Description: "Get a task by ID"}, t.getTask)
	mcp.AddTool(s, &mcp.Tool{Name: "tasks.create", Description: "Create a new task"}, t.createTask)
	mcp.AddTool(s, &mcp.Tool{Name: "tasks.update", Description: "Update a task"}, t.updateTask)
	mcp.AddTool(s, &mcp.Tool{Name: "tasks.archive", Description: "Archive a task"}, t.archiveTask)
	mcp.AddTool(s, &mcp.Tool{Name: "tasks.complete", Description: "Complete a task"}, t.completeTask)
}

// Project tools
func registerProjectTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "projects.list", Description: "List projects with pagination and filters"}, t.listProjects)
	mcp.AddTool(s, &mcp.Tool{Name: "projects.get", Description: "Get a project by ID"}, t.getProject)
	mcp.AddTool(s, &mcp.Tool{Name: "projects.create", Description: "Create a new project"}, t.createProject)
	mcp.AddTool(s, &mcp.Tool{Name: "projects.update", Description: "Update a project"}, t.updateProject)
	mcp.AddTool(s, &mcp.Tool{Name: "projects.archive", Description: "Archive a project"}, t.archiveProject)
}

// Search tools
func registerSearchTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "search.global", Description: "Global full-text search across all entities"}, t.searchGlobal)
}

// Email tools
func registerEmailTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "emails.list", Description: "List emails with pagination and filters"}, t.listEmails)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.get", Description: "Get an email by ID"}, t.getEmail)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.create", Description: "Create a new email draft"}, t.createEmail)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.update", Description: "Update an email draft"}, t.updateEmail)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.delete", Description: "Delete an email draft"}, t.archiveEmail)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.approve", Description: "Approve an email for sending"}, t.approveEmail)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.send", Description: "Send an email"}, t.sendEmail)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.retry", Description: "Retry failed emails"}, t.retryFailedEmails)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.templates.list", Description: "List email templates"}, t.listEmailTemplates)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.templates.get", Description: "Get an email template by ID"}, t.getEmailTemplate)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.templates.create", Description: "Create a new email template"}, t.createEmailTemplate)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.templates.update", Description: "Update an email template"}, t.updateEmailTemplate)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.templates.delete", Description: "Delete an email template"}, t.archiveEmailTemplate)
	mcp.AddTool(s, &mcp.Tool{Name: "emails.templates.apply", Description: "Apply a template to create a draft"}, t.applyEmailTemplate)
}

// Calendar tools
func registerCalendarTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "calendar_events.list", Description: "List calendar events with pagination and filters"}, t.listCalendarEvents)
	mcp.AddTool(s, &mcp.Tool{Name: "calendar_events.get", Description: "Get a calendar event by ID"}, t.getCalendarEvent)
	mcp.AddTool(s, &mcp.Tool{Name: "calendar_events.create", Description: "Create a new calendar event"}, t.createCalendarEvent)
	mcp.AddTool(s, &mcp.Tool{Name: "calendar_events.update", Description: "Update a calendar event"}, t.updateCalendarEvent)
	mcp.AddTool(s, &mcp.Tool{Name: "calendar_events.delete", Description: "Delete a calendar event"}, t.archiveCalendarEvent)
}

// Workflow tools
func registerWorkflowTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.list", Description: "List workflows with pagination and filters"}, t.listWorkflows)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.get", Description: "Get a workflow by ID"}, t.getWorkflow)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.create", Description: "Create a new workflow"}, t.createWorkflow)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.update", Description: "Update a workflow"}, t.updateWorkflow)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.delete", Description: "Delete a workflow"}, t.archiveWorkflow)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.execute", Description: "Execute a workflow"}, t.executeWorkflow)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.executions.list", Description: "List workflow executions"}, t.listWorkflowExecutions)
	mcp.AddTool(s, &mcp.Tool{Name: "workflows.executions.get", Description: "Get a workflow execution by ID"}, t.getWorkflowExecution)
}

// Integration tools
func registerIntegrationTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "integrations.list", Description: "List integrations with pagination and filters"}, t.listIntegrations)
	mcp.AddTool(s, &mcp.Tool{Name: "integrations.get", Description: "Get an integration by ID"}, t.getIntegration)
	mcp.AddTool(s, &mcp.Tool{Name: "integrations.create", Description: "Create a new integration"}, t.createIntegration)
	mcp.AddTool(s, &mcp.Tool{Name: "integrations.update", Description: "Update an integration"}, t.updateIntegration)
	mcp.AddTool(s, &mcp.Tool{Name: "integrations.delete", Description: "Delete an integration"}, t.archiveIntegration)
	mcp.AddTool(s, &mcp.Tool{Name: "integrations.sync", Description: "Sync an integration"}, t.syncIntegration)
}

// Webhook tools
func registerWebhookTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "webhooks.list", Description: "List webhooks with pagination and filters"}, t.listWebhooks)
	mcp.AddTool(s, &mcp.Tool{Name: "webhooks.get", Description: "Get a webhook by ID"}, t.getWebhook)
	mcp.AddTool(s, &mcp.Tool{Name: "webhooks.create", Description: "Create a new webhook"}, t.createWebhook)
	mcp.AddTool(s, &mcp.Tool{Name: "webhooks.update", Description: "Update a webhook"}, t.updateWebhook)
	mcp.AddTool(s, &mcp.Tool{Name: "webhooks.delete", Description: "Delete a webhook"}, t.archiveWebhook)
	mcp.AddTool(s, &mcp.Tool{Name: "webhooks.test", Description: "Test a webhook"}, t.testWebhook)
}

// Admin tools
func registerAdminTools(s *mcp.Server, t *CRMTools) {
	mcp.AddTool(s, &mcp.Tool{Name: "admin.audit_logs.list", Description: "List audit logs with pagination and filters"}, t.listAuditLogs)
	mcp.AddTool(s, &mcp.Tool{Name: "admin.settings.get", Description: "Get settings for a category"}, t.getSettings)
	mcp.AddTool(s, &mcp.Tool{Name: "admin.settings.set", Description: "Set a setting value"}, t.setSettings)
	mcp.AddTool(s, &mcp.Tool{Name: "admin.schema.get", Description: "Get database schema information"}, t.getSchema)
	mcp.AddTool(s, &mcp.Tool{Name: "admin.stats.get", Description: "Get system statistics"}, t.getStats)
}

// Company implementations
func (t *CRMTools) listCompanies(ctx context.Context, req *mcp.CallToolRequest, args listCompaniesArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := companies.ListInput{
		Query:  args.Query,
		Status: args.Status,
		Limit:  args.Limit,
		Offset: args.Offset,
		Sort:   args.Sort,
		Order:  args.Order,
	}
	page, err := t.companySvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"companies": page.Items, "total": page.Total}}, map[string]any{"companies": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getCompany(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	company, err := t.companySvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"company": company}}, map[string]any{"company": company}, nil
}

func (t *CRMTools) createCompany(ctx context.Context, req *mcp.CallToolRequest, args companies.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	company, err := t.companySvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"company": company}}, map[string]any{"company": company}, nil
}

func (t *CRMTools) updateCompany(ctx context.Context, req *mcp.CallToolRequest, args companies.UpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	company, err := t.companySvc.Update(ctx, actor, args.ID, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"company": company}}, map[string]any{"company": company}, nil
}

func (t *CRMTools) archiveCompany(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.companySvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Company archived"}}}, map[string]any{}, nil
}

// Contact implementations
func (t *CRMTools) listContacts(ctx context.Context, req *mcp.CallToolRequest, args listContactsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := contacts.ListInput{
		Query:  args.Query,
		Status: args.Status,
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	page, err := t.contactSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"contacts": page.Items, "total": page.Total}}, map[string]any{"contacts": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getContact(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	contact, err := t.contactSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"contact": contact}}, map[string]any{"contact": contact}, nil
}

func (t *CRMTools) createContact(ctx context.Context, req *mcp.CallToolRequest, args contacts.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	contact, err := t.contactSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"contact": contact}}, map[string]any{"contact": contact}, nil
}

func (t *CRMTools) updateContact(ctx context.Context, req *mcp.CallToolRequest, args contacts.UpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	contact, err := t.contactSvc.Update(ctx, actor, args.ID, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"contact": contact}}, map[string]any{"contact": contact}, nil
}

func (t *CRMTools) archiveContact(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.contactSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Contact archived"}}}, map[string]any{}, nil
}

// Lead implementations
func (t *CRMTools) listLeads(ctx context.Context, req *mcp.CallToolRequest, args listLeadsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := leads.ListInput{
		Query:  args.Query,
		Status: args.Status,
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	page, err := t.leadSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"leads": page.Items, "total": page.Total}}, map[string]any{"leads": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getLead(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	lead, err := t.leadSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"lead": lead}}, map[string]any{"lead": lead}, nil
}

func (t *CRMTools) createLead(ctx context.Context, req *mcp.CallToolRequest, args leads.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	lead, err := t.leadSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"lead": lead}}, map[string]any{"lead": lead}, nil
}

func (t *CRMTools) updateLead(ctx context.Context, req *mcp.CallToolRequest, args leads.UpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	lead, err := t.leadSvc.Update(ctx, actor, args.ID, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"lead": lead}}, map[string]any{"lead": lead}, nil
}

func (t *CRMTools) archiveLead(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.leadSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Lead archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) convertLead(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	leads.ConvertInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	lead, err := t.leadSvc.Convert(ctx, actor, args.ID, args.ConvertInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"lead": lead}}, map[string]any{"lead": lead}, nil
}

// Deal implementations
func (t *CRMTools) listDeals(ctx context.Context, req *mcp.CallToolRequest, args listDealsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := deals.ListInput{
		Query:  args.Query,
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	page, err := t.dealSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"deals": page.Items, "total": page.Total}}, map[string]any{"deals": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getDeal(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	deal, err := t.dealSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"deal": deal}}, map[string]any{"deal": deal}, nil
}

func (t *CRMTools) createDeal(ctx context.Context, req *mcp.CallToolRequest, args deals.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	deal, err := t.dealSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"deal": deal}}, map[string]any{"deal": deal}, nil
}

func (t *CRMTools) updateDeal(ctx context.Context, req *mcp.CallToolRequest, args deals.UpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	deal, err := t.dealSvc.Update(ctx, actor, args.ID, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"deal": deal}}, map[string]any{"deal": deal}, nil
}

func (t *CRMTools) archiveDeal(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.dealSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Deal archived"}}}, map[string]any{}, nil
}

// Task implementations
func (t *CRMTools) listTasks(ctx context.Context, req *mcp.CallToolRequest, args listTasksArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := tasks.ListInput{
		Query:  args.Query,
		Status: args.Status,
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	page, err := t.taskSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"tasks": page.Items, "total": page.Total}}, map[string]any{"tasks": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getTask(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	task, err := t.taskSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"task": task}}, map[string]any{"task": task}, nil
}

func (t *CRMTools) createTask(ctx context.Context, req *mcp.CallToolRequest, args tasks.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	task, err := t.taskSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"task": task}}, map[string]any{"task": task}, nil
}

func (t *CRMTools) updateTask(ctx context.Context, req *mcp.CallToolRequest, args tasks.UpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	task, err := t.taskSvc.Update(ctx, actor, args.ID, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"task": task}}, map[string]any{"task": task}, nil
}

func (t *CRMTools) archiveTask(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.taskSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Task archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) completeTask(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.taskSvc.Complete(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Task completed"}}}, map[string]any{}, nil
}

// Project implementations
func (t *CRMTools) listProjects(ctx context.Context, req *mcp.CallToolRequest, args listProjectsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := projects.ListInput{
		Query:  args.Query,
		Status: args.Status,
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	page, err := t.projectSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"projects": page.Items, "total": page.Total}}, map[string]any{"projects": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getProject(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	project, err := t.projectSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"project": project}}, map[string]any{"project": project}, nil
}

func (t *CRMTools) createProject(ctx context.Context, req *mcp.CallToolRequest, args projects.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	project, err := t.projectSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"project": project}}, map[string]any{"project": project}, nil
}

func (t *CRMTools) updateProject(ctx context.Context, req *mcp.CallToolRequest, args projects.UpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	project, err := t.projectSvc.Update(ctx, actor, args.ID, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"project": project}}, map[string]any{"project": project}, nil
}

func (t *CRMTools) archiveProject(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.projectSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Project archived"}}}, map[string]any{}, nil
}

// Search implementation
func (t *CRMTools) searchGlobal(ctx context.Context, req *mcp.CallToolRequest, args searchArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	opts := search.SearchOptions{
		Query:     args.Query,
		Limit:     args.Limit,
		Workspace: actor.WorkspaceID,
		Actor:     actor,
	}
	resp, err := t.searchSvc.Search(ctx, opts)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"results": resp.Results}}, map[string]any{"results": resp.Results}, nil
}

// Argument structs for typed handlers
type listCompaniesArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Order  string `json:"order,omitempty"`
}

type listContactsArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type listLeadsArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type listDealsArgs struct {
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type listTasksArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type listProjectsArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type getEntityArgs struct {
	ID string `json:"id"`
}

type searchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// Helpers
func mcpAuthContext(ctx context.Context) permissions.AuthContext { return auth.FromContext(ctx) }

func handleError(err error) *mcp.CallToolResult {
	if err == nil {
		return &mcp.CallToolResult{}
	}
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}
}

// Email implementations
func (t *CRMTools) listEmails(ctx context.Context, req *mcp.CallToolRequest, args listEmailsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := email.ListInput{
		Status:      args.Status,
		RelatedType: args.RelatedType,
		Limit:       args.Limit,
		Offset:      args.Offset,
	}
	page, err := t.emailSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"emails": page.Items, "total": page.Total}}, map[string]any{"emails": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getEmail(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	email, err := t.emailSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"email": email}}, map[string]any{"email": email}, nil
}

func (t *CRMTools) createEmail(ctx context.Context, req *mcp.CallToolRequest, args email.CreateDraftInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	email, err := t.emailSvc.CreateDraft(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"email": email}}, map[string]any{"email": email}, nil
}

func (t *CRMTools) updateEmail(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	email.UpdateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	email, err := t.emailSvc.Update(ctx, actor, args.ID, args.UpdateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"email": email}}, map[string]any{"email": email}, nil
}

func (t *CRMTools) archiveEmail(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.emailSvc.Delete(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Email archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) approveEmail(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	email, err := t.emailSvc.Approve(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"email": email}}, map[string]any{"email": email}, nil
}

func (t *CRMTools) sendEmail(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	email, err := t.emailSvc.Send(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"email": email}}, map[string]any{"email": email}, nil
}

func (t *CRMTools) retryFailedEmails(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	result, err := t.emailSvc.RetryFailed(ctx, actor)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"result": result}}, map[string]any{"result": result}, nil
}

// Email template implementations
func (t *CRMTools) listEmailTemplates(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	templates, err := t.emailSvc.ListTemplates(ctx, actor)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"templates": templates}}, map[string]any{"templates": templates}, nil
}

func (t *CRMTools) getEmailTemplate(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	template, err := t.emailSvc.GetTemplate(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"template": template}}, map[string]any{"template": template}, nil
}

func (t *CRMTools) createEmailTemplate(ctx context.Context, req *mcp.CallToolRequest, args email.CreateTemplateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	template, err := t.emailSvc.CreateTemplate(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"template": template}}, map[string]any{"template": template}, nil
}

func (t *CRMTools) updateEmailTemplate(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	email.UpdateTemplateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	template, err := t.emailSvc.UpdateTemplate(ctx, actor, args.ID, args.UpdateTemplateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"template": template}}, map[string]any{"template": template}, nil
}

func (t *CRMTools) archiveEmailTemplate(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.emailSvc.ArchiveTemplate(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Email template archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) applyEmailTemplate(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	email.ApplyTemplateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	email, err := t.emailSvc.ApplyTemplate(ctx, actor, args.ID, args.ApplyTemplateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"email": email}}, map[string]any{"email": email}, nil
}

// Calendar implementations
func (t *CRMTools) listCalendarEvents(ctx context.Context, req *mcp.CallToolRequest, args listCalendarEventsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := calendar.ListInput{
		Query:     args.Query,
		StartFrom: args.StartFrom,
		StartTo:   args.StartTo,
		Limit:     args.Limit,
		Offset:    args.Offset,
	}
	page, err := t.calendarSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"events": page.Items, "total": page.Total}}, map[string]any{"events": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getCalendarEvent(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	event, err := t.calendarSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"event": event}}, map[string]any{"event": event}, nil
}

func (t *CRMTools) createCalendarEvent(ctx context.Context, req *mcp.CallToolRequest, args calendar.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	event, err := t.calendarSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"event": event}}, map[string]any{"event": event}, nil
}

func (t *CRMTools) updateCalendarEvent(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	calendar.UpdateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	event, err := t.calendarSvc.Update(ctx, actor, args.ID, args.UpdateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"event": event}}, map[string]any{"event": event}, nil
}

func (t *CRMTools) archiveCalendarEvent(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.calendarSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Calendar event archived"}}}, map[string]any{}, nil
}

// Workflow implementations
func (t *CRMTools) listWorkflows(ctx context.Context, req *mcp.CallToolRequest, args listWorkflowsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := workflows.ListInput{
		Query:  args.Query,
		Status: args.Status,
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	page, err := t.workflowSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"workflows": page.Items, "total": page.Total}}, map[string]any{"workflows": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getWorkflow(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	workflow, err := t.workflowSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"workflow": workflow}}, map[string]any{"workflow": workflow}, nil
}

func (t *CRMTools) createWorkflow(ctx context.Context, req *mcp.CallToolRequest, args workflows.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	workflow, err := t.workflowSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"workflow": workflow}}, map[string]any{"workflow": workflow}, nil
}

func (t *CRMTools) updateWorkflow(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	workflows.UpdateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	workflow, err := t.workflowSvc.Update(ctx, actor, args.ID, args.UpdateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"workflow": workflow}}, map[string]any{"workflow": workflow}, nil
}

func (t *CRMTools) archiveWorkflow(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.workflowSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Workflow archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) executeWorkflow(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	TriggerData map[string]any `json:"trigger_data"`
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	execution, err := t.workflowSvc.Execute(ctx, actor, args.ID, args.TriggerData)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"execution": execution}}, map[string]any{"execution": execution}, nil
}

func (t *CRMTools) listWorkflowExecutions(ctx context.Context, req *mcp.CallToolRequest, args listWorkflowExecutionsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	executions, err := t.workflowSvc.ListExecutions(ctx, actor, args.WorkflowID, args.Limit, args.Offset)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"executions": executions, "total": len(executions)}}, map[string]any{"executions": executions, "total": len(executions)}, nil
}

func (t *CRMTools) getWorkflowExecution(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	execution, err := t.workflowSvc.GetExecution(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"execution": execution}}, map[string]any{"execution": execution}, nil
}

// Integration implementations
func (t *CRMTools) listIntegrations(ctx context.Context, req *mcp.CallToolRequest, args listIntegrationsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := integrations.ListInput{
		WorkspaceID: actor.WorkspaceID,
		Provider:    args.Provider,
		Status:      args.Status,
		Limit:       args.Limit,
		Offset:      args.Offset,
		Sort:        args.Sort,
		Order:       args.Order,
	}
	integrationsList, err := t.integrationSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"integrations": integrationsList, "total": len(integrationsList)}}, map[string]any{"integrations": integrationsList, "total": len(integrationsList)}, nil
}

func (t *CRMTools) getIntegration(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	integration, err := t.integrationSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"integration": integration}}, map[string]any{"integration": integration}, nil
}

func (t *CRMTools) createIntegration(ctx context.Context, req *mcp.CallToolRequest, args integrations.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	integration, err := t.integrationSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"integration": integration}}, map[string]any{"integration": integration}, nil
}

func (t *CRMTools) updateIntegration(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	integrations.UpdateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	integration, err := t.integrationSvc.Update(ctx, actor, args.ID, args.UpdateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"integration": integration}}, map[string]any{"integration": integration}, nil
}

func (t *CRMTools) archiveIntegration(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.integrationSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Integration archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) syncIntegration(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.integrationSvc.Sync(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Integration synced"}}}, map[string]any{}, nil
}

// Webhook implementations
func (t *CRMTools) listWebhooks(ctx context.Context, req *mcp.CallToolRequest, args listWebhooksArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := webhooks.ListInput{
		WorkspaceID: actor.WorkspaceID,
		Status:      args.Status,
		Limit:       args.Limit,
		Offset:      args.Offset,
		Sort:        args.Sort,
		Order:       args.Order,
	}
	webhooksList, err := t.webhookSvc.List(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"webhooks": webhooksList, "total": len(webhooksList)}}, map[string]any{"webhooks": webhooksList, "total": len(webhooksList)}, nil
}

func (t *CRMTools) getWebhook(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	webhook, err := t.webhookSvc.Get(ctx, actor, args.ID)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"webhook": webhook}}, map[string]any{"webhook": webhook}, nil
}

func (t *CRMTools) createWebhook(ctx context.Context, req *mcp.CallToolRequest, args webhooks.CreateInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	webhook, err := t.webhookSvc.Create(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"webhook": webhook}}, map[string]any{"webhook": webhook}, nil
}

func (t *CRMTools) updateWebhook(ctx context.Context, req *mcp.CallToolRequest, args struct {
	getEntityArgs
	webhooks.UpdateInput
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	webhook, err := t.webhookSvc.Update(ctx, actor, args.ID, args.UpdateInput)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"webhook": webhook}}, map[string]any{"webhook": webhook}, nil
}

func (t *CRMTools) archiveWebhook(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.webhookSvc.Archive(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Webhook archived"}}}, map[string]any{}, nil
}

func (t *CRMTools) testWebhook(ctx context.Context, req *mcp.CallToolRequest, args getEntityArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	if err := t.webhookSvc.Test(ctx, actor, args.ID); err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Webhook tested"}}}, map[string]any{}, nil
}

// Admin implementations
func (t *CRMTools) listAuditLogs(ctx context.Context, req *mcp.CallToolRequest, args listAuditLogsArgs) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	in := admin.ListAuditLogsInput{
		WorkspaceID:  actor.WorkspaceID,
		ActorType:    args.ActorType,
		ActorID:      args.ActorID,
		Action:       args.Query,
		ResourceType: args.ResourceType,
		ResourceID:   args.ResourceID,
		Limit:        args.Limit,
		Offset:       args.Offset,
		Sort:         args.Sort,
		Order:        args.Order,
	}
	page, err := t.adminSvc.ListAuditLogs(ctx, actor, in)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"logs": page.Items, "total": page.Total}}, map[string]any{"logs": page.Items, "total": page.Total}, nil
}

func (t *CRMTools) getSettings(ctx context.Context, req *mcp.CallToolRequest, args struct {
	Category string `json:"category,omitempty"`
}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	settings, err := t.adminSvc.ListSettings(ctx, actor, args.Category)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"settings": settings}}, map[string]any{"settings": settings}, nil
}

func (t *CRMTools) setSettings(ctx context.Context, req *mcp.CallToolRequest, args admin.SetSettingInput) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	_, err := t.adminSvc.SetSetting(ctx, actor, args)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Settings updated"}}}, map[string]any{}, nil
}

func (t *CRMTools) getSchema(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	schema, err := t.adminSvc.GetSchemaTables(ctx, actor)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"schema": schema}}, map[string]any{"schema": schema}, nil
}

func (t *CRMTools) getStats(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, map[string]any, error) {
	actor := mcpAuthContext(ctx)
	workflowStats, err := t.adminSvc.GetWorkflowStats(ctx, actor)
	if err != nil {
		return handleError(err), nil, err
	}
	integrationStats, err := t.adminSvc.GetIntegrationStats(ctx, actor)
	if err != nil {
		return handleError(err), nil, err
	}
	agentStats, err := t.adminSvc.GetAgentStats(ctx, actor)
	if err != nil {
		return handleError(err), nil, err
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"workflows": workflowStats, "integrations": integrationStats, "agents": agentStats}}, map[string]any{"workflows": workflowStats, "integrations": integrationStats, "agents": agentStats}, nil
}

// Argument structs for typed handlers (new ones for Level 7)
type listEmailsArgs struct {
	Status      string `json:"status,omitempty"`
	RelatedType string `json:"related_type,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

type listCalendarEventsArgs struct {
	Query     string `json:"query,omitempty"`
	StartFrom string `json:"start_from,omitempty"`
	StartTo   string `json:"start_to,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type listWorkflowsArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type listWorkflowExecutionsArgs struct {
	WorkflowID string `json:"workflow_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

type listIntegrationsArgs struct {
	Query    string `json:"query,omitempty"`
	Provider string `json:"provider,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
	Sort     string `json:"sort,omitempty"`
	Order    string `json:"order,omitempty"`
}

type listWebhooksArgs struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Order  string `json:"order,omitempty"`
}

type listAuditLogsArgs struct {
	Query        string `json:"query,omitempty"`
	ActorType    string `json:"actor_type,omitempty"`
	ActorID      string `json:"actor_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	Sort         string `json:"sort,omitempty"`
	Order        string `json:"order,omitempty"`
}

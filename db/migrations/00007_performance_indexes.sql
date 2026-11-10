-- goose dialect: sqlite3
-- +goose Up
-- Level 9: Performance indexes for high-traffic queries
-- Added based on query plan analysis of list/search/report queries

-- Companies: search by name, domain, industry (common filters)
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(name);
CREATE INDEX IF NOT EXISTS idx_companies_ws_name ON companies(workspace_id, name);
CREATE INDEX IF NOT EXISTS idx_companies_is_client ON companies(workspace_id, is_client, archived_at);

-- Contacts: search by email, company, name
CREATE INDEX IF NOT EXISTS idx_contacts_ws_email ON contacts(workspace_id, email, archived_at);
CREATE INDEX IF NOT EXISTS idx_contacts_ws_name ON contacts(workspace_id, first_name, last_name, archived_at);

-- Leads: search by email, status, company
CREATE INDEX IF NOT EXISTS idx_leads_ws_email ON leads(workspace_id, email, archived_at);
CREATE INDEX IF NOT EXISTS idx_leads_ws_company ON leads(workspace_id, company_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_leads_source ON leads(workspace_id, source, archived_at);

-- Deals: pipeline reports, forecasts, stage breakdowns
CREATE INDEX IF NOT EXISTS idx_deals_ws_pipeline_stage ON deals(workspace_id, pipeline_id, stage_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_deals_ws_expected_close ON deals(workspace_id, expected_close, archived_at);
CREATE INDEX IF NOT EXISTS idx_deals_ws_amount ON deals(workspace_id, amount, archived_at);
CREATE INDEX IF NOT EXISTS idx_deals_company ON deals(workspace_id, company_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_deals_lead ON deals(workspace_id, lead_id, archived_at);

-- Tasks: overdue, due today, by assignee
CREATE INDEX IF NOT EXISTS idx_tasks_ws_assignee ON tasks(workspace_id, assignee_id, status, due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_ws_due_date ON tasks(workspace_id, due_date, status);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(workspace_id, project_id, status);

-- Projects: by company, by status
CREATE INDEX IF NOT EXISTS idx_projects_company ON projects(workspace_id, company_id, status);

-- Activities: customer timeline (high-traffic)
CREATE INDEX IF NOT EXISTS idx_activities_ws_related_time ON activities(workspace_id, related_type, related_id, occurred_at);

-- Emails: by related entity, by status
CREATE INDEX IF NOT EXISTS idx_emails_ws_related ON emails(workspace_id, related_type, related_id, created_at);
CREATE INDEX IF NOT EXISTS idx_emails_ws_status ON emails(workspace_id, status);

-- Workflows: by next run time (scheduler)
CREATE INDEX IF NOT EXISTS idx_workflows_ws_next_run ON workflows(workspace_id, next_run_at, status);

-- Workflow executions: by workflow, by status
CREATE INDEX IF NOT EXISTS idx_workflow_execs_ws_status ON workflow_executions(workspace_id, status, created_at);

-- Calendar events: by time range, by organizer
CREATE INDEX IF NOT EXISTS idx_calendar_ws_time_range ON calendar_events(workspace_id, start_time, end_time);

-- Audit logs: compliance queries
CREATE INDEX IF NOT EXISTS idx_audit_logs_ws_action ON audit_logs(workspace_id, action, created_at);

-- Webhooks: by status for delivery
CREATE INDEX IF NOT EXISTS idx_webhooks_ws_status ON webhooks(workspace_id, status);

-- Integrations: by provider, status
CREATE INDEX IF NOT EXISTS idx_integrations_ws_provider ON integrations(workspace_id, provider, status);

-- Settings: by category
CREATE INDEX IF NOT EXISTS idx_settings_ws_category ON settings(workspace_id, category);

-- +goose Down
DROP INDEX IF EXISTS idx_settings_ws_category;
DROP INDEX IF EXISTS idx_integrations_ws_provider;
DROP INDEX IF EXISTS idx_webhooks_ws_status;
DROP INDEX IF EXISTS idx_audit_logs_ws_action;
DROP INDEX IF EXISTS idx_calendar_ws_time_range;
DROP INDEX IF EXISTS idx_workflow_execs_ws_status;
DROP INDEX IF EXISTS idx_workflows_ws_next_run;
DROP INDEX IF EXISTS idx_emails_ws_status;
DROP INDEX IF EXISTS idx_emails_ws_related;
DROP INDEX IF EXISTS idx_activities_ws_related_time;
DROP INDEX IF EXISTS idx_projects_company;
DROP INDEX IF EXISTS idx_tasks_project;
DROP INDEX IF EXISTS idx_tasks_ws_due_date;
DROP INDEX IF EXISTS idx_tasks_ws_assignee;
DROP INDEX IF EXISTS idx_deals_lead;
DROP INDEX IF EXISTS idx_deals_company;
DROP INDEX IF EXISTS idx_deals_ws_amount;
DROP INDEX IF EXISTS idx_deals_ws_expected_close;
DROP INDEX IF EXISTS idx_deals_ws_pipeline_stage;
DROP INDEX IF EXISTS idx_leads_source;
DROP INDEX IF EXISTS idx_leads_ws_company;
DROP INDEX IF EXISTS idx_leads_ws_email;
DROP INDEX IF EXISTS idx_contacts_ws_name;
DROP INDEX IF EXISTS idx_contacts_ws_email;
DROP INDEX IF EXISTS idx_companies_is_client;
DROP INDEX IF EXISTS idx_companies_ws_name;
DROP INDEX IF EXISTS idx_companies_name;
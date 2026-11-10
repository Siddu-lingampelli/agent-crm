-- goose dialect: sqlite3
-- +goose Up
-- Level 7: Email, Calendar, Workflows, Integrations, Operational Administration
-- Creates tables for emails, email_templates, workflows, workflow_executions, calendar_events, integrations, webhooks, audit_logs, settings

-- Email tables
CREATE TABLE IF NOT EXISTS emails (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    to_addr TEXT NOT NULL,
    cc TEXT DEFAULT '',
    bcc TEXT DEFAULT '',
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    body_html TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft', -- draft, queued, sending, sent, failed
    provider TEXT NOT NULL DEFAULT 'smtp',
    provider_message_id TEXT DEFAULT '',
    related_type TEXT DEFAULT '',
    related_id TEXT DEFAULT '',
    created_by TEXT NOT NULL,
    approved_by TEXT DEFAULT '',
    approved_at TEXT,
    sent_at TEXT,
    failed_at TEXT,
    error_message TEXT DEFAULT '',
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_emails_workspace ON emails(workspace_id);
CREATE INDEX IF NOT EXISTS idx_emails_status ON emails(status);
CREATE INDEX IF NOT EXISTS idx_emails_related ON emails(related_type, related_id);
CREATE INDEX IF NOT EXISTS idx_emails_created_at ON emails(created_at);

CREATE TABLE IF NOT EXISTS email_templates (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    body_html TEXT DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_email_templates_workspace ON email_templates(workspace_id);

-- Workflow tables
CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    trigger_type TEXT NOT NULL, -- manual, schedule, event, webhook
    trigger_config TEXT NOT NULL DEFAULT '{}', -- JSON
    steps TEXT NOT NULL DEFAULT '[]', -- JSON array of WorkflowStep
    status TEXT NOT NULL DEFAULT 'draft', -- active, inactive, draft
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    last_run_at TEXT,
    next_run_at TEXT,
    run_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflows_workspace ON workflows(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_trigger_type ON workflows(trigger_type);
CREATE INDEX IF NOT EXISTS idx_workflows_next_run_at ON workflows(next_run_at);

CREATE TABLE IF NOT EXISTS workflow_executions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status TEXT NOT NULL, -- running, completed, failed, cancelled
    trigger_data TEXT NOT NULL DEFAULT '{}', -- JSON
    steps_run INTEGER NOT NULL DEFAULT 0,
    steps_total INTEGER NOT NULL DEFAULT 0,
    error_message TEXT DEFAULT '',
    started_at TEXT NOT NULL,
    completed_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_workspace ON workflow_executions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_created_at ON workflow_executions(created_at);

-- Calendar events
CREATE TABLE IF NOT EXISTS calendar_events (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    all_day INTEGER NOT NULL DEFAULT 0,
    location TEXT DEFAULT '',
    meeting_url TEXT DEFAULT '',
    organizer_id TEXT NOT NULL,
    attendee_ids TEXT NOT NULL DEFAULT '[]', -- JSON array
    related_type TEXT DEFAULT '',
    related_id TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'confirmed', -- confirmed, tentative, cancelled
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_workspace ON calendar_events(workspace_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_time ON calendar_events(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_organizer ON calendar_events(organizer_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_related ON calendar_events(related_type, related_id);

-- Integrations
CREATE TABLE IF NOT EXISTS integrations (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    provider TEXT NOT NULL, -- google, outlook, slack, hubspot, salesforce, etc.
    config TEXT NOT NULL DEFAULT '{}', -- JSON
    credentials TEXT DEFAULT '', -- encrypted credentials
    status TEXT NOT NULL DEFAULT 'disconnected', -- connected, disconnected, error
    last_sync_at TEXT,
    last_error TEXT DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_integrations_workspace ON integrations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_integrations_provider ON integrations(provider);
CREATE INDEX IF NOT EXISTS idx_integrations_status ON integrations(status);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    events TEXT NOT NULL DEFAULT '[]', -- JSON array of event types
    secret TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- active, inactive, disabled
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_triggered_at TEXT,
    last_success_at TEXT,
    last_failure_at TEXT,
    last_failure_reason TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhooks_workspace ON webhooks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);

-- Audit logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_type TEXT NOT NULL, -- user, agent, system
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    old_values TEXT DEFAULT '', -- JSON
    new_values TEXT DEFAULT '', -- JSON
    ip_address TEXT DEFAULT '',
    user_agent TEXT DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_workspace ON audit_logs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_type, actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Settings (workspace-level configuration)
CREATE TABLE IF NOT EXISTS settings (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    category TEXT NOT NULL, -- email, calendar, workflow, integration, security, ui
    key TEXT NOT NULL,
    value TEXT NOT NULL DEFAULT '', -- JSON
    description TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(workspace_id, category, key)
);

CREATE INDEX IF NOT EXISTS idx_settings_workspace ON settings(workspace_id);
CREATE INDEX IF NOT EXISTS idx_settings_category ON settings(category);

-- +goose Down
DROP INDEX IF EXISTS idx_settings_category;
DROP INDEX IF EXISTS idx_settings_workspace;
DROP TABLE IF EXISTS settings;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_actor;
DROP INDEX IF EXISTS idx_audit_logs_workspace;
DROP TABLE IF EXISTS audit_logs;
DROP INDEX IF EXISTS idx_webhooks_status;
DROP INDEX IF EXISTS idx_webhooks_workspace;
DROP TABLE IF EXISTS webhooks;
DROP INDEX IF EXISTS idx_integrations_status;
DROP INDEX IF EXISTS idx_integrations_provider;
DROP INDEX IF EXISTS idx_integrations_workspace;
DROP TABLE IF EXISTS integrations;
DROP INDEX IF EXISTS idx_calendar_events_related;
DROP INDEX IF EXISTS idx_calendar_events_organizer;
DROP INDEX IF EXISTS idx_calendar_events_time;
DROP INDEX IF EXISTS idx_calendar_events_workspace;
DROP TABLE IF EXISTS calendar_events;
DROP INDEX IF EXISTS idx_workflow_executions_created_at;
DROP INDEX IF EXISTS idx_workflow_executions_status;
DROP INDEX IF EXISTS idx_workflow_executions_workspace;
DROP INDEX IF EXISTS idx_workflow_executions_workflow;
DROP TABLE IF EXISTS workflow_executions;
DROP INDEX IF EXISTS idx_workflows_next_run_at;
DROP INDEX IF EXISTS idx_workflows_trigger_type;
DROP INDEX IF EXISTS idx_workflows_status;
DROP INDEX IF EXISTS idx_workflows_workspace;
DROP TABLE IF EXISTS workflows;
DROP INDEX IF EXISTS idx_email_templates_workspace;
DROP TABLE IF EXISTS email_templates;
DROP INDEX IF EXISTS idx_emails_created_at;
DROP INDEX IF EXISTS idx_emails_related;
DROP INDEX IF EXISTS idx_emails_status;
DROP INDEX IF EXISTS idx_emails_workspace;
DROP TABLE IF EXISTS emails;
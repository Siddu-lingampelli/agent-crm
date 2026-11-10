-- +goose Up
-- Core CRM entities. Public IDs are UUID strings; never expose rowids.

-- ---------------------------------------------------------------- companies
CREATE TABLE IF NOT EXISTS companies (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    name          TEXT NOT NULL,
    domain        TEXT,
    industry      TEXT,
    size_band     TEXT,
    website       TEXT,
    phone         TEXT,
    email         TEXT,
    address       TEXT,
    is_client     INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'active',
    owner_id      TEXT,
    tags          TEXT NOT NULL DEFAULT '[]',
    custom_fields TEXT NOT NULL DEFAULT '{}',
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    archived_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_companies_ws ON companies(workspace_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_companies_domain ON companies(domain);

-- ----------------------------------------------------------------- contacts
CREATE TABLE IF NOT EXISTS contacts (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    first_name    TEXT NOT NULL,
    last_name     TEXT,
    email         TEXT,
    phone         TEXT,
    job_title     TEXT,
    company_id    TEXT,
    owner_id      TEXT,
    status        TEXT NOT NULL DEFAULT 'active',
    tags          TEXT NOT NULL DEFAULT '[]',
    custom_fields TEXT NOT NULL DEFAULT '{}',
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    archived_at   TEXT,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_contacts_ws ON contacts(workspace_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_contacts_company ON contacts(company_id);
CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);

-- ------------------------------------------------------------- pipelines
CREATE TABLE IF NOT EXISTS pipelines (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name         TEXT NOT NULL,
    is_default   INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id           TEXT PRIMARY KEY,
    pipeline_id  TEXT NOT NULL,
    name         TEXT NOT NULL,
    position     INTEGER NOT NULL,
    probability  INTEGER NOT NULL DEFAULT 0,
    category     TEXT NOT NULL DEFAULT 'open',
    created_at   TEXT NOT NULL,
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_stages_pipeline ON pipeline_stages(pipeline_id, position);

-- -------------------------------------------------------------------- leads
CREATE TABLE IF NOT EXISTS leads (
    id              TEXT PRIMARY KEY,
    workspace_id    TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    first_name      TEXT,
    last_name       TEXT,
    email           TEXT,
    phone           TEXT,
    company_name    TEXT,
    company_id      TEXT,
    job_title       TEXT,
    status          TEXT NOT NULL DEFAULT 'new',
    source          TEXT,
    owner_id        TEXT,
    priority        TEXT NOT NULL DEFAULT 'normal',
    estimated_value INTEGER,
    currency        TEXT,
    lead_score      INTEGER,
    expected_close  TEXT,
    follow_up_date  TEXT,
    tags            TEXT NOT NULL DEFAULT '[]',
    custom_fields   TEXT NOT NULL DEFAULT '{}',
    description     TEXT NOT NULL DEFAULT '',
    qualification   TEXT NOT NULL DEFAULT '',
    created_by      TEXT,
    updated_by      TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    archived_at     TEXT,
    converted_at    TEXT,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_leads_ws ON leads(workspace_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_leads_status ON leads(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(email);

-- -------------------------------------------------------------------- deals
CREATE TABLE IF NOT EXISTS deals (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL,
    name           TEXT NOT NULL,
    pipeline_id    TEXT NOT NULL,
    stage_id       TEXT NOT NULL,
    amount         INTEGER,
    currency       TEXT NOT NULL,
    probability    INTEGER NOT NULL DEFAULT 0,
    expected_close TEXT,
    company_id     TEXT,
    primary_contact_id TEXT,
    lead_id        TEXT,
    owner_id       TEXT,
    source         TEXT,
    description    TEXT NOT NULL DEFAULT '',
    lost_reason    TEXT,
    won_at         TEXT,
    lost_at        TEXT,
    custom_fields  TEXT NOT NULL DEFAULT '{}',
    created_by     TEXT,
    updated_by     TEXT,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL,
    archived_at    TEXT,
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id),
    FOREIGN KEY (stage_id) REFERENCES pipeline_stages(id),
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL,
    FOREIGN KEY (primary_contact_id) REFERENCES contacts(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_deals_ws ON deals(workspace_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(pipeline_id, stage_id);

-- -------------------------------------------------------------------- tasks
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'open',
    priority      TEXT NOT NULL DEFAULT 'normal',
    due_date      TEXT,
    assignee_id   TEXT,
    related_type  TEXT,
    related_id    TEXT,
    project_id    TEXT,
    created_by    TEXT,
    completed_at  TEXT,
    custom_fields TEXT NOT NULL DEFAULT '{}',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    archived_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_tasks_ws ON tasks(workspace_id, status, due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_related ON tasks(related_type, related_id);

-- ----------------------------------------------------------------- projects
CREATE TABLE IF NOT EXISTS projects (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'planning',
    priority      TEXT NOT NULL DEFAULT 'normal',
    owner_id      TEXT,
    company_id    TEXT,
    start_date    TEXT,
    end_date      TEXT,
    progress      INTEGER NOT NULL DEFAULT 0,
    custom_fields TEXT NOT NULL DEFAULT '{}',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    archived_at   TEXT,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_projects_ws ON projects(workspace_id, status);

-- --------------------------------------------------------------- activities
CREATE TABLE IF NOT EXISTS activities (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    type         TEXT NOT NULL,
    summary      TEXT NOT NULL,
    actor_type   TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    related_type TEXT,
    related_id   TEXT,
    metadata     TEXT NOT NULL DEFAULT '{}',
    occurred_at  TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_activities_related ON activities(related_type, related_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_activities_ws ON activities(workspace_id, occurred_at);

-- -------------------------------------------------------------------- notes
CREATE TABLE IF NOT EXISTS notes (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    body         TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    related_type TEXT,
    related_id   TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    archived_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_notes_related ON notes(related_type, related_id);

-- -------------------------------------------------------------- audit events
CREATE TABLE IF NOT EXISTS audit_events (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT,
    actor_type   TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    actor_name   TEXT,
    action       TEXT NOT NULL,
    resource     TEXT,
    resource_id  TEXT,
    transport    TEXT,
    result       TEXT,
    ip           TEXT,
    request_id   TEXT,
    diff         TEXT,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_ws ON audit_events(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_events(actor_type, actor_id);

-- ------------------------------------------------------------------ agents
CREATE TABLE IF NOT EXISTS agents (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    provider     TEXT,
    enabled      INTEGER NOT NULL DEFAULT 1,
    scopes       TEXT NOT NULL DEFAULT '[]',
    rate_limit   INTEGER,
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS api_keys (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    agent_id     TEXT NOT NULL,
    name         TEXT NOT NULL,
    prefix       TEXT NOT NULL,
    hash         TEXT NOT NULL,
    scopes       TEXT NOT NULL DEFAULT '[]',
    rate_limit   INTEGER,
    expires_at   TEXT,
    revoked_at   TEXT,
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(hash);

-- -------------------------------------------------------------- attachments
CREATE TABLE IF NOT EXISTS attachments (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    filename      TEXT NOT NULL,
    storage_name  TEXT NOT NULL,
    content_type  TEXT NOT NULL,
    size_bytes    INTEGER NOT NULL,
    related_type  TEXT,
    related_id    TEXT,
    uploaded_by   TEXT,
    created_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_attachments_related ON attachments(related_type, related_id);

-- --------------------------------------------------------------- saved views
CREATE TABLE IF NOT EXISTS saved_views (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    entity       TEXT NOT NULL,
    name         TEXT NOT NULL,
    filters      TEXT NOT NULL DEFAULT '{}',
    owner_id     TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS saved_views;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS deals;
DROP TABLE IF EXISTS leads;
DROP TABLE IF EXISTS pipeline_stages;
DROP TABLE IF EXISTS pipelines;
DROP TABLE IF EXISTS contacts;
DROP TABLE IF EXISTS companies;

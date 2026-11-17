-- +goose Up
-- Mail connections (IMAP inbox sync), inbound messages, and email engagement tracking.

CREATE TABLE IF NOT EXISTS mail_connections (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'imap', -- imap | gmail | outlook
    email_address TEXT NOT NULL,
    imap_host TEXT NOT NULL,
    imap_port INTEGER NOT NULL DEFAULT 993,
    use_tls INTEGER NOT NULL DEFAULT 1,
    username TEXT NOT NULL,
    password_sealed TEXT NOT NULL DEFAULT '', -- enc:v1:... via security.SealCredentials; empty when auth_mode=oauth
    auth_mode TEXT NOT NULL DEFAULT 'password', -- password | oauth
    oauth_provider TEXT NOT NULL DEFAULT '', -- google | microsoft (when auth_mode=oauth)
    oauth_sealed TEXT NOT NULL DEFAULT '', -- enc:v1:{access_token,refresh_token,expiry} JSON, sealed
    oauth_state TEXT NOT NULL DEFAULT '', -- pending connect state (nonce), cleared on callback
    mailbox TEXT NOT NULL DEFAULT 'INBOX',
    enabled INTEGER NOT NULL DEFAULT 1,
    poll_interval_seconds INTEGER NOT NULL DEFAULT 60,
    last_sync_at TEXT,
    last_error TEXT DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mail_connections_workspace ON mail_connections(workspace_id);

CREATE TABLE IF NOT EXISTS inbound_messages (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    connection_id TEXT NOT NULL REFERENCES mail_connections(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL, -- RFC 5322 Message-ID
    uid INTEGER NOT NULL,     -- IMAP UID (per connection+mailbox)
    thread_key TEXT NOT NULL DEFAULT '', -- References/Subject grouping
    from_addr TEXT NOT NULL DEFAULT '',
    from_name TEXT NOT NULL DEFAULT '',
    to_addr TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL DEFAULT '',
    body_text TEXT NOT NULL DEFAULT '',
    body_html TEXT NOT NULL DEFAULT '',
    snippet TEXT NOT NULL DEFAULT '',
    received_at TEXT NOT NULL,
    is_read INTEGER NOT NULL DEFAULT 0,
    is_archived INTEGER NOT NULL DEFAULT 0,
    related_type TEXT NOT NULL DEFAULT '', -- contact, lead, company, deal
    related_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(connection_id, uid)
);
CREATE INDEX IF NOT EXISTS idx_inbound_workspace ON inbound_messages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_inbound_connection ON inbound_messages(connection_id, uid);
CREATE INDEX IF NOT EXISTS idx_inbound_message_id ON inbound_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_inbound_related ON inbound_messages(related_type, related_id);
CREATE INDEX IF NOT EXISTS idx_inbound_received ON inbound_messages(received_at);

-- Engagement tracking for outbound emails: one row per tracked email, events in email_events.
CREATE TABLE IF NOT EXISTS email_tracking (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email_id TEXT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE, -- unguessable per-email token used in tracking URLs
    open_count INTEGER NOT NULL DEFAULT 0,
    click_count INTEGER NOT NULL DEFAULT 0,
    first_opened_at TEXT,
    last_opened_at TEXT,
    first_clicked_at TEXT,
    last_clicked_at TEXT,
    delivered INTEGER NOT NULL DEFAULT 0,
    bounced INTEGER NOT NULL DEFAULT 0,
    bounce_reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE(email_id)
);
CREATE INDEX IF NOT EXISTS idx_email_tracking_token ON email_tracking(token);
CREATE INDEX IF NOT EXISTS idx_email_tracking_workspace ON email_tracking(workspace_id);

CREATE TABLE IF NOT EXISTS email_events (
    id TEXT PRIMARY KEY,
    tracking_id TEXT NOT NULL REFERENCES email_tracking(id) ON DELETE CASCADE,
    email_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    type TEXT NOT NULL, -- open, click, delivered, bounced
    url TEXT NOT NULL DEFAULT '', -- for click events
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    occurred_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_email_events_tracking ON email_events(tracking_id);
CREATE INDEX IF NOT EXISTS idx_email_events_email ON email_events(email_id);
CREATE INDEX IF NOT EXISTS idx_email_events_type ON email_events(type, occurred_at);

-- +goose StatementBegin
-- OAuth connect flow state for mail providers (Google/Microsoft), stored per workspace.
CREATE TABLE IF NOT EXISTS mail_oauth_states (
    state TEXT PRIMARY KEY, -- random nonce used as OAuth 'state' and CSRF token
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- google | microsoft
    created_by TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_mail_oauth_states_workspace ON mail_oauth_states(workspace_id);
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_mail_oauth_states_workspace;
DROP TABLE IF EXISTS mail_oauth_states;
DROP INDEX IF EXISTS idx_email_events_type;
DROP INDEX IF EXISTS idx_email_events_email;
DROP INDEX IF EXISTS idx_email_events_tracking;
DROP TABLE IF EXISTS email_events;
DROP INDEX IF EXISTS idx_email_tracking_workspace;
DROP INDEX IF EXISTS idx_email_tracking_token;
DROP TABLE IF EXISTS email_tracking;
DROP INDEX IF EXISTS idx_inbound_received;
DROP INDEX IF EXISTS idx_inbound_related;
DROP INDEX IF EXISTS idx_inbound_message_id;
DROP INDEX IF EXISTS idx_inbound_connection;
DROP INDEX IF EXISTS idx_inbound_workspace;
DROP TABLE IF EXISTS inbound_messages;
DROP INDEX IF EXISTS idx_mail_connections_workspace;
DROP TABLE IF EXISTS mail_connections;

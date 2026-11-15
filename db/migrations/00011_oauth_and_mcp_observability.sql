-- +goose Up
-- Production-grade MCP support: OAuth clients/tokens for the MCP endpoint,
-- per-tool-call observability (metrics + audit trail).

CREATE TABLE IF NOT EXISTS oauth_clients (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    client_id TEXT NOT NULL UNIQUE,
    client_secret_hash TEXT,           -- NULL for public clients (PKCE only)
    name TEXT NOT NULL,
    redirect_uris TEXT NOT NULL,       -- JSON array
    scopes TEXT NOT NULL,              -- JSON array
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    code TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    scopes TEXT NOT NULL,              -- JSON array
    expires_at TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oauth_codes_client ON oauth_authorization_codes(client_id);

CREATE TABLE IF NOT EXISTS oauth_access_tokens (
    token_hash TEXT PRIMARY KEY,       -- SHA256 of the JWT/opaque token
    client_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    scopes TEXT NOT NULL,              -- JSON array
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user ON oauth_access_tokens(user_id);

-- Per-tool MCP call metrics/audit, queryable via admin APIs.
CREATE TABLE IF NOT EXISTS mcp_tool_calls (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    actor_type TEXT NOT NULL,          -- 'agent' or 'user'
    actor_id TEXT NOT NULL,
    tool TEXT NOT NULL,
    status TEXT NOT NULL,              -- 'ok' or 'error'
    duration_us INTEGER NOT NULL,
    error TEXT,
    ip_address TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_calls_workspace ON mcp_tool_calls(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_calls_tool ON mcp_tool_calls(tool, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS mcp_tool_calls;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_clients;

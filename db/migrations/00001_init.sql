-- goose dialect: sqlite3
-- +goose Up

PRAGMA foreign_keys = ON;

-- Workspace boundary. Single-owner first release, but the schema keeps an
-- explicit workspace column so multi-user support can be added later.
CREATE TABLE IF NOT EXISTS workspaces (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

-- The owner identity. This is the only user in the first release.
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    UNIQUE (workspace_id, email)
);

-- Sessions are opaque random tokens; only the hash is stored.
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Future-proofing for Level 2 authorization vocabulary.
CREATE TABLE IF NOT EXISTS scopes (
    name TEXT PRIMARY KEY
);

INSERT OR IGNORE INTO scopes (name) VALUES
    ('contacts:read'),
    ('contacts:write'),
    ('companies:read'),
    ('companies:write'),
    ('leads:read'),
    ('leads:write'),
    ('deals:read'),
    ('deals:write'),
    ('tasks:read'),
    ('tasks:write'),
    ('projects:read'),
    ('projects:write'),
    ('emails:read'),
    ('emails:send'),
    ('audit:read');

-- +goose Down

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS scopes;
DROP TABLE IF EXISTS workspaces;

-- goose dialect: sqlite3
-- +goose Up
-- FTS5 indexes for emails and workflows
CREATE VIRTUAL TABLE IF NOT EXISTS emails_fts USING fts5(
    id UNINDEXED, workspace_id UNINDEXED,
    to_addr, cc, bcc, subject, body,
    related_type UNINDEXED, related_id UNINDEXED,
    content='emails', content_rowid='rowid'
);

CREATE VIRTUAL TABLE IF NOT EXISTS workflows_fts USING fts5(
    id UNINDEXED, workspace_id UNINDEXED,
    name, description,
    content='workflows', content_rowid='rowid'
);

-- Triggers for emails_fts
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS emails_fts_ai AFTER INSERT ON emails BEGIN
    INSERT INTO emails_fts(rowid, id, workspace_id, to_addr, cc, bcc, subject, body, related_type, related_id) VALUES (new.rowid, new.id, new.workspace_id, new.to_addr, new.cc, new.bcc, new.subject, new.body, new.related_type, new.related_id);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS emails_fts_ad AFTER DELETE ON emails BEGIN
    INSERT INTO emails_fts(emails_fts, rowid, id, workspace_id, to_addr, cc, bcc, subject, body, related_type, related_id) VALUES ('delete', old.rowid, old.id, old.workspace_id, old.to_addr, old.cc, old.bcc, old.subject, old.body, old.related_type, old.related_id);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS emails_fts_au AFTER UPDATE ON emails BEGIN
    INSERT INTO emails_fts(emails_fts, rowid, id, workspace_id, to_addr, cc, bcc, subject, body, related_type, related_id) VALUES ('delete', old.rowid, old.id, old.workspace_id, old.to_addr, old.cc, old.bcc, old.subject, old.body, old.related_type, old.related_id);
    INSERT INTO emails_fts(rowid, id, workspace_id, to_addr, cc, bcc, subject, body, related_type, related_id) VALUES (new.rowid, new.id, new.workspace_id, new.to_addr, new.cc, new.bcc, new.subject, new.body, new.related_type, new.related_id);
END;
-- +goose StatementEnd

-- Triggers for workflows_fts
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS workflows_fts_ai AFTER INSERT ON workflows BEGIN
    INSERT INTO workflows_fts(rowid, id, workspace_id, name, description) VALUES (new.rowid, new.id, new.workspace_id, new.name, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS workflows_fts_ad AFTER DELETE ON workflows BEGIN
    INSERT INTO workflows_fts(workflows_fts, rowid, id, workspace_id, name, description) VALUES ('delete', old.rowid, old.id, old.workspace_id, old.name, old.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS workflows_fts_au AFTER UPDATE ON workflows BEGIN
    INSERT INTO workflows_fts(workflows_fts, rowid, id, workspace_id, name, description) VALUES ('delete', old.rowid, old.id, old.workspace_id, old.name, old.description);
    INSERT INTO workflows_fts(rowid, id, workspace_id, name, description) VALUES (new.rowid, new.id, new.workspace_id, new.name, new.description);
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS workflows_fts_au;
DROP TRIGGER IF EXISTS workflows_fts_ad;
DROP TRIGGER IF EXISTS workflows_fts_ai;
DROP TRIGGER IF EXISTS emails_fts_au;
DROP TRIGGER IF EXISTS emails_fts_ad;
DROP TRIGGER IF EXISTS emails_fts_ai;
-- +goose StatementEnd

DROP TABLE IF EXISTS workflows_fts;
DROP TABLE IF EXISTS emails_fts;
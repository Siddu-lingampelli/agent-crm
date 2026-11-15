-- +goose Up
-- Pipelines were missing the soft-archive column added to other resources in 00009.
ALTER TABLE pipelines ADD COLUMN archived_at TEXT;

-- Sync history for integrations, backing GET /api/v1/integrations/{id}/logs.
CREATE TABLE IF NOT EXISTS integration_sync_logs (
    id TEXT PRIMARY KEY,
    integration_id TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'success', -- success, failed, partial
    records_processed INTEGER NOT NULL DEFAULT 0,
    records_failed INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_integration_sync_logs_integration ON integration_sync_logs(integration_id, started_at);

-- +goose Down
DROP TABLE IF EXISTS integration_sync_logs;
ALTER TABLE pipelines DROP COLUMN archived_at;

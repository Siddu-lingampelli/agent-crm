-- +goose Up
ALTER TABLE calendar_events ADD COLUMN archived_at TEXT;
ALTER TABLE workflows ADD COLUMN archived_at TEXT;
ALTER TABLE integrations ADD COLUMN archived_at TEXT;
ALTER TABLE webhooks ADD COLUMN archived_at TEXT;
UPDATE pipelines SET workspace_id=(SELECT workspace_id FROM users LIMIT 1) WHERE workspace_id='ws_default' AND (SELECT COUNT(*) FROM users)=1;
-- +goose Down
ALTER TABLE webhooks DROP COLUMN archived_at;
ALTER TABLE integrations DROP COLUMN archived_at;
ALTER TABLE workflows DROP COLUMN archived_at;
ALTER TABLE calendar_events DROP COLUMN archived_at;

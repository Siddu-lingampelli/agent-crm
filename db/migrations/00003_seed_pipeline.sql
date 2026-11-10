-- +goose Up
-- Seed default pipeline and stages for the default workspace.

INSERT OR IGNORE INTO pipelines (id, workspace_id, name, is_default, created_at, updated_at)
VALUES ('pipeline_default', 'ws_default', 'Default Pipeline', 1,
        datetime('now'), datetime('now'));

INSERT OR IGNORE INTO pipeline_stages (id, pipeline_id, name, position, probability, category, created_at)
VALUES
    ('stage_discovery', 'pipeline_default', 'Discovery', 1, 20, 'open', datetime('now')),
    ('stage_proposal', 'pipeline_default', 'Proposal', 2, 50, 'open', datetime('now')),
    ('stage_negotiation', 'pipeline_default', 'Negotiation', 3, 75, 'open', datetime('now')),
    ('stage_won', 'pipeline_default', 'Won', 4, 100, 'won', datetime('now')),
    ('stage_lost', 'pipeline_default', 'Lost', 5, 0, 'lost', datetime('now'));

-- +goose Down
DELETE FROM pipeline_stages WHERE pipeline_id = 'pipeline_default';
DELETE FROM pipelines WHERE id = 'pipeline_default';
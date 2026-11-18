-- +goose Up
-- Sealed full API key material so owners can re-reveal a key (for MCP clients) without rotating.
ALTER TABLE api_keys ADD COLUMN key_sealed TEXT;

-- +goose Down
ALTER TABLE api_keys DROP COLUMN key_sealed;

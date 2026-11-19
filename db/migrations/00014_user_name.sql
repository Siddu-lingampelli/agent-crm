-- goose dialect: sqlite3
-- +goose Up

ALTER TABLE users ADD COLUMN display_name TEXT DEFAULT '';

-- +goose Down

-- SQLite cannot DROP a column with a DEFAULT in older versions; the column is
-- harmless to leave behind on rollback and is safe to ignore.
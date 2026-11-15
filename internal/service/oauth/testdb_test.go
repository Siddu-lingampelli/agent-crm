package oauth

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// openTestDB creates a temp SQLite DB with the oauth tables from the migration.
func openTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	migration, err := os.ReadFile("../../../db/migrations/00011_oauth_and_mcp_observability.sql")
	if err != nil {
		return nil, err
	}
	// strip goose markers and execute statements
	stmts := splitSQL(string(migration))
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return nil, err
		}
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, nil
}

func splitSQL(mig string) []string {
	// keep only the Up section
	up := mig
	if i := indexOf(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	if i := indexOf(up, "-- +goose Up"); i >= 0 {
		up = up[i:]
	}
	parts := []string{}
	cur := ""
	for _, line := range splitLines(up) {
		if line == "-- +goose StatementBegin" || line == "-- +goose StatementEnd" {
			continue
		}
		cur += line + "\n"
		if line != "" && line[len(line)-1] == ';' {
			parts = append(parts, cur)
			cur = ""
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func splitLines(s string) []string {
	out := []string{}
	cur := ""
	for _, c := range s {
		if c == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

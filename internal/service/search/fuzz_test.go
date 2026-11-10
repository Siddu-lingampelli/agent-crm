// internal/service/search/fuzz_test.go
package search

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func FuzzSearchQuery(f *testing.F) {
	testCases := []string{
		"test",
		"test query",
		"test*",
		"test?",
		"test AND query",
		"test OR query",
		"\"exact phrase\"",
		"field:value",
		"+required -excluded",
		strings.Repeat("a", 1000),
		"",
		" ",
		"' OR '1'='1",
		"<script>alert(1)</script>",
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, query string) {
		// Create in-memory database for fuzzing
		db, err := sql.Open("sqlite", "file::memory:?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// Create minimal schema for FTS
		_, err = db.Exec(`
			CREATE VIRTUAL TABLE fts_companies USING fts5(name, domain, industry, notes);
			CREATE VIRTUAL TABLE fts_contacts USING fts5(first_name, last_name, email, phone, job_title, notes);
			CREATE VIRTUAL TABLE fts_leads USING fts5(display_name, first_name, last_name, email, phone, company_name, notes);
			CREATE VIRTUAL TABLE fts_deals USING fts5(name, description);
			CREATE VIRTUAL TABLE fts_tasks USING fts5(title, description);
			CREATE VIRTUAL TABLE fts_projects USING fts5(name, description);
		`)
		if err != nil {
			t.Fatal(err)
		}

		svc := New(db)
		opts := SearchOptions{
			Query:     query,
			Workspace: "test",
			Limit:     10,
		}
		// Should not panic, should handle gracefully
		_, _ = svc.Search(context.Background(), opts)
	})
}

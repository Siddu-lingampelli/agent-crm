package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseURIAndConnectionPragmas(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "spaces & query # file.db")
	database, err := Open(ctx, Config{Path: filename, MaxOpenConns: 3, MaxIdleConns: 3, BusyTimeout: 1234 * time.Millisecond, PRAGMASynchronous: "NORMAL", PRAGMATempStore: "MEMORY"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("configured database was not created: %v", err)
	}
	for range 3 {
		conn, err := database.SQL().Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		for query, want := range map[string]int{"PRAGMA foreign_keys": 1, "PRAGMA busy_timeout": 1234, "PRAGMA synchronous": 1, "PRAGMA temp_store": 2} {
			var got int
			if err := conn.QueryRowContext(ctx, query).Scan(&got); err != nil || got != want {
				t.Fatalf("%s: got %d, want %d: %v", query, got, want, err)
			}
		}
	}
}

func TestLegacyFilenamePreservesData(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "legacy.db")
	cfg := Config{Path: filename, MaxOpenConns: 1, MaxIdleConns: 1}
	database, err := Open(ctx, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec("CREATE TABLE marker(value TEXT); INSERT INTO marker VALUES('preserved')"); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filename, filename+"&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"); err != nil {
		t.Fatal(err)
	}
	database, err = Open(ctx, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var value string
	if err := database.SQL().QueryRow("SELECT value FROM marker").Scan(&value); err != nil || value != "preserved" {
		t.Fatalf("lost legacy data: %s %v", value, err)
	}
}

package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenWithWAL(t *testing.T) {
	ctx := context.Background()
	tmpDBPath := filepath.Join(t.TempDir(), "wal.db")
	tmpDB, err := Open(ctx, Config{
		Path:              tmpDBPath,
		MaxOpenConns:      4,
		MaxIdleConns:      2,
		BusyTimeout:       5 * time.Second,
		PRAGMAJournalMode: "WAL",
	}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		_ = tmpDB.Close()
	}()

	// Debug: check the journal mode
	var journalMode string
	if err := tmpDB.SQL().QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("check journal mode: %v", err)
	}
	t.Logf("Journal mode after open: %s", journalMode)

	if journalMode != "wal" {
		t.Errorf("Expected journal_mode=wal, got %s", journalMode)
	}
}

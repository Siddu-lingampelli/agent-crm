// Package auth — authentication tests.
package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"crm.local/internal/db"
)

func TestOwnerBootstrap(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{
		Path:              filepath.Join(t.TempDir(), "auth.db"),
		MaxOpenConns:      4,
		MaxIdleConns:      2,
		BusyTimeout:       5 * time.Second,
		PRAGMAJournalMode: "WAL",
	}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	// Debug: check the journal mode
	var journalMode string
	if err := database.SQL().QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("check journal mode: %v", err)
	}
	t.Logf("Journal mode after open: %s", journalMode)

	// Bootstrap first owner
	err = OwnerBootstrap(ctx, database, "admin@test.com", "password123")
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// Login should work
	result, err := Login(ctx, database, "admin@test.com", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !result.Success {
		t.Fatal("Login: expected success")
	}
	if result.Token == "" {
		t.Fatal("Login: token empty")
	}

	// Validate session
	userID, err := ValidateSession(ctx, database, result.Token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if userID == "" {
		t.Fatal("ValidateSession: userID empty")
	}

	// Wrong password
	result, err = Login(ctx, database, "admin@test.com", "wrongpassword")
	if err != nil {
		t.Fatalf("Login wrong password: %v", err)
	}
	if result.Success {
		t.Fatal("Login wrong password: expected failure")
	}
}

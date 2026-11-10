// Package repository — contacts tests.
package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"crm.local/internal/db"
	"crm.local/internal/model"
	"crm.local/internal/uuid"
)

func TestContactRepository_CRUD(t *testing.T) {
	ctx := context.Background()
	tmpDBPath := filepath.Join(t.TempDir(), "contacts.db")
	tmpDB, err := db.Open(ctx, db.Config{
		Path:              tmpDBPath,
		MaxOpenConns:      2,
		MaxIdleConns:      1,
		BusyTimeout:       5 * time.Second,
		PRAGMAJournalMode: "WAL",
	}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		_ = tmpDB.Close()
	}()

	repo := NewContactRepository(tmpDB.SQL())

	c := &model.Contact{
		ID:           uuid.New(),
		WorkspaceID:  "ws_test",
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "john@test.com",
		Phone:        "+1-555-0100",
		JobTitle:     "CTO",
		CompanyID:    "",
		OwnerID:      "user-1",
		Status:       "active",
		Tags:         []string{"decision-maker"},
		CustomFields: map[string]any{"source": "referral"},
		Notes:        "Test contact",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Create
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get: contact not found")
	}
	if got.FirstName != c.FirstName || got.Email != c.Email {
		t.Errorf("Get: mismatched fields: got %+v", got)
	}

	// List
	page, err := repo.List(ctx, ContactListOptions{
		WorkspaceID: "ws_test",
		Limit:       25,
		Offset:      0,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Errorf("List: expected 1 item, got total=%d len=%d", page.Total, len(page.Items))
	}

	// Update
	c.FirstName = "Jane"
	c.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, _ := repo.Get(ctx, c.ID)
	if updated.FirstName != "Jane" {
		t.Errorf("Update: name not updated, got %s", updated.FirstName)
	}

	// Archive
	now := time.Now().UTC().Format(time.RFC3339)
	if err := repo.Archive(ctx, c.ID, now); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	archived, _ := repo.Get(ctx, c.ID)
	if archived != nil {
		t.Errorf("Archive: contact still found after archive")
	}
}

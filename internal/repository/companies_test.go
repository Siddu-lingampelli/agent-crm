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

func TestCompanyRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	// Create temporary database with unique path
	tmpDBPath := filepath.Join(t.TempDir(), "companies.db")
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

	repo := NewCompanyRepository(tmpDB.SQL())

	now := time.Now().UTC().Format(time.RFC3339)
	c := &model.Company{
		ID:           uuid.New(),
		WorkspaceID:  "ws_test",
		Name:         "Test Corp",
		Domain:       "test.com",
		Industry:     "Technology",
		SizeBand:     "11-50",
		Website:      "https://test.com",
		Phone:        "+1-555-0100",
		Email:        "contact@test.com",
		Address:      "123 Test St",
		IsClient:     true,
		Status:       "active",
		OwnerID:      "user-1",
		Tags:         []string{"enterprise", "partner"},
		CustomFields: map[string]any{"source": "referral"},
		Notes:        "Test notes",
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
		t.Fatal("Get: company not found")
	}
	if got.Name != c.Name || got.Domain != c.Domain {
		t.Errorf("Get: mismatched fields: got %+v", got)
	}

	// List
	page, err := repo.List(ctx, CompanyListOptions{
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
	c.Name = "Updated Corp"
	c.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, _ := repo.Get(ctx, c.ID)
	if updated.Name != "Updated Corp" {
		t.Errorf("Update: name not updated, got %s", updated.Name)
	}

	// Archive
	if err := repo.Archive(ctx, c.ID, now); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	archived, _ := repo.Get(ctx, c.ID)
	if archived != nil {
		t.Errorf("Archive: company still found after archive")
	}
}

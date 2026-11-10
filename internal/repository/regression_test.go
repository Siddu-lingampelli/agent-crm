package repository

import (
	"context"
	"crm.local/internal/db"
	"crm.local/internal/model"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceReferencesArchiveAndDashboardTotals(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{Path: filepath.Join(t.TempDir(), "regression.db"), MaxOpenConns: 2, MaxIdleConns: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	for _, workspace := range []string{"one", "two"} {
		if _, err := database.SQL().ExecContext(ctx, "INSERT INTO workspaces(id,name,created_at) VALUES(?,?,?)", workspace, workspace, now); err != nil {
			t.Fatal(err)
		}
	}
	companyRepo := NewCompanyRepository(database.SQL())
	if err := companyRepo.Create(ctx, &model.Company{ID: "company", WorkspaceID: "one", Name: "Private Company", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	contactRepo := NewContactRepository(database.SQL())
	err = contactRepo.Create(ctx, &model.Contact{ID: "contact", WorkspaceID: "two", FirstName: "Intruder", CompanyID: "company", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("cross-workspace reference accepted: %v", err)
	}
	webhookRepo := NewWebhookRepository(database.SQL())
	if err := webhookRepo.Create(ctx, &model.Webhook{ID: "hook", WorkspaceID: "one", Name: "Hook", URL: "https://example.com", Events: "[]", Secret: "secret", Status: "active", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := webhookRepo.Archive(ctx, "hook", now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if item, err := webhookRepo.Get(ctx, "hook"); err != nil || item != nil {
		t.Fatalf("archived webhook remains visible: %v", err)
	}
	tx, err := database.SQL().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for i := range 300 {
		if _, err := tx.ExecContext(ctx, "INSERT INTO leads(id,workspace_id,display_name,status,created_at,updated_at) VALUES(?,?,'Lead','new',?,?)", i, "one", now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	totals, err := NewDealRepository(database.SQL()).DashboardTotals(ctx, "one", now)
	if err != nil {
		t.Fatal(err)
	}
	if totals.NewLeads != 300 {
		t.Fatalf("dashboard truncated at page size: %d", totals.NewLeads)
	}
	if _, err := NewAdminRepository(database.SQL()).GetWorkflowStats(ctx, "empty"); err != nil {
		t.Fatal(err)
	}
}

package repository

import (
	"context"
	"crm.local/internal/db"
	"crm.local/internal/model"
	"path/filepath"
	"testing"
	"time"
)

func TestLeadConversionIsAtomicAndCreatesRelations(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{Path: filepath.Join(t.TempDir(), "conversion.db"), MaxOpenConns: 1, MaxIdleConns: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC()
	if _, err := database.SQL().ExecContext(ctx, "INSERT INTO workspaces(id,name,created_at) VALUES('one','One',?)", now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, "INSERT INTO pipelines(id,workspace_id,name,is_default,created_at,updated_at) VALUES('sales','one','Sales',1,?,?)", now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, "INSERT INTO pipeline_stages(id,pipeline_id,name,position,probability,category,created_at) VALUES('open','sales','Open',1,25,'open',?)", now); err != nil {
		t.Fatal(err)
	}
	repo := NewLeadRepository(database.SQL())
	if err := repo.Create(ctx, &model.Lead{ID: "lead", WorkspaceID: "one", DisplayName: "Test Person", CompanyName: "Test Company", Email: "person@example.com", Status: "new", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ConvertWithRelations(ctx, "one", "owner", "lead", LeadConversion{DealName: "Invalid deal", PipelineID: "sales", StageID: "invalid"}); err == nil {
		t.Fatal("invalid stage accepted")
	}
	var count int
	if err := database.SQL().QueryRow("SELECT COUNT(*) FROM companies WHERE workspace_id='one'").Scan(&count); err != nil || count != 0 {
		t.Fatal("failed conversion left a company")
	}
	if err := database.SQL().QueryRow("SELECT COUNT(*) FROM contacts WHERE workspace_id='one'").Scan(&count); err != nil || count != 0 {
		t.Fatal("failed conversion left a contact")
	}
	lead, err := repo.Get(ctx, "lead")
	if err != nil || lead.Status != "new" {
		t.Fatal("failed conversion changed lead status")
	}
	lead, err = repo.ConvertWithRelations(ctx, "one", "owner", "lead", LeadConversion{DealName: "New deal", Amount: 10000, Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	if lead.Status != "converted" || lead.CompanyID == "" {
		t.Fatal("conversion was not persisted")
	}
	if err := database.SQL().QueryRow("SELECT COUNT(*) FROM deals d JOIN contacts c ON c.id=d.primary_contact_id JOIN companies co ON co.id=d.company_id WHERE d.lead_id='lead' AND d.workspace_id='one' AND c.company_id=co.id").Scan(&count); err != nil || count != 1 {
		t.Fatalf("missing conversion relationships: %d %v", count, err)
	}
	if _, err := repo.ConvertWithRelations(ctx, "one", "owner", "lead", LeadConversion{DealName: "Duplicate"}); err == nil {
		t.Fatal("duplicate conversion accepted")
	}
}

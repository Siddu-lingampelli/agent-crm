package search

import (
	"context"
	"crm.local/internal/db"
	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"path/filepath"
	"testing"
	"time"
)

func TestSearchRowidsAfterUpdatesAndArchives(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{Path: filepath.Join(t.TempDir(), "search.db"), MaxOpenConns: 2, MaxIdleConns: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repo := repository.NewCompanyRepository(database.SQL())
	now := time.Now().UTC()
	for _, c := range []*model.Company{{ID: "one", WorkspaceID: "workspace", Name: "First", CreatedAt: now, UpdatedAt: now}, {ID: "two", WorkspaceID: "workspace", Name: "Second", CreatedAt: now, UpdatedAt: now}} {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	one, err := repo.Get(ctx, "one")
	if err != nil {
		t.Fatal(err)
	}
	one.Name = "UniqueUpdated"
	if err := repo.Update(ctx, one); err != nil {
		t.Fatal(err)
	}
	svc := New(database.SQL())
	opts := SearchOptions{Query: "UniqueUpdated", Actor: permissions.AuthContext{ActorID: "owner", WorkspaceID: "workspace", Scopes: []string{"companies:read"}}}
	result, err := svc.Search(ctx, opts)
	if err != nil || len(result.Results) != 1 || result.Results[0].EntityID != "one" {
		t.Fatalf("FTS rowid mapping corrupted: %+v %v", result, err)
	}
	opts.Actor.Scopes = nil
	result, err = svc.Search(ctx, opts)
	if err != nil || len(result.Results) != 0 {
		t.Fatal("search bypassed read scopes")
	}
	opts.Actor.Scopes = []string{"companies:read"}
	if err := repo.Archive(ctx, "one", time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	result, err = svc.Search(ctx, opts)
	if err != nil || len(result.Results) != 0 {
		t.Fatal("archived record visible in search")
	}
}

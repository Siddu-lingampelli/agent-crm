package email

import (
	"context"
	"crm.local/internal/auth"
	"crm.local/internal/db"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestAgentCannotApproveOrSendUnapprovedOwnDraft(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{Path: filepath.Join(t.TempDir(), "email.db"), MaxOpenConns: 2, MaxIdleConns: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := auth.OwnerBootstrap(ctx, database, "owner@test.local", "password123"); err != nil {
		t.Fatal(err)
	}
	var workspace, user string
	if err := database.SQL().QueryRow("SELECT workspace_id,id FROM users LIMIT 1").Scan(&workspace, &user); err != nil {
		t.Fatal(err)
	}
	svc := New(repository.NewEmailRepository(database.SQL()), slog.Default())
	agent := permissions.AuthContext{ActorID: "agent", ActorType: "agent", WorkspaceID: workspace, Scopes: []string{"emails:write", "emails:send"}}
	draft, err := svc.CreateDraft(ctx, agent, CreateDraftInput{To: "test@example.com", Subject: "Test", Body: "Draft"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Send(ctx, agent, draft.ID); err == nil {
		t.Fatal("agent sent its own unapproved draft")
	}
	if _, err := svc.Approve(ctx, agent, draft.ID); !errors.Is(err, permissions.ErrScopeDenied) {
		t.Fatalf("agent approved its own draft: %v", err)
	}
	owner := permissions.AuthContext{ActorID: user, ActorType: "user", WorkspaceID: workspace, Scopes: auth.OwnerScopes()}
	if _, err := svc.Approve(ctx, owner, draft.ID); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewEmailRepository(database.SQL())
	first, err := repo.ClaimForSending(ctx, draft.ID)
	if err != nil || !first {
		t.Fatal("approved draft could not be claimed")
	}
	if second, err := repo.ClaimForSending(ctx, draft.ID); err != nil || second {
		t.Fatal("same email claimed twice")
	}
}

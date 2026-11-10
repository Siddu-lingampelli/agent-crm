package auth

import (
	"context"
	"crm.local/internal/db"
	"crm.local/internal/permissions"
	"errors"
	"path/filepath"
	"testing"
)

func TestChangePasswordRetainsCurrentSessionAndRevokesOthers(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, db.Config{Path: filepath.Join(t.TempDir(), "password.db"), MaxOpenConns: 2, MaxIdleConns: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := OwnerBootstrap(ctx, database, "owner@example.com", "old-password"); err != nil {
		t.Fatal(err)
	}
	first, err := Login(ctx, database, "owner@example.com", "old-password")
	if err != nil || !first.Success {
		t.Fatal("first login failed")
	}
	second, err := Login(ctx, database, "owner@example.com", "old-password")
	if err != nil || !second.Success {
		t.Fatal("second login failed")
	}
	var id, workspace string
	if err := database.SQL().QueryRow("SELECT id,workspace_id FROM users").Scan(&id, &workspace); err != nil {
		t.Fatal(err)
	}
	actor := permissions.AuthContext{ActorType: "user", ActorID: id, WorkspaceID: workspace, Scopes: OwnerScopes()}
	if err := ChangePassword(ctx, database, actor, "wrong-password", "new-password", first.Token); !errors.Is(err, ErrCurrentPassword) {
		t.Fatalf("incorrect password accepted: %v", err)
	}
	if err := ChangePassword(ctx, database, actor, "old-password", "new-password", first.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateSession(ctx, database, first.Token); err != nil {
		t.Fatal("current session was revoked")
	}
	if _, err := ValidateSession(ctx, database, second.Token); err == nil {
		t.Fatal("other session survived password change")
	}
	if result, err := Login(ctx, database, "owner@example.com", "old-password"); err != nil || result.Success {
		t.Fatal("old password still works")
	}
	if result, err := Login(ctx, database, "owner@example.com", "new-password"); err != nil || !result.Success {
		t.Fatal("new password does not work")
	}
}

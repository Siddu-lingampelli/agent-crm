package auth

import (
	"context"
	"crm.local/internal/db"
	"crm.local/internal/permissions"
	"database/sql"
	"errors"
)

var ErrCurrentPassword = errors.New("current password is incorrect")
var ErrPasswordLength = errors.New("password must contain between 8 and 4096 bytes")
var ErrPasswordChanged = errors.New("password changed concurrently; try again")

// ChangePassword verifies the existing password and revokes every other session
// in the same transaction. A concurrent password change cannot overwrite it.
func ChangePassword(ctx context.Context, database *db.DB, actor permissions.AuthContext, current, next, keepToken string) error {
	if actor.ActorType != "user" {
		return permissions.ErrScopeDenied
	}
	if err := permissions.Authorize(actor, "settings:write"); err != nil {
		return err
	}
	if len(next) < 8 || len(next) > 4096 || len(current) > 4096 {
		return ErrPasswordLength
	}
	var oldHash string
	if err := database.SQL().QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id=? AND workspace_id=?", actor.ActorID, actor.WorkspaceID).Scan(&oldHash); err != nil {
		return ErrCurrentPassword
	}
	valid, err := VerifyArgon2(oldHash, current)
	if err != nil {
		return err
	}
	if !valid {
		return ErrCurrentPassword
	}
	newHash, err := HashArgon2(next)
	if err != nil {
		return err
	}
	return database.Transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, "UPDATE users SET password_hash=?, updated_at=datetime('now') WHERE id=? AND workspace_id=? AND password_hash=?", newHash, actor.ActorID, actor.WorkspaceID, oldHash)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrPasswordChanged
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=? AND id<>?", actor.ActorID, sessionHash(keepToken))
		return err
	})
}

// Package auth implements owner bootstrap, password hashing, session cookies
// and CSRF protection. See PRD §10.1.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"crm.local/internal/db"
	"crm.local/internal/permissions"
	"crm.local/internal/uuid"
)

// SessionCookie settings.
type SessionCookie struct {
	Name     string
	HTTPOnly bool
	SameSite string
	Secure   bool
}

// Context key types to avoid collisions with built-in types
type contextKey string

const (
	contextKeyUserID      contextKey = "user_id"
	contextKeyWorkspaceID contextKey = "workspace_id"
)

// DefaultCookie returns the production cookie settings.
func DefaultCookie() SessionCookie {
	return SessionCookie{
		Name:     "crm_session",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("CRM_COOKIE_SECURE") == "1",
	}
}

// CookieName is the session cookie name.
const CookieName = "crm_session"

type SessionOptions struct {
	CookieName string
	Secure     bool
	SameSite   http.SameSite
	MaxAge     time.Duration
}

func DefaultSessionOptions() SessionOptions {
	return SessionOptions{CookieName: CookieName, Secure: DefaultCookie().Secure, SameSite: http.SameSiteLaxMode, MaxAge: 24 * time.Hour}
}

// OwnerScopes is the complete scope vocabulary for the single owner release.
func OwnerScopes() []string {
	return strings.Fields("search:read companies:read companies:write contacts:read contacts:write leads:read leads:write deals:read deals:write tasks:read tasks:write projects:read projects:write emails:read emails:write emails:send calendar:read calendar:write workflows:read workflows:write workflows:execute integrations:read integrations:write webhooks:read webhooks:write audit:read audit:write agents:read agents:write dashboard:read settings:read settings:write admin:read admin:write")
}

const actorContextKey contextKey = "authenticated_actor"

func WithAuth(ctx context.Context, actor permissions.AuthContext) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

func FromContext(ctx context.Context) permissions.AuthContext {
	actor, _ := ctx.Value(actorContextKey).(permissions.AuthContext)
	return actor
}

func sessionHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Password hashing uses a modern algorithm. argon2id is preferred when
// available; scrypt is the fallback.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (bool, error)
}

// OwnerBootstrap creates the first owner user and workspace. It is a no-op
// if an owner already exists, so it is safe to call on every startup.
func OwnerBootstrap(ctx context.Context, database *db.DB, email, password string) error {
	if database == nil {
		return errors.New("database is nil")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	return database.Transaction(ctx, func(tx *sql.Tx) error {
		var existing int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&existing); err != nil {
			return err
		}
		if existing > 0 {
			return nil
		}

		hash, err := HashArgon2(password)
		if err != nil {
			return err
		}

		wsID := uuid.New()
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`INSERT INTO workspaces (id, name, created_at) VALUES (?, ?, ?)`,
			wsID, "Default Workspace", now)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, "UPDATE pipelines SET workspace_id = ? WHERE workspace_id = ?", wsID, "ws_default"); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO users (id, workspace_id, email, password_hash, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.New(), wsID, email, hash, now, now)
		if err != nil {
			return err
		}

		return nil
	})
}

// LoginResult describes the outcome of a login attempt.
type LoginResult struct {
	Success   bool
	Token     string
	ExpiresAt time.Time
	Reason    string
}

// Login verifies credentials and issues a session token.
func Login(ctx context.Context, database *db.DB, email, password string) (LoginResult, error) {
	return LoginWithOptions(ctx, database, email, password, DefaultSessionOptions())
}

func LoginWithOptions(ctx context.Context, database *db.DB, email, password string, options SessionOptions) (LoginResult, error) {
	var result LoginResult
	if len(password) > 4096 || options.MaxAge <= 0 {
		result.Reason = "invalid credentials"
		return result, nil
	}

	var hash string
	var userID string
	err := database.SQL().QueryRowContext(ctx,
		`SELECT id, password_hash FROM users WHERE email = ?`, email).Scan(&userID, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			result.Reason = "invalid credentials"
			return result, nil
		}
		return result, fmt.Errorf("lookup user: %w", err)
	}

	ok, err := VerifyArgon2(hash, password)
	if err != nil {
		return result, fmt.Errorf("verify: %w", err)
	}
	if !ok {
		result.Reason = "invalid credentials"
		return result, nil
	}

	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return result, err
	}
	token := hex.EncodeToString(random[:])

	expiresAt := time.Now().UTC().Add(options.MaxAge)
	if _, err := database.SQL().ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return result, fmt.Errorf("clean expired sessions: %w", err)
	}
	_, err = database.SQL().ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		sessionHash(token), userID, expiresAt.Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return result, fmt.Errorf("create session: %w", err)
	}

	result.Success = true
	result.Token = token
	result.ExpiresAt = expiresAt
	return result, nil
}

// ValidateSession returns the user ID for a token, or an error.
func ValidateSession(ctx context.Context, database *db.DB, token string) (string, error) {
	var userID string
	var expiresAt string
	err := database.SQL().QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM sessions WHERE id = ?`, sessionHash(token)).Scan(&userID, &expiresAt)
	if err != nil {
		return "", errors.New("invalid session")
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return "", fmt.Errorf("parse expiry: %w", err)
	}
	if time.Now().UTC().After(expiry) {
		return "", errors.New("session expired")
	}

	return userID, nil
}

// Logout invalidates a session token.
func Logout(ctx context.Context, database *db.DB, token string) error {
	_, err := database.SQL().ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", sessionHash(token))
	return err
}

// SessionMiddleware creates middleware that validates session cookies.
func SessionMiddleware(db *db.DB, log *slog.Logger, configured ...SessionOptions) func(http.Handler) http.Handler {
	options := DefaultSessionOptions()
	if len(configured) > 0 {
		options = configured[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(options.CookieName)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := ValidateSession(r.Context(), db, cookie.Value)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Fetch workspace_id from users table
			var workspaceID string
			err = db.SQL().QueryRowContext(r.Context(),
				`SELECT workspace_id FROM users WHERE id = ?`, userID).Scan(&workspaceID)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
			ctx = context.WithValue(ctx, contextKeyWorkspaceID, workspaceID)
			ctx = WithAuth(ctx, permissions.AuthContext{ActorType: "user", ActorID: userID, WorkspaceID: workspaceID, Scopes: OwnerScopes()})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuth extracts auth context from request.
func GetAuth(r *http.Request) permissions.AuthContext {
	if actor := FromContext(r.Context()); actor.ActorID != "" {
		return actor
	}
	if ctx := r.Context(); ctx != nil {
		if userID, ok := ctx.Value(contextKeyUserID).(string); ok {
			var workspaceID string
			if wsID, ok := ctx.Value(contextKeyWorkspaceID).(string); ok {
				workspaceID = wsID
			} else {
				return permissions.AuthContext{}
			}
			return permissions.AuthContext{
				ActorType:   "user",
				ActorID:     userID,
				WorkspaceID: workspaceID,
				Scopes:      OwnerScopes(),
			}
		}
	}
	return permissions.AuthContext{}
}

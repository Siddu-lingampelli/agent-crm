// Package repository — OAuth 2.1 and MCP observability persistence.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"crm.local/internal/model"
)

// OAuthRepository persists OAuth clients, authorization codes, tokens,
// and per-tool MCP call records.
type OAuthRepository struct {
	db *sql.DB
}

// NewOAuthRepository returns a repository over db.
func NewOAuthRepository(db *sql.DB) *OAuthRepository { return &OAuthRepository{db: db} }

func jsonList(v []string) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func parseList(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

// --- Clients ---

func (r *OAuthRepository) CreateClient(ctx context.Context, c *model.OAuthClient) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_clients (id, workspace_id, client_id, client_secret_hash, name, redirect_uris, scopes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.ClientID, c.ClientSecretHash, c.Name,
		jsonList(c.RedirectURIs), jsonList(c.Scopes),
		c.CreatedAt.UTC().Format(time.RFC3339Nano), c.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert oauth client: %w", err)
	}
	return nil
}

func (r *OAuthRepository) GetClientByClientID(ctx context.Context, clientID string) (*model.OAuthClient, error) {
	var c model.OAuthClient
	var secret sql.NullString
	var redirects, scopes, createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, client_id, client_secret_hash, name, redirect_uris, scopes, created_at, updated_at
		FROM oauth_clients WHERE client_id = ?`, clientID).
		Scan(&c.ID, &c.WorkspaceID, &c.ClientID, &secret, &c.Name, &redirects, &scopes, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth client: %w", err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if secret.Valid {
		c.ClientSecretHash = &secret.String
	}
	c.RedirectURIs = parseList(redirects)
	c.Scopes = parseList(scopes)
	return &c, nil
}

// --- Authorization codes ---

func (r *OAuthRepository) CreateAuthorizationCode(ctx context.Context, a *model.OAuthAuthorizationCode) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_authorization_codes (code, client_id, workspace_id, user_id, redirect_uri, code_challenge, code_challenge_method, scopes, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Code, a.ClientID, a.WorkspaceID, a.UserID, a.RedirectURI,
		a.CodeChallenge, a.CodeChallengeMethod, jsonList(a.Scopes),
		a.ExpiresAt.UTC().Format(time.RFC3339Nano), a.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert auth code: %w", err)
	}
	return nil
}

// ConsumeAuthorizationCode atomically marks a code used and returns it.
// It fails if the code is missing, expired, or already used. Timestamps are
// compared as julianday values because the driver stores them as TEXT.
func (r *OAuthRepository) ConsumeAuthorizationCode(ctx context.Context, code, clientID string) (*model.OAuthAuthorizationCode, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE oauth_authorization_codes SET used_at = datetime('now')
		WHERE code = ? AND client_id = ? AND used_at IS NULL AND julianday(expires_at) > julianday('now')`,
		code, clientID)
	if err != nil {
		return nil, fmt.Errorf("consume auth code: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("invalid or expired authorization code")
	}
	var a model.OAuthAuthorizationCode
	var scopes, expiresAt, createdAt string
	err = r.db.QueryRowContext(ctx, `
		SELECT code, client_id, workspace_id, user_id, redirect_uri, code_challenge, code_challenge_method, scopes, expires_at, created_at
		FROM oauth_authorization_codes WHERE code = ?`, code).
		Scan(&a.Code, &a.ClientID, &a.WorkspaceID, &a.UserID, &a.RedirectURI,
			&a.CodeChallenge, &a.CodeChallengeMethod, &scopes, &expiresAt, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("load auth code: %w", err)
	}
	a.Scopes = parseList(scopes)
	a.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &a, nil
}

// --- Listing (admin/MCP-connections UI) ---

// ListClients returns OAuth clients for a workspace, newest first.
func (r *OAuthRepository) ListClients(ctx context.Context, workspaceID string, limit int) ([]model.OAuthClient, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, client_id, client_secret_hash, name, redirect_uris, scopes, created_at, updated_at
		FROM oauth_clients WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ?`,
		workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list oauth clients: %w", err)
	}
	defer rows.Close()
	var out []model.OAuthClient
	for rows.Next() {
		c, err := scanOAuthClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanOAuthClient(rows *sql.Rows) (model.OAuthClient, error) {
	var c model.OAuthClient
	var secret sql.NullString
	var redirects, scopes, createdAt, updatedAt string
	if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.ClientID, &secret, &c.Name, &redirects, &scopes, &createdAt, &updatedAt); err != nil {
		return model.OAuthClient{}, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if secret.Valid {
		c.ClientSecretHash = &secret.String
	}
	c.RedirectURIs = parseList(redirects)
	c.Scopes = parseList(scopes)
	return c, nil
}

// ListAccessTokens returns issued access tokens for a workspace (newest first).
// The raw token value is never stored, only its SHA-256 hash.
func (r *OAuthRepository) ListAccessTokens(ctx context.Context, workspaceID string, limit int) ([]model.OAuthAccessToken, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT token_hash, client_id, workspace_id, user_id, scopes, expires_at, revoked_at, created_at
		FROM oauth_access_tokens WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ?`,
		workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list access tokens: %w", err)
	}
	defer rows.Close()
	var out []model.OAuthAccessToken
	for rows.Next() {
		var t model.OAuthAccessToken
		var scopes, expiresAt, createdAt string
		var revoked sql.NullString
		if err := rows.Scan(&t.TokenHash, &t.ClientID, &t.WorkspaceID, &t.UserID, &scopes, &expiresAt, &revoked, &createdAt); err != nil {
			return nil, err
		}
		t.Scopes = parseList(scopes)
		t.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if revoked.Valid {
			parsed, perr := time.Parse(time.RFC3339Nano, revoked.String)
			if perr == nil {
				t.RevokedAt = &parsed
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// --- Access tokens ---

func (r *OAuthRepository) CreateAccessToken(ctx context.Context, t *model.OAuthAccessToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_access_tokens (token_hash, client_id, workspace_id, user_id, scopes, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.TokenHash, t.ClientID, t.WorkspaceID, t.UserID, jsonList(t.Scopes),
		t.ExpiresAt.UTC().Format(time.RFC3339Nano), t.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert access token: %w", err)
	}
	return nil
}

// GetAccessToken looks up a live token by its hash.
func (r *OAuthRepository) GetAccessToken(ctx context.Context, tokenHash string) (*model.OAuthAccessToken, error) {
	var t model.OAuthAccessToken
	var scopes, expiresAt, createdAt string
	var revoked sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT token_hash, client_id, workspace_id, user_id, scopes, expires_at, revoked_at, created_at
		FROM oauth_access_tokens WHERE token_hash = ?`, tokenHash).
		Scan(&t.TokenHash, &t.ClientID, &t.WorkspaceID, &t.UserID, &scopes, &expiresAt, &revoked, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	t.Scopes = parseList(scopes)
	t.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if revoked.Valid {
		parsed, perr := time.Parse(time.RFC3339Nano, revoked.String)
		if perr == nil {
			t.RevokedAt = &parsed
		}
	}
	if t.RevokedAt != nil || time.Now().UTC().After(t.ExpiresAt) {
		return nil, fmt.Errorf("token revoked or expired")
	}
	// token must be a live (unrevoked) row: re-check via julianday comparison
	var live int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oauth_access_tokens WHERE token_hash = ? AND revoked_at IS NULL AND julianday(expires_at) > julianday('now')`, tokenHash).Scan(&live); err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	if live == 0 {
		return nil, fmt.Errorf("token revoked or expired")
	}
	return &t, nil
}

func (r *OAuthRepository) RevokeAccessToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oauth_access_tokens SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339Nano), tokenHash)
	return err
}

// --- MCP tool call records ---

func (r *OAuthRepository) RecordMCPToolCall(ctx context.Context, c *model.MCPToolCall) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO mcp_tool_calls (id, workspace_id, actor_type, actor_id, tool, status, duration_us, error, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.ActorType, c.ActorID, c.Tool, c.Status, c.DurationUS, c.Error, c.IPAddress,
		c.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert mcp tool call: %w", err)
	}
	return nil
}

// MCPToolCallStats aggregates per-tool counts over an optional time window.
type MCPToolStat struct {
	Tool      string `json:"tool"`
	Calls     int    `json:"calls"`
	Errors    int    `json:"errors"`
	TotalUS   int64  `json:"total_duration_us"`
	AvgUS     int64  `json:"avg_duration_us"`
}

func (r *OAuthRepository) MCPToolCallStats(ctx context.Context, workspaceID string, since time.Time) ([]MCPToolStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tool, COUNT(*), SUM(CASE WHEN status='error' THEN 1 ELSE 0 END),
		       SUM(duration_us), AVG(duration_us)
		FROM mcp_tool_calls
		WHERE workspace_id = ? AND created_at >= ?
		GROUP BY tool ORDER BY COUNT(*) DESC`,
		workspaceID, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("mcp tool stats: %w", err)
	}
	defer rows.Close()
	var out []MCPToolStat
	for rows.Next() {
		var s MCPToolStat
		var avg float64
		if err := rows.Scan(&s.Tool, &s.Calls, &s.Errors, &s.TotalUS, &avg); err != nil {
			return nil, err
		}
		s.AvgUS = int64(avg)
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListMCPToolCalls returns recent MCP tool calls (newest first).
func (r *OAuthRepository) ListMCPToolCalls(ctx context.Context, workspaceID string, limit int) ([]model.MCPToolCall, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, actor_type, actor_id, tool, status, duration_us, error, ip_address, created_at
		FROM mcp_tool_calls WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ?`,
		workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list mcp tool calls: %w", err)
	}
	defer rows.Close()
	var out []model.MCPToolCall
	for rows.Next() {
		var c model.MCPToolCall
		var errStr sql.NullString
		var createdAt string
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.ActorType, &c.ActorID, &c.Tool, &c.Status, &c.DurationUS, &errStr, &c.IPAddress, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if errStr.Valid {
			c.Error = &errStr.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

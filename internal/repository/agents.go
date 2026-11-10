// Package repository — agents and API keys.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/uuid"
)

const agentColumns = `
    id, workspace_id, name, description, provider, enabled, scopes, rate_limit,
    last_used_at, created_at, updated_at
`

const apiKeyColumns = `
    id, workspace_id, agent_id, name, prefix, hash, scopes, rate_limit,
    expires_at, revoked_at, last_used_at, created_at
`

// AgentListOptions extends ListOptions with agent filters.
type AgentListOptions struct {
	WorkspaceID string
	Query       string
	Enabled     *bool
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

// ApiKeyListOptions extends ListOptions with API key filters.
type ApiKeyListOptions struct {
	WorkspaceID string
	AgentID     string
	Query       string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

// AgentRepository persists agents.
type AgentRepository struct {
	db *sql.DB
}

// NewAgentRepository returns a repository over db.
func NewAgentRepository(db *sql.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create inserts an agent.
func (r *AgentRepository) Create(ctx context.Context, a *model.Agent) error {
	scopes := jsonOrDefault(a.Scopes)
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO agents (
            id, workspace_id, name, description, provider, enabled, scopes, rate_limit,
            last_used_at, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.WorkspaceID, a.Name, a.Description, a.Provider, boolToInt(a.Enabled),
		scopes, a.RateLimit, a.LastUsedAt, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}
	return nil
}

// Get returns an agent by id.
func (r *AgentRepository) Get(ctx context.Context, id string) (*model.Agent, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+agentColumns+" FROM agents WHERE id = ?", id)
	a, err := scanAgent(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

// List returns a page of agents.
func (r *AgentRepository) List(ctx context.Context, opts AgentListOptions) (Page[model.Agent], error) {
	where, args := agentFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM agents" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Agent]{}, fmt.Errorf("count agents: %w", err)
	}

	sort := agentSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := max(opts.Offset, 0)

	// sort and order are validated by agentSort and NormalizeOrder, safe to use in query
	//nolint:gosec // G202: sort and order are validated by agentSort and NormalizeOrder
	sqlText := "SELECT " + agentColumns + " FROM agents" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Agent]{}, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	items := []model.Agent{}
	for rows.Next() {
		item, err := scanAgentRows(rows)
		if err != nil {
			return Page[model.Agent]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Agent]{}, fmt.Errorf("iterate agents: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies an agent.
func (r *AgentRepository) Update(ctx context.Context, a *model.Agent) error {
	scopes := jsonOrDefault(a.Scopes)
	_, err := r.db.ExecContext(ctx, `
        UPDATE agents SET
            name = ?, description = ?, provider = ?, enabled = ?, scopes = ?,
            rate_limit = ?, last_used_at = ?, updated_at = ?
        WHERE id = ?`,
		a.Name, a.Description, a.Provider, boolToInt(a.Enabled), scopes,
		a.RateLimit, a.LastUsedAt, a.UpdatedAt, a.ID)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	return nil
}

// Archive soft-deletes an agent.
func (r *AgentRepository) Archive(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE agents SET enabled = 0, updated_at = ? WHERE id = ?", time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("archive agent: %w", err)
	}
	return nil
}

// Restore re-enables an agent.
func (r *AgentRepository) Restore(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE agents SET enabled = 1, updated_at = ? WHERE id = ?", time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("restore agent: %w", err)
	}
	return nil
}

// ApiKeyRepository persists API keys.
type ApiKeyRepository struct {
	db *sql.DB
}

// NewApiKeyRepository returns a repository over db.
func NewApiKeyRepository(db *sql.DB) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

// Create inserts an API key.
func (r *ApiKeyRepository) Create(ctx context.Context, k *model.APIKey) error {
	scopes := jsonOrDefault(k.Scopes)
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO api_keys (
            id, workspace_id, agent_id, name, prefix, hash, scopes, rate_limit,
            expires_at, revoked_at, last_used_at, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.WorkspaceID, k.AgentID, k.Name, k.Prefix, k.Hash, scopes,
		k.RateLimit, k.ExpiresAt, k.RevokedAt, k.LastUsedAt, k.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}
	return nil
}

// Get returns an API key by id.
func (r *ApiKeyRepository) Get(ctx context.Context, id string) (*model.APIKey, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+apiKeyColumns+" FROM api_keys WHERE id = ?", id)
	k, err := scanApiKey(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return k, err
}

// GetByHash returns an API key by hash.
func (r *ApiKeyRepository) GetByHash(ctx context.Context, hash string) (*model.APIKey, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+apiKeyColumns+" FROM api_keys WHERE hash = ? AND revoked_at IS NULL", hash)
	k, err := scanApiKey(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return k, err
}

// List returns a page of API keys.
func (r *ApiKeyRepository) List(ctx context.Context, opts ApiKeyListOptions) (Page[model.APIKey], error) {
	where, args := apiKeyFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM api_keys" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.APIKey]{}, fmt.Errorf("count api keys: %w", err)
	}

	sort := apiKeySort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := max(opts.Offset, 0)

	// sort and order are validated by apiKeySort and NormalizeOrder, safe to use in query
	//nolint:gosec // G202: sort and order are validated by apiKeySort and NormalizeOrder
	sqlText := "SELECT " + apiKeyColumns + " FROM api_keys" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.APIKey]{}, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	items := []model.APIKey{}
	for rows.Next() {
		item, err := scanApiKeyRows(rows)
		if err != nil {
			return Page[model.APIKey]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.APIKey]{}, fmt.Errorf("iterate api keys: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies an API key.
func (r *ApiKeyRepository) Update(ctx context.Context, k *model.APIKey) error {
	scopes := jsonOrDefault(k.Scopes)
	_, err := r.db.ExecContext(ctx, `
        UPDATE api_keys SET
            name = ?, scopes = ?, rate_limit = ?, expires_at = ?, revoked_at = ?
        WHERE id = ?`,
		k.Name, scopes, k.RateLimit, k.ExpiresAt, k.RevokedAt, k.ID)
	if err != nil {
		return fmt.Errorf("update api key: %w", err)
	}
	return nil
}

// Revoke marks an API key as revoked.
func (r *ApiKeyRepository) Revoke(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE api_keys SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL", now, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp.
func (r *ApiKeyRepository) UpdateLastUsed(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE api_keys SET last_used_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}

func agentFilters(opts AgentListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}

	if opts.Query != "" {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term)
	}
	if opts.Enabled != nil {
		conditions = append(conditions, "enabled = ?")
		if *opts.Enabled {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func agentSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "name":
		return "name"
	case "last_used_at":
		return "last_used_at"
	default:
		return "created_at"
	}
}

func scanAgent(row *sql.Row) (*model.Agent, error) {
	a := &model.Agent{}
	var scopesJSON string
	var createdAt, updatedAt string
	var lastUsedAt *string
	var enabled int
	var rateLimit *int
	if err := row.Scan(
		&a.ID, &a.WorkspaceID, &a.Name, &a.Description, &a.Provider,
		&enabled, &scopesJSON, &rateLimit, &lastUsedAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	a.Enabled = enabled == 1
	a.Scopes = []string{}
	_ = json.Unmarshal([]byte(scopesJSON), &a.Scopes)
	a.RateLimit = rateLimit
	if lastUsedAt != nil {
		if t := parseTime(*lastUsedAt); !t.IsZero() {
			a.LastUsedAt = &t
		}
	}
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	return a, nil
}

func scanAgentRows(rows *sql.Rows) (*model.Agent, error) {
	a := &model.Agent{}
	var scopesJSON string
	var createdAt, updatedAt string
	var lastUsedAt *string
	var enabled int
	var rateLimit *int
	if err := rows.Scan(
		&a.ID, &a.WorkspaceID, &a.Name, &a.Description, &a.Provider,
		&enabled, &scopesJSON, &rateLimit, &lastUsedAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	a.Enabled = enabled == 1
	a.Scopes = []string{}
	_ = json.Unmarshal([]byte(scopesJSON), &a.Scopes)
	a.RateLimit = rateLimit
	if lastUsedAt != nil {
		if t := parseTime(*lastUsedAt); !t.IsZero() {
			a.LastUsedAt = &t
		}
	}
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	return a, nil
}

func apiKeyFilters(opts ApiKeyListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	if opts.AgentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, opts.AgentID)
	}
	if opts.Query != "" {
		conditions = append(conditions, "(name LIKE ?)")
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func apiKeySort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "name":
		return "name"
	case "last_used_at":
		return "last_used_at"
	case "expires_at":
		return "expires_at"
	default:
		return "created_at"
	}
}

func scanApiKey(row *sql.Row) (*model.APIKey, error) {
	k := &model.APIKey{}
	var scopesJSON string
	var createdAt string
	var expiresAt, revokedAt, lastUsedAt *string
	var rateLimit *int
	if err := row.Scan(
		&k.ID, &k.WorkspaceID, &k.AgentID, &k.Name, &k.Prefix, &k.Hash,
		&scopesJSON, &rateLimit, &expiresAt, &revokedAt, &lastUsedAt, &createdAt); err != nil {
		return nil, err
	}
	k.Scopes = []string{}
	_ = json.Unmarshal([]byte(scopesJSON), &k.Scopes)
	k.RateLimit = rateLimit
	if expiresAt != nil {
		if t := parseTime(*expiresAt); !t.IsZero() {
			k.ExpiresAt = &t
		}
	}
	if revokedAt != nil {
		if t := parseTime(*revokedAt); !t.IsZero() {
			k.RevokedAt = &t
		}
	}
	if lastUsedAt != nil {
		if t := parseTime(*lastUsedAt); !t.IsZero() {
			k.LastUsedAt = &t
		}
	}
	k.CreatedAt = parseTime(createdAt)
	return k, nil
}

func scanApiKeyRows(rows *sql.Rows) (*model.APIKey, error) {
	k := &model.APIKey{}
	var scopesJSON string
	var createdAt string
	var expiresAt, revokedAt, lastUsedAt *string
	var rateLimit *int
	if err := rows.Scan(
		&k.ID, &k.WorkspaceID, &k.AgentID, &k.Name, &k.Prefix, &k.Hash,
		&scopesJSON, &rateLimit, &expiresAt, &revokedAt, &lastUsedAt, &createdAt); err != nil {
		return nil, err
	}
	k.Scopes = []string{}
	_ = json.Unmarshal([]byte(scopesJSON), &k.Scopes)
	k.RateLimit = rateLimit
	if expiresAt != nil {
		if t := parseTime(*expiresAt); !t.IsZero() {
			k.ExpiresAt = &t
		}
	}
	if revokedAt != nil {
		if t := parseTime(*revokedAt); !t.IsZero() {
			k.RevokedAt = &t
		}
	}
	if lastUsedAt != nil {
		if t := parseTime(*lastUsedAt); !t.IsZero() {
			k.LastUsedAt = &t
		}
	}
	k.CreatedAt = parseTime(createdAt)
	return k, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GenerateAPIKey creates a new API key with prefix and hash.
// Returns the raw key (shown once), prefix, and the hash (stored).
func GenerateAPIKey() (rawKey, prefix, hash string) {
	rawKey = uuid.NewAPIKey()
	prefix = rawKey[:8]
	hash = uuid.HashAPIKey(rawKey)
	return rawKey, prefix, hash
}

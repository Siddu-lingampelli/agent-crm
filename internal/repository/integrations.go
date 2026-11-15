// Package repository — integrations.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
)

// IntegrationListOptions extends ListOptions with integration-specific filters.
type IntegrationListOptions struct {
	Query       string
	WorkspaceID string
	Provider    string
	Status      string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const integrationsColumns = `
    id, workspace_id, name, provider, config, credentials, status,
    last_sync_at, last_error, enabled, created_at, updated_at
`

// IntegrationRepository persists integrations.
type IntegrationRepository struct {
	db *sql.DB
}

// NewIntegrationRepository returns a repository over db.
func NewIntegrationRepository(db *sql.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

// Create inserts an integration.
func (r *IntegrationRepository) Create(ctx context.Context, i *model.Integration) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO integrations (
            id, workspace_id, name, provider, config, credentials, status,
            last_sync_at, last_error, enabled, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		i.ID, i.WorkspaceID, i.Name, i.Provider, i.Config, i.Credentials, i.Status,
		i.LastSyncAt, i.LastError, i.Enabled, i.CreatedAt, i.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert integration: %w", err)
	}
	return nil
}

// Get returns an integration by id, or nil if not found.
func (r *IntegrationRepository) Get(ctx context.Context, id string) (*model.Integration, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+integrationsColumns+` FROM integrations WHERE id = ? AND archived_at IS NULL`, id)
	i, err := scanIntegration(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

// List returns integrations with filters.
func (r *IntegrationRepository) List(ctx context.Context, opts IntegrationListOptions) ([]model.Integration, error) {
	where, args := integrationFilters(opts)

	sort := integrationSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := `SELECT ` + integrationsColumns + ` FROM integrations` + where +
		` ORDER BY ` + sort + ` ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()

	var integrations []model.Integration
	for rows.Next() {
		item, err := scanIntegrationRows(rows)
		if err != nil {
			return nil, err
		}
		integrations = append(integrations, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate integrations: %w", err)
	}
	return integrations, nil
}

// Update modifies an integration.
func (r *IntegrationRepository) Update(ctx context.Context, i *model.Integration) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE integrations SET
            name = ?, provider = ?, config = ?, credentials = ?, status = ?,
            last_sync_at = ?, last_error = ?, enabled = ?, updated_at = ?
        WHERE id = ?`,
		i.Name, i.Provider, i.Config, i.Credentials, i.Status,
		i.LastSyncAt, i.LastError, i.Enabled, i.UpdatedAt, i.ID)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}
	return nil
}

// Archive soft-deletes an integration.
func (r *IntegrationRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE integrations SET archived_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("archive integration: %w", err)
	}
	return nil
}

// UpdateCredentials updates encrypted credentials for an integration.
func (r *IntegrationRepository) UpdateCredentials(ctx context.Context, id string, credentials string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE integrations SET credentials = ?, updated_at = datetime('now') WHERE id = ?`, credentials, id)
	if err != nil {
		return fmt.Errorf("update credentials: %w", err)
	}
	return nil
}

// UpdateLastSync updates the last sync time and error.
func (r *IntegrationRepository) UpdateLastSync(ctx context.Context, id string, lastSyncAt string, lastError string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE integrations SET last_sync_at = ?, last_error = ?, updated_at = datetime('now') WHERE id = ?`, lastSyncAt, lastError, id)
	if err != nil {
		return fmt.Errorf("update last sync: %w", err)
	}
	return nil
}

func integrationFilters(opts IntegrationListOptions) (string, []any) {
	conditions := []string{"archived_at IS NULL"}
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}

	if query := strings.TrimSpace(opts.Query); query != "" {
		conditions = append(conditions, "(name LIKE ? OR provider LIKE ?)")
		args = append(args, "%"+query+"%", "%"+query+"%")
	}

	if opts.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, opts.Provider)
	}

	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func integrationSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "name":
		return "name"
	case "provider":
		return "provider"
	case "status":
		return "status"
	default:
		return "created_at"
	}
}

func scanIntegration(row *sql.Row) (*model.Integration, error) {
	i := &model.Integration{}
	var createdAt, updatedAt string
	if err := row.Scan(
		&i.ID, &i.WorkspaceID, &i.Name, &i.Provider, &i.Config, &i.Credentials, &i.Status,
		&i.LastSyncAt, &i.LastError, &i.Enabled, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	i.CreatedAt = parseTime(createdAt)
	i.UpdatedAt = parseTime(updatedAt)
	return i, nil
}

func scanIntegrationRows(rows *sql.Rows) (*model.Integration, error) {
	i := &model.Integration{}
	var createdAt, updatedAt string
	if err := rows.Scan(
		&i.ID, &i.WorkspaceID, &i.Name, &i.Provider, &i.Config, &i.Credentials, &i.Status,
		&i.LastSyncAt, &i.LastError, &i.Enabled, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	i.CreatedAt = parseTime(createdAt)
	i.UpdatedAt = parseTime(updatedAt)
	return i, nil
}
func (r *IntegrationRepository) Count(ctx context.Context, opts IntegrationListOptions) (int, error) {
	where, args := integrationFilters(opts)
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM integrations"+where, args...).Scan(&total)
	return total, err
}

const syncLogColumns = `
    id, integration_id, status, records_processed, records_failed,
    started_at, completed_at, error_message
`

// RecordSyncLog appends a sync log entry for an integration.
func (r *IntegrationRepository) RecordSyncLog(ctx context.Context, log *model.IntegrationSyncLog) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO integration_sync_logs (
            id, integration_id, status, records_processed, records_failed,
            started_at, completed_at, error_message
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.IntegrationID, log.Status, log.RecordsProcessed, log.RecordsFailed,
		log.StartedAt.Format(time.RFC3339Nano),
		nullTime(log.CompletedAt),
		nullString(log.ErrorMessage))
	if err != nil {
		return fmt.Errorf("insert sync log: %w", err)
	}
	return nil
}

// ListSyncLogs returns the most recent sync log entries for an integration.
func (r *IntegrationRepository) ListSyncLogs(ctx context.Context, integrationID string, limit int) ([]model.IntegrationSyncLog, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT `+syncLogColumns+` FROM integration_sync_logs
        WHERE integration_id = ?
        ORDER BY started_at DESC, id DESC LIMIT ?`, integrationID, ClampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list sync logs: %w", err)
	}
	defer rows.Close()

	out := []model.IntegrationSyncLog{}
	for rows.Next() {
		var log model.IntegrationSyncLog
		var startedAt string
		var completedAt, errorMessage *string
		if err := rows.Scan(&log.ID, &log.IntegrationID, &log.Status, &log.RecordsProcessed,
			&log.RecordsFailed, &startedAt, &completedAt, &errorMessage); err != nil {
			return nil, err
		}
		log.StartedAt = parseTime(startedAt)
		if completedAt != nil {
			t := parseTime(*completedAt)
			log.CompletedAt = &t
		}
		if errorMessage != nil {
			log.ErrorMessage = errorMessage
		}
		out = append(out, log)
	}
	return out, rows.Err()
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

func nullString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

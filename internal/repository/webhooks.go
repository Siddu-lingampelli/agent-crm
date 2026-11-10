// Package repository — webhooks.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
)

// WebhookListOptions extends ListOptions with webhook-specific filters.
type WebhookListOptions struct {
	WorkspaceID string
	Status      string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const webhooksColumns = `
    id, workspace_id, name, url, events, secret, status,
    failure_count, last_triggered_at, last_success_at, last_failure_at, last_failure_reason,
    created_at, updated_at
`

// WebhookRepository persists webhooks.
type WebhookRepository struct {
	db *sql.DB
}

// NewWebhookRepository returns a repository over db.
func NewWebhookRepository(db *sql.DB) *WebhookRepository { return &WebhookRepository{db: db} }

// Create inserts a webhook.
func (r *WebhookRepository) Create(ctx context.Context, w *model.Webhook) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO webhooks (
            id, workspace_id, name, url, events, secret, status,
            failure_count, last_triggered_at, last_success_at, last_failure_at, last_failure_reason,
            created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.WorkspaceID, w.Name, w.URL, w.Events, w.Secret, w.Status,
		w.FailureCount, w.LastTriggeredAt, w.LastSuccessAt, w.LastFailureAt, w.LastFailureReason,
		w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert webhook: %w", err)
	}
	return nil
}

// Get returns a webhook by id, or nil if not found.
func (r *WebhookRepository) Get(ctx context.Context, id string) (*model.Webhook, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+webhooksColumns+` FROM webhooks WHERE id = ? AND archived_at IS NULL`, id)
	w, err := scanWebhook(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// List returns webhooks with filters.
func (r *WebhookRepository) List(ctx context.Context, opts WebhookListOptions) ([]model.Webhook, error) {
	where, args := webhookFilters(opts)

	sort := webhookSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := `SELECT ` + webhooksColumns + ` FROM webhooks` + where +
		` ORDER BY ` + sort + ` ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []model.Webhook
	for rows.Next() {
		item, err := scanWebhookRows(rows)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhooks: %w", err)
	}
	return webhooks, nil
}

// Update modifies a webhook.
func (r *WebhookRepository) Update(ctx context.Context, w *model.Webhook) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE webhooks SET
            name = ?, url = ?, events = ?, secret = ?, status = ?,
            failure_count = ?, last_triggered_at = ?, last_success_at = ?,
            last_failure_at = ?, last_failure_reason = ?, updated_at = ?
        WHERE id = ?`,
		w.Name, w.URL, w.Events, w.Secret, w.Status,
		w.FailureCount, w.LastTriggeredAt, w.LastSuccessAt,
		w.LastFailureAt, w.LastFailureReason, w.UpdatedAt, w.ID)
	if err != nil {
		return fmt.Errorf("update webhook: %w", err)
	}
	return nil
}

// Archive soft-deletes a webhook.
func (r *WebhookRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE webhooks SET archived_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("archive webhook: %w", err)
	}
	return nil
}

// RecordTrigger records a webhook trigger attempt.
func (r *WebhookRepository) RecordTrigger(ctx context.Context, id string, success bool, errorMsg string) error {
	now := fmt.Sprintf("'%s'", time.Now().UTC().Format(time.RFC3339))
	if success {
		_, err := r.db.ExecContext(ctx, `
            UPDATE webhooks SET
                failure_count = 0,
                last_triggered_at = ?,
                last_success_at = ?,
                updated_at = ?
            WHERE id = ?`, now, now, now, id)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
        UPDATE webhooks SET
            failure_count = failure_count + 1,
            last_triggered_at = ?,
            last_failure_at = ?,
            last_failure_reason = ?,
            updated_at = ?
        WHERE id = ?`, now, now, errorMsg, now, id)
	return err
}

func webhookFilters(opts WebhookListOptions) (string, []any) {
	conditions := []string{"archived_at IS NULL"}
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
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

func webhookSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "name":
		return "name"
	case "status":
		return "status"
	default:
		return "created_at"
	}
}

func scanWebhook(row *sql.Row) (*model.Webhook, error) {
	w := &model.Webhook{}
	var createdAt, updatedAt string
	if err := row.Scan(
		&w.ID, &w.WorkspaceID, &w.Name, &w.URL, &w.Events, &w.Secret, &w.Status,
		&w.FailureCount, &w.LastTriggeredAt, &w.LastSuccessAt, &w.LastFailureAt, &w.LastFailureReason,
		&createdAt, &updatedAt); err != nil {
		return nil, err
	}
	w.CreatedAt = parseTime(createdAt)
	w.UpdatedAt = parseTime(updatedAt)
	return w, nil
}

func scanWebhookRows(rows *sql.Rows) (*model.Webhook, error) {
	w := &model.Webhook{}
	var createdAt, updatedAt string
	if err := rows.Scan(
		&w.ID, &w.WorkspaceID, &w.Name, &w.URL, &w.Events, &w.Secret, &w.Status,
		&w.FailureCount, &w.LastTriggeredAt, &w.LastSuccessAt, &w.LastFailureAt, &w.LastFailureReason,
		&createdAt, &updatedAt); err != nil {
		return nil, err
	}
	w.CreatedAt = parseTime(createdAt)
	w.UpdatedAt = parseTime(updatedAt)
	return w, nil
}
func (r *WebhookRepository) Count(ctx context.Context, opts WebhookListOptions) (int, error) {
	where, args := webhookFilters(opts)
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhooks"+where, args...).Scan(&total)
	return total, err
}

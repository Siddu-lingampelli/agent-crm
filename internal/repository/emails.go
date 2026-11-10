// Package repository — emails.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"crm.local/internal/model"
)

// EmailListOptions extends ListOptions with email-specific filters.
type EmailListOptions struct {
	Query       string
	WorkspaceID string
	Status      string
	RelatedType string
	RelatedID   string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const emailsColumns = `
    id, workspace_id, to_addr, cc, bcc, subject, body, body_html, status,
    provider, provider_message_id, related_type, related_id, created_by,
    approved_by, approved_at, sent_at, failed_at, error_message, retry_count,
    created_at, updated_at
`

// EmailRepository persists emails.
type EmailRepository struct {
	db *sql.DB
}

// NewEmailRepository returns a repository over db.
func NewEmailRepository(db *sql.DB) *EmailRepository { return &EmailRepository{db: db} }

// Create inserts an email.
func (r *EmailRepository) Create(ctx context.Context, e *model.Email) error {
	if err := validateEntityReferences(ctx, r.db, e); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO emails (
            id, workspace_id, to_addr, cc, bcc, subject, body, body_html, status,
            provider, provider_message_id, related_type, related_id, created_by,
            approved_by, approved_at, sent_at, failed_at, error_message, retry_count,
            created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WorkspaceID, e.To, e.CC, e.BCC, e.Subject, e.Body, e.BodyHTML, e.Status,
		e.Provider, e.ProviderMessageID, e.RelatedType, e.RelatedID, e.CreatedBy,
		e.ApprovedBy, e.ApprovedAt, e.SentAt, e.FailedAt, e.ErrorMessage, e.RetryCount,
		e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert email: %w", err)
	}
	return nil
}

// Get returns an email by id, or nil if not found.
func (r *EmailRepository) Get(ctx context.Context, id string) (*model.Email, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+emailsColumns+` FROM emails WHERE id = ?`, id)
	e, err := scanEmail(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return e, err
}

// List returns a page of emails with server-side pagination.
func (r *EmailRepository) List(ctx context.Context, opts EmailListOptions) (Page[model.Email], error) {
	where, args := emailFilters(opts)

	var total int
	countSQL := `SELECT COUNT(*) FROM emails` + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Email]{}, fmt.Errorf("count emails: %w", err)
	}

	sort := emailSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := `SELECT ` + emailsColumns + ` FROM emails` + where +
		` ORDER BY ` + sort + ` ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Email]{}, fmt.Errorf("list emails: %w", err)
	}
	defer rows.Close()

	items := []model.Email{}
	for rows.Next() {
		item, err := scanEmailRows(rows)
		if err != nil {
			return Page[model.Email]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Email]{}, fmt.Errorf("iterate emails: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies an email.
func (r *EmailRepository) Update(ctx context.Context, e *model.Email) error {
	if err := validateEntityReferences(ctx, r.db, e); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
        UPDATE emails SET
            to_addr = ?, cc = ?, bcc = ?, subject = ?, body = ?, body_html = ?,
            status = ?, provider = ?, provider_message_id = ?, related_type = ?,
            related_id = ?, created_by = ?, approved_by = ?, approved_at = ?,
            sent_at = ?, failed_at = ?, error_message = ?, retry_count = ?,
            updated_at = ?
        WHERE id = ?`,
		e.To, e.CC, e.BCC, e.Subject, e.Body, e.BodyHTML,
		e.Status, e.Provider, e.ProviderMessageID, e.RelatedType,
		e.RelatedID, e.CreatedBy, e.ApprovedBy, e.ApprovedAt,
		e.SentAt, e.FailedAt, e.ErrorMessage, e.RetryCount,
		e.UpdatedAt, e.ID)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	return nil
}

// Delete removes an email (draft only).
func (r *EmailRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM emails WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete email: %w", err)
	}
	return nil
}

// ListFailed returns failed emails eligible for retry.
func (r *EmailRepository) ListFailed(ctx context.Context, workspaceID string) ([]model.Email, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT `+emailsColumns+`
        FROM emails
        WHERE workspace_id = ? AND status = 'failed' AND retry_count < 3
        ORDER BY created_at ASC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list failed emails: %w", err)
	}
	defer rows.Close()

	var emails []model.Email
	for rows.Next() {
		e, err := scanEmailRows(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate failed emails: %w", err)
	}
	return emails, nil
}

// CreateTemplate creates an email template.
func (r *EmailRepository) CreateTemplate(ctx context.Context, t *model.EmailTemplate) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO email_templates (id, workspace_id, name, subject, body, body_html, created_by, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.WorkspaceID, t.Name, t.Subject, t.Body, t.BodyHTML, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert email template: %w", err)
	}
	return nil
}

// GetTemplate returns an email template by id.
func (r *EmailRepository) GetTemplate(ctx context.Context, id string) (*model.EmailTemplate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, name, subject, body, body_html, created_by, created_at, updated_at FROM email_templates WHERE id = ?`, id)
	t, err := scanEmailTemplate(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// ListTemplates returns email templates for a workspace.
func (r *EmailRepository) ListTemplates(ctx context.Context, workspaceID string) ([]model.EmailTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, workspace_id, name, subject, body, body_html, created_by, created_at, updated_at
        FROM email_templates WHERE workspace_id = ? ORDER BY name`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list email templates: %w", err)
	}
	defer rows.Close()

	var templates []model.EmailTemplate
	for rows.Next() {
		t, err := scanEmailTemplateRows(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate email templates: %w", err)
	}
	return templates, nil
}

// UpdateTemplate updates an email template.
func (r *EmailRepository) UpdateTemplate(ctx context.Context, t *model.EmailTemplate) error {
	_, err := r.db.ExecContext(ctx, `
	        UPDATE email_templates SET
	            name = ?, subject = ?, body = ?, body_html = ?, updated_at = ?
	        WHERE id = ?`,
		t.Name, t.Subject, t.Body, t.BodyHTML, t.UpdatedAt, t.ID)
	if err != nil {
		return fmt.Errorf("update email template: %w", err)
	}
	return nil
}

// DeleteTemplate removes an email template.
func (r *EmailRepository) DeleteTemplate(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM email_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete email template: %w", err)
	}
	return nil
}

func emailFilters(opts EmailListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}

	if query := strings.TrimSpace(opts.Query); query != "" {
		conditions = append(conditions, "(subject LIKE ? OR to_addr LIKE ? OR body LIKE ?)")
		term := "%" + query + "%"
		args = append(args, term, term, term)
	}

	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}

	if opts.RelatedType != "" {
		conditions = append(conditions, "related_type = ?")
		args = append(args, opts.RelatedType)
	}

	if opts.RelatedID != "" {
		conditions = append(conditions, "related_id = ?")
		args = append(args, opts.RelatedID)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func emailSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "subject":
		return "subject"
	case "status":
		return "status"
	case "to_addr":
		return "to_addr"
	default:
		return "created_at"
	}
}

func scanEmail(row *sql.Row) (*model.Email, error) {
	e := &model.Email{}
	var approvedAt, sentAt, failedAt *string
	var createdAt, updatedAt string
	if err := row.Scan(
		&e.ID, &e.WorkspaceID, &e.To, &e.CC, &e.BCC, &e.Subject, &e.Body, &e.BodyHTML, &e.Status,
		&e.Provider, &e.ProviderMessageID, &e.RelatedType, &e.RelatedID, &e.CreatedBy,
		&e.ApprovedBy, &approvedAt, &sentAt, &failedAt, &e.ErrorMessage, &e.RetryCount,
		&createdAt, &updatedAt); err != nil {
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	e.UpdatedAt = parseTime(updatedAt)
	if approvedAt != nil {
		t := parseTime(*approvedAt)
		e.ApprovedAt = &t
	}
	if sentAt != nil {
		t := parseTime(*sentAt)
		e.SentAt = &t
	}
	if failedAt != nil {
		t := parseTime(*failedAt)
		e.FailedAt = &t
	}
	return e, nil
}

func scanEmailRows(rows *sql.Rows) (*model.Email, error) {
	e := &model.Email{}
	var approvedAt, sentAt, failedAt *string
	var createdAt, updatedAt string
	if err := rows.Scan(
		&e.ID, &e.WorkspaceID, &e.To, &e.CC, &e.BCC, &e.Subject, &e.Body, &e.BodyHTML, &e.Status,
		&e.Provider, &e.ProviderMessageID, &e.RelatedType, &e.RelatedID, &e.CreatedBy,
		&e.ApprovedBy, &approvedAt, &sentAt, &failedAt, &e.ErrorMessage, &e.RetryCount,
		&createdAt, &updatedAt); err != nil {
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	e.UpdatedAt = parseTime(updatedAt)
	if approvedAt != nil {
		t := parseTime(*approvedAt)
		e.ApprovedAt = &t
	}
	if sentAt != nil {
		t := parseTime(*sentAt)
		e.SentAt = &t
	}
	if failedAt != nil {
		t := parseTime(*failedAt)
		e.FailedAt = &t
	}
	return e, nil
}

func scanEmailTemplate(row *sql.Row) (*model.EmailTemplate, error) {
	t := &model.EmailTemplate{}
	var createdAt, updatedAt string
	if err := row.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Subject, &t.Body, &t.BodyHTML, &t.CreatedBy, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return t, nil
}

func scanEmailTemplateRows(rows *sql.Rows) (*model.EmailTemplate, error) {
	t := &model.EmailTemplate{}
	var createdAt, updatedAt string
	if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Subject, &t.Body, &t.BodyHTML, &t.CreatedBy, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return t, nil
}
func (r *EmailRepository) ClaimForSending(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE emails SET status = 'sending' WHERE id = ? AND status = 'queued'", id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *EmailRepository) QueueForSending(ctx context.Context, e *model.Email) (bool, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE emails SET status='queued', approved_by=?, approved_at=?, updated_at=? WHERE id=? AND status='draft'", e.ApprovedBy, e.ApprovedAt, e.UpdatedAt, e.ID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *EmailRepository) QueueRetry(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE emails SET status='queued' WHERE id=? AND status='failed' AND retry_count<3", id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

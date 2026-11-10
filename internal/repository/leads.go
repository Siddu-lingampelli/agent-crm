// Package repository - leads.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"crm.local/internal/model"
)

// LeadListOptions extends ListOptions with lead filters.
type LeadListOptions struct {
	WorkspaceID string
	Query       string
	Status      string
	Source      string
	CreatedFrom string
	OwnerID     string
	Priority    string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const leadsColumns = `
    id, workspace_id, display_name, first_name, last_name, email, phone, company_name,
    company_id, job_title, status, source, owner_id, priority, estimated_value, currency,
    lead_score, expected_close, follow_up_date, tags, custom_fields, description,
    qualification, created_by, updated_by, created_at, updated_at, archived_at, converted_at
`

// LeadRepository persists leads.
type LeadRepository struct {
	db *sql.DB
}

// NewLeadRepository returns a repository over db.
func NewLeadRepository(db *sql.DB) *LeadRepository {
	return &LeadRepository{db: db}
}

// Create inserts a lead.
func (r *LeadRepository) Create(ctx context.Context, l *model.Lead) error {
	if err := validateEntityReferences(ctx, r.db, l); err != nil {
		return err
	}
	tags := jsonOrDefault(l.Tags)
	custom := jsonOrDefaultMap(l.CustomFields)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO leads (
            id, workspace_id, display_name, first_name, last_name, email, phone, company_name,
            company_id, job_title, status, source, owner_id, priority, estimated_value, currency,
            lead_score, expected_close, follow_up_date, tags, custom_fields, description,
            qualification, created_by, updated_by, created_at, updated_at, archived_at, converted_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.WorkspaceID, l.DisplayName, l.FirstName, l.LastName, l.Email, l.Phone,
		l.CompanyName, nullIfEmpty(l.CompanyID), l.JobTitle, l.Status, l.Source, nullIfEmpty(l.OwnerID), l.Priority,
		l.EstimatedValue, l.Currency, l.LeadScore, l.ExpectedClose, l.FollowUpDate,
		tags, custom, l.Description, l.Qualification, l.CreatedBy, l.UpdatedBy,
		l.CreatedAt, l.UpdatedAt, l.ArchivedAt, l.ConvertedAt)
	if err != nil {
		return fmt.Errorf("insert lead: %w", err)
	}
	return nil
}

// Get returns a lead by id.
func (r *LeadRepository) Get(ctx context.Context, id string) (*model.Lead, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+leadsColumns+" FROM leads WHERE id = ? AND archived_at IS NULL", id)
	l, err := scanLead(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

// List returns a page of leads.
func (r *LeadRepository) List(ctx context.Context, opts LeadListOptions) (Page[model.Lead], error) {
	where, args := leadFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM leads" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Lead]{}, fmt.Errorf("count leads: %w", err)
	}

	sort := leadSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := "SELECT " + leadsColumns + " FROM leads" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Lead]{}, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	items := []model.Lead{}
	for rows.Next() {
		item, err := scanLeadRows(rows)
		if err != nil {
			return Page[model.Lead]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Lead]{}, fmt.Errorf("iterate leads: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies a lead.
func (r *LeadRepository) Update(ctx context.Context, l *model.Lead) error {
	if err := validateEntityReferences(ctx, r.db, l); err != nil {
		return err
	}
	tags := jsonOrDefault(l.Tags)
	custom := jsonOrDefaultMap(l.CustomFields)

	companyID := nullIfEmpty(l.CompanyID)
	ownerID := nullIfEmpty(l.OwnerID)

	_, err := r.db.ExecContext(ctx, `
        UPDATE leads SET
            display_name = ?, first_name = ?, last_name = ?, email = ?, phone = ?,
            company_name = ?, company_id = ?, job_title = ?, status = ?, source = ?,
            owner_id = ?, priority = ?, estimated_value = ?, currency = ?, lead_score = ?,
            expected_close = ?, follow_up_date = ?, tags = ?, custom_fields = ?,
            description = ?, qualification = ?, updated_by = ?, updated_at = ?,
            archived_at = ?, converted_at = ?
        WHERE id = ? AND archived_at IS NULL`,
		l.DisplayName, l.FirstName, l.LastName, l.Email, l.Phone, l.CompanyName,
		companyID, l.JobTitle, l.Status, l.Source, ownerID, l.Priority,
		l.EstimatedValue, l.Currency, l.LeadScore, l.ExpectedClose, l.FollowUpDate,
		tags, custom, l.Description, l.Qualification, l.UpdatedBy, l.UpdatedAt,
		l.ArchivedAt, l.ConvertedAt, l.ID)
	if err != nil {
		return fmt.Errorf("update lead: %w", err)
	}
	return nil
}

// Archive soft-deletes a lead.
func (r *LeadRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE leads SET archived_at = ? WHERE id = ? AND archived_at IS NULL", now, id)
	if err != nil {
		return fmt.Errorf("archive lead: %w", err)
	}
	return nil
}

// Restore un-archives a lead.
func (r *LeadRepository) Restore(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE leads SET archived_at = NULL WHERE id = ? AND archived_at IS NOT NULL", id)
	if err != nil {
		return fmt.Errorf("restore lead: %w", err)
	}
	return nil
}

// Convert marks a lead as converted.
func (r *LeadRepository) Convert(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE leads SET status = ?, converted_at = ? WHERE id = ?", "converted", now, id)
	if err != nil {
		return fmt.Errorf("convert lead: %w", err)
	}
	return nil
}

// FindDuplicate returns a lead with matching email or phone.
func (r *LeadRepository) FindDuplicate(ctx context.Context, workspaceID, email, phone string) (*model.Lead, error) {
	if email == "" && phone == "" {
		return nil, nil
	}
	var row *sql.Row
	if email != "" {
		row = r.db.QueryRowContext(ctx,
			"SELECT "+leadsColumns+" FROM leads WHERE workspace_id = ? AND archived_at IS NULL AND email = ? LIMIT 1",
			workspaceID, email)
	} else {
		row = r.db.QueryRowContext(ctx,
			"SELECT "+leadsColumns+" FROM leads WHERE workspace_id = ? AND archived_at IS NULL AND phone = ? LIMIT 1",
			workspaceID, phone)
	}
	l, err := scanLead(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

func leadFilters(opts LeadListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	conditions = append(conditions, "archived_at IS NULL")

	if opts.Query != "" {
		conditions = append(conditions, "(display_name LIKE ? OR email LIKE ? OR company_name LIKE ? OR phone LIKE ?)")
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term, term, term)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, opts.Source)
	}
	if opts.OwnerID != "" {
		conditions = append(conditions, "owner_id = ?")
		args = append(args, opts.OwnerID)
	}
	if opts.Priority != "" {
		conditions = append(conditions, "priority = ?")
		args = append(args, opts.Priority)
	}
	if opts.CreatedFrom != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, opts.CreatedFrom)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func leadSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "display_name":
		return "display_name"
	case "estimated_value":
		return "estimated_value"
	case "follow_up_date":
		return "follow_up_date"
	default:
		return "created_at"
	}
}

func scanLead(row *sql.Row) (*model.Lead, error) {
	l := &model.Lead{}
	var tagsJSON, customJSON string
	var createdAt, updatedAt string
	var archivedAt, convertedAt *string
	var firstName, lastName, email, phone, companyName, companyID, jobTitle, source, ownerID, currency, expectedClose, followUpDate, description, qualification, createdBy, updatedBy sql.NullString
	if err := row.Scan(
		&l.ID, &l.WorkspaceID, &l.DisplayName, &firstName, &lastName,
		&email, &phone, &companyName, &companyID, &jobTitle,
		&l.Status, &source, &ownerID, &l.Priority, &l.EstimatedValue,
		&currency, &l.LeadScore, &expectedClose, &followUpDate,
		&tagsJSON, &customJSON, &description, &qualification,
		&createdBy, &updatedBy, &createdAt, &updatedAt, &archivedAt, &convertedAt); err != nil {
		return nil, err
	}
	l.FirstName = firstName.String
	l.LastName = lastName.String
	l.Email = email.String
	l.Phone = phone.String
	l.CompanyName = companyName.String
	l.CompanyID = companyID.String
	l.JobTitle = jobTitle.String
	l.Source = source.String
	l.OwnerID = ownerID.String
	l.Currency = currency.String
	l.ExpectedClose = expectedClose.String
	l.FollowUpDate = followUpDate.String
	l.Description = description.String
	l.Qualification = qualification.String
	l.CreatedBy = createdBy.String
	l.UpdatedBy = updatedBy.String
	l.CreatedAt = parseTime(createdAt)
	l.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		l.ArchivedAt = &t
	}
	if convertedAt != nil {
		t := parseTime(*convertedAt)
		l.ConvertedAt = &t
	}
	l.Tags = []string{}
	_ = json.Unmarshal([]byte(tagsJSON), &l.Tags)
	l.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &l.CustomFields)
	return l, nil
}

func scanLeadRows(rows *sql.Rows) (*model.Lead, error) {
	l := &model.Lead{}
	var tagsJSON, customJSON string
	var createdAt, updatedAt string
	var archivedAt, convertedAt *string
	var firstName, lastName, email, phone, companyName, companyID, jobTitle, source, ownerID, currency, expectedClose, followUpDate, description, qualification, createdBy, updatedBy sql.NullString
	if err := rows.Scan(
		&l.ID, &l.WorkspaceID, &l.DisplayName, &firstName, &lastName,
		&email, &phone, &companyName, &companyID, &jobTitle,
		&l.Status, &source, &ownerID, &l.Priority, &l.EstimatedValue,
		&currency, &l.LeadScore, &expectedClose, &followUpDate,
		&tagsJSON, &customJSON, &description, &qualification,
		&createdBy, &updatedBy, &createdAt, &updatedAt, &archivedAt, &convertedAt); err != nil {
		return nil, err
	}
	l.FirstName = firstName.String
	l.LastName = lastName.String
	l.Email = email.String
	l.Phone = phone.String
	l.CompanyName = companyName.String
	l.CompanyID = companyID.String
	l.JobTitle = jobTitle.String
	l.Source = source.String
	l.OwnerID = ownerID.String
	l.Currency = currency.String
	l.ExpectedClose = expectedClose.String
	l.FollowUpDate = followUpDate.String
	l.Description = description.String
	l.Qualification = qualification.String
	l.CreatedBy = createdBy.String
	l.UpdatedBy = updatedBy.String
	l.CreatedAt = parseTime(createdAt)
	l.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		l.ArchivedAt = &t
	}
	if convertedAt != nil {
		t := parseTime(*convertedAt)
		l.ConvertedAt = &t
	}
	l.Tags = []string{}
	_ = json.Unmarshal([]byte(tagsJSON), &l.Tags)
	l.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &l.CustomFields)
	return l, nil
}

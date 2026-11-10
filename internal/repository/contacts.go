// Package repository - contacts.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"crm.local/internal/model"
)

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ContactListOptions extends ListOptions with contact filters.
type ContactListOptions struct {
	WorkspaceID string
	Query       string
	CompanyID   string
	Status      string
	CreatedFrom string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const contactsColumns = `
    id, workspace_id, first_name, last_name, email, phone, job_title, company_id,
    owner_id, status, tags, custom_fields, notes, created_at, updated_at, archived_at
`

// ContactRepository persists contacts.
type ContactRepository struct {
	db *sql.DB
}

// NewContactRepository returns a repository over db.
func NewContactRepository(db *sql.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

// Create inserts a contact.
func (r *ContactRepository) Create(ctx context.Context, c *model.Contact) error {
	if err := validateEntityReferences(ctx, r.db, c); err != nil {
		return err
	}
	tags := jsonOrDefault(c.Tags)
	custom := jsonOrDefaultMap(c.CustomFields)

	companyID := nullIfEmpty(c.CompanyID)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO contacts (
            id, workspace_id, first_name, last_name, email, phone, job_title, company_id,
            owner_id, status, tags, custom_fields, notes, created_at, updated_at, archived_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.FirstName, c.LastName, c.Email, c.Phone, c.JobTitle,
		companyID, c.OwnerID, c.Status, tags, custom, c.Notes, c.CreatedAt, c.UpdatedAt, c.ArchivedAt)
	if err != nil {
		return fmt.Errorf("insert contact: %w", err)
	}
	return nil
}

// Get returns a contact by id, or nil if not found.
func (r *ContactRepository) Get(ctx context.Context, id string) (*model.Contact, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+contactsColumns+" FROM contacts WHERE id = ? AND archived_at IS NULL", id)
	c, err := scanContact(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// List returns a page of contacts.
func (r *ContactRepository) List(ctx context.Context, opts ContactListOptions) (Page[model.Contact], error) {
	where, args := contactFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM contacts" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Contact]{}, fmt.Errorf("count contacts: %w", err)
	}

	sort := contactSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// sort and order are validated by contactSort and NormalizeOrder, safe to use in query
	//nolint:gosec // G202: sort and order are validated by contactSort and NormalizeOrder
	sqlText := "SELECT " + contactsColumns + " FROM contacts" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Contact]{}, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	items := []model.Contact{}
	for rows.Next() {
		item, err := scanContactRows(rows)
		if err != nil {
			return Page[model.Contact]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Contact]{}, fmt.Errorf("iterate contacts: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies a contact.
func (r *ContactRepository) Update(ctx context.Context, c *model.Contact) error {
	if err := validateEntityReferences(ctx, r.db, c); err != nil {
		return err
	}
	tags := jsonOrDefault(c.Tags)
	custom := jsonOrDefaultMap(c.CustomFields)

	companyID := nullIfEmpty(c.CompanyID)

	_, err := r.db.ExecContext(ctx, `
        UPDATE contacts SET
            first_name = ?, last_name = ?, email = ?, phone = ?, job_title = ?,
            company_id = ?, owner_id = ?, status = ?, tags = ?, custom_fields = ?,
            notes = ?, updated_at = ?, archived_at = ?
        WHERE id = ? AND archived_at IS NULL`,
		c.FirstName, c.LastName, c.Email, c.Phone, c.JobTitle, companyID,
		c.OwnerID, c.Status, tags, custom, c.Notes, c.UpdatedAt, c.ArchivedAt, c.ID)
	if err != nil {
		return fmt.Errorf("update contact: %w", err)
	}
	return nil
}

// Archive soft-deletes a contact.
func (r *ContactRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE contacts SET archived_at = ? WHERE id = ? AND archived_at IS NULL", now, id)
	if err != nil {
		return fmt.Errorf("archive contact: %w", err)
	}
	return nil
}

// Restore un-archives a contact.
func (r *ContactRepository) Restore(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE contacts SET archived_at = NULL WHERE id = ? AND archived_at IS NOT NULL", id)
	if err != nil {
		return fmt.Errorf("restore contact: %w", err)
	}
	return nil
}

// FindDuplicate returns a contact with matching email or exact first+last name.
func (r *ContactRepository) FindDuplicate(ctx context.Context, workspaceID, email, firstName, lastName string) (*model.Contact, error) {
	if email == "" {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx,
		"SELECT "+contactsColumns+" FROM contacts WHERE workspace_id = ? AND archived_at IS NULL AND email = ? LIMIT 1",
		workspaceID, email)
	c, err := scanContact(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func contactFilters(opts ContactListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	conditions = append(conditions, "archived_at IS NULL")

	if opts.Query != "" {
		conditions = append(conditions, "(first_name LIKE ? OR last_name LIKE ? OR email LIKE ?)")
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term, term)
	}
	if opts.CompanyID != "" {
		conditions = append(conditions, "company_id = ?")
		args = append(args, opts.CompanyID)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
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

func contactSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "first_name":
		return "first_name"
	case "last_name":
		return "last_name"
	case "email":
		return "email"
	default:
		return "created_at"
	}
}

func scanContact(row *sql.Row) (*model.Contact, error) {
	c := &model.Contact{}
	var tagsJSON, customJSON string
	var createdAt, updatedAt string
	var archivedAt *string
	var companyID *string
	if err := row.Scan(
		&c.ID, &c.WorkspaceID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.JobTitle, &companyID, &c.OwnerID, &c.Status, &tagsJSON, &customJSON,
		&c.Notes, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	if companyID != nil {
		c.CompanyID = *companyID
	}
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		c.ArchivedAt = &t
	}
	c.Tags = []string{}
	_ = json.Unmarshal([]byte(tagsJSON), &c.Tags)
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	return c, nil
}

func scanContactRows(rows *sql.Rows) (*model.Contact, error) {
	c := &model.Contact{}
	var tagsJSON, customJSON string
	var createdAt, updatedAt string
	var archivedAt *string
	var companyID *string
	if err := rows.Scan(
		&c.ID, &c.WorkspaceID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.JobTitle, &companyID, &c.OwnerID, &c.Status, &tagsJSON, &customJSON,
		&c.Notes, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	if companyID != nil {
		c.CompanyID = *companyID
	}
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		c.ArchivedAt = &t
	}
	c.Tags = []string{}
	_ = json.Unmarshal([]byte(tagsJSON), &c.Tags)
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	return c, nil
}

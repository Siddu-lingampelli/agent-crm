// Package repository — projects.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"crm.local/internal/model"
)

// ProjectListOptions extends ListOptions with project-specific filters.
type ProjectListOptions struct {
	Priority    string
	WorkspaceID string
	Query       string
	Status      string
	OwnerID     string
	CompanyID   string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const projectsColumns = `
    id, workspace_id, name, description, status, priority, owner_id, company_id,
    start_date, end_date, progress, custom_fields, created_at, updated_at, archived_at
`

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository { return &ProjectRepository{db: db} }

func (r *ProjectRepository) Create(ctx context.Context, c *model.Project) error {
	if err := validateEntityReferences(ctx, r.db, c); err != nil {
		return err
	}
	custom := jsonOrDefaultMap(c.CustomFields)
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO projects (
            id, workspace_id, name, description, status, priority, owner_id, company_id,
            start_date, end_date, progress, custom_fields, created_at, updated_at, archived_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.Name, c.Description, c.Status, c.Priority, nullIfEmpty(c.OwnerID),
		nullIfEmpty(c.CompanyID), c.StartDate, c.EndDate, c.Progress, custom, c.CreatedAt, c.UpdatedAt, c.ArchivedAt)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) Get(ctx context.Context, id string) (*model.Project, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+projectsColumns+" FROM projects WHERE id = ? AND archived_at IS NULL", id)
	c, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *ProjectRepository) List(ctx context.Context, opts ProjectListOptions) (Page[model.Project], error) {
	where, args := projectFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM projects" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Project]{}, fmt.Errorf("count projects: %w", err)
	}

	sort := projectSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := "SELECT " + projectsColumns + " FROM projects" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Project]{}, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	items := []model.Project{}
	for rows.Next() {
		item, err := scanProjectRows(rows)
		if err != nil {
			return Page[model.Project]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Project]{}, fmt.Errorf("iterate projects: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

func (r *ProjectRepository) Update(ctx context.Context, c *model.Project) error {
	if err := validateEntityReferences(ctx, r.db, c); err != nil {
		return err
	}
	custom := jsonOrDefaultMap(c.CustomFields)
	_, err := r.db.ExecContext(ctx, `
        UPDATE projects SET
            name = ?, description = ?, status = ?, priority = ?, owner_id = ?,
            company_id = ?, start_date = ?, end_date = ?, progress = ?,
            custom_fields = ?, updated_at = ?, archived_at = ?
        WHERE id = ? AND archived_at IS NULL`,
		c.Name, c.Description, c.Status, c.Priority, nullIfEmpty(c.OwnerID),
		nullIfEmpty(c.CompanyID), c.StartDate, c.EndDate, c.Progress,
		custom, c.UpdatedAt, c.ArchivedAt, c.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE projects SET archived_at = ? WHERE id = ? AND archived_at IS NULL", now, id)
	if err != nil {
		return fmt.Errorf("archive project: %w", err)
	}
	return nil
}

func projectFilters(opts ProjectListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	conditions = append(conditions, "archived_at IS NULL")

	if opts.Query != "" {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.Priority != "" {
		conditions = append(conditions, "priority = ?")
		args = append(args, opts.Priority)
	}
	if opts.OwnerID != "" {
		conditions = append(conditions, "owner_id = ?")
		args = append(args, opts.OwnerID)
	}
	if opts.CompanyID != "" {
		conditions = append(conditions, "company_id = ?")
		args = append(args, opts.CompanyID)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func projectSort(field string) string {
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

func scanProject(row *sql.Row) (*model.Project, error) {
	c := &model.Project{}
	var customJSON string
	var progress int
	var createdAt, updatedAt string
	var archivedAt *string
	var ownerID, companyID sql.NullString
	if err := row.Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Description, &c.Status, &c.Priority,
		&ownerID, &companyID, &c.StartDate, &c.EndDate, &progress, &customJSON,
		&createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	c.OwnerID = ownerID.String
	c.CompanyID = companyID.String
	c.Progress = progress
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		if t := parseTime(*archivedAt); !t.IsZero() {
			c.ArchivedAt = &t
		}
	}
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	return c, nil
}

func scanProjectRows(rows *sql.Rows) (*model.Project, error) {
	c := &model.Project{}
	var customJSON string
	var progress int
	var createdAt, updatedAt string
	var archivedAt *string
	var ownerID, companyID sql.NullString
	if err := rows.Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Description, &c.Status, &c.Priority,
		&ownerID, &companyID, &c.StartDate, &c.EndDate, &progress, &customJSON,
		&createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	c.OwnerID = ownerID.String
	c.CompanyID = companyID.String
	c.Progress = progress
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		if t := parseTime(*archivedAt); !t.IsZero() {
			c.ArchivedAt = &t
		}
	}
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	return c, nil
}

// Package repository — tasks.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
)

func nullIfEmptyTime(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// TaskListOptions extends ListOptions with task-specific filters.
type TaskListOptions struct {
	ActiveOnly  bool
	DueAfter    string
	Unscheduled bool
	DueBefore   string
	DueOn       string
	WorkspaceID string
	Query       string
	Status      string
	Priority    string
	AssigneeID  string
	RelatedType string
	RelatedID   string
	ProjectID   string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const tasksColumns = `
    id, workspace_id, title, description, status, priority, COALESCE(due_date,'') AS due_date, assignee_id,
    related_type, related_id, project_id, created_by, completed_at, custom_fields,
    created_at, updated_at, archived_at
`

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository { return &TaskRepository{db: db} }

func (r *TaskRepository) Create(ctx context.Context, c *model.Task) error {
	if err := validateEntityReferences(ctx, r.db, c); err != nil {
		return err
	}
	custom := jsonOrDefaultMap(c.CustomFields)
	completedAt := nullIfEmptyTime(c.CompletedAt)
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO tasks (
            id, workspace_id, title, description, status, priority, due_date, assignee_id,
            related_type, related_id, project_id, created_by, completed_at, custom_fields,
            created_at, updated_at, archived_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.Title, c.Description, c.Status, c.Priority, c.DueDate,
		c.AssigneeID, c.RelatedType, c.RelatedID, c.ProjectID, c.CreatedBy, completedAt,
		custom, c.CreatedAt, c.UpdatedAt, c.ArchivedAt)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepository) Get(ctx context.Context, id string) (*model.Task, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+tasksColumns+" FROM tasks WHERE id = ? AND archived_at IS NULL", id)
	c, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *TaskRepository) List(ctx context.Context, opts TaskListOptions) (Page[model.Task], error) {
	where, args := taskFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM tasks" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Task]{}, fmt.Errorf("count tasks: %w", err)
	}

	sort := taskSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := "SELECT " + tasksColumns + " FROM tasks" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Task]{}, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	items := []model.Task{}
	for rows.Next() {
		item, err := scanTaskRows(rows)
		if err != nil {
			return Page[model.Task]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Task]{}, fmt.Errorf("iterate tasks: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

func (r *TaskRepository) Update(ctx context.Context, c *model.Task) error {
	if err := validateEntityReferences(ctx, r.db, c); err != nil {
		return err
	}
	custom := jsonOrDefaultMap(c.CustomFields)
	completedAt := nullIfEmptyTime(c.CompletedAt)
	_, err := r.db.ExecContext(ctx, `
        UPDATE tasks SET
            title = ?, description = ?, status = ?, priority = ?, due_date = ?,
            assignee_id = ?, related_type = ?, related_id = ?, project_id = ?,
            custom_fields = ?, updated_at = ?, completed_at = ?, archived_at = ?
        WHERE id = ? AND archived_at IS NULL`,
		c.Title, c.Description, c.Status, c.Priority, c.DueDate,
		nullIfEmpty(c.AssigneeID), nullIfEmpty(c.RelatedType), nullIfEmpty(c.RelatedID), nullIfEmpty(c.ProjectID),
		custom, c.UpdatedAt, completedAt, c.ArchivedAt, c.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *TaskRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE tasks SET archived_at = ? WHERE id = ? AND archived_at IS NULL", now, id)
	if err != nil {
		return fmt.Errorf("archive task: %w", err)
	}
	return nil
}

func taskFilters(opts TaskListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	conditions = append(conditions, "archived_at IS NULL")

	if opts.Query != "" {
		conditions = append(conditions, "(title LIKE ? OR description LIKE ?)")
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.ActiveOnly {
		conditions = append(conditions, "status NOT IN ('done','completed','cancelled')")
	}
	if opts.Unscheduled {
		conditions = append(conditions, "(due_date IS NULL OR due_date = '')")
	}
	if opts.DueAfter != "" {
		conditions = append(conditions, "due_date != '' AND substr(due_date,1,10)>?")
		args = append(args, opts.DueAfter)
	}
	if opts.DueBefore != "" {
		conditions = append(conditions, "due_date != '' AND substr(due_date, 1, 10) < ? AND status NOT IN ('done', 'completed', 'cancelled')")
		args = append(args, opts.DueBefore)
	}
	if opts.DueOn != "" {
		conditions = append(conditions, "substr(due_date, 1, 10) = ? AND status NOT IN ('done', 'completed', 'cancelled')")
		args = append(args, opts.DueOn)
	}

	if opts.Priority != "" {
		conditions = append(conditions, "priority = ?")
		args = append(args, opts.Priority)
	}
	if opts.AssigneeID != "" {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, opts.AssigneeID)
	}
	if opts.RelatedType != "" {
		conditions = append(conditions, "related_type = ?")
		args = append(args, opts.RelatedType)
	}
	if opts.RelatedID != "" {
		conditions = append(conditions, "related_id = ?")
		args = append(args, opts.RelatedID)
	}
	if opts.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, opts.ProjectID)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func taskSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "due_date", "due":
		return "CASE WHEN due_date IS NULL OR due_date = '' THEN 1 ELSE 0 END, due_date"
	case "title":
		return "title"
	default:
		return "created_at"
	}
}

func scanTask(row *sql.Row) (*model.Task, error) {
	c := &model.Task{}
	var customJSON string
	var createdAt, updatedAt string
	var archivedAt *string
	var completedAt *string
	var assigneeID, relatedType, relatedID, projectID sql.NullString
	if err := row.Scan(
		&c.ID, &c.WorkspaceID, &c.Title, &c.Description, &c.Status, &c.Priority,
		&c.DueDate, &assigneeID, &relatedType, &relatedID, &projectID,
		&c.CreatedBy, &completedAt, &customJSON, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	c.AssigneeID = assigneeID.String
	c.RelatedType = relatedType.String
	c.RelatedID = relatedID.String
	c.ProjectID = projectID.String
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		if t := parseTime(*archivedAt); !t.IsZero() {
			c.ArchivedAt = &t
		}
	}
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	if completedAt != nil {
		if t := parseTime(*completedAt); !t.IsZero() {
			c.CompletedAt = &t
		}
	}
	return c, nil
}

func scanTaskRows(rows *sql.Rows) (*model.Task, error) {
	c := &model.Task{}
	var customJSON string
	var createdAt, updatedAt string
	var archivedAt *string
	var completedAt *string
	var assigneeID, relatedType, relatedID, projectID sql.NullString
	if err := rows.Scan(
		&c.ID, &c.WorkspaceID, &c.Title, &c.Description, &c.Status, &c.Priority,
		&c.DueDate, &assigneeID, &relatedType, &relatedID, &projectID,
		&c.CreatedBy, &completedAt, &customJSON, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	c.AssigneeID = assigneeID.String
	c.RelatedType = relatedType.String
	c.RelatedID = relatedID.String
	c.ProjectID = projectID.String
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		if t := parseTime(*archivedAt); !t.IsZero() {
			c.ArchivedAt = &t
		}
	}
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	if completedAt != nil {
		if t := parseTime(*completedAt); !t.IsZero() {
			c.CompletedAt = &t
		}
	}
	return c, nil
}

// Package repository — workflows.
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

func (r *WorkflowRepository) RecordResult(ctx context.Context, id string, success bool, now time.Time) error {
	succeeded, failed := 0, 1
	if success {
		succeeded, failed = 1, 0
	}
	_, err := r.db.ExecContext(ctx, "UPDATE workflows SET run_count=run_count+1, success_count=success_count+?, failure_count=failure_count+?, last_run_at=?, updated_at=? WHERE id=?", succeeded, failed, now, now, id)
	return err
}

// WorkflowListOptions extends ListOptions with workflow-specific filters.
type WorkflowListOptions struct {
	WorkspaceID string
	Query       string
	Status      string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const workflowsColumns = `
    id, workspace_id, name, description, trigger_type, trigger_config, steps,
    status, created_by, updated_by, last_run_at, next_run_at, run_count,
    success_count, failure_count, created_at, updated_at
`

const executionsColumns = `
    id, workflow_id, workspace_id, status, trigger_data, steps_run, steps_total,
    error_message, started_at, completed_at, created_at
`

// WorkflowRepository persists workflows.
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository returns a repository over db.
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository { return &WorkflowRepository{db: db} }

// Create inserts a workflow.
func (r *WorkflowRepository) Create(ctx context.Context, w *model.Workflow) error {
	triggerConfig, _ := json.Marshal(w.TriggerConfig)
	steps, _ := json.Marshal(w.Steps)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO workflows (
            id, workspace_id, name, description, trigger_type, trigger_config, steps,
            status, created_by, updated_by, last_run_at, next_run_at, run_count,
            success_count, failure_count, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.WorkspaceID, w.Name, w.Description, w.TriggerType, triggerConfig, steps,
		w.Status, w.CreatedBy, w.UpdatedBy, w.LastRunAt, w.NextRunAt, w.RunCount,
		w.SuccessCount, w.FailureCount, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert workflow: %w", err)
	}
	return nil
}

// Get returns a workflow by id, or nil if not found.
func (r *WorkflowRepository) Get(ctx context.Context, id string) (*model.Workflow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+workflowsColumns+` FROM workflows WHERE id = ? AND archived_at IS NULL`, id)
	w, err := scanWorkflow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// List returns a page of workflows with server-side pagination.
func (r *WorkflowRepository) List(ctx context.Context, opts WorkflowListOptions) (Page[model.Workflow], error) {
	where, args := workflowFilters(opts)

	var total int
	countSQL := `SELECT COUNT(*) FROM workflows` + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Workflow]{}, fmt.Errorf("count workflows: %w", err)
	}

	sort := workflowSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := `SELECT ` + workflowsColumns + ` FROM workflows` + where +
		` ORDER BY ` + sort + ` ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Workflow]{}, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	items := []model.Workflow{}
	for rows.Next() {
		item, err := scanWorkflowRows(rows)
		if err != nil {
			return Page[model.Workflow]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Workflow]{}, fmt.Errorf("iterate workflows: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies a workflow.
func (r *WorkflowRepository) Update(ctx context.Context, w *model.Workflow) error {
	triggerConfig, _ := json.Marshal(w.TriggerConfig)
	steps, _ := json.Marshal(w.Steps)

	_, err := r.db.ExecContext(ctx, `
        UPDATE workflows SET
            name = ?, description = ?, trigger_type = ?, trigger_config = ?, steps = ?,
            status = ?, created_by = ?, updated_by = ?, last_run_at = ?, next_run_at = ?,
            run_count = ?, success_count = ?, failure_count = ?, updated_at = ?
        WHERE id = ?`,
		w.Name, w.Description, w.TriggerType, triggerConfig, steps,
		w.Status, w.CreatedBy, w.UpdatedBy, w.LastRunAt, w.NextRunAt,
		w.RunCount, w.SuccessCount, w.FailureCount, w.UpdatedAt, w.ID)
	if err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}
	return nil
}

// Archive soft-deletes a workflow.
func (r *WorkflowRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workflows SET archived_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("archive workflow: %w", err)
	}
	return nil
}

// CreateExecution creates a workflow execution.
func (r *WorkflowRepository) CreateExecution(ctx context.Context, e *model.Execution) error {
	triggerData, _ := json.Marshal(e.TriggerData)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO workflow_executions (
            id, workflow_id, workspace_id, status, trigger_data, steps_run, steps_total,
            error_message, started_at, completed_at, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WorkflowID, e.WorkspaceID, e.Status, triggerData, e.StepsRun, e.StepsTotal,
		e.ErrorMessage, e.StartedAt, e.CompletedAt, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}
	return nil
}

// GetExecution returns an execution by id.
func (r *WorkflowRepository) GetExecution(ctx context.Context, id string) (*model.Execution, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+executionsColumns+` FROM workflow_executions WHERE id = ?`, id)
	e, err := scanExecution(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return e, err
}

// UpdateExecution updates an execution.
func (r *WorkflowRepository) UpdateExecution(ctx context.Context, e *model.Execution) error {
	triggerData, _ := json.Marshal(e.TriggerData)

	_, err := r.db.ExecContext(ctx, `
        UPDATE workflow_executions SET
            status = ?, trigger_data = ?, steps_run = ?, steps_total = ?,
            error_message = ?, started_at = ?, completed_at = ?
        WHERE id = ?`,
		e.Status, triggerData, e.StepsRun, e.StepsTotal,
		e.ErrorMessage, e.StartedAt, e.CompletedAt, e.ID)
	if err != nil {
		return fmt.Errorf("update execution: %w", err)
	}
	return nil
}

// ListExecutions returns executions for a workflow.
func (r *WorkflowRepository) ListExecutions(ctx context.Context, workspaceID, workflowID string, limit, offset int) ([]model.Execution, error) {
	var args []any
	sqlText := `SELECT ` + executionsColumns + ` FROM workflow_executions WHERE workspace_id = ?`
	args = append(args, workspaceID)

	if workflowID != "" {
		sqlText += ` AND workflow_id = ?`
		args = append(args, workflowID)
	}

	sqlText += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()

	var executions []model.Execution
	for rows.Next() {
		e, err := scanExecutionRows(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate executions: %w", err)
	}
	return executions, nil
}

func workflowFilters(opts WorkflowListOptions) (string, []any) {
	conditions := []string{"archived_at IS NULL"}
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}

	if opts.Query != "" {
		conditions = append(conditions, `(name LIKE ? OR description LIKE ?)`)
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term)
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

func workflowSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "name":
		return "name"
	case "status":
		return "status"
	case "run_count":
		return "run_count"
	default:
		return "created_at"
	}
}

func scanWorkflow(row *sql.Row) (*model.Workflow, error) {
	w := &model.Workflow{}
	var triggerConfigJSON, stepsJSON string
	var lastRunAt, nextRunAt *string
	var createdAt, updatedAt string
	if err := row.Scan(
		&w.ID, &w.WorkspaceID, &w.Name, &w.Description, &w.TriggerType, &triggerConfigJSON, &stepsJSON,
		&w.Status, &w.CreatedBy, &w.UpdatedBy, &lastRunAt, &nextRunAt, &w.RunCount,
		&w.SuccessCount, &w.FailureCount, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	w.CreatedAt = parseTime(createdAt)
	w.UpdatedAt = parseTime(updatedAt)
	if lastRunAt != nil {
		t := parseTime(*lastRunAt)
		w.LastRunAt = &t
	}
	if nextRunAt != nil {
		t := parseTime(*nextRunAt)
		w.NextRunAt = &t
	}
	_ = json.Unmarshal([]byte(triggerConfigJSON), &w.TriggerConfig)
	_ = json.Unmarshal([]byte(stepsJSON), &w.Steps)
	return w, nil
}

func scanWorkflowRows(rows *sql.Rows) (*model.Workflow, error) {
	w := &model.Workflow{}
	var triggerConfigJSON, stepsJSON string
	var lastRunAt, nextRunAt *string
	var createdAt, updatedAt string
	if err := rows.Scan(
		&w.ID, &w.WorkspaceID, &w.Name, &w.Description, &w.TriggerType, &triggerConfigJSON, &stepsJSON,
		&w.Status, &w.CreatedBy, &w.UpdatedBy, &lastRunAt, &nextRunAt, &w.RunCount,
		&w.SuccessCount, &w.FailureCount, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	w.CreatedAt = parseTime(createdAt)
	w.UpdatedAt = parseTime(updatedAt)
	if lastRunAt != nil {
		t := parseTime(*lastRunAt)
		w.LastRunAt = &t
	}
	if nextRunAt != nil {
		t := parseTime(*nextRunAt)
		w.NextRunAt = &t
	}
	_ = json.Unmarshal([]byte(triggerConfigJSON), &w.TriggerConfig)
	_ = json.Unmarshal([]byte(stepsJSON), &w.Steps)
	return w, nil
}

func scanExecution(row *sql.Row) (*model.Execution, error) {
	e := &model.Execution{}
	var triggerDataJSON string
	var completedAt *string
	var startedAt, createdAt string
	if err := row.Scan(
		&e.ID, &e.WorkflowID, &e.WorkspaceID, &e.Status, &triggerDataJSON, &e.StepsRun, &e.StepsTotal,
		&e.ErrorMessage, &startedAt, &completedAt, &createdAt); err != nil {
		return nil, err
	}
	e.StartedAt = parseTime(startedAt)
	e.CreatedAt = parseTime(createdAt)
	if completedAt != nil {
		t := parseTime(*completedAt)
		e.CompletedAt = &t
	}
	_ = json.Unmarshal([]byte(triggerDataJSON), &e.TriggerData)
	return e, nil
}

func scanExecutionRows(rows *sql.Rows) (*model.Execution, error) {
	e := &model.Execution{}
	var triggerDataJSON string
	var completedAt *string
	var startedAt, createdAt string
	if err := rows.Scan(
		&e.ID, &e.WorkflowID, &e.WorkspaceID, &e.Status, &triggerDataJSON, &e.StepsRun, &e.StepsTotal,
		&e.ErrorMessage, &startedAt, &completedAt, &createdAt); err != nil {
		return nil, err
	}
	e.StartedAt = parseTime(startedAt)
	e.CreatedAt = parseTime(createdAt)
	if completedAt != nil {
		t := parseTime(*completedAt)
		e.CompletedAt = &t
	}
	_ = json.Unmarshal([]byte(triggerDataJSON), &e.TriggerData)
	return e, nil
}

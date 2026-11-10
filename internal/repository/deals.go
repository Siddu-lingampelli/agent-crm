// Package repository - deals.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"crm.local/internal/model"
)

// DealListOptions extends ListOptions with deal filters.
type DealListOptions struct {
	Category       string
	CloseFrom      string
	CloseBefore    string
	InactiveBefore string
	WorkspaceID    string
	Query          string
	PipelineID     string
	StageID        string
	CompanyID      string
	OwnerID        string
	CreatedFrom    string
	Limit          int
	Offset         int
	Sort           string
	Order          string
}

const dealsColumns = `
    id, workspace_id, name, pipeline_id, stage_id, amount, currency, probability,
    COALESCE(expected_close,'') AS expected_close, company_id, primary_contact_id, lead_id, owner_id, source,
    description, lost_reason, won_at, lost_at, custom_fields, created_by,
    updated_by, created_at, updated_at, archived_at
`

// DealRepository persists deals.
type DealRepository struct {
	db *sql.DB
}

// NewDealRepository returns a repository over db.
func NewDealRepository(db *sql.DB) *DealRepository { return &DealRepository{db: db} }

// Create inserts a deal.
func (r *DealRepository) Create(ctx context.Context, d *model.Deal) error {
	if err := validateEntityReferences(ctx, r.db, d); err != nil {
		return err
	}
	custom := jsonOrDefaultMap(d.CustomFields)
	_, err := r.db.ExecContext(ctx, `
	    INSERT INTO deals (
	        id, workspace_id, name, pipeline_id, stage_id, amount, currency, probability,
	        expected_close, company_id, primary_contact_id, lead_id, owner_id, source,
	        description, lost_reason, won_at, lost_at, custom_fields, created_by,
	        updated_by, created_at, updated_at, archived_at
	    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.WorkspaceID, d.Name, d.PipelineID, d.StageID, d.Amount, d.Currency,
		d.Probability, d.ExpectedClose, nullIfEmpty(d.CompanyID), nullIfEmpty(d.PrimaryContactID), nullIfEmpty(d.LeadID),
		nullIfEmpty(d.OwnerID), d.Source, d.Description, d.LostReason, d.WonAt, d.LostAt,
		custom, d.CreatedBy, d.UpdatedBy, d.CreatedAt, d.UpdatedAt, d.ArchivedAt)
	if err != nil {
		return fmt.Errorf("insert deal: %w", err)
	}
	return nil
}

// Get returns a deal by id.
func (r *DealRepository) Get(ctx context.Context, id string) (*model.Deal, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+dealsColumns+" FROM deals WHERE id = ? AND archived_at IS NULL", id)
	d, err := scanDeal(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// List returns a page of deals.
func (r *DealRepository) List(ctx context.Context, opts DealListOptions) (Page[model.Deal], error) {
	where, args := dealFilters(opts)

	var total int
	countSQL := "SELECT COUNT(*) FROM deals" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Deal]{}, fmt.Errorf("count deals: %w", err)
	}

	sort := dealSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	sqlText := "SELECT " + dealsColumns + " FROM deals" + where +
		" ORDER BY " + sort + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Deal]{}, fmt.Errorf("list deals: %w", err)
	}
	defer rows.Close()

	items := []model.Deal{}
	for rows.Next() {
		item, err := scanDealRows(rows)
		if err != nil {
			return Page[model.Deal]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Deal]{}, fmt.Errorf("iterate deals: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies a deal.
func (r *DealRepository) Update(ctx context.Context, d *model.Deal) error {
	if err := validateEntityReferences(ctx, r.db, d); err != nil {
		return err
	}
	custom := jsonOrDefaultMap(d.CustomFields)
	_, err := r.db.ExecContext(ctx, `
	    UPDATE deals SET
	        name = ?, pipeline_id = ?, stage_id = ?, amount = ?, currency = ?, probability = ?,
	        expected_close = ?, company_id = ?, primary_contact_id = ?, lead_id = ?,
	        owner_id = ?, source = ?, description = ?, lost_reason = ?, won_at = ?, lost_at = ?,
	        custom_fields = ?, updated_by = ?, updated_at = ?, archived_at = ?
	    WHERE id = ? AND archived_at IS NULL`,
		d.Name, d.PipelineID, d.StageID, d.Amount, d.Currency, d.Probability,
		d.ExpectedClose, nullIfEmpty(d.CompanyID), nullIfEmpty(d.PrimaryContactID), nullIfEmpty(d.LeadID), nullIfEmpty(d.OwnerID),
		d.Source, d.Description, d.LostReason, d.WonAt, d.LostAt, custom,
		d.UpdatedBy, d.UpdatedAt, d.ArchivedAt, d.ID)
	if err != nil {
		return fmt.Errorf("update deal: %w", err)
	}
	return nil
}

// Archive soft-deletes a deal.
func (r *DealRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deals SET archived_at = ? WHERE id = ? AND archived_at IS NULL", now, id)
	if err != nil {
		return fmt.Errorf("archive deal: %w", err)
	}
	return nil
}

// Restore un-archives a deal.
func (r *DealRepository) Restore(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deals SET archived_at = NULL WHERE id = ? AND archived_at IS NOT NULL", id)
	if err != nil {
		return fmt.Errorf("restore deal: %w", err)
	}
	return nil
}

// MoveStage changes the stage and probability atomically.
func (r *DealRepository) MoveStage(ctx context.Context, id, stageID string, probability int, now string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deals SET stage_id = ?, probability = ?, updated_at = ? WHERE id = ? AND archived_at IS NULL",
		stageID, probability, now, id)
	if err != nil {
		return fmt.Errorf("move deal stage: %w", err)
	}
	return nil
}

// PipelineSummary returns aggregate totals per stage for a pipeline.
type PipelineSummary struct {
	StageID     string `json:"stage_id"`
	StageName   string `json:"stage_name"`
	Count       int    `json:"count"`
	Total       int64  `json:"total"`
	Currency    string `json:"currency"`
	Probability int    `json:"probability"`
}

// Summary returns pipeline totals.
func (r *DealRepository) Summary(ctx context.Context, workspaceID, pipelineID string) ([]PipelineSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
	        SELECT d.stage_id, s.name, COUNT(*) as cnt, COALESCE(SUM(d.amount),0) as total, d.currency, s.probability
	        FROM deals d
	        JOIN pipeline_stages s ON s.id = d.stage_id
	        WHERE d.workspace_id = ? AND d.pipeline_id = ? AND d.archived_at IS NULL
	        GROUP BY d.stage_id, d.currency
	        ORDER BY s.position ASC`, workspaceID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline summary: %w", err)
	}
	defer rows.Close()

	out := []PipelineSummary{}
	for rows.Next() {
		var s PipelineSummary
		if err := rows.Scan(&s.StageID, &s.StageName, &s.Count, &s.Total, &s.Currency, &s.Probability); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListDefaultPipeline returns the default pipeline for a workspace.
func (r *DealRepository) ListDefaultPipeline(ctx context.Context, workspaceID string) ([]model.Pipeline, error) {
	rows, err := r.db.QueryContext(ctx, `
	        SELECT id, workspace_id, name, is_default, created_at, updated_at
	        FROM pipelines
	        WHERE workspace_id = ? AND is_default = 1`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list default pipeline: %w", err)
	}
	defer rows.Close()

	out := []model.Pipeline{}
	for rows.Next() {
		var p model.Pipeline
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.IsDefault, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt = parseTime(createdAt)
		p.UpdatedAt = parseTime(updatedAt)
		out = append(out, p)
	}
	return out, rows.Err()
}

func dealFilters(opts DealListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	conditions = append(conditions, "archived_at IS NULL")

	if opts.Query != "" {
		conditions = append(conditions, "(name LIKE ?)")
		args = append(args, "%"+strings.TrimSpace(opts.Query)+"%")
	}
	if opts.PipelineID != "" {
		conditions = append(conditions, "pipeline_id = ?")
		args = append(args, opts.PipelineID)
	}
	if opts.StageID != "" {
		conditions = append(conditions, "stage_id = ?")
		args = append(args, opts.StageID)
	}
	if opts.CompanyID != "" {
		conditions = append(conditions, "company_id = ?")
		args = append(args, opts.CompanyID)
	}
	if opts.OwnerID != "" {
		conditions = append(conditions, "owner_id = ?")
		args = append(args, opts.OwnerID)
	}
	if opts.Category != "" {
		conditions = append(conditions, "stage_id IN (SELECT id FROM pipeline_stages WHERE category=?)")
		args = append(args, opts.Category)
	}
	if opts.CloseFrom != "" {
		conditions = append(conditions, "expected_close!='' AND substr(expected_close,1,10)>=?")
		args = append(args, opts.CloseFrom)
	}
	if opts.CloseBefore != "" {
		conditions = append(conditions, "expected_close!='' AND substr(expected_close,1,10)<?")
		args = append(args, opts.CloseBefore)
	}
	if opts.InactiveBefore != "" {
		conditions = append(conditions, sqliteInstant("updated_at")+"<julianday(?)")
		args = append(args, opts.InactiveBefore)
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

func dealSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return sqliteInstant("updated_at")
	case "name":
		return "name"
	case "amount":
		return "amount"
	case "probability":
		return "probability"
	case "expected_close":
		return "CASE WHEN expected_close IS NULL OR expected_close = '' THEN 1 ELSE 0 END, expected_close"
	default:
		return "created_at"
	}
}

func scanDeal(row *sql.Row) (*model.Deal, error) {
	d := &model.Deal{}
	var customJSON string
	var createdAt, updatedAt string
	var wonAt, lostAt, archivedAt *string
	var companyID, primaryContactID, leadID, ownerID sql.NullString
	if err := row.Scan(
		&d.ID, &d.WorkspaceID, &d.Name, &d.PipelineID, &d.StageID, &d.Amount, &d.Currency,
		&d.Probability, &d.ExpectedClose, &companyID, &primaryContactID, &leadID,
		&ownerID, &d.Source, &d.Description, &d.LostReason, &wonAt, &lostAt,
		&customJSON, &d.CreatedBy, &d.UpdatedBy, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	d.CompanyID = companyID.String
	d.PrimaryContactID = primaryContactID.String
	d.LeadID = leadID.String
	d.OwnerID = ownerID.String
	d.CreatedAt = parseTime(createdAt)
	d.UpdatedAt = parseTime(updatedAt)
	if wonAt != nil {
		t := parseTime(*wonAt)
		d.WonAt = &t
	}
	if lostAt != nil {
		t := parseTime(*lostAt)
		d.LostAt = &t
	}
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		d.ArchivedAt = &t
	}
	d.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &d.CustomFields)
	return d, nil
}

func scanDealRows(rows *sql.Rows) (*model.Deal, error) {
	d := &model.Deal{}
	var customJSON string
	var createdAt, updatedAt string
	var wonAt, lostAt, archivedAt *string
	var companyID, primaryContactID, leadID, ownerID sql.NullString
	if err := rows.Scan(
		&d.ID, &d.WorkspaceID, &d.Name, &d.PipelineID, &d.StageID, &d.Amount, &d.Currency,
		&d.Probability, &d.ExpectedClose, &companyID, &primaryContactID, &leadID,
		&ownerID, &d.Source, &d.Description, &d.LostReason, &wonAt, &lostAt,
		&customJSON, &d.CreatedBy, &d.UpdatedBy, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}
	d.CompanyID = companyID.String
	d.PrimaryContactID = primaryContactID.String
	d.LeadID = leadID.String
	d.OwnerID = ownerID.String
	d.CreatedAt = parseTime(createdAt)
	d.UpdatedAt = parseTime(updatedAt)
	if wonAt != nil {
		t := parseTime(*wonAt)
		d.WonAt = &t
	}
	if lostAt != nil {
		t := parseTime(*lostAt)
		d.LostAt = &t
	}
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		d.ArchivedAt = &t
	}
	d.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &d.CustomFields)
	return d, nil
}

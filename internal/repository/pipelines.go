// Package repository - pipelines and stages.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/uuid"
)

const pipelineColumns = "id, workspace_id, name, is_default, created_at, updated_at"
const stageColumns = "id, pipeline_id, name, position, probability, category, created_at"

// PipelineRepository persists pipelines.
type PipelineRepository struct {
	db *sql.DB
}

// NewPipelineRepository returns a repository over db.
func NewPipelineRepository(db *sql.DB) *PipelineRepository {
	return &PipelineRepository{db: db}
}

// Create inserts a pipeline with its stages in one transaction.
func (r *PipelineRepository) Create(ctx context.Context, p *model.Pipeline, stages []model.PipelineStage) error {
	return r.withTx(ctx, func(tx *sql.Tx) error {
		if p.ID == "" {
			p.ID = uuid.New()
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO pipelines (id, workspace_id, name, is_default, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?)`,
			p.ID, p.WorkspaceID, p.Name, p.IsDefault, p.CreatedAt, p.UpdatedAt); err != nil {
			return fmt.Errorf("insert pipeline: %w", err)
		}
		for i := range stages {
			s := stages[i]
			if s.ID == "" {
				s.ID = uuid.New()
			}
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO pipeline_stages (id, pipeline_id, name, position, probability, category, created_at)
                VALUES (?, ?, ?, ?, ?, ?, ?)`,
				s.ID, p.ID, s.Name, s.Position, s.Probability, s.Category, s.CreatedAt); err != nil {
				return fmt.Errorf("insert stage: %w", err)
			}
		}
		return nil
	})
}

// Get returns a pipeline by id with its stages.
func (r *PipelineRepository) Get(ctx context.Context, id string) (*model.Pipeline, []model.PipelineStage, error) {
	var p model.Pipeline
	var createdAt, updatedAt string
	var isDef int
	err := r.db.QueryRowContext(ctx, "SELECT "+pipelineColumns+" FROM pipelines WHERE id = ?", id).
		Scan(&p.ID, &p.WorkspaceID, &p.Name, &isDef, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get pipeline: %w", err)
	}
	p.IsDefault = isDef == 1
	p.CreatedAt = parseTime(createdAt)
	p.UpdatedAt = parseTime(updatedAt)

	stages, err := r.ListStages(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return &p, stages, nil
}

// ListStages returns stages for a pipeline ordered by position.
func (r *PipelineRepository) ListStages(ctx context.Context, pipelineID string) ([]model.PipelineStage, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+stageColumns+" FROM pipeline_stages WHERE pipeline_id = ? ORDER BY position ASC", pipelineID)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	defer rows.Close()

	stages := []model.PipelineStage{}
	for rows.Next() {
		var s model.PipelineStage
		var createdAt string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Name, &s.Position, &s.Probability, &s.Category, &createdAt); err != nil {
			return nil, err
		}
		s.CreatedAt = parseTime(createdAt)
		stages = append(stages, s)
	}
	return stages, rows.Err()
}

// List returns all pipelines in the workspace.
func (r *PipelineRepository) List(ctx context.Context, workspaceID string) ([]model.Pipeline, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+pipelineColumns+" FROM pipelines WHERE workspace_id = ? ORDER BY name ASC", workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := []model.Pipeline{}
	for rows.Next() {
		var p model.Pipeline
		var createdAt, updatedAt string
		var isDef int
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &isDef, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.IsDefault = isDef == 1
		p.CreatedAt = parseTime(createdAt)
		p.UpdatedAt = parseTime(updatedAt)
		pipelines = append(pipelines, p)
	}
	return pipelines, rows.Err()
}

// Default returns the default pipeline for a workspace, creating one if none exists.
func (r *PipelineRepository) Default(ctx context.Context, workspaceID string) (*model.Pipeline, []model.PipelineStage, error) {
	var id string
	var isDef int
	err := r.db.QueryRowContext(ctx,
		"SELECT id, is_default FROM pipelines WHERE workspace_id = ? ORDER BY is_default DESC LIMIT 1", workspaceID).
		Scan(&id, &isDef)
	if err == sql.ErrNoRows {
		now := time.Now().UTC()
		p := &model.Pipeline{
			ID: uuid.New(), WorkspaceID: workspaceID, Name: "Sales", IsDefault: true,
			CreatedAt: now, UpdatedAt: now,
		}
		stages := []model.PipelineStage{
			{ID: uuid.New(), Name: "Discovery", Position: 1, Probability: 20, Category: "open", CreatedAt: now},
			{ID: uuid.New(), Name: "Proposal", Position: 2, Probability: 50, Category: "open", CreatedAt: now},
			{ID: uuid.New(), Name: "Negotiation", Position: 3, Probability: 75, Category: "open", CreatedAt: now},
			{ID: uuid.New(), Name: "Won", Position: 4, Probability: 100, Category: "won", CreatedAt: now},
			{ID: uuid.New(), Name: "Lost", Position: 5, Probability: 0, Category: "lost", CreatedAt: now},
		}
		if err := r.Create(ctx, p, stages); err != nil {
			return nil, nil, err
		}
		return p, stages, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("find default pipeline: %w", err)
	}
	return r.Get(ctx, id)
}

// withTx runs fn in a transaction.
func (r *PipelineRepository) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

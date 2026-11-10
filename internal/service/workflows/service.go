// Package workflows implements workflow automation engine.
package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"crm.local/internal/jobs"
	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/service"
	"crm.local/internal/uuid"
)

// Service provides workflow business logic.
type Service struct {
	repo    *repository.WorkflowRepository
	log     *slog.Logger
	jobPool *jobs.Pool
}

// New returns a workflow service.
func New(repo *repository.WorkflowRepository, log *slog.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

// SetJobPool sets the job pool for async execution.
func (s *Service) SetJobPool(pool *jobs.Pool) {
	s.jobPool = pool
}

// CreateInput is the input for creating a workflow.
type CreateInput struct {
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	TriggerType   model.TriggerType    `json:"trigger_type"`
	TriggerConfig map[string]any       `json:"trigger_config"`
	Steps         []model.WorkflowStep `json:"steps"`
	Status        model.WorkflowStatus `json:"status"`
}

// Create creates a workflow.
func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Workflow, error) {
	if err := permissions.Authorize(actor, "workflows:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Status == "" {
		in.Status = model.WorkflowDraft
	}

	now := time.Now().UTC()
	w := &model.Workflow{
		ID:            uuid.New(),
		WorkspaceID:   actor.WorkspaceID,
		Name:          in.Name,
		Description:   in.Description,
		TriggerType:   in.TriggerType,
		TriggerConfig: in.TriggerConfig,
		Steps:         in.Steps,
		Status:        in.Status,
		CreatedBy:     actor.ActorID,
		UpdatedBy:     actor.ActorID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}

	s.log.Info("workflow created", "id", w.ID, "actor", actor.ActorID)
	return w, nil
}

// Get returns a workflow by ID.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Workflow, error) {
	if err := permissions.Authorize(actor, "workflows:read"); err != nil {
		return nil, err
	}
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil || w.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return w, nil
}

// ListInput holds pagination and filter params.
type ListInput struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	Status      string `json:"status"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// List returns a page of workflows.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Workflow], error) {
	if err := permissions.Authorize(actor, "workflows:read"); err != nil {
		return repository.Page[model.Workflow]{}, err
	}
	return s.repo.List(ctx, repository.WorkflowListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		Status:      in.Status,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	Name          *string               `json:"name,omitempty"`
	Description   *string               `json:"description,omitempty"`
	TriggerType   *model.TriggerType    `json:"trigger_type,omitempty"`
	TriggerConfig map[string]any        `json:"trigger_config,omitempty"`
	Steps         []model.WorkflowStep  `json:"steps,omitempty"`
	Status        *model.WorkflowStatus `json:"status,omitempty"`
}

// Update modifies a workflow.
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Workflow, error) {
	if err := permissions.Authorize(actor, "workflows:write"); err != nil {
		return nil, err
	}
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil || w.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}

	applyUpdate(w, in)
	w.UpdatedAt = time.Now().UTC()
	w.UpdatedBy = actor.ActorID

	if err := s.repo.Update(ctx, w); err != nil {
		return nil, fmt.Errorf("update workflow: %w", err)
	}

	s.log.Info("workflow updated", "id", w.ID, "actor", actor.ActorID)
	return w, nil
}

// Archive soft-deletes a workflow.
func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "workflows:write"); err != nil {
		return err
	}
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if w == nil || w.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.repo.Archive(ctx, id, now); err != nil {
		return fmt.Errorf("archive workflow: %w", err)
	}

	s.log.Info("workflow archived", "id", id, "actor", actor.ActorID)
	return nil
}

// Execute triggers a workflow execution.
func (s *Service) Execute(ctx context.Context, actor permissions.AuthContext, id string, triggerData map[string]any) (*model.Execution, error) {
	if err := permissions.Authorize(actor, "workflows:execute"); err != nil {
		return nil, err
	}
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil || w.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if w.Status != model.WorkflowActive {
		return nil, fmt.Errorf("workflow is not active")
	}

	// Create execution record
	exec := &model.Execution{
		ID:          uuid.New(),
		WorkflowID:  w.ID,
		WorkspaceID: actor.WorkspaceID,
		Status:      "running",
		TriggerData: triggerData,
		StepsTotal:  len(w.Steps),
		StartedAt:   time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}

	// Workers own a copy; returning the same pointer would race JSON encoding.
	work := *exec
	if s.jobPool == nil {
		return nil, fmt.Errorf("workflow worker pool unavailable")
	}
	if err := s.jobPool.Enqueue(ctx, "workflow_execution", func(jobCtx context.Context) { s.runWorkflow(jobCtx, &work, w) }); err != nil {
		exec.Status = "failed"
		exec.ErrorMessage = err.Error()
		now := time.Now().UTC()
		exec.CompletedAt = &now
		_ = s.repo.UpdateExecution(ctx, exec)
		return nil, err
	}

	s.log.Info("workflow execution started", "execution_id", exec.ID, "workflow_id", w.ID, "actor", actor.ActorID)
	return exec, nil
}

// runWorkflow executes the workflow steps.
func (s *Service) runWorkflow(ctx context.Context, exec *model.Execution, w *model.Workflow) {
	for i, step := range w.Steps {
		// Check if cancelled
		if exec.Status == "cancelled" {
			return
		}

		exec.StepsRun = i
		if err := s.repo.UpdateExecution(ctx, exec); err != nil {
			s.log.Error("failed to update execution progress", "error", err)
		}

		if err := s.executeStep(ctx, exec, w, step); err != nil {
			s.log.Error("workflow step failed", "execution_id", exec.ID, "step", step.Name, "error", err)
			exec.Status = "failed"
			exec.ErrorMessage = err.Error()
			now := time.Now().UTC()
			exec.CompletedAt = &now
			if err := s.repo.UpdateExecution(ctx, exec); err != nil {
				s.log.Error("failed to update failed execution", "error", err)
			}
			s.updateWorkflowStats(ctx, w, false)
			return
		}
	}

	// Success
	exec.Status = "completed"
	exec.StepsRun = len(w.Steps)
	now := time.Now().UTC()
	exec.CompletedAt = &now
	if err := s.repo.UpdateExecution(ctx, exec); err != nil {
		s.log.Error("failed to update completed execution", "error", err)
	}
	s.updateWorkflowStats(ctx, w, true)
	s.log.Info("workflow execution completed", "execution_id", exec.ID)
}

// executeStep executes a single workflow step.
func (s *Service) executeStep(ctx context.Context, exec *model.Execution, w *model.Workflow, step model.WorkflowStep) error {
	s.log.Debug("executing workflow step", "execution_id", exec.ID, "step", step.Name, "type", step.Type)

	switch step.Type {
	case model.StepAction:
		return s.executeAction(ctx, exec, w, step)
	case model.StepCondition:
		return s.executeCondition(ctx, exec, w, step)
	case model.StepDelay:
		return s.executeDelay(ctx, exec, w, step)
	case model.StepWebhook:
		return s.executeWebhook(ctx, exec, w, step)
	case model.StepMCPCall:
		return s.executeMCPCall(ctx, exec, w, step)
	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}
}

func (s *Service) executeAction(ctx context.Context, exec *model.Execution, w *model.Workflow, step model.WorkflowStep) error {
	action := step.Config["action"]
	if action == nil {
		return fmt.Errorf("action not specified")
	}

	switch action {
	case "create_task":
		return s.actionCreateTask(ctx, exec, step)
	case "assign_owner":
		return s.actionAssignOwner(ctx, exec, step)
	case "add_note":
		return s.actionAddNote(ctx, exec, step)
	case "update_field":
		return s.actionUpdateField(ctx, exec, step)
	case "create_email_draft":
		return s.actionCreateEmailDraft(ctx, exec, step)
	case "notify":
		return s.actionNotify(ctx, exec, step)
	default:
		return fmt.Errorf("unknown action: %v", action)
	}
}

func (s *Service) actionCreateTask(ctx context.Context, exec *model.Execution, step model.WorkflowStep) error {
	s.log.Info("action: create_task", "execution_id", exec.ID)
	// Implementation would use task service
	return service.ErrNotConfigured
}

func (s *Service) actionAssignOwner(ctx context.Context, exec *model.Execution, step model.WorkflowStep) error {
	s.log.Info("action: assign_owner", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) actionAddNote(ctx context.Context, exec *model.Execution, step model.WorkflowStep) error {
	s.log.Info("action: add_note", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) actionUpdateField(ctx context.Context, exec *model.Execution, step model.WorkflowStep) error {
	s.log.Info("action: update_field", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) actionCreateEmailDraft(ctx context.Context, exec *model.Execution, step model.WorkflowStep) error {
	s.log.Info("action: create_email_draft", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) actionNotify(ctx context.Context, exec *model.Execution, step model.WorkflowStep) error {
	s.log.Info("action: notify", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) executeCondition(ctx context.Context, exec *model.Execution, w *model.Workflow, step model.WorkflowStep) error {
	// Evaluate condition, skip subsequent steps if false
	s.log.Info("condition step", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) executeDelay(ctx context.Context, exec *model.Execution, w *model.Workflow, step model.WorkflowStep) error {
	durationStr, ok := step.Config["duration"].(string)
	if !ok {
		return fmt.Errorf("delay duration not specified")
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	s.log.Info("delay step", "execution_id", exec.ID, "duration", duration)
	if duration < 0 || duration > 24*time.Hour {
		return fmt.Errorf("delay duration must be between zero and 24 hours")
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) executeWebhook(ctx context.Context, exec *model.Execution, w *model.Workflow, step model.WorkflowStep) error {
	s.log.Info("webhook step", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) executeMCPCall(ctx context.Context, exec *model.Execution, w *model.Workflow, step model.WorkflowStep) error {
	s.log.Info("mcp_call step", "execution_id", exec.ID)
	return service.ErrNotConfigured
}

func (s *Service) updateWorkflowStats(ctx context.Context, w *model.Workflow, success bool) {
	if err := s.repo.RecordResult(ctx, w.ID, success, time.Now().UTC()); err != nil {
		s.log.Error("failed to update workflow stats", "error", err)
	}
}

// GetExecution returns a workflow execution by ID.
func (s *Service) GetExecution(ctx context.Context, actor permissions.AuthContext, id string) (*model.Execution, error) {
	if err := permissions.Authorize(actor, "workflows:read"); err != nil {
		return nil, err
	}
	exec, err := s.repo.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	if exec == nil || exec.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return exec, nil
}

// ListExecutions returns a page of workflow executions.
func (s *Service) ListExecutions(ctx context.Context, actor permissions.AuthContext, workflowID string, limit, offset int) ([]model.Execution, error) {
	if err := permissions.Authorize(actor, "workflows:read"); err != nil {
		return nil, err
	}
	return s.repo.ListExecutions(ctx, actor.WorkspaceID, workflowID, limit, offset)
}

func applyUpdate(w *model.Workflow, in UpdateInput) {
	if in.Name != nil {
		w.Name = *in.Name
	}
	if in.Description != nil {
		w.Description = *in.Description
	}
	if in.TriggerType != nil {
		w.TriggerType = *in.TriggerType
	}
	if in.TriggerConfig != nil {
		w.TriggerConfig = in.TriggerConfig
	}
	if in.Steps != nil {
		w.Steps = in.Steps
	}
	if in.Status != nil {
		w.Status = *in.Status
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

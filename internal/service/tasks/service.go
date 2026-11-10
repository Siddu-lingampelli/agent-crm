// Package tasks implements the task domain service.
package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/uuid"
)

// Service provides task business logic with authorization.
type Service struct {
	repo *repository.TaskRepository
	log  *slog.Logger
}

// New returns a task service.
func New(repo *repository.TaskRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

type CreateInput struct {
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	Priority     string         `json:"priority"`
	DueDate      string         `json:"due_date"`
	AssigneeID   string         `json:"assignee_id"`
	RelatedType  string         `json:"related_type"`
	RelatedID    string         `json:"related_id"`
	ProjectID    string         `json:"project_id"`
	CreatedBy    string         `json:"created_by"`
	CustomFields map[string]any `json:"custom_fields"`
}

type ListInput struct {
	ActiveOnly  bool   `json:"active_only"`
	DueAfter    string `json:"due_after"`
	Unscheduled bool   `json:"unscheduled"`
	DueBefore   string `json:"due_before"`
	DueOn       string `json:"due_on"`
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assignee_id"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
	ProjectID   string `json:"project_id"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

type UpdateInput struct {
	ID           string         `json:"id"`
	Title        *string        `json:"title,omitempty"`
	Description  *string        `json:"description,omitempty"`
	Status       *string        `json:"status,omitempty"`
	Priority     *string        `json:"priority,omitempty"`
	DueDate      *string        `json:"due_date,omitempty"`
	AssigneeID   *string        `json:"assignee_id,omitempty"`
	RelatedType  *string        `json:"related_type,omitempty"`
	RelatedID    *string        `json:"related_id,omitempty"`
	ProjectID    *string        `json:"project_id,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Task, error) {
	if err := permissions.Authorize(actor, "tasks:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("title required")
	}
	if in.Status == "" {
		in.Status = "open"
	}
	if in.Priority == "" {
		in.Priority = "normal"
	}
	c := &model.Task{
		ID:           uuid.New(),
		WorkspaceID:  actor.WorkspaceID,
		Title:        in.Title,
		Description:  in.Description,
		Status:       in.Status,
		Priority:     in.Priority,
		DueDate:      in.DueDate,
		AssigneeID:   in.AssigneeID,
		RelatedType:  in.RelatedType,
		RelatedID:    in.RelatedID,
		ProjectID:    in.ProjectID,
		CreatedBy:    actor.ActorID,
		CustomFields: in.CustomFields,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if c.Status == "completed" {
		completed := time.Now().UTC()
		c.CompletedAt = &completed
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	s.log.Info("task created", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Task, error) {
	if err := permissions.Authorize(actor, "tasks:read"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Task], error) {
	if err := permissions.Authorize(actor, "tasks:read"); err != nil {
		return repository.Page[model.Task]{}, err
	}
	return s.repo.List(ctx, repository.TaskListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		ActiveOnly:  in.ActiveOnly,
		DueAfter:    in.DueAfter,
		Unscheduled: in.Unscheduled,
		DueBefore:   in.DueBefore,
		DueOn:       in.DueOn,
		Status:      in.Status,
		Priority:    in.Priority,
		AssigneeID:  in.AssigneeID,
		RelatedType: in.RelatedType,
		RelatedID:   in.RelatedID,
		ProjectID:   in.ProjectID,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Task, error) {
	if err := permissions.Authorize(actor, "tasks:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	applyTaskUpdate(c, in)
	c.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	s.log.Info("task updated", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "tasks:write"); err != nil {
		return err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.repo.Archive(ctx, id, now); err != nil {
		return fmt.Errorf("archive task: %w", err)
	}
	s.log.Info("task archived", "id", id, "actor", actor.ActorID)
	return nil
}

func (s *Service) Complete(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "tasks:write"); err != nil {
		return err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	now := time.Now().UTC()
	c.Status = "completed"
	c.CompletedAt = &now
	c.UpdatedAt = now
	if err := s.repo.Update(ctx, c); err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	s.log.Info("task completed", "id", c.ID, "actor", actor.ActorID)
	return nil
}

func applyTaskUpdate(c *model.Task, in UpdateInput) {
	if in.Title != nil {
		c.Title = *in.Title
	}
	if in.Description != nil {
		c.Description = *in.Description
	}
	if in.Status != nil {
		c.Status = *in.Status
		if c.Status == "completed" {
			if c.CompletedAt == nil {
				completed := time.Now().UTC()
				c.CompletedAt = &completed
			}
		} else {
			c.CompletedAt = nil
		}
	}
	if in.Priority != nil {
		c.Priority = *in.Priority
	}
	if in.DueDate != nil {
		c.DueDate = *in.DueDate
	}
	if in.AssigneeID != nil {
		c.AssigneeID = *in.AssigneeID
	}
	if in.RelatedType != nil {
		c.RelatedType = *in.RelatedType
	}
	if in.RelatedID != nil {
		c.RelatedID = *in.RelatedID
	}
	if in.ProjectID != nil {
		c.ProjectID = *in.ProjectID
	}
	if in.CustomFields != nil {
		c.CustomFields = in.CustomFields
	}
}

// Package projects implements the project domain service.
package projects

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

// Service provides project business logic with authorization.
type Service struct {
	repo *repository.ProjectRepository
	log  *slog.Logger
}

// New returns a project service.
func New(repo *repository.ProjectRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

type CreateInput struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	Priority     string         `json:"priority"`
	OwnerID      string         `json:"owner_id"`
	CompanyID    string         `json:"company_id"`
	StartDate    string         `json:"start_date"`
	EndDate      string         `json:"end_date"`
	Progress     int            `json:"progress"`
	CustomFields map[string]any `json:"custom_fields"`
}

type ListInput struct {
	Priority    string `json:"priority"`
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	Status      string `json:"status"`
	OwnerID     string `json:"owner_id"`
	CompanyID   string `json:"company_id"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

type UpdateInput struct {
	ID           string         `json:"id"`
	Name         *string        `json:"name,omitempty"`
	Description  *string        `json:"description,omitempty"`
	Status       *string        `json:"status,omitempty"`
	Priority     *string        `json:"priority,omitempty"`
	OwnerID      *string        `json:"owner_id,omitempty"`
	CompanyID    *string        `json:"company_id,omitempty"`
	StartDate    *string        `json:"start_date,omitempty"`
	EndDate      *string        `json:"end_date,omitempty"`
	Progress     *int           `json:"progress,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Project, error) {
	if err := permissions.Authorize(actor, "projects:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Status == "" {
		in.Status = "planning"
	}
	if in.Priority == "" {
		in.Priority = "normal"
	}
	c := &model.Project{
		ID:           uuid.New(),
		WorkspaceID:  actor.WorkspaceID,
		Name:         in.Name,
		Description:  in.Description,
		Status:       in.Status,
		Priority:     in.Priority,
		OwnerID:      in.OwnerID,
		CompanyID:    in.CompanyID,
		StartDate:    in.StartDate,
		EndDate:      in.EndDate,
		Progress:     in.Progress,
		CustomFields: in.CustomFields,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	s.log.Info("project created", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Project, error) {
	if err := permissions.Authorize(actor, "projects:read"); err != nil {
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

func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Project], error) {
	if err := permissions.Authorize(actor, "projects:read"); err != nil {
		return repository.Page[model.Project]{}, err
	}
	return s.repo.List(ctx, repository.ProjectListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		Priority:    in.Priority,
		Status:      in.Status,
		OwnerID:     in.OwnerID,
		CompanyID:   in.CompanyID,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Project, error) {
	if err := permissions.Authorize(actor, "projects:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	applyProjectUpdate(c, in)
	c.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	s.log.Info("project updated", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "projects:write"); err != nil {
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
		return fmt.Errorf("archive project: %w", err)
	}
	s.log.Info("project archived", "id", id, "actor", actor.ActorID)
	return nil
}

func applyProjectUpdate(c *model.Project, in UpdateInput) {
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.Description != nil {
		c.Description = *in.Description
	}
	if in.Status != nil {
		c.Status = *in.Status
	}
	if in.Priority != nil {
		c.Priority = *in.Priority
	}
	if in.OwnerID != nil {
		c.OwnerID = *in.OwnerID
	}
	if in.CompanyID != nil {
		c.CompanyID = *in.CompanyID
	}
	if in.StartDate != nil {
		c.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		c.EndDate = *in.EndDate
	}
	if in.Progress != nil {
		c.Progress = *in.Progress
	}
	if in.CustomFields != nil {
		c.CustomFields = in.CustomFields
	}
}

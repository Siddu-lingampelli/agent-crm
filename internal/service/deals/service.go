// Package deals implements the deal domain service.
package deals

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

// Service provides deal business logic with authorization.
type Service struct {
	repo *repository.DealRepository
	log  *slog.Logger
}

// New returns a deal service.
func New(repo *repository.DealRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

type CreateInput struct {
	Name             string         `json:"name"`
	PipelineID       string         `json:"pipeline_id"`
	StageID          string         `json:"stage_id"`
	Amount           *int64         `json:"amount"`
	Currency         string         `json:"currency"`
	Probability      int            `json:"probability"`
	ExpectedClose    string         `json:"expected_close"`
	CompanyID        string         `json:"company_id"`
	PrimaryContactID string         `json:"primary_contact_id"`
	LeadID           string         `json:"lead_id"`
	OwnerID          string         `json:"owner_id"`
	Source           string         `json:"source"`
	Description      string         `json:"description"`
	LostReason       string         `json:"lost_reason"`
	CustomFields     map[string]any `json:"custom_fields"`
}

type ListInput struct {
	Category       string `json:"category"`
	CloseFrom      string `json:"close_from"`
	CloseBefore    string `json:"close_before"`
	InactiveBefore string `json:"inactive_before"`
	WorkspaceID    string `json:"workspace_id"`
	Query          string `json:"q"`
	PipelineID     string `json:"pipeline_id"`
	StageID        string `json:"stage_id"`
	OwnerID        string `json:"owner_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	Sort           string `json:"sort"`
	Order          string `json:"order"`
}

type UpdateInput struct {
	ID               string         `json:"id"`
	Name             *string        `json:"name,omitempty"`
	PipelineID       *string        `json:"pipeline_id,omitempty"`
	StageID          *string        `json:"stage_id,omitempty"`
	Amount           *int64         `json:"amount,omitempty"`
	Currency         *string        `json:"currency,omitempty"`
	Probability      *int           `json:"probability,omitempty"`
	ExpectedClose    *string        `json:"expected_close,omitempty"`
	CompanyID        *string        `json:"company_id,omitempty"`
	PrimaryContactID *string        `json:"primary_contact_id,omitempty"`
	LeadID           *string        `json:"lead_id,omitempty"`
	OwnerID          *string        `json:"owner_id,omitempty"`
	Source           *string        `json:"source,omitempty"`
	Description      *string        `json:"description,omitempty"`
	LostReason       *string        `json:"lost_reason,omitempty"`
	CustomFields     map[string]any `json:"custom_fields,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Deal, error) {
	if err := permissions.Authorize(actor, "deals:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	c := &model.Deal{
		ID:               uuid.New(),
		WorkspaceID:      actor.WorkspaceID,
		Name:             in.Name,
		PipelineID:       in.PipelineID,
		StageID:          in.StageID,
		Amount:           in.Amount,
		Currency:         in.Currency,
		Probability:      in.Probability,
		ExpectedClose:    in.ExpectedClose,
		CompanyID:        in.CompanyID,
		PrimaryContactID: in.PrimaryContactID,
		LeadID:           in.LeadID,
		OwnerID:          in.OwnerID,
		Source:           in.Source,
		Description:      in.Description,
		LostReason:       in.LostReason,
		CustomFields:     in.CustomFields,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		CreatedBy:        actor.ActorID,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create deal: %w", err)
	}
	s.log.Info("deal created", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Deal, error) {
	if err := permissions.Authorize(actor, "deals:read"); err != nil {
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

func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Deal], error) {
	if err := permissions.Authorize(actor, "deals:read"); err != nil {
		return repository.Page[model.Deal]{}, err
	}
	return s.repo.List(ctx, repository.DealListOptions{
		Category:       in.Category,
		CloseFrom:      in.CloseFrom,
		CloseBefore:    in.CloseBefore,
		InactiveBefore: in.InactiveBefore,
		WorkspaceID:    actor.WorkspaceID,
		Query:          in.Query,
		PipelineID:     in.PipelineID,
		StageID:        in.StageID,
		OwnerID:        in.OwnerID,
		Limit:          in.Limit,
		Offset:         in.Offset,
		Sort:           in.Sort,
		Order:          in.Order,
	})
}

func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Deal, error) {
	if err := permissions.Authorize(actor, "deals:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	applyDealUpdate(c, in)
	c.UpdatedAt = time.Now().UTC()
	c.UpdatedBy = actor.ActorID
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update deal: %w", err)
	}
	s.log.Info("deal updated", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "deals:write"); err != nil {
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
		return fmt.Errorf("archive deal: %w", err)
	}
	s.log.Info("deal archived", "id", id, "actor", actor.ActorID)
	return nil
}

func applyDealUpdate(c *model.Deal, in UpdateInput) {
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.PipelineID != nil {
		c.PipelineID = *in.PipelineID
	}
	if in.StageID != nil {
		c.StageID = *in.StageID
	}
	if in.Amount != nil {
		c.Amount = in.Amount
	}
	if in.Currency != nil {
		c.Currency = *in.Currency
	}
	if in.Probability != nil {
		c.Probability = *in.Probability
	}
	if in.ExpectedClose != nil {
		c.ExpectedClose = *in.ExpectedClose
	}
	if in.CompanyID != nil {
		c.CompanyID = *in.CompanyID
	}
	if in.PrimaryContactID != nil {
		c.PrimaryContactID = *in.PrimaryContactID
	}
	if in.LeadID != nil {
		c.LeadID = *in.LeadID
	}
	if in.OwnerID != nil {
		c.OwnerID = *in.OwnerID
	}
	if in.Source != nil {
		c.Source = *in.Source
	}
	if in.Description != nil {
		c.Description = *in.Description
	}
	if in.LostReason != nil {
		c.LostReason = *in.LostReason
	}
	if in.CustomFields != nil {
		c.CustomFields = in.CustomFields
	}
}

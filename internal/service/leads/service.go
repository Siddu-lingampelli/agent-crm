// Package leads implements the lead domain service.
package leads

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

// Service provides lead business logic with authorization.
type Service struct {
	repo *repository.LeadRepository
	log  *slog.Logger
}

// New returns a lead service.
func New(repo *repository.LeadRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

type CreateInput struct {
	DisplayName    string         `json:"display_name"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	CompanyName    string         `json:"company_name"`
	CompanyID      string         `json:"company_id"`
	JobTitle       string         `json:"job_title"`
	Status         string         `json:"status"`
	Source         string         `json:"source"`
	OwnerID        string         `json:"owner_id"`
	Priority       string         `json:"priority"`
	EstimatedValue *int64         `json:"estimated_value"`
	Currency       string         `json:"currency"`
	LeadScore      *int           `json:"lead_score"`
	ExpectedClose  string         `json:"expected_close"`
	FollowUpDate   string         `json:"follow_up_date"`
	Tags           []string       `json:"tags"`
	CustomFields   map[string]any `json:"custom_fields"`
	Description    string         `json:"description"`
	Qualification  string         `json:"qualification"`
}

type ListInput struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	OwnerID     string `json:"owner_id"`
	Priority    string `json:"priority"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

type UpdateInput struct {
	ID             string         `json:"id"`
	DisplayName    *string        `json:"display_name,omitempty"`
	FirstName      *string        `json:"first_name,omitempty"`
	LastName       *string        `json:"last_name,omitempty"`
	Email          *string        `json:"email,omitempty"`
	Phone          *string        `json:"phone,omitempty"`
	CompanyName    *string        `json:"company_name,omitempty"`
	CompanyID      *string        `json:"company_id,omitempty"`
	JobTitle       *string        `json:"job_title,omitempty"`
	Status         *string        `json:"status,omitempty"`
	Source         *string        `json:"source,omitempty"`
	OwnerID        *string        `json:"owner_id,omitempty"`
	Priority       *string        `json:"priority,omitempty"`
	EstimatedValue *int64         `json:"estimated_value,omitempty"`
	Currency       *string        `json:"currency,omitempty"`
	LeadScore      *int           `json:"lead_score,omitempty"`
	ExpectedClose  *string        `json:"expected_close,omitempty"`
	FollowUpDate   *string        `json:"follow_up_date,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	CustomFields   map[string]any `json:"custom_fields,omitempty"`
	Description    *string        `json:"description,omitempty"`
	Qualification  *string        `json:"qualification,omitempty"`
}

type ConvertInput struct {
	ContactID     string `json:"contact_id"`
	CompanyID     string `json:"company_id"`
	DealName      string `json:"deal_name"`
	PipelineID    string `json:"pipeline_id"`
	StageID       string `json:"stage_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	ExpectedClose string `json:"expected_close"`
}

func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Lead, error) {
	if err := permissions.Authorize(actor, "leads:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.DisplayName) == "" {
		return nil, fmt.Errorf("display name required")
	}
	if in.Status == "" {
		in.Status = "new"
	}
	if in.Priority == "" {
		in.Priority = "normal"
	}
	c := &model.Lead{
		ID:             uuid.New(),
		WorkspaceID:    actor.WorkspaceID,
		DisplayName:    in.DisplayName,
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		Email:          in.Email,
		Phone:          in.Phone,
		CompanyName:    in.CompanyName,
		CompanyID:      in.CompanyID,
		JobTitle:       in.JobTitle,
		Status:         in.Status,
		Source:         in.Source,
		OwnerID:        in.OwnerID,
		Priority:       in.Priority,
		EstimatedValue: in.EstimatedValue,
		Currency:       in.Currency,
		LeadScore:      in.LeadScore,
		ExpectedClose:  in.ExpectedClose,
		FollowUpDate:   in.FollowUpDate,
		Tags:           in.Tags,
		CustomFields:   in.CustomFields,
		Description:    in.Description,
		Qualification:  in.Qualification,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		CreatedBy:      actor.ActorID,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create lead: %w", err)
	}
	s.log.Info("lead created", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Lead, error) {
	if err := permissions.Authorize(actor, "leads:read"); err != nil {
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

func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Lead], error) {
	if err := permissions.Authorize(actor, "leads:read"); err != nil {
		return repository.Page[model.Lead]{}, err
	}
	return s.repo.List(ctx, repository.LeadListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		Status:      in.Status,
		Source:      in.Source,
		// OwnerID uses query param directly in repo via where clause
		OwnerID:  in.OwnerID,
		Priority: in.Priority,
		Limit:    in.Limit,
		Offset:   in.Offset,
		Sort:     in.Sort,
		Order:    in.Order,
	})
}

func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Lead, error) {
	if err := permissions.Authorize(actor, "leads:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	applyLeadUpdate(c, in)
	c.UpdatedAt = time.Now().UTC()
	c.UpdatedBy = actor.ActorID
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update lead: %w", err)
	}
	s.log.Info("lead updated", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "leads:write"); err != nil {
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
		return fmt.Errorf("archive lead: %w", err)
	}
	s.log.Info("lead archived", "id", id, "actor", actor.ActorID)
	return nil
}

func (s *Service) Convert(ctx context.Context, actor permissions.AuthContext, id string, in ConvertInput) (*model.Lead, error) {
	if err := permissions.Authorize(actor, "leads:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if c.Status == "converted" {
		return nil, fmt.Errorf("lead already converted")
	}
	if in.ContactID == "" {
		if err := permissions.Authorize(actor, "contacts:write"); err != nil {
			return nil, err
		}
	}
	if in.CompanyID == "" && c.CompanyID == "" && strings.TrimSpace(c.CompanyName) != "" {
		if err := permissions.Authorize(actor, "companies:write"); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(in.DealName) != "" {
		if err := permissions.Authorize(actor, "deals:write"); err != nil {
			return nil, err
		}
	}
	c, err = s.repo.ConvertWithRelations(ctx, actor.WorkspaceID, actor.ActorID, id, repository.LeadConversion{ContactID: in.ContactID, CompanyID: in.CompanyID, DealName: in.DealName, PipelineID: in.PipelineID, StageID: in.StageID, Amount: in.Amount, Currency: in.Currency, ExpectedClose: in.ExpectedClose})
	if err != nil {
		return nil, err
	}
	s.log.Info("lead converted", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

func applyLeadUpdate(c *model.Lead, in UpdateInput) {
	if in.DisplayName != nil {
		c.DisplayName = *in.DisplayName
	}
	if in.FirstName != nil {
		c.FirstName = *in.FirstName
	}
	if in.LastName != nil {
		c.LastName = *in.LastName
	}
	if in.Email != nil {
		c.Email = *in.Email
	}
	if in.Phone != nil {
		c.Phone = *in.Phone
	}
	if in.CompanyName != nil {
		c.CompanyName = *in.CompanyName
	}
	if in.CompanyID != nil {
		c.CompanyID = *in.CompanyID
	}
	if in.JobTitle != nil {
		c.JobTitle = *in.JobTitle
	}
	if in.Status != nil {
		c.Status = *in.Status
	}
	if in.Source != nil {
		c.Source = *in.Source
	}
	if in.OwnerID != nil {
		c.OwnerID = *in.OwnerID
	}
	if in.Priority != nil {
		c.Priority = *in.Priority
	}
	if in.EstimatedValue != nil {
		c.EstimatedValue = in.EstimatedValue
	}
	if in.Currency != nil {
		c.Currency = *in.Currency
	}
	if in.LeadScore != nil {
		c.LeadScore = in.LeadScore
	}
	if in.ExpectedClose != nil {
		c.ExpectedClose = *in.ExpectedClose
	}
	if in.FollowUpDate != nil {
		c.FollowUpDate = *in.FollowUpDate
	}
	if in.Tags != nil {
		c.Tags = in.Tags
	}
	if in.CustomFields != nil {
		c.CustomFields = in.CustomFields
	}
	if in.Description != nil {
		c.Description = *in.Description
	}
	if in.Qualification != nil {
		c.Qualification = *in.Qualification
	}
}

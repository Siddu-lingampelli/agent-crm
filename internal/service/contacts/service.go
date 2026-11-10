// Package contacts implements the contact domain service.
package contacts

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

// Service provides contact business logic with authorization.
type Service struct {
	repo *repository.ContactRepository
	log  *slog.Logger
}

// New returns a contact service.
func New(repo *repository.ContactRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// CreateInput is the input for creating a contact.
type CreateInput struct {
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	Email        string         `json:"email"`
	Phone        string         `json:"phone"`
	JobTitle     string         `json:"job_title"`
	CompanyID    string         `json:"company_id"`
	OwnerID      string         `json:"owner_id"`
	Status       string         `json:"status"`
	Tags         []string       `json:"tags"`
	CustomFields map[string]any `json:"custom_fields"`
	Notes        string         `json:"notes"`
}

// ListInput holds pagination and filter params.
type ListInput struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	CompanyID   string `json:"company_id"`
	Status      string `json:"status"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	ID           string         `json:"id"`
	FirstName    *string        `json:"first_name,omitempty"`
	LastName     *string        `json:"last_name,omitempty"`
	Email        *string        `json:"email,omitempty"`
	Phone        *string        `json:"phone,omitempty"`
	JobTitle     *string        `json:"job_title,omitempty"`
	CompanyID    *string        `json:"company_id,omitempty"`
	OwnerID      *string        `json:"owner_id,omitempty"`
	Status       *string        `json:"status,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
	Notes        *string        `json:"notes,omitempty"`
}

// Create creates a contact with authorization.
func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Contact, error) {
	if err := permissions.Authorize(actor, "contacts:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.FirstName) == "" {
		return nil, fmt.Errorf("first name required")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	c := &model.Contact{
		ID:           uuid.New(),
		WorkspaceID:  actor.WorkspaceID,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Email:        in.Email,
		Phone:        in.Phone,
		JobTitle:     in.JobTitle,
		CompanyID:    in.CompanyID,
		OwnerID:      in.OwnerID,
		Status:       in.Status,
		Tags:         in.Tags,
		CustomFields: in.CustomFields,
		Notes:        in.Notes,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}
	s.log.Info("contact created", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

// Get returns a contact by id.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Contact, error) {
	if err := permissions.Authorize(actor, "contacts:read"); err != nil {
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

// List returns a page of contacts.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Contact], error) {
	if err := permissions.Authorize(actor, "contacts:read"); err != nil {
		return repository.Page[model.Contact]{}, err
	}
	return s.repo.List(ctx, repository.ContactListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		CompanyID:   in.CompanyID,
		Status:      in.Status,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// Update modifies a contact.
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Contact, error) {
	if err := permissions.Authorize(actor, "contacts:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	applyContactUpdate(c, in)
	c.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update contact: %w", err)
	}
	s.log.Info("contact updated", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

// Archive soft-deletes a contact.
func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "contacts:write"); err != nil {
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
		return fmt.Errorf("archive contact: %w", err)
	}
	s.log.Info("contact archived", "id", id, "actor", actor.ActorID)
	return nil
}

func applyContactUpdate(c *model.Contact, in UpdateInput) {
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
	if in.JobTitle != nil {
		c.JobTitle = *in.JobTitle
	}
	if in.CompanyID != nil {
		c.CompanyID = *in.CompanyID
	}
	if in.OwnerID != nil {
		c.OwnerID = *in.OwnerID
	}
	if in.Status != nil {
		c.Status = *in.Status
	}
	if in.Tags != nil {
		c.Tags = in.Tags
	}
	if in.CustomFields != nil {
		c.CustomFields = in.CustomFields
	}
	if in.Notes != nil {
		c.Notes = *in.Notes
	}
}

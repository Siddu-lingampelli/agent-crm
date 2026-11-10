// Package companies implements the company domain service.
package companies

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

// Service provides company business logic with authorization and audit.
type Service struct {
	repo *repository.CompanyRepository
	log  *slog.Logger
}

// New returns a company service.
func New(repo *repository.CompanyRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// CreateInput is the input for creating a company.
type CreateInput struct {
	Name         string         `json:"name"`
	Domain       string         `json:"domain"`
	Industry     string         `json:"industry"`
	SizeBand     string         `json:"size_band"`
	Website      string         `json:"website"`
	Phone        string         `json:"phone"`
	Email        string         `json:"email"`
	Address      string         `json:"address"`
	IsClient     bool           `json:"is_client"`
	Status       string         `json:"status"`
	OwnerID      string         `json:"owner_id"`
	Tags         []string       `json:"tags"`
	CustomFields map[string]any `json:"custom_fields"`
	Notes        string         `json:"notes"`
}

// ListInput holds pagination and filter params.
type ListInput struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	IsClient    *bool  `json:"is_client"`
	Status      string `json:"status"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// Create creates a company with authorization and audit.
func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Company, error) {
	if err := permissions.Authorize(actor, "companies:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	c := &model.Company{
		ID:           uuid.New(),
		WorkspaceID:  actor.WorkspaceID,
		Name:         in.Name,
		Domain:       in.Domain,
		Industry:     in.Industry,
		SizeBand:     in.SizeBand,
		Website:      in.Website,
		Phone:        in.Phone,
		Email:        in.Email,
		Address:      in.Address,
		IsClient:     in.IsClient,
		Status:       in.Status,
		OwnerID:      in.OwnerID,
		Tags:         in.Tags,
		CustomFields: in.CustomFields,
		Notes:        in.Notes,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}
	s.log.Info("company created", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

// Get returns a company by id.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Company, error) {
	if err := permissions.Authorize(actor, "companies:read"); err != nil {
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

// List returns a page of companies.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Company], error) {
	if err := permissions.Authorize(actor, "companies:read"); err != nil {
		return repository.Page[model.Company]{}, err
	}
	return s.repo.List(ctx, repository.CompanyListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		IsClient:    in.IsClient,
		Status:      in.Status,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	ID           string         `json:"id"`
	Name         *string        `json:"name,omitempty"`
	Domain       *string        `json:"domain,omitempty"`
	Industry     *string        `json:"industry,omitempty"`
	SizeBand     *string        `json:"size_band,omitempty"`
	Website      *string        `json:"website,omitempty"`
	Phone        *string        `json:"phone,omitempty"`
	Email        *string        `json:"email,omitempty"`
	Address      *string        `json:"address,omitempty"`
	IsClient     *bool          `json:"is_client,omitempty"`
	Status       *string        `json:"status,omitempty"`
	OwnerID      *string        `json:"owner_id,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
	Notes        *string        `json:"notes,omitempty"`
}

// Update modifies a company.
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Company, error) {
	if err := permissions.Authorize(actor, "companies:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	applyUpdate(c, in)
	c.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update company: %w", err)
	}
	s.log.Info("company updated", "id", c.ID, "actor", actor.ActorID)
	return c, nil
}

// Archive soft-deletes a company.
func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "companies:write"); err != nil {
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
		return fmt.Errorf("archive company: %w", err)
	}
	s.log.Info("company archived", "id", id, "actor", actor.ActorID)
	return nil
}

func applyUpdate(c *model.Company, in UpdateInput) {
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.Domain != nil {
		c.Domain = *in.Domain
	}
	if in.Industry != nil {
		c.Industry = *in.Industry
	}
	if in.SizeBand != nil {
		c.SizeBand = *in.SizeBand
	}
	if in.Website != nil {
		c.Website = *in.Website
	}
	if in.Phone != nil {
		c.Phone = *in.Phone
	}
	if in.Email != nil {
		c.Email = *in.Email
	}
	if in.Address != nil {
		c.Address = *in.Address
	}
	if in.IsClient != nil {
		c.IsClient = *in.IsClient
	}
	if in.Status != nil {
		c.Status = *in.Status
	}
	if in.OwnerID != nil {
		c.OwnerID = *in.OwnerID
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

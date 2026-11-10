// Package integrations implements third-party integrations management.
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/security"
	"crm.local/internal/service"
	"crm.local/internal/uuid"
)

// Service provides integrations business logic.
type Service struct {
	repo *repository.IntegrationRepository
	log  *slog.Logger
}

// New returns an integrations service.
func New(repo *repository.IntegrationRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// CreateInput is the input for creating an integration.
type CreateInput struct {
	Name     string         `json:"name"`
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
	Enabled  bool           `json:"enabled"`
}

// Create creates an integration.
func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Integration, error) {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if strings.TrimSpace(in.Provider) == "" {
		return nil, fmt.Errorf("provider required")
	}

	now := time.Now().UTC()
	configJSON, _ := json.Marshal(in.Config)

	integration := &model.Integration{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		Name:        in.Name,
		Provider:    in.Provider,
		Config:      string(configJSON),
		Enabled:     in.Enabled,
		Status:      "disconnected",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, integration); err != nil {
		return nil, fmt.Errorf("create integration: %w", err)
	}

	s.log.Info("integration created", "id", integration.ID, "actor", actor.ActorID)
	return integration, nil
}

// Get returns an integration by ID.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Integration, error) {
	if err := permissions.Authorize(actor, "integrations:read"); err != nil {
		return nil, err
	}
	integration, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if integration == nil || integration.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return integration, nil
}

// ListInput holds pagination and filter params.
type ListInput struct {
	Query       string `json:"q"`
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider"`
	Status      string `json:"status"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// List returns a page of integrations.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) ([]model.Integration, error) {
	if err := permissions.Authorize(actor, "integrations:read"); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, repository.IntegrationListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		Provider:    in.Provider,
		Status:      in.Status,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	Name    *string        `json:"name,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
	Enabled *bool          `json:"enabled,omitempty"`
	Status  *string        `json:"status,omitempty"`
}

// Update modifies an integration.
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Integration, error) {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return nil, err
	}
	integration, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if integration == nil || integration.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}

	if in.Name != nil {
		integration.Name = *in.Name
	}
	if in.Config != nil {
		configJSON, _ := json.Marshal(in.Config)
		integration.Config = string(configJSON)
	}
	if in.Enabled != nil {
		integration.Enabled = *in.Enabled
	}
	if in.Status != nil {
		integration.Status = *in.Status
	}

	integration.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, integration); err != nil {
		return nil, fmt.Errorf("update integration: %w", err)
	}

	s.log.Info("integration updated", "id", integration.ID, "actor", actor.ActorID)
	return integration, nil
}

// Archive soft-deletes an integration.
func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return err
	}
	integration, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if integration == nil || integration.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.repo.Archive(ctx, id, now); err != nil {
		return fmt.Errorf("archive integration: %w", err)
	}

	s.log.Info("integration archived", "id", id, "actor", actor.ActorID)
	return nil
}

// UpdateCredentials updates encrypted credentials for an integration.
func (s *Service) UpdateCredentials(ctx context.Context, actor permissions.AuthContext, id string, credentials string) error {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return err
	}
	integration, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if integration == nil || integration.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	sealed, err := security.SealCredentials(credentials, os.Getenv("CRM_ENCRYPTION_KEY"))
	if err != nil {
		return err
	}
	return s.repo.UpdateCredentials(ctx, id, sealed)
}

// UpdateLastSync updates the last sync time and error.
func (s *Service) UpdateLastSync(ctx context.Context, id string, lastSyncAt string, lastError string) error {
	return s.repo.UpdateLastSync(ctx, id, lastSyncAt, lastError)
}

// Sync triggers a sync for an integration.
func (s *Service) Sync(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return err
	}
	integration, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if integration == nil || integration.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	// Validate integration config URLs for SSRF protection
	var config map[string]any
	if err := json.Unmarshal([]byte(integration.Config), &config); err == nil {
		if urls, ok := config["urls"].([]any); ok {
			secConfig := security.SecurityConfig{
				SSRFBlockPrivateIPs: true,
				SSRFAllowedCIDRs:    []string{},
			}
			for _, u := range urls {
				if urlStr, ok := u.(string); ok {
					if err := security.ValidateSSRFURL(urlStr, secConfig); err != nil {
						s.log.Warn("integration sync URL blocked by SSRF protection", "id", id, "url", urlStr, "error", err)
						return fmt.Errorf("integration URL blocked by SSRF protection: %w", err)
					}
				}
			}
		}
	}

	// Integration sync logic would go here
	s.log.Info("integration sync triggered", "id", id, "actor", actor.ActorID)
	return service.ErrNotConfigured
}
func (s *Service) ListPage(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Integration], error) {
	items, err := s.List(ctx, actor, in)
	if err != nil {
		return repository.Page[model.Integration]{}, err
	}
	total, err := s.repo.Count(ctx, repository.IntegrationListOptions{WorkspaceID: actor.WorkspaceID, Query: in.Query, Status: in.Status, Provider: in.Provider})
	if err != nil {
		return repository.Page[model.Integration]{}, err
	}
	return repository.NewPage(items, total, repository.ClampLimit(in.Limit), max(0, in.Offset)), nil
}

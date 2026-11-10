// Package admin implements operational administration features.
package admin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/uuid"
)

// Service provides admin business logic.
type Service struct {
	repo *repository.AdminRepository
	log  *slog.Logger
}

// New returns an admin service.
func New(repo *repository.AdminRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Audit Logs

// CreateAuditLog creates an audit log entry.
func (s *Service) CreateAuditLog(ctx context.Context, actor permissions.AuthContext, in CreateAuditLogInput) error {
	if err := permissions.Authorize(actor, "audit:write"); err != nil {
		return err
	}

	now := time.Now().UTC()
	log := &model.AuditLog{
		ID:           uuid.New(),
		WorkspaceID:  actor.WorkspaceID,
		ActorType:    in.ActorType,
		ActorID:      in.ActorID,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		OldValues:    in.OldValues,
		NewValues:    in.NewValues,
		IPAddress:    in.IPAddress,
		UserAgent:    in.UserAgent,
		CreatedAt:    now,
	}

	return s.repo.CreateAuditLog(ctx, log)
}

// CreateAuditLogInput is the input for creating an audit log.
type CreateAuditLogInput struct {
	ActorType    string `json:"actor_type"`
	ActorID      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	OldValues    string `json:"old_values"`
	NewValues    string `json:"new_values"`
	IPAddress    string `json:"ip_address"`
	UserAgent    string `json:"user_agent"`
}

// ListAuditLogs returns a page of audit logs.
func (s *Service) ListAuditLogs(ctx context.Context, actor permissions.AuthContext, in ListAuditLogsInput) (repository.Page[model.AuditLog], error) {
	if err := permissions.Authorize(actor, "audit:read"); err != nil {
		return repository.Page[model.AuditLog]{}, err
	}
	return s.repo.ListAuditLogs(ctx, repository.AuditLogListOptions{
		Query:        in.Query,
		WorkspaceID:  actor.WorkspaceID,
		ActorType:    in.ActorType,
		ActorID:      in.ActorID,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		StartFrom:    in.StartFrom,
		StartTo:      in.StartTo,
		EndBefore:    in.EndBefore,
		Limit:        in.Limit,
		Offset:       in.Offset,
		Sort:         in.Sort,
		Order:        in.Order,
	})
}

// ListAuditLogsInput holds pagination and filter params for audit logs.
type ListAuditLogsInput struct {
	Query        string `json:"q"`
	WorkspaceID  string `json:"workspace_id"`
	ActorType    string `json:"actor_type"`
	ActorID      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	StartFrom    string `json:"start_from"`
	StartTo      string `json:"start_to"`
	EndBefore    string `json:"end_before"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
	Sort         string `json:"sort"`
	Order        string `json:"order"`
}

// Settings

// GetSetting returns a setting value.
func (s *Service) GetSetting(ctx context.Context, actor permissions.AuthContext, category, key string) (*model.Setting, error) {
	if err := permissions.Authorize(actor, "settings:read"); err != nil {
		return nil, err
	}
	return s.repo.GetSetting(ctx, actor.WorkspaceID, category, key)
}

// SetSetting creates or updates a setting.
func (s *Service) SetSetting(ctx context.Context, actor permissions.AuthContext, in SetSettingInput) (*model.Setting, error) {
	if err := permissions.Authorize(actor, "settings:write"); err != nil {
		return nil, err
	}

	setting, err := s.repo.GetSetting(ctx, actor.WorkspaceID, in.Category, in.Key)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if setting == nil {
		setting = &model.Setting{
			ID:          uuid.New(),
			WorkspaceID: actor.WorkspaceID,
			Category:    in.Category,
			Key:         in.Key,
			Value:       in.Value,
			Description: in.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.repo.CreateSetting(ctx, setting); err != nil {
			return nil, fmt.Errorf("create setting: %w", err)
		}
	} else {
		setting.Value = in.Value
		if in.Description != "" {
			setting.Description = in.Description
		}
		setting.UpdatedAt = now
		if err := s.repo.UpdateSetting(ctx, setting); err != nil {
			return nil, fmt.Errorf("update setting: %w", err)
		}
	}

	s.log.Info("setting updated", "category", in.Category, "key", in.Key, "actor", actor.ActorID)
	return setting, nil
}

// SetSettingInput is the input for setting a setting.
type SetSettingInput struct {
	Category    string `json:"category"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// ListSettings returns all settings for a workspace.
func (s *Service) ListSettings(ctx context.Context, actor permissions.AuthContext, category string) ([]model.Setting, error) {
	if err := permissions.Authorize(actor, "settings:read"); err != nil {
		return nil, err
	}
	return s.repo.ListSettings(ctx, actor.WorkspaceID, category)
}

// DeleteSetting deletes a setting.
func (s *Service) DeleteSetting(ctx context.Context, actor permissions.AuthContext, category, key string) error {
	if err := permissions.Authorize(actor, "settings:write"); err != nil {
		return err
	}
	return s.repo.DeleteSetting(ctx, actor.WorkspaceID, category, key)
}

// Data Model

// GetSchemaTables returns database schema information.
func (s *Service) GetSchemaTables(ctx context.Context, actor permissions.AuthContext) ([]repository.SchemaTable, error) {
	if err := permissions.Authorize(actor, "admin:read"); err != nil {
		return nil, err
	}
	return s.repo.GetSchemaTables(ctx)
}

// GetSchemaMigrations returns applied migrations.
func (s *Service) GetSchemaMigrations(ctx context.Context, actor permissions.AuthContext) ([]repository.SchemaMigration, error) {
	if err := permissions.Authorize(actor, "admin:read"); err != nil {
		return nil, err
	}
	return s.repo.GetSchemaMigrations(ctx)
}

// Workflows Stats

// GetWorkflowStats returns workflow statistics.
func (s *Service) GetWorkflowStats(ctx context.Context, actor permissions.AuthContext) (map[string]any, error) {
	if err := permissions.Authorize(actor, "workflows:read"); err != nil {
		return nil, err
	}
	return s.repo.GetWorkflowStats(ctx, actor.WorkspaceID)
}

// Integrations Stats

// GetIntegrationStats returns integration statistics.
func (s *Service) GetIntegrationStats(ctx context.Context, actor permissions.AuthContext) (map[string]any, error) {
	if err := permissions.Authorize(actor, "integrations:read"); err != nil {
		return nil, err
	}
	return s.repo.GetIntegrationStats(ctx, actor.WorkspaceID)
}

// Agents Stats

// GetAgentStats returns agent statistics.
func (s *Service) GetAgentStats(ctx context.Context, actor permissions.AuthContext) (map[string]any, error) {
	if err := permissions.Authorize(actor, "agents:read"); err != nil {
		return nil, err
	}
	return s.repo.GetAgentStats(ctx, actor.WorkspaceID)
}

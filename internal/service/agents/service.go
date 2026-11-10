// Package agents implements the agent domain service.
package agents

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/time/rate"
	"log/slog"
	"strings"
	"sync"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/uuid"
)

// Service provides agent business logic with authorization.
type Service struct {
	agentRepo  *repository.AgentRepository
	apiKeyRepo *repository.ApiKeyRepository
	log        *slog.Logger
	limitMu    sync.Mutex
	keyLimits  map[string]*rate.Limiter
}

// New returns an agent service.
func New(agentRepo *repository.AgentRepository, apiKeyRepo *repository.ApiKeyRepository, log *slog.Logger) *Service {
	return &Service{agentRepo: agentRepo, apiKeyRepo: apiKeyRepo, log: log, keyLimits: make(map[string]*rate.Limiter)}
}

type CreateAgentInput struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Provider    string         `json:"provider"`
	Enabled     bool           `json:"enabled"`
	Scopes      []string       `json:"scopes"`
	RateLimit   *int           `json:"rate_limit"`
	Config      map[string]any `json:"config"`
}

type ListAgentInput struct {
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	Enabled     *bool  `json:"enabled"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

type UpdateAgentInput struct {
	ID          string         `json:"id"`
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Provider    *string        `json:"provider,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
	Scopes      []string       `json:"scopes,omitempty"`
	RateLimit   *int           `json:"rate_limit,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

type CreateApiKeyInput struct {
	Name      string   `json:"name"`
	AgentID   string   `json:"agent_id"`
	Scopes    []string `json:"scopes"`
	RateLimit *int     `json:"rate_limit"`
	ExpiresAt *string  `json:"expires_at"`
}

type ListApiKeyInput struct {
	WorkspaceID string `json:"workspace_id"`
	AgentID     string `json:"agent_id"`
	Query       string `json:"q"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

type UpdateApiKeyInput struct {
	ID        string   `json:"id"`
	Name      *string  `json:"name,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	RateLimit *int     `json:"rate_limit,omitempty"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

func (s *Service) CreateAgent(ctx context.Context, actor permissions.AuthContext, in CreateAgentInput) (*model.Agent, error) {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if err := validateGrant(actor, in.Scopes, in.RateLimit); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	a := &model.Agent{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		Name:        in.Name,
		Description: in.Description,
		Provider:    in.Provider,
		Enabled:     in.Enabled,
		Scopes:      in.Scopes,
		RateLimit:   in.RateLimit,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.agentRepo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	s.log.Info("agent created", "id", a.ID, "actor", actor.ActorID)
	return a, nil
}

func (s *Service) GetAgent(ctx context.Context, actor permissions.AuthContext, id string) (*model.Agent, error) {
	if err := permissions.Authorize(actor, "agents:read"); err != nil {
		return nil, err
	}
	a, err := s.agentRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil || a.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return a, nil
}

func (s *Service) ListAgents(ctx context.Context, actor permissions.AuthContext, in ListAgentInput) (repository.Page[model.Agent], error) {
	if err := permissions.Authorize(actor, "agents:read"); err != nil {
		return repository.Page[model.Agent]{}, err
	}
	return s.agentRepo.List(ctx, repository.AgentListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		Enabled:     in.Enabled,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

func (s *Service) UpdateAgent(ctx context.Context, actor permissions.AuthContext, id string, in UpdateAgentInput) (*model.Agent, error) {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return nil, err
	}
	a, err := s.agentRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil || a.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if err := validateGrant(actor, in.Scopes, in.RateLimit); err != nil {
		return nil, err
	}
	if in.Name != nil {
		a.Name = *in.Name
	}
	if in.Description != nil {
		a.Description = *in.Description
	}
	if in.Provider != nil {
		a.Provider = *in.Provider
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if in.Scopes != nil {
		a.Scopes = in.Scopes
	}
	if in.RateLimit != nil {
		a.RateLimit = in.RateLimit
	}
	a.UpdatedAt = time.Now().UTC()
	if err := s.agentRepo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	s.log.Info("agent updated", "id", a.ID, "actor", actor.ActorID)
	return a, nil
}

func (s *Service) ArchiveAgent(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return err
	}
	a, err := s.agentRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if a == nil || a.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	if err := s.agentRepo.Archive(ctx, id); err != nil {
		return fmt.Errorf("archive agent: %w", err)
	}
	s.log.Info("agent archived", "id", id, "actor", actor.ActorID)
	return nil
}

func (s *Service) RestoreAgent(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return err
	}
	a, err := s.agentRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if a == nil || a.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	if err := s.agentRepo.Restore(ctx, id); err != nil {
		return fmt.Errorf("restore agent: %w", err)
	}
	s.log.Info("agent restored", "id", id, "actor", actor.ActorID)
	return nil
}

func (s *Service) CreateApiKey(ctx context.Context, actor permissions.AuthContext, in CreateApiKeyInput) (string, *model.APIKey, error) {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return "", nil, fmt.Errorf("name required")
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", nil, fmt.Errorf("agent_id required")
	}
	if len(in.Scopes) == 0 {
		return "", nil, fmt.Errorf("at least one scope required")
	}
	// Verify agent exists and belongs to workspace
	agent, err := s.agentRepo.Get(ctx, in.AgentID)
	if err != nil {
		return "", nil, err
	}
	if agent == nil || agent.WorkspaceID != actor.WorkspaceID {
		return "", nil, fmt.Errorf("agent not found")
	}
	if !agent.Enabled {
		return "", nil, fmt.Errorf("agent is disabled")
	}
	if err := validateGrant(actor, in.Scopes, in.RateLimit); err != nil {
		return "", nil, err
	}
	for _, scope := range in.Scopes {
		if !permissions.HasScope(agent.Scopes, scope) {
			return "", nil, permissions.ErrScopeDenied
		}
	}

	rawKey, prefix, hash := repository.GenerateAPIKey()
	now := time.Now().UTC()

	k := &model.APIKey{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		AgentID:     in.AgentID,
		Name:        in.Name,
		Prefix:      prefix,
		Hash:        hash,
		Scopes:      in.Scopes,
		RateLimit:   in.RateLimit,
		ExpiresAt:   nil,
		CreatedAt:   now,
	}
	if in.ExpiresAt != nil && *in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
		if err != nil || !t.After(now) {
			return "", nil, fmt.Errorf("expires_at must be a future RFC3339 time")
		}
		k.ExpiresAt = &t
	}

	if err := s.apiKeyRepo.Create(ctx, k); err != nil {
		return "", nil, fmt.Errorf("create api key: %w", err)
	}
	s.log.Info("api key created", "id", k.ID, "agent_id", k.AgentID, "actor", actor.ActorID)
	return rawKey, k, nil
}

func (s *Service) GetApiKey(ctx context.Context, actor permissions.AuthContext, id string) (*model.APIKey, error) {
	if err := permissions.Authorize(actor, "agents:read"); err != nil {
		return nil, err
	}
	k, err := s.apiKeyRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if k == nil || k.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return k, nil
}

func (s *Service) ListApiKeys(ctx context.Context, actor permissions.AuthContext, in ListApiKeyInput) (repository.Page[model.APIKey], error) {
	if err := permissions.Authorize(actor, "agents:read"); err != nil {
		return repository.Page[model.APIKey]{}, err
	}
	return s.apiKeyRepo.List(ctx, repository.ApiKeyListOptions{
		WorkspaceID: actor.WorkspaceID,
		AgentID:     in.AgentID,
		Query:       in.Query,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

func (s *Service) UpdateApiKey(ctx context.Context, actor permissions.AuthContext, id string, in UpdateApiKeyInput) (*model.APIKey, error) {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return nil, err
	}
	k, err := s.apiKeyRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if k == nil || k.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if err := validateGrant(actor, in.Scopes, in.RateLimit); err != nil {
		return nil, err
	}
	if in.Scopes != nil {
		agent, err := s.agentRepo.Get(ctx, k.AgentID)
		if err != nil {
			return nil, err
		}
		if agent == nil {
			return nil, permissions.ErrScopeDenied
		}
		for _, scope := range in.Scopes {
			if !permissions.HasScope(agent.Scopes, scope) {
				return nil, permissions.ErrScopeDenied
			}
		}
	}
	if in.Name != nil {
		k.Name = *in.Name
	}
	if in.Scopes != nil {
		k.Scopes = in.Scopes
	}
	if in.RateLimit != nil {
		k.RateLimit = in.RateLimit
	}
	if in.ExpiresAt != nil {
		if *in.ExpiresAt == "" {
			k.ExpiresAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
			if err != nil || !t.After(time.Now()) {
				return nil, fmt.Errorf("expires_at must be a future RFC3339 time")
			}
			k.ExpiresAt = &t
		}
	}
	if err := s.apiKeyRepo.Update(ctx, k); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}
	s.log.Info("api key updated", "id", k.ID, "actor", actor.ActorID)
	return k, nil
}

func validateGrant(actor permissions.AuthContext, scopes []string, rateLimit *int) error {
	if rateLimit != nil && *rateLimit <= 0 {
		return fmt.Errorf("rate_limit must be positive")
	}
	for _, scope := range scopes {
		if !permissions.HasScope(actor.Scopes, scope) {
			return permissions.ErrScopeDenied
		}
	}
	return nil
}

func (s *Service) RevokeApiKey(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "agents:write"); err != nil {
		return err
	}
	k, err := s.apiKeyRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if k == nil || k.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.apiKeyRepo.Revoke(ctx, id, now); err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	s.log.Info("api key revoked", "id", id, "actor", actor.ActorID)
	return nil
}

func (s *Service) ValidateApiKey(ctx context.Context, hash string) (*model.APIKey, error) {
	k, err := s.apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, fmt.Errorf("invalid key")
	}
	if k.RevokedAt != nil {
		return nil, fmt.Errorf("key revoked")
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("key expired")
	}
	agent, err := s.agentRepo.Get(ctx, k.AgentID)
	if err != nil {
		return nil, err
	}
	if agent == nil || !agent.Enabled || agent.WorkspaceID != k.WorkspaceID {
		return nil, fmt.Errorf("agent is disabled or unavailable")
	}
	// A key can never grant more authority than its agent.
	var effective []string
	for _, scope := range k.Scopes {
		if permissions.HasScope(agent.Scopes, scope) {
			effective = append(effective, scope)
		}
	}
	k.Scopes = effective
	limit := k.RateLimit
	if agent.RateLimit != nil && (limit == nil || *agent.RateLimit < *limit) {
		limit = agent.RateLimit
	}
	if limit != nil && *limit > 0 && !s.allowKey(k.ID, *limit) {
		return nil, ErrRateLimited
	}
	// Update last used
	now := time.Now().UTC().Format(time.RFC3339)
	_ = s.apiKeyRepo.UpdateLastUsed(ctx, k.ID, now)
	return k, nil
}

var ErrRateLimited = errors.New("API key rate limit exceeded")

func (s *Service) allowKey(id string, perMinute int) bool {
	s.limitMu.Lock()
	defer s.limitMu.Unlock()
	if len(s.keyLimits) >= 10000 {
		if _, exists := s.keyLimits[id]; !exists {
			id = "overflow"
		}
	}
	limiter := s.keyLimits[id]
	if limiter == nil {
		limiter = rate.NewLimiter(rate.Limit(perMinute)/60, min(perMinute, 200))
		s.keyLimits[id] = limiter
	} else {
		limiter.SetLimit(rate.Limit(perMinute) / 60)
		limiter.SetBurst(min(perMinute, 200))
	}
	return limiter.Allow()
}

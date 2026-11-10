// Package webhooks implements outgoing webhook management.
package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/security"
	"crm.local/internal/uuid"
)

// Service provides webhook business logic.
type Service struct {
	repo *repository.WebhookRepository
	log  *slog.Logger
}

// New returns a webhook service.
func New(repo *repository.WebhookRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// CreateInput is the input for creating a webhook.
type CreateInput struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
	Status string   `json:"status"`
}

// Create creates a webhook.
func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.Webhook, error) {
	if err := permissions.Authorize(actor, "webhooks:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if strings.TrimSpace(in.URL) == "" {
		return nil, fmt.Errorf("url required")
	}
	if len(in.Events) == 0 {
		return nil, fmt.Errorf("at least one event required")
	}
	if strings.TrimSpace(in.Secret) == "" {
		return nil, fmt.Errorf("secret required")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if err := validateWebhook(in.Name, in.URL, in.Events, in.Secret, in.Status); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	eventsJSON, _ := json.Marshal(in.Events)

	webhook := &model.Webhook{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		Name:        in.Name,
		URL:         in.URL,
		Events:      string(eventsJSON),
		Secret:      in.Secret,
		Status:      in.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, webhook); err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}

	s.log.Info("webhook created", "id", webhook.ID, "actor", actor.ActorID)
	return webhook, nil
}

// Get returns a webhook by ID.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Webhook, error) {
	if err := permissions.Authorize(actor, "webhooks:read"); err != nil {
		return nil, err
	}
	webhook, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil || webhook.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return webhook, nil
}

// ListInput holds pagination and filter params.
type ListInput struct {
	WorkspaceID string `json:"workspace_id"`
	Status      string `json:"status"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// List returns a page of webhooks.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) ([]model.Webhook, error) {
	if err := permissions.Authorize(actor, "webhooks:read"); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, repository.WebhookListOptions{
		WorkspaceID: actor.WorkspaceID,
		Status:      in.Status,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	Name   *string  `json:"name,omitempty"`
	URL    *string  `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Secret *string  `json:"secret,omitempty"`
	Status *string  `json:"status,omitempty"`
}

// Update modifies a webhook.
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Webhook, error) {
	if err := permissions.Authorize(actor, "webhooks:write"); err != nil {
		return nil, err
	}
	webhook, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil || webhook.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}

	if in.Name != nil {
		webhook.Name = *in.Name
	}
	if in.URL != nil {
		webhook.URL = *in.URL
	}
	if in.Events != nil {
		eventsJSON, _ := json.Marshal(in.Events)
		webhook.Events = string(eventsJSON)
	}
	if in.Secret != nil {
		webhook.Secret = *in.Secret
	}
	if in.Status != nil {
		webhook.Status = *in.Status
	}

	webhook.UpdatedAt = time.Now().UTC()
	var events []string
	_ = json.Unmarshal([]byte(webhook.Events), &events)
	if err := validateWebhook(webhook.Name, webhook.URL, events, webhook.Secret, webhook.Status); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, webhook); err != nil {
		return nil, fmt.Errorf("update webhook: %w", err)
	}

	s.log.Info("webhook updated", "id", webhook.ID, "actor", actor.ActorID)
	return webhook, nil
}

func validateWebhook(name, rawURL string, events []string, secret, status string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return &repository.ValidationError{Message: "webhook URL must use HTTP or HTTPS and contain no credentials"}
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(secret) == "" || len(events) == 0 {
		return &repository.ValidationError{Message: "webhook name, secret and events are required"}
	}
	if status != "active" && status != "inactive" && status != "disabled" {
		return &repository.ValidationError{Message: "invalid webhook status"}
	}
	return nil
}

// Archive soft-deletes a webhook.
func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "webhooks:write"); err != nil {
		return err
	}
	webhook, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if webhook == nil || webhook.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.repo.Archive(ctx, id, now); err != nil {
		return fmt.Errorf("archive webhook: %w", err)
	}

	s.log.Info("webhook archived", "id", id, "actor", actor.ActorID)
	return nil
}

// RecordTrigger records a webhook trigger attempt.
func (s *Service) RecordTrigger(ctx context.Context, id string, success bool, errorMsg string) error {
	return s.repo.RecordTrigger(ctx, id, success, errorMsg)
}

// Test sends a test webhook payload.
func (s *Service) Test(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "webhooks:write"); err != nil {
		return err
	}
	webhook, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if webhook == nil || webhook.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	// Validate URL for SSRF protection
	secConfig := security.SecurityConfig{
		SSRFBlockPrivateIPs: true,
		SSRFAllowedCIDRs:    []string{},
	}
	if err := security.ValidateSSRFURL(webhook.URL, secConfig); err != nil {
		s.log.Warn("webhook test blocked by SSRF protection", "id", id, "url", webhook.URL, "error", err)
		return fmt.Errorf("webhook URL blocked by SSRF protection: %w", err)
	}

	// Send test payload using secure HTTP client
	client := security.SecureHTTPClient(security.SecureHTTPClientConfig{
		Timeout:         10 * time.Second,
		BlockPrivateIPs: true,
		AllowedCIDRs:    []string{},
		MaxResponseSize: 1024 * 1024, // 1 MB
	})

	testPayload := map[string]any{
		"event":   "webhook.test",
		"webhook": webhook.ID,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}

	jsonPayload, _ := json.Marshal(testPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return fmt.Errorf("create test request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CRM-Webhook/1.0")
	mac := hmac.New(sha256.New, []byte(webhook.Secret))
	_, _ = mac.Write(jsonPayload)
	req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send test webhook: %w", err)
	}
	defer resp.Body.Close()
	defer client.CloseIdleConnections()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}

	s.log.Info("webhook test sent", "id", id, "status", resp.StatusCode, "actor", actor.ActorID)
	return nil
}

func (s *Service) ListPage(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Webhook], error) {
	items, err := s.List(ctx, actor, in)
	if err != nil {
		return repository.Page[model.Webhook]{}, err
	}
	total, err := s.repo.Count(ctx, repository.WebhookListOptions{WorkspaceID: actor.WorkspaceID, Status: in.Status})
	if err != nil {
		return repository.Page[model.Webhook]{}, err
	}
	return repository.NewPage(items, total, repository.ClampLimit(in.Limit), max(0, in.Offset)), nil
}

// Package calendar implements calendar event management.
package calendar

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

// Service provides calendar business logic.
type Service struct {
	repo *repository.CalendarRepository
	log  *slog.Logger
}

// New returns a calendar service.
func New(repo *repository.CalendarRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// CreateInput is the input for creating a calendar event.
type CreateInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	AllDay      bool     `json:"all_day"`
	Location    string   `json:"location"`
	MeetingURL  string   `json:"meeting_url"`
	AttendeeIDs []string `json:"attendee_ids"`
	RelatedType string   `json:"related_type"`
	RelatedID   string   `json:"related_id"`
	Status      string   `json:"status"`
}

// Create creates a calendar event.
func (s *Service) Create(ctx context.Context, actor permissions.AuthContext, in CreateInput) (*model.CalendarEvent, error) {
	if err := permissions.Authorize(actor, "calendar:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("title required")
	}
	if strings.TrimSpace(in.StartTime) == "" || strings.TrimSpace(in.EndTime) == "" {
		return nil, fmt.Errorf("start_time and end_time required")
	}

	now := time.Now().UTC()
	event := &model.CalendarEvent{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		Title:       in.Title,
		Description: in.Description,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		AllDay:      in.AllDay,
		Location:    in.Location,
		MeetingURL:  in.MeetingURL,
		OrganizerID: actor.ActorID,
		AttendeeIDs: in.AttendeeIDs,
		RelatedType: in.RelatedType,
		RelatedID:   in.RelatedID,
		Status:      in.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if event.Status == "" {
		event.Status = "confirmed"
	}

	if err := s.repo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	s.log.Info("calendar event created", "id", event.ID, "actor", actor.ActorID)
	return event, nil
}

// Get returns a calendar event by ID.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.CalendarEvent, error) {
	if err := permissions.Authorize(actor, "calendar:read"); err != nil {
		return nil, err
	}
	event, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil || event.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return event, nil
}

// ListInput holds pagination and filter params.
type ListInput struct {
	Status      string `json:"status"`
	RangeStart  string `json:"range_start"`
	RangeEnd    string `json:"range_end"`
	WorkspaceID string `json:"workspace_id"`
	Query       string `json:"q"`
	StartFrom   string `json:"start_from"`
	StartTo     string `json:"start_to"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// List returns a page of calendar events.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.CalendarEvent], error) {
	if err := permissions.Authorize(actor, "calendar:read"); err != nil {
		return repository.Page[model.CalendarEvent]{}, err
	}
	return s.repo.List(ctx, repository.CalendarListOptions{
		WorkspaceID: actor.WorkspaceID,
		Status:      in.Status,
		RangeStart:  in.RangeStart,
		RangeEnd:    in.RangeEnd,
		Query:       in.Query,
		StartFrom:   in.StartFrom,
		StartTo:     in.StartTo,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	StartTime   *string  `json:"start_time,omitempty"`
	EndTime     *string  `json:"end_time,omitempty"`
	AllDay      *bool    `json:"all_day,omitempty"`
	Location    *string  `json:"location,omitempty"`
	MeetingURL  *string  `json:"meeting_url,omitempty"`
	AttendeeIDs []string `json:"attendee_ids,omitempty"`
	Status      *string  `json:"status,omitempty"`
}

// Update modifies a calendar event.
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.CalendarEvent, error) {
	if err := permissions.Authorize(actor, "calendar:write"); err != nil {
		return nil, err
	}
	event, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil || event.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}

	applyUpdate(event, in)
	event.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	s.log.Info("calendar event updated", "id", event.ID, "actor", actor.ActorID)
	return event, nil
}

// Archive soft-deletes a calendar event.
func (s *Service) Archive(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "calendar:write"); err != nil {
		return err
	}
	event, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if event == nil || event.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.repo.Archive(ctx, id, now); err != nil {
		return fmt.Errorf("archive event: %w", err)
	}

	s.log.Info("calendar event archived", "id", id, "actor", actor.ActorID)
	return nil
}

func applyUpdate(event *model.CalendarEvent, in UpdateInput) {
	if in.Title != nil {
		event.Title = *in.Title
	}
	if in.Description != nil {
		event.Description = *in.Description
	}
	if in.StartTime != nil {
		event.StartTime = *in.StartTime
	}
	if in.EndTime != nil {
		event.EndTime = *in.EndTime
	}
	if in.AllDay != nil {
		event.AllDay = *in.AllDay
	}
	if in.Location != nil {
		event.Location = *in.Location
	}
	if in.MeetingURL != nil {
		event.MeetingURL = *in.MeetingURL
	}
	if in.AttendeeIDs != nil {
		event.AttendeeIDs = in.AttendeeIDs
	}
	if in.Status != nil {
		event.Status = *in.Status
	}
}

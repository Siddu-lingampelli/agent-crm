// Package repository — calendar.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"crm.local/internal/model"
)

// CalendarListOptions extends ListOptions with calendar-specific filters.
type CalendarListOptions struct {
	Status      string
	RangeStart  string
	RangeEnd    string
	WorkspaceID string
	Query       string
	StartFrom   string
	StartTo     string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

const calendarColumns = `
    id, workspace_id, title, description, start_time, end_time, all_day,
    location, meeting_url, organizer_id, attendee_ids, related_type, related_id,
    status, created_at, updated_at
`

// CalendarRepository persists calendar events.
type CalendarRepository struct {
	db *sql.DB
}

// NewCalendarRepository returns a repository over db.
func NewCalendarRepository(db *sql.DB) *CalendarRepository { return &CalendarRepository{db: db} }

// Create inserts a calendar event.
func (r *CalendarRepository) Create(ctx context.Context, e *model.CalendarEvent) error {
	if err := validateEntityReferences(ctx, r.db, e); err != nil {
		return err
	}
	attendeeIDs, _ := json.Marshal(e.AttendeeIDs)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO calendar_events (
            id, workspace_id, title, description, start_time, end_time, all_day,
            location, meeting_url, organizer_id, attendee_ids, related_type, related_id,
            status, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WorkspaceID, e.Title, e.Description, e.StartTime, e.EndTime, e.AllDay,
		e.Location, e.MeetingURL, e.OrganizerID, attendeeIDs, e.RelatedType, e.RelatedID,
		e.Status, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert calendar event: %w", err)
	}
	return nil
}

// Get returns a calendar event by id, or nil if not found.
func (r *CalendarRepository) Get(ctx context.Context, id string) (*model.CalendarEvent, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+calendarColumns+` FROM calendar_events WHERE id = ? AND archived_at IS NULL`, id)
	e, err := scanCalendarEvent(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return e, err
}

// List returns a page of calendar events with server-side pagination.
func (r *CalendarRepository) List(ctx context.Context, opts CalendarListOptions) (Page[model.CalendarEvent], error) {
	where, args := calendarFilters(opts)

	var total int
	countSQL := `SELECT COUNT(*) FROM calendar_events` + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.CalendarEvent]{}, fmt.Errorf("count calendar events: %w", err)
	}

	sort := calendarSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := max(opts.Offset, 0)

	// sort and order are validated by calendarSort and NormalizeOrder, safe to use in query
	//nolint:gosec // G202: sort and order are validated by calendarSort and NormalizeOrder
	sqlText := `SELECT ` + calendarColumns + ` FROM calendar_events` + where +
		` ORDER BY ` + sort + ` ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.CalendarEvent]{}, fmt.Errorf("list calendar events: %w", err)
	}
	defer rows.Close()

	items := []model.CalendarEvent{}
	for rows.Next() {
		item, err := scanCalendarEventRows(rows)
		if err != nil {
			return Page[model.CalendarEvent]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.CalendarEvent]{}, fmt.Errorf("iterate calendar events: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies a calendar event.
func (r *CalendarRepository) Update(ctx context.Context, e *model.CalendarEvent) error {
	if err := validateEntityReferences(ctx, r.db, e); err != nil {
		return err
	}
	attendeeIDs, _ := json.Marshal(e.AttendeeIDs)

	_, err := r.db.ExecContext(ctx, `
        UPDATE calendar_events SET
            title = ?, description = ?, start_time = ?, end_time = ?, all_day = ?,
            location = ?, meeting_url = ?, organizer_id = ?, attendee_ids = ?,
            related_type = ?, related_id = ?, status = ?, updated_at = ?
        WHERE id = ?`,
		e.Title, e.Description, e.StartTime, e.EndTime, e.AllDay,
		e.Location, e.MeetingURL, e.OrganizerID, attendeeIDs,
		e.RelatedType, e.RelatedID, e.Status, e.UpdatedAt, e.ID)
	if err != nil {
		return fmt.Errorf("update calendar event: %w", err)
	}
	return nil
}

// Archive soft-deletes a calendar event.
func (r *CalendarRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE calendar_events SET archived_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("archive calendar event: %w", err)
	}
	return nil
}

func calendarFilters(opts CalendarListOptions) (string, []any) {
	conditions := []string{"archived_at IS NULL"}
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}

	if opts.Query != "" {
		conditions = append(conditions, `(title LIKE ? OR description LIKE ? OR location LIKE ?)`)
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term, term)
	}

	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.RangeStart != "" {
		conditions = append(conditions, "julianday(end_time) > julianday(?)")
		args = append(args, opts.RangeStart)
	}
	if opts.RangeEnd != "" {
		conditions = append(conditions, "julianday(start_time) < julianday(?)")
		args = append(args, opts.RangeEnd)
	}

	if opts.StartFrom != "" {
		conditions = append(conditions, "julianday(start_time) >= julianday(?)")
		args = append(args, opts.StartFrom)
	}

	if opts.StartTo != "" {
		conditions = append(conditions, "julianday(start_time) <= julianday(?)")
		args = append(args, opts.StartTo)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func calendarSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "title":
		return "title"
	case "start_time", "start":
		return "julianday(start_time)"
	case "end_time", "end":
		return "julianday(end_time)"
	case "status":
		return "status"
	default:
		return "julianday(start_time)"
	}
}

func scanCalendarEvent(row *sql.Row) (*model.CalendarEvent, error) {
	e := &model.CalendarEvent{}
	var attendeeIDsJSON string
	var createdAt, updatedAt string
	if err := row.Scan(
		&e.ID, &e.WorkspaceID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.AllDay,
		&e.Location, &e.MeetingURL, &e.OrganizerID, &attendeeIDsJSON, &e.RelatedType, &e.RelatedID,
		&e.Status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	e.UpdatedAt = parseTime(updatedAt)
	_ = json.Unmarshal([]byte(attendeeIDsJSON), &e.AttendeeIDs)
	return e, nil
}

func scanCalendarEventRows(rows *sql.Rows) (*model.CalendarEvent, error) {
	e := &model.CalendarEvent{}
	var attendeeIDsJSON string
	var createdAt, updatedAt string
	if err := rows.Scan(
		&e.ID, &e.WorkspaceID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.AllDay,
		&e.Location, &e.MeetingURL, &e.OrganizerID, &attendeeIDsJSON, &e.RelatedType, &e.RelatedID,
		&e.Status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	e.UpdatedAt = parseTime(updatedAt)
	_ = json.Unmarshal([]byte(attendeeIDsJSON), &e.AttendeeIDs)
	return e, nil
}

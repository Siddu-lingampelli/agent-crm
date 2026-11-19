// Package repository — mail connections, inbound messages, email tracking.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
)

// MailRepository persists mail connections, inbound messages and tracking data.
type MailRepository struct {
	db *sql.DB
}

// NewMailRepository returns a repository over db.
func NewMailRepository(db *sql.DB) *MailRepository { return &MailRepository{db: db} }

// ---------- mail connections ----------

const mailConnectionColumns = `
    id, workspace_id, integration_id, name, provider, email_address,
    imap_host, imap_port, use_tls, username, password_sealed, auth_mode,
    oauth_provider, oauth_sealed, oauth_state, mailbox,
    enabled, poll_interval_seconds, last_sync_at, last_error, created_by,
    created_at, updated_at
`

func scanMailConnection(row interface{ Scan(...any) error }) (*model.MailConnection, error) {
	var c model.MailConnection
	var integrationID sql.NullString
	var useTLS, enabled bool
	var lastSync sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(&c.ID, &c.WorkspaceID, &integrationID, &c.Name, &c.Provider, &c.EmailAddress,
		&c.IMAPHost, &c.IMAPPort, &useTLS, &c.Username, &c.PasswordSealed, &c.AuthMode,
		&c.OAuthProvider, &c.OAuthSealed, &c.OAuthState, &c.Mailbox,
		&enabled, &c.PollIntervalSecs, &lastSync, &c.LastError, &c.CreatedBy, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.IntegrationID = integrationID.String
	c.UseTLS = useTLS
	c.Enabled = enabled
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if lastSync.Valid && lastSync.String != "" {
		t := parseTime(lastSync.String)
		c.LastSyncAt = &t
	}
	return &c, nil
}

// CreateConnection inserts a mail connection.
func (r *MailRepository) CreateConnection(ctx context.Context, c *model.MailConnection) error {
	var integrationID any
	if c.IntegrationID != "" {
		integrationID = c.IntegrationID
	}
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO mail_connections (
            id, workspace_id, integration_id, name, provider, email_address,
            imap_host, imap_port, use_tls, username, password_sealed, auth_mode,
            oauth_provider, oauth_sealed, oauth_state, mailbox,
            enabled, poll_interval_seconds, last_sync_at, last_error, created_by,
            created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, integrationID, c.Name, c.Provider, c.EmailAddress,
		c.IMAPHost, c.IMAPPort, c.UseTLS, c.Username, c.PasswordSealed, c.AuthMode,
		c.OAuthProvider, c.OAuthSealed, c.OAuthState, c.Mailbox,
		c.Enabled, c.PollIntervalSecs, c.LastSyncAt, nullString(&c.LastError), c.CreatedBy,
		c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert mail connection: %w", err)
	}
	return nil
}

// GetConnection returns a connection by id (includes sealed password for internal use).
func (r *MailRepository) GetConnection(ctx context.Context, id string) (*model.MailConnection, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+mailConnectionColumns+` FROM mail_connections WHERE id = ?`, id)
	c, err := scanMailConnection(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// ListConnections returns all connections for a workspace.
func (r *MailRepository) ListConnections(ctx context.Context, workspaceID string) ([]model.MailConnection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+mailConnectionColumns+` FROM mail_connections WHERE workspace_id = ? ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.MailConnection{}
	for rows.Next() {
		c, err := scanMailConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ListEnabledConnections returns enabled connections due for polling.
func (r *MailRepository) ListEnabledConnections(ctx context.Context) ([]model.MailConnection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+mailConnectionColumns+` FROM mail_connections WHERE enabled = 1 ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.MailConnection{}
	for rows.Next() {
		c, err := scanMailConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// UpdateConnection modifies a connection (password_sealed empty = keep existing).
func (r *MailRepository) UpdateConnection(ctx context.Context, c *model.MailConnection) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE mail_connections SET name=?, provider=?, email_address=?, imap_host=?, imap_port=?,
            use_tls=?, username=?, password_sealed=?, auth_mode=?, oauth_provider=?, oauth_sealed=?, oauth_state=?,
            mailbox=?, enabled=?, poll_interval_seconds=?,
            last_sync_at=?, last_error=?, updated_at=? WHERE id=? AND workspace_id=?`,
		c.Name, c.Provider, c.EmailAddress, c.IMAPHost, c.IMAPPort, c.UseTLS, c.Username,
		c.PasswordSealed, c.AuthMode, c.OAuthProvider, c.OAuthSealed, c.OAuthState,
		c.Mailbox, c.Enabled, c.PollIntervalSecs, c.LastSyncAt, c.LastError, c.UpdatedAt,
		c.ID, c.WorkspaceID)
	return err
}

// DeleteConnection removes a connection (cascades to inbound messages).
func (r *MailRepository) DeleteConnection(ctx context.Context, workspaceID, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mail_connections WHERE id = ? AND workspace_id = ?`, id, workspaceID)
	return err
}

// SetConnectionEnabled toggles a connection and records an error note.
func (r *MailRepository) SetConnectionEnabled(ctx context.Context, id string, enabled bool, lastError string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mail_connections SET enabled = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		enabled, lastError, time.Now().UTC(), id)
	return err
}

// UpdateConnectionSync records the result of a poll cycle.
func (r *MailRepository) UpdateConnectionSync(ctx context.Context, id string, syncAt *time.Time, lastError string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mail_connections SET last_sync_at = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		syncAt, lastError, time.Now().UTC(), id)
	return err
}

// UpdateConnectionOAuth stores sealed OAuth tokens and clears the pending state.
func (r *MailRepository) UpdateConnectionOAuth(ctx context.Context, id, oauthSealed, provider, emailAddress string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE mail_connections SET oauth_sealed = ?, oauth_state = '', oauth_provider = ?,
            email_address = COALESCE(NULLIF(?, ''), email_address), auth_mode = 'oauth',
            last_error = '', updated_at = ? WHERE id = ?`,
		oauthSealed, provider, emailAddress, time.Now().UTC(), id)
	return err
}

// CreateOAuthState stores a pending connect flow.
func (r *MailRepository) CreateOAuthState(ctx context.Context, s *model.MailOAuthState) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO mail_oauth_states (state, workspace_id, provider, created_by, expires_at, used)
        VALUES (?, ?, ?, ?, ?, ?)`,
		s.State, s.WorkspaceID, s.Provider, s.CreatedBy, s.ExpiresAt.UTC().Format(time.RFC3339Nano), s.Used)
	return err
}

// ResolveOAuthState returns workspace/user for a pending, unexpired state without consuming it.
func (r *MailRepository) ResolveOAuthState(ctx context.Context, state string) (workspaceID, userID string, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT workspace_id, created_by FROM mail_oauth_states WHERE state = ? AND used = 0 AND julianday(expires_at) > julianday('now')`, state)
	err = row.Scan(&workspaceID, &userID)
	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("invalid or expired state")
	}
	return workspaceID, userID, err
}

// ConsumeOAuthState atomically marks a state used and returns it if valid (single-use, unexpired, same workspace).
func (r *MailRepository) ConsumeOAuthState(ctx context.Context, state, workspaceID string) (*model.MailOAuthState, error) {
	res, err := r.db.ExecContext(ctx, `
        UPDATE mail_oauth_states SET used = 1
        WHERE state = ? AND workspace_id = ? AND used = 0 AND julianday(expires_at) > julianday('now')`,
		state, workspaceID)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("invalid or expired state")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT state, workspace_id, provider, created_by, expires_at, used FROM mail_oauth_states WHERE state = ?`, state)
	var s model.MailOAuthState
	var expiresAt string
	var used bool
	if err := row.Scan(&s.State, &s.WorkspaceID, &s.Provider, &s.CreatedBy, &expiresAt, &used); err != nil {
		return nil, err
	}
	s.ExpiresAt = parseTime(expiresAt)
	s.Used = used
	return &s, nil
}

// ---------- inbound messages ----------

const inboundColumns = `
    id, workspace_id, connection_id, message_id, uid, thread_key, from_addr, from_name,
    to_addr, subject, body_text, body_html, snippet, received_at, is_read, is_archived,
    related_type, related_id, created_at, updated_at
`

func scanInbound(row interface{ Scan(...any) error }) (*model.InboundMessage, error) {
	var m model.InboundMessage
	var isRead, isArchived bool
	var receivedAt, createdAt, updatedAt string
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.ConnectionID, &m.MessageID, &m.UID, &m.ThreadKey,
		&m.FromAddr, &m.FromName, &m.ToAddr, &m.Subject, &m.BodyText, &m.BodyHTML, &m.Snippet,
		&receivedAt, &isRead, &isArchived, &m.RelatedType, &m.RelatedID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	m.IsRead = isRead
	m.IsArchived = isArchived
	m.ReceivedAt = parseTime(receivedAt)
	m.CreatedAt = parseTime(createdAt)
	m.UpdatedAt = parseTime(updatedAt)
	return &m, nil
}

// InsertInbound inserts an inbound message, ignoring duplicates (same connection+uid).
// Returns true when a new row was inserted.
func (r *MailRepository) InsertInbound(ctx context.Context, m *model.InboundMessage) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
        INSERT OR IGNORE INTO inbound_messages (
            id, workspace_id, connection_id, message_id, uid, thread_key, from_addr, from_name,
            to_addr, subject, body_text, body_html, snippet, received_at, is_read, is_archived,
            related_type, related_id, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.WorkspaceID, m.ConnectionID, m.MessageID, m.UID, m.ThreadKey,
		m.FromAddr, m.FromName, m.ToAddr, m.Subject, m.BodyText, m.BodyHTML, m.Snippet,
		m.ReceivedAt, m.IsRead, m.IsArchived, m.RelatedType, m.RelatedID, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return false, fmt.Errorf("insert inbound message: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetInbound returns an inbound message by id.
func (r *MailRepository) GetInbound(ctx context.Context, id string) (*model.InboundMessage, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+inboundColumns+` FROM inbound_messages WHERE id = ?`, id)
	m, err := scanInbound(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// InboundListOptions filters inbound messages.
type InboundListOptions struct {
	WorkspaceID  string
	ConnectionID string
	Query        string
	IsRead       *bool
	IsArchived   *bool
	RelatedType  string
	RelatedID    string
	Limit        int
	Offset       int
}

// ListInbound returns a page of inbound messages.
func (r *MailRepository) ListInbound(ctx context.Context, opts InboundListOptions) (Page[model.InboundMessage], error) {
	var where []string
	var args []any
	where = append(where, "workspace_id = ?")
	args = append(args, opts.WorkspaceID)
	if opts.ConnectionID != "" {
		where = append(where, "connection_id = ?")
		args = append(args, opts.ConnectionID)
	}
	if opts.IsRead != nil {
		where = append(where, "is_read = ?")
		args = append(args, *opts.IsRead)
	}
	if opts.IsArchived != nil {
		where = append(where, "is_archived = ?")
		args = append(args, *opts.IsArchived)
	}
	if opts.RelatedType != "" {
		where = append(where, "related_type = ?")
		args = append(args, opts.RelatedType)
	}
	if opts.RelatedID != "" {
		where = append(where, "related_id = ?")
		args = append(args, opts.RelatedID)
	}
	if q := strings.TrimSpace(opts.Query); q != "" {
		where = append(where, "(subject LIKE ? OR from_addr LIKE ? OR from_name LIKE ? OR body_text LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	w := " WHERE " + strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM inbound_messages`+w, args...).Scan(&total); err != nil {
		return Page[model.InboundMessage]{}, fmt.Errorf("count inbound: %w", err)
	}

	limit := ClampLimit(opts.Limit)
	offset := max(opts.Offset, 0)
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+inboundColumns+` FROM inbound_messages`+w+` ORDER BY received_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return Page[model.InboundMessage]{}, err
	}
	defer rows.Close()
	items := []model.InboundMessage{}
	for rows.Next() {
		m, err := scanInbound(rows)
		if err != nil {
			return Page[model.InboundMessage]{}, err
		}
		items = append(items, *m)
	}
	if err := rows.Err(); err != nil {
		return Page[model.InboundMessage]{}, err
	}
	return NewPage(items, total, limit, offset), nil
}

// UpdateInboundFlags sets read/archived flags and optional related record link.
func (r *MailRepository) UpdateInboundFlags(ctx context.Context, workspaceID, id string, isRead, isArchived bool, relatedType, relatedID string) (*model.InboundMessage, error) {
	_, err := r.db.ExecContext(ctx, `
        UPDATE inbound_messages SET is_read = ?, is_archived = ?, related_type = ?, related_id = ?, updated_at = ?
        WHERE id = ? AND workspace_id = ?`,
		isRead, isArchived, relatedType, relatedID, time.Now().UTC(), id, workspaceID)
	if err != nil {
		return nil, err
	}
	return r.GetInbound(ctx, id)
}

// ---------- email tracking ----------

// CreateTracking registers tracking for an outbound email. Returns existing row if present.
func (r *MailRepository) CreateTracking(ctx context.Context, t *model.EmailTracking) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
        INSERT OR IGNORE INTO email_tracking (
            id, workspace_id, email_id, token, open_count, click_count,
            first_opened_at, last_opened_at, first_clicked_at, last_clicked_at,
            delivered, bounced, bounce_reason, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.WorkspaceID, t.EmailID, t.Token, t.OpenCount, t.ClickCount,
		t.FirstOpenedAt, t.LastOpenedAt, t.FirstClickedAt, t.LastClickedAt,
		t.Delivered, t.Bounced, t.BounceReason, t.CreatedAt)
	if err != nil {
		return false, fmt.Errorf("insert email tracking: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func scanTracking(row interface{ Scan(...any) error }) (*model.EmailTracking, error) {
	var t model.EmailTracking
	var deliv, bounced bool
	var fOpen, lOpen, fClick, lClick, createdAt sql.NullString
	err := row.Scan(&t.ID, &t.WorkspaceID, &t.EmailID, &t.Token, &t.OpenCount, &t.ClickCount,
		&fOpen, &lOpen, &fClick, &lClick, &deliv, &bounced, &t.BounceReason, &createdAt)
	if err != nil {
		return nil, err
	}
	t.Delivered = deliv
	t.Bounced = bounced
	t.FirstOpenedAt = parseNullTime(fOpen)
	t.LastOpenedAt = parseNullTime(lOpen)
	t.FirstClickedAt = parseNullTime(fClick)
	t.LastClickedAt = parseNullTime(lClick)
	if createdAt.Valid {
		t.CreatedAt = parseTime(createdAt.String)
	}
	return &t, nil
}

// GetTrackingByToken loads tracking by its public token (for the tracking endpoints).
func (r *MailRepository) GetTrackingByToken(ctx context.Context, token string) (*model.EmailTracking, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT id, workspace_id, email_id, token, open_count, click_count,
               first_opened_at, last_opened_at, first_clicked_at, last_clicked_at,
               delivered, bounced, bounce_reason, created_at
        FROM email_tracking WHERE token = ?`, token)
	t, err := scanTracking(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// GetTrackingByEmail loads tracking for an email id.
func (r *MailRepository) GetTrackingByEmail(ctx context.Context, emailID string) (*model.EmailTracking, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT id, workspace_id, email_id, token, open_count, click_count,
               first_opened_at, last_opened_at, first_clicked_at, last_clicked_at,
               delivered, bounced, bounce_reason, created_at
        FROM email_tracking WHERE email_id = ?`, emailID)
	t, err := scanTracking(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// RecordEvent inserts one engagement event and updates the aggregate counters.
func (r *MailRepository) RecordEvent(ctx context.Context, ev *model.EmailEvent) error {
	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `
        INSERT INTO email_events (id, tracking_id, email_id, workspace_id, type, url, ip, user_agent, occurred_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.TrackingID, ev.EmailID, ev.WorkspaceID, ev.Type, ev.URL, ev.IP, ev.UserAgent, ev.OccurredAt); err != nil {
		return fmt.Errorf("insert email event: %w", err)
	}
	switch ev.Type {
	case "open":
		_, err := r.db.ExecContext(ctx, `
            UPDATE email_tracking SET open_count = open_count + 1,
                first_opened_at = COALESCE(first_opened_at, ?), last_opened_at = ?
            WHERE id = ?`, now, now, ev.TrackingID)
		return err
	case "click":
		_, err := r.db.ExecContext(ctx, `
            UPDATE email_tracking SET click_count = click_count + 1,
                first_clicked_at = COALESCE(first_clicked_at, ?), last_clicked_at = ?
            WHERE id = ?`, now, now, ev.TrackingID)
		return err
	case "delivered":
		_, err := r.db.ExecContext(ctx, `UPDATE email_tracking SET delivered = 1 WHERE id = ?`, ev.TrackingID)
		return err
	case "bounced":
		_, err := r.db.ExecContext(ctx, `UPDATE email_tracking SET bounced = 1, bounce_reason = ? WHERE id = ?`, ev.URL, ev.TrackingID)
		return err
	}
	return nil
}

// ListEvents returns recent events for an email.
func (r *MailRepository) ListEvents(ctx context.Context, emailID string, limit int) ([]model.EmailEvent, error) {
	limit = ClampLimit(limit)
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, tracking_id, email_id, workspace_id, type, url, ip, user_agent, occurred_at
        FROM email_events WHERE email_id = ? ORDER BY occurred_at DESC LIMIT ?`, emailID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.EmailEvent{}
	for rows.Next() {
		var ev model.EmailEvent
		var occurredAt string
		if err := rows.Scan(&ev.ID, &ev.TrackingID, &ev.EmailID, &ev.WorkspaceID, &ev.Type, &ev.URL, &ev.IP, &ev.UserAgent, &occurredAt); err != nil {
			return nil, err
		}
		ev.OccurredAt = parseTime(occurredAt)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// parseNullTime parses a nullable RFC3339 timestamp column.
func parseNullTime(v sql.NullString) *time.Time {
	if !v.Valid || v.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, v.String)
	if err != nil {
		return nil
	}
	return &t
}

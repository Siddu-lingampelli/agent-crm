// Package mail implements inbox connections (IMAP), inbound message
// ingestion with automatic contact/lead linking, and outbound email
// engagement tracking (opens, clicks, delivery, bounces).
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	netmail "net/mail"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/security"
	"crm.local/internal/uuid"
)

// Config controls the mail service.
type Config struct {
	// TrackingBaseURL is the public base used to build tracking URLs
	// (e.g. https://crm.example.com). Defaults to CRM_PUBLIC_URL or http://<addr>.
	TrackingBaseURL string
	// EncryptionKey is CRM_ENCRYPTION_KEY (hex, 32 bytes) used to unseal IMAP passwords.
	EncryptionKey string
	// PollInterval is the default background poll interval.
	PollInterval time.Duration
	// FetchLimit caps messages fetched per poll cycle per connection.
	FetchLimit int
}

// Service provides mail connection, inbox and tracking logic.
type Service struct {
	repo  *repository.MailRepository
	contacts *repository.ContactRepository
	leads *repository.LeadRepository
	log   *slog.Logger
	cfg   Config

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
}

// New returns a mail service.
func New(repo *repository.MailRepository, contacts *repository.ContactRepository, leads *repository.LeadRepository, cfg Config, log *slog.Logger) *Service {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 60 * time.Second
	}
	if cfg.FetchLimit <= 0 {
		cfg.FetchLimit = 50
	}
	if cfg.TrackingBaseURL == "" {
		cfg.TrackingBaseURL = strings.TrimRight(os.Getenv("CRM_PUBLIC_URL"), "/")
	}
	return &Service{repo: repo, contacts: contacts, leads: leads, cfg: cfg, log: log}
}

// Start launches the background inbox poller. Safe to call once.
func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()
	go s.pollLoop(ctx)
}

// Stop halts the background poller.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.started = false
}

func (s *Service) pollLoop(ctx context.Context) {
	t := time.NewTicker(s.cfg.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.PollAll(ctx)
		}
	}
}

// ---------- connections ----------

// CreateConnectionInput is the input for a new mail connection.
type CreateConnectionInput struct {
	Name         string `json:"name"`
	EmailAddress string `json:"email_address"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	UseTLS       bool   `json:"use_tls"`
	Username     string `json:"username"`
	Password     string `json:"password"` // plaintext, sealed before storage
	Mailbox      string `json:"mailbox"`
	PollIntervalSecs int `json:"poll_interval_seconds"`
}

// CreateConnection validates, seals credentials and stores a connection.
func (s *Service) CreateConnection(ctx context.Context, actor permissions.AuthContext, in CreateConnectionInput) (*model.MailConnection, error) {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	in.EmailAddress = strings.TrimSpace(in.EmailAddress)
	in.IMAPHost = strings.TrimSpace(in.IMAPHost)
	in.Username = strings.TrimSpace(in.Username)
	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if _, err := netmail.ParseAddress(in.EmailAddress); err != nil {
		return nil, fmt.Errorf("invalid email address")
	}
	if in.IMAPHost == "" {
		return nil, fmt.Errorf("imap host required")
	}
	if in.IMAPPort == 0 {
		in.IMAPPort = 993
	}
	if in.Username == "" {
		in.Username = in.EmailAddress
	}
	if in.Password == "" {
		return nil, fmt.Errorf("password required")
	}
	if in.Mailbox == "" {
		in.Mailbox = "INBOX"
	}
	if in.PollIntervalSecs <= 0 {
		in.PollIntervalSecs = int(s.cfg.PollInterval.Seconds())
	}
	sealed, err := security.SealCredentials(in.Password, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("seal credentials: %w", err)
	}
	now := time.Now().UTC()
	c := &model.MailConnection{
		ID:               uuid.New(),
		WorkspaceID:      actor.WorkspaceID,
		Name:             in.Name,
		Provider:         "imap",
		EmailAddress:     in.EmailAddress,
		IMAPHost:         in.IMAPHost,
		IMAPPort:         in.IMAPPort,
		UseTLS:           in.UseTLS || in.IMAPPort == 993,
		Username:         in.Username,
		PasswordSealed:   sealed,
		Mailbox:          in.Mailbox,
		Enabled:          true,
		PollIntervalSecs: in.PollIntervalSecs,
		CreatedBy:        actor.ActorID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateConnection(ctx, c); err != nil {
		return nil, err
	}
	c.PasswordSealed = ""
	return c, nil
}

// ListConnections returns connections for the actor's workspace (no secrets).
func (s *Service) ListConnections(ctx context.Context, actor permissions.AuthContext) ([]model.MailConnection, error) {
	if err := permissions.Authorize(actor, "integrations:read"); err != nil {
		return nil, err
	}
	conns, err := s.repo.ListConnections(ctx, actor.WorkspaceID)
	if err != nil {
		return nil, err
	}
	for i := range conns {
		conns[i].PasswordSealed = ""
	}
	return conns, nil
}

// GetConnection returns one connection (secrets stripped).
func (s *Service) GetConnection(ctx context.Context, actor permissions.AuthContext, id string) (*model.MailConnection, error) {
	if err := permissions.Authorize(actor, "integrations:read"); err != nil {
		return nil, err
	}
	c, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	c.PasswordSealed = ""
	return c, nil
}

// UpdateConnectionInput updates mutable fields; empty Password keeps the stored one.
type UpdateConnectionInput struct {
	Name         string `json:"name"`
	EmailAddress string `json:"email_address"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	UseTLS       bool   `json:"use_tls"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Mailbox      string `json:"mailbox"`
	Enabled      *bool  `json:"enabled"`
	PollIntervalSecs *int `json:"poll_interval_seconds"`
}

// UpdateConnection modifies a connection owned by the actor's workspace.
func (s *Service) UpdateConnection(ctx context.Context, actor permissions.AuthContext, id string, in UpdateConnectionInput) (*model.MailConnection, error) {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return nil, err
	}
	c, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if in.Name != "" {
		c.Name = in.Name
	}
	if in.EmailAddress != "" {
		if _, err := netmail.ParseAddress(in.EmailAddress); err != nil {
			return nil, fmt.Errorf("invalid email address")
		}
		c.EmailAddress = in.EmailAddress
	}
	if in.IMAPHost != "" {
		c.IMAPHost = in.IMAPHost
	}
	if in.IMAPPort != 0 {
		c.IMAPPort = in.IMAPPort
	}
	c.UseTLS = in.UseTLS || c.IMAPPort == 993
	if in.Username != "" {
		c.Username = in.Username
	}
	if in.Mailbox != "" {
		c.Mailbox = in.Mailbox
	}
	if in.Enabled != nil {
		c.Enabled = *in.Enabled
	}
	if in.PollIntervalSecs != nil && *in.PollIntervalSecs > 0 {
		c.PollIntervalSecs = *in.PollIntervalSecs
	}
	if in.Password != "" {
		sealed, err := security.SealCredentials(in.Password, s.cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("seal credentials: %w", err)
		}
		c.PasswordSealed = sealed
	}
	c.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateConnection(ctx, c); err != nil {
		return nil, err
	}
	c.PasswordSealed = ""
	return c, nil
}

// DeleteConnection removes a connection owned by the workspace.
func (s *Service) DeleteConnection(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return err
	}
	return s.repo.DeleteConnection(ctx, actor.WorkspaceID, id)
}

// TestConnection performs a real IMAP login + mailbox select.
func (s *Service) TestConnection(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "integrations:write"); err != nil {
		return err
	}
	c, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return err
	}
	if c == nil || c.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	password, err := s.unseal(c)
	if err != nil {
		return err
	}
	return withClient(ctx, c, password, func(cl *client.Client, mbox string) error {
		_, err := cl.Select(mbox, true)
		return err
	})
}

// ---------- inbox ----------

// ListInbound returns a page of inbound messages.
func (s *Service) ListInbound(ctx context.Context, actor permissions.AuthContext, opts repository.InboundListOptions) (repository.Page[model.InboundMessage], error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return repository.Page[model.InboundMessage]{}, err
	}
	opts.WorkspaceID = actor.WorkspaceID
	return s.repo.ListInbound(ctx, opts)
}

// GetInbound returns one inbound message.
func (s *Service) GetInbound(ctx context.Context, actor permissions.AuthContext, id string) (*model.InboundMessage, error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return nil, err
	}
	m, err := s.repo.GetInbound(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil || m.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return m, nil
}

// UpdateInboundFlagsInput updates read/archive/link flags.
type UpdateInboundFlagsInput struct {
	IsRead      *bool  `json:"is_read"`
	IsArchived  *bool  `json:"is_archived"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
}

// UpdateInboundFlags updates flags and the related-record link.
func (s *Service) UpdateInboundFlags(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInboundFlagsInput) (*model.InboundMessage, error) {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return nil, err
	}
	m, err := s.GetInbound(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	isRead, isArchived := m.IsRead, m.IsArchived
	if in.IsRead != nil {
		isRead = *in.IsRead
	}
	if in.IsArchived != nil {
		isArchived = *in.IsArchived
	}
	relatedType, relatedID := m.RelatedType, m.RelatedID
	if in.RelatedType != "" {
		relatedType = in.RelatedType
		relatedID = in.RelatedID
	}
	return s.repo.UpdateInboundFlags(ctx, actor.WorkspaceID, id, isRead, isArchived, relatedType, relatedID)
}

// PollAll polls every enabled connection once. Used by the background loop
// and by the manual "sync now" endpoint.
func (s *Service) PollAll(ctx context.Context) {
	conns, err := s.repo.ListEnabledConnections(ctx)
	if err != nil {
		s.log.Error("list mail connections for polling", "error", err)
		return
	}
	for i := range conns {
		c := conns[i]
		if !dueForPoll(&c) {
			continue
		}
		n, err := s.PollConnection(ctx, &c)
		now := time.Now().UTC()
		errStr := ""
		if err != nil {
			errStr = err.Error()
			s.log.Warn("mail connection poll failed", "id", c.ID, "host", c.IMAPHost, "error", err)
		} else {
			s.log.Info("mail connection polled", "id", c.ID, "host", c.IMAPHost, "new_messages", n)
		}
		if uerr := s.repo.UpdateConnectionSync(ctx, c.ID, &now, errStr); uerr != nil {
			s.log.Error("update mail connection sync state", "id", c.ID, "error", uerr)
		}
	}
}

// PollConnection polls one connection once and returns the number of new messages.
func (s *Service) PollConnection(ctx context.Context, c *model.MailConnection) (int, error) {
	password, err := s.unseal(c)
	if err != nil {
		return 0, err
	}
	newCount := 0
	err = withClient(ctx, c, password, func(cl *client.Client, mbox string) error {
		m, err := cl.Select(mbox, true)
		if err != nil {
			return fmt.Errorf("select %s: %w", mbox, err)
		}
		if m.Messages == 0 {
			return nil
		}
		// Fetch the most recent N messages.
		limit := uint32(s.cfg.FetchLimit)
		seqset := new(imap.SeqSet)
		from := m.Messages - limit + 1
		if from < 1 {
			from = 1
		}
		seqset.AddRange(from, m.Messages)
		items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, "BODY[]"}
		ch := make(chan *imap.Message, 10)
		done := make(chan error, 1)
		go func() { done <- cl.UidFetch(seqset, items, ch) }()
		for msg := range ch {
			if msg == nil {
				continue
			}
			in, perr := parseIMAPMessage(msg)
			if perr != nil {
				s.log.Warn("skip unparsable message", "uid", msg.Uid, "error", perr)
				continue
			}
			in.ConnectionID = c.ID
			in.WorkspaceID = c.WorkspaceID
			linkRelated(ctx, s, c.WorkspaceID, in)
			inserted, ierr := s.repo.InsertInbound(ctx, in)
			if ierr != nil {
				return ierr
			}
			if inserted {
				newCount++
			}
		}
		return <-done
	})
	return newCount, err
}

// dueForPoll reports whether a connection is due for a poll based on its interval.
func dueForPoll(c *model.MailConnection) bool {
	if c.LastSyncAt == nil {
		return true
	}
	return time.Since(*c.LastSyncAt) >= time.Duration(c.PollIntervalSecs)*time.Second
}

// unseal decrypts the stored IMAP password.
func (s *Service) unseal(c *model.MailConnection) (string, error) {
	if c.PasswordSealed == "" {
		return "", fmt.Errorf("no credentials stored")
	}
	pw, err := security.UnsealCredentials(c.PasswordSealed, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("unseal imap password: %w", err)
	}
	return pw, nil
}

// withClient dials, logs in, selects the mailbox and runs fn, then logs out.
func withClient(ctx context.Context, c *model.MailConnection, password string, fn func(*client.Client, string) error) error {
	addr := fmt.Sprintf("%s:%d", c.IMAPHost, c.IMAPPort)
	var cl *client.Client
	var err error
	if c.UseTLS {
		cl, err = client.DialTLS(addr, nil)
	} else {
		cl, err = client.Dial(addr)
	}
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer cl.Logout()
	cl.Timeout = 30 * time.Second
	if err := cl.Login(c.Username, password); err != nil {
		return fmt.Errorf("imap login: %w", err)
	}
	return fn(cl, c.Mailbox)
}

// parseIMAPMessage converts an IMAP message into an InboundMessage.
func parseIMAPMessage(msg *imap.Message) (*model.InboundMessage, error) {
	now := time.Now().UTC()
	in := &model.InboundMessage{
		ID:        uuid.New(),
		MessageID: "",
		UID:       int(msg.Uid),
		ReceivedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if !msg.InternalDate.IsZero() {
		in.ReceivedAt = msg.InternalDate.UTC()
	}
	if env := msg.Envelope; env != nil {
		in.Subject = env.Subject
		in.MessageID = env.MessageId
		if len(env.From) > 0 {
			in.FromAddr = env.From[0].Address()
			in.FromName = env.From[0].PersonalName
		}
		if len(env.To) > 0 {
			in.ToAddr = env.To[0].Address()
		}
		// Thread key: In-Reply-To, else Message-Id, else subject.
		if env.InReplyTo != "" {
			in.ThreadKey = env.InReplyTo
		} else if env.MessageId != "" {
			in.ThreadKey = env.MessageId
		} else {
			in.ThreadKey = "subject:" + env.Subject
		}
	}
	// Body: fetch the full message (BODY[]) and parse it once; message.Read
	// handles multipart walking for us.
	var raw []byte
	for _, section := range msg.Body {
		if section == nil {
			continue
		}
		b, rerr := io.ReadAll(section)
		if rerr != nil {
			continue
		}
		raw = append(raw, b...)
	}
	if len(raw) > 0 {
		ent, perr := message.Read(bytes.NewReader(raw))
		if perr == nil {
			collectParts(ent, in)
		}
	}
	if in.BodyText == "" && in.BodyHTML != "" {
		in.BodyText = stripHTML(in.BodyHTML)
	}
	if in.BodyText == "" && in.BodyHTML != "" {
		in.BodyText = stripHTML(in.BodyHTML)
	}
	runes := []rune(strings.Join(strings.Fields(in.BodyText), " "))
	if len(runes) > 200 {
		runes = runes[:200]
	}
	in.Snippet = string(runes)
	if in.MessageID == "" {
		in.MessageID = fmt.Sprintf("uid-%d@%s", msg.Uid, "local")
	}
	return in, nil
}

// collectParts walks a parsed MIME entity and extracts text/plain and text/html parts.
func collectParts(ent *message.Entity, in *model.InboundMessage) {
	ct, _, _ := ent.Header.ContentType()
	mediaType := strings.ToLower(ct)
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := ent.MultipartReader()
		if mr == nil {
			return
		}
		for {
			p, err := mr.NextPart()
			if err != nil {
				return
			}
			collectParts(p, in)
		}
	}
	buf, err := io.ReadAll(io.LimitReader(ent.Body, 512*1024))
	if err != nil || len(buf) == 0 {
		return
	}
	switch mediaType {
	case "text/plain", "":
		in.BodyText += string(buf)
	case "text/html":
		in.BodyHTML += string(buf)
	}
}

// stripHTML removes tags crudely for snippet purposes.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// linkRelated auto-links an inbound message to a contact or lead by sender email.
func linkRelated(ctx context.Context, s *Service, workspaceID string, in *model.InboundMessage) {
	from := strings.ToLower(strings.TrimSpace(in.FromAddr))
	if from == "" || s.contacts == nil {
		return
	}
	if c, err := s.contacts.FindDuplicate(ctx, workspaceID, from, "", ""); err == nil && c != nil {
		in.RelatedType = "contact"
		in.RelatedID = c.ID
		return
	}
	if s.leads != nil {
		if l, err := s.leads.FindDuplicate(ctx, workspaceID, from, ""); err == nil && l != nil {
			in.RelatedType = "lead"
			in.RelatedID = l.ID
		}
	}
}

// ---------- tracking ----------

// TrackingEnabled reports whether tracking URLs can be built (base URL configured).
func (s *Service) TrackingBase() string { return s.cfg.TrackingBaseURL }

// EnsureTracking creates a tracking row for an outbound email if missing.
// Returns the tracking token.
func (s *Service) EnsureTracking(ctx context.Context, workspaceID, emailID string) (string, error) {
	existing, err := s.repo.GetTrackingByEmail(ctx, emailID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.Token, nil
	}
	tok, err := newToken()
	if err != nil {
		return "", err
	}
	t := &model.EmailTracking{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		EmailID:     emailID,
		Token:       tok,
		CreatedAt:   time.Now().UTC(),
	}
	if _, err := s.repo.CreateTracking(ctx, t); err != nil {
		return "", err
	}
	return tok, nil
}

// TrackingPixelURL returns the 1x1 open-tracking URL for a token.
func (s *Service) TrackingPixelURL(token string) string {
	return s.cfg.TrackingBaseURL + "/t/o/" + token + ".gif"
}

// InjectPixel appends the open-tracking pixel to an HTML body.
func (s *Service) InjectPixel(bodyHTML, token string) string {
	if s.cfg.TrackingBaseURL == "" || token == "" {
		return bodyHTML
	}
	pixel := `<img src="` + s.TrackingPixelURL(token) + `" width="1" height="1" alt="" style="display:none"/>`
	if i := strings.LastIndex(bodyHTML, "</body>"); i >= 0 {
		return bodyHTML[:i] + pixel + bodyHTML[i:]
	}
	return bodyHTML + pixel
}

// TrackingClickURL returns the redirect URL that records a click then forwards.
func (s *Service) TrackingClickURL(token, target string) string {
	return s.cfg.TrackingBaseURL + "/t/c/" + token + "?u=" + url.QueryEscape(target)
}

// RecordEvent records one engagement event (used by the public tracking endpoints).
func (s *Service) RecordEvent(ctx context.Context, token, eventType, targetURL, ip, userAgent string) error {
	t, err := s.repo.GetTrackingByToken(ctx, token)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("unknown token")
	}
	ev := &model.EmailEvent{
		ID:          uuid.New(),
		TrackingID:  t.ID,
		EmailID:     t.EmailID,
		WorkspaceID: t.WorkspaceID,
		Type:        eventType,
		URL:         targetURL,
		IP:          ip,
		UserAgent:   userAgent,
		OccurredAt:  time.Now().UTC(),
	}
	return s.repo.RecordEvent(ctx, ev)
}

// GetTracking returns aggregate tracking for an outbound email.
func (s *Service) GetTracking(ctx context.Context, actor permissions.AuthContext, emailID string) (*model.EmailTracking, []model.EmailEvent, error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return nil, nil, err
	}
	t, err := s.repo.GetTrackingByEmail(ctx, emailID)
	if err != nil {
		return nil, nil, err
	}
	if t == nil || t.WorkspaceID != actor.WorkspaceID {
		return nil, nil, fmt.Errorf("not found")
	}
	events, err := s.repo.ListEvents(ctx, emailID, 100)
	if err != nil {
		return nil, nil, err
	}
	return t, events, nil
}

// RewriteLinks rewrites http(s) links in an HTML body to tracking click URLs.
func (s *Service) RewriteLinks(token, html string) string {
	if s.cfg.TrackingBaseURL == "" || token == "" {
		return html
	}
	// Simple, safe rewrite: href="http..." -> href="<click url>"
	var b strings.Builder
	rest := html
	for {
		idx := strings.Index(rest, `href="http`)
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:idx])
		rest = rest[idx+len(`href="`):]
		end := strings.Index(rest, `"`)
		if end < 0 {
			b.WriteString(rest)
			break
		}
		target := rest[:end]
		rest = rest[end+1:]
		b.WriteString(`href="` + s.TrackingClickURL(token, target) + `"`)
	}
	return b.String()
}

// newToken returns a URL-safe random token.
func newToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}


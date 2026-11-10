// Package email implements email sending, drafts, history, and provider abstraction.
package email

import (
	"context"
	"crm.local/internal/jobs"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/uuid"
)

// SMTPConfig holds SMTP server configuration.
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"use_tls"`
}

// Service provides email business logic.
type Service struct {
	repo        *repository.EmailRepository
	log         *slog.Logger
	smtpConfig  *SMTPConfig
	approvalReq bool
	mu          sync.RWMutex
	jobPool     *jobs.Pool
}

// New returns an email service.
func New(repo *repository.EmailRepository, log *slog.Logger) *Service {
	return &Service{
		repo:        repo,
		log:         log,
		approvalReq: true, // Require human approval for agent-generated emails
	}
}

// SetSMTPConfig configures the SMTP connection.
func (s *Service) SetSMTPConfig(cfg SMTPConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.smtpConfig = &cfg
}

// GetSMTPConfig returns the current SMTP configuration (without password).
func (s *Service) GetSMTPConfig() *SMTPConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.smtpConfig == nil {
		return nil
	}
	return &SMTPConfig{
		Host:     s.smtpConfig.Host,
		Port:     s.smtpConfig.Port,
		Username: s.smtpConfig.Username,
		From:     s.smtpConfig.From,
		UseTLS:   s.smtpConfig.UseTLS,
	}
}

// CreateDraft creates an email draft.
func (s *Service) CreateDraft(ctx context.Context, actor permissions.AuthContext, in CreateDraftInput) (*model.Email, error) {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.To) == "" {
		return nil, fmt.Errorf("recipient required")
	}
	if strings.TrimSpace(in.Subject) == "" {
		return nil, fmt.Errorf("subject required")
	}
	if strings.TrimSpace(in.Body) == "" && strings.TrimSpace(in.BodyHTML) == "" {
		return nil, fmt.Errorf("body required")
	}

	now := time.Now().UTC()
	email := &model.Email{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		To:          in.To,
		CC:          in.CC,
		BCC:         in.BCC,
		Subject:     in.Subject,
		Body:        in.Body,
		BodyHTML:    in.BodyHTML,
		Status:      "draft",
		Provider:    "smtp",
		RelatedType: in.RelatedType,
		RelatedID:   in.RelatedID,
		CreatedBy:   actor.ActorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, email); err != nil {
		return nil, fmt.Errorf("create draft: %w", err)
	}

	s.log.Info("email draft created", "id", email.ID, "actor", actor.ActorID)
	return email, nil
}

// CreateDraftInput is the input for creating an email draft.
type CreateDraftInput struct {
	To          string `json:"to"`
	CC          string `json:"cc"`
	BCC         string `json:"bcc"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	BodyHTML    string `json:"body_html"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
}

// Get returns an email by ID.
func (s *Service) Get(ctx context.Context, actor permissions.AuthContext, id string) (*model.Email, error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return nil, err
	}
	email, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if email == nil || email.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return email, nil
}

// List returns a page of emails.
func (s *Service) List(ctx context.Context, actor permissions.AuthContext, in ListInput) (repository.Page[model.Email], error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return repository.Page[model.Email]{}, err
	}
	return s.repo.List(ctx, repository.EmailListOptions{
		WorkspaceID: actor.WorkspaceID,
		Query:       in.Query,
		Status:      in.Status,
		RelatedType: in.RelatedType,
		RelatedID:   in.RelatedID,
		Limit:       in.Limit,
		Offset:      in.Offset,
		Sort:        in.Sort,
		Order:       in.Order,
	})
}

// ListInput holds pagination and filter params.
type ListInput struct {
	Query       string `json:"q"`
	WorkspaceID string `json:"workspace_id"`
	Status      string `json:"status"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
}

// Update modifies an email (draft only).
func (s *Service) Update(ctx context.Context, actor permissions.AuthContext, id string, in UpdateInput) (*model.Email, error) {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return nil, err
	}
	email, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if email == nil || email.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if email.Status != "draft" {
		return nil, fmt.Errorf("can only update drafts")
	}

	applyUpdate(email, in)
	email.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, email); err != nil {
		return nil, fmt.Errorf("update email: %w", err)
	}

	s.log.Info("email draft updated", "id", email.ID, "actor", actor.ActorID)
	return email, nil
}

// UpdateInput holds updatable fields.
type UpdateInput struct {
	To       *string `json:"to,omitempty"`
	CC       *string `json:"cc,omitempty"`
	BCC      *string `json:"bcc,omitempty"`
	Subject  *string `json:"subject,omitempty"`
	Body     *string `json:"body,omitempty"`
	BodyHTML *string `json:"body_html,omitempty"`
}

// Delete removes a draft email.
func (s *Service) Delete(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return err
	}
	email, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if email == nil || email.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}
	if email.Status != "draft" {
		return fmt.Errorf("can only delete drafts")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete email: %w", err)
	}

	s.log.Info("email draft deleted", "id", id, "actor", actor.ActorID)
	return nil
}

// Approve approves a draft for sending (human approval for agent-generated).
func (s *Service) Approve(ctx context.Context, actor permissions.AuthContext, id string) (*model.Email, error) {
	if err := permissions.Authorize(actor, "emails:send"); err != nil {
		return nil, err
	}
	if actor.ActorType != "user" {
		return nil, permissions.ErrScopeDenied
	}
	email, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if email == nil || email.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if email.Status != "draft" {
		return nil, fmt.Errorf("can only approve drafts")
	}

	now := time.Now().UTC()
	email.Status = "queued"
	email.ApprovedBy = actor.ActorID
	email.ApprovedAt = &now
	email.UpdatedAt = now

	queued, err := s.repo.QueueForSending(ctx, email)
	if err != nil {
		return nil, err
	}
	if !queued {
		return nil, &repository.ValidationError{Message: "email is already queued or sending"}
	}

	s.log.Info("email approved", "id", email.ID, "actor", actor.ActorID)
	return email, nil
}

// Send queues an email for immediate sending (requires approval if agent-generated).
func (s *Service) Send(ctx context.Context, actor permissions.AuthContext, id string) (*model.Email, error) {
	if err := permissions.Authorize(actor, "emails:send"); err != nil {
		return nil, err
	}
	email, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if email == nil || email.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	if email.Status != "draft" && email.Status != "queued" {
		return nil, fmt.Errorf("can only send draft or queued emails")
	}

	// Check if approval is required
	if s.approvalReq && actor.ActorType == "agent" && email.ApprovedAt == nil {
		return nil, fmt.Errorf("email requires human approval before sending")
	}

	now := time.Now().UTC()
	wasDraft := email.Status == "draft"
	email.Status = "queued"
	if email.ApprovedAt == nil {
		email.ApprovedBy = actor.ActorID
		email.ApprovedAt = &now
	}
	email.UpdatedAt = now

	if wasDraft {
		queued, err := s.repo.QueueForSending(ctx, email)
		if err != nil {
			return nil, err
		}
		if !queued {
			return nil, &repository.ValidationError{Message: "email is already queued or sending"}
		}
	}

	if err := s.enqueueEmail(ctx, email); err != nil {
		return nil, err
	}

	s.log.Info("email queued for sending", "id", email.ID, "actor", actor.ActorID)
	return email, nil
}

// sendEmail sends the email via SMTP.
func (s *Service) sendEmail(ctx context.Context, email *model.Email) {

	if s.smtpSnapshot() == nil {
		s.markFailed(ctx, email, "SMTP not configured")
		return
	}

	claimed, err := s.repo.ClaimForSending(ctx, email.ID)
	if err != nil || !claimed {
		return
	}
	email.Status = "sending"

	if err := s.doSend(ctx, email); err != nil {
		s.markFailed(ctx, email, err.Error())
		return
	}

	now := time.Now().UTC()
	email.Status = "sent"
	email.SentAt = &now
	email.UpdatedAt = now
	if err := s.repo.Update(ctx, email); err != nil {
		s.log.Error("failed to update email status to sent", "error", err)
	}

	s.log.Info("email sent successfully", "id", email.ID)
}

// doSend performs the actual SMTP send.
func (s *Service) doSend(ctx context.Context, email *model.Email) error {
	cfg := s.smtpSnapshot()
	if cfg == nil {
		return fmt.Errorf("SMTP not configured")
	}
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}
	if err := validateHeaders(from, email.To, email.CC, email.BCC, email.Subject); err != nil {
		return err
	}
	to := parseAddresses(email.To)
	cc := parseAddresses(email.CC)
	bcc := parseAddresses(email.BCC)
	recipients := append(append(to, cc...), bcc...)
	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	baseConn := conn
	stop := context.AfterFunc(ctx, func() { _ = baseConn.Close() })
	defer stop()
	tlsConfig := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	if cfg.UseTLS {
		secured := tls.Client(conn, tlsConfig)
		if err := secured.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = secured
	}
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if !cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server must support TLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return err
		}
	}
	sender, err := mail.ParseAddress(from)
	if err != nil {
		return err
	}
	if err := client.Mail(sender.Address); err != nil {
		return err
	}
	for _, recipient := range recipients {
		address, err := mail.ParseAddress(recipient)
		if err != nil {
			return err
		}
		if err := client.Rcpt(address.Address); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(buildMessage(from, to, cc, email.Subject, email.Body, email.BodyHTML))); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// markFailed marks an email as failed with retry logic.
func (s *Service) markFailed(ctx context.Context, email *model.Email, errMsg string) {
	email.RetryCount++
	email.ErrorMessage = errMsg
	email.FailedAt = ptrTime(time.Now().UTC())
	email.UpdatedAt = time.Now().UTC()

	email.Status = "failed"

	if err := s.repo.Update(ctx, email); err != nil {
		s.log.Error("failed to update email status to failed", "error", err)
	}
}

// RetryFailed retries failed emails.
func (s *Service) RetryFailed(ctx context.Context, actor permissions.AuthContext) (int, error) {
	if err := permissions.Authorize(actor, "emails:send"); err != nil {
		return 0, err
	}

	emails, err := s.repo.ListFailed(ctx, actor.WorkspaceID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, email := range emails {
		if email.RetryCount < 3 {
			if actor.ActorType == "agent" && email.ApprovedAt == nil {
				continue
			}
			email.Status = "queued"
			email.UpdatedAt = time.Now().UTC()
			queued, err := s.repo.QueueRetry(ctx, email.ID)
			if err != nil || !queued {
				continue
			}
			if err := s.enqueueEmail(ctx, &email); err != nil {
				continue
			}
			count++
		}
	}

	return count, nil
}

// CreateTemplate creates an email template.
func (s *Service) CreateTemplate(ctx context.Context, actor permissions.AuthContext, in CreateTemplateInput) (*model.EmailTemplate, error) {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name required")
	}
	if strings.TrimSpace(in.Subject) == "" {
		return nil, fmt.Errorf("subject required")
	}

	now := time.Now().UTC()
	template := &model.EmailTemplate{
		ID:          uuid.New(),
		WorkspaceID: actor.WorkspaceID,
		Name:        in.Name,
		Subject:     in.Subject,
		Body:        in.Body,
		BodyHTML:    in.BodyHTML,
		CreatedBy:   actor.ActorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}

	s.log.Info("email template created", "id", template.ID, "actor", actor.ActorID)
	return template, nil
}

// CreateTemplateInput is the input for creating an email template.
type CreateTemplateInput struct {
	Name     string `json:"name"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	BodyHTML string `json:"body_html"`
}

// GetTemplate returns an email template by ID.
func (s *Service) GetTemplate(ctx context.Context, actor permissions.AuthContext, id string) (*model.EmailTemplate, error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return nil, err
	}
	template, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if template == nil || template.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}
	return template, nil
}

// ListTemplates returns a page of email templates.
func (s *Service) ListTemplates(ctx context.Context, actor permissions.AuthContext) ([]model.EmailTemplate, error) {
	if err := permissions.Authorize(actor, "emails:read"); err != nil {
		return nil, err
	}
	return s.repo.ListTemplates(ctx, actor.WorkspaceID)
}

// ApplyTemplate applies a template to create a draft.
func (s *Service) ApplyTemplate(ctx context.Context, actor permissions.AuthContext, templateID string, in ApplyTemplateInput) (*model.Email, error) {
	template, err := s.GetTemplate(ctx, actor, templateID)
	if err != nil {
		return nil, err
	}

	return s.CreateDraft(ctx, actor, CreateDraftInput{
		To:          in.To,
		CC:          in.CC,
		BCC:         in.BCC,
		Subject:     template.Subject,
		Body:        template.Body,
		BodyHTML:    template.BodyHTML,
		RelatedType: in.RelatedType,
		RelatedID:   in.RelatedID,
	})
}

// ApplyTemplateInput is the input for applying a template.
type ApplyTemplateInput struct {
	To          string `json:"to"`
	CC          string `json:"cc"`
	BCC         string `json:"bcc"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
}

func applyUpdate(email *model.Email, in UpdateInput) {
	if in.To != nil {
		email.To = *in.To
	}
	if in.CC != nil {
		email.CC = *in.CC
	}
	if in.BCC != nil {
		email.BCC = *in.BCC
	}
	if in.Subject != nil {
		email.Subject = *in.Subject
	}
	if in.Body != nil {
		email.Body = *in.Body
	}
	if in.BodyHTML != nil {
		email.BodyHTML = *in.BodyHTML
	}
}

// UpdateTemplate updates an email template.
func (s *Service) UpdateTemplate(ctx context.Context, actor permissions.AuthContext, id string, in UpdateTemplateInput) (*model.EmailTemplate, error) {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return nil, err
	}
	template, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if template == nil || template.WorkspaceID != actor.WorkspaceID {
		return nil, fmt.Errorf("not found")
	}

	if in.Name != nil {
		template.Name = *in.Name
	}
	if in.Subject != nil {
		template.Subject = *in.Subject
	}
	if in.Body != nil {
		template.Body = *in.Body
	}
	if in.BodyHTML != nil {
		template.BodyHTML = *in.BodyHTML
	}
	template.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}

	s.log.Info("email template updated", "id", template.ID, "actor", actor.ActorID)
	return template, nil
}

// UpdateTemplateInput holds updatable fields for a template.
type UpdateTemplateInput struct {
	Name     *string `json:"name,omitempty"`
	Subject  *string `json:"subject,omitempty"`
	Body     *string `json:"body,omitempty"`
	BodyHTML *string `json:"body_html,omitempty"`
}

// ArchiveTemplate deletes an email template.
func (s *Service) ArchiveTemplate(ctx context.Context, actor permissions.AuthContext, id string) error {
	if err := permissions.Authorize(actor, "emails:write"); err != nil {
		return err
	}
	template, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return err
	}
	if template == nil || template.WorkspaceID != actor.WorkspaceID {
		return fmt.Errorf("not found")
	}

	if err := s.repo.DeleteTemplate(ctx, id); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	s.log.Info("email template archived", "id", id, "actor", actor.ActorID)
	return nil
}

func parseAddresses(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func buildMessage(from string, to, cc []string, subject, body, bodyHTML string) string {
	var msg strings.Builder
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	if len(cc) > 0 {
		msg.WriteString("Cc: " + strings.Join(cc, ", ") + "\r\n")
	}
	msg.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")

	if bodyHTML != "" {
		msg.WriteString("Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n\r\n")
		msg.WriteString("--boundary123\r\n")
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(body + "\r\n\r\n")
		msg.WriteString("--boundary123\r\n")
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		msg.WriteString(bodyHTML + "\r\n\r\n")
		msg.WriteString("--boundary123--\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(body + "\r\n")
	}

	return msg.String()
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
func (s *Service) SetJobPool(pool *jobs.Pool) { s.jobPool = pool }
func (s *Service) smtpSnapshot() *SMTPConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.smtpConfig == nil {
		return nil
	}
	copy := *s.smtpConfig
	return &copy
}
func (s *Service) enqueueEmail(ctx context.Context, email *model.Email) error {
	if s.jobPool == nil {
		return fmt.Errorf("email worker pool unavailable")
	}
	copy := *email
	return s.jobPool.Enqueue(ctx, "send_email", func(jobCtx context.Context) { s.sendEmail(jobCtx, &copy) })
}
func validateHeaders(from, to, cc, bcc, subject string) error {
	for _, value := range []string{from, to, cc, bcc, subject} {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("email headers must not contain newlines")
		}
	}
	for _, value := range []string{from, to, cc, bcc} {
		if value == "" {
			continue
		}
		addresses, err := mail.ParseAddressList(value)
		if err != nil || len(addresses) > 100 {
			return fmt.Errorf("invalid email address list")
		}
	}
	return nil
}

package model

import "time"

// MailConnection is an IMAP inbox connection for a workspace.
type MailConnection struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspace_id"`
	IntegrationID     string     `json:"integration_id,omitempty"`
	Name              string     `json:"name"`
	Provider          string     `json:"provider"` // imap | gmail | outlook
	EmailAddress      string     `json:"email_address"`
	IMAPHost          string     `json:"imap_host"`
	IMAPPort          int        `json:"imap_port"`
	UseTLS            bool       `json:"use_tls"`
	Username          string     `json:"username"`
	PasswordSealed    string     `json:"-"` // never serialized; set only when creating/updating credentials
	AuthMode          string     `json:"auth_mode"` // password | oauth
	OAuthProvider     string     `json:"oauth_provider,omitempty"` // google | microsoft
	OAuthSealed       string     `json:"-"` // sealed token JSON; internal only
	OAuthState        string     `json:"-"` // pending connect nonce; internal only
	Mailbox           string     `json:"mailbox"`
	Enabled           bool       `json:"enabled"`
	PollIntervalSecs  int        `json:"poll_interval_seconds"`
	LastSyncAt        *time.Time `json:"last_sync_at"`
	LastError         string     `json:"last_error"`
	CreatedBy         string     `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// MailOAuthState is a pending OAuth connect flow (CSRF state + expiry).
type MailOAuthState struct {
	State       string    `json:"state"`
	WorkspaceID string    `json:"workspace_id"`
	Provider    string    `json:"provider"`
	CreatedBy   string    `json:"created_by"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
}

// OAuthTokens is the sealed JSON payload stored in mail_connections.oauth_sealed.
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Expiry       string `json:"expiry,omitempty"` // RFC3339
	Scope        string `json:"scope,omitempty"`
}

// InboundMessage is a message ingested from a connected inbox.
type InboundMessage struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspace_id"`
	ConnectionID string     `json:"connection_id"`
	MessageID    string     `json:"message_id"`
	UID          int        `json:"uid"`
	ThreadKey    string     `json:"thread_key"`
	FromAddr     string     `json:"from_addr"`
	FromName     string     `json:"from_name"`
	ToAddr       string     `json:"to_addr"`
	Subject      string     `json:"subject"`
	BodyText     string     `json:"body_text"`
	BodyHTML     string     `json:"body_html"`
	Snippet      string     `json:"snippet"`
	ReceivedAt   time.Time  `json:"received_at"`
	IsRead       bool       `json:"is_read"`
	IsArchived   bool       `json:"is_archived"`
	RelatedType  string     `json:"related_type"`
	RelatedID    string     `json:"related_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EmailTracking aggregates engagement for one outbound email.
type EmailTracking struct {
	ID             string     `json:"id"`
	WorkspaceID    string     `json:"workspace_id"`
	EmailID        string     `json:"email_id"`
	Token          string     `json:"-"`
	OpenCount      int        `json:"open_count"`
	ClickCount     int        `json:"click_count"`
	FirstOpenedAt  *time.Time `json:"first_opened_at"`
	LastOpenedAt   *time.Time `json:"last_opened_at"`
	FirstClickedAt *time.Time `json:"first_clicked_at"`
	LastClickedAt  *time.Time `json:"last_clicked_at"`
	Delivered      bool       `json:"delivered"`
	Bounced        bool       `json:"bounced"`
	BounceReason   string     `json:"bounce_reason"`
	CreatedAt      time.Time  `json:"created_at"`
}

// EmailEvent is a single engagement event (open/click/delivered/bounced).
type EmailEvent struct {
	ID         string    `json:"id"`
	TrackingID string    `json:"tracking_id"`
	EmailID    string    `json:"email_id"`
	WorkspaceID string   `json:"workspace_id"`
	Type       string    `json:"type"` // open | click | delivered | bounced
	URL        string    `json:"url"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	OccurredAt time.Time `json:"occurred_at"`
}

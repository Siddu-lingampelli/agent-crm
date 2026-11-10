// Package model holds shared domain types and the canonical entity structs.
package model

import "time"

// Timestamps carries the common audit columns.
type Timestamps struct {
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt *time.Time
}

// Company is a customer or prospect company record.
type Company struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspace_id"`
	Name         string         `json:"name"`
	Domain       string         `json:"domain"`
	Industry     string         `json:"industry"`
	SizeBand     string         `json:"size_band"`
	Website      string         `json:"website"`
	Phone        string         `json:"phone"`
	Email        string         `json:"email"`
	Address      string         `json:"address"`
	IsClient     bool           `json:"is_client"`
	Status       string         `json:"status"`
	OwnerID      string         `json:"owner_id"`
	Tags         []string       `json:"tags"`
	CustomFields map[string]any `json:"custom_fields"`
	Notes        string         `json:"notes"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ArchivedAt   *time.Time     `json:"archived_at"`
}

// Contact is a person record.
type Contact struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspace_id"`
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	Email        string         `json:"email"`
	Phone        string         `json:"phone"`
	JobTitle     string         `json:"job_title"`
	CompanyID    string         `json:"company_id"`
	OwnerID      string         `json:"owner_id"`
	Status       string         `json:"status"`
	Tags         []string       `json:"tags"`
	CustomFields map[string]any `json:"custom_fields"`
	Notes        string         `json:"notes"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ArchivedAt   *time.Time     `json:"archived_at"`
}

// Lead is a prospect record.
type Lead struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspace_id"`
	DisplayName    string         `json:"display_name"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	CompanyName    string         `json:"company_name"`
	CompanyID      string         `json:"company_id"`
	JobTitle       string         `json:"job_title"`
	Status         string         `json:"status"`
	Source         string         `json:"source"`
	OwnerID        string         `json:"owner_id"`
	Priority       string         `json:"priority"`
	EstimatedValue *int64         `json:"estimated_value"`
	Currency       string         `json:"currency"`
	LeadScore      *int           `json:"lead_score"`
	ExpectedClose  string         `json:"expected_close"`
	FollowUpDate   string         `json:"follow_up_date"`
	Tags           []string       `json:"tags"`
	CustomFields   map[string]any `json:"custom_fields"`
	Description    string         `json:"description"`
	Qualification  string         `json:"qualification"`
	CreatedBy      string         `json:"created_by"`
	UpdatedBy      string         `json:"updated_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ArchivedAt     *time.Time     `json:"archived_at"`
	ConvertedAt    *time.Time     `json:"converted_at"`
}

// Deal is a sales opportunity.
type Deal struct {
	ID               string         `json:"id"`
	WorkspaceID      string         `json:"workspace_id"`
	Name             string         `json:"name"`
	PipelineID       string         `json:"pipeline_id"`
	StageID          string         `json:"stage_id"`
	Amount           *int64         `json:"amount"`
	Currency         string         `json:"currency"`
	Probability      int            `json:"probability"`
	ExpectedClose    string         `json:"expected_close"`
	CompanyID        string         `json:"company_id"`
	PrimaryContactID string         `json:"primary_contact_id"`
	LeadID           string         `json:"lead_id"`
	OwnerID          string         `json:"owner_id"`
	Source           string         `json:"source"`
	Description      string         `json:"description"`
	LostReason       string         `json:"lost_reason"`
	WonAt            *time.Time     `json:"won_at"`
	LostAt           *time.Time     `json:"lost_at"`
	CustomFields     map[string]any `json:"custom_fields"`
	CreatedBy        string         `json:"created_by"`
	UpdatedBy        string         `json:"updated_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	ArchivedAt       *time.Time     `json:"archived_at"`
}

// Pipeline is a set of stages.
type Pipeline struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PipelineStage is one stage within a pipeline.
type PipelineStage struct {
	ID          string    `json:"id"`
	PipelineID  string    `json:"pipeline_id"`
	Name        string    `json:"name"`
	Position    int       `json:"position"`
	Probability int       `json:"probability"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
}

// Task is a follow-up work item.
type Task struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspace_id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	Priority     string         `json:"priority"`
	DueDate      string         `json:"due_date"`
	AssigneeID   string         `json:"assignee_id"`
	RelatedType  string         `json:"related_type"`
	RelatedID    string         `json:"related_id"`
	ProjectID    string         `json:"project_id"`
	CreatedBy    string         `json:"created_by"`
	CompletedAt  *time.Time     `json:"completed_at"`
	CustomFields map[string]any `json:"custom_fields"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ArchivedAt   *time.Time     `json:"archived_at"`
}

// Project is a lightweight client project.
type Project struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspace_id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	Priority     string         `json:"priority"`
	OwnerID      string         `json:"owner_id"`
	CompanyID    string         `json:"company_id"`
	StartDate    string         `json:"start_date"`
	EndDate      string         `json:"end_date"`
	Progress     int            `json:"progress"`
	CustomFields map[string]any `json:"custom_fields"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ArchivedAt   *time.Time     `json:"archived_at"`
}

// Activity is a timeline event.
type Activity struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Type        string         `json:"type"`
	Summary     string         `json:"summary"`
	ActorType   string         `json:"actor_type"`
	ActorID     string         `json:"actor_id"`
	RelatedType string         `json:"related_type"`
	RelatedID   string         `json:"related_id"`
	Metadata    map[string]any `json:"metadata"`
	OccurredAt  time.Time      `json:"occurred_at"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Agent is an external agent identity.
type Agent struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Provider    string     `json:"provider"`
	Enabled     bool       `json:"enabled"`
	Scopes      []string   `json:"scopes"`
	RateLimit   *int       `json:"rate_limit"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// APIKey is a hashed agent API key.
type APIKey struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	AgentID     string     `json:"agent_id"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	Hash        string     `json:"-"`
	Scopes      []string   `json:"scopes"`
	RateLimit   *int       `json:"rate_limit"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Email represents an email message.
type Email struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspace_id"`
	To                string     `json:"to"`
	CC                string     `json:"cc"`
	BCC               string     `json:"bcc"`
	Subject           string     `json:"subject"`
	Body              string     `json:"body"`
	BodyHTML          string     `json:"body_html"`
	Status            string     `json:"status"`
	Provider          string     `json:"provider"`
	ProviderMessageID string     `json:"provider_message_id"`
	RelatedType       string     `json:"related_type"`
	RelatedID         string     `json:"related_id"`
	CreatedBy         string     `json:"created_by"`
	ApprovedBy        string     `json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	SentAt            *time.Time `json:"sent_at"`
	FailedAt          *time.Time `json:"failed_at"`
	ErrorMessage      string     `json:"error_message"`
	RetryCount        int        `json:"retry_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// EmailTemplate represents an email template.
type EmailTemplate struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	BodyHTML    string    `json:"body_html"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TriggerType represents the type of workflow trigger.
type TriggerType string

const (
	TriggerManual   TriggerType = "manual"
	TriggerSchedule TriggerType = "schedule"
	TriggerEvent    TriggerType = "event"
	TriggerWebhook  TriggerType = "webhook"
)

// StepType represents the type of workflow step.
type StepType string

const (
	StepAction    StepType = "action"
	StepCondition StepType = "condition"
	StepDelay     StepType = "delay"
	StepWebhook   StepType = "webhook"
	StepMCPCall   StepType = "mcp_call"
)

// WorkflowStatus represents the status of a workflow.
type WorkflowStatus string

const (
	WorkflowActive   WorkflowStatus = "active"
	WorkflowInactive WorkflowStatus = "inactive"
	WorkflowDraft    WorkflowStatus = "draft"
)

// WorkflowStep represents a single step in a workflow.
type WorkflowStep struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   StepType       `json:"type"`
	Config map[string]any `json:"config"`
	Order  int            `json:"order"`
}

// Workflow represents an automation workflow.
type Workflow struct {
	ID            string         `json:"id"`
	WorkspaceID   string         `json:"workspace_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	TriggerType   TriggerType    `json:"trigger_type"`
	TriggerConfig map[string]any `json:"trigger_config"`
	Steps         []WorkflowStep `json:"steps"`
	Status        WorkflowStatus `json:"status"`
	CreatedBy     string         `json:"created_by"`
	UpdatedBy     string         `json:"updated_by"`
	LastRunAt     *time.Time     `json:"last_run_at"`
	NextRunAt     *time.Time     `json:"next_run_at"`
	RunCount      int            `json:"run_count"`
	SuccessCount  int            `json:"success_count"`
	FailureCount  int            `json:"failure_count"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// Execution represents a workflow execution.
type Execution struct {
	ID           string         `json:"id"`
	WorkflowID   string         `json:"workflow_id"`
	WorkspaceID  string         `json:"workspace_id"`
	Status       string         `json:"status"` // running, completed, failed, cancelled
	TriggerData  map[string]any `json:"trigger_data"`
	StepsRun     int            `json:"steps_run"`
	StepsTotal   int            `json:"steps_total"`
	ErrorMessage string         `json:"error_message"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	CreatedAt    time.Time      `json:"created_at"`
}

// CalendarEvent represents a calendar event.
type CalendarEvent struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	AllDay      bool      `json:"all_day"`
	Location    string    `json:"location"`
	MeetingURL  string    `json:"meeting_url"`
	OrganizerID string    `json:"organizer_id"`
	AttendeeIDs []string  `json:"attendee_ids"`
	RelatedType string    `json:"related_type"`
	RelatedID   string    `json:"related_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Integration represents a third-party integration.
type Integration struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"`
	Config      string    `json:"config"` // JSON
	Credentials string    `json:"-"`      // Encrypted credentials
	Status      string    `json:"status"` // connected, disconnected, error
	LastSyncAt  string    `json:"last_sync_at"`
	LastError   string    `json:"last_error"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Webhook represents an outgoing webhook.
type Webhook struct {
	ID                string    `json:"id"`
	WorkspaceID       string    `json:"workspace_id"`
	Name              string    `json:"name"`
	URL               string    `json:"url"`
	Events            string    `json:"events"` // JSON array
	Secret            string    `json:"-"`
	Status            string    `json:"status"` // active, inactive, disabled
	FailureCount      int       `json:"failure_count"`
	LastTriggeredAt   string    `json:"last_triggered_at"`
	LastSuccessAt     string    `json:"last_success_at"`
	LastFailureAt     string    `json:"last_failure_at"`
	LastFailureReason string    `json:"last_failure_reason"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	ActorType    string    `json:"actor_type"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	OldValues    string    `json:"old_values"`
	NewValues    string    `json:"new_values"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}

// Setting represents a workspace setting.
type Setting struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Category    string    `json:"category"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

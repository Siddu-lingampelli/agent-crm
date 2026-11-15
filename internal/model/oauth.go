package model

import "time"

// OAuthClient is a registered OAuth 2.1 client (Dynamic Client Registration).
type OAuthClient struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspace_id"`
	ClientID         string    `json:"client_id"`
	ClientSecretHash *string   `json:"-"` // NULL for public clients
	Name             string    `json:"name"`
	RedirectURIs     []string  `json:"redirect_uris"`
	Scopes           []string  `json:"scopes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// OAuthAuthorizationCode is a pending authorization code (PKCE required).
type OAuthAuthorizationCode struct {
	Code                string     `json:"code"`
	ClientID            string     `json:"client_id"`
	WorkspaceID         string     `json:"workspace_id"`
	UserID              string     `json:"user_id"`
	RedirectURI         string     `json:"redirect_uri"`
	CodeChallenge       string     `json:"-"`
	CodeChallengeMethod string     `json:"-"`
	Scopes              []string   `json:"scopes"`
	ExpiresAt           time.Time  `json:"expires_at"`
	UsedAt              *time.Time `json:"used_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// OAuthAccessToken is an issued access token (stored hashed).
type OAuthAccessToken struct {
	TokenHash   string    `json:"-"`
	ClientID    string    `json:"client_id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Scopes      []string  `json:"scopes"`
	ExpiresAt   time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// MCPToolCall records one MCP tools/call invocation for observability.
type MCPToolCall struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ActorType   string    `json:"actor_type"`
	ActorID     string    `json:"actor_id"`
	Tool        string    `json:"tool"`
	Status      string    `json:"status"` // "ok" | "error"
	DurationUS  int64     `json:"duration_us"`
	Error       *string   `json:"error,omitempty"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}

package mail

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/security"
	"crm.local/internal/uuid"
)

// Provider describes a supported OAuth mail provider (Google/Microsoft).
type Provider struct {
	ID           string // google | microsoft
	Name         string // Google | Microsoft
	AuthEndpoint string
	TokenEndpoint string
	UserInfoEndpoint string // optional override (tests)
	IMAPHost     string
	IMAPPort     int
	Scopes       []string
	ClientID     string // from env CRM_MAIL_OAUTH_<ID>_CLIENT_ID
	ClientSecret string // from env CRM_MAIL_OAUTH_<ID>_CLIENT_SECRET
	Configured   bool
}

// providerList is the mutable provider registry (overridable in tests).
var providerList = []Provider{
	{
		ID:            "google",
		Name:          "Google (Gmail)",
		AuthEndpoint:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint: "https://oauth2.googleapis.com/token",
		IMAPHost:      "imap.gmail.com",
		IMAPPort:      993,
		Scopes:        []string{"https://mail.google.com/"},
	},
	{
		ID:            "microsoft",
		Name:          "Microsoft (Outlook/365)",
		AuthEndpoint:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenEndpoint: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		IMAPHost:      "outlook.office365.com",
		IMAPPort:      993,
		Scopes:        []string{"https://outlook.office365.com/IMAP.AccessAsUser.All", "offline_access"},
	},
}

// Providers returns the OAuth mail providers with env configuration applied.
func Providers() []Provider {
	out := make([]Provider, len(providerList))
	copy(out, providerList)
	for i := range out {
		if out[i].ClientID == "" {
			out[i].ClientID = envOr("CRM_MAIL_OAUTH_" + strings.ToUpper(out[i].ID) + "_CLIENT_ID")
		}
		if out[i].ClientSecret == "" {
			out[i].ClientSecret = envOr("CRM_MAIL_OAUTH_" + strings.ToUpper(out[i].ID) + "_CLIENT_SECRET")
		}
	}
	return out
}

func envOr(key string) string { return strings.TrimSpace(os.Getenv(key)) }

// ProviderByID returns a provider by id, with Configured set from env.
func ProviderByID(id string) (Provider, bool) {
	for _, p := range Providers() {
		p.Configured = p.ClientID != "" && p.ClientSecret != ""
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

// SetProvidersForTest replaces the provider registry (tests only).
func SetProvidersForTest(ps []Provider) func() {
	old := providerList
	providerList = ps
	return func() { providerList = old }
}

// RedirectBaseURL is where the provider sends the user back: <CRM_PUBLIC_URL>/mail/oauth/callback.
func (s *Service) RedirectURI() string {
	base := strings.TrimRight(s.cfg.TrackingBaseURL, "/")
	return base + "/api/v1/mail/oauth/callback"
}

// BeginConnect creates a single-use state row and returns the provider's authorization URL.
func (s *Service) BeginConnect(ctx context.Context, workspaceID, userID, providerID string) (string, error) {
	p, ok := ProviderByID(providerID)
	if !ok {
		return "", fmt.Errorf("unknown provider %q", providerID)
	}
	if !p.Configured {
		return "", fmt.Errorf("provider %q is not configured on this server (set CRM_MAIL_OAUTH_%s_CLIENT_ID/SECRET)", providerID, strings.ToUpper(providerID))
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	st := &model.MailOAuthState{
		State:       state,
		WorkspaceID: workspaceID,
		Provider:    p.ID,
		CreatedBy:   userID,
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	}
	if err := s.repo.CreateOAuthState(ctx, st); err != nil {
		return "", err
	}
	q := url.Values{}
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", s.RedirectURI())
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(p.Scopes, " "))
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	if p.ID == "microsoft" {
		q.Set("response_mode", "query")
	}
	return p.AuthEndpoint + "?" + q.Encode(), nil
}

// tokenResponse is the OAuth token endpoint payload.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// ResolveState returns the workspace/user bound to a pending state (used by the public callback).
func (s *Service) ResolveState(ctx context.Context, state string) (workspaceID, userID string, err error) {
	return s.repo.ResolveOAuthState(ctx, state)
}

// CompleteConnect exchanges the code, fetches the user's email, seals tokens, and creates the connection.
func (s *Service) CompleteConnect(ctx context.Context, workspaceID, userID, providerID, state, code string) (*model.MailConnection, error) {
	if code == "" {
		return nil, fmt.Errorf("missing code")
	}
	st, err := s.repo.ConsumeOAuthState(ctx, state, workspaceID)
	if err != nil {
		return nil, err
	}
	if st.Provider != providerID {
		return nil, fmt.Errorf("state/provider mismatch")
	}
	p, _ := ProviderByID(providerID)

	form := url.Values{}
	form.Set("client_id", p.ClientID)
	form.Set("client_secret", p.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", s.RedirectURI())
	form.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil || resp.StatusCode != 200 || tok.AccessToken == "" {
		if tok.Error != "" {
			return nil, fmt.Errorf("provider rejected code: %s: %s", tok.Error, tok.ErrorDesc)
		}
		return nil, fmt.Errorf("token exchange failed (HTTP %d)", resp.StatusCode)
	}

	// Resolve the account email address from the provider userinfo endpoint.
	email := s.fetchUserEmail(ctx, p, tok.AccessToken)

	// Seal tokens.
	payload, _ := json.Marshal(model.OAuthTokens{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Expiry:       time.Now().UTC().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339),
		Scope:        tok.Scope,
	})
	sealed, err := security.SealCredentials(string(payload), s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("seal oauth tokens: %w", err)
	}

	now := time.Now().UTC()
	name := p.Name
	if email != "" {
		name = email
	}
	c := &model.MailConnection{
		ID:               uuid.New(),
		WorkspaceID:      workspaceID,
		Name:             name,
		Provider:         mapProviderID(providerID),
		EmailAddress:     email,
		IMAPHost:         p.IMAPHost,
		IMAPPort:         p.IMAPPort,
		UseTLS:           true,
		Username:         email, // XOAUTH2/Bearer uses the email as the user
		AuthMode:         "oauth",
		OAuthProvider:    p.ID,
		OAuthSealed:      sealed,
		Mailbox:          "INBOX",
		Enabled:          true,
		PollIntervalSecs: int(s.cfg.PollInterval.Seconds()),
		CreatedBy:        userID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateConnection(ctx, c); err != nil {
		return nil, err
	}
	c.PasswordSealed = ""
	c.OAuthSealed = ""
	return c, nil
}

func mapProviderID(id string) string {
	switch id {
	case "google":
		return "gmail"
	case "microsoft":
		return "outlook"
	}
	return id
}

// fetchUserEmail asks the provider who the token belongs to (best effort).
func (s *Service) fetchUserEmail(ctx context.Context, p Provider, accessToken string) string {
	var endpoint string
	switch p.ID {
	case "google":
		endpoint = p.UserInfoEndpoint
		if endpoint == "" {
			endpoint = "https://www.googleapis.com/oauth2/v2/userinfo"
		}
	case "microsoft":
		endpoint = p.UserInfoEndpoint
		if endpoint == "" {
			endpoint = "https://graph.microsoft.com/v1.0/me"
		}
	default:
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out struct {
		Email string `json:"email"`
		Mail  string `json:"mail"`
		UPN   string `json:"userPrincipalName"`
	}
	if json.Unmarshal(body, &out) != nil {
		return ""
	}
	if out.Email != "" {
		return out.Email
	}
	if out.Mail != "" {
		return out.Mail
	}
	return out.UPN
}

// RefreshAccessToken exchanges the refresh token for a new access token and re-seals.
func (s *Service) RefreshAccessToken(ctx context.Context, c *model.MailConnection) error {
	toks, err := s.unsealOAuth(c)
	if err != nil {
		return err
	}
	if toks.RefreshToken == "" {
		return fmt.Errorf("no refresh token stored")
	}
	p, _ := ProviderByID(c.OAuthProvider)
	form := url.Values{}
	form.Set("client_id", p.ClientID)
	form.Set("client_secret", p.ClientSecret)
	form.Set("refresh_token", toks.RefreshToken)
	form.Set("grant_type", "refresh_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil || resp.StatusCode != 200 || tok.AccessToken == "" {
		return fmt.Errorf("refresh failed (HTTP %d)", resp.StatusCode)
	}
	if tok.RefreshToken != "" {
		toks.RefreshToken = tok.RefreshToken
	}
	toks.AccessToken = tok.AccessToken
	toks.Expiry = time.Now().UTC().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339)
	payload, _ := json.Marshal(toks)
	sealed, err := security.SealCredentials(string(payload), s.cfg.EncryptionKey)
	if err != nil {
		return err
	}
	c.OAuthSealed = sealed
	return s.repo.UpdateConnection(ctx, c)
}

// unsealOAuth decrypts the stored token bundle.
func (s *Service) unsealOAuth(c *model.MailConnection) (*model.OAuthTokens, error) {
	if c.OAuthSealed == "" {
		return nil, fmt.Errorf("no oauth tokens stored")
	}
	raw, err := security.UnsealCredentials(c.OAuthSealed, s.cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	var t model.OAuthTokens
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("bad token payload: %w", err)
	}
	return &t, nil
}

// accessToken returns a valid access token, refreshing if expired or near expiry.
func (s *Service) accessToken(ctx context.Context, c *model.MailConnection) (string, error) {
	toks, err := s.unsealOAuth(c)
	if err != nil {
		return "", err
	}
	if toks.Expiry != "" {
		if exp, perr := time.Parse(time.RFC3339, toks.Expiry); perr == nil {
			if time.Until(exp) < 2*time.Minute {
				if err := s.RefreshAccessToken(ctx, c); err != nil {
					return "", fmt.Errorf("token refresh: %w", err)
				}
				toks, err = s.unsealOAuth(c)
				if err != nil {
					return "", err
				}
			}
		}
	}
	return toks.AccessToken, nil
}

func (s *Service) httpClient() *http.Client {
	if s.cfg.HTTPClient != nil {
		return s.cfg.HTTPClient
	}
	return &http.Client{Timeout: 20 * time.Second}
}

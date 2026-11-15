// Package oauth implements OAuth 2.1 + PKCE for the CRM MCP endpoint:
// dynamic client registration, authorization-code flow with mandatory PKCE,
// HMAC-signed JWT access tokens, and RFC 9728 protected-resource metadata.
package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"crm.local/internal/model"
	"crm.local/internal/oauthscopes"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/uuid"
)

const (
	// CodeTTL is how long an authorization code lives.
	CodeTTL = 10 * time.Minute
	// AccessTokenTTL is how long an access token lives.
	AccessTokenTTL = time.Hour
)

// Service issues and validates OAuth tokens for MCP.
type Service struct {
	repo     *repository.OAuthRepository
	issuer   string // base URL, e.g. http://127.0.0.1:8080
	secret   []byte // HMAC key for access-token JWTs
	codeTTL  time.Duration
	tokenTTL time.Duration
}

// New creates the OAuth service. issuer is the public base URL of the app;
// secret is the HMAC signing key (generate once, keep stable across restarts).
func New(repo *repository.OAuthRepository, issuer string, secret []byte) *Service {
	return &Service{repo: repo, issuer: strings.TrimRight(issuer, "/"), secret: secret, codeTTL: CodeTTL, tokenTTL: AccessTokenTTL}
}

func randString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// --- Dynamic Client Registration (RFC 7591) ---

type RegisterClientInput struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	WorkspaceID  string   `json:"-"` // set by caller
}

type RegisteredClient struct {
	ClientID              string   `json:"client_id"`
	ClientSecret          string   `json:"client_secret,omitempty"` // confidential client only
	ClientName            string   `json:"client_name"`
	RedirectURIs          []string `json:"redirect_uris"`
	Scopes                []string `json:"scopes"`
	GrantTypes            []string `json:"grant_types"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
}

// RegisterClient registers a new OAuth client. The client is public (PKCE-only)
// unless registerSecret is true.
func (s *Service) RegisterClient(ctx context.Context, in RegisterClientInput, workspaceID string, confidential bool) (*RegisteredClient, error) {
	if len(in.RedirectURIs) == 0 {
		return nil, fmt.Errorf("redirect_uris required")
	}
	if in.WorkspaceID == "" {
		in.WorkspaceID = workspaceID
	}
	clientID := "crm_client_" + randString(16)
	out := &RegisteredClient{
		ClientID:     clientID,
		ClientName:   in.ClientName,
		RedirectURIs: in.RedirectURIs,
		Scopes:       in.Scopes,
		GrantTypes:   []string{"authorization_code", "refresh_token"},
	}
	c := &model.OAuthClient{
		ID:           uuid.New(),
		WorkspaceID:  workspaceID,
		ClientID:     clientID,
		Name:         in.ClientName,
		RedirectURIs: in.RedirectURIs,
		Scopes:       in.Scopes,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if confidential {
		secret := randString(32)
		h := hashString(secret)
		c.ClientSecretHash = &h
		out.ClientSecret = secret
		out.TokenEndpointAuthMethod = "client_secret_basic"
	} else {
		out.TokenEndpointAuthMethod = "none" // public client, PKCE required
	}
	if err := s.repo.CreateClient(ctx, c); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Authorization endpoint ---

type AuthorizeInput struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	WorkspaceID         string
	UserID              string
}

// Authorize validates an authorization request and returns the redirect URL
// carrying a fresh code. PKCE (S256) is mandatory.
func (s *Service) Authorize(ctx context.Context, in AuthorizeInput) (redirectURL string, err error) {
	client, err := s.repo.GetClientByClientID(ctx, in.ClientID)
	if err != nil {
		return "", err
	}
	if client == nil {
		return "", fmt.Errorf("unknown client_id")
	}
	if !slices.Contains(client.RedirectURIs, in.RedirectURI) {
		return "", fmt.Errorf("redirect_uri not registered")
	}
	if in.ResponseType != "code" {
		return "", fmt.Errorf("unsupported response_type")
	}
	if in.CodeChallenge == "" || !strings.EqualFold(in.CodeChallengeMethod, "S256") {
		return "", fmt.Errorf("PKCE code_challenge (S256) required")
	}
	// requested scopes must be within the client's registered scopes
	want := strings.Fields(in.Scope)
	allowed := map[string]bool{}
	for _, sc := range client.Scopes {
		allowed[sc] = true
	}
	granted := []string{}
	for _, sc := range want {
		if allowed[sc] {
			granted = append(granted, sc)
		}
	}
	code := randString(32)
	now := time.Now().UTC()
	ac := &model.OAuthAuthorizationCode{
		Code:                code,
		ClientID:            in.ClientID,
		WorkspaceID:         in.WorkspaceID,
		UserID:              in.UserID,
		RedirectURI:         in.RedirectURI,
		CodeChallenge:       in.CodeChallenge,
		CodeChallengeMethod: "S256",
		Scopes:              granted,
		ExpiresAt:           now.Add(s.codeTTL),
		CreatedAt:           now,
	}
	if err := s.repo.CreateAuthorizationCode(ctx, ac); err != nil {
		return "", err
	}
	sep := "?"
	if strings.Contains(in.RedirectURI, "?") {
		sep = "&"
	}
	u := in.RedirectURI + sep + "code=" + code
	if in.State != "" {
		u += "&state=" + in.State
	}
	return u, nil
}

// --- Token endpoint ---

type TokenInput struct {
	GrantType       string
	Code            string
	RedirectURI     string
	ClientID        string
	ClientSecret    string
	CodeVerifier    string
	RefreshToken    string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"` // "Bearer" (JWT per RFC 9068)
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Token exchanges an authorization code (with PKCE verification) or a refresh
// token for a signed JWT access token.
func (s *Service) Token(ctx context.Context, in TokenInput) (*TokenResponse, error) {
	switch in.GrantType {
	case "authorization_code":
		return s.exchangeCode(ctx, in)
	case "refresh_token":
		return s.refresh(ctx, in)
	default:
		return nil, fmt.Errorf("unsupported grant_type")
	}
}

func (s *Service) verifyClient(ctx context.Context, clientID, secret string) (*model.OAuthClient, error) {
	client, err := s.repo.GetClientByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("invalid client")
	}
	if client.ClientSecretHash != nil {
		// confidential client must present the secret
		if secret == "" || hashString(secret) != *client.ClientSecretHash {
			return nil, fmt.Errorf("invalid client credentials")
		}
	}
	return client, nil
}

func (s *Service) exchangeCode(ctx context.Context, in TokenInput) (*TokenResponse, error) {
	if _, err := s.verifyClient(ctx, in.ClientID, in.ClientSecret); err != nil {
		return nil, err
	}
	ac, err := s.repo.ConsumeAuthorizationCode(ctx, in.Code, in.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid_grant: %v", err)
	}
	if ac.RedirectURI != in.RedirectURI {
		return nil, fmt.Errorf("invalid_grant: redirect_uri mismatch")
	}
	// PKCE verify: base64url(sha256(verifier)) == challenge
	if in.CodeVerifier == "" {
		return nil, fmt.Errorf("invalid_grant: code_verifier required")
	}
	sum := sha256.Sum256([]byte(in.CodeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	if computed != ac.CodeChallenge {
		return nil, fmt.Errorf("invalid_grant: PKCE verification failed")
	}
	return s.issueTokens(ctx, ac.ClientID, ac.WorkspaceID, ac.UserID, ac.Scopes)
}

func (s *Service) refresh(ctx context.Context, in TokenInput) (*TokenResponse, error) {
	if _, err := s.verifyClient(ctx, in.ClientID, in.ClientSecret); err != nil {
		return nil, err
	}
	old, err := s.repo.GetAccessToken(ctx, hashString(in.RefreshToken))
	if err != nil {
		return nil, fmt.Errorf("invalid_grant: %v", err)
	}
	if old.ClientID != in.ClientID {
		return nil, fmt.Errorf("invalid_grant: client mismatch")
	}
	// rotate: revoke the old refresh token family member
	_ = s.repo.RevokeAccessToken(ctx, old.TokenHash)
	return s.issueTokens(ctx, old.ClientID, old.WorkspaceID, old.UserID, old.Scopes)
}

func (s *Service) issueTokens(ctx context.Context, clientID, workspaceID, userID string, scopes []string) (*TokenResponse, error) {
	now := time.Now().UTC()
	claims := map[string]any{
		"iss": s.issuer,
		"sub": userID,
		"aud": s.issuer + "/mcp",
		"iat": now.Unix(),
		"exp": now.Add(s.tokenTTL).Unix(),
		"jti": uuid.New(),
		"scope": strings.Join(scopes, " "),
		"client_id": clientID,
		"workspace_id": workspaceID,
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	body := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	jwt := body + "." + sig

	tok := &model.OAuthAccessToken{
		TokenHash:   hashString(jwt),
		ClientID:    clientID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Scopes:      scopes,
		ExpiresAt:   now.Add(s.tokenTTL),
		CreatedAt:   now,
	}
	if err := s.repo.CreateAccessToken(ctx, tok); err != nil {
		return nil, err
	}
	refresh := randString(32)
	rtok := &model.OAuthAccessToken{
		TokenHash:   hashString(refresh),
		ClientID:    clientID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Scopes:      scopes,
		ExpiresAt:   now.Add(30 * 24 * time.Hour),
		CreatedAt:   now,
	}
	_ = s.repo.CreateAccessToken(ctx, rtok) // best-effort refresh token store
	return &TokenResponse{
		AccessToken:  jwt,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenTTL.Seconds()),
		RefreshToken: refresh,
		Scope:        strings.Join(scopes, " "),
	}, nil
}

// --- Token validation for /mcp (RFC 9068 JWT profile) ---

// ValidateAccessToken verifies a JWT access token and returns the actor context.
func (s *Service) ValidateAccessToken(ctx context.Context, raw string) (*permissions.AuthContext, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}
	body := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(body))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding")
	}
	var claims struct {
		Sub         string `json:"sub"`
		Iss         string `json:"iss"`
		Aud         string `json:"aud"`
		Exp         int64  `json:"exp"`
		Scope       string `json:"scope"`
		ClientID    string `json:"client_id"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims")
	}
	if claims.Iss != s.issuer || claims.Aud != s.issuer+"/mcp" {
		return nil, fmt.Errorf("invalid issuer/audience")
	}
	if time.Now().Unix() >= claims.Exp {
		return nil, fmt.Errorf("token expired")
	}
	// ensure the token is still tracked (not revoked)
	if _, err := s.repo.GetAccessToken(ctx, hashString(raw)); err != nil {
		return nil, err
	}
	return &permissions.AuthContext{
		ActorType:   "user",
		ActorID:     claims.Sub,
		WorkspaceID: claims.WorkspaceID,
		Scopes:      strings.Fields(claims.Scope),
	}, nil
}

// Revoke invalidates a token by value.
func (s *Service) Revoke(ctx context.Context, raw string) error {
	return s.repo.RevokeAccessToken(ctx, hashString(raw))
}

// --- RFC 9728 protected resource metadata ---

// ProtectedResourceMetadata returns the /.well-known/oauth-protected-resource document.
func (s *Service) ProtectedResourceMetadata() map[string]any {
	return map[string]any{
		"resource":                         s.issuer + "/mcp",
		"authorization_servers":            []string{s.issuer},
		"bearer_methods_supported":         []string{"header"},
		"scopes_supported":                 oauthscopes.All(),
		"resource_documentation":           s.issuer + "/docs/mcp",
	}
}

// AuthorizationServerMetadata returns the /.well-known/oauth-authorization-server document.
func (s *Service) AuthorizationServerMetadata() map[string]any {
	return map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/oauth/authorize",
		"token_endpoint":                        s.issuer + "/oauth/token",
		"registration_endpoint":                 s.issuer + "/oauth/register",
		"revocation_endpoint":                   s.issuer + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "none"},
	}
}

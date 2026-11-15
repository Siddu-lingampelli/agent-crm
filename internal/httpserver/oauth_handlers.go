package httpserver

import (
	"crm.local/internal/auth"
	"crm.local/internal/db"
	"crm.local/internal/jsonutil"
	"crm.local/internal/permissions"
	"crm.local/internal/service/oauth"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

// OAuthHandler serves the OAuth 2.1 endpoints for the MCP resource:
// discovery metadata (RFC 8414/9728), dynamic client registration,
// authorization (with mandatory PKCE), token, and revocation.
type OAuthHandler struct {
	svc *oauth.Service
	db  *db.DB
}

func NewOAuthHandler(svc *oauth.Service, database *db.DB) *OAuthHandler {
	return &OAuthHandler{svc: svc, db: database}
}

func writeJSON(w http.ResponseWriter, status int, v any) { jsonutil.Write(w, status, v) }

func oauthError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","error_description":"` + desc + `"}`))
}

// WellKnownProtectedResource serves /.well-known/oauth-protected-resource (RFC 9728).
func (h *OAuthHandler) WellKnownProtectedResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, h.svc.ProtectedResourceMetadata())
}

// WellKnownAuthorizationServer serves /.well-known/oauth-authorization-server (RFC 8414).
func (h *OAuthHandler) WellKnownAuthorizationServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, h.svc.AuthorizationServerMetadata())
}

// Register handles POST /oauth/register — dynamic client registration (RFC 7591).
// The caller must be an authenticated owner; workspace comes from the session.
func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	if actor.ActorType != "user" || actor.ActorID == "" {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized", "login required")
		return
	}
	var in struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
		Confidential bool     `json:"confidential"`
	}
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if in.ClientName == "" {
		in.ClientName = "MCP client"
	}
	// default scopes: read-only CRM access
	if len(in.Scopes) == 0 {
		in.Scopes = []string{"search:read", "companies:read", "contacts:read", "leads:read", "tasks:read", "calendar:read"}
	}
	reg, err := h.svc.RegisterClient(r.Context(), oauth.RegisterClientInput{
		ClientName:   in.ClientName,
		RedirectURIs: in.RedirectURIs,
		Scopes:       in.Scopes,
	}, actor.WorkspaceID, in.Confidential)
	if err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reg)
}

// Authorize handles GET/POST /oauth/authorize — validates the request and, for a
// logged-in owner, issues a code and 302-redirects back to the client.
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	q := r.URL.Query()
	actor := auth.GetAuth(r)
	// /oauth/authorize is a public (NoAuth) route, so the session middleware never
	// ran — validate the session cookie directly here.
	if actor.ActorID == "" && h.svc != nil {
		if c, err := r.Cookie("crm_session"); err == nil && c.Value != "" {
			userID, verr := auth.ValidateSession(r.Context(), h.db, c.Value)
			if verr == nil && userID != "" {
				var wsID string
				if qerr := h.db.SQL().QueryRowContext(r.Context(),
					`SELECT workspace_id FROM users WHERE id = ?`, userID).Scan(&wsID); qerr == nil {
					actor = permissions.AuthContext{ActorType: "user", ActorID: userID, WorkspaceID: wsID, Scopes: auth.OwnerScopes()}
				}
			}
		}
	}
	if actor.ActorType != "user" || actor.ActorID == "" {
		// not logged in: send to the SPA login page with a return URL
		loginURL := "/login?next=" + url.QueryEscape(r.URL.RequestURI())
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}
	redirect, err := h.svc.Authorize(r.Context(), oauth.AuthorizeInput{
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		ResponseType:        q.Get("response_type"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
		WorkspaceID:         actor.WorkspaceID,
		UserID:              actor.ActorID,
	})
	if err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

// Token handles POST /oauth/token — authorization_code (PKCE) or refresh_token grants.
func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "bad form")
		return
	}
	// client auth: Authorization: Basic base64(client_id:client_secret) or form fields
	clientID, clientSecret := r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	if clientID == "" {
		if u, p, ok := r.BasicAuth(); ok {
			clientID, clientSecret = u, p
		}
	}
	resp, err := h.svc.Token(r.Context(), oauth.TokenInput{
		GrantType:    r.PostForm.Get("grant_type"),
		Code:         r.PostForm.Get("code"),
		RedirectURI:  r.PostForm.Get("redirect_uri"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CodeVerifier: r.PostForm.Get("code_verifier"),
		RefreshToken: r.PostForm.Get("refresh_token"),
	})
	if err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

// Revoke handles POST /oauth/revoke (RFC 7009).
func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	_ = r.ParseForm()
	token := r.PostForm.Get("token")
	if token != "" {
		_ = h.svc.Revoke(r.Context(), token)
	}
	w.WriteHeader(http.StatusOK)
}

// BearerToken extracts a bearer token from the Authorization header.
func BearerToken(r *http.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		// distinguish JWT (3 segments) from opaque API key
		if strings.Count(parts[1], ".") == 2 {
			return parts[1]
		}
		_ = base64.RawURLEncoding // keep import if unused paths change
		return parts[1]
	}
	return ""
}

// IsJWT reports whether a bearer token looks like a JWT.
func IsJWT(token string) bool { return strings.Count(token, ".") == 2 }

var _ = subtle.ConstantTimeCompare // reserved for future constant-time checks
var _ = auth.FromContext

// Package httpserver — REST handlers for mail connections, inbox and tracking.
package httpserver

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"crm.local/internal/jsonutil"
	"crm.local/internal/repository"
	"crm.local/internal/service/mail"
)

// MailHandler wraps the mail service for HTTP.
type MailHandler struct {
	svc *mail.Service
}

// NewMailHandler returns a handler for mail endpoints.
func NewMailHandler(svc *mail.Service) *MailHandler { return &MailHandler{svc: svc} }

func (h *MailHandler) handleError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", "not found")
	case strings.Contains(msg, "not configured") || strings.Contains(msg, "unknown provider") ||
		strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "missing") ||
		strings.Contains(msg, "blocked") || strings.Contains(msg, "port must be"):
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden"):
		jsonutil.Error(w, http.StatusForbidden, "forbidden", msg)
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

// ---------- connections ----------

func (h *MailHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	out, err := h.svc.ListConnections(r.Context(), getAuth(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"items": out})
}

func (h *MailHandler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var in mail.CreateConnectionInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.CreateConnection(r.Context(), getAuth(r), in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

// HandleConnection routes GET/PATCH/DELETE for one connection.
func (h *MailHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		out, err := h.svc.GetConnection(r.Context(), getAuth(r), id)
		if err != nil {
			h.handleError(w, err)
			return
		}
		jsonutil.Write(w, http.StatusOK, out)
	case http.MethodPatch:
		var in mail.UpdateConnectionInput
		if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
			jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
		out, err := h.svc.UpdateConnection(r.Context(), getAuth(r), id, in)
		if err != nil {
			h.handleError(w, err)
			return
		}
		jsonutil.Write(w, http.StatusOK, out)
	case http.MethodDelete:
		if err := h.svc.DeleteConnection(r.Context(), getAuth(r), id); err != nil {
			h.handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *MailHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	id := r.PathValue("id")
	if err := h.svc.TestConnection(r.Context(), getAuth(r), id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SyncNow triggers an immediate poll of all enabled connections.
func (h *MailHandler) SyncNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	// Run in background with a detached context: the request returns immediately
	// and the poll is not cancelled if the client disconnects mid-request.
	go h.svc.PollAll(context.Background())
	jsonutil.Write(w, http.StatusOK, map[string]any{"ok": true, "actor": actor.ActorID})
}

// ---------- inbox ----------

func (h *MailHandler) ListInbound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	q := r.URL.Query()
	opts := repository.InboundListOptions{
		ConnectionID: q.Get("connection_id"),
		Query:        q.Get("q"),
		RelatedType:  q.Get("related_type"),
		RelatedID:    q.Get("related_id"),
	}
	if v := q.Get("is_read"); v != "" {
		b := v == "1" || v == "true"
		opts.IsRead = &b
	}
	if v := q.Get("is_archived"); v != "" {
		b := v == "1" || v == "true"
		opts.IsArchived = &b
	}
	if v := q.Get("limit"); v != "" {
		opts.Limit, _ = strconv.Atoi(v)
	}
	if v := q.Get("offset"); v != "" {
		opts.Offset, _ = strconv.Atoi(v)
	}
	page, err := h.svc.ListInbound(r.Context(), getAuth(r), opts)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, page)
}

// HandleInbound routes GET/PATCH for one inbound message.
func (h *MailHandler) HandleInbound(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		out, err := h.svc.GetInbound(r.Context(), getAuth(r), id)
		if err != nil {
			h.handleError(w, err)
			return
		}
		jsonutil.Write(w, http.StatusOK, out)
	case http.MethodPatch:
		var in mail.UpdateInboundFlagsInput
		if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
			jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
		out, err := h.svc.UpdateInboundFlags(r.Context(), getAuth(r), id, in)
		if err != nil {
			h.handleError(w, err)
			return
		}
		jsonutil.Write(w, http.StatusOK, out)
	default:
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

// ---------- tracking (public, token-authenticated) ----------

// pixel is a 1x1 transparent GIF.
var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// TrackOpen handles GET /t/o/{token}.gif — records an open and returns the pixel.
func (h *MailHandler) TrackOpen(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSuffix(r.PathValue("token"), ".gif")
	if token == "" {
		http.NotFound(w, r)
		return
	}
	_ = h.svc.RecordEvent(r.Context(), token, "open", "", clientIP(r), r.UserAgent())
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
	_, _ = w.Write(pixelGIF)
}

// TrackClick handles GET /t/c/{token}?u=<target> — records a click and redirects.
func (h *MailHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	target := r.URL.Query().Get("u")
	if target == "" || !(strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")) {
		http.Error(w, "bad target", http.StatusBadRequest)
		return
	}
	_ = h.svc.RecordEvent(r.Context(), token, "click", target, clientIP(r), r.UserAgent())
	http.Redirect(w, r, target, http.StatusFound)
}

// TrackingStats handles GET /api/v1/emails/{id}/tracking (protected).
func (h *MailHandler) TrackingStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	id := r.PathValue("id")
	t, events, err := h.svc.GetTracking(r.Context(), getAuth(r), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"tracking": t, "events": events})
}

// ---------- OAuth "connect" flow (HubSpot/Salesforce style) ----------

// ListOAuthProviders returns which mail providers are configured server-side.
func (h *MailHandler) ListOAuthProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if _, err := h.svc.ListConnections(r.Context(), getAuth(r)); err != nil {
		// just checking auth scope
	}
	out := []map[string]any{}
	for _, p := range mail.Providers() {
		p.Configured = p.ClientID != "" && p.ClientSecret != ""
		out = append(out, map[string]any{"id": p.ID, "name": p.Name, "configured": p.Configured})
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"providers": out})
}

// BeginOAuthConnect starts the connect flow: returns the provider's authorization URL.
// The SPA opens it in a popup/new tab; the provider redirects to /api/v1/mail/oauth/callback.
func (h *MailHandler) BeginOAuthConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in struct {
		Provider string `json:"provider"`
	}
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	authURL, err := h.svc.BeginConnect(r.Context(), actor.WorkspaceID, actor.ActorID, in.Provider)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"authorization_url": authURL, "provider": in.Provider})
}

// OAuthCallback handles the provider redirect (NoAuth route; state row is the CSRF guard).
// On success it creates the connection and redirects back to the SPA inbox with a flag.
func (h *MailHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		http.Redirect(w, r, "/inbox?mail_error="+url.QueryEscape(q.Get("error_description")+" "+e), http.StatusFound)
		return
	}
	code, state, provider := q.Get("code"), q.Get("state"), q.Get("provider")
	if code == "" || state == "" || provider == "" {
		http.Redirect(w, r, "/inbox?mail_error="+url.QueryEscape("missing code/state/provider"), http.StatusFound)
		return
	}
	// The state row is bound to the workspace that started the flow. We need the
	// workspace to complete; look it up from the state row itself (public route).
	wsID, userID, err := h.svc.ResolveState(r.Context(), state)
	if err != nil {
		http.Redirect(w, r, "/inbox?mail_error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	conn, err := h.svc.CompleteConnect(r.Context(), wsID, userID, provider, state, code)
	if err != nil {
		http.Redirect(w, r, "/inbox?mail_error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/inbox?mail_connected="+url.QueryEscape(conn.Name), http.StatusFound)
}

// clientIP extracts the remote IP, honoring X-Forwarded-For only from the first hop.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}

// Package httpserver — REST handlers for mail connections, inbox and tracking.
package httpserver

import (
	"net/http"
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
	switch {
	case err.Error() == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", "not found")
	case strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid"):
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", err.Error())
	case strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "forbidden"):
		jsonutil.Error(w, http.StatusForbidden, "forbidden", err.Error())
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
	// run in background so the request returns fast; poll is quick enough to do inline too
	h.svc.PollAll(r.Context())
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

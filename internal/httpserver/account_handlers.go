package httpserver

import (
	"crm.local/internal/auth"
	"crm.local/internal/jsonutil"
	"errors"
	"net/http"
)

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	actor := auth.GetAuth(r)
	var email string
	if err := h.db.SQL().QueryRowContext(r.Context(), "SELECT email FROM users WHERE id=? AND workspace_id=?", actor.ActorID, actor.WorkspaceID).Scan(&email); err != nil {
		jsonutil.Error(w, 401, "unauthorized", "account unavailable")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonutil.Write(w, 200, map[string]any{"id": actor.ActorID, "email": email, "workspace_id": actor.WorkspaceID})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	if !h.loginLimit.Allow() {
		w.Header().Set("Retry-After", "1")
		jsonutil.Error(w, 429, "rate_limited", "too many password attempts")
		return
	}
	var in struct {
		Current string `json:"current_password"`
		Next    string `json:"new_password"`
	}
	if err := jsonutil.Decode(r, 16384, &in); err != nil {
		jsonutil.Error(w, 400, "invalid_body", err.Error())
		return
	}
	var token string
	if cookie, err := r.Cookie(h.options.CookieName); err == nil {
		token = cookie.Value
	}
	err := auth.ChangePassword(r.Context(), h.db, auth.GetAuth(r), in.Current, in.Next, token)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCurrentPassword), errors.Is(err, auth.ErrPasswordLength):
			jsonutil.Error(w, 400, "validation_failed", err.Error())
		case errors.Is(err, auth.ErrPasswordChanged):
			jsonutil.Error(w, 409, "conflict", err.Error())
		default:
			if !writeKnownServiceError(w, err) {
				jsonutil.Error(w, 500, "password_change_failed", "could not change password")
			}
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonutil.Write(w, 200, map[string]bool{"success": true})
}

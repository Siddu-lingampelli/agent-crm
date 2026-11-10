// Package httpserver — REST handlers for authentication.
package httpserver

import (
	"golang.org/x/time/rate"
	"net/http"

	"crm.local/internal/auth"
	"crm.local/internal/db"
	"crm.local/internal/jsonutil"
)

// AuthHandler wraps the auth functions for HTTP.
type AuthHandler struct {
	db         *db.DB
	options    auth.SessionOptions
	loginLimit *rate.Limiter
}

// NewAuthHandler returns a handler for auth endpoints.
func NewAuthHandler(database *db.DB, configured ...auth.SessionOptions) *AuthHandler {
	options := auth.DefaultSessionOptions()
	if len(configured) > 0 {
		options = configured[0]
	}
	return &AuthHandler{db: database, options: options, loginLimit: rate.NewLimiter(1, 5)}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.loginLimit.Allow() {
		w.Header().Set("Retry-After", "1")
		jsonutil.Error(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts")
		return
	}

	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	result, err := auth.LoginWithOptions(r.Context(), h.db, in.Email, in.Password, h.options)
	if err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	if !result.Success {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized", result.Reason)
		return
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     h.options.CookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: h.options.SameSite,
		Secure:   h.options.Secure || r.TLS != nil,
		Expires:  result.ExpiresAt,
	}
	http.SetCookie(w, cookie)

	jsonutil.Write(w, http.StatusOK, map[string]any{
		"token":      result.Token,
		"expires_at": result.ExpiresAt,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	cookie, err := r.Cookie(h.options.CookieName)
	if err == nil {
		if err := auth.Logout(r.Context(), h.db, cookie.Value); err != nil {
			jsonutil.Error(w, http.StatusInternalServerError, "logout_failed", "could not invalidate session")
			return
		}
	}

	// Clear session cookie
	clearCookie := &http.Cookie{
		Name:     h.options.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: h.options.SameSite,
		Secure:   h.options.Secure || r.TLS != nil,
		MaxAge:   -1,
	}
	http.SetCookie(w, clearCookie)

	jsonutil.Write(w, http.StatusOK, map[string]any{"success": true})
}

// Package httpserver — REST handlers for dashboard endpoints.
package httpserver

import (
	"net/http"
	"time"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/dashboard"
)

// DashboardHandler wraps dashboard endpoints.
type DashboardHandler struct {
	svc *dashboard.Service
}

// NewDashboardHandler returns a handler for dashboard endpoints.
func NewDashboardHandler(svc *dashboard.Service) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	actor := getAuth(r)
	day := time.Now().UTC()
	if value := r.URL.Query().Get("day"); value != "" {
		parsed, parseErr := time.Parse("2006-01-02", value)
		if parseErr != nil {
			jsonutil.Error(w, 400, "invalid_date", "day must be a valid YYYY-MM-DD date")
			return
		}
		day = parsed
	}
	metrics, err := h.svc.GetDashboardMetrics(r.Context(), actor, day)
	if err != nil {
		if writeKnownServiceError(w, err) {
			return
		}
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "could not load dashboard metrics")
		return
	}

	jsonutil.Write(w, http.StatusOK, metrics)
}

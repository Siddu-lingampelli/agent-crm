// Package httpserver — REST handlers for calendar.
package httpserver

import (
	"net/http"
	"strconv"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/calendar"
)

// CalendarHandler wraps the calendar service for HTTP.
type CalendarHandler struct {
	svc *calendar.Service
}

// NewCalendarHandler returns a handler for calendar endpoints.
func NewCalendarHandler(svc *calendar.Service) *CalendarHandler {
	return &CalendarHandler{svc: svc}
}

func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in calendar.CreateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.Create(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *CalendarHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	id := r.PathValue("id")
	if id == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	out, err := h.svc.Get(r.Context(), actor, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *CalendarHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := calendar.ListInput{
		Status:     r.URL.Query().Get("status"),
		RangeStart: r.URL.Query().Get("range_start"),
		RangeEnd:   r.URL.Query().Get("range_end"),
		Query:      r.URL.Query().Get("q"),
		StartFrom:  r.URL.Query().Get("start_from"),
		StartTo:    r.URL.Query().Get("start_to"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		in.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		in.Offset, _ = strconv.Atoi(v)
	}
	in.Sort = r.URL.Query().Get("sort")
	in.Order = r.URL.Query().Get("order")
	page, err := h.svc.List(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, page)
}

func (h *CalendarHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	id := r.PathValue("id")
	if id == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	var in calendar.UpdateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.Update(r.Context(), actor, id, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *CalendarHandler) Archive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	id := r.PathValue("id")
	if id == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	if err := h.svc.Archive(r.Context(), actor, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CalendarHandler) handleError(w http.ResponseWriter, err error) {
	if writeKnownServiceError(w, err) {
		return
	}
	msg := err.Error()
	switch {
	case msg == "title required", msg == "start_time and end_time required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

// Package httpserver — tasks handler.
package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/tasks"
)

// TaskHandler wraps the task service for HTTP.
type TaskHandler struct {
	svc *tasks.Service
	log *slog.Logger
}

// NewTaskHandler returns a handler for task endpoints.
func NewTaskHandler(svc *tasks.Service, log *slog.Logger) *TaskHandler {
	return &TaskHandler{svc: svc, log: log}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in tasks.CreateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.Create(r.Context(), actor, in)
	if err != nil {
		h.log.Error("task create failed", "error", err)
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
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

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := tasks.ListInput{
		ActiveOnly:  r.URL.Query().Get("active_only") == "true",
		DueAfter:    r.URL.Query().Get("due_after"),
		Unscheduled: r.URL.Query().Get("unscheduled") == "true",
		DueBefore:   r.URL.Query().Get("due_before"),
		DueOn:       r.URL.Query().Get("due_on"),
		Query:       r.URL.Query().Get("q"),
		Status:      r.URL.Query().Get("status"),
		Priority:    r.URL.Query().Get("priority"),
		AssigneeID:  r.URL.Query().Get("assignee_id"),
		RelatedType: r.URL.Query().Get("related_type"),
		RelatedID:   r.URL.Query().Get("related_id"),
		ProjectID:   r.URL.Query().Get("project_id"),
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

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var in tasks.UpdateInput
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

func (h *TaskHandler) Archive(w http.ResponseWriter, r *http.Request) {
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

func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	id := r.PathValue("id")
	if id == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	if err := h.svc.Complete(r.Context(), actor, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) handleError(w http.ResponseWriter, err error) {
	if writeKnownServiceError(w, err) {
		return
	}
	msg := err.Error()
	switch {
	case msg == "title required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	default:
		h.log.Error("task handler error", "error", err)
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

var _ = strconv.Atoi

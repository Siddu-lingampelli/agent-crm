// Package httpserver — leads handler.
package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/leads"
)

// LeadHandler wraps the lead service for HTTP.
type LeadHandler struct {
	svc *leads.Service
	log *slog.Logger
}

// NewLeadHandler returns a handler for lead endpoints.
func NewLeadHandler(svc *leads.Service, log *slog.Logger) *LeadHandler {
	return &LeadHandler{svc: svc, log: log}
}

func (h *LeadHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in leads.CreateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.Create(r.Context(), actor, in)
	if err != nil {
		h.log.Error("lead create failed", "error", err)
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *LeadHandler) Get(w http.ResponseWriter, r *http.Request) {
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

func (h *LeadHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := leads.ListInput{
		Query:    r.URL.Query().Get("q"),
		Status:   r.URL.Query().Get("status"),
		Source:   r.URL.Query().Get("source"),
		OwnerID:  r.URL.Query().Get("owner_id"),
		Priority: r.URL.Query().Get("priority"),
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

func (h *LeadHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var in leads.UpdateInput
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

func (h *LeadHandler) Archive(w http.ResponseWriter, r *http.Request) {
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

func (h *LeadHandler) handleError(w http.ResponseWriter, err error) {
	if writeKnownServiceError(w, err) {
		return
	}
	msg := err.Error()
	switch {
	case msg == "display name required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	case msg == "lead already converted":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	default:
		h.log.Error("lead handler error", "error", err)
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func (h *LeadHandler) Convert(w http.ResponseWriter, r *http.Request) {
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
	var in leads.ConvertInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.Convert(r.Context(), actor, id, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

var _ = strconv.Atoi

// Package httpserver — deals handler.
package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/deals"
)

// DealHandler wraps the deal service for HTTP.
type DealHandler struct {
	svc *deals.Service
	log *slog.Logger
}

// NewDealHandler returns a handler for deal endpoints.
func NewDealHandler(svc *deals.Service, log *slog.Logger) *DealHandler {
	return &DealHandler{svc: svc, log: log}
}

func (h *DealHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in deals.CreateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.Create(r.Context(), actor, in)
	if err != nil {
		h.log.Error("deal create failed", "error", err)
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *DealHandler) Get(w http.ResponseWriter, r *http.Request) {
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

func (h *DealHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := deals.ListInput{
		Category:       r.URL.Query().Get("category"),
		CloseFrom:      r.URL.Query().Get("close_from"),
		CloseBefore:    r.URL.Query().Get("close_before"),
		InactiveBefore: r.URL.Query().Get("inactive_before"),
		Query:          r.URL.Query().Get("q"),
		PipelineID:     r.URL.Query().Get("pipeline_id"),
		StageID:        r.URL.Query().Get("stage_id"),
		OwnerID:        r.URL.Query().Get("owner_id"),
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

func (h *DealHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var in deals.UpdateInput
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

func (h *DealHandler) Archive(w http.ResponseWriter, r *http.Request) {
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

func (h *DealHandler) handleError(w http.ResponseWriter, err error) {
	if writeKnownServiceError(w, err) {
		return
	}
	msg := err.Error()
	switch {
	case msg == "name required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	default:
		h.log.Error("deal handler error", "error", err)
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

var _ = strconv.Atoi

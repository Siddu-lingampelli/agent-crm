// Package httpserver — REST handlers for emails.
package httpserver

import (
	"net/http"
	"strconv"

	"crm.local/internal/jsonutil"
	"crm.local/internal/model"
	"crm.local/internal/service/email"
)

// EmailHandler wraps the email service for HTTP.
type EmailHandler struct {
	svc *email.Service
}

// NewEmailHandler returns a handler for email endpoints.
func NewEmailHandler(svc *email.Service) *EmailHandler {
	return &EmailHandler{svc: svc}
}

func (h *EmailHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in email.CreateDraftInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.CreateDraft(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *EmailHandler) Get(w http.ResponseWriter, r *http.Request) {
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

func (h *EmailHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := email.ListInput{
		Query:       r.URL.Query().Get("q"),
		Status:      r.URL.Query().Get("status"),
		RelatedType: r.URL.Query().Get("related_type"),
		RelatedID:   r.URL.Query().Get("related_id"),
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

func (h *EmailHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var in email.UpdateInput
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

func (h *EmailHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.Delete(r.Context(), actor, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EmailHandler) Approve(w http.ResponseWriter, r *http.Request) {
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
	out, err := h.svc.Approve(r.Context(), actor, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *EmailHandler) Send(w http.ResponseWriter, r *http.Request) {
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
	out, err := h.svc.Send(r.Context(), actor, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *EmailHandler) RetryFailed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	count, err := h.svc.RetryFailed(r.Context(), actor)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]int{"retried": count})
}

func (h *EmailHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in email.CreateTemplateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.CreateTemplate(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *EmailHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
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
	out, err := h.svc.GetTemplate(r.Context(), actor, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *EmailHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	out, err := h.svc.ListTemplates(r.Context(), actor)
	if err != nil {
		h.handleError(w, err)
		return
	}
	if out == nil {
		out = []model.EmailTemplate{}
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"items": out})
}

func (h *EmailHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	var in email.UpdateTemplateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, 400, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.UpdateTemplate(r.Context(), getAuth(r), r.PathValue("id"), in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, 200, out)
}
func (h *EmailHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonutil.Error(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	if err := h.svc.ArchiveTemplate(r.Context(), getAuth(r), r.PathValue("id")); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *EmailHandler) ApplyTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		jsonutil.Error(w, http.StatusBadRequest, "missing_id", "id required")
		return
	}
	var in email.ApplyTemplateInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.ApplyTemplate(r.Context(), actor, templateID, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *EmailHandler) handleError(w http.ResponseWriter, err error) {
	if writeKnownServiceError(w, err) {
		return
	}
	msg := err.Error()
	switch {
	case msg == "recipient required", msg == "subject required", msg == "body required",
		msg == "name required", msg == "can only update drafts", msg == "can only delete drafts",
		msg == "can only approve drafts", msg == "can only send draft or queued emails",
		msg == "email requires human approval before sending":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

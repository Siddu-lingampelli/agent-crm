// Package httpserver — REST handlers for agents and API keys.
package httpserver

import (
	"fmt"
	"net/http"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/agents"
)

// AgentHandler wraps the agent service for HTTP.
type AgentHandler struct {
	svc *agents.Service
}

// NewAgentHandler returns a handler for agent endpoints.
func NewAgentHandler(svc *agents.Service) *AgentHandler {
	return &AgentHandler{svc: svc}
}

// Agent endpoints
func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in agents.CreateAgentInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.CreateAgent(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, out)
}

func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	out, err := h.svc.GetAgent(r.Context(), actor, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := agents.ListAgentInput{
		Query:   r.URL.Query().Get("q"),
		Enabled: parseBoolPtr(r.URL.Query().Get("enabled")),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		in.Limit = parseInt(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		in.Offset = parseInt(v)
	}
	in.Sort = r.URL.Query().Get("sort")
	in.Order = r.URL.Query().Get("order")
	page, err := h.svc.ListAgents(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, page)
}

func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var in agents.UpdateAgentInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.UpdateAgent(r.Context(), actor, id, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *AgentHandler) Archive(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.ArchiveAgent(r.Context(), actor, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentHandler) Restore(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.RestoreAgent(r.Context(), actor, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// API Key endpoints
func (h *AgentHandler) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	var in agents.CreateApiKeyInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	rawKey, out, err := h.svc.CreateApiKey(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusCreated, map[string]any{
		"key":     rawKey,
		"api_key": out,
	})
}

func (h *AgentHandler) GetApiKey(w http.ResponseWriter, r *http.Request) {
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
	out, err := h.svc.GetApiKey(r.Context(), actor, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *AgentHandler) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	in := agents.ListApiKeyInput{
		Query:   r.URL.Query().Get("q"),
		AgentID: r.URL.Query().Get("agent_id"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		in.Limit = parseInt(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		in.Offset = parseInt(v)
	}
	in.Sort = r.URL.Query().Get("sort")
	in.Order = r.URL.Query().Get("order")
	page, err := h.svc.ListApiKeys(r.Context(), actor, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, page)
}

func (h *AgentHandler) UpdateApiKey(w http.ResponseWriter, r *http.Request) {
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
	var in agents.UpdateApiKeyInput
	if err := jsonutil.Decode(r, jsonutil.MaxBodyBytes, &in); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	out, err := h.svc.UpdateApiKey(r.Context(), actor, id, in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	jsonutil.Write(w, http.StatusOK, out)
}

func (h *AgentHandler) RevokeApiKey(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.RevokeApiKey(r.Context(), actor, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentHandler) handleError(w http.ResponseWriter, err error) {
	if writeKnownServiceError(w, err) {
		return
	}
	msg := err.Error()
	switch {
	case msg == "name required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "agent_id required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "at least one scope required":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "agent is disabled":
		jsonutil.Error(w, http.StatusBadRequest, "validation_failed", msg)
	case msg == "agent not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	case msg == "not found":
		jsonutil.Error(w, http.StatusNotFound, "not_found", msg)
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func parseBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	b := s == "true" || s == "1"
	return &b
}

func parseInt(s string) int {
	var i int
	_, _ = fmt.Sscanf(s, "%d", &i)
	return i
}

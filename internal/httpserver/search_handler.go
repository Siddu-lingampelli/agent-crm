// Package httpserver — search handler.
package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"crm.local/internal/jsonutil"
	"crm.local/internal/service/search"
)

// SearchHandler wraps the search service for HTTP.
type SearchHandler struct {
	svc *search.Service
	log *slog.Logger
}

// NewSearchHandler returns a handler for search endpoints.
func NewSearchHandler(svc *search.Service, log *slog.Logger) *SearchHandler {
	return &SearchHandler{svc: svc, log: log}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)

	query := r.URL.Query().Get("q")
	if query == "" {
		jsonutil.Write(w, http.StatusOK, search.SearchResponse{Results: nil, Total: 0, Query: "", Limit: 10})
		return
	}

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	entities := []string{"companies", "contacts", "leads", "deals", "tasks", "projects"}
	if v := r.URL.Query().Get("entities"); v != "" {
		entities = strings.Split(v, ",")
	}

	opts := search.SearchOptions{
		Query:     query,
		Entities:  entities,
		Limit:     limit,
		Workspace: actor.WorkspaceID,
		Actor:     actor,
	}

	resp, err := h.svc.Search(r.Context(), opts)
	if err != nil {
		h.log.Error("search failed", "error", err)
		jsonutil.Error(w, http.StatusInternalServerError, "search_failed", "search failed")
		return
	}

	jsonutil.Write(w, http.StatusOK, resp)
}

// Package search implements global full-text search across CRM entities.
package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"crm.local/internal/permissions"
)

// SearchResult represents a single search hit.
type SearchResult struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Title      string         `json:"title"`
	Subtitle   string         `json:"subtitle"`
	URL        string         `json:"url"`
	Score      float64        `json:"score"`
	Metadata   map[string]any `json:"metadata"`
}

// SearchResponse is the search results envelope.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
	Limit   int            `json:"limit"`
}

// Service provides full-text search.
type Service struct {
	db *sql.DB
}

// New returns a search service.
func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// SearchOptions configures the search.
type SearchOptions struct {
	Query     string
	Entities  []string // "companies", "contacts", "leads", "deals", "tasks", "projects"
	Limit     int
	Actor     permissions.AuthContext
	Workspace string
}

// Search performs a global full-text search across entities.
func (s *Service) Search(ctx context.Context, opts SearchOptions) (SearchResponse, error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Limit > 50 {
		opts.Limit = 50
	}
	if opts.Query == "" {
		return SearchResponse{Results: []SearchResult{}, Total: 0, Query: "", Limit: opts.Limit}, nil
	}

	allResults := []SearchResult{}
	entities := opts.Entities
	if len(entities) == 0 {
		entities = []string{"companies", "contacts", "leads", "deals", "tasks", "projects"}
	}

	for _, entity := range entities {
		if err := permissions.Authorize(opts.Actor, entity+":read"); err != nil {
			continue
		}
		entityResults, err := s.searchEntity(ctx, entity, opts.Query, opts.Actor.WorkspaceID, opts.Limit)
		if err != nil {
			continue // Skip failing entity, log in production
		}
		allResults = append(allResults, entityResults...)
	}

	// Sort by score descending
	sortByScore(allResults)

	if len(allResults) > opts.Limit {
		allResults = allResults[:opts.Limit]
	}

	return SearchResponse{
		Results: allResults,
		Total:   len(allResults),
		Query:   opts.Query,
		Limit:   opts.Limit,
	}, nil
}

func (s *Service) searchEntity(ctx context.Context, entity, query, workspace string, limit int) ([]SearchResult, error) {
	var sqlText string
	var args []any

	// Build FTS query with wildcard suffix for prefix matching
	words := strings.Fields(query)
	if len(words) == 0 {
		return []SearchResult{}, nil
	}
	for i, word := range words {
		words[i] = "\"" + strings.ReplaceAll(word, "\"", "\"\"") + "\"*"
	}
	ftsQuery := strings.Join(words, " AND ")

	switch entity {
	case "companies":
		sqlText = `SELECT c.id, c.name, COALESCE(c.domain || ' ' || c.industry, ''), 'companies', 1.0
			FROM fts_companies f
			JOIN companies c ON c.rowid = f.rowid
			WHERE fts_companies MATCH ? AND c.workspace_id = ? AND c.archived_at IS NULL
			ORDER BY bm25(fts_companies) LIMIT ?`
		args = []any{ftsQuery, workspace, limit}
	case "contacts":
		sqlText = `SELECT c.id, c.first_name || ' ' || COALESCE(c.last_name, ''), COALESCE(c.email || ' ' || c.job_title, ''), 'contacts', 1.0
			FROM fts_contacts f
			JOIN contacts c ON c.rowid = f.rowid
			WHERE fts_contacts MATCH ? AND c.workspace_id = ? AND c.archived_at IS NULL
			ORDER BY bm25(fts_contacts) LIMIT ?`
		args = []any{ftsQuery, workspace, limit}
	case "leads":
		sqlText = `SELECT l.id, l.display_name, COALESCE(l.email || ' ' || l.company_name, ''), 'leads', 1.0
			FROM fts_leads f
			JOIN leads l ON l.rowid = f.rowid
			WHERE fts_leads MATCH ? AND l.workspace_id = ? AND l.archived_at IS NULL
			ORDER BY bm25(fts_leads) LIMIT ?`
		args = []any{ftsQuery, workspace, limit}
	case "deals":
		sqlText = `SELECT d.id, d.name, COALESCE(d.description, ''), 'deals', 1.0
			FROM fts_deals f
			JOIN deals d ON d.rowid = f.rowid
			WHERE fts_deals MATCH ? AND d.workspace_id = ? AND d.archived_at IS NULL
			ORDER BY bm25(fts_deals) LIMIT ?`
		args = []any{ftsQuery, workspace, limit}
	case "tasks":
		sqlText = `SELECT t.id, t.title, COALESCE(t.description, ''), 'tasks', 1.0
			FROM fts_tasks f
			JOIN tasks t ON t.rowid = f.rowid
			WHERE fts_tasks MATCH ? AND t.workspace_id = ? AND t.archived_at IS NULL
			ORDER BY bm25(fts_tasks) LIMIT ?`
		args = []any{ftsQuery, workspace, limit}
	case "projects":
		sqlText = `SELECT p.id, p.name, COALESCE(p.description, ''), 'projects', 1.0
			FROM fts_projects f
			JOIN projects p ON p.rowid = f.rowid
			WHERE fts_projects MATCH ? AND p.workspace_id = ? AND p.archived_at IS NULL
			ORDER BY bm25(fts_projects) LIMIT ?`
		args = []any{ftsQuery, workspace, limit}
	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id, title, subtitle, entityType string
		var score float64
		if err := rows.Scan(&id, &title, &subtitle, &entityType, &score); err != nil {
			continue
		}
		url := fmt.Sprintf("/%s/%s", entityType, id)
		if entityType == "companies" || entityType == "contacts" || entityType == "leads" || entityType == "deals" || entityType == "tasks" || entityType == "projects" {
			url = fmt.Sprintf("/api/v1/%s/%s", entityType, id)
		}
		results = append(results, SearchResult{
			EntityType: entityType,
			EntityID:   id,
			Title:      title,
			Subtitle:   subtitle,
			URL:        url,
			Score:      score,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func sortByScore(results []SearchResult) {
	// Simple bubble sort for small n
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

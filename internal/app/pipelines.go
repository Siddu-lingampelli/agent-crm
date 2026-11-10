package app

import (
	"net/http"

	"crm.local/internal/auth"
	"crm.local/internal/jsonutil"
	"crm.local/internal/model"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
)

// handlePipelines supplies workspace-scoped choices for the deal editor.
func (a *App) handlePipelines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := auth.GetAuth(r)
	if err := permissions.Authorize(actor, "deals:read"); err != nil {
		jsonutil.Error(w, http.StatusForbidden, "forbidden", "deal read access required")
		return
	}
	repo := repository.NewPipelineRepository(a.db.SQL())
	pipelines, err := repo.List(r.Context(), actor.WorkspaceID)
	if err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "pipelines_failed", "could not load pipelines")
		return
	}
	type choice struct {
		model.Pipeline
		Stages []model.PipelineStage `json:"stages"`
	}
	items := make([]choice, 0, len(pipelines))
	for _, pipeline := range pipelines {
		stages, err := repo.ListStages(r.Context(), pipeline.ID)
		if err != nil {
			jsonutil.Error(w, http.StatusInternalServerError, "pipelines_failed", "could not load pipeline stages")
			return
		}
		items = append(items, choice{Pipeline: pipeline, Stages: stages})
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

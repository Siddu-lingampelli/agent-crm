package repository

import (
	"context"
	"crm.local/internal/model"
	"database/sql"
	"errors"
	"fmt"
)

var ErrInvalidReference = errors.New("referenced record is unavailable in this workspace")

// SQLite foreign keys establish existence, but workspace ownership must also
// be checked before attaching a record to another entity.
type referenceQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func validateEntityReferences(ctx context.Context, db referenceQuerier, record any) error {
	if err := validateEntityInput(record); err != nil {
		return err
	}
	var workspace, pipeline, stage, relatedType, relatedID string
	refs := map[string]string{}
	switch v := record.(type) {
	case *model.Contact:
		workspace = v.WorkspaceID
		refs["companies"] = v.CompanyID
	case *model.Lead:
		workspace = v.WorkspaceID
		refs["companies"] = v.CompanyID
	case *model.Deal:
		workspace = v.WorkspaceID
		refs["companies"] = v.CompanyID
		refs["contacts"] = v.PrimaryContactID
		refs["leads"] = v.LeadID
		refs["pipelines"] = v.PipelineID
		pipeline = v.PipelineID
		stage = v.StageID
	case *model.Task:
		workspace = v.WorkspaceID
		refs["projects"] = v.ProjectID
		relatedType = v.RelatedType
		relatedID = v.RelatedID
	case *model.Project:
		workspace = v.WorkspaceID
		refs["companies"] = v.CompanyID
	case *model.CalendarEvent:
		workspace = v.WorkspaceID
		relatedType = v.RelatedType
		relatedID = v.RelatedID
	case *model.Email:
		workspace = v.WorkspaceID
		relatedType = v.RelatedType
		relatedID = v.RelatedID
	default:
		return nil
	}
	if relatedID != "" {
		table := map[string]string{"company": "companies", "companies": "companies", "contact": "contacts", "contacts": "contacts", "lead": "leads", "leads": "leads", "deal": "deals", "deals": "deals", "task": "tasks", "tasks": "tasks", "project": "projects", "projects": "projects"}[relatedType]
		if table == "" {
			return ErrInvalidReference
		}
		refs[table] = relatedID
	}
	for table, id := range refs {
		if id == "" {
			continue
		}
		query := "SELECT COUNT(*) FROM " + table + " WHERE id = ? AND workspace_id = ?"
		if table != "pipelines" {
			query += " AND archived_at IS NULL"
		}
		var count int
		if err := db.QueryRowContext(ctx, query, id, workspace).Scan(&count); err != nil {
			return fmt.Errorf("validate reference: %w", err)
		}
		if count != 1 {
			return ErrInvalidReference
		}
	}
	if stage != "" {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pipeline_stages WHERE id = ? AND pipeline_id = ?", stage, pipeline).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return ErrInvalidReference
		}
	}
	return nil
}

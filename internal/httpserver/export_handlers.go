// Package httpserver — REST handlers for CSV export/import per entity.
package httpserver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"crm.local/internal/jsonutil"
	"crm.local/internal/permissions"
	"crm.local/internal/service/calendar"
	"crm.local/internal/service/companies"
	"crm.local/internal/service/contacts"
	"crm.local/internal/service/deals"
	"crm.local/internal/service/leads"
	"crm.local/internal/service/projects"
	"crm.local/internal/service/tasks"
)

// ExportHandler wraps the CSV export/import logic for one entity type.
type ExportHandler struct {
	svcName  string
	listFn   func(ctx context.Context, actor permissions.AuthContext) ([]any, error)
	createFn func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error
	fields   []string
}

// NewExportHandler returns a handler for export/import of one entity type.
func NewExportHandler(
	svcName string,
	listFn func(ctx context.Context, actor permissions.AuthContext) ([]any, error),
	createFn func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error,
	fields []string,
) *ExportHandler {
	return &ExportHandler{svcName: svcName, listFn: listFn, createFn: createFn, fields: fields}
}

// ExportCSV streams all rows of this entity as CSV.
func (h *ExportHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	if actor.ActorType != "user" {
		jsonutil.Error(w, http.StatusForbidden, "forbidden", "export requires a human user session")
		return
	}
	rows, err := h.listFn(r.Context(), actor)
	if err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-%s.csv\"", h.svcName, time.Now().UTC().Format("20060102-150405")))
	cw := csv.NewWriter(w)
	defer cw.Flush()
	// Header row
	if err := cw.Write(h.fields); err != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "cannot write CSV header")
		return
	}
	// Data rows
	for _, row := range rows {
		m, err := structToMap(row)
		if err != nil {
			continue
		}
		record := make([]string, len(h.fields))
		for i, field := range h.fields {
			v := m[field]
			record[i] = fmt.Sprintf("%v", v)
		}
		if err := cw.Write(record); err != nil {
			jsonutil.Error(w, http.StatusInternalServerError, "internal_error", "cannot write CSV row")
			return
		}
	}
}

// ImportCSV reads a CSV upload and creates each row via the service.
func (h *ExportHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonutil.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	actor := getAuth(r)
	if actor.ActorType != "user" {
		jsonutil.Error(w, http.StatusForbidden, "forbidden", "import requires a human user session")
		return
	}
	// Parse multipart form (max 10 MiB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", "multipart form parse failed")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_body", "missing 'file' field")
		return
	}
	defer file.Close()
	// Basic validation: must end in .csv
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_file", "file must be a CSV (.csv)")
		return
	}
	// Read and parse CSV
	cr := csv.NewReader(file)
	headerRow, err := cr.Read()
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid_csv", "cannot read CSV header: "+err.Error())
		return
	}
	// Build column index
	colIndex := make(map[string]int)
	for i, name := range headerRow {
		colIndex[strings.TrimSpace(name)] = i
	}
	// Validate required fields are present
	for _, field := range h.fields {
		if _, ok := colIndex[field]; !ok {
			jsonutil.Error(w, http.StatusBadRequest, "invalid_csv", fmt.Sprintf("missing column: %s", field))
			return
		}
	}
	var imported, skipped int
	var lastErr error
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			lastErr = fmt.Errorf("CSV row parse error: %w", err)
			break
		}
		// Build map from record
		row := make(map[string]any)
		for i, field := range h.fields {
			if i < len(record) {
				row[field] = record[i]
			}
		}
		// Create via service
		if err := h.createFn(r.Context(), actor, row); err != nil {
			lastErr = fmt.Errorf("failed to create row: %w", err)
			skipped++
			continue
		}
		imported++
	}
	if lastErr != nil {
		jsonutil.Error(w, http.StatusInternalServerError, "import_failed", fmt.Sprintf("imported %d, skipped %d: %v", imported, skipped, lastErr))
		return
	}
	jsonutil.Write(w, http.StatusOK, map[string]any{
		"imported": imported,
		"skipped":  skipped,
	})
}

// Helper: convert struct to map via JSON round-trip.
func structToMap(s any) (map[string]any, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// --- Constructors for each entity ---

func NewCompaniesExportHandler(companySvc *companies.Service) *ExportHandler {
	return NewExportHandler(
		"companies",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := companySvc.List(ctx, actor, companies.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := companySvc.Create(ctx, actor, companies.CreateInput{
				Name:        getString(row, "name"),
				Domain:      getString(row, "domain"),
				Industry:    getString(row, "industry"),
				SizeBand:    getString(row, "size_band"),
				Website:     getString(row, "website"),
				Phone:       getString(row, "phone"),
				Email:       getString(row, "email"),
				Address:     getString(row, "address"),
				IsClient:    getBool(row, "is_client"),
				Status:      getString(row, "status"),
				OwnerID:     getString(row, "owner_id"),
				Tags:        getSlice(row, "tags"),
				Notes:       getString(row, "notes"),
			})
			return err
		},
		[]string{"name", "domain", "industry", "size_band", "website", "phone", "email", "address", "is_client", "status", "owner_id", "tags", "notes"},
	)
}

func NewContactsExportHandler(contactSvc *contacts.Service) *ExportHandler {
	return NewExportHandler(
		"contacts",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := contactSvc.List(ctx, actor, contacts.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := contactSvc.Create(ctx, actor, contacts.CreateInput{
				FirstName: getString(row, "first_name"),
				LastName:  getString(row, "last_name"),
				Email:     getString(row, "email"),
				Phone:     getString(row, "phone"),
				JobTitle:  getString(row, "job_title"),
				CompanyID: getString(row, "company_id"),
				OwnerID:   getString(row, "owner_id"),
				Status:    getString(row, "status"),
				Tags:      getSlice(row, "tags"),
				Notes:     getString(row, "notes"),
			})
			return err
		},
		[]string{"first_name", "last_name", "email", "phone", "job_title", "company_id", "owner_id", "status", "tags", "notes"},
	)
}

func NewLeadsExportHandler(leadSvc *leads.Service) *ExportHandler {
	return NewExportHandler(
		"leads",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := leadSvc.List(ctx, actor, leads.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := leadSvc.Create(ctx, actor, leads.CreateInput{
				DisplayName:    getString(row, "display_name"),
				FirstName:      getString(row, "first_name"),
				LastName:       getString(row, "last_name"),
				Email:          getString(row, "email"),
				Phone:          getString(row, "phone"),
				CompanyName:    getString(row, "company_name"),
				CompanyID:      getString(row, "company_id"),
				JobTitle:       getString(row, "job_title"),
				Status:         getString(row, "status"),
				Source:         getString(row, "source"),
				OwnerID:        getString(row, "owner_id"),
				Priority:       getString(row, "priority"),
				Currency:       getString(row, "currency"),
				ExpectedClose:  getString(row, "expected_close"),
				FollowUpDate:   getString(row, "follow_up_date"),
				Tags:           getSlice(row, "tags"),
				Description:    getString(row, "description"),
				Qualification:  getString(row, "qualification"),
			})
			return err
		},
		[]string{"display_name", "first_name", "last_name", "email", "phone", "company_name", "company_id", "job_title", "status", "source", "owner_id", "priority", "currency", "expected_close", "follow_up_date", "tags", "description", "qualification"},
	)
}

func NewDealsExportHandler(dealSvc *deals.Service) *ExportHandler {
	return NewExportHandler(
		"deals",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := dealSvc.List(ctx, actor, deals.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := dealSvc.Create(ctx, actor, deals.CreateInput{
				Name:             getString(row, "name"),
				PipelineID:       getString(row, "pipeline_id"),
				StageID:          getString(row, "stage_id"),
				Currency:         getString(row, "currency"),
				Probability:      getInt(row, "probability"),
				ExpectedClose:    getString(row, "expected_close"),
				CompanyID:        getString(row, "company_id"),
				PrimaryContactID: getString(row, "primary_contact_id"),
				LeadID:           getString(row, "lead_id"),
				OwnerID:          getString(row, "owner_id"),
				Source:           getString(row, "source"),
				Description:      getString(row, "description"),
				LostReason:       getString(row, "lost_reason"),
			})
			return err
		},
		[]string{"name", "pipeline_id", "stage_id", "currency", "probability", "expected_close", "company_id", "primary_contact_id", "lead_id", "owner_id", "source", "description", "lost_reason"},
	)
}

func NewTasksExportHandler(taskSvc *tasks.Service) *ExportHandler {
	return NewExportHandler(
		"tasks",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := taskSvc.List(ctx, actor, tasks.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := taskSvc.Create(ctx, actor, tasks.CreateInput{
				Title:       getString(row, "title"),
				Description: getString(row, "description"),
				Status:      getString(row, "status"),
				Priority:    getString(row, "priority"),
				DueDate:     getString(row, "due_date"),
				AssigneeID:  getString(row, "assignee_id"),
				RelatedType: getString(row, "related_type"),
				RelatedID:   getString(row, "related_id"),
				ProjectID:   getString(row, "project_id"),
				CreatedBy:   getString(row, "created_by"),
			})
			return err
		},
		[]string{"title", "description", "status", "priority", "due_date", "assignee_id", "related_type", "related_id", "project_id", "created_by"},
	)
}

func NewProjectsExportHandler(projectSvc *projects.Service) *ExportHandler {
	return NewExportHandler(
		"projects",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := projectSvc.List(ctx, actor, projects.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := projectSvc.Create(ctx, actor, projects.CreateInput{
				Name:        getString(row, "name"),
				Description: getString(row, "description"),
				Status:      getString(row, "status"),
				Priority:    getString(row, "priority"),
				OwnerID:     getString(row, "owner_id"),
				CompanyID:   getString(row, "company_id"),
				StartDate:   getString(row, "start_date"),
				EndDate:     getString(row, "end_date"),
				Progress:    getInt(row, "progress"),
			})
			return err
		},
		[]string{"name", "description", "status", "priority", "owner_id", "company_id", "start_date", "end_date", "progress"},
	)
}

func NewCalendarExportHandler(calendarSvc *calendar.Service) *ExportHandler {
	return NewExportHandler(
		"calendar",
		func(ctx context.Context, actor permissions.AuthContext) ([]any, error) {
			page, err := calendarSvc.List(ctx, actor, calendar.ListInput{WorkspaceID: actor.WorkspaceID, Limit: 100000})
			if err != nil {
				return nil, err
			}
			out := make([]any, len(page.Items))
			for i, v := range page.Items {
				out[i] = v
			}
			return out, nil
		},
		func(ctx context.Context, actor permissions.AuthContext, row map[string]any) error {
			_, err := calendarSvc.Create(ctx, actor, calendar.CreateInput{
				Title:       getString(row, "title"),
				Description: getString(row, "description"),
				StartTime:   getString(row, "start_time"),
				EndTime:     getString(row, "end_time"),
				AllDay:      getBool(row, "all_day"),
				Location:    getString(row, "location"),
				MeetingURL:  getString(row, "meeting_url"),
				AttendeeIDs: getSlice(row, "attendee_ids"),
				RelatedType: getString(row, "related_type"),
				RelatedID:   getString(row, "related_id"),
				Status:      getString(row, "status"),
			})
			return err
		},
		[]string{"title", "description", "start_time", "end_time", "all_day", "location", "meeting_url", "attendee_ids", "related_type", "related_id", "status"},
	)
}

// Helper functions for extracting values from row maps.
func getString(row map[string]any, field string) string {
	if v, ok := row[field]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getBool(row map[string]any, field string) bool {
	if v, ok := row[field]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			b, _ := strconv.ParseBool(val)
			return b
		}
	}
	return false
}

func getInt(row map[string]any, field string) int {
	if v, ok := row[field]; ok {
		switch val := v.(type) {
		case int:
			return val
		case string:
			n, _ := strconv.Atoi(val)
			return n
		}
	}
	return 0
}

func getSlice(row map[string]any, field string) []string {
	if v, ok := row[field]; ok {
		switch val := v.(type) {
		case []string:
			return val
		case string:
			// Comma-separated
			parts := strings.Split(val, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
	}
	return nil
}
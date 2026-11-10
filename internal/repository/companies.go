// Package repository — companies.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
)

// CompanyListOptions extends ListOptions with company-specific filters.
type CompanyListOptions struct {
	WorkspaceID string
	Query       string
	IsClient    *bool
	Status      string
	CreatedFrom string
	Limit       int
	Offset      int
	Sort        string
	Order       string
}

// companiesColumns is the canonical column order used by every query.
const companiesColumns = `
    id, workspace_id, name, domain, industry, size_band, website, phone, email,
    address, is_client, status, owner_id, tags, custom_fields, notes,
    created_at, updated_at, archived_at
`

// CompanyRepository persists companies.
type CompanyRepository struct {
	db *sql.DB
}

// NewCompanyRepository returns a repository over db.
func NewCompanyRepository(db *sql.DB) *CompanyRepository { return &CompanyRepository{db: db} }

// Create inserts a company.
func (r *CompanyRepository) Create(ctx context.Context, c *model.Company) error {
	if err := validateEntityInput(c); err != nil {
		return err
	}
	tags := jsonOrDefault(c.Tags)
	custom := jsonOrDefaultMap(c.CustomFields)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO companies (
            id, workspace_id, name, domain, industry, size_band, website, phone, email,
            address, is_client, status, owner_id, tags, custom_fields, notes,
            created_at, updated_at, archived_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.Name, c.Domain, c.Industry, c.SizeBand, c.Website,
		c.Phone, c.Email, c.Address, c.IsClient, c.Status, c.OwnerID, tags, custom,
		c.Notes, c.CreatedAt, c.UpdatedAt, c.ArchivedAt)
	if err != nil {
		return fmt.Errorf("insert company: %w", err)
	}
	return nil
}

// Get returns a company by id, or nil if not found.
func (r *CompanyRepository) Get(ctx context.Context, id string) (*model.Company, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+companiesColumns+` FROM companies WHERE id = ? AND archived_at IS NULL`, id)
	c, err := scanCompany(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// List returns a page of companies with server-side pagination.
func (r *CompanyRepository) List(ctx context.Context, opts CompanyListOptions) (Page[model.Company], error) {
	where, args := companyFilters(opts)

	// Real total, independent of the page.
	var total int
	countSQL := `SELECT COUNT(*) FROM companies` + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.Company]{}, fmt.Errorf("count companies: %w", err)
	}

	sort := companySort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := max(opts.Offset, 0)

	// sort and order are validated by companySort and NormalizeOrder, safe to use in query
	//nolint:gosec // G202: sort and order are validated by companySort and NormalizeOrder
	sqlText := `SELECT ` + companiesColumns + ` FROM companies` + where +
		` ORDER BY ` + sort + ` ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.Company]{}, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	items := []model.Company{}
	for rows.Next() {
		item, err := scanCompanyRows(rows)
		if err != nil {
			return Page[model.Company]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.Company]{}, fmt.Errorf("iterate companies: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

// Update modifies a company. It never writes fields that are not present.
// Caller passes the full struct; unchanged fields are written unchanged.
func (r *CompanyRepository) Update(ctx context.Context, c *model.Company) error {
	if err := validateEntityInput(c); err != nil {
		return err
	}
	tags := jsonOrDefault(c.Tags)
	custom := jsonOrDefaultMap(c.CustomFields)

	_, err := r.db.ExecContext(ctx, `
        UPDATE companies SET
            name = ?, domain = ?, industry = ?, size_band = ?, website = ?,
            phone = ?, email = ?, address = ?, is_client = ?, status = ?,
            owner_id = ?, tags = ?, custom_fields = ?, notes = ?,
            updated_at = ?, archived_at = ?
        WHERE id = ? AND archived_at IS NULL`,
		c.Name, c.Domain, c.Industry, c.SizeBand, c.Website, c.Phone, c.Email,
		c.Address, c.IsClient, c.Status, c.OwnerID, tags, custom, c.Notes,
		c.UpdatedAt, c.ArchivedAt, c.ID)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}
	return nil
}

// Archive soft-deletes a company.
func (r *CompanyRepository) Archive(ctx context.Context, id string, now string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE companies SET archived_at = ? WHERE id = ? AND archived_at IS NULL`, now, id)
	if err != nil {
		return fmt.Errorf("archive company: %w", err)
	}
	return nil
}

// companyFilters builds a safe WHERE clause. All user input is bound.
func companyFilters(opts CompanyListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	conditions = append(conditions, "archived_at IS NULL")

	if opts.Query != "" {
		conditions = append(conditions, `(name LIKE ? OR domain LIKE ? OR industry LIKE ?)`)
		term := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, term, term, term)
	}
	if opts.IsClient != nil {
		conditions = append(conditions, "is_client = ?")
		args = append(args, *opts.IsClient)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.CreatedFrom != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, opts.CreatedFrom)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// companySort maps a client sort key to a safe column. Never trusts input.
func companySort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return "created_at"
	case "updated_at", "updated":
		return "updated_at"
	case "name":
		return "name"
	case "domain":
		return "domain"
	case "industry":
		return "industry"
	default:
		return "created_at"
	}
}

// scanCompany reads one row from row (used by Get).
func scanCompany(row *sql.Row) (*model.Company, error) {
	c := &model.Company{}
	var tagsJSON, customJSON string
	var isClient int
	var createdAt, updatedAt string
	var archivedAt *string
	if err := row.Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Domain, &c.Industry, &c.SizeBand,
		&c.Website, &c.Phone, &c.Email, &c.Address, &isClient, &c.Status,
		&c.OwnerID, &tagsJSON, &customJSON, &c.Notes, &createdAt, &updatedAt,
		&archivedAt); err != nil {
		return nil, err
	}
	c.IsClient = isClient == 1
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		c.ArchivedAt = &t
	}
	c.Tags = []string{}
	_ = json.Unmarshal([]byte(tagsJSON), &c.Tags)
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	return c, nil
}

// scanCompanyRows reads one row from *sql.Rows (used by List).
func scanCompanyRows(rows *sql.Rows) (*model.Company, error) {
	c := &model.Company{}
	var tagsJSON, customJSON string
	var isClient int
	var createdAt, updatedAt string
	var archivedAt *string
	if err := rows.Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Domain, &c.Industry, &c.SizeBand,
		&c.Website, &c.Phone, &c.Email, &c.Address, &isClient, &c.Status,
		&c.OwnerID, &tagsJSON, &customJSON, &c.Notes, &createdAt, &updatedAt,
		&archivedAt); err != nil {
		return nil, err
	}
	c.IsClient = isClient == 1
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	if archivedAt != nil {
		t := parseTime(*archivedAt)
		c.ArchivedAt = &t
	}
	c.Tags = []string{}
	_ = json.Unmarshal([]byte(tagsJSON), &c.Tags)
	c.CustomFields = map[string]any{}
	_ = json.Unmarshal([]byte(customJSON), &c.CustomFields)
	return c, nil
}

// parseTime accepts API timestamps and legacy Go/SQLite driver timestamps.
func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999 -0700 MST", "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.Parse(layout, s); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func jsonOrDefault(v []string) string {
	if v == nil {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func jsonOrDefaultMap(v map[string]any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

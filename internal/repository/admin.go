// Package repository — admin.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"crm.local/internal/model"
)

// AdminRepository persists admin data.
type AdminRepository struct {
	db *sql.DB
}

// NewAdminRepository returns a repository over db.
func NewAdminRepository(db *sql.DB) *AdminRepository { return &AdminRepository{db: db} }

// Audit Logs

// AuditLogListOptions extends ListOptions with audit-specific filters.
type AuditLogListOptions struct {
	Query        string
	WorkspaceID  string
	ActorType    string
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	StartFrom    string
	EndBefore    string
	StartTo      string
	Limit        int
	Offset       int
	Sort         string
	Order        string
}

// SQLite cannot parse the legacy Go driver UTC suffix directly. Normalize it
// when reading; existing audit rows remain unchanged.
const auditTimestamp = "COALESCE(julianday(created_at), julianday(replace(replace(created_at, ' +0000 UTC', 'Z'), ' ', 'T')))"

const auditLogsColumns = `
    id, workspace_id, actor_type, actor_id, action, resource_type, resource_id,
    old_values, new_values, ip_address, user_agent, created_at
`

// CreateAuditLog creates an audit log entry.
func (r *AdminRepository) CreateAuditLog(ctx context.Context, a *model.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO audit_logs (
            id, workspace_id, actor_type, actor_id, action, resource_type, resource_id,
            old_values, new_values, ip_address, user_agent, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.WorkspaceID, a.ActorType, a.ActorID, a.Action, a.ResourceType, a.ResourceID,
		a.OldValues, a.NewValues, a.IPAddress, a.UserAgent, a.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// ListAuditLogs returns a page of audit logs.
func (r *AdminRepository) ListAuditLogs(ctx context.Context, opts AuditLogListOptions) (Page[model.AuditLog], error) {
	where, args := auditLogFilters(opts)

	var total int
	countSQL := `SELECT COUNT(*) FROM audit_logs` + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[model.AuditLog]{}, fmt.Errorf("count audit logs: %w", err)
	}

	sort := auditLogSort(opts.Sort)
	order := NormalizeOrder(opts.Order)
	limit := ClampLimit(opts.Limit)
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// sort and order are validated by auditLogSort and NormalizeOrder, safe to use in query
	//nolint:gosec // G202: sort and order are validated by auditLogSort and NormalizeOrder
	sqlText := `SELECT ` + auditLogsColumns + ` FROM audit_logs` + where +
		` ORDER BY ` + sort + ` ` + order + `, id ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return Page[model.AuditLog]{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	items := []model.AuditLog{}
	for rows.Next() {
		item, err := scanAuditLogRows(rows)
		if err != nil {
			return Page[model.AuditLog]{}, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[model.AuditLog]{}, fmt.Errorf("iterate audit logs: %w", err)
	}

	return NewPage(items, total, limit, offset), nil
}

func auditLogFilters(opts AuditLogListOptions) (string, []any) {
	var conditions []string
	var args []any

	if opts.WorkspaceID != "" {
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}

	if query := strings.TrimSpace(opts.Query); query != "" {
		conditions = append(conditions, "(id LIKE ? OR actor_id LIKE ? OR actor_type LIKE ? OR action LIKE ? OR resource_type LIKE ? OR resource_id LIKE ? OR ip_address LIKE ?)")
		term := "%" + query + "%"
		args = append(args, term, term, term, term, term, term, term)
	}

	if opts.ActorType != "" {
		conditions = append(conditions, "actor_type = ?")
		args = append(args, opts.ActorType)
	}

	if opts.ActorID != "" {
		conditions = append(conditions, "actor_id = ?")
		args = append(args, opts.ActorID)
	}

	if opts.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, opts.Action)
	}

	if opts.ResourceType != "" {
		conditions = append(conditions, "resource_type = ?")
		args = append(args, opts.ResourceType)
	}

	if opts.ResourceID != "" {
		conditions = append(conditions, "resource_id = ?")
		args = append(args, opts.ResourceID)
	}

	if opts.StartFrom != "" {
		conditions = append(conditions, auditTimestamp+" >= julianday(?)")
		args = append(args, opts.StartFrom)
	}

	if opts.EndBefore != "" {
		conditions = append(conditions, auditTimestamp+" < julianday(?)")
		args = append(args, opts.EndBefore)
	}

	if opts.StartTo != "" {
		conditions = append(conditions, auditTimestamp+" <= julianday(?)")
		args = append(args, opts.StartTo)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func auditLogSort(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at", "created":
		return auditTimestamp
	case "action":
		return "action"
	case "resource_type":
		return "resource_type"
	case "actor_type":
		return "actor_type"
	default:
		return auditTimestamp
	}
}

func scanAuditLogRows(rows *sql.Rows) (*model.AuditLog, error) {
	a := &model.AuditLog{}
	var createdAt string
	if err := rows.Scan(
		&a.ID, &a.WorkspaceID, &a.ActorType, &a.ActorID, &a.Action, &a.ResourceType, &a.ResourceID,
		&a.OldValues, &a.NewValues, &a.IPAddress, &a.UserAgent, &createdAt); err != nil {
		return nil, err
	}
	a.CreatedAt = parseTime(createdAt)
	return a, nil
}

// Settings

// GetSetting returns a setting by workspace, category, key.
func (r *AdminRepository) GetSetting(ctx context.Context, workspaceID, category, key string) (*model.Setting, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, category, key, value, description, created_at, updated_at FROM settings WHERE workspace_id = ? AND category = ? AND key = ?`,
		workspaceID, category, key)
	s, err := scanSetting(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// CreateSetting creates a setting.
func (r *AdminRepository) CreateSetting(ctx context.Context, s *model.Setting) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO settings (id, workspace_id, category, key, value, description, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.WorkspaceID, s.Category, s.Key, s.Value, s.Description, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert setting: %w", err)
	}
	return nil
}

// UpdateSetting updates a setting.
func (r *AdminRepository) UpdateSetting(ctx context.Context, s *model.Setting) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE settings SET value = ?, description = ?, updated_at = ? WHERE workspace_id = ? AND category = ? AND key = ?`,
		s.Value, s.Description, s.UpdatedAt, s.WorkspaceID, s.Category, s.Key)
	if err != nil {
		return fmt.Errorf("update setting: %w", err)
	}
	return nil
}

// DeleteSetting deletes a setting.
func (r *AdminRepository) DeleteSetting(ctx context.Context, workspaceID, category, key string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM settings WHERE workspace_id = ? AND category = ? AND key = ?`,
		workspaceID, category, key)
	if err != nil {
		return fmt.Errorf("delete setting: %w", err)
	}
	return nil
}

// ListSettings returns all settings for a workspace.
func (r *AdminRepository) ListSettings(ctx context.Context, workspaceID, category string) ([]model.Setting, error) {
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, workspace_id, category, key, value, description, created_at, updated_at FROM settings WHERE workspace_id = ? AND category = ? ORDER BY key`,
			workspaceID, category)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, workspace_id, category, key, value, description, created_at, updated_at FROM settings WHERE workspace_id = ? ORDER BY category, key`,
			workspaceID)
	}
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	var settings []model.Setting
	for rows.Next() {
		s, err := scanSettingRows(rows)
		if err != nil {
			return nil, err
		}
		settings = append(settings, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}
	return settings, nil
}

func scanSetting(row *sql.Row) (*model.Setting, error) {
	s := &model.Setting{}
	var createdAt, updatedAt string
	if err := row.Scan(&s.ID, &s.WorkspaceID, &s.Category, &s.Key, &s.Value, &s.Description, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return s, nil
}

func scanSettingRows(rows *sql.Rows) (*model.Setting, error) {
	s := &model.Setting{}
	var createdAt, updatedAt string
	if err := rows.Scan(&s.ID, &s.WorkspaceID, &s.Category, &s.Key, &s.Value, &s.Description, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return s, nil
}

// Schema

// SchemaTable represents a database table schema.
type SchemaTable struct {
	Name    string         `json:"name"`
	Columns []SchemaColumn `json:"columns"`
	Indexes []string       `json:"indexes"`
}

// SchemaColumn represents a column in a table.
type SchemaColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"not_null"`
	Default string `json:"default"`
	PK      bool   `json:"pk"`
}

// GetSchemaTables returns database schema information.
func (r *AdminRepository) GetSchemaTables(ctx context.Context) ([]SchemaTable, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	tables := []SchemaTable{}
	for _, tableName := range names {
		columns, err := r.getTableColumns(ctx, tableName)
		if err != nil {
			return nil, err
		}
		indexes, err := r.getTableIndexes(ctx, tableName)
		if err != nil {
			return nil, err
		}
		tables = append(tables, SchemaTable{
			Name:    tableName,
			Columns: columns,
			Indexes: indexes,
		})
	}
	return tables, nil
}

func (r *AdminRepository) getTableColumns(ctx context.Context, tableName string) ([]SchemaColumn, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT cid,name,type,"notnull",dflt_value,pk FROM pragma_table_info(?)`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []SchemaColumn
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, SchemaColumn{
			Name:    name,
			Type:    colType,
			NotNull: notNull == 1,
			Default: dfltValue.String,
			PK:      pk == 1,
		})
	}
	return columns, nil
}

func (r *AdminRepository) getTableIndexes(ctx context.Context, tableName string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT seq,name,"unique",origin,partial FROM pragma_index_list(?)`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}
		indexes = append(indexes, name)
	}
	return indexes, nil
}

// SchemaMigration represents an applied migration.
type SchemaMigration struct {
	Version   int    `json:"version"`
	Name      string `json:"name"`
	AppliedAt string `json:"applied_at"`
}

// GetSchemaMigrations returns applied migrations.
func (r *AdminRepository) GetSchemaMigrations(ctx context.Context) ([]SchemaMigration, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT version_id, CAST(version_id AS TEXT), tstamp FROM goose_db_version WHERE is_applied = 1 AND version_id > 0 ORDER BY version_id`)
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	defer rows.Close()

	var migrations []SchemaMigration
	for rows.Next() {
		var m SchemaMigration
		if err := rows.Scan(&m.Version, &m.Name, &m.AppliedAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}

// Stats

// GetWorkflowStats returns workflow statistics.
func (r *AdminRepository) GetWorkflowStats(ctx context.Context, workspaceID string) (map[string]any, error) {
	var total, active, inactive, draft int
	err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*),
            COALESCE(SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END),0),
            COALESCE(SUM(CASE WHEN status = 'inactive' THEN 1 ELSE 0 END),0),
            COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END),0)
        FROM workflows WHERE workspace_id = ?`, workspaceID).Scan(&total, &active, &inactive, &draft)
	if err != nil {
		return nil, fmt.Errorf("workflow stats: %w", err)
	}

	var totalRuns, totalSuccess, totalFailure int
	err = r.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(run_count), 0), COALESCE(SUM(success_count), 0), COALESCE(SUM(failure_count), 0)
        FROM workflows WHERE workspace_id = ?`, workspaceID).Scan(&totalRuns, &totalSuccess, &totalFailure)
	if err != nil {
		return nil, fmt.Errorf("workflow run stats: %w", err)
	}

	return map[string]any{
		"total":         total,
		"active":        active,
		"inactive":      inactive,
		"draft":         draft,
		"total_runs":    totalRuns,
		"total_success": totalSuccess,
		"total_failure": totalFailure,
	}, nil
}

// GetIntegrationStats returns integration statistics.
func (r *AdminRepository) GetIntegrationStats(ctx context.Context, workspaceID string) (map[string]any, error) {
	var total, connected, disconnected, errorCount int
	err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*),
            COALESCE(SUM(CASE WHEN status = 'connected' THEN 1 ELSE 0 END),0),
            COALESCE(SUM(CASE WHEN status = 'disconnected' THEN 1 ELSE 0 END),0),
            COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END),0)
        FROM integrations WHERE workspace_id = ?`, workspaceID).Scan(&total, &connected, &disconnected, &errorCount)
	if err != nil {
		return nil, fmt.Errorf("integration stats: %w", err)
	}

	return map[string]any{
		"total":        total,
		"connected":    connected,
		"disconnected": disconnected,
		"error":        errorCount,
	}, nil
}

// GetAgentStats returns agent statistics.
func (r *AdminRepository) GetAgentStats(ctx context.Context, workspaceID string) (map[string]any, error) {
	var total, enabled int
	err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*), COALESCE(SUM(CASE WHEN enabled = 1 THEN 1 ELSE 0 END),0)
        FROM agents WHERE workspace_id = ?`, workspaceID).Scan(&total, &enabled)
	if err != nil {
		return nil, fmt.Errorf("agent stats: %w", err)
	}

	var totalKeys, activeKeys int
	err = r.db.QueryRowContext(ctx, `
        SELECT COUNT(*), COALESCE(SUM(CASE WHEN revoked_at IS NULL AND (expires_at IS NULL OR expires_at > datetime('now')) THEN 1 ELSE 0 END),0)
        FROM api_keys WHERE workspace_id = ?`, workspaceID).Scan(&totalKeys, &activeKeys)
	if err != nil {
		return nil, fmt.Errorf("api key stats: %w", err)
	}

	return map[string]any{
		"total_agents":   total,
		"enabled_agents": enabled,
		"total_keys":     totalKeys,
		"active_keys":    activeKeys,
	}, nil
}

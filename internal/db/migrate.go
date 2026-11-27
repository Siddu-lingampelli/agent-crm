// Package db — migrations helpers.
//
// Migrations are embedded SQL files under db/migrations. goose creates the
// schema_migrations table. This file is the entry point; the SQL lives in
// db/migrations/*.sql.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"crm.local/db/migrations"
	"github.com/pressly/goose/v3"
)

// Migrate applies all pending migrations to the database.
//
// The goose schema_migrations table is created on first run. Up() is
// idempotent, so running migrations on every startup is safe.
func Migrate(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	// Use the embedded filesystem. The migrations are at the root of the embed FS.
	goose.SetBaseFS(migrations.FS)

	// Silence goose's default stdlib logger; slog owns logging.
	goose.SetLogger(goose.NopLogger())

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := ensureMailConnectionColumns(ctx, db); err != nil {
		return fmt.Errorf("repair mail connection schema: %w", err)
	}

	// After migrations, ensure FTS triggers exist (goose doesn't handle triggers well)
	if err := createFTS5Triggers(ctx, db); err != nil {
		return fmt.Errorf("create FTS triggers: %w", err)
	}

	return nil
}

// ensureMailConnectionColumns repairs databases imported from older builds
// where the goose version was newer than the actual mail_connections schema.
func ensureMailConnectionColumns(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(mail_connections)")
	if err != nil {
		return err
	}
	columns := make(map[string]struct{})
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		columns[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, column := range []struct{ name, declaration string }{
		{"auth_mode", `TEXT NOT NULL DEFAULT 'password'`},
		{"oauth_provider", `TEXT NOT NULL DEFAULT ''`},
		{"oauth_sealed", `TEXT NOT NULL DEFAULT ''`},
		{"oauth_state", `TEXT NOT NULL DEFAULT ''`},
	} {
		if _, exists := columns[column.name]; exists {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE mail_connections ADD COLUMN "+column.name+" "+column.declaration); err != nil {
			return fmt.Errorf("add %s: %w", column.name, err)
		}
	}
	return nil
}

// createFTS5Triggers creates the FTS5 triggers after migrations.
// Goose's SQL splitter can't handle triggers with BEGIN/END blocks.
func createFTS5Triggers(ctx context.Context, db *sql.DB) error {
	triggers := []string{
		// companies
		`CREATE TRIGGER IF NOT EXISTS trg_companies_ai AFTER INSERT ON companies BEGIN
  INSERT INTO fts_companies(name, domain, industry, notes) VALUES (new.name, new.domain, new.industry, new.notes);
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_companies_ad AFTER DELETE ON companies BEGIN
  DELETE FROM fts_companies WHERE rowid = old.rowid;
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_companies_au AFTER UPDATE ON companies BEGIN
  DELETE FROM fts_companies WHERE rowid = old.rowid;
  INSERT INTO fts_companies(name, domain, industry, notes) VALUES (new.name, new.domain, new.industry, new.notes);
END;`,

		// contacts
		`CREATE TRIGGER IF NOT EXISTS trg_contacts_ai AFTER INSERT ON contacts BEGIN
  INSERT INTO fts_contacts(first_name, last_name, email, phone, job_title, notes) VALUES (new.first_name, new.last_name, new.email, new.phone, new.job_title, new.notes);
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_contacts_ad AFTER DELETE ON contacts BEGIN
  DELETE FROM fts_contacts WHERE rowid = old.rowid;
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_contacts_au AFTER UPDATE ON contacts BEGIN
  DELETE FROM fts_contacts WHERE rowid = old.rowid;
  INSERT INTO fts_contacts(first_name, last_name, email, phone, job_title, notes) VALUES (new.first_name, new.last_name, new.email, new.phone, new.job_title, new.notes);
END;`,

		// leads
		`CREATE TRIGGER IF NOT EXISTS trg_leads_ai AFTER INSERT ON leads BEGIN
  INSERT INTO fts_leads(display_name, first_name, last_name, email, phone, company_name, notes) VALUES (new.display_name, new.first_name, new.last_name, new.email, new.phone, new.company_name, new.description);
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_leads_ad AFTER DELETE ON leads BEGIN
  DELETE FROM fts_leads WHERE rowid = old.rowid;
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_leads_au AFTER UPDATE ON leads BEGIN
  DELETE FROM fts_leads WHERE rowid = old.rowid;
  INSERT INTO fts_leads(display_name, first_name, last_name, email, phone, company_name, notes) VALUES (new.display_name, new.first_name, new.last_name, new.email, new.phone, new.company_name, new.description);
END;`,

		// deals
		`CREATE TRIGGER IF NOT EXISTS trg_deals_ai AFTER INSERT ON deals BEGIN
  INSERT INTO fts_deals(name, description) VALUES (new.name, new.description);
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_deals_ad AFTER DELETE ON deals BEGIN
  DELETE FROM fts_deals WHERE rowid = old.rowid;
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_deals_au AFTER UPDATE ON deals BEGIN
  DELETE FROM fts_deals WHERE rowid = old.rowid;
  INSERT INTO fts_deals(name, description) VALUES (new.name, new.description);
END;`,

		// tasks
		`CREATE TRIGGER IF NOT EXISTS trg_tasks_ai AFTER INSERT ON tasks BEGIN
  INSERT INTO fts_tasks(title, description) VALUES (new.title, new.description);
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_tasks_ad AFTER DELETE ON tasks BEGIN
  DELETE FROM fts_tasks WHERE rowid = old.rowid;
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_tasks_au AFTER UPDATE ON tasks BEGIN
  DELETE FROM fts_tasks WHERE rowid = old.rowid;
  INSERT INTO fts_tasks(title, description) VALUES (new.title, new.description);
END;`,

		// projects
		`CREATE TRIGGER IF NOT EXISTS trg_projects_ai AFTER INSERT ON projects BEGIN
  INSERT INTO fts_projects(name, description) VALUES (new.name, new.description);
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_projects_ad AFTER DELETE ON projects BEGIN
  DELETE FROM fts_projects WHERE rowid = old.rowid;
END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_projects_au AFTER UPDATE ON projects BEGIN
  DELETE FROM fts_projects WHERE rowid = old.rowid;
  INSERT INTO fts_projects(name, description) VALUES (new.name, new.description);
END;`,
	}

	for _, t := range triggers {
		if _, err := db.ExecContext(ctx, t); err != nil {
			// Ignore "already exists" errors
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("create trigger: %w", err)
		}
	}
	return nil
}

var _ = errors.New

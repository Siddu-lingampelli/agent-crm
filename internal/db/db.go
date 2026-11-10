// Package db opens and configures the SQLite database.
//
// WAL, foreign keys and a bounded pool are mandatory. These PRAGMA values are
// starting points to be tuned by measurement, not claimed optima. See
// Tech.md section 4 and PRD section 9.3.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the database handle with CRM-specific helpers.
type DB struct {
	sql       *sql.DB
	dsn       string
	closed    atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

// Config holds database settings.
type Config struct {
	Path              string
	MaxOpenConns      int
	MaxIdleConns      int
	ConnMaxLifetime   time.Duration
	BusyTimeout       time.Duration
	CacheSizeKiB      int
	MmapSizeBytes     int64
	PRAGMAForeignKeys bool
	PRAGMAJournalMode string
	PRAGMASynchronous string
	PRAGMATempStore   string
	PRAGMACacheSize   int
	PRAGMAPageSize    int
	PRAGMAMmapSize    int64
}

// Open initializes SQLite with the required PRAGMAs, runs embedded migrations.
func Open(ctx context.Context, cfg Config, log *slog.Logger) (*DB, error) {
	if log == nil {
		log = slog.Default()
	}
	if cfg.MaxOpenConns <= 0 {
		return nil, errors.New("db.MaxOpenConns must be >= 1")
	}
	if cfg.MaxIdleConns < 0 {
		return nil, errors.New("db.MaxIdleConns must be >= 0")
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return nil, fmt.Errorf("db.MaxIdleConns (%d) must not exceed db.MaxOpenConns (%d)", cfg.MaxIdleConns, cfg.MaxOpenConns)
	}
	if cfg.BusyTimeout <= 0 {
		cfg.BusyTimeout = 5 * time.Second
	}

	// Ensure the parent directory exists. Without this, sqlite reports a
	// misleading "out of memory" error instead of "no such directory".
	if dir := filepath.Dir(cfg.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create data directory %s: %w", dir, err)
		}
	}

	if strings.TrimSpace(cfg.Path) == "" {
		return nil, errors.New("database path must not be empty")
	}
	// Use an absolute, escaped file URI. A query must start with '?', otherwise
	// SQLite treats the PRAGMAs as part of the filename and never applies them.
	absolutePath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("database path: %w", err)
	}
	// Earlier builds appended PRAGMAs to the filename. Reuse a unique legacy
	// file when the configured file is absent so upgrades cannot hide user data.
	if _, err := os.Stat(absolutePath); errors.Is(err, os.ErrNotExist) {
		entries, err := os.ReadDir(filepath.Dir(absolutePath))
		if err != nil {
			return nil, err
		}
		var legacy []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), filepath.Base(absolutePath)+"&_pragma=") && !strings.HasSuffix(entry.Name(), "-wal") && !strings.HasSuffix(entry.Name(), "-shm") {
				legacy = append(legacy, filepath.Join(filepath.Dir(absolutePath), entry.Name()))
			}
		}
		if len(legacy) > 1 {
			return nil, errors.New("multiple legacy database files found; set CRM_DATABASE_PATH to the intended database")
		}
		if len(legacy) == 1 {
			absolutePath = legacy[0]
			log.Warn("using legacy database filename", "path", absolutePath)
		}
	}
	uriPath := filepath.ToSlash(absolutePath)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	dbURI := url.URL{Scheme: "file", Path: uriPath}
	// Build DSN with all PRAGMA settings
	journalMode := "WAL"
	if cfg.PRAGMAJournalMode != "" {
		journalMode = cfg.PRAGMAJournalMode
	}
	if !strings.EqualFold(journalMode, "WAL") {
		return nil, errors.New("database journal mode must be WAL")
	}
	for name, value := range map[string]string{"synchronous": cfg.PRAGMASynchronous, "temp_store": cfg.PRAGMATempStore} {
		valid := value == ""
		allowed := []string{"OFF", "NORMAL", "FULL", "EXTRA", "0", "1", "2", "3"}
		if name == "temp_store" {
			allowed = []string{"DEFAULT", "FILE", "MEMORY", "0", "1", "2"}
		}
		for _, item := range allowed {
			if strings.EqualFold(value, item) {
				valid = true
			}
		}
		if !valid {
			return nil, fmt.Errorf("invalid PRAGMA %s", name)
		}
	}
	dsnParts := []string{
		fmt.Sprintf("_pragma=busy_timeout(%d)", cfg.BusyTimeout.Milliseconds()),
		"_pragma=foreign_keys(1)",
	}

	if cfg.PRAGMASynchronous != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("_pragma=synchronous(%s)", cfg.PRAGMASynchronous))
	}
	if cfg.PRAGMATempStore != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("_pragma=temp_store(%s)", cfg.PRAGMATempStore))
	}

	if cfg.PRAGMACacheSize != 0 {
		dsnParts = append(dsnParts, fmt.Sprintf("_pragma=cache_size(%d)", cfg.PRAGMACacheSize))
	}
	if cfg.PRAGMAPageSize != 0 {
		dsnParts = append(dsnParts, fmt.Sprintf("_pragma=page_size(%d)", cfg.PRAGMAPageSize))
	}
	if cfg.PRAGMAMmapSize > 0 {
		dsnParts = append(dsnParts, fmt.Sprintf("_pragma=mmap_size(%d)", cfg.PRAGMAMmapSize))
	}

	dsn := dbURI.String() + "?" + strings.Join(dsnParts, "&")
	log.Info("SQLite DSN", "dsn", dsn)

	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	handle.SetMaxOpenConns(cfg.MaxOpenConns)
	handle.SetMaxIdleConns(cfg.MaxIdleConns)
	if cfg.ConnMaxLifetime > 0 {
		handle.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	db := &DB{sql: handle, dsn: dsn}

	if err := db.ping(ctx); err != nil {
		_ = handle.Close()
		return nil, err
	}

	// Set journal mode via PRAGMA (not via DSN, as it's persisted at DB level)
	if _, err := handle.ExecContext(ctx, fmt.Sprintf("PRAGMA journal_mode=%s;", journalMode)); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("set journal_mode: %w", err)
	}

	// Enable foreign keys (must be set per-connection)
	if _, err := handle.ExecContext(ctx, "PRAGMA foreign_keys=ON;"); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("set foreign_keys: %w", err)
	}

	if err := db.verifyPragmas(ctx, log); err != nil {
		_ = handle.Close()
		return nil, err
	}

	if err := Migrate(ctx, handle); err != nil {
		_ = handle.Close()
		return nil, err
	}

	return db, nil
}

// ping verifies the database is reachable.
func (d *DB) ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := d.sql.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}
	return nil
}

// verifyPragmas reads back and logs the applied PRAGMA values so that the
// actual settings are visible, not assumed.
func (d *DB) verifyPragmas(ctx context.Context, log *slog.Logger) error {
	checks := []struct {
		label  string
		query  string
		expect string
		warn   bool
	}{
		{label: "journal_mode", query: "PRAGMA journal_mode;", expect: "wal"},
		{label: "foreign_keys", query: "PRAGMA foreign_keys;", expect: "1"},
	}

	for _, c := range checks {
		var got string
		if err := d.sql.QueryRowContext(ctx, c.query).Scan(&got); err != nil {
			return fmt.Errorf("read %s: %w", c.label, err)
		}
		log.Debug("pragma value", "pragma", c.label, "got", got, "expect", c.expect)
		// Case-insensitive comparison for journal_mode
		if strings.EqualFold(c.label, "journal_mode") {
			if !strings.EqualFold(got, c.expect) {
				if c.warn {
					log.Warn("pragma mismatch", "pragma", c.label, "got", got, "want", c.expect)
					continue
				}
				return fmt.Errorf("pragma %s = %q, want %q", c.label, got, c.expect)
			}
		} else if got != c.expect {
			if c.warn {
				log.Warn("pragma mismatch", "pragma", c.label, "got", got, "want", c.expect)
				continue
			}
			return fmt.Errorf("pragma %s = %q, want %q", c.label, got, c.expect)
		}
		log.Debug("pragma verified", "pragma", c.label, "value", got)
	}
	return nil
}

// SQL exposes the underlying handle for repositories.
func (d *DB) SQL() *sql.DB { return d.sql }

// Ready reports whether the database is usable.
func (d *DB) Ready() bool {
	if d == nil || d.sql == nil || d.closed.Load() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return d.sql.PingContext(ctx) == nil
}

// Status returns database metrics for the system page.
func (d *DB) Status() map[string]any {
	if d == nil || d.sql == nil {
		return map[string]any{"status": "unavailable"}
	}
	stats := d.sql.Stats()
	return map[string]any{
		"status":               "ok",
		"driver":               "modernc.org/sqlite",
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"max_open_connections": stats.MaxOpenConnections,
	}
}

// Close shuts down the database cleanly, checkpointing WAL.
func (d *DB) Close() error {
	if d == nil {
		return nil
	}
	d.closeOnce.Do(func() {
		d.closed.Store(true)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// TRUNCATE checkpoint so the WAL does not leave recovery work behind.
		if d.sql != nil {
			_, _ = d.sql.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);")
		}
		if d.sql != nil {
			d.closeErr = d.sql.Close()
		}
	})
	return d.closeErr
}

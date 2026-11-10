// Package config loads and validates application configuration.
package config

import (
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the validated runtime configuration.
type Config struct {
	HTTP     HTTP           `json:"http"`
	Database Database       `json:"database"`
	Log      Log            `json:"log"`
	Server   Server         `json:"server"`
	Jobs     Jobs           `json:"jobs"`
	Security SecurityConfig `json:"security"`
}

type HTTP struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

func (h HTTP) Addr() string {
	return netJoinHostPort(h.Host, h.Port)
}

func netJoinHostPort(host string, port int) string {
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

type Database struct {
	Path              string        `json:"path"`
	MaxOpenConns      int           `json:"max_open_conns"`
	MaxIdleConns      int           `json:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `json:"conn_max_lifetime"`
	BusyTimeout       time.Duration `json:"busy_timeout"`
	CacheSizeKiB      int           `json:"cache_size_kib"`
	MmapSizeBytes     int64         `json:"mmap_size_bytes"`
	PRAGMAForeignKeys bool          `json:"pragma_foreign_keys"`
	PRAGMAJournalMode string        `json:"pragma_journal_mode"`
	PRAGMASynchronous string        `json:"pragma_synchronous"`
	PRAGMATempStore   string        `json:"pragma_temp_store"`
	PRAGMACacheSize   int           `json:"pragma_cache_size"`
	PRAGMAPageSize    int           `json:"pragma_page_size"`
	PRAGMAMmapSize    int64         `json:"pragma_mmap_size"`
}

type Log struct {
	Format string `json:"format"`
	Level  string `json:"level"`
}

type Jobs struct {
	Workers   int `json:"workers"`
	QueueSize int `json:"queue_size"`
}

type Server struct {
	Version string `json:"version"`
	BuildID string `json:"build_id"`
	GitSHA  string `json:"git_sha"`
	Commit  string `json:"commit"`
	Env     string `json:"env"`
}

// Defaults returns the default configuration.
func Defaults() Config {
	return Config{
		HTTP: HTTP{
			Host:            "127.0.0.1",
			Port:            8080,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Database: Database{
			Path:              "data/crm.db",
			MaxOpenConns:      4,
			MaxIdleConns:      2,
			ConnMaxLifetime:   30 * time.Minute,
			BusyTimeout:       5 * time.Second,
			CacheSizeKiB:      16000,
			MmapSizeBytes:     64 << 20,
			PRAGMAForeignKeys: true,
			PRAGMAJournalMode: "WAL",
			PRAGMASynchronous: "NORMAL",
			PRAGMATempStore:   "MEMORY",
			PRAGMACacheSize:   -16000, // negative = KiB, positive = pages
			PRAGMAPageSize:    4096,
			PRAGMAMmapSize:    64 << 20,
		},
		Log: Log{
			Format: "text",
			Level:  "info",
		},
		Server: Server{
			Version: "0.1.0",
			BuildID: "dev",
			GitSHA:  "",
			Commit:  "",
			Env:     "development",
		},
		Jobs: Jobs{
			Workers:   8,
			QueueSize: 256,
		},
		Security: DefaultSecurityConfig(),
	}
}

// Load constructs configuration from the environment with validation.
func Load() (Config, error) {
	cfg := Defaults()

	// Environment overrides.
	if v := os.Getenv("CRM_HTTP_HOST"); v != "" {
		cfg.HTTP.Host = strings.TrimSpace(v)
	}
	if v := os.Getenv("CRM_HTTP_PORT"); v != "" {
		port, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || port < 1 || port > 65535 {
			return cfg, fmt.Errorf("invalid CRM_HTTP_PORT: %q", v)
		}
		cfg.HTTP.Port = port
	}
	if v := os.Getenv("CRM_DATABASE_PATH"); v != "" {
		cfg.Database.Path = strings.TrimSpace(v)
	}
	if v := os.Getenv("CRM_LOG_FORMAT"); v != "" {
		cfg.Log.Format = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("CRM_LOG_LEVEL"); v != "" {
		cfg.Log.Level = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("CRM_SERVER_VERSION"); v != "" {
		cfg.Server.Version = strings.TrimSpace(v)
	}
	if v := os.Getenv("CRM_BUILD_ID"); v != "" {
		cfg.Server.BuildID = strings.TrimSpace(v)
	}
	if v := os.Getenv("CRM_GIT_SHA"); v != "" {
		cfg.Server.GitSHA = strings.TrimSpace(v)
		cfg.Server.Commit = strings.TrimSpace(v)
	}
	if v := os.Getenv("CRM_ENV"); v != "" {
		cfg.Server.Env = strings.TrimSpace(v)
	}

	if err := loadRuntimeEnvironment(&cfg); err != nil {
		return cfg, err
	}
	if err := loadSecurityEnvironment(&cfg.Security); err != nil {
		return cfg, err
	}
	if strings.EqualFold(cfg.Server.Env, "production") && os.Getenv("CRM_SECURITY_SESSION_COOKIE_SECURE") == "" {
		cfg.Security.SessionCookieSecure = true
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// Validate checks configuration invariants.
func (c Config) Validate() error {
	if c.Security.HTTPClientSkipVerify {
		return fmt.Errorf("TLS certificate verification cannot be disabled")
	}
	if c.Jobs.Workers < 1 || c.Jobs.Workers > 128 || c.Jobs.QueueSize < 1 || c.Jobs.QueueSize > 10000 {
		return fmt.Errorf("jobs require 1..128 workers and a queue size of 1..10000")
	}
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		return fmt.Errorf("http.port must be between 1 and 65535, got %d", c.HTTP.Port)
	}
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("database.max_open_conns must be >= 1, got %d", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database.max_idle_conns must be >= 0, got %d", c.Database.MaxIdleConns)
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("database.max_idle_conns (%d) must not exceed max_open_conns (%d)", c.Database.MaxIdleConns, c.Database.MaxOpenConns)
	}
	if strings.TrimSpace(c.Database.Path) == "" {
		return fmt.Errorf("database.path must not be empty")
	}
	switch strings.ToLower(c.Log.Format) {
	case "text", "json":
	default:
		return fmt.Errorf("log.format must be text or json, got %q", c.Log.Format)
	}
	switch strings.ToLower(c.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level must be debug, info, warn or error, got %q", c.Log.Level)
	}

	// Security validation
	if c.Security.RateLimitEnabled && (c.Security.RateLimitRPS <= 0 || math.IsNaN(c.Security.RateLimitRPS) || math.IsInf(c.Security.RateLimitRPS, 0)) {
		return fmt.Errorf("security.rate_limit_rps must be > 0 when rate limiting enabled, got %f", c.Security.RateLimitRPS)
	}
	if c.Security.RateLimitEnabled && c.Security.RateLimitBurst <= 0 {
		return fmt.Errorf("security.rate_limit_burst must be > 0 when rate limiting enabled, got %d", c.Security.RateLimitBurst)
	}
	if c.Security.MaxRequestBodySize <= 0 {
		return fmt.Errorf("security.max_request_body_size must be > 0, got %d", c.Security.MaxRequestBodySize)
	}
	if c.Security.MaxUploadSize <= 0 {
		return fmt.Errorf("security.max_upload_size must be > 0, got %d", c.Security.MaxUploadSize)
	}
	if c.Security.MaxUploadSize < c.Security.MaxRequestBodySize {
		return fmt.Errorf("security.max_upload_size (%d) must be >= max_request_body_size (%d)", c.Security.MaxUploadSize, c.Security.MaxRequestBodySize)
	}
	if len(c.Security.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("security.cors_allowed_origins must not be empty")
	}
	if c.Security.RequestTimeout <= 0 {
		return fmt.Errorf("security.request_timeout must be > 0, got %v", c.Security.RequestTimeout)
	}
	if c.Security.HTTPClientTimeout <= 0 {
		return fmt.Errorf("security.http_client_timeout must be > 0, got %v", c.Security.HTTPClientTimeout)
	}
	if c.Security.HTTPClientMaxResponseSize <= 0 {
		return fmt.Errorf("security.http_client_max_response_size must be > 0, got %d", c.Security.HTTPClientMaxResponseSize)
	}
	if c.Security.IdempotencyTTL <= 0 {
		return fmt.Errorf("security.idempotency_ttl must be > 0, got %v", c.Security.IdempotencyTTL)
	}
	if c.Security.AuditLogMaxSize <= 0 {
		return fmt.Errorf("security.audit_log_max_size must be > 0, got %d", c.Security.AuditLogMaxSize)
	}
	if c.Security.AuditLogMaxEntries <= 0 {
		return fmt.Errorf("security.audit_log_max_entries must be > 0, got %d", c.Security.AuditLogMaxEntries)
	}
	if strings.TrimSpace(c.Security.SessionCookieName) == "" {
		return fmt.Errorf("security.session_cookie_name must not be empty")
	}
	if c.Security.SessionMaxAge <= 0 {
		return fmt.Errorf("security.session_max_age must be > 0, got %v", c.Security.SessionMaxAge)
	}
	if c.Security.CSRFTokenLength < 16 {
		return fmt.Errorf("security.csrf_token_length must be >= 16, got %d", c.Security.CSRFTokenLength)
	}

	if err := (&http.Cookie{Name: c.Security.SessionCookieName, Value: "valid", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}).Valid(); err != nil {
		return fmt.Errorf("invalid session cookie name: %w", err)
	}
	switch strings.ToLower(c.Security.SessionCookieSameSite) {
	case "lax", "strict":
	case "none":
		if !c.Security.SessionCookieSecure {
			return fmt.Errorf("SameSite=None requires secure cookies")
		}
	default:
		return fmt.Errorf("invalid session cookie SameSite")
	}
	if strings.EqualFold(c.Server.Env, "production") && (!c.Security.SessionCookieSecure || !c.Security.CSRFEnabled || c.Security.HTTPClientSkipVerify) {
		return fmt.Errorf("production requires secure cookies, CSRF protection and verified TLS")
	}
	for _, origin := range c.Security.CORSAllowedOrigins {
		if origin == "*" {
			if c.Security.CORSAllowCredentials {
				return fmt.Errorf("credentialed CORS requires explicit origins")
			}
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid CORS origin %q", origin)
		}
	}
	for _, cidr := range append(append([]string{}, c.Security.SSRFAllowedCIDRs...), c.Security.HTTPClientAllowedCIDRs...) {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid allowed CIDR %q", cidr)
		}
	}
	if c.HTTP.ReadTimeout <= 0 || c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 || c.HTTP.ShutdownTimeout <= 0 {
		return fmt.Errorf("HTTP timeouts must be positive")
	}
	return nil
}

// slogLevel returns the slog level for a string.
func slogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Level returns the slog level.
func (l Log) LevelVar() slog.Level {
	return slogLevel(l.Level)
}

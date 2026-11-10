// Package config holds application configuration including security settings.
package config

import (
	"time"

	"crm.local/internal/security"
)

// DefaultSecurityConfig returns the default security configuration.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		RateLimitEnabled:          true,
		RateLimitRPS:              100,
		RateLimitBurst:            200,
		MaxRequestBodySize:        10 * 1024 * 1024, // 10 MB
		MaxUploadSize:             50 * 1024 * 1024, // 50 MB
		CORSAllowedOrigins:        []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:5175"},
		CORSAllowedMethods:        []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:        []string{"Content-Type", "Authorization", "X-Requested-With", "Idempotency-Key"},
		CORSAllowCredentials:      true,
		SSRFBlockPrivateIPs:       true,
		RequestTimeout:            30 * time.Second,
		HTTPClientTimeout:         30 * time.Second,
		HTTPClientSkipVerify:      false,
		HTTPClientMaxResponseSize: 10 * 1024 * 1024, // 10 MB
		IdempotencyKeyHeader:      "Idempotency-Key",
		IdempotencyTTL:            24 * time.Hour,
		AuditLogMaxSize:           1024 * 1024, // 1 MB
		AuditLogMaxEntries:        10000,
		SessionCookieName:         "crm_session",
		SessionCookieSecure:       false,
		SessionCookieSameSite:     "Lax",
		SessionMaxAge:             24 * time.Hour,
		CSRFEnabled:               true,
		CSRFHeaderName:            "X-CSRF-Token",
		CSRFCookieName:            "csrf_token",
		CSRFTokenLength:           32,
	}
}

// SecurityConfig holds security-related configuration.
type SecurityConfig struct {
	// Rate limiting
	RateLimitEnabled bool    `env:"CRM_SECURITY_RATE_LIMIT_ENABLED" envDefault:"true"`
	RateLimitRPS     float64 `env:"CRM_SECURITY_RATE_LIMIT_RPS" envDefault:"100"`
	RateLimitBurst   int     `env:"CRM_SECURITY_RATE_LIMIT_BURST" envDefault:"200"`

	// Request size limits
	MaxRequestBodySize int64 `env:"CRM_SECURITY_MAX_REQUEST_BODY" envDefault:"10485760"` // 10 MB
	MaxUploadSize      int64 `env:"CRM_SECURITY_MAX_UPLOAD_SIZE" envDefault:"52428800"`  // 50 MB

	// CORS
	CORSAllowedOrigins   []string `env:"CRM_SECURITY_CORS_ORIGINS" envDefault:"http://localhost:5173,http://localhost:5174,http://localhost:5175"`
	CORSAllowedMethods   []string `env:"CRM_SECURITY_CORS_METHODS" envDefault:"GET,POST,PATCH,DELETE,OPTIONS"`
	CORSAllowedHeaders   []string `env:"CRM_SECURITY_CORS_HEADERS" envDefault:"Content-Type,Authorization,X-Requested-With,Idempotency-Key"`
	CORSAllowCredentials bool     `env:"CRM_SECURITY_CORS_CREDENTIALS" envDefault:"true"`

	// SSRF protection
	SSRFBlockPrivateIPs bool     `env:"CRM_SECURITY_SSRF_BLOCK_PRIVATE" envDefault:"true"`
	SSRFAllowedCIDRs    []string `env:"CRM_SECURITY_SSRF_ALLOWED_CIDRS" envDefault:""`

	// Timeouts
	RequestTimeout time.Duration `env:"CRM_SECURITY_REQUEST_TIMEOUT" envDefault:"30s"`

	// Secure HTTP client
	HTTPClientTimeout         time.Duration `env:"CRM_SECURITY_HTTP_CLIENT_TIMEOUT" envDefault:"30s"`
	HTTPClientSkipVerify      bool          `env:"CRM_SECURITY_HTTP_CLIENT_SKIP_VERIFY" envDefault:"false"`
	HTTPClientMaxResponseSize int64         `env:"CRM_SECURITY_HTTP_CLIENT_MAX_RESPONSE" envDefault:"10485760"` // 10 MB
	HTTPClientAllowedCIDRs    []string      `env:"CRM_SECURITY_HTTP_CLIENT_ALLOWED_CIDRS" envDefault:""`

	// Idempotency
	IdempotencyKeyHeader string        `env:"CRM_SECURITY_IDEMPOTENCY_HEADER" envDefault:"Idempotency-Key"`
	IdempotencyTTL       time.Duration `env:"CRM_SECURITY_IDEMPOTENCY_TTL" envDefault:"24h"`

	// Audit
	AuditLogMaxSize    int64 `env:"CRM_SECURITY_AUDIT_MAX_SIZE" envDefault:"1048576"` // 1 MB per entry
	AuditLogMaxEntries int   `env:"CRM_SECURITY_AUDIT_MAX_ENTRIES" envDefault:"10000"`

	// Session
	SessionCookieName     string        `env:"CRM_SECURITY_SESSION_COOKIE_NAME" envDefault:"crm_session"`
	SessionCookieSecure   bool          `env:"CRM_SECURITY_SESSION_COOKIE_SECURE" envDefault:"false"`
	SessionCookieSameSite string        `env:"CRM_SECURITY_SESSION_COOKIE_SAMESITE" envDefault:"Lax"`
	SessionMaxAge         time.Duration `env:"CRM_SECURITY_SESSION_MAX_AGE" envDefault:"24h"`

	// CSRF
	CSRFEnabled     bool   `env:"CRM_SECURITY_CSRF_ENABLED" envDefault:"true"`
	CSRFHeaderName  string `env:"CRM_SECURITY_CSRF_HEADER" envDefault:"X-CSRF-Token"`
	CSRFCookieName  string `env:"CRM_SECURITY_CSRF_COOKIE" envDefault:"csrf_token"`
	CSRFTokenLength int    `env:"CRM_SECURITY_CSRF_TOKEN_LENGTH" envDefault:"32"`
}

// ToSecurityConfig converts to internal/security.SecurityConfig
func (c *SecurityConfig) ToSecurityConfig() security.SecurityConfig {
	return security.SecurityConfig{
		RateLimitEnabled:     c.RateLimitEnabled,
		RateLimitRPS:         c.RateLimitRPS,
		RateLimitBurst:       c.RateLimitBurst,
		MaxRequestBodySize:   c.MaxRequestBodySize,
		MaxUploadSize:        c.MaxUploadSize,
		CORSAllowedOrigins:   c.CORSAllowedOrigins,
		CORSAllowedMethods:   c.CORSAllowedMethods,
		CORSAllowedHeaders:   c.CORSAllowedHeaders,
		CORSAllowCredentials: c.CORSAllowCredentials,
		SSRFBlockPrivateIPs:  c.SSRFBlockPrivateIPs,
		SSRFAllowedCIDRs:     c.SSRFAllowedCIDRs,
		RequestTimeout:       c.RequestTimeout,
	}
}

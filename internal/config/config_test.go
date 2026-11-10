package config

import (
	"testing"
	"time"
)

func TestEnvironmentOverridesAreApplied(t *testing.T) {
	t.Setenv("CRM_HTTP_READ_TIMEOUT", "7s")
	t.Setenv("CRM_DATABASE_MAX_OPEN_CONNS", "7")
	t.Setenv("CRM_JOBS_WORKERS", "3")
	t.Setenv("CRM_SECURITY_SESSION_MAX_AGE", "2h")
	t.Setenv("CRM_SECURITY_CORS_ORIGINS", "http://localhost:5173, http://localhost:5174")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTP.ReadTimeout != 7*time.Second || c.Database.MaxOpenConns != 7 || c.Jobs.Workers != 3 || c.Security.SessionMaxAge != 2*time.Hour || c.Security.CORSAllowedOrigins[1] != "http://localhost:5174" {
		t.Fatalf("ignored overrides: %+v", c)
	}
}
func TestInvalidSecurityOverridesFail(t *testing.T) {
	for _, name := range []string{"CRM_SECURITY_CSRF_ENABLED", "CRM_SECURITY_RATE_LIMIT_RPS", "CRM_SECURITY_SESSION_MAX_AGE", "CRM_HTTP_READ_TIMEOUT"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, "invalid")
			if _, err := Load(); err == nil {
				t.Fatal("invalid override silently accepted")
			}
		})
	}
}
func TestIPv6Address(t *testing.T) {
	if got := (HTTP{Host: "::1", Port: 8080}).Addr(); got != "[::1]:8080" {
		t.Fatal(got)
	}
}

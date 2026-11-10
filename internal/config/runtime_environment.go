package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func loadRuntimeEnvironment(c *Config) error {
	fields := map[string]any{
		"CRM_DATABASE_MAX_OPEN_CONNS": &c.Database.MaxOpenConns, "CRM_DATABASE_MAX_IDLE_CONNS": &c.Database.MaxIdleConns, "CRM_DATABASE_BUSY_TIMEOUT": &c.Database.BusyTimeout, "CRM_DATABASE_CONN_MAX_LIFETIME": &c.Database.ConnMaxLifetime,
		"CRM_HTTP_READ_TIMEOUT": &c.HTTP.ReadTimeout, "CRM_HTTP_WRITE_TIMEOUT": &c.HTTP.WriteTimeout, "CRM_HTTP_IDLE_TIMEOUT": &c.HTTP.IdleTimeout, "CRM_HTTP_SHUTDOWN_TIMEOUT": &c.HTTP.ShutdownTimeout,
		"CRM_JOBS_WORKERS": &c.Jobs.Workers, "CRM_JOBS_QUEUE_SIZE": &c.Jobs.QueueSize,
	}
	for name, target := range fields {
		raw, present := os.LookupEnv(name)
		if !present {
			continue
		}
		raw = strings.TrimSpace(raw)
		var err error
		switch value := target.(type) {
		case *int:
			var parsed int
			parsed, err = strconv.Atoi(raw)
			if err == nil {
				*value = parsed
			}
		case *time.Duration:
			var parsed time.Duration
			parsed, err = time.ParseDuration(raw)
			if err == nil {
				*value = parsed
			}
		}
		if err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
	}
	if os.Getenv("CRM_ENV") == "" && os.Getenv("CRM_SERVER_ENV") != "" {
		c.Server.Env = os.Getenv("CRM_SERVER_ENV")
	}
	if os.Getenv("CRM_BUILD_ID") == "" && os.Getenv("CRM_SERVER_BUILD_ID") != "" {
		c.Server.BuildID = os.Getenv("CRM_SERVER_BUILD_ID")
	}
	if os.Getenv("CRM_GIT_SHA") == "" && os.Getenv("CRM_SERVER_GIT_SHA") != "" {
		c.Server.GitSHA = os.Getenv("CRM_SERVER_GIT_SHA")
		c.Server.Commit = c.Server.GitSHA
	}
	return nil
}

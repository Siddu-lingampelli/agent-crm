// Command crm runs the self-hosted CRM application process.
//
// One process serves: the embedded React SPA, the REST API under /api/v1,
// the MCP Streamable HTTP endpoint, the shared application service layer and
// the bounded background worker pool. See Tech.md section 1.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"crm.local/internal/app"
	"crm.local/internal/config"
	"crm.local/internal/logging"
)

// Set by the container build; explicit runtime environment overrides win.
var version, buildID, gitSHA string

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	generatedEncryptionKey, err := ensureLocalEncryptionKey(cfg.Database.Path, cfg.Server.Env)
	if err != nil {
		return err
	}
	if version != "" && os.Getenv("CRM_SERVER_VERSION") == "" {
		cfg.Server.Version = version
	}
	if buildID != "" && os.Getenv("CRM_BUILD_ID") == "" && os.Getenv("CRM_SERVER_BUILD_ID") == "" {
		cfg.Server.BuildID = buildID
	}
	if gitSHA != "" && os.Getenv("CRM_GIT_SHA") == "" && os.Getenv("CRM_SERVER_GIT_SHA") == "" {
		cfg.Server.GitSHA = gitSHA
		cfg.Server.Commit = gitSHA
	}

	logger := logging.New(cfg.Log.Format, cfg.Log.Level)
	slog.SetDefault(logger)
	if generatedEncryptionKey {
		logger.Warn("generated a local credential-encryption key", "path", filepath.Join(filepath.Dir(cfg.Database.Path), "credentials.key"))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer application.Close()

	server := &http.Server{
		Addr:              cfg.HTTP.Addr(),
		Handler:           application.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	// Listen before announcing readiness so a bind failure is a clean error.
	ln, err := net.Listen("tcp", cfg.HTTP.Addr())
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "addr", cfg.HTTP.Addr())
		if serveErr := server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case serveErr := <-errCh:
		if serveErr != nil {
			return serveErr
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	// StopInFlightWork is separate from closing the listener: in-flight requests
	// need time to finish after Serve returns.
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return err
	}
	application.Shutdown(shutdownCtx)

	logger.Info("shutdown complete")
	return nil
}

// ensureLocalEncryptionKey provides a persistent key for local development.
// Production and any non-local deployment must supply CRM_ENCRYPTION_KEY
// through their secret manager so the key is not stored with application
// files. Only explicitly local environments get an auto-generated key file;
// anything else (staging, or an unknown value) must fail loudly rather than
// silently writing a fresh key beside the database, which would make every
// previously sealed credential undecryptable.
func ensureLocalEncryptionKey(databasePath, environment string) (bool, error) {
	if strings.TrimSpace(os.Getenv("CRM_ENCRYPTION_KEY")) != "" {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "local", "test", "":
		// known-local: fall through to key-file handling below
	default:
		return false, fmt.Errorf("CRM_ENCRYPTION_KEY is required when CRM_ENV=%q (only development/local/test may use a generated key file)", environment)
	}
	keyPath := filepath.Join(filepath.Dir(databasePath), "credentials.key")
	readKey := func() error {
		encoded, err := os.ReadFile(keyPath)
		if err != nil {
			return err
		}
		value := strings.TrimSpace(string(encoded))
		decoded, err := hex.DecodeString(value)
		if err != nil || len(decoded) != 32 {
			return fmt.Errorf("local credential-encryption key at %s is invalid", keyPath)
		}
		return os.Setenv("CRM_ENCRYPTION_KEY", value)
	}
	if err := readKey(); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return false, fmt.Errorf("generate local credential-encryption key: %w", err)
	}
	encoded := []byte(hex.EncodeToString(key) + "\n")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return false, fmt.Errorf("create local key directory: %w", err)
	}
	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, readKey()
	}
	if err != nil {
		return false, fmt.Errorf("create local credential-encryption key: %w", err)
	}
	if _, err := file.Write(encoded); err != nil {
		_ = file.Close()
		_ = os.Remove(keyPath)
		return false, fmt.Errorf("write local credential-encryption key: %w", err)
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("close local credential-encryption key: %w", err)
	}
	if err := os.Setenv("CRM_ENCRYPTION_KEY", strings.TrimSpace(string(encoded))); err != nil {
		return false, err
	}
	return true, nil
}

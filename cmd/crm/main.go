// Command crm runs the self-hosted CRM application process.
//
// One process serves: the embedded React SPA, the REST API under /api/v1,
// the MCP Streamable HTTP endpoint, the shared application service layer and
// the bounded background worker pool. See Tech.md section 1.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
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

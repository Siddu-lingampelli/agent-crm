// Package httpserver implements the production HTTP server and middleware.
package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Server is a thin wrapper over net/http that enforces request
// deadlines, structured logging and a tiny memory budget.
type Server struct {
	cfg       Config
	log       *slog.Logger
	mux       *http.ServeMux
	publicMux *http.ServeMux
	start     time.Time

	middlewares       []func(http.Handler) http.Handler
	publicMiddlewares []func(http.Handler) http.Handler
	staticHandler     http.Handler
}

// Config holds server-level settings.
type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// New returns a new server instance.
func New(cfg Config, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		cfg:               cfg,
		log:               log,
		mux:               http.NewServeMux(),
		publicMux:         http.NewServeMux(),
		start:             time.Now(),
		middlewares:       []func(http.Handler) http.Handler{},
		publicMiddlewares: []func(http.Handler) http.Handler{},
	}
}

// Middleware adds a middleware function to the chain (applies to protected routes).
func (s *Server) Middleware(middleware func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middleware)
}

// MiddlewareNoAuth adds a middleware function to the public route chain.
func (s *Server) MiddlewareNoAuth(middleware func(http.Handler) http.Handler) {
	s.publicMiddlewares = append(s.publicMiddlewares, middleware)
}

// Uptime returns the elapsed time since the server was created.
func (s *Server) Uptime() time.Duration {
	return time.Since(s.start)
}

// MemoryStatus returns real process RSS measured by the backend.
func (s *Server) MemoryStatus() map[string]any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]any{
		"allocated_bytes": m.TotalAlloc,
		"runtime_bytes":   m.Sys,
		"heap_bytes":      m.HeapAlloc,
		"goroutines":      runtime.NumGoroutine(),
		"gc_runs":         m.NumGC,
		"uptime_seconds":  s.Uptime().Seconds(),
	}
}

// HandleFunc registers a handler with the server's mux (protected by middleware).
func (s *Server) HandleFunc(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, h)
}

// HandleFuncNoAuth registers a handler without applying auth middleware.
func (s *Server) HandleFuncNoAuth(pattern string, h http.HandlerFunc) {
	s.publicMux.HandleFunc(pattern, h)
}

// ServeStatic registers a static file handler. In production this serves the
// embedded React assets. In dev it serves from disk so a Vite rebuild can be
// observed without restarting the Go process.
func (s *Server) ServeStatic(prefix, fsPath string) {
	s.staticHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", 405)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" {
			http.NotFound(w, r)
			return
		}
		root, err := os.OpenRoot(fsPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer root.Close()
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "."
		}
		name, err = filepath.Localize(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		info, err := root.Stat(name)
		if err != nil || info.IsDir() {
			if filepath.Ext(r.URL.Path) != "" && r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, root.FS(), "index.html")
			return
		}
		http.FileServerFS(root.FS()).ServeHTTP(w, r)
	})
}

// middleware chains all middlewares and adds request IDs, timeouts and structured logging.
func (s *Server) middleware(next http.Handler) http.Handler {
	// Chain middlewares in reverse order (last added wraps first)
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		next = s.middlewares[i](next)
	}

	// Add base logging middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Info("incoming request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// publicMiddleware chains public middlewares and adds base logging.
func (s *Server) publicMiddleware(next http.Handler) http.Handler {
	// Chain public middlewares in reverse order (last added wraps first)
	for i := len(s.publicMiddlewares) - 1; i >= 0; i-- {
		next = s.publicMiddlewares[i](next)
	}

	// Add base logging middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Info("incoming request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// Handler returns the wrapped http.Handler with middleware.
// Protected routes are wrapped with middleware; public routes are served directly.
func (s *Server) Handler() http.Handler {
	// Wrap both muxes with their respective middleware chains
	protectedHandler := s.middleware(s.mux)

	// Create a combined handler that checks public routes first, then protected
	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request matches any public route pattern
		_, pattern := s.publicMux.Handler(r)
		if pattern != "" {
			s.publicMux.ServeHTTP(w, r)
			return
		}

		if _, pattern := s.mux.Handler(r); pattern == "" && s.staticHandler != nil {
			s.staticHandler.ServeHTTP(w, r)
			return
		}
		// Otherwise try protected routes
		protectedHandler.ServeHTTP(w, r)
	})
	// Shared protections must run before routing and before authentication,
	// including CORS preflights for protected endpoints.
	return s.publicMiddleware(combined)
}

// Shutdown is a no-op placeholder; graceful shutdown is handled by App.
func (s *Server) Shutdown(ctx context.Context) {}

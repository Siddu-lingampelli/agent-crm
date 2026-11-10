package httpserver

import (
	"context"
	"crm.local/internal/auth"
	"crm.local/internal/model"
	"crm.local/internal/repository"
	"crm.local/internal/uuid"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type auditResponse struct {
	http.ResponseWriter
	status int
}

func (w *auditResponse) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *auditResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(body)
}
func (w *auditResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Record successful REST writes without request bodies, credentials or tokens.
// Authentication must wrap this middleware so the verified actor is available.
func AuditMiddleware(repo *repository.AdminRepository, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (r.Method != "POST" && r.Method != "PATCH" && r.Method != "DELETE" && r.Method != "PUT") || !strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1/mcp" {
				next.ServeHTTP(w, r)
				return
			}
			response := &auditResponse{ResponseWriter: w}
			next.ServeHTTP(response, r)
			if response.status < 200 || response.status >= 300 {
				return
			}
			actor := auth.GetAuth(r)
			if actor.ActorID == "" {
				return
			}
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
			resource := parts[0]
			id := r.PathValue("id")
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
			defer cancel()
			event := &model.AuditLog{ID: uuid.New(), WorkspaceID: actor.WorkspaceID, ActorType: actor.ActorType, ActorID: actor.ActorID, Action: strings.ToLower(r.Method), ResourceType: resource, ResourceID: id, OldValues: "{}", NewValues: "{}", IPAddress: r.RemoteAddr, UserAgent: r.UserAgent()[:min(len(r.UserAgent()), 512)], CreatedAt: time.Now().UTC()}
			if err := repo.CreateAuditLog(ctx, event); err != nil {
				log.Error("could not persist REST audit event", "error", err, "resource", resource)
			}
		})
	}
}

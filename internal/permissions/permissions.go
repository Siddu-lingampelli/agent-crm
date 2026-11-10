// Package permissions defines the scope vocabulary and checks.
// Authorization is deny-by-default and enforced in backend services.
package permissions

import "strings"

// AuthContext describes who is performing an action: a human user or an agent.
type AuthContext struct {
	ActorType   string // "user" or "agent"
	ActorID     string
	WorkspaceID string
	Scopes      []string
}

// HasScope reports whether scopes includes the required scope.
// An exact match is required; there is no wildcard expansion.
func HasScope(scopes []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return false
	}
	for _, s := range scopes {
		if strings.TrimSpace(s) == required {
			return true
		}
	}
	return false
}

// Authorize returns an error if ctx may not perform required scope.
func Authorize(ctx AuthContext, required string) error {
	if ctx.WorkspaceID == "" {
		return ErrNoWorkspace
	}
	if ctx.ActorID == "" {
		return ErrNoActor
	}
	if !HasScope(ctx.Scopes, required) {
		return ErrScopeDenied
	}
	return nil
}

// Sentinel errors for authorization failures.
var (
	ErrNoWorkspace = err("no workspace context")
	ErrNoActor     = err("no actor identity")
	ErrScopeDenied = err("scope denied")
)

func err(s string) error { return &permError{s} }

type permError struct{ s string }

func (e *permError) Error() string { return e.s }

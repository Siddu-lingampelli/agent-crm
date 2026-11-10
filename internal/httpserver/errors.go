package httpserver

import (
	"crm.local/internal/jsonutil"
	"crm.local/internal/permissions"
	"crm.local/internal/repository"
	"crm.local/internal/service"
	"errors"
	"net/http"
)

func writeKnownServiceError(w http.ResponseWriter, err error) bool {
	var validation *repository.ValidationError
	if errors.As(err, &validation) {
		jsonutil.Error(w, 400, "validation_failed", validation.Error())
		return true
	}
	switch {
	case errors.Is(err, service.ErrNotConfigured):
		jsonutil.Error(w, http.StatusNotImplemented, "not_configured", err.Error())
		return true
	case errors.Is(err, permissions.ErrScopeDenied), errors.Is(err, permissions.ErrNoActor), errors.Is(err, permissions.ErrNoWorkspace):
		jsonutil.Error(w, http.StatusForbidden, "forbidden", "permission denied")
		return true
	case errors.Is(err, repository.ErrInvalidReference):
		jsonutil.Error(w, http.StatusBadRequest, "invalid_reference", repository.ErrInvalidReference.Error())
		return true
	}
	return false
}

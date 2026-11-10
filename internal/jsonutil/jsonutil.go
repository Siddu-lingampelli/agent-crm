// Package jsonutil provides small, bounded JSON helpers for HTTP and MCP.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrTooLarge is returned when a payload exceeds the limit.
var ErrTooLarge = errors.New("payload exceeds max body size")

// MaxBodyBytes caps decoded request bodies defensively; callers apply their
// own stricter limits. This is a backstop against accidental unbounded reads.
const MaxBodyBytes = 8 << 20 // 8 MiB backstop

// Write encodes v as JSON with the given status. It never panics on unwritable
// values and always emits a consistent content type.
func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	// Write errors are unrecoverable at this point; the response is already
	// partially flushed, so we simply stop.
	_ = enc.Encode(v)
}

// Error writes the canonical error envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	Write(w, status, ErrorBody{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorBody is the canonical error envelope.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes a single error.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Decode reads at most limit bytes from r into v. Rejects unknown fields and
// trailing data so partial or malformed payloads fail instead of silently
// yielding zero values.
func Decode(r *http.Request, limit int64, v any) error {
	if limit <= 0 || limit > MaxBodyBytes {
		limit = MaxBodyBytes
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > limit {
		return ErrTooLarge
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return err
	}

	// Detect oversize payloads that were truncated by the limit.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must contain exactly one JSON value")
	}

	return nil
}

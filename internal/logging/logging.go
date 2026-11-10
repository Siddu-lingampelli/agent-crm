// Package logging provides structured logging helpers with secret redaction.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

// Redacted is the placeholder substituted for secret values in log output.
const Redacted = "[REDACTED]"

// sensitiveKeys are key fragments that trigger value redaction.
var sensitiveKeys = []string{
	"password", "passwd", "secret", "token", "apikey", "api_key",
	"authorization", "cookie", "session", "credential", "key",
}

// New builds a slog logger writing to stdout at the given format and level.
func New(format, level string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level), AddSource: true}

	var w io.Writer = os.Stdout
	var handler slog.Handler

	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Redact returns s with secret-looking values masked. Used for
// error messages that may embed credentials before reaching the log.
func Redact(s string) string {
	if s == "" {
		return s
	}
	out := s
	// Collapse obvious bearer/basic credentials.
	out = bearerRe.ReplaceAllString(out, "Bearer "+Redacted)
	out = basicRe.ReplaceAllString(out, "Basic "+Redacted)
	return out
}

var (
	bearerRe = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`)
	basicRe  = regexp.MustCompile(`(?i)basic\s+[A-Za-z0-9+/]+=*`)
)

// RedactAttr drops or masks attribute values whose key looks sensitive.
func RedactAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindString {
		return a
	}
	k := strings.ToLower(a.Key)
	for _, frag := range sensitiveKeys {
		if strings.Contains(k, frag) {
			return slog.String(a.Key, Redacted)
		}
	}
	// Mask embedded credentials inside otherwise ordinary strings.
	v := a.Value.String()
	if red := Redact(v); red != v {
		a.Value = slog.StringValue(red)
	}
	return a
}

// SafeError formats an error for logging, applying redaction.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	return Redact(fmt.Sprintf("%v", err))
}

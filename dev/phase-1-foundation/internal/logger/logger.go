// Package logger provides a structured slog logger and the canonical field
// names used across the service (architecture §23).
package logger

import (
	"io"
	"log/slog"
)

// Canonical structured log field names (architecture §23).
const (
	FieldRequestID  = "request_id"
	FieldUserID     = "user_id"
	FieldMethod     = "method"
	FieldPath       = "path"
	FieldStatusCode = "status_code"
	FieldDurationMS = "duration_ms"
	FieldOrderID    = "order_id"
	FieldError      = "error"
)

// New returns a JSON slog.Logger writing to w at the given level. Unknown
// levels fall back to info (LOG_LEVEL is validated in package config).
func New(level string, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(level)}))
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

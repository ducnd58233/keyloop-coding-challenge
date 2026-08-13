// Package observability holds the cross-cutting logging, metrics and tracing
// adapters (SPEC.md §3). Metrics and tracing arrive in T7; this file carries
// the structured logger every other package depends on from the start.
package observability

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// Logger is the logging port for adapters and infra. *slog.Logger satisfies
// it, so call sites depend on this interface rather than the concrete type.
type Logger interface {
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// NewLogger builds the process logger: stdlib log/slog with a JSON handler
// (SPEC.md §3), so every log line is a structured record from the first one
// emitted, not a later retrofit.
func NewLogger(level string, w io.Writer) *slog.Logger {
	lvl := new(slog.LevelVar)
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl.Set(slog.LevelDebug)
	case "warn":
		lvl.Set(slog.LevelWarn)
	case "error":
		lvl.Set(slog.LevelError)
	default:
		lvl.Set(slog.LevelInfo)
	}

	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
}

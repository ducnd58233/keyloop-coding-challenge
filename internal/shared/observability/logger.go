// Package observability holds logging, metrics and tracing adapters (SPEC.md §3).
// Metrics and tracing arrive in T7.
package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
)

const (
	defaultDir = "logs"
	dirPerm    = 0o750
	filePerm   = 0o600
)

// Logger is the logging port. *slog.Logger satisfies it so call sites do not
// depend on a concrete handler.
type Logger interface {
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// Options has no format switch: console is always tinted, the file always JSON.
type Options struct {
	Service string
	Level   string
	Stdout  io.Writer // defaults to os.Stdout
	Dir     string    // defaults to defaultDir; tests pass t.TempDir()
}

// NewLogger writes tinted console and a JSON file. Close the closer on shutdown.
func NewLogger(opt Options) (*slog.Logger, io.Closer, error) {
	service := strings.TrimSpace(opt.Service)
	if !validService(service) {
		return nil, nil, fmt.Errorf("observability: invalid service name %q", service)
	}

	level := new(slog.LevelVar)
	switch strings.ToLower(strings.TrimSpace(opt.Level)) {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	stdout := opt.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	dir := strings.TrimSpace(opt.Dir)
	if dir == "" {
		dir = defaultDir
	}

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, nil, fmt.Errorf("observability: mkdir %s: %w", dir, err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("observability: open log dir %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()
	f, err := root.OpenFile(service+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePerm)
	if err != nil {
		return nil, nil, fmt.Errorf("observability: open %s.log: %w", service, err)
	}

	h := slog.NewMultiHandler(
		tint.NewTextHandler(colorableWriter(stdout), &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
			NoColor:    false,
		}),
		slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level}),
	)
	// Always present so three processes in one terminal stay distinguishable.
	return slog.New(h).With(slog.String("name", service)), f, nil
}

// Discard drops all records. Content assertions use slog.NewTextHandler on a buffer.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func validService(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func colorableWriter(w io.Writer) io.Writer {
	if f, ok := w.(*os.File); ok {
		return colorable.NewColorable(f)
	}
	return w
}

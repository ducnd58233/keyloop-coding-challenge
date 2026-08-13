package observability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Capture records log lines for tests. It satisfies Logger.
type Capture struct {
	Lines []string
}

// Text joins captured lines with spaces.
func (c *Capture) Text() string {
	return strings.Join(c.Lines, " ")
}

// Info implements Logger.
func (c *Capture) Info(msg string, args ...any) {
	c.append("info", msg, args...)
}

// Warn implements Logger.
func (c *Capture) Warn(msg string, args ...any) {
	c.append("warn", msg, args...)
}

// Error implements Logger.
func (c *Capture) Error(msg string, args ...any) {
	c.append("error", msg, args...)
}

// InfoContext implements Logger.
func (c *Capture) InfoContext(_ context.Context, msg string, args ...any) {
	c.Info(msg, args...)
}

// WarnContext implements Logger.
func (c *Capture) WarnContext(_ context.Context, msg string, args ...any) {
	c.Warn(msg, args...)
}

// ErrorContext implements Logger.
func (c *Capture) ErrorContext(_ context.Context, msg string, args ...any) {
	c.Error(msg, args...)
}

func (c *Capture) append(level, msg string, args ...any) {
	var b strings.Builder
	b.WriteString(level)
	b.WriteByte(' ')
	b.WriteString(msg)
	for _, arg := range args {
		if a, ok := arg.(slog.Attr); ok {
			fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)
			continue
		}
		fmt.Fprintf(&b, " %v", arg)
	}
	c.Lines = append(c.Lines, b.String())
}

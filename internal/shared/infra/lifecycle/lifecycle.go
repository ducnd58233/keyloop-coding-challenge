// Package lifecycle runs start/stop hooks in order so a composition root
// can register postgres, redis, or OTel later without rewriting Run (R6).
// Stop always runs in reverse. cmd/ still owns signals; this package never
// calls os.Exit.
package lifecycle

import (
	"context"
	"fmt"
	"sync"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Hook is one process resource. Start or Stop may be nil.
type Hook struct {
	Name  string
	Start func(ctx context.Context) error
	Stop  func(ctx context.Context) error
}

// Chain is owned by one composition root, not a process global.
type Chain struct {
	mu      sync.Mutex
	log     observability.Logger
	hooks   []Hook
	started int
	stopped bool
}

// New does not start anything.
func New(log observability.Logger) *Chain {
	return &Chain{log: log}
}

// Add appends a hook. Call it before Start, or after a failed Start that
// already stopped earlier hooks (then Start again). After a successful Start,
// Add is ignored until Stop.
func (c *Chain) Add(h Hook) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started > 0 && !c.stopped {
		return
	}
	c.hooks = append(c.hooks, h)
}

// Start runs each Start in add order. A failure stops already-started hooks
// in reverse and returns that error.
func (c *Chain) Start(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		c.stopped = false
	}
	for i := c.started; i < len(c.hooks); i++ {
		h := c.hooks[i]
		c.info("lifecycle starting", h.Name)
		if h.Start != nil {
			if err := h.Start(ctx); err != nil {
				c.stopLocked(ctx, i-1)
				c.stopped = true
				c.started = 0
				return fmt.Errorf("lifecycle %s: %w", h.Name, err)
			}
		}
		c.started = i + 1
		c.info("lifecycle started", h.Name)
	}
	return nil
}

// Stop runs Stop on started hooks in reverse. It is safe to call twice.
func (c *Chain) Stop(ctx context.Context) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return
	}
	c.stopped = true
	c.stopLocked(ctx, c.started-1)
	c.started = 0
}

func (c *Chain) stopLocked(ctx context.Context, last int) {
	for i := last; i >= 0; i-- {
		h := c.hooks[i]
		c.info("lifecycle stopping", h.Name)
		if h.Stop != nil {
			if err := h.Stop(ctx); err != nil {
				c.error("lifecycle stop failed", h.Name, err)
			}
		}
		c.info("lifecycle stopped", h.Name)
	}
}

func (c *Chain) info(msg, name string) {
	if c.log == nil {
		return
	}
	c.log.Info(msg, "hook", name)
}

func (c *Chain) error(msg, name string, err error) {
	if c.log == nil {
		return
	}
	// hook name only plus a short error; callers must not put DSN or VIN in err.
	c.log.Error(msg, "hook", name, "error", err.Error())
}

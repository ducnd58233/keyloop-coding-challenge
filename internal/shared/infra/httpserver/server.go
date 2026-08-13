// Package httpserver holds the process HTTP server and the request/response
// helpers every module's api layer builds on (R1: modules never share this
// kind of code directly with each other, only through here).
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// readHeaderTimeout guards against a slow-header client tying up a
// connection; shutdownTimeout bounds how long an in-flight request gets
// during a graceful shutdown. Neither is a documented env var (SPEC.md §6)
// because both are server hygiene, not a tunable requirement.
const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// Serve runs h until ctx is cancelled, then drains in-flight requests for up
// to shutdownTimeout before returning.
func Serve(ctx context.Context, addr string, h http.Handler, l observability.Logger) error {
	s := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		l.Info("http server started", "address", addr)
		errChan <- s.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		l.Info("http server shutting down", "reason", ctx.Err())
		// ctx is already Done, so its own deadline cannot bound the shutdown;
		// WithoutCancel keeps any request-scoped values while dropping that
		// cancellation, and a fresh timeout gives Shutdown its own budget.
		sdCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		if err := s.Shutdown(sdCtx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}
		l.Info("http server stopped")
		return nil
	case err := <-errChan:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

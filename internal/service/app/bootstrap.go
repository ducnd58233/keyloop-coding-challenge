// Package app is the service-mock composition root (R6).
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ducnd58233/unified-document-viewer/internal/service/modules/service"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver/middleware"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockfault"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// RunOptions is the process flag surface. Faults are not read from the environment.
type RunOptions struct {
	Addr          string
	BaseURL       string
	Down          bool
	Deterministic bool
	Log           observability.Logger
}

// Run wires the service mock and serves until ctx is cancelled.
func Run(ctx context.Context, opt RunOptions) error {
	if opt.Log == nil {
		return fmt.Errorf("service: logger is required")
	}
	addr := opt.Addr
	if addr == "" {
		addr = ":9101"
	}

	fault := mockfault.Config{Down: opt.Down}
	generate := false
	if !opt.Deterministic && !opt.Down {
		fault = mockfault.Live()
		generate = true
	}

	opt.Log.Info("service mock listening",
		slog.String("addr", addr),
		slog.Bool("down", opt.Down),
		slog.Bool("deterministic", opt.Deterministic),
		slog.Float64("timeout_rate", fault.TimeoutRate),
		slog.Float64("error_rate", fault.ErrorRate),
		slog.Float64("latency_rate", fault.LatencyRate),
		slog.Duration("latency_min", fault.LatencyMin),
		slog.Duration("latency_max", fault.LatencyMax),
		slog.Duration("timeout_for", fault.TimeoutFor),
	)

	h := middleware.Chain(
		service.New(service.Options{
			Fault:    fault,
			BaseURL:  opt.BaseURL,
			Generate: generate,
			Log:      opt.Log,
		}),
		middleware.RequestID,
	)
	return httpserver.Serve(ctx, addr, h, opt.Log)
}

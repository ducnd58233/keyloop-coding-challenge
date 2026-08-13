// Package app is the sales-mock composition root (R6).
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	salesdocs "github.com/ducnd58233/unified-document-viewer/api/sales/http/docs"
	"github.com/ducnd58233/unified-document-viewer/internal/sales/modules/sales"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver/middleware"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/lifecycle"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/openapidocs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockfault"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// RunOptions is CLI only; faults are not read from the environment.
type RunOptions struct {
	Addr          string
	Down          bool
	Deterministic bool
}

// Run is the composition root; cmd/ only handles signals (R6).
func Run(ctx context.Context, opt RunOptions) error {
	logger, logClose, err := observability.NewLogger(observability.Options{
		Service: "sales",
		Level:   "info",
	})
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}

	lc := lifecycle.New(logger)
	if err := lc.Start(ctx); err != nil {
		_ = logClose.Close()
		return err
	}
	defer func() {
		lc.Stop(context.WithoutCancel(ctx))
		_ = logClose.Close()
	}()

	addr := opt.Addr
	if addr == "" {
		addr = ":9100"
	}

	fault := mockfault.Config{Down: opt.Down}
	generate := false
	if !opt.Deterministic && !opt.Down {
		fault = mockfault.Live()
		generate = true
	}

	logger.Info("sales mock config",
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

	mux := http.NewServeMux()
	openapidocs.Mount(mux, salesdocs.SwaggerInfo)
	mux.Handle("/", sales.New(sales.Options{Fault: fault, Generate: generate, Log: logger}))
	h := middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Recover(logger),
	)
	return httpserver.Serve(ctx, addr, h, logger)
}

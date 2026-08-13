// Package app is the document-viewer composition root (R6): it constructs
// adapters and wires them to ports for this binary only.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ducnd58233/unified-document-viewer/configs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Run is the composition root; cmd/ only handles signals (R6).
func Run(ctx context.Context) error {
	cfg, err := configs.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, logClose, err := observability.NewLogger(observability.Options{
		Service: "documentviewer",
		Level:   cfg.Log.Level,
	})
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer func() { _ = logClose.Close() }()
	slog.SetDefault(logger)

	// Fail fast if Postgres is down; T6 will inject this pool into cache and audit.
	pool, err := postgres.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	handler := mountHTTP(httpDeps{log: logger})

	return httpserver.Serve(ctx, cfg.HTTP.Address, handler, logger)
}

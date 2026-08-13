// Package app is the composition root (R6, AGENTS.md): the only place that
// constructs concrete adapter types and wires them to the ports they
// satisfy. Everything else depends on interfaces.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ducnd58233/unified-document-viewer/configs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Run loads configuration, wires the process, and serves until ctx is
// cancelled (SIGINT/SIGTERM from cmd/api).
func Run(ctx context.Context) error {
	cfg, err := configs.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := observability.NewLogger(cfg.Log.Level, os.Stdout)
	slog.SetDefault(logger)

	handler := mountHTTP(httpDeps{cfg: cfg, log: logger})

	return httpserver.Serve(ctx, cfg.HTTP.Address, handler, logger)
}

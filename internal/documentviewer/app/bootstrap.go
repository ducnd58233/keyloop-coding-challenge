// Package app is the document-viewer composition root (R6): it constructs
// adapters and wires them to ports for this binary only.
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
// cancelled (SIGINT/SIGTERM from cmd/documentviewer).
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

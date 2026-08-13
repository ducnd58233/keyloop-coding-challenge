package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	app "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("application stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

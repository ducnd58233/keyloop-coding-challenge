// Command service serves the Service System mock on :9101 (A5).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	app "github.com/ducnd58233/unified-document-viewer/internal/service/app"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func main() {
	addr := flag.String("addr", ":9101", "listen address")
	baseURL := flag.String("base-url", "http://localhost:9101", "absolute base used in file.uri")
	down := flag.Bool("down", false, "simulate a full outage: every route returns 503")
	deterministic := flag.Bool("deterministic", false, "stable catalog only; no random faults or generated VINs")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.NewLogger("info", os.Stdout)
	if err := app.Run(ctx, app.RunOptions{
		Addr:          *addr,
		BaseURL:       *baseURL,
		Down:          *down,
		Deterministic: *deterministic,
		Log:           logger,
	}); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("service mock stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

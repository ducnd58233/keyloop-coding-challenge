// Command sales serves the Sales System mock on :9100 (A5).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	app "github.com/ducnd58233/unified-document-viewer/internal/sales/app"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func main() {
	addr := flag.String("addr", ":9100", "listen address")
	down := flag.Bool("down", false, "simulate a full outage: every route returns 503")
	deterministic := flag.Bool("deterministic", false, "stable catalog only; no random faults or generated VINs")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.NewLogger("info", os.Stdout)
	if err := app.Run(ctx, app.RunOptions{
		Addr:          *addr,
		Down:          *down,
		Deterministic: *deterministic,
		Log:           logger,
	}); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("sales mock stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

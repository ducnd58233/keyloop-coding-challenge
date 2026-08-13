// mock-sales is one of two separate mock upstream servers (A5). This is the
// walking-skeleton version: it listens and answers /healthz. Document
// payloads, latency and error-rate fault injection arrive in T3.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func main() {
	addr := flag.String("addr", ":9100", "listen address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.NewLogger("info", os.Stdout)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if err := httpserver.Serve(ctx, *addr, mux, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("mock-sales stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

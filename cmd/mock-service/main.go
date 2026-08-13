// mock-service is one of two separate mock upstream servers (A5). This is
// the walking-skeleton version: it listens and answers /healthz, honouring
// -down so `make demo-degraded` (FR7) is meaningful from T2 onward. Document
// payloads and the remaining fault-injection flags arrive in T3.
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
	addr := flag.String("addr", ":9101", "listen address")
	down := flag.Bool("down", false, "simulate a full outage: every route returns 503")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.NewLogger("info", os.Stdout)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if *down {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := httpserver.Serve(ctx, *addr, mux, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("mock-service stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

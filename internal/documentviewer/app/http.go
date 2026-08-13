package app

import (
	"net/http"

	"github.com/ducnd58233/unified-document-viewer/configs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver/middleware"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

type httpDeps struct {
	cfg configs.Config
	log observability.Logger
}

// T6 will mount document routes here; the skeleton only exposes health.
func mountHTTP(d httpDeps) http.Handler {
	mux := http.NewServeMux()
	registerHealth(mux)

	return middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Recover(d.log),
	)
}

// /readyz stays a liveness alias until T5 adds a DB ping: alive ≠ ready.
func registerHealth(mux *http.ServeMux) {
	ok := func(w http.ResponseWriter, _ *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
	mux.HandleFunc("GET /healthz", ok)
	mux.HandleFunc("GET /readyz", ok)
}

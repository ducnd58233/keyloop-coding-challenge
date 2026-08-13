package app

import (
	"net/http"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver/middleware"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

type httpDeps struct {
	log observability.Logger
}

// T6 mounts document routes here; until then only liveness is exposed.
func mountHTTP(d httpDeps) http.Handler {
	mux := http.NewServeMux()
	registerHealth(mux)

	return middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Recover(d.log),
	)
}

func registerHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

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

// mountHTTP assembles the process mux. The documentviewer module mounts its
// own routes here once it has a handler to mount (T6); for now the walking
// skeleton only exposes the health surface.
func mountHTTP(d httpDeps) http.Handler {
	mux := http.NewServeMux()
	registerHealth(mux)

	return middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Recover(d.log),
	)
}

// registerHealth serves /healthz and /readyz. Both report process liveness
// only until shared/infra/postgres exists (T5), at which point /readyz gains
// a database ping - a service that answers requests but cannot reach its
// database is not ready, even though it is alive.
func registerHealth(mux *http.ServeMux) {
	ok := func(w http.ResponseWriter, _ *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
	mux.HandleFunc("GET /healthz", ok)
	mux.HandleFunc("GET /readyz", ok)
}

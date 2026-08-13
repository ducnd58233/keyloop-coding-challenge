package app

import (
	"net/http"
	"time"

	viewdocs "github.com/ducnd58233/unified-document-viewer/api/documentviewer/http/docs"
	docsapi "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/api"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver/middleware"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/openapidocs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

type httpDeps struct {
	log            observability.Logger
	docs           *docsapi.Handler
	requestTimeout time.Duration
}

func mountHTTP(d httpDeps) http.Handler {
	mux := http.NewServeMux()
	registerHealth(mux)
	openapidocs.Mount(mux, viewdocs.SwaggerInfo)
	if d.docs != nil {
		d.docs.Register(mux)
	}

	return middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Timeout(d.requestTimeout),
		middleware.Recover(d.log),
	)
}

type healthResponse struct {
	Status string `json:"status"`
}

func registerHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpserver.JSON(w, http.StatusOK, healthResponse{Status: "ok"})
	})
}

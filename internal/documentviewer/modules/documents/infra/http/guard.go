package http

import (
	"context"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/circuitbreaker"
)

type guardedSource struct {
	inner app.DocumentSource
	cb    *circuitbreaker.Breaker
}

// WithBreaker fail-fasts one DocumentSource. A nil breaker is a no-op.
func WithBreaker(inner app.DocumentSource, cb *circuitbreaker.Breaker) app.DocumentSource {
	if inner == nil || cb == nil {
		return inner
	}
	return &guardedSource{inner: inner, cb: cb}
}

func (g *guardedSource) Name() domain.SourceName { return g.inner.Name() }

func (g *guardedSource) Fetch(ctx context.Context, vin string) ([]domain.Document, error) {
	var docs []domain.Document
	err := g.cb.Execute(func() error {
		var e error
		docs, e = g.inner.Fetch(ctx, vin)
		return e
	})
	return docs, err
}

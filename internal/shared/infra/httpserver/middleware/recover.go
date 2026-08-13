package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Recover turns a panic into a logged 500 instead of a crashed process.
// INTERNAL_ERROR is the process safety net; it is not part of the FR7 per-source list.
func Recover(l observability.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if x := recover(); x != nil {
					// Type only: panic values can carry a VIN or *url.Error.
					l.ErrorContext(ctx, "panic recovered",
						"panic_type", fmt.Sprintf("%T", x),
						"stack", string(debug.Stack()),
					)
					httpserver.JSON(w, http.StatusInternalServerError, map[string]any{
						"error": map[string]string{"code": "INTERNAL_ERROR"},
					})
				}
			}(r.Context())
			next.ServeHTTP(w, r)
		})
	}
}

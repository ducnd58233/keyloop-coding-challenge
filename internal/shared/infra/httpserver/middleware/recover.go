package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Recover turns a panic in any handler into a logged 500 instead of a
// crashed process. INTERNAL_ERROR is the one error code outside the FR7
// per-source vocabulary in SPEC.md §2, added there for this safety net.
func Recover(l observability.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if x := recover(); x != nil {
					l.ErrorContext(r.Context(), "panic recovered", "panic", x, "stack", string(debug.Stack()))
					httpserver.JSON(w, http.StatusInternalServerError, map[string]any{
						"error": map[string]string{"code": "INTERNAL_ERROR"},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

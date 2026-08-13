package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

const msgInternal = "internal server error"

type errorResponse struct {
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Recover turns a panic into a logged 500 instead of a crashed process.
// Status is the code; the body never carries INTERNAL_ERROR or panic text.
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
					httpserver.JSON(w, http.StatusInternalServerError, errorResponse{Message: msgInternal})
				}
			}(r.Context())
			next.ServeHTTP(w, r)
		})
	}
}

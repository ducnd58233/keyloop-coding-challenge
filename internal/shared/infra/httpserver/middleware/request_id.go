package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
)

// RequestID honours an inbound X-Request-Id (SPEC.md §2) or generates one,
// echoes it on the response, and carries it on the request context so
// logging and the audit trail (A10, FR8) can correlate on it.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(httpserver.WithRequestID(r.Context(), id)))
	})
}

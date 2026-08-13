package middleware

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
)

const maxRequestIDLen = 128

// RequestID honours inbound X-Request-Id (SPEC.md §2) or generates one.
// Length is clipped so FR8 cannot persist an unbounded header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := clipRequestID(strings.TrimSpace(r.Header.Get("X-Request-Id")))
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(httpserver.WithRequestID(r.Context(), id)))
	})
}

func clipRequestID(id string) string {
	if utf8.RuneCountInString(id) <= maxRequestIDLen {
		return id
	}
	return string([]rune(id)[:maxRequestIDLen])
}

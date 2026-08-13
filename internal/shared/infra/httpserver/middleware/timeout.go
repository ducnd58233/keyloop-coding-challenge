package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout applies NFR4 to the request context. Handlers and upstream calls
// honour ctx; a zero duration is a no-op so tests can omit the budget.
func Timeout(d time.Duration) Middleware {
	if d <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Package middleware holds the HTTP middleware every module's api layer
// composes onto the process mux (request ID, panic recovery).
package middleware

import "net/http"

// Middleware wraps a handler to run logic before and/or after it.
type Middleware func(http.Handler) http.Handler

// Chain applies m in the order given, so the first middleware listed is the
// outermost: it sees the request first and the response last.
func Chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

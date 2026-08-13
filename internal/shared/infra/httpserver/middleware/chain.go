// Package middleware is shared HTTP wrappers (request ID, panic recovery).
package middleware

import "net/http"

// Middleware is composed by Chain.
type Middleware func(http.Handler) http.Handler

// Chain applies m so the first listed is outermost (sees the request first).
func Chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

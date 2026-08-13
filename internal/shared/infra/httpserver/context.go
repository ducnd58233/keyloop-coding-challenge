package httpserver

import "context"

type key string

const requestIDKey key = "request_id"

// WithRequestID stores id on ctx for logs and audit (A10, FR8).
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFrom is empty when RequestID middleware has not run.
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

package httpserver

import "context"

type key string

const requestIDKey key = "request_id"

// WithRequestID attaches the request ID to ctx.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFrom returns the request ID carried on ctx, or "" if none.
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

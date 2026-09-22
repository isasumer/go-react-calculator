package httpapi

import "context"

// ctxKey is the unexported key type for this package's context values, so a
// value stored here cannot be read — or overwritten — by accident.
type ctxKey int

// requestIDKey holds the request's X-Request-ID.
const requestIDKey ctxKey = iota

// ContextWithRequestID returns a copy of ctx carrying the request's ID.
//
// The request-ID middleware calls this once per request and [Write] reads the
// value back, so every problem document echoes the ID the client was given
// without the caller having to thread it through. The key lives here rather
// than in internal/middleware because that package imports this one for the
// problem writer, and the reverse import would be a cycle;
// middleware.FromContext is the accessor handlers use.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request ID stored by
// [ContextWithRequestID], or "" when there is none — a handler exercised in a
// test without the middleware chain, for example.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

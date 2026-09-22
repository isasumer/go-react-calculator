// Package middleware holds the net/http middleware chain that wraps the
// router: panic recovery, request IDs, structured access logging, a
// per-request timeout, security headers, CORS, per-client rate limiting and
// Prometheus metrics.
//
// Each concern is one file with one constructor returning a [Middleware], and
// [Chain] composes them in the order a request travels through them
// (outermost first):
//
//	Recover → RequestID → Logger → Timeout → SecurityHeaders → CORS → RateLimit → Metrics → router
//
// The order is deliberate. Recover is outermost so a panic anywhere — in a
// middleware as much as in a handler — still becomes a problem+json 500.
// RequestID comes next so everything below it, including the recovery
// handler, can put the ID in the response and in its logs. Logger sits above
// the timeout so a timed-out request is still logged exactly once, with the
// 503 the client actually received. The price of having Recover outside
// Logger is that a panicking request produces no access log line — it unwinds
// past the logger before there is a status to report — and is logged once by
// Recover instead, at error level with the same request ID. Security headers and CORS run before the
// rate limiter so a rejected request is still a well-formed browser response,
// and the rate limiter is the last gate before the router so a client over
// its budget costs nothing but a token lookup.
//
// Every error this package returns to a client is an
// [github.com/isasumer/go-react-calculator/backend/internal/httpapi.Problem]
// with a stable code from docs/errors.md: INTERNAL, TIMEOUT or RATE_LIMITED.
package middleware

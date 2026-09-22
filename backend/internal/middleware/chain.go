package middleware

import "net/http"

// A Middleware wraps a handler with one cross-cutting concern and returns the
// wrapped handler.
type Middleware func(http.Handler) http.Handler

// Chain wraps h with mws and returns the outermost handler. The first
// middleware listed is the outermost one, so the call reads in the order a
// request travels and the reverse order a response does:
//
//	Chain(router, Recover(log), RequestID(), Logger(log))
//
// is Recover → RequestID → Logger → router.
//
// A nil entry is skipped. That is what keeps a slot in the chain honest: the
// composition root can declare where the metrics middleware (B1-05) goes
// before it exists, instead of leaving the position to a comment.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		if mws[i] != nil {
			h = mws[i](h)
		}
	}
	return h
}

// operationalPaths are the endpoints the platform calls, not API clients.
// They are logged at debug and never rate limited: a probe throttled because
// a client is hammering the API would take a healthy instance out of rotation
// exactly when it is busiest. /metrics is listed ahead of B1-05 so the
// scraper is covered the moment the endpoint exists.
var operationalPaths = map[string]bool{
	"/healthz": true,
	"/readyz":  true,
	"/metrics": true,
}

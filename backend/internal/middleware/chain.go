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
// A nil entry is skipped, so a layer that is not configured — [Metrics]
// without a registry, say — is left out by passing nil rather than by
// rebuilding the list, and the order stays one readable expression.
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
// exactly when it is busiest. /metrics is here for the same reason: a scrape
// that is throttled is a gap in the graph precisely during the traffic spike
// the graph exists to show.
var operationalPaths = map[string]bool{
	"/healthz": true,
	"/readyz":  true,
	"/metrics": true,
}

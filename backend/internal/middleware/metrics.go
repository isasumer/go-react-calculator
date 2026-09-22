package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// routeUnmatched is the route label for a response the router never
// produced. With the chain assembled as cmd/server assembles it — metrics
// innermost, directly around the router — that is only a request the mux
// itself did not match. The API's catch-all pattern makes even a 404 a
// matched route ("/"), which is the point: the label set is the router's,
// not the internet's.
const routeUnmatched = "unmatched"

// Metrics records every request that reaches it in the Prometheus families
// m owns: the request counter, the latency histogram and the in-flight
// gauge.
//
// It sits innermost in the chain, directly around the router, for the label:
// http.ServeMux records the pattern it matched on the request, so reading
// r.Pattern *after* the inner handler has returned gives the route
// ("/api/v1/calculate") rather than the URL ("/api/v1/calculate?x=1", or one
// of the unbounded number of paths a 404 can have). A metric labeled by raw
// path is a time series per URL a client invents, which is how a metrics
// backend is taken down from the outside.
//
// The price of being innermost is that what the layers above answer is not
// counted here: a CORS preflight, a 429 from the rate limiter, and the 503
// [Timeout] sends when a handler runs long — that last one is counted as
// whatever status the handler produces when it eventually returns, which is
// not the status the client received. Those failures have their own signals
// (the access log, and the RATE_LIMITED and TIMEOUT problem counts), and
// moving metrics outward would cost the route label, because above the
// router r.Pattern is still empty. The trade is recorded in
// docs/adr/0006-runtime-stack.md.
//
// It skips nothing below it. The probes and /metrics itself are counted:
// three label sets and an atomic add, against the value of seeing at a
// glance that the scraper is scraping and the probes are green.
//
// A nil m returns the handler unwrapped, so a chain stays assemblable in a
// test that does not care about metrics.
func Metrics(m *observability.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		if m == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			m.IncInFlight()
			// Deferred, not called after ServeHTTP: a panic unwinds past
			// this middleware to Recover, and a gauge that only comes back
			// down on the happy path drifts up forever.
			defer m.DecInFlight()

			rec := newRecorder(w)
			next.ServeHTTP(rec, r)

			m.ObserveRequest(r.Method, routeOf(r), rec.status, time.Since(start))
		})
	}
}

// routeOf is the request's route label: the path part of the pattern the
// router matched, or [routeUnmatched] when nothing did.
//
// http.Request.Pattern is the whole registered pattern, "[METHOD ][HOST]/path"
// — "POST /api/v1/calculate" for this API. The method is already a label of
// its own, so leaving it in the route would split every route in two and make
// sum by (route) count each request twice; the host is one value per
// deployment. What is left is the path pattern, wildcards and all, which is
// the dimension a dashboard groups by.
func routeOf(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" {
		return routeUnmatched
	}
	if _, rest, found := strings.Cut(pattern, " "); found {
		pattern = rest
	}
	// Drop a host prefix, keeping the leading slash of the path itself.
	if i := strings.Index(pattern, "/"); i > 0 {
		pattern = pattern[i:]
	}
	return pattern
}

package middleware

import "net/http"

// The preflight answer. The API has exactly one POST endpoint, one GET
// endpoint and the preflight itself; Content-Type is what makes a calculate
// request non-simple, and X-Request-ID lets a caller carry its own trace in.
// Ten minutes of Max-Age is long enough that a session costs one preflight
// and short enough that widening the allowlist takes effect the same day.
const (
	allowMethods = "POST, GET, OPTIONS"
	allowHeaders = "Content-Type, X-Request-ID"
	maxAgeSecs   = "600"
)

// CORS answers cross-origin browser requests for the origins in allowed, and
// only those.
//
// An empty allowlist — the production configuration, where nginx serves the
// frontend from the same origin as the API (ADR-0009) — disables the
// middleware entirely: the handler is returned unwrapped, so no response
// carries a CORS header, not even Vary. The allowlist exists for local
// development, where the Vite dev server is a different origin.
//
// The allowed origin is echoed, never "*": the allowlist is the contract, and
// a wildcard would silently start allowing everything the day someone adds
// credentials. A disallowed origin gets no CORS headers at all and the
// request is served normally — the browser is what enforces CORS, and
// pretending the route does not exist would only make debugging harder.
func CORS(allowed []string) Middleware {
	origins := make(map[string]bool, len(allowed))
	for _, origin := range allowed {
		origins[origin] = true
	}
	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			// Every answer from here on depends on the Origin header, even
			// the ones that ignore it: without Vary a shared cache could
			// hand one origin the response computed for another.
			h.Add("Vary", "Origin")

			origin := r.Header.Get("Origin")
			if origin == "" || !origins[origin] {
				next.ServeHTTP(w, r)
				return
			}
			h.Set("Access-Control-Allow-Origin", origin)

			if r.Method != http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			// A preflight is answered here and never reaches the router: the
			// API defines no OPTIONS route, so the only other outcome would
			// be the 405 the router gives a disallowed origin.
			h.Set("Access-Control-Allow-Methods", allowMethods)
			h.Set("Access-Control-Allow-Headers", allowHeaders)
			h.Set("Access-Control-Max-Age", maxAgeSecs)
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

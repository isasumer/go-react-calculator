package middleware

import (
	"net/http"
	"strings"
)

// contentSecurityPolicy is the policy for a host that serves JSON and
// nothing else: load nothing, and refuse to be framed even by browsers that
// only understand CSP. frame-ancestors is not covered by default-src, which
// is why it is spelled out next to X-Frame-Options rather than instead of it.
const contentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'"

// apiPrefix is the versioned API surface. Only these answers are per-request
// data worth forbidding a cache to keep; the operational endpoints set
// Cache-Control themselves.
const apiPrefix = "/api/"

// SecurityHeaders sets the response headers that cost nothing and remove a
// class of browser-side mistakes:
//
//   - X-Content-Type-Options: nosniff — a browser must believe the declared
//     application/json instead of guessing something executable from the body.
//   - X-Frame-Options: DENY and a frame-ancestors 'none' policy — this origin
//     is never framed, so a clickjacking overlay has nothing to overlay.
//   - Referrer-Policy: no-referrer — a request path can carry operands; it
//     does not need to travel to whatever a page links to next.
//   - Content-Security-Policy: default-src 'none' — nothing here should ever
//     load a script, a font or an image; if a response somehow renders, it
//     renders inert.
//   - Cache-Control: no-store on /api/ — a calculation is cheap to redo and a
//     stale answer in a shared cache is worse than no answer.
//
// They are set on the way in, before the handler runs, so an error response
// carries them as much as a successful one.
func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", contentSecurityPolicy)
			if strings.HasPrefix(r.URL.Path, apiPrefix) {
				h.Set("Cache-Control", "no-store")
			}
			next.ServeHTTP(w, r)
		})
	}
}

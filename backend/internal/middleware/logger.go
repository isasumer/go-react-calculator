package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Logger writes one structured line per request, after the response, with the
// fields an operator greps for: the request ID that is also in the client's
// copy of the error, the method and path, the route pattern the router
// matched (so lines aggregate per route rather than per URL), the status,
// the response size, how long it took, and who asked.
//
// One line per request, not one on the way in and one on the way out: the
// interesting fields — status, size, duration — only exist afterwards, and a
// pair of lines doubles the log bill to say the same thing.
//
// The level follows the status, so an alert can be "count of level=error":
// 5xx is an error, 4xx a warning (the client's fault, worth seeing but not
// worth paging), everything else info. The operational endpoints drop to
// debug when they answer normally: a probe every few seconds would otherwise
// be the entire log. They keep the status levels when they fail, so a
// draining /readyz is visible for as long as it lasts.
//
// A request whose handler panics has no line here: the panic unwinds past
// this middleware before there is a status to log, and [Recover] — which
// wraps this one — logs it instead.
//
// A nil logger discards; the chain stays assemblable in a test that does not
// care about output.
func Logger(log *slog.Logger) Middleware {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newRecorder(w)

			next.ServeHTTP(rec, r)

			// http.ServeMux records the pattern it matched on the request it
			// was handed — ours, because [Timeout] hands its copy's value
			// back. It is empty when nothing matched, or when the response
			// never reached the router (a preflight, a 429, a timeout).
			log.LogAttrs(r.Context(), levelFor(rec.status, r.URL.Path), "request",
				slog.String("request_id", requestIDOf(w, r)),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("route", r.Pattern),
				slog.Int("status", rec.status),
				slog.Int64("bytes", rec.bytes),
				slog.Float64("duration_ms", milliseconds(time.Since(start))),
				slog.String("remote_ip", remoteIP(r.RemoteAddr)),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// levelFor picks the level for one finished request. Status first, so a
// failing probe is never hidden at debug.
func levelFor(status int, path string) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	case operationalPaths[path]:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// milliseconds renders a duration the way a dashboard wants it: milliseconds
// with microsecond resolution, as a number rather than Go's 1.234567ms
// string, so it can be aggregated without parsing.
func milliseconds(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

// remoteIP is the host part of RemoteAddr — the peer the kernel accepted the
// connection from, which behind a proxy is the proxy. It is deliberately not
// the X-Forwarded-For value: the log records what happened, and a header
// anyone can set is not that. The rate limiter is where trusting the header
// is a decision (TRUST_PROXY_HEADERS), because there it changes behavior
// rather than a field.
func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// A listener that is not TCP (a unix socket in a test) has no port.
		return remoteAddr
	}
	return host
}

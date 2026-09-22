package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/isasumer/go-react-calculator/backend/internal/httpapi"
)

// Recover turns a panic into a problem+json 500 instead of a dropped
// connection, and keeps the process serving the next request.
//
// It is the outermost middleware, so it covers the rest of the chain as well
// as the handlers. What the client gets is the generic INTERNAL problem from
// the error catalog with its request ID; what the panic actually was, and
// the stack it came from, goes to the log at error level under that same ID.
// The two are joined by the ID and nothing else: a panic value can contain
// anything the process knows.
//
// http.ErrAbortHandler is re-panicked untouched. It is how a handler says
// "stop, do not answer this"; net/http's own recovery expects to see it and
// closes the connection without logging.
//
// A nil logger discards.
func Recover(log *slog.Logger) Middleware {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := newRecorder(w)
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				log.LogAttrs(r.Context(), slog.LevelError, "panic recovered",
					slog.String("request_id", requestIDOf(w, r)),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Any("panic", v),
					slog.String("stack", string(debug.Stack())),
				)
				if rec.wroteHeader {
					// The status line is already on the wire: a second one
					// would only produce a "superfluous WriteHeader" log and
					// a body the client cannot parse. The truncated response
					// is the honest outcome; the log line has the reason.
					return
				}
				httpapi.Write(rec, r, httpapi.NewProblem(httpapi.CodeInternal,
					"an unexpected error occurred"))
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

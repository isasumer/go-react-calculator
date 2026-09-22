package middleware

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/isasumer/go-react-calculator/backend/internal/httpapi"
)

// timeoutDetail is deliberately free of numbers: the budget is a deployment
// setting, and a client can do nothing with it but retry.
const timeoutDetail = "the server took too long to produce a response"

// Timeout bounds how long a handler may take. The handler runs on its own
// goroutine with a context carrying the deadline; when the deadline passes
// first, the client gets a problem+json 503 with code TIMEOUT and the
// handler's later writes are dropped.
//
// Not http.TimeoutHandler: its timeout response is an HTML page written as
// text/html, and every error this API returns is application/problem+json
// with a stable code (ADR-0005). A client that branches on `code` would have
// to special-case one response shape for one failure mode. The mechanics are
// TimeoutHandler's, because they are the right ones: run the handler
// elsewhere, hand it a context that expires, and put a mutex between it and
// the response so it cannot write over the timeout answer afterwards.
//
// A timeout of zero or less disables the middleware — the handler is returned
// unwrapped, so REQUEST_TIMEOUT=0 costs nothing rather than expiring
// immediately.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		if d <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			inner := r.WithContext(ctx)
			tw := &timeoutWriter{ResponseWriter: w, staged: make(http.Header)}
			done := make(chan struct{})
			panicked := make(chan any, 1)

			go func() {
				defer func() {
					if v := recover(); v != nil {
						panicked <- v
					}
				}()
				next.ServeHTTP(tw, inner)
				close(done)
			}()

			select {
			case v := <-panicked:
				// Re-panic on this goroutine: a panic on the handler's own
				// goroutine would take the process down, because Recover is
				// not on its stack.
				panic(v)

			case <-done:
				// The router records the pattern it matched on the request
				// it was handed, which is our copy. Hand it back so the
				// logger outside can report the route. Reading inner after
				// <-done is safe: the handler has returned.
				r.Pattern = inner.Pattern
				tw.finish()

			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
					// The client hung up. Shut the writer so the handler
					// stops writing into a dead connection, but do not try
					// to answer someone who is no longer there.
					tw.stop()
					return
				}
				if tw.stop() {
					// The handler had already started its response; there is
					// no way to replace it with a 503 now.
					return
				}
				httpapi.Write(w, r, httpapi.NewProblem(httpapi.CodeTimeout, timeoutDetail))
			}
		})
	}
}

// timeoutWriter is the ResponseWriter the handler sees. Everything it does
// goes through the mutex, so the goroutine watching the deadline can close
// the writer and know that no write is in flight.
//
// The handler gets a header map of its own rather than the response's: it
// keeps running after a timeout, and a handler still setting headers on a
// response that is already on the wire is a data race in every sense. The
// staged map is copied over the real one the moment the handler commits a
// status, which is the last moment it can still win the race.
//
// The cost is that a timed-out response carries only the headers set outside
// this middleware — the request ID, which is what correlates it with the log
// — and not the ones the security or CORS layers staged inside it.
// http.TimeoutHandler drops them for the same reason.
type timeoutWriter struct {
	http.ResponseWriter

	mu          sync.Mutex
	staged      http.Header
	wroteHeader bool
	closed      bool
}

// Header returns the staged map; see the type comment.
func (tw *timeoutWriter) Header() http.Header { return tw.staged }

func (tw *timeoutWriter) WriteHeader(status int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.writeHeaderLocked(status)
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.closed {
		return 0, http.ErrHandlerTimeout
	}
	tw.writeHeaderLocked(http.StatusOK)
	return tw.ResponseWriter.Write(b)
}

// Flush commits the status line, so it counts as writing the header.
func (tw *timeoutWriter) Flush() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.closed {
		return
	}
	tw.writeHeaderLocked(http.StatusOK)
	_ = http.NewResponseController(tw.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the real writer.
func (tw *timeoutWriter) Unwrap() http.ResponseWriter { return tw.ResponseWriter }

func (tw *timeoutWriter) writeHeaderLocked(status int) {
	if tw.closed || tw.wroteHeader {
		return
	}
	tw.wroteHeader = true
	tw.commitHeaderLocked()
	tw.ResponseWriter.WriteHeader(status)
}

// commitHeaderLocked copies what the handler staged onto the real response,
// leaving headers set outside this middleware in place unless the handler
// chose the same name.
func (tw *timeoutWriter) commitHeaderLocked() {
	dst := tw.ResponseWriter.Header()
	for name, values := range tw.staged {
		dst[name] = values
	}
}

// finish is the in-time path: the handler returned, so whatever it staged is
// final. A handler that set headers and wrote nothing still gets them, on
// net/http's implicit 200.
func (tw *timeoutWriter) finish() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.wroteHeader {
		tw.commitHeaderLocked()
	}
	tw.closed = true
}

// stop closes the writer to anything the handler does from here on and
// reports whether it had already started the response. It blocks until a
// write in flight has finished, so afterwards the caller owns the underlying
// writer outright and can answer without racing the handler.
func (tw *timeoutWriter) stop() (started bool) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.closed = true
	return tw.wroteHeader
}

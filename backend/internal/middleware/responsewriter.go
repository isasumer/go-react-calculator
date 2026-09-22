package middleware

import "net/http"

// recorder wraps a ResponseWriter to remember what went out: the status the
// access log reports, how many body bytes followed it, and whether anything
// was written at all — which is how [Recover] knows a panic left the response
// untouched and a problem document can still be sent.
//
// Flush and Unwrap are implemented so http.ResponseController keeps reaching
// the real writer through however many layers of the chain wrap it.
type recorder struct {
	http.ResponseWriter

	status      int
	bytes       int64
	wroteHeader bool
}

func newRecorder(w http.ResponseWriter) *recorder {
	// 200 is what net/http sends when a handler writes a body without a
	// status, and what it logs for a handler that writes nothing at all.
	return &recorder{ResponseWriter: w, status: http.StatusOK}
}

func (rec *recorder) WriteHeader(status int) {
	if rec.wroteHeader {
		// net/http would warn about a superfluous WriteHeader; the first
		// status is the one the client got, so keep it.
		return
	}
	rec.wroteHeader = true
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

func (rec *recorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		rec.WriteHeader(http.StatusOK)
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += int64(n)
	return n, err
}

// Flush sends what has been written so far. A flush commits the status line,
// so it counts as having written the header.
func (rec *recorder) Flush() {
	if !rec.wroteHeader {
		rec.WriteHeader(http.StatusOK)
	}
	// A writer that cannot flush is not an error: the response is simply
	// delivered when the handler returns.
	_ = http.NewResponseController(rec.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the writer underneath for the
// interfaces this type does not implement itself (hijacking, deadlines).
func (rec *recorder) Unwrap() http.ResponseWriter { return rec.ResponseWriter }

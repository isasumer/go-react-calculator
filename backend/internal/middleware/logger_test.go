package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// logged runs one request through Logger with a JSON handler writing to a
// buffer — the LOG_FORMAT=json production setup — and returns the decoded
// line. A debug level is used so nothing is filtered out before it is seen.
func logged(t *testing.T, r *http.Request, next http.Handler) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	Chain(next, RequestID(), Logger(log)).ServeHTTP(httptest.NewRecorder(), r)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, buf.String())
	}
	return entry
}

func TestLoggerFields(t *testing.T) {
	// A real router, so the logged route is the pattern it matched and not
	// something the test made up.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/calculate", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("0123456789"))
	})

	r := newRequest(t, http.MethodPost, "/api/v1/calculate")
	r.Header.Set("User-Agent", "calculator-tests/1.0")
	r.RemoteAddr = "203.0.113.7:54321"

	entry := logged(t, r, mux)

	want := map[string]any{
		"msg":        "request",
		"level":      "INFO",
		"request_id": exampleRequestID,
		"method":     http.MethodPost,
		"path":       "/api/v1/calculate",
		"route":      "POST /api/v1/calculate",
		"status":     float64(http.StatusCreated),
		"bytes":      float64(10),
		"remote_ip":  "203.0.113.7",
		"user_agent": "calculator-tests/1.0",
	}
	for key, wantValue := range want {
		if got := entry[key]; got != wantValue {
			t.Errorf("%s = %v, want %v", key, got, wantValue)
		}
	}
	if ms, ok := entry["duration_ms"].(float64); !ok || ms < 0 {
		t.Errorf("duration_ms = %v, want a non-negative number", entry["duration_ms"])
	}
}

func TestLoggerLevelByStatusAndPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
		want   string
	}{
		{"success", "/api/v1/calculate", http.StatusOK, "INFO"},
		{"client error", "/api/v1/calculate", http.StatusBadRequest, "WARN"},
		{"rate limited", "/api/v1/calculate", http.StatusTooManyRequests, "WARN"},
		{"server error", "/api/v1/calculate", http.StatusInternalServerError, "ERROR"},
		{"liveness probe", "/healthz", http.StatusOK, "DEBUG"},
		{"readiness probe", "/readyz", http.StatusOK, "DEBUG"},
		{"metrics scrape", "/metrics", http.StatusOK, "DEBUG"},
		{"draining readiness probe stays visible", "/readyz", http.StatusServiceUnavailable, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := logged(t, newRequest(t, http.MethodGet, tt.path),
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.status)
				}))
			if got := entry["level"]; got != tt.want {
				t.Errorf("level = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLoggerDefaultsToTwoHundred: a handler that writes a body without a
// status, and one that writes nothing at all, are both 200s.
func TestLoggerDefaultsToTwoHundred(t *testing.T) {
	for name, handler := range map[string]http.Handler{
		"body without a status": ok(),
		"nothing at all":        http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	} {
		t.Run(name, func(t *testing.T) {
			entry := logged(t, newRequest(t, http.MethodGet, "/api/v1/operations"), handler)
			if got := entry["status"]; got != float64(http.StatusOK) {
				t.Errorf("status = %v, want 200", got)
			}
			if got := entry["route"]; got != "" {
				t.Errorf("route = %v, want empty when nothing routed the request", got)
			}
		})
	}
}

// TestLoggerNilIsUsable: the chain must assemble without a logger.
func TestLoggerNilIsUsable(t *testing.T) {
	rec := httptest.NewRecorder()
	Logger(nil)(ok()).ServeHTTP(rec, newRequest(t, http.MethodGet, "/"))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// TestRecorderFlushAndUnwrap: the wrapper must not break a handler that
// flushes, and http.ResponseController has to reach the real writer through
// however many recorders wrap it.
func TestRecorderFlushAndUnwrap(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	rec := httptest.NewRecorder()
	// Two recorders deep: Recover's and Logger's, as in the real chain.
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("first"))
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush through the chain: %v", err)
		}
		_, _ = w.Write([]byte("second"))
	}), Recover(log), Logger(log))
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, "/api/v1/operations"))

	if !rec.Flushed {
		t.Error("the flush never reached the real writer")
	}
	if got := rec.Body.String(); got != "firstsecond" {
		t.Errorf("body = %q", got)
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["bytes"] != float64(11) {
		t.Errorf("bytes = %v, want 11", entry["bytes"])
	}
}

// TestRecorderFlushWithoutWriteCommitsTheStatus covers a handler that flushes
// before writing anything: the status line is on the wire either way.
func TestRecorderFlushWithoutWriteCommitsTheStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := newRecorder(rec)
	wrapped.Flush()
	if !wrapped.wroteHeader || wrapped.status != http.StatusOK {
		t.Errorf("after Flush: wroteHeader=%v status=%d", wrapped.wroteHeader, wrapped.status)
	}
	// A second status is ignored, as net/http would ignore it.
	wrapped.WriteHeader(http.StatusTeapot)
	if wrapped.status != http.StatusOK {
		t.Errorf("status = %d, want the first one", wrapped.status)
	}
}

// TestResponseControllerUnwrapsThroughTheChain: the wrappers only implement
// Flush themselves, so everything else — deadlines, hijacking — reaches the
// real writer through Unwrap. Against a real server, because httptest's
// recorder supports none of it.
func TestResponseControllerUnwrapsThroughTheChain(t *testing.T) {
	failures := make(chan error, 1)
	srv := httptest.NewServer(Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Two recorders and the timeout writer sit between this and the
		// connection.
		failures <- http.NewResponseController(w).SetWriteDeadline(time.Now().Add(time.Minute))
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}), Recover(nil), RequestID(), Logger(nil), Timeout(time.Minute)))
	defer srv.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := <-failures; err != nil {
		t.Errorf("SetWriteDeadline through the chain: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRemoteIP(t *testing.T) {
	tests := map[string]string{
		"203.0.113.7:54321":   "203.0.113.7",
		"[2001:db8::1]:54321": "2001:db8::1",
		"/tmp/server.sock":    "/tmp/server.sock", // not host:port at all
	}
	for addr, want := range tests {
		if got := remoteIP(addr); got != want {
			t.Errorf("remoteIP(%q) = %q, want %q", addr, got, want)
		}
	}
}

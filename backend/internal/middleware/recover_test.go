package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const secret = "connection string: postgres://user:hunter2@db/prod"

func TestRecoverWritesAProblem(t *testing.T) {
	tests := []struct {
		name       string
		panicValue any
	}{
		{"string", secret},
		{"error", io.ErrUnexpectedEOF},
		{"runtime error", nil}, // triggered by the handler below
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&logs, nil))

			h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				if tt.panicValue == nil {
					operands, i := []float64{1, 2}, 7
					_ = operands[i] // panics: index out of range
					return
				}
				panic(tt.panicValue)
			}), Recover(log), RequestID())

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newRequest(t, http.MethodPost, "/api/v1/calculate"))

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q", ct)
			}
			var problem struct {
				Code      string `json:"code"`
				Detail    string `json:"detail"`
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Code != "INTERNAL" {
				t.Errorf("code = %q, want INTERNAL", problem.Code)
			}
			// The ID the client is told to quote is the one on the response
			// and in the log line, which is the whole point of the pairing.
			if problem.RequestID != exampleRequestID {
				t.Errorf("requestId = %q, want %q", problem.RequestID, exampleRequestID)
			}
			if got := rec.Header().Get(HeaderRequestID); got != exampleRequestID {
				t.Errorf("X-Request-ID = %q", got)
			}
			if strings.Contains(rec.Body.String(), "hunter2") {
				t.Errorf("the panic value leaked to the client: %s", rec.Body)
			}

			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("log is not JSON: %v\n%s", err, logs.String())
			}
			if entry["level"] != "ERROR" || entry["msg"] != "panic recovered" {
				t.Errorf("log entry = %v", entry)
			}
			if entry["request_id"] != exampleRequestID {
				t.Errorf("log request_id = %v, want %v", entry["request_id"], exampleRequestID)
			}
			if stack, _ := entry["stack"].(string); !strings.Contains(stack, "recover_test.go") {
				t.Errorf("log has no usable stack: %q", stack)
			}
			if tt.panicValue == secret && !strings.Contains(logs.String(), "hunter2") {
				t.Error("the panic value must be in the log, where only operators can see it")
			}
		})
	}
}

// TestRecoverKeepsTheStatusOfAStartedResponse: once the status line is out
// there is nothing to replace it with, so the log line is the only record.
func TestRecoverLeavesAStartedResponseAlone(t *testing.T) {
	var logs bytes.Buffer
	h := Recover(slog.New(slog.NewJSONHandler(&logs, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"partial":`))
			panic("boom")
		}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, "/api/v1/operations"))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the 200 the handler already sent", rec.Code)
	}
	if rec.Body.String() != `{"partial":` {
		t.Errorf("body = %q, want the partial response untouched", rec.Body)
	}
	if !strings.Contains(logs.String(), "panic recovered") {
		t.Error("the panic was not logged")
	}
}

// TestRecoverRepanicsErrAbortHandler: net/http's own recovery expects to see
// this one; it closes the connection without logging.
func TestRecoverRepanicsErrAbortHandler(t *testing.T) {
	var logs bytes.Buffer
	h := Recover(slog.New(slog.NewJSONHandler(&logs, nil)))(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic(http.ErrAbortHandler)
		}))

	defer func() {
		v := recover()
		if v != http.ErrAbortHandler { //nolint:errorlint // identity is the contract
			t.Errorf("recovered %v, want ErrAbortHandler to be re-panicked", v)
		}
		if logs.Len() != 0 {
			t.Errorf("an aborted handler must not log: %s", logs.String())
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), newRequest(t, http.MethodGet, "/"))
	t.Error("ServeHTTP returned; the panic should have propagated")
}

// TestRecoverKeepsTheServerServing is the acceptance criterion: a panic is
// one failed request, not a dead process. A real server, because that is the
// claim being made.
func TestRecoverKeepsTheServerServing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /boom", func(http.ResponseWriter, *http.Request) { panic("boom") })
	mux.Handle("GET /fine", ok())

	srv := httptest.NewServer(Chain(mux, Recover(nil), RequestID()))
	defer srv.Close()

	for _, path := range []string{"/boom", "/fine", "/boom", "/fine"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("GET %s after a panic: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		want := http.StatusOK
		if path == "/boom" {
			want = http.StatusInternalServerError
		}
		if resp.StatusCode != want {
			t.Errorf("GET %s = %d, want %d (%s)", path, resp.StatusCode, want, body)
		}
		if resp.Header.Get(HeaderRequestID) == "" {
			t.Errorf("GET %s carries no request ID", path)
		}
	}
}

func TestRecoverNilLoggerIsUsable(t *testing.T) {
	rec := httptest.NewRecorder()
	Recover(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})).ServeHTTP(rec, newRequest(t, http.MethodGet, "/"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

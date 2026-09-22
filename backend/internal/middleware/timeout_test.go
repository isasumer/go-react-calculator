package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTimeoutAnswersWithAProblem is the whole point of not using
// http.TimeoutHandler: the answer is problem+json with a stable code, and the
// late handler cannot write over it afterwards.
func TestTimeoutAnswersWithAProblem(t *testing.T) {
	release := make(chan struct{})
	lateWrite := make(chan error, 1)

	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		// Everything from here on is too late and must be dropped.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"result":3}`))
		_ = http.NewResponseController(w).Flush()
		lateWrite <- err
	}), RequestID(), Timeout(time.Millisecond))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "/api/v1/calculate"))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q", ct)
	}
	assertProblemGolden(t, "timeout", rec.Body.Bytes())

	// Let the handler finish and prove it was locked out.
	close(release)
	select {
	case err := <-lateWrite:
		if !errors.Is(err, http.ErrHandlerTimeout) {
			t.Errorf("late Write returned %v, want ErrHandlerTimeout", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the handler never finished")
	}
	if got := rec.Body.String(); strings.Contains(got, "result") {
		t.Errorf("the late handler wrote over the timeout response: %s", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("the late handler changed Content-Type to %q", got)
	}
	if rec.Flushed {
		t.Error("the late handler's flush reached the real writer")
	}
}

// TestTimeoutPassesThroughInTime: a handler that answers in time is not
// touched — status, body and the headers it staged all arrive.
func TestTimeoutPassesThroughInTime(t *testing.T) {
	h := Timeout(10 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Layer", "handler")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":3}`))
	}))

	rec := httptest.NewRecorder()
	rec.Header().Set(HeaderRequestID, exampleRequestID) // set outside, as RequestID does
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "/api/v1/calculate"))

	if rec.Code != http.StatusCreated || rec.Body.String() != `{"result":3}` {
		t.Errorf("got %d %q", rec.Code, rec.Body)
	}
	for name, want := range map[string]string{
		"Content-Type":  "application/json",
		"X-Layer":       "handler",
		HeaderRequestID: exampleRequestID,
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

// TestTimeoutCommitsStagedHeadersWithoutAWrite: a handler that only sets
// headers still gets them onto the response net/http sends for it.
func TestTimeoutCommitsStagedHeadersWithoutAWrite(t *testing.T) {
	h := Timeout(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Layer", "handler")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, "/api/v1/operations"))
	if got := rec.Header().Get("X-Layer"); got != "handler" {
		t.Errorf("X-Layer = %q, want the staged value", got)
	}
}

// TestTimeoutHandsBackTheRoutePattern: the router records the pattern on the
// copy of the request the timeout made, and the logger outside reads the
// original. Without the hand-back every line would log an empty route.
func TestTimeoutHandsBackTheRoutePattern(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/operations", ok())

	var outer string
	spy := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			outer = r.Pattern
		})
	}
	Chain(mux, spy, Timeout(time.Second)).
		ServeHTTP(httptest.NewRecorder(), newRequest(t, http.MethodGet, "/api/v1/operations"))

	if outer != "GET /api/v1/operations" {
		t.Errorf("route seen outside the timeout = %q", outer)
	}
}

// TestTimeoutDisabled: REQUEST_TIMEOUT=0 means no budget, not a budget of
// zero.
func TestTimeoutDisabled(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		rec := httptest.NewRecorder()
		Timeout(d)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(5 * time.Millisecond)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})).ServeHTTP(rec, newRequest(t, http.MethodGet, "/api/v1/operations"))

		if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
			t.Errorf("Timeout(%s): got %d %q, want the handler's own answer", d, rec.Code, rec.Body)
		}
	}
}

// TestTimeoutClientGone: a canceled request is not a timeout. Nobody is
// listening, so nothing is written — writing would only log a broken pipe.
func TestTimeoutClientGone(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	release := make(chan struct{})
	lateWrite := make(chan error, 1)

	h := Timeout(10 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		// Released only once the middleware has given up on this request,
		// so what follows is unambiguously "after".
		<-release
		_, err := w.Write([]byte("too late"))
		lateWrite <- err
	}))

	req := newRequest(t, http.MethodPost, "/api/v1/calculate").WithContext(ctx)
	rec := httptest.NewRecorder()
	go func() {
		<-started
		cancel()
	}()
	h.ServeHTTP(rec, req)
	close(release)

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want nothing written to a client that left", rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "" {
		t.Errorf("Content-Type = %q, want no response at all", ct)
	}
	select {
	case err := <-lateWrite:
		if !errors.Is(err, http.ErrHandlerTimeout) {
			t.Errorf("late Write returned %v, want ErrHandlerTimeout", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the handler never finished")
	}
}

// TestTimeoutPanicReachesRecover: the handler runs on its own goroutine, so
// its panic has to be carried back to the one Recover is on — otherwise it
// takes the process down.
func TestTimeoutPanicReachesRecover(t *testing.T) {
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), Recover(nil), RequestID(), Timeout(time.Second))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "/api/v1/calculate"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var problem struct {
		Code      string `json:"code"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != "INTERNAL" || problem.RequestID != exampleRequestID {
		t.Errorf("problem = %+v", problem)
	}
}

// TestTimeoutFlush: a handler that flushes has committed its status, so the
// timeout can no longer replace the response.
func TestTimeoutFlush(t *testing.T) {
	release := make(chan struct{})
	done := make(chan struct{})

	h := Timeout(20 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("partial"))
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush: %v", err)
		}
		<-release
		close(done)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, "/api/v1/operations"))

	if rec.Code != http.StatusAccepted || rec.Body.String() != "partial" {
		t.Errorf("got %d %q, want the response the handler had already started", rec.Code, rec.Body)
	}
	if !rec.Flushed {
		t.Error("the flush never reached the real writer")
	}
	close(release)
	<-done
}

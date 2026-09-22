package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/isasumer/go-react-calculator/backend/internal/config"
	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// newTestMetrics gives a test its own registry, so nothing it asserts can
// have been recorded by another one.
func newTestMetrics(t *testing.T) *observability.Metrics {
	t.Helper()
	return observability.NewMetrics(prometheus.NewRegistry(), observability.BuildInfo{
		Version: "v0.0.0-test", Commit: "testing",
	})
}

// exposition renders m the way GET /metrics does, so a test can assert on
// the text an operator would actually see.
func exposition(t *testing.T, m *observability.Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("rendering metrics = %d, want 200", rec.Code)
	}
	return rec.Body.String()
}

// envFunc turns a map into the getenv run() reads its configuration from.
// A map beats t.Setenv here: it leaves the process environment alone, so the
// lifecycle tests can run in parallel with everything else.
func envFunc(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// safeBuffer collects the server's log output. run() writes from its own
// goroutines, so the test reads it under a mutex.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitForLog polls the log until a JSON line with msg == want shows up and
// returns it decoded.
func waitForLog(t *testing.T, out *safeBuffer, want string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		for _, line := range strings.Split(out.String(), "\n") {
			var entry map[string]any
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue // a partially written line
			}
			if entry["msg"] == want {
				return entry
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no %q log line within 10s; log so far:\n%s", want, out.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// probeClient never reuses a connection, so every probe below is answered
// over a fresh one — exactly what a load balancer's health check does.
func probeClient() *http.Client {
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
}

// get performs one GET and returns the status and the decoded JSON body.
func get(t *testing.T, client *http.Client, url string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("GET %s: Cache-Control = %q, want no-store", url, got)
	}
	return resp.StatusCode, body
}

// TestRunLifecycle is the whole contract of B1-03 in one run: the server
// binds an ephemeral port and says so, the probes answer, and a SIGTERM
// (here: a canceled context) takes it out of rotation before it stops
// listening, lets the in-flight request finish, and returns nil in time.
func TestRunLifecycle(t *testing.T) {
	const (
		preStopDelay    = time.Second
		shutdownTimeout = 5 * time.Second
	)
	env := map[string]string{
		"HOST":             "127.0.0.1",
		"PORT":             "0", // no fixed port: the kernel picks a free one
		"LOG_FORMAT":       "json",
		"LOG_LEVEL":        "info",
		"PRE_STOP_DELAY":   preStopDelay.String(),
		"SHUTDOWN_TIMEOUT": shutdownTimeout.String(),
		// The in-flight request below is deliberately slower than a real
		// one; the timeout middleware would otherwise abandon it mid-drain.
		"REQUEST_TIMEOUT": "30s",
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := &safeBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"server"}, envFunc(env), out) }()

	// 1. The startup line reports the bound address and the resolved config.
	start := waitForLog(t, out, "listening")
	addr, _ := start["addr"].(string)
	if addr == "" || strings.HasSuffix(addr, ":0") {
		t.Fatalf("listening line has no bound address: %v", start)
	}
	resolved, _ := start["config"].(string)
	for _, want := range []string{"host=127.0.0.1", "port=0", "preStopDelay=1s", "shutdownTimeout=5s"} {
		if !strings.Contains(resolved, want) {
			t.Errorf("startup config %q is missing %q", resolved, want)
		}
	}
	base := "http://" + addr
	client := probeClient()

	// 2. The probes answer while the server serves.
	if status, body := get(t, client, base+"/healthz"); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("GET /healthz = %d %v, want 200 {status:ok}", status, body)
	}
	if status, body := get(t, client, base+"/readyz"); status != http.StatusOK || body["status"] != "ready" {
		t.Errorf("GET /readyz = %d %v, want 200 {status:ready}", status, body)
	}
	status, version := get(t, client, base+"/version")
	if status != http.StatusOK {
		t.Errorf("GET /version = %d, want 200", status)
	}
	for _, field := range []string{"version", "commit", "buildDate", "goVersion"} {
		if s, _ := version[field].(string); s == "" {
			t.Errorf("GET /version: %s is empty in %v", field, version)
		}
	}

	// 3. A request that is still in flight when the signal arrives. The body
	// is sent in two halves; "Expect: 100-continue" tells us exactly when the
	// handler has started reading it, with no sleeps to guess at.
	inflight := newSlowRequest(t, base+"/api/v1/calculate")
	inflight.waitUntilHandlerReads(t)

	// 4. Shut down.
	cancel()
	waitForLog(t, out, "shutting down")

	// 5. Readiness flips before the listener closes: during the pre-stop
	// delay a brand-new connection is still accepted and answered 503, which
	// is what makes a load balancer drain this instance.
	status, problem := get(t, client, base+"/readyz")
	if status != http.StatusServiceUnavailable || problem["code"] != "NOT_READY" {
		t.Errorf("GET /readyz while draining = %d %v, want 503 NOT_READY", status, problem)
	}
	// Liveness stays green: a draining process must not be restarted.
	if status, body := get(t, client, base+"/healthz"); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("GET /healthz while draining = %d %v, want 200 {status:ok}", status, body)
	}

	// 6. The in-flight request completes, with the right answer.
	inflight.finish(t)

	// 7. run returns nil, within the shutdown budget.
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run = %v, want nil", err)
		}
	case <-time.After(preStopDelay + shutdownTimeout):
		t.Fatalf("run did not return within %s; log:\n%s", preStopDelay+shutdownTimeout, out.String())
	}

	stopped := waitForLog(t, out, "stopped")
	drain, ok := stopped["drain"].(float64)
	if !ok || drain < float64(preStopDelay) {
		t.Errorf("stopped line = %v, want a drain duration of at least the pre-stop delay", stopped)
	}
}

// TestChain asserts the composition root wires the chain in the order
// docs/PLAN.md §1.2 documents, through the behavior only that order can
// produce. The middleware themselves are tested in internal/middleware; what
// is being checked here is the assembly.
func TestChain(t *testing.T) {
	cfg := config.Config{
		RequestTimeout:     50 * time.Millisecond,
		RateLimitRPS:       1,
		RateLimitBurst:     3,
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}
	router := http.NewServeMux()
	router.Handle("POST /api/v1/calculate", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":3}`))
	}))
	router.Handle("GET /slow", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // the timeout cancels it
	}))
	router.Handle("GET /boom", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	var out safeBuffer
	metrics := newTestMetrics(t)
	h := chain(router, cfg, newLogger(&out, config.Config{LogFormat: config.FormatJSON, LogLevel: slog.LevelDebug}), metrics)

	do := func(method, path string, headers map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
		for name, value := range headers {
			req.Header.Set(name, value)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// Every response carries the request ID and the security headers,
	// because RequestID and SecurityHeaders wrap everything below them.
	assertWrapped := func(t *testing.T, rec *httptest.ResponseRecorder, what string) {
		t.Helper()
		if rec.Header().Get("X-Request-ID") == "" {
			t.Errorf("%s: no X-Request-ID", what)
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q", what, got)
		}
	}

	t.Run("router is reached last", func(t *testing.T) {
		rec := do(http.MethodPost, "/api/v1/calculate", nil)
		if rec.Code != http.StatusOK || rec.Body.String() != `{"result":3}` {
			t.Errorf("got %d %q", rec.Code, rec.Body)
		}
		assertWrapped(t, rec, "200")
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q", got)
		}
	})

	t.Run("preflight is answered above the router", func(t *testing.T) {
		rec := do(http.MethodOptions, "/api/v1/calculate", map[string]string{
			"Origin":                        "http://localhost:5173",
			"Access-Control-Request-Method": http.MethodPost,
		})
		if rec.Code != http.StatusNoContent {
			t.Errorf("preflight = %d, want 204", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		assertWrapped(t, rec, "preflight")
	})

	t.Run("a panic becomes a problem", func(t *testing.T) {
		rec := do(http.MethodGet, "/boom", nil)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		var problem map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
			t.Fatal(err)
		}
		if problem["code"] != "INTERNAL" || problem["requestId"] != rec.Header().Get("X-Request-ID") {
			t.Errorf("problem = %v, X-Request-ID = %q", problem, rec.Header().Get("X-Request-ID"))
		}
	})

	t.Run("a slow handler times out", func(t *testing.T) {
		rec := do(http.MethodGet, "/slow", nil)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
		var problem map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
			t.Fatal(err)
		}
		if problem["code"] != "TIMEOUT" {
			t.Errorf("code = %v, want TIMEOUT", problem["code"])
		}
		if rec.Header().Get("X-Request-ID") == "" {
			t.Error("the timeout response carries no X-Request-ID")
		}
	})

	t.Run("over the limit is refused before the router", func(t *testing.T) {
		// The burst of 3 is spent by now; this client is done.
		rec := do(http.MethodPost, "/api/v1/calculate", nil)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", rec.Code)
		}
		if got := rec.Header().Get("Retry-After"); got != "1" {
			t.Errorf("Retry-After = %q, want 1", got)
		}
		assertWrapped(t, rec, "429")
	})

	// One access log line per request, with every documented field — except
	// the panicking one, which unwinds past the logger before there is a
	// status to report and is logged by Recover instead.
	var access, panics int
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		switch entry["msg"] {
		case "request":
			access++
			for _, field := range []string{
				"request_id", "method", "path", "route", "status",
				"bytes", "duration_ms", "remote_ip", "user_agent",
			} {
				if _, ok := entry[field]; !ok {
					t.Errorf("access log line is missing %s: %v", field, entry)
				}
			}
		case "panic recovered":
			panics++
			if entry["level"] != "ERROR" || entry["request_id"] == "" {
				t.Errorf("panic line = %v", entry)
			}
		}
	}
	if access != 4 || panics != 1 {
		t.Errorf("%d access lines and %d panic lines, want 4 and 1:\n%s", access, panics, out.String())
	}

	// The metrics slot is filled, and its position in the chain is visible
	// in the labels: the route is the pattern the router matched, and what
	// the layers above the slot answered was never counted.
	t.Run("metrics see the router's route and nothing above it", func(t *testing.T) {
		got := exposition(t, metrics)
		for _, want := range []string{
			`http_requests_total{method="POST",route="/api/v1/calculate",status="200"} 1`,
			`http_request_duration_seconds_count{method="POST",route="/api/v1/calculate"} 1`,
			"http_in_flight_requests 0",
			`build_info{commit="testing",version="v0.0.0-test"} 1`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("exposition is missing %q", want)
			}
		}
		// What the layers above the slot answered themselves is not here.
		// /slow deliberately is not asserted either way: the timeout
		// answers the client, but the handler keeps running and is observed
		// whenever it finally returns, which races this assertion. See the
		// middleware test and ADR-0006 for what that costs.
		for _, unwanted := range []string{
			`status="429"`,       // the rate limiter is outside the slot
			`status="204"`,       // so is the CORS preflight
			`route="/boom"`,      // the panic unwound past the slot
			"/api/v1/calculate?", // a raw target never becomes a label
		} {
			if strings.Contains(got, unwanted) {
				t.Errorf("exposition should not contain %q:\n%s", unwanted, got)
			}
		}
	})
}

// TestRunExposesMetrics is the endpoint as a scraper meets it: the real
// binary, on its real listener, after real traffic. It is the one place the
// whole wiring — registry, middleware, handler, route — is exercised at once.
func TestRunExposesMetrics(t *testing.T) {
	env := map[string]string{
		"HOST":       "127.0.0.1",
		"PORT":       "0",
		"LOG_FORMAT": "json",
		"LOG_LEVEL":  "info",
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := &safeBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"server"}, envFunc(env), out) }()

	addr, _ := waitForLog(t, out, "listening")["addr"].(string)
	base := "http://" + addr
	client := probeClient()

	// One calculation that works and one that does not, so both outcomes of
	// calc_operations_total have something in them.
	for _, body := range []string{
		`{"operation":"add","a":1,"b":2}`,
		`{"operation":"divide","a":1,"b":0}`,
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, base+"/api/v1/calculate", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", body, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base+"/metrics", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain…", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
	scraped := string(body)
	for _, want := range []string{
		`http_requests_total{method="POST",route="/api/v1/calculate",status="200"} 1`,
		`http_requests_total{method="POST",route="/api/v1/calculate",status="422"} 1`,
		`calc_operations_total{operation="add",outcome="ok"} 1`,
		`calc_operations_total{operation="divide",outcome="DIVISION_BY_ZERO"} 1`,
		"http_request_duration_seconds_bucket",
		"http_in_flight_requests 1", // this scrape is itself in flight
		"build_info{",
		"go_goroutines",              // the Go collector
		"process_start_time_seconds", // the process collector
	} {
		if !strings.Contains(scraped, want) {
			t.Errorf("scrape is missing %q", want)
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run = %v, want nil", err)
	}
}

// slowRequest is a calculate request whose body arrives in two halves, so it
// can be left in flight across a shutdown.
type slowRequest struct {
	body      *io.PipeWriter
	reading   chan struct{} // closed when the server asks for the body
	responses chan *http.Response
	errs      chan error
}

func newSlowRequest(t *testing.T, url string) *slowRequest {
	t.Helper()
	pr, pw := io.Pipe()
	s := &slowRequest{
		body:      pw,
		reading:   make(chan struct{}),
		responses: make(chan *http.Response, 1),
		errs:      make(chan error, 1),
	}

	// t.Context, not the server's context: canceling the server must not
	// cancel the request that is meant to survive the drain.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, pr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// The server answers 100 Continue when the handler first reads the body.
	req.Header.Set("Expect", "100-continue")
	var once sync.Once
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		Got1xxResponse: func(code int, _ textproto.MIMEHeader) error {
			if code == http.StatusContinue {
				once.Do(func() { close(s.reading) })
			}
			return nil
		},
	}))

	client := &http.Client{Transport: &http.Transport{ExpectContinueTimeout: 30 * time.Second}}
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			s.errs <- err
			return
		}
		s.responses <- resp
	}()
	go func() { _, _ = pw.Write([]byte(`{"operation":"add","a":1`)) }()
	return s
}

// waitUntilHandlerReads blocks until the server has read the request headers
// and the handler is waiting on the body: from here on the connection counts
// as in flight and Shutdown has to wait for it.
func (s *slowRequest) waitUntilHandlerReads(t *testing.T) {
	t.Helper()
	select {
	case <-s.reading:
	case err := <-s.errs:
		t.Fatalf("in-flight request failed before it started: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("server never started reading the in-flight request body")
	}
}

// finish sends the rest of the body and asserts the request was answered.
func (s *slowRequest) finish(t *testing.T) {
	t.Helper()
	if _, err := s.body.Write([]byte(`,"b":2}`)); err != nil {
		t.Fatalf("write rest of body: %v", err)
	}
	if err := s.body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}

	select {
	case resp := <-s.responses:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("in-flight request: status = %d, want 200", resp.StatusCode)
		}
		var got struct {
			Result float64 `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Result != 3 {
			t.Errorf("in-flight request: result = %v, want 3", got.Result)
		}
	case err := <-s.errs:
		t.Fatalf("in-flight request did not survive the drain: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("in-flight request was never answered")
	}
}

// TestRunInvalidConfigDoesNotListen: a bad environment fails fast, names the
// variable, and the process never binds a port.
func TestRunInvalidConfigDoesNotListen(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"port is not a number", map[string]string{"PORT": "eight thousand"}, "PORT"},
		{"port out of range", map[string]string{"PORT": "70000"}, "PORT"},
		{"unknown log level", map[string]string{"LOG_LEVEL": "verbose"}, "LOG_LEVEL"},
		{"bad duration", map[string]string{"SHUTDOWN_TIMEOUT": "15"}, "SHUTDOWN_TIMEOUT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out safeBuffer
			err := run(t.Context(), []string{"server"}, envFunc(tt.env), &out)
			if err == nil {
				t.Fatal("run returned nil, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not name %s", err, tt.wantErr)
			}
			if out.String() != "" {
				t.Errorf("run logged %q; it must fail before it listens", out.String())
			}
		})
	}
}

// TestRunListenFails: the port is already taken, so run reports it instead of
// logging and hanging.
func TestRunListenFails(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	_, port, err := net.SplitHostPort(taken.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	var out safeBuffer
	err = run(t.Context(), []string{"server"}, envFunc(map[string]string{
		"HOST": "127.0.0.1",
		"PORT": port,
	}), &out)
	if err == nil {
		t.Fatal("run returned nil, want a listen error")
	}
	if !strings.Contains(err.Error(), "listen on 127.0.0.1:"+port) {
		t.Errorf("error %q does not name the address", err)
	}
}

// TestRunVersionFlag: -version prints the build identity and exits 0 without
// starting a server.
func TestRunVersionFlag(t *testing.T) {
	var out safeBuffer
	if err := run(t.Context(), []string{"server", "-version"}, envFunc(nil), &out); err != nil {
		t.Fatalf("run = %v, want nil", err)
	}
	got := out.String()
	for _, want := range []string{"server version", "commit", "built", "go1."} {
		if !strings.Contains(got, want) {
			t.Errorf("-version printed %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, `"msg"`) {
		t.Errorf("-version started the server: %q", got)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	var out safeBuffer
	if err := run(t.Context(), []string{"server", "-nope"}, envFunc(nil), &out); err == nil {
		t.Fatal("run returned nil, want an error")
	}
}

// TestNewLogger: LOG_FORMAT and LOG_LEVEL decide the handler and what it
// keeps.
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.Config
		want     string
		unwanted string
	}{
		{
			name: "json handler",
			cfg:  config.Config{LogFormat: config.FormatJSON, LogLevel: slog.LevelInfo},
			want: `"msg":"hello"`,
		},
		{
			name: "text handler",
			cfg:  config.Config{LogFormat: config.FormatText, LogLevel: slog.LevelInfo},
			want: `msg=hello`,
		},
		{
			name:     "level filters",
			cfg:      config.Config{LogFormat: config.FormatJSON, LogLevel: slog.LevelWarn},
			want:     `"msg":"loud"`,
			unwanted: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			logger := newLogger(&out, tt.cfg)
			logger.Info("hello")
			logger.Warn("loud")
			if got := out.String(); !strings.Contains(got, tt.want) {
				t.Errorf("log = %q, missing %q", got, tt.want)
			}
			if tt.unwanted != "" && strings.Contains(out.String(), tt.unwanted) {
				t.Errorf("log = %q, should not contain %q", out.String(), tt.unwanted)
			}
		})
	}
}

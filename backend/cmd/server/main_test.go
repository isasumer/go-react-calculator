package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/isasumer/go-react-calculator/backend/internal/config"
)

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

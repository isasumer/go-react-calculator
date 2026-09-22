package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/isasumer/go-react-calculator/backend/internal/httpapi"
	"github.com/isasumer/go-react-calculator/backend/internal/middleware"
)

// The limits the integration server runs with. They are small enough that a
// handful of requests exhausts the bucket and slow enough that the refill is
// a fraction of a second, which is the one thing in this file that has to be
// waited for rather than observed.
const (
	integrationRPS   = 10
	integrationBurst = 10
	// The in-flight request below outlives the shutdown signal, so the
	// per-request budget has to outlive it too; it is still well under the
	// 5 s default.
	integrationRequestTimeout = 2 * time.Second
	// Readiness flips before the listener closes, and the pre-stop delay is
	// the window in which a load balancer — here, the test — can still open a
	// connection and be told 503.
	integrationPreStopDelay = time.Second
)

// TestIntegration drives the whole assembled process over TCP: run() with a
// real listener, the real middleware chain, and an ordinary HTTP client on
// the other end. The unit tests cover each layer in isolation; what only this
// can show is that the layers are wired together in the order the service
// promises — a request ID on every answer, a rate limiter in front of the
// router, metrics behind it, and a drain that lets an in-flight request
// finish.
//
// The subtests share one server and run in order: each leaves the token
// bucket, the metric counters and the process lifecycle where the next one
// expects them.
func TestIntegration(t *testing.T) {
	env := map[string]string{
		"HOST":             "127.0.0.1",
		"PORT":             "0", // no fixed port: the kernel picks a free one
		"LOG_FORMAT":       "json",
		"LOG_LEVEL":        "info",
		"RATE_LIMIT_RPS":   strconv.Itoa(integrationRPS),
		"RATE_LIMIT_BURST": strconv.Itoa(integrationBurst),
		"REQUEST_TIMEOUT":  integrationRequestTimeout.String(),
		"PRE_STOP_DELAY":   integrationPreStopDelay.String(),
		"SHUTDOWN_TIMEOUT": "5s",
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := &safeBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"server"}, envFunc(env), out) }()

	addr, _ := waitForLog(t, out, "listening")["addr"].(string)
	if addr == "" {
		t.Fatal("the server did not report a bound address")
	}
	base := "http://" + addr
	// Keep-alives on: what is under test is the chain, not connection setup.
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("happy path", func(t *testing.T) {
		resp, body := do(t, client, http.MethodPost, base+"/api/v1/calculate",
			`{"operation":"add","a":12,"b":7}`, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
		}
		if got := resp.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := string(body); got != `{"operation":"add","a":12,"b":7,"result":19}` {
			t.Errorf("body = %s", got)
		}
		if resp.Header.Get("X-Request-ID") == "" {
			t.Error("no X-Request-ID on a successful response")
		}
	})

	t.Run("division by zero is a problem document", func(t *testing.T) {
		resp, body := do(t, client, http.MethodPost, base+"/api/v1/calculate",
			`{"operation":"divide","a":1,"b":0}`, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422; body: %s", resp.StatusCode, body)
		}
		if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("Content-Type = %q, want application/problem+json", got)
		}
		p := decodeProblem(t, body)
		if p.Code != httpapi.CodeDivisionByZero || p.Status != http.StatusUnprocessableEntity {
			t.Errorf("problem = %+v", p)
		}
		// The ID the client holds is the one in the log: that is the whole
		// point of echoing it in the body as well as the header.
		if p.RequestID == "" || p.RequestID != resp.Header.Get("X-Request-ID") {
			t.Errorf("requestId %q does not match X-Request-ID %q", p.RequestID, resp.Header.Get("X-Request-ID"))
		}
	})

	t.Run("request id is echoed when usable and replaced when not", func(t *testing.T) {
		const mine = "integration-0123456789"
		resp, _ := do(t, client, http.MethodPost, base+"/api/v1/calculate",
			`{"operation":"add","a":1,"b":2}`, map[string]string{middleware.HeaderRequestID: mine})
		if got := resp.Header.Get(middleware.HeaderRequestID); got != mine {
			t.Errorf("X-Request-ID = %q, want the one sent (%q)", got, mine)
		}

		// A value that could forge a log line or an HTTP header is not
		// reused; the server substitutes one of its own.
		const forged = "bad id\twith spaces"
		resp, _ = do(t, client, http.MethodPost, base+"/api/v1/calculate",
			`{"operation":"add","a":1,"b":2}`, map[string]string{middleware.HeaderRequestID: forged})
		switch got := resp.Header.Get(middleware.HeaderRequestID); got {
		case "":
			t.Error("no X-Request-ID after an unusable one was sent")
		case forged:
			t.Errorf("X-Request-ID = %q, want a replacement", got)
		}
	})

	t.Run("the contract is served", func(t *testing.T) {
		resp, body := do(t, client, http.MethodGet, base+"/api/v1/openapi.yaml", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Type"); got != "application/yaml" {
			t.Errorf("Content-Type = %q, want application/yaml", got)
		}
		if !strings.Contains(string(body), "openapi: 3.1") ||
			!strings.Contains(string(body), "/api/v1/calculate") {
			t.Errorf("body is not the OpenAPI document:\n%.200s", body)
		}
	})

	t.Run("a burst is rate limited", func(t *testing.T) {
		// The bucket started with integrationBurst tokens and the subtests
		// above spent some of them, so this needs no more than a burst's
		// worth of requests to find the wall.
		var resp *http.Response
		var body []byte
		for range integrationBurst + 1 {
			resp, body = do(t, client, http.MethodPost, base+"/api/v1/calculate",
				`{"operation":"add","a":1,"b":2}`, nil)
			if resp.StatusCode == http.StatusTooManyRequests {
				break
			}
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("no 429 within %d requests past the burst", integrationBurst+1)
		}
		if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("Content-Type = %q, want application/problem+json", got)
		}
		retryAfter, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil || retryAfter < 1 {
			t.Errorf("Retry-After = %q, want whole seconds of at least 1", resp.Header.Get("Retry-After"))
		}
		if p := decodeProblem(t, body); p.Code != httpapi.CodeRateLimited {
			t.Errorf("code = %s, want RATE_LIMITED", p.Code)
		}
	})

	t.Run("metrics count the traffic", func(t *testing.T) {
		// /metrics is never rate limited, which is why this subtest can run
		// with the bucket empty.
		resp, body := do(t, client, http.MethodGet, base+"/metrics", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		for _, want := range []string{
			"http_requests_total",
			`route="/api/v1/calculate"`,
			`status="200"`,
			"calc_operations_total",
		} {
			if !strings.Contains(string(body), want) {
				t.Errorf("exposition does not mention %s", want)
			}
		}
	})

	// The only sleep in the file: a token bucket refills with time and with
	// nothing else, so the request that has to survive the drain has to wait
	// for one. Everything else here waits for an event.
	time.Sleep(3 * time.Second / integrationRPS)

	t.Run("the drain lets an in-flight request finish", func(t *testing.T) {
		// A request whose body arrives in two halves, so the shutdown signal
		// lands while the handler is provably still reading it.
		inflight := newSlowRequest(t, base+"/api/v1/calculate")
		inflight.waitUntilHandlerReads(t)

		cancel()
		waitForLog(t, out, "shutting down")

		// Readiness flips before the listener closes: a connection opened
		// during the pre-stop delay is still accepted and answered 503, which
		// is what takes the instance out of rotation.
		status, problem := get(t, probeClient(), base+"/readyz")
		if status != http.StatusServiceUnavailable || problem["code"] != string(httpapi.CodeNotReady) {
			t.Errorf("GET /readyz while draining = %d %v, want 503 NOT_READY", status, problem)
		}

		inflight.finish(t)

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("run = %v, want nil", err)
			}
		case <-time.After(integrationPreStopDelay + 5*time.Second):
			t.Fatalf("run did not return; log:\n%s", out.String())
		}
	})
}

// do performs one request and returns the response with its body already
// read, so a caller can assert on both without worrying about the reader.
func do(t *testing.T, client *http.Client, method, url, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader = http.NoBody
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, url, err)
	}
	return resp, out
}

// decodeProblem reads an application/problem+json body.
func decodeProblem(t *testing.T, body []byte) httpapi.Problem {
	t.Helper()
	var p httpapi.Problem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decode problem: %v\n%s", err, body)
	}
	return p
}

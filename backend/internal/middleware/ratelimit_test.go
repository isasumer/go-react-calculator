package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// clock is the injected time source: a rate limiter tested against the real
// clock either sleeps through its own refills or is flaky, and usually both.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *clock {
	return &clock{now: time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)}
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// call sends one request through h and reports the status and Retry-After.
func call(t *testing.T, h http.Handler, from string, headers map[string]string) (int, string) {
	t.Helper()
	req := newRequest(t, http.MethodPost, "/api/v1/calculate")
	if from != "" {
		req.RemoteAddr = from
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Header().Get("Retry-After")
}

// TestRateLimitBurstThenRefill is the acceptance criterion: a client may
// spend its burst, is refused while it is empty, and is served again once the
// bucket has refilled.
func TestRateLimitBurstThenRefill(t *testing.T) {
	c := newClock()
	h := RateLimit(RateLimitConfig{RPS: 2, Burst: 3, Now: c.Now})(ok())

	for i := range 3 {
		if status, _ := call(t, h, "", nil); status != http.StatusOK {
			t.Fatalf("request %d of the burst = %d, want 200", i+1, status)
		}
	}

	status, retryAfter := call(t, h, "", nil)
	if status != http.StatusTooManyRequests {
		t.Fatalf("over the burst = %d, want 429", status)
	}
	// 2 rps means the next token is half a second away; Retry-After is whole
	// seconds and never 0.
	if retryAfter != "1" {
		t.Errorf("Retry-After = %q, want 1", retryAfter)
	}

	// Not yet.
	c.advance(200 * time.Millisecond)
	if status, _ := call(t, h, "", nil); status != http.StatusTooManyRequests {
		t.Errorf("still empty = %d, want 429", status)
	}
	// One token's worth of time later, the client is served again.
	c.advance(500 * time.Millisecond)
	if status, _ := call(t, h, "", nil); status != http.StatusOK {
		t.Errorf("after the refill = %d, want 200", status)
	}
}

// TestRateLimitProblemGolden: the 429 body is the documented one.
func TestRateLimitProblemGolden(t *testing.T) {
	h := Chain(ok(), RequestID(), RateLimit(RateLimitConfig{RPS: 1, Burst: 1, Now: newClock().Now}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "/api/v1/calculate"))
	if rec.Code != http.StatusOK {
		t.Fatalf("first request = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "/api/v1/calculate"))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q", ct)
	}
	assertProblemGolden(t, "rate_limited", rec.Body.Bytes())
}

// TestRateLimitIsPerClient: one noisy client must not spend another's budget.
func TestRateLimitIsPerClient(t *testing.T) {
	c := newClock()
	h := RateLimit(RateLimitConfig{RPS: 1, Burst: 1, Now: c.Now})(ok())

	if status, _ := call(t, h, "203.0.113.7:1111", nil); status != http.StatusOK {
		t.Fatalf("first client = %d, want 200", status)
	}
	if status, _ := call(t, h, "203.0.113.7:2222", nil); status != http.StatusTooManyRequests {
		// Same host, different port: still the same client.
		t.Errorf("same host on a new connection = %d, want 429", status)
	}
	if status, _ := call(t, h, "198.51.100.4:1111", nil); status != http.StatusOK {
		t.Errorf("a different client = %d, want 200", status)
	}
}

// TestRateLimitSkipsOperationalPaths: throttling a probe would take a healthy
// instance out of rotation exactly when it is busiest.
func TestRateLimitSkipsOperationalPaths(t *testing.T) {
	h := RateLimit(RateLimitConfig{RPS: 1, Burst: 1, Now: newClock().Now})(ok())

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		for range 5 {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newRequest(t, http.MethodGet, path))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200 every time", path, rec.Code)
			}
		}
	}
	// The API is still limited after all that.
	if status, _ := call(t, h, "", nil); status != http.StatusOK {
		t.Fatalf("first API request = %d, want 200", status)
	}
	if status, _ := call(t, h, "", nil); status != http.StatusTooManyRequests {
		t.Errorf("second API request = %d, want 429", status)
	}
}

// TestRateLimitTrustProxyHeaders: X-Forwarded-For changes the answer only
// where it has been declared trustworthy.
func TestRateLimitTrustProxyHeaders(t *testing.T) {
	const proxy = "10.0.0.1:443"

	t.Run("untrusted header is ignored", func(t *testing.T) {
		h := RateLimit(RateLimitConfig{RPS: 1, Burst: 1, Now: newClock().Now})(ok())
		if status, _ := call(t, h, proxy, map[string]string{"X-Forwarded-For": "203.0.113.1"}); status != http.StatusOK {
			t.Fatalf("first = %d, want 200", status)
		}
		// A new forged address must not buy a new bucket.
		if status, _ := call(t, h, proxy, map[string]string{"X-Forwarded-For": "203.0.113.2"}); status != http.StatusTooManyRequests {
			t.Errorf("second = %d, want 429: the header is not trusted", status)
		}
	})

	t.Run("trusted header separates clients", func(t *testing.T) {
		h := RateLimit(RateLimitConfig{RPS: 1, Burst: 1, TrustProxyHeaders: true, Now: newClock().Now})(ok())
		if status, _ := call(t, h, proxy, map[string]string{"X-Forwarded-For": "203.0.113.1"}); status != http.StatusOK {
			t.Fatalf("first client = %d, want 200", status)
		}
		if status, _ := call(t, h, proxy, map[string]string{"X-Forwarded-For": "203.0.113.2"}); status != http.StatusOK {
			t.Errorf("second client = %d, want 200", status)
		}
		if status, _ := call(t, h, proxy, map[string]string{"X-Forwarded-For": "203.0.113.1"}); status != http.StatusTooManyRequests {
			t.Errorf("first client again = %d, want 429", status)
		}
	})
}

func TestClientKey(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		trust      bool
		want       string
	}{
		{name: "peer address", remoteAddr: "203.0.113.7:54321", want: "203.0.113.7"},
		{name: "header ignored by default", remoteAddr: "10.0.0.1:443", xff: "203.0.113.7", want: "10.0.0.1"},
		{name: "trusted header", remoteAddr: "10.0.0.1:443", xff: "203.0.113.7", trust: true, want: "203.0.113.7"},
		{
			name: "trusted chain uses the original client", remoteAddr: "10.0.0.1:443",
			xff: " 203.0.113.7 , 198.51.100.1 ", trust: true, want: "203.0.113.7",
		},
		{name: "trusted but absent", remoteAddr: "10.0.0.1:443", trust: true, want: "10.0.0.1"},
		{name: "trusted but blank", remoteAddr: "10.0.0.1:443", xff: " , 198.51.100.1", trust: true, want: "10.0.0.1"},
		{name: "ipv6 peer", remoteAddr: "[2001:db8::1]:443", want: "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := clientKey(r, tt.trust); got != tt.want {
				t.Errorf("clientKey = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRateLimitEvictsIdleBuckets: one bucket per source address is unbounded
// memory if nothing removes them, so a sweep runs every sweepEvery inserts.
func TestRateLimitEvictsIdleBuckets(t *testing.T) {
	c := newClock()
	store := newBucketStore(RateLimitConfig{RPS: 1, Burst: 1, IdleTTL: time.Minute})

	for i := range sweepEvery {
		store.allow(fmt.Sprintf("client-a-%d", i), c.Now())
	}
	if got := store.size(); got != sweepEvery {
		t.Fatalf("after the first batch: %d buckets, want %d — nothing was idle yet", got, sweepEvery)
	}

	// Long enough that the first batch has gone idle.
	c.advance(2 * time.Minute)
	for i := range sweepEvery {
		store.allow(fmt.Sprintf("client-b-%d", i), c.Now())
	}
	if got := store.size(); got != sweepEvery {
		t.Errorf("after the second batch: %d buckets, want %d — the idle ones should be gone", got, sweepEvery)
	}

	// A client that keeps coming back keeps its bucket, and its state: the
	// burst of 1 it spent above is still spent.
	if _, ok := store.allow("client-b-0", c.Now()); ok {
		t.Error("a bucket that survived the sweep was reset")
	}
}

// TestRateLimitIdleTTLDefaults covers the zero value of the knob.
func TestRateLimitIdleTTLDefaults(t *testing.T) {
	if got := newBucketStore(RateLimitConfig{RPS: 1, Burst: 1}).ttl; got != DefaultIdleTTL {
		t.Errorf("ttl = %s, want %s", got, DefaultIdleTTL)
	}
	if got := newBucketStore(RateLimitConfig{RPS: 1, Burst: 1, IdleTTL: time.Second}).ttl; got != time.Second {
		t.Errorf("ttl = %s, want 1s", got)
	}
}

// TestRateLimitDisabled: a nonsensical limit turns the middleware off rather
// than refusing everything.
func TestRateLimitDisabled(t *testing.T) {
	for _, cfg := range []RateLimitConfig{{RPS: 0, Burst: 10}, {RPS: 10, Burst: 0}, {}} {
		h := RateLimit(cfg)(ok())
		for range 5 {
			if status, _ := call(t, h, "", nil); status != http.StatusOK {
				t.Fatalf("RateLimit(%+v) refused a request", cfg)
			}
		}
	}
}

// TestRateLimitConcurrentClients runs the store the way the server does, so
// -race has something to say about the map and the buckets in it.
func TestRateLimitConcurrentClients(t *testing.T) {
	h := RateLimit(RateLimitConfig{RPS: 100, Burst: 100})(ok())

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/calculate", http.NoBody)
				req.RemoteAddr = fmt.Sprintf("203.0.113.%d:1234", i)
				h.ServeHTTP(httptest.NewRecorder(), req)
			}
		}()
	}
	wg.Wait()
}

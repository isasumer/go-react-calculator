package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// metricsFixture is a registry, the Metrics on it and the chain-of-one under
// test, wrapped around a router with two routes and no catch-all — so a path
// nobody registered really is unmatched, which is what production's catch-all
// "/" route hides.
type metricsFixture struct {
	reg     *prometheus.Registry
	metrics *observability.Metrics
	handler http.Handler
}

func newMetricsFixture(t *testing.T, router http.Handler) *metricsFixture {
	t.Helper()
	reg := prometheus.NewRegistry()
	m := observability.NewMetrics(reg, observability.BuildInfo{Version: "v1.2.3", Commit: "abc1234"})
	return &metricsFixture{reg: reg, metrics: m, handler: Metrics(m)(router)}
}

func (f *metricsFixture) do(t *testing.T, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, http.NoBody))
	return rec
}

// counter returns the value of one http_requests_total series, or -1 when
// that label set was never observed.
func (f *metricsFixture) counter(t *testing.T, method, route, status string) float64 {
	t.Helper()
	return f.series(t, "http_requests_total", map[string]string{
		"method": method, "route": route, "status": status,
	})
}

// series finds the sample of name carrying exactly labels. Counters and
// gauges report their value; a histogram reports its sample count, which is
// what "observed once" means.
func (f *metricsFixture) series(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := f.reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			got := make(map[string]string, len(metric.GetLabel()))
			for _, pair := range metric.GetLabel() {
				got[pair.GetName()] = pair.GetValue()
			}
			if len(got) != len(labels) {
				continue
			}
			match := true
			for name, want := range labels {
				if got[name] != want {
					match = false
					break
				}
			}
			if !match {
				continue
			}
			switch {
			case metric.Counter != nil:
				return metric.GetCounter().GetValue()
			case metric.Gauge != nil:
				return metric.GetGauge().GetValue()
			default:
				return float64(metric.GetHistogram().GetSampleCount())
			}
		}
	}
	return -1
}

// router is the fixture's router: the two patterns below and nothing else.
func router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/calculate", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"code":"DIVISION_BY_ZERO"}`))
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}

// TestMetricsLabelsTheRouteWithTheMatchedPattern is the cardinality contract:
// the label is the router's pattern, so a query string, and anything else a
// client can put in a URL, collapses onto one series.
func TestMetricsLabelsTheRouteWithTheMatchedPattern(t *testing.T) {
	f := newMetricsFixture(t, router())

	if rec := f.do(t, http.MethodPost, "/api/v1/calculate?x=1"); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}

	// The pattern, not the request target.
	if got := f.counter(t, http.MethodPost, "/api/v1/calculate", "422"); got != 1 {
		t.Errorf(`http_requests_total{route="/api/v1/calculate",status="422"} = %v, want 1`, got)
	}
	if got := f.counter(t, http.MethodPost, "/api/v1/calculate?x=1", "422"); got != -1 {
		t.Errorf("the raw target became a series: %v", got)
	}

	// The latency histogram carries the same route and no status, and this
	// request was observed exactly once.
	observed := f.series(t, "http_request_duration_seconds",
		map[string]string{"method": http.MethodPost, "route": "/api/v1/calculate"})
	if observed != 1 {
		t.Errorf("http_request_duration_seconds sample count = %v, want 1", observed)
	}

	// Two more requests to the same route: one new series for the new
	// status, and the histogram keeps counting on the one it has.
	f.do(t, http.MethodPost, "/api/v1/calculate?x=2")
	f.do(t, http.MethodPost, "/api/v1/calculate")
	if got := f.counter(t, http.MethodPost, "/api/v1/calculate", "422"); got != 3 {
		t.Errorf("counter = %v after three requests, want 3", got)
	}
	if got := testutil.CollectAndCount(f.reg, "http_request_duration_seconds"); got != 1 {
		t.Errorf("http_request_duration_seconds has %d series, want 1", got)
	}
}

// TestMetricsLabelsAnUnmatchedRequest: nothing matched, so there is no
// pattern to report and the label is a constant rather than the path.
func TestMetricsLabelsAnUnmatchedRequest(t *testing.T) {
	f := newMetricsFixture(t, router())

	for _, target := range []string{"/nope", "/also/nope", "/nope?and=again"} {
		if rec := f.do(t, http.MethodGet, target); rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", target, rec.Code)
		}
	}

	if got := f.counter(t, http.MethodGet, routeUnmatched, "404"); got != 3 {
		t.Errorf(`http_requests_total{route=%q,status="404"} = %v, want 3`, routeUnmatched, got)
	}
	if got := testutil.CollectAndCount(f.reg, "http_requests_total"); got != 1 {
		t.Errorf("three unmatched paths became %d series, want 1", got)
	}
}

// TestMetricsCountsTheOperationalEndpoints: the middleware skips nothing.
// A scrape and a probe are three label sets and an atomic add, and seeing
// them is how an operator knows the scraper and the load balancer are alive.
func TestMetricsCountsTheOperationalEndpoints(t *testing.T) {
	f := newMetricsFixture(t, router())

	f.do(t, http.MethodGet, "/healthz")
	f.do(t, http.MethodGet, "/healthz")

	if got := f.counter(t, http.MethodGet, "/healthz", "200"); got != 2 {
		t.Errorf(`http_requests_total{route="/healthz"} = %v, want 2`, got)
	}
}

// TestMetricsInFlightGauge: the gauge is 1 while the handler runs and back to
// 0 the moment it returns.
func TestMetricsInFlightGauge(t *testing.T) {
	var during float64
	f := newMetricsFixture(t, nil)
	f.handler = Metrics(f.metrics)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		during = f.series(t, "http_in_flight_requests", map[string]string{})
		w.WriteHeader(http.StatusOK)
	}))

	f.do(t, http.MethodGet, "/healthz")

	if during != 1 {
		t.Errorf("http_in_flight_requests = %v inside the handler, want 1", during)
	}
	if after := f.series(t, "http_in_flight_requests", map[string]string{}); after != 0 {
		t.Errorf("http_in_flight_requests = %v after the request, want 0", after)
	}
}

// TestMetricsInFlightGaugeSurvivesAPanic: the decrement is deferred, so a
// panic unwinding to Recover does not leak a permanent +1 — a gauge that only
// comes back down on the happy path reads as a wedged server after any 500.
func TestMetricsInFlightGaugeSurvivesAPanic(t *testing.T) {
	f := newMetricsFixture(t, nil)
	f.handler = Metrics(f.metrics)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not reach the caller; Recover would never see it")
			}
		}()
		f.do(t, http.MethodGet, "/boom")
	}()

	if got := f.series(t, "http_in_flight_requests", map[string]string{}); got != 0 {
		t.Errorf("http_in_flight_requests = %v after a panic, want 0", got)
	}
	// There is no status to report for a request that never produced one.
	if got := testutil.CollectAndCount(f.reg, "http_requests_total"); got != 0 {
		t.Errorf("a panicking request produced %d counter series, want 0", got)
	}
}

// comparableHandler is a handler that can be compared with ==, so the test
// below can assert the *same* handler came back rather than an equivalent
// one. A func value cannot be compared at all.
type comparableHandler struct{ body string }

func (h comparableHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(h.body))
}

// TestMetricsRecordsTheHandlerNotTheTimeoutResponse pins down the one place
// where http_requests_total and the client disagree.
//
// Metrics is innermost, so [Timeout] is above it: when the deadline fires,
// the client is answered 503 by a layer this middleware never sees, and the
// handler goroutine carries on. Whatever status that handler eventually
// produces is what gets counted — 200 here, from the recorder's default,
// because the writer it holds is already closed.
//
// The alternative is moving metrics above the timeout, which costs the route
// label, because the router has not run yet and r.Pattern is still empty.
// Latency for these requests is truthful either way — the histogram sees how
// long the handler really took — and the 503 the client got is in the access
// log and in the TIMEOUT problem count. Recorded here so the next person to
// read a dashboard knows it.
func TestMetricsRecordsTheHandlerNotTheTimeoutResponse(t *testing.T) {
	released := make(chan struct{})
	finished := make(chan struct{})

	router := http.NewServeMux()
	router.HandleFunc("GET /slow", func(http.ResponseWriter, *http.Request) {
		<-released
		close(finished)
	})

	f := newMetricsFixture(t, router)
	f.handler = Timeout(10 * time.Millisecond)(f.handler)

	rec := f.do(t, http.MethodGet, "/slow")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("the client got %d, want the timeout's 503", rec.Code)
	}

	// Nothing is recorded while the handler is still running.
	if got := f.counter(t, http.MethodGet, "/slow", "200"); got != -1 {
		t.Errorf("the request was counted before the handler returned: %v", got)
	}
	if got := f.series(t, "http_in_flight_requests", map[string]string{}); got != 1 {
		t.Errorf("http_in_flight_requests = %v with the handler still running, want 1", got)
	}

	close(released)
	<-finished
	// The observation happens after the handler returns, so give the
	// middleware's remaining statements a moment to run.
	deadline := time.Now().Add(5 * time.Second)
	for f.counter(t, http.MethodGet, "/slow", "200") != 1 {
		if time.Now().After(deadline) {
			t.Fatalf(`http_requests_total{route="/slow",status="200"} never appeared`)
		}
		time.Sleep(time.Millisecond)
	}
	if got := f.counter(t, http.MethodGet, "/slow", "503"); got != -1 {
		t.Errorf("the 503 the client received was counted: %v", got)
	}
}

// TestMetricsWithoutARegistryIsTransparent: a nil Metrics leaves the chain
// assemblable and the handler untouched.
func TestMetricsWithoutARegistryIsTransparent(t *testing.T) {
	var inner http.Handler = comparableHandler{body: `{"status":"ok"}`}
	if got := Metrics(nil)(inner); got != inner {
		t.Error("Metrics(nil) wrapped the handler")
	}

	rec := httptest.NewRecorder()
	Metrics(nil)(inner).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("got %d %q", rec.Code, rec.Body)
	}
}

// TestRouteOf covers the pattern shapes http.ServeMux can report, including
// the ones this API does not register but the helper must still not mangle.
func TestRouteOf(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"method and path", "POST /api/v1/calculate", "/api/v1/calculate"},
		{"path only", "/healthz", "/healthz"},
		{"catch-all", "/", "/"},
		{"host and path", "example.com/api/v1/calculate", "/api/v1/calculate"},
		{"method, host and path", "GET example.com/healthz", "/healthz"},
		{"wildcard is kept", "GET /files/{name}", "/files/{name}"},
		{"nothing matched", "", routeUnmatched},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/whatever", http.NoBody)
			r.Pattern = tt.pattern
			if got := routeOf(r); got != tt.want {
				t.Errorf("routeOf(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

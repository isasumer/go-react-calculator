package observability

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// testBuild is the identity every test in this file exports, so the
// build_info assertions have something specific to match.
var testBuild = BuildInfo{Version: "v1.2.3", Commit: "abc1234", BuildDate: "2026-09-22T00:00:00Z"}

// newTestMetrics gives each test a registry of its own: registrations are
// then independent, and Gather sees nothing but what the test produced.
func newTestMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	return NewMetrics(reg, testBuild), reg
}

// gatheredNames is the sorted list of metric family names on reg.
func gatheredNames(t *testing.T, reg *prometheus.Registry) []string {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	names := make([]string, 0, len(families))
	for _, f := range families {
		names = append(names, f.GetName())
	}
	slices.Sort(names)
	return names
}

// TestNewMetricsRegistersEveryFamily: the five families the service promises
// are on the registry, and nothing else is.
func TestNewMetricsRegistersEveryFamily(t *testing.T) {
	m, reg := newTestMetrics(t)

	// A *Vec exports nothing until a label set exists, so touch each one.
	m.ObserveRequest(http.MethodPost, "/api/v1/calculate", http.StatusOK, 3*time.Millisecond)
	m.ObserveCalculation("add", OutcomeOK)
	m.IncInFlight()
	defer m.DecInFlight()

	want := []string{
		"build_info",
		"calc_operations_total",
		"http_in_flight_requests",
		"http_request_duration_seconds",
		"http_requests_total",
	}
	if got := gatheredNames(t, reg); !slices.Equal(got, want) {
		t.Errorf("families =\n%v\nwant\n%v", got, want)
	}
}

// TestBuildInfoIsOneWithTheBuildLabels: the info-metric convention — a
// constant 1 whose labels carry the facts a dashboard joins on.
func TestBuildInfoIsOneWithTheBuildLabels(t *testing.T) {
	m, _ := newTestMetrics(t)

	if got := testutil.ToFloat64(m.buildInfo.WithLabelValues(testBuild.Version, testBuild.Commit)); got != 1 {
		t.Errorf("build_info = %v, want 1", got)
	}
	if got := testutil.CollectAndCount(m.buildInfo); got != 1 {
		t.Errorf("build_info has %d series, want exactly 1", got)
	}

	const want = `# HELP build_info Always 1; the build identity of the running binary is in the labels.
# TYPE build_info gauge
build_info{commit="abc1234",version="v1.2.3"} 1
`
	if err := testutil.CollectAndCompare(m.buildInfo, strings.NewReader(want), "build_info"); err != nil {
		t.Error(err)
	}
}

// TestObserveCalculationCountsByOperationAndOutcome: the counter the API
// handler drives, including that "ok" and a problem code are separate series
// of the same operation.
func TestObserveCalculationCountsByOperationAndOutcome(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.ObserveCalculation("divide", OutcomeOK)
	m.ObserveCalculation("divide", OutcomeOK)
	m.ObserveCalculation("divide", "DIVISION_BY_ZERO")
	m.ObserveCalculation("sqrt", "DOMAIN_ERROR")

	tests := []struct {
		operation, outcome string
		want               float64
	}{
		{"divide", OutcomeOK, 2},
		{"divide", "DIVISION_BY_ZERO", 1},
		{"sqrt", "DOMAIN_ERROR", 1},
	}
	for _, tt := range tests {
		got := testutil.ToFloat64(m.calcOps.WithLabelValues(tt.operation, tt.outcome))
		if got != tt.want {
			t.Errorf("calc_operations_total{operation=%q,outcome=%q} = %v, want %v",
				tt.operation, tt.outcome, got, tt.want)
		}
	}
	if got := testutil.CollectAndCount(m.calcOps); got != 3 {
		t.Errorf("calc_operations_total has %d series, want 3", got)
	}
}

// TestObserveRequestCountsAndTimes: one call feeds both HTTP families, the
// status is a label rather than a metric of its own, and the observation
// lands in a bucket instead of being dropped into +Inf.
func TestObserveRequestCountsAndTimes(t *testing.T) {
	m, reg := newTestMetrics(t)

	m.ObserveRequest(http.MethodPost, "/api/v1/calculate", http.StatusOK, 4*time.Millisecond)
	m.ObserveRequest(http.MethodPost, "/api/v1/calculate", http.StatusUnprocessableEntity, 2*time.Millisecond)

	if got := testutil.ToFloat64(m.requests.WithLabelValues(http.MethodPost, "/api/v1/calculate", "200")); got != 1 {
		t.Errorf(`http_requests_total{status="200"} = %v, want 1`, got)
	}
	if got := testutil.ToFloat64(m.requests.WithLabelValues(http.MethodPost, "/api/v1/calculate", "422")); got != 1 {
		t.Errorf(`http_requests_total{status="422"} = %v, want 1`, got)
	}

	// The histogram is one series for both, because it carries no status.
	if got := testutil.CollectAndCount(m.duration); got != 1 {
		t.Fatalf("http_request_duration_seconds has %d series, want 1", got)
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() != "http_request_duration_seconds" {
			continue
		}
		h := f.GetMetric()[0].GetHistogram()
		if h.GetSampleCount() != 2 {
			t.Errorf("sample count = %d, want 2", h.GetSampleCount())
		}
		if got, want := h.GetSampleSum(), 0.006; got < want-1e-9 || got > want+1e-9 {
			t.Errorf("sample sum = %v, want %v", got, want)
		}
		// 4 ms and 2 ms both sit under the 5 ms bound, which is the point
		// of having buckets at millisecond resolution.
		for _, b := range h.GetBucket() {
			if b.GetUpperBound() == 0.005 && b.GetCumulativeCount() != 2 {
				t.Errorf("le=0.005 bucket = %d, want 2", b.GetCumulativeCount())
			}
		}
	}
}

// TestInFlightGaugeGoesUpAndDown: the gauge is a level, so what matters is
// that it comes back to where it started.
func TestInFlightGaugeGoesUpAndDown(t *testing.T) {
	m, _ := newTestMetrics(t)

	if got := testutil.ToFloat64(m.inFlight); got != 0 {
		t.Fatalf("http_in_flight_requests = %v before anything happened, want 0", got)
	}
	m.IncInFlight()
	m.IncInFlight()
	if got := testutil.ToFloat64(m.inFlight); got != 2 {
		t.Errorf("http_in_flight_requests = %v with two requests in flight, want 2", got)
	}
	m.DecInFlight()
	m.DecInFlight()
	if got := testutil.ToFloat64(m.inFlight); got != 0 {
		t.Errorf("http_in_flight_requests = %v after both finished, want 0", got)
	}
}

// TestHandlerServesTheExposition: GET /metrics answers the Prometheus text
// format with the families on this Metrics' own registry.
func TestHandlerServesTheExposition(t *testing.T) {
	m, _ := newTestMetrics(t)
	m.ObserveRequest(http.MethodGet, "/healthz", http.StatusOK, time.Millisecond)
	m.ObserveCalculation("divide", "DIVISION_BY_ZERO")

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain…", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`http_requests_total{method="GET",route="/healthz",status="200"} 1`,
		`calc_operations_total{operation="divide",outcome="DIVISION_BY_ZERO"} 1`,
		`build_info{commit="abc1234",version="v1.2.3"} 1`,
		"http_in_flight_requests 0",
		`http_request_duration_seconds_bucket{method="GET",route="/healthz",le="0.001"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("exposition is missing %q\n%s", want, body)
		}
	}
}

// TestHandlerWithoutAGatherer: a registerer that cannot gather is write-only
// by design, so the endpoint says so instead of serving an empty 200 that
// looks like a service producing no traffic.
func TestHandlerWithoutAGatherer(t *testing.T) {
	writeOnly := prometheus.WrapRegistererWith(prometheus.Labels{"replica": "a"}, prometheus.NewRegistry())
	m := NewMetrics(writeOnly, testBuild)

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cannot be gathered") {
		t.Errorf("body = %q", rec.Body)
	}
	// It still records: the families are registered on the wrapped registry.
	m.ObserveCalculation("add", OutcomeOK)
	if got := testutil.ToFloat64(m.calcOps.WithLabelValues("add", OutcomeOK)); got != 1 {
		t.Errorf("calc_operations_total = %v, want 1", got)
	}
}

// TestNewMetricsWithoutARegistry: a nil registerer registers nowhere, so a
// caller that does not care about metrics still gets a usable handle rather
// than a nil dereference.
func TestNewMetricsWithoutARegistry(t *testing.T) {
	m := NewMetrics(nil, testBuild)
	m.ObserveRequest(http.MethodGet, "/healthz", http.StatusOK, time.Millisecond)
	m.ObserveCalculation("add", OutcomeOK)
	m.IncInFlight()
	m.DecInFlight()

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// TestNewRegistryCarriesTheRuntimeCollectors: what a bare registry does not
// have and an operator needs — goroutines, heap, file descriptors, process
// start time — comes from the two collectors NewRegistry adds.
func TestNewRegistryCarriesTheRuntimeCollectors(t *testing.T) {
	reg := NewRegistry()
	NewMetrics(reg, testBuild)

	names := gatheredNames(t, reg)
	for _, want := range []string{"build_info", "go_goroutines", "process_start_time_seconds"} {
		if !slices.Contains(names, want) {
			t.Errorf("registry is missing %q; it has %v", want, names)
		}
	}
}

// TestNewMetricsPanicsOnASecondRegistration: a duplicate registration is a
// wiring bug, and MustRegister makes it a startup panic rather than a silent
// double count.
func TestNewMetricsPanicsOnASecondRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	NewMetrics(reg, testBuild)

	defer func() {
		if recover() == nil {
			t.Error("a second NewMetrics on the same registry did not panic")
		}
	}()
	NewMetrics(reg, testBuild)
}

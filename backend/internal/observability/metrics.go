package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// OutcomeOK is the calc_operations_total outcome of a calculation that
// produced a result. Every other outcome is the stable problem code of the
// failure (DIVISION_BY_ZERO, VALIDATION_FAILED, …), so one query splits
// "people are dividing by zero" from "a client is sending broken bodies".
const OutcomeOK = "ok"

// durationBuckets are the upper bounds, in seconds, of
// http_request_duration_seconds. They stop at 2.5 s because this service
// computes one floating-point operation: everything it does belongs in the
// first few buckets, and a request past 2.5 s is not a latency figure but an
// incident — REQUEST_TIMEOUT cuts it off at 5 s by default. The resolution
// is where the answers are, between 1 ms and 100 ms, so a p99 read off the
// histogram is a real number rather than an interpolation across one wide
// bucket. Eleven buckets × the route/method cardinality is the whole cost.
var durationBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5}

// Metrics is the service's Prometheus instrumentation: the metric families,
// the registry they live on, and the handler that exposes them. It is
// created once in the composition root and injected into the middleware and
// the API handler, so nothing here touches a package-level registry and a
// test can hold a registry of its own.
//
// Every method is safe for concurrent use; that is what the underlying
// prometheus collectors are for.
type Metrics struct {
	reg prometheus.Registerer

	requests  *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	inFlight  prometheus.Gauge
	calcOps   *prometheus.CounterVec
	buildInfo *prometheus.GaugeVec
}

// NewRegistry returns the registry the running process exports: a private
// one with the Go runtime and process collectors on it.
//
// Deliberately not prometheus.DefaultRegisterer. A global registry is shared
// with every library that has ever imported the client, it cannot be reset
// between tests, and a duplicate registration in one of them panics the
// process at startup. Owning the registry makes the exported surface exactly
// what this repository registers.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}

// NewMetrics registers this service's metric families on reg and returns the
// handle the rest of the process records through. build is exported once, as
// build_info, and never read again.
//
// reg is a [prometheus.Registerer] rather than a *[prometheus.Registry] so a
// test can pass a fresh registry — or a wrapped one — and assert on it with
// prometheus/testutil. A nil reg registers nowhere, which keeps a handler
// constructible in a test that does not care about metrics.
//
// It panics if reg already carries these families, which is what
// MustRegister is for: two registrations of the same metric name is a wiring
// bug, and it is better found at startup than as a silent double count.
func NewMetrics(reg prometheus.Registerer, build BuildInfo) *Metrics {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	m := &Metrics{
		reg: reg,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests that reached the router, by method, matched route pattern and response status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "How long the server took to answer, in seconds, by method and matched route pattern.",
			Buckets: durationBuckets,
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "HTTP requests being served right now.",
		}),
		calcOps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "calc_operations_total",
			Help: `Calculations attempted, by operation and outcome ("ok" or the problem code of the failure).`,
		}, []string{"operation", "outcome"}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "build_info",
			Help: "Always 1; the build identity of the running binary is in the labels.",
		}, []string{"version", "commit"}),
	}
	reg.MustRegister(m.requests, m.duration, m.inFlight, m.calcOps, m.buildInfo)

	// The info-metric convention: a constant 1 whose labels carry the facts,
	// so a dashboard can join a deploy onto a latency graph without a
	// separate data source.
	m.buildInfo.WithLabelValues(build.Version, build.Commit).Set(1)
	return m
}

// Handler serves the registry in the Prometheus text exposition format. It
// is mounted at GET /metrics on the service's own listener; moving it to a
// separate admin port is follow-up #38.
//
// A [prometheus.Registerer] that cannot also gather — a wrapped registerer,
// which is write-only by design — has nothing to expose, so the endpoint
// says so with a 500 instead of lying with an empty 200.
func (m *Metrics) Handler() http.Handler {
	gatherer, ok := m.reg.(prometheus.Gatherer)
	if !ok {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "metrics registry cannot be gathered", http.StatusInternalServerError)
		})
	}
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}

// IncInFlight counts a request that has just started, and DecInFlight one
// that has finished. The middleware pairs them with a defer.
func (m *Metrics) IncInFlight() { m.inFlight.Inc() }

// DecInFlight counts a finished request; see [Metrics.IncInFlight].
func (m *Metrics) DecInFlight() { m.inFlight.Dec() }

// ObserveRequest records one finished request: the counter by outcome and
// the latency histogram. route is the router's matched pattern, never the
// raw path — see [github.com/isasumer/go-react-calculator/backend/internal/middleware.Metrics].
func (m *Metrics) ObserveRequest(method, route string, status int, d time.Duration) {
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route).Observe(d.Seconds())
}

// ObserveCalculation records one attempted calculation. operation is a name
// from the calc registry, outcome is [OutcomeOK] or the problem code the
// request failed with. Both label values come from fixed sets, so the
// family's cardinality is bounded by the code, not by what clients send.
func (m *Metrics) ObserveCalculation(operation, outcome string) {
	m.calcOps.WithLabelValues(operation, outcome).Inc()
}

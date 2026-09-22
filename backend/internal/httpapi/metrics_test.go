package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// instrumented builds a handler recording into a registry of its own, and
// returns both. Every test here gets a fresh pair, so the counts it asserts
// are the ones it produced.
func instrumented(t *testing.T) (http.Handler, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	metrics := observability.NewMetrics(reg, observability.BuildInfo{Version: "v1.2.3", Commit: "abc1234"})
	h := NewHandler(calc.NewRegistry(), nil, WithMetrics(metrics))
	return h.Routes(), reg
}

// post sends one calculate request through h.
func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/calculate", strings.NewReader(body))
	req.Header.Set("Content-Type", jsonCT)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// calcCount is the value of one calc_operations_total series, or -1 when
// that label set was never recorded.
func calcCount(t *testing.T, reg *prometheus.Registry, operation, outcome string) float64 {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "calc_operations_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := make(map[string]string, 2)
			for _, pair := range metric.GetLabel() {
				labels[pair.GetName()] = pair.GetValue()
			}
			if labels["operation"] == operation && labels["outcome"] == outcome {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return -1
}

// TestCalculateRecordsItsOutcome walks the calculate endpoint's outcomes and
// asserts each lands on its own series. The outcome is either "ok" or the
// stable problem code the client was given, so the counter and the client's
// error agree on what happened.
func TestCalculateRecordsItsOutcome(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		operation  string
		outcome    string
	}{
		{
			name:       "division by zero",
			body:       `{"operation":"divide","a":1,"b":0}`,
			wantStatus: http.StatusUnprocessableEntity,
			operation:  "divide",
			outcome:    "DIVISION_BY_ZERO",
		},
		{
			name:       "a result",
			body:       `{"operation":"divide","a":10,"b":4}`,
			wantStatus: http.StatusOK,
			operation:  "divide",
			outcome:    observability.OutcomeOK,
		},
		{
			name:       "sqrt of a negative",
			body:       `{"operation":"sqrt","a":-1}`,
			wantStatus: http.StatusUnprocessableEntity,
			operation:  "sqrt",
			outcome:    "DOMAIN_ERROR",
		},
		{
			name:       "overflow",
			body:       `{"operation":"power","a":1e308,"b":2}`,
			wantStatus: http.StatusUnprocessableEntity,
			operation:  "power",
			outcome:    "RESULT_NOT_FINITE",
		},
		{
			name:       "a missing operand never names an operation",
			body:       `{"operation":"add","a":1}`,
			wantStatus: http.StatusBadRequest,
			operation:  string(unknownOperation),
			outcome:    "VALIDATION_FAILED",
		},
		{
			name:       "a malformed body never names an operation",
			body:       `{"operation":"add","a":`,
			wantStatus: http.StatusBadRequest,
			operation:  string(unknownOperation),
			outcome:    "INVALID_BODY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reg := instrumented(t)

			if rec := post(t, h, tt.body); rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.wantStatus, rec.Body)
			}

			if got := calcCount(t, reg, tt.operation, tt.outcome); got != 1 {
				t.Errorf("calc_operations_total{operation=%q,outcome=%q} = %v, want 1",
					tt.operation, tt.outcome, got)
			}
			if got := testutil.CollectAndCount(reg, "calc_operations_total"); got != 1 {
				t.Errorf("one request produced %d series, want 1", got)
			}
		})
	}
}

// TestCalculateDoesNotLabelWithClientInput is the cardinality guard: the
// operation label comes from the registry, so an unsupported operation is
// counted as "unknown" rather than as whatever string arrived. Otherwise one
// request per invented name is one time series per invented name, and the
// metrics backend is a denial-of-service target from the open internet.
func TestCalculateDoesNotLabelWithClientInput(t *testing.T) {
	h, reg := instrumented(t)

	for _, name := range []string{"modulo", "√", strings.Repeat("x", 200)} {
		body := `{"operation":"` + name + `","a":1,"b":2}`
		if rec := post(t, h, body); rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("operation %q = %d, want 422", name, rec.Code)
		}
		if got := calcCount(t, reg, name, "UNSUPPORTED_OPERATION"); got != -1 {
			t.Errorf("the client's operation name became a series: %v", got)
		}
	}

	if got := calcCount(t, reg, string(unknownOperation), "UNSUPPORTED_OPERATION"); got != 3 {
		t.Errorf(`calc_operations_total{operation="unknown"} = %v, want 3`, got)
	}
	if got := testutil.CollectAndCount(reg, "calc_operations_total"); got != 1 {
		t.Errorf("three made-up operations produced %d series, want 1", got)
	}
}

// TestMetricsEndpoint: GET /metrics is served on the API's own listener, as
// an uncacheable snapshot of this process.
func TestMetricsEndpoint(t *testing.T) {
	h, _ := instrumented(t)
	post(t, h, `{"operation":"divide","a":1,"b":0}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != noStore {
		t.Errorf("Cache-Control = %q, want %q", got, noStore)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain…", ct)
	}
	for _, want := range []string{
		`calc_operations_total{operation="divide",outcome="DIVISION_BY_ZERO"} 1`,
		`build_info{commit="abc1234",version="v1.2.3"} 1`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("exposition is missing %q\n%s", want, rec.Body)
		}
	}
}

// TestMetricsEndpointRejectsOtherMethods: /metrics answers like the other
// operational endpoints — a problem document with an Allow header, not the
// catch-all 404.
func TestMetricsEndpointRejectsOtherMethods(t *testing.T) {
	h, _ := instrumented(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/metrics", http.NoBody))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /metrics = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want %q", got, "GET, HEAD")
	}
}

// TestWithoutMetricsNothingIsExposedOrCounted: the option is what turns the
// endpoint on. A handler built without it serves the API exactly as before
// and has no /metrics route to leak.
func TestWithoutMetricsNothingIsExposedOrCounted(t *testing.T) {
	h := NewHandler(calc.NewRegistry(), nil).Routes()

	if rec := post(t, h, `{"operation":"divide","a":1,"b":0}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("calculate = %d, want 422", rec.Code)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /metrics without WithMetrics = %d, want 404", rec.Code)
	}
}

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// testBuild is a fully stamped build identity, so /version assertions do not
// depend on how the test binary itself was built.
var testBuild = observability.BuildInfo{
	Version:   "v1.2.3",
	Commit:    "abc1234",
	BuildDate: "2026-09-22T10:11:12Z",
	GoVersion: "go1.27.1",
}

// serveProbe runs one request through h's routes, with the X-Request-ID the
// middleware (B1-04) will set, as [serve] does for the API routes.
func serveProbe(t *testing.T, h *Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", exampleRequestID)
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func probeHandler(t *testing.T, ready func() bool) *Handler {
	t.Helper()
	return NewHandler(calc.NewRegistry(), nil, WithReadiness(ready), WithBuildInfo(testBuild))
}

func TestProbes(t *testing.T) {
	notReady := func() bool { return false }

	tests := []struct {
		name       string
		method     string
		path       string
		ready      func() bool
		wantStatus int
		wantBody   string
		wantCT     string
	}{
		{
			name: "healthz", method: http.MethodGet, path: "/healthz",
			wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`,
			wantCT: "application/json; charset=utf-8",
		},
		{
			name: "healthz stays ok while draining", method: http.MethodGet, path: "/healthz",
			ready:      notReady,
			wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`,
			wantCT: "application/json; charset=utf-8",
		},
		{
			name: "readyz ready", method: http.MethodGet, path: "/readyz",
			wantStatus: http.StatusOK, wantBody: `{"status":"ready"}`,
			wantCT: "application/json; charset=utf-8",
		},
		{
			name: "readyz draining", method: http.MethodGet, path: "/readyz",
			ready:      notReady,
			wantStatus: http.StatusServiceUnavailable,
			wantCT:     "application/problem+json",
		},
		{
			name: "version", method: http.MethodGet, path: "/version",
			wantStatus: http.StatusOK,
			wantBody:   `{"version":"v1.2.3","commit":"abc1234","buildDate":"2026-09-22T10:11:12Z","goVersion":"go1.27.1"}`,
			wantCT:     "application/json; charset=utf-8",
		},
		// A HEAD is served by the GET pattern: same headers, no body.
		{
			name: "healthz head", method: http.MethodHead, path: "/healthz",
			wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`,
			wantCT: "application/json; charset=utf-8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveProbe(t, probeHandler(t, tt.ready), tt.method, tt.path)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantCT {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantCT)
			}
			// A probe answer must never be cached: a stale readiness would
			// keep a load balancer routing to a draining instance.
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want %q", got, "no-store")
			}
			if tt.wantBody != "" {
				if got := rec.Body.String(); got != tt.wantBody {
					t.Errorf("body = %s, want %s", got, tt.wantBody)
				}
			}
		})
	}
}

// TestReadyzFlipsWithTheFlag drives the probe the way the server lifecycle
// does: one atomic flag, flipped off before the drain starts.
func TestReadyzFlipsWithTheFlag(t *testing.T) {
	var ready atomic.Bool
	h := probeHandler(t, ready.Load)

	if rec := serveProbe(t, h, http.MethodGet, "/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("before the listener is bound: status = %d, want 503", rec.Code)
	}

	ready.Store(true)
	rec := serveProbe(t, h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ready"}` {
		t.Errorf("while serving: status = %d body = %s, want 200 {\"status\":\"ready\"}", rec.Code, rec.Body)
	}

	ready.Store(false)
	rec = serveProbe(t, h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("while draining: status = %d, want 503", rec.Code)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if p.Code != CodeNotReady || p.Status != http.StatusServiceUnavailable || p.Instance != "/readyz" {
		t.Errorf("problem = %+v", p)
	}
	assertGolden(t, "not_ready", rec.Body.Bytes())
}

// TestProbesMethodNotAllowed: probes answer a wrong method the way every
// other route does.
func TestProbesMethodNotAllowed(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz", "/version"} {
		t.Run(path, func(t *testing.T) {
			rec := serveProbe(t, probeHandler(t, nil), http.MethodPost, path)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want 405", rec.Code)
			}
			if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
				t.Errorf("Allow = %q, want %q", got, "GET, HEAD")
			}
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", got)
			}
		})
	}
}

// TestHandlerDefaults: without options a handler is ready and reports this
// binary's own build information, so /readyz and /version never answer with
// an empty document.
func TestHandlerDefaults(t *testing.T) {
	h := NewHandler(calc.NewRegistry(), nil)

	if rec := serveProbe(t, h, http.MethodGet, "/readyz"); rec.Code != http.StatusOK {
		t.Errorf("/readyz status = %d, want 200", rec.Code)
	}

	rec := serveProbe(t, h, http.MethodGet, "/version")
	if rec.Code != http.StatusOK {
		t.Fatalf("/version status = %d, want 200", rec.Code)
	}
	var got VersionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := observability.Build()
	if got != (VersionResponse{want.Version, want.Commit, want.BuildDate, want.GoVersion}) {
		t.Errorf("/version = %+v, want %+v", got, want)
	}
}

// TestWithReadinessNil: a nil function would panic on the first probe, so it
// leaves the default in place.
func TestWithReadinessNil(t *testing.T) {
	h := NewHandler(calc.NewRegistry(), nil, WithReadiness(nil))
	if rec := serveProbe(t, h, http.MethodGet, "/readyz"); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

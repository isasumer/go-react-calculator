package middleware

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

// probe is a middleware that records when it is entered and when it is left,
// so one slice shows the whole path of a request through the chain.
func probe(name string, trace *[]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*trace = append(*trace, "enter "+name)
			next.ServeHTTP(w, r)
			*trace = append(*trace, "exit "+name)
		})
	}
}

func TestChainOrder(t *testing.T) {
	var trace []string
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		trace = append(trace, "handler")
	})

	h := Chain(handler, probe("a", &trace), probe("b", &trace), probe("c", &trace))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	// First listed is outermost: entered first, left last.
	want := []string{"enter a", "enter b", "enter c", "handler", "exit c", "exit b", "exit a"}
	if !slices.Equal(trace, want) {
		t.Errorf("trace =\n%v\nwant\n%v", trace, want)
	}
}

func TestChainSkipsNilSlots(t *testing.T) {
	var trace []string
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		trace = append(trace, "handler")
	})

	// An unconfigured layer is passed as nil rather than omitted.
	h := Chain(handler, probe("a", &trace), nil, probe("b", &trace), nil)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	want := []string{"enter a", "enter b", "handler", "exit b", "exit a"}
	if !slices.Equal(trace, want) {
		t.Errorf("trace =\n%v\nwant\n%v", trace, want)
	}
}

func TestChainWithoutMiddlewareReturnsTheHandler(t *testing.T) {
	handler := ok()
	if got := Chain(handler); got == nil {
		t.Fatal("Chain returned nil")
	}
	rec := httptest.NewRecorder()
	Chain(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("got %d %q", rec.Code, rec.Body)
	}
}

// TestChainProductionOrder assembles the chain the way cmd/server does and
// asserts the properties only that order produces: the request ID is set
// before anything can answer, the rate limiter's rejection still carries the
// security headers and the ID, and the router is reached last.
func TestChainProductionOrder(t *testing.T) {
	var reached int
	router := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})
	h := Chain(router,
		Recover(nil),
		RequestID(),
		Logger(nil),
		Timeout(time.Second),
		SecurityHeaders(),
		CORS([]string{"http://localhost:5173"}),
		RateLimit(RateLimitConfig{RPS: 1, Burst: 1}),
		Metrics(nil), // a chain assembled without a registry
	)

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/v1/calculate", http.NoBody))
	if first.Code != http.StatusOK || reached != 1 {
		t.Fatalf("first request = %d, router reached %d times", first.Code, reached)
	}

	// Second request from the same client: the rate limiter answers, the
	// router is never reached, and the response is still a complete one.
	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/v1/calculate", http.NoBody))
	if second.Code != http.StatusTooManyRequests || reached != 1 {
		t.Fatalf("second request = %d, router reached %d times", second.Code, reached)
	}
	for header, want := range map[string]string{
		"Content-Type":           "application/problem+json",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "no-store",
		"Retry-After":            "1",
	} {
		if got := second.Header().Get(header); got != want {
			t.Errorf("429 %s = %q, want %q", header, got, want)
		}
	}
	if second.Header().Get(HeaderRequestID) == "" {
		t.Error("429 carries no X-Request-ID")
	}
}

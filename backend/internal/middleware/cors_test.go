package middleware

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

const devOrigin = "http://localhost:5173"

// corsHeaders are the ones a response must not carry when the origin is not
// allowed or the allowlist is empty.
var corsHeaders = []string{
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Max-Age",
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		method      string
		origin      string
		wantStatus  int
		wantReached bool
		wantHeaders map[string]string
		wantVary    []string
		wantNoCORS  bool
	}{
		{
			name:        "preflight from an allowed origin",
			allowed:     []string{devOrigin, "https://calc.example"},
			method:      http.MethodOptions,
			origin:      devOrigin,
			wantStatus:  http.StatusNoContent,
			wantReached: false,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  devOrigin,
				"Access-Control-Allow-Methods": "POST, GET, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, X-Request-ID",
				"Access-Control-Max-Age":       "600",
			},
			wantVary: []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"},
		},
		{
			name:        "preflight from a disallowed origin",
			allowed:     []string{devOrigin},
			method:      http.MethodOptions,
			origin:      "https://evil.example",
			wantStatus:  http.StatusOK, // the router answers; no CORS headers
			wantReached: true,
			wantVary:    []string{"Origin"},
			wantNoCORS:  true,
		},
		{
			name:        "simple request from an allowed origin",
			allowed:     []string{devOrigin},
			method:      http.MethodPost,
			origin:      devOrigin,
			wantStatus:  http.StatusOK,
			wantReached: true,
			wantHeaders: map[string]string{"Access-Control-Allow-Origin": devOrigin},
			wantVary:    []string{"Origin"},
		},
		{
			name:        "request from a disallowed origin is still served",
			allowed:     []string{devOrigin},
			method:      http.MethodPost,
			origin:      "https://evil.example",
			wantStatus:  http.StatusOK,
			wantReached: true,
			wantVary:    []string{"Origin"},
			wantNoCORS:  true,
		},
		{
			name:        "same-origin request, no Origin header",
			allowed:     []string{devOrigin},
			method:      http.MethodPost,
			wantStatus:  http.StatusOK,
			wantReached: true,
			wantVary:    []string{"Origin"},
			wantNoCORS:  true,
		},
		{
			name:        "an origin that only looks allowed",
			allowed:     []string{"https://calc.example"},
			method:      http.MethodPost,
			origin:      "https://calc.example.evil.test",
			wantStatus:  http.StatusOK,
			wantReached: true,
			wantVary:    []string{"Origin"},
			wantNoCORS:  true,
		},
		{
			name:        "empty allowlist adds nothing at all",
			allowed:     nil,
			method:      http.MethodOptions,
			origin:      devOrigin,
			wantStatus:  http.StatusOK,
			wantReached: true,
			wantNoCORS:  true,
			// Not even Vary: this middleware never looks at the header.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			h := CORS(tt.allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/api/v1/calculate", http.NoBody)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
				req.Header.Set("Access-Control-Request-Headers", "content-type")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if reached != tt.wantReached {
				t.Errorf("handler reached = %v, want %v", reached, tt.wantReached)
			}
			for name, want := range tt.wantHeaders {
				if got := rec.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			if tt.wantNoCORS {
				for _, name := range corsHeaders {
					if got := rec.Header().Get(name); got != "" {
						t.Errorf("%s = %q, want no CORS headers", name, got)
					}
				}
			}
			if gotVary := rec.Header().Values("Vary"); !slices.Equal(gotVary, tt.wantVary) {
				t.Errorf("Vary = %v, want %v", gotVary, tt.wantVary)
			}
			// The allowlist is a list, never a wildcard.
			if strings.Contains(rec.Header().Get("Access-Control-Allow-Origin"), "*") {
				t.Error("Access-Control-Allow-Origin is a wildcard")
			}
		})
	}
}

// TestCORSPreflightCarriesTheRestOfTheChain: the preflight answer never
// reaches the router, so it has to be the middleware above CORS that gives it
// its security headers and request ID.
func TestCORSPreflightCarriesTheRestOfTheChain(t *testing.T) {
	h := Chain(ok(), RequestID(), SecurityHeaders(), CORS([]string{devOrigin}))

	req := newRequest(t, http.MethodOptions, "/api/v1/calculate")
	req.Header.Set("Origin", devOrigin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("preflight = %d with a %d-byte body, want 204 and nothing", rec.Code, rec.Body.Len())
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get(HeaderRequestID); got != exampleRequestID {
		t.Errorf("X-Request-ID = %q", got)
	}
}

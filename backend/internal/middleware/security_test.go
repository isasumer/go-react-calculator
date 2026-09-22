package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	constant := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	}
	tests := []struct {
		name        string
		path        string
		wantNoStore bool
	}{
		{name: "api route", path: "/api/v1/calculate", wantNoStore: true},
		{name: "api route with a query", path: "/api/v1/operations?x=1", wantNoStore: true},
		{name: "probe sets its own cache header", path: "/healthz"},
		{name: "unknown path", path: "/nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Set on the way in, so the handler can already see them.
				for name, want := range constant {
					if got := w.Header().Get(name); got != want {
						t.Errorf("handler sees %s = %q, want %q", name, got, want)
					}
				}
				w.WriteHeader(http.StatusOK)
			}))

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, http.NoBody))

			for name, want := range constant {
				if got := rec.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			got := rec.Header().Get("Cache-Control")
			if tt.wantNoStore && got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store on an API route", got)
			}
			if !tt.wantNoStore && got != "" {
				t.Errorf("Cache-Control = %q, want none outside /api/", got)
			}
		})
	}
}

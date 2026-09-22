package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHealthcheckURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"default port", nil, "http://127.0.0.1:8081/readyz"},
		{"explicit port", map[string]string{"PORT": "9000"}, "http://127.0.0.1:9000/readyz"},
		{"blank port falls back", map[string]string{"PORT": "  "}, "http://127.0.0.1:8081/readyz"},
		{"unparsable port falls back", map[string]string{"PORT": "eight"}, "http://127.0.0.1:8081/readyz"},
		{"out-of-range port falls back", map[string]string{"PORT": "70000"}, "http://127.0.0.1:8081/readyz"},
		// HOST is deliberately ignored: the probe runs inside the container.
		{"host ignored", map[string]string{"HOST": "0.0.0.0", "PORT": "8082"}, "http://127.0.0.1:8082/readyz"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := healthcheckURL(envFunc(tc.env)); got != tc.want {
				t.Errorf("healthcheckURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHealthcheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		wantErr string // substring; empty means success
	}{
		{"ready", http.StatusOK, ""},
		{"not ready", http.StatusServiceUnavailable, "503"},
		{"server error", http.StatusInternalServerError, "500"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var gotPath, gotMethod string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotMethod = r.URL.Path, r.Method
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("body"))
			}))
			defer srv.Close()

			err := healthcheck(t.Context(), srv.URL+"/readyz")
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("healthcheck() = %v, want nil", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("healthcheck() = nil, want error containing %q", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("healthcheck() = %v, want error containing %q", err, tc.wantErr)
			}
			if gotMethod != http.MethodGet {
				t.Errorf("method = %q, want GET", gotMethod)
			}
			if gotPath != "/readyz" {
				t.Errorf("path = %q, want /readyz", gotPath)
			}
		})
	}
}

func TestHealthcheckNoListener(t *testing.T) {
	t.Parallel()

	// A port nothing is listening on: the probe must fail rather than hang.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed := srv.URL
	srv.Close()

	if err := healthcheck(t.Context(), closed+"/readyz"); err == nil {
		t.Fatal("healthcheck() = nil, want a connection error")
	}
}

func TestHealthcheckBadURL(t *testing.T) {
	t.Parallel()

	if err := healthcheck(t.Context(), "http://127.0.0.1:1\x7f/readyz"); err == nil {
		t.Fatal("healthcheck() = nil, want a request-construction error")
	}
}

func TestHealthcheckCanceledContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := healthcheck(ctx, srv.URL+"/readyz")
	if err == nil {
		t.Fatal("healthcheck() = nil, want a context error")
	}
	var uerr *url.Error
	if !errors.As(err, &uerr) {
		t.Fatalf("healthcheck() = %v, want a *url.Error", err)
	}
}

// TestRunHealthcheckFlag drives the flag the way Docker's HEALTHCHECK does:
// through run(), with no server listening, so it must return an error (which
// main turns into exit 1).
func TestRunHealthcheckFlag(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	// Port 1 is privileged and unbound in the test environment; the probe
	// fails fast instead of reaching anything real.
	err := run(t.Context(), []string{"server", "-healthcheck"}, envFunc(map[string]string{"PORT": "1"}), &out)
	if err == nil {
		t.Fatal("run(-healthcheck) = nil, want an error when nothing is listening")
	}
	if !strings.Contains(err.Error(), "healthcheck") {
		t.Errorf("run(-healthcheck) = %v, want the error to name the healthcheck", err)
	}
	if out.String() != "" {
		t.Errorf("run(-healthcheck) wrote %q to stdout, want nothing", out.String())
	}
}

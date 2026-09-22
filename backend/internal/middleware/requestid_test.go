package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDIncoming(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		wantKept bool
	}{
		{"absent", "", false},
		{"plain", "abc123", true},
		{"dashes and underscores", "trace_id-42", true},
		{"64 characters", strings.Repeat("a", 64), true},
		{"65 characters", strings.Repeat("a", 65), false},
		{"uuid from an upstream proxy", exampleRequestID, true},
		{"spaces", "two words", false},
		{"header injection", "abc\r\nX-Admin: true", false},
		{"punctuation", "id:42", false},
		{"unicode", "kimlik-ç", false},
		{"only whitespace", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The header is set before the handler runs, so a handler
				// that writes its own response can echo it.
				if got := w.Header().Get(HeaderRequestID); got == "" {
					t.Error("X-Request-ID is not on the response when the handler runs")
				}
				seen = FromContext(r.Context())
			}))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			if tt.incoming != "" {
				req.Header.Set(HeaderRequestID, tt.incoming)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			got := rec.Header().Get(HeaderRequestID)
			if got != seen {
				t.Errorf("header %q and context %q disagree", got, seen)
			}
			switch {
			case tt.wantKept && got != tt.incoming:
				t.Errorf("incoming %q was replaced by %q", tt.incoming, got)
			case !tt.wantKept && got == tt.incoming:
				t.Errorf("invalid incoming ID %q was reused", tt.incoming)
			case !tt.wantKept && !requestIDPattern.MatchString(got):
				t.Errorf("generated ID %q does not match the pattern", got)
			}
		})
	}
}

func TestNewRequestIDIsRandomAndSafe(t *testing.T) {
	const n = 1000
	seen := make(map[string]bool, n)
	for range n {
		id := newRequestID()
		// 16 bytes of base32 without padding.
		if len(id) != 26 {
			t.Fatalf("id %q has length %d, want 26", id, len(id))
		}
		if !requestIDPattern.MatchString(id) {
			t.Fatalf("generated id %q would not be accepted as an incoming one", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q after %d draws", id, len(seen))
		}
		seen[id] = true
	}
}

func TestFromContextWithoutMiddleware(t *testing.T) {
	if got := FromContext(t.Context()); got != "" {
		t.Errorf("FromContext = %q, want empty outside the chain", got)
	}
}

// TestRequestIDOfFallsBackToTheHeader covers what Recover relies on: it wraps
// RequestID, so it only ever sees the outer request, without the context.
func TestRequestIDOfFallsBackToTheHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	if got := requestIDOf(rec, req); got != "" {
		t.Errorf("requestIDOf = %q with nothing set, want empty", got)
	}
	rec.Header().Set(HeaderRequestID, "from-header")
	if got := requestIDOf(rec, req); got != "from-header" {
		t.Errorf("requestIDOf = %q, want the response header", got)
	}
}

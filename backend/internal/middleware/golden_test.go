package middleware

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the problem golden files this package owns")

// exampleRequestID is the placeholder docs/errors.md and the golden files
// show. It matches requestIDPattern, so the middleware honors it as an
// incoming ID and the examples are what a real client would receive.
const exampleRequestID = "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"

// assertProblemGolden compares an indented copy of body with the golden file
// in the httpapi package's testdata.
//
// The goldens for the problems this package produces — TIMEOUT and
// RATE_LIMITED — live next to the ones httpapi produces because
// httpapi.TestErrorCatalogueMatchesGoldenFiles reads them from there to check
// docs/errors.md. Written here, from the real middleware response, so the
// documentation cannot drift from what the chain actually sends.
func assertProblemGolden(t *testing.T, name string, body []byte) {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, body)
	}
	buf.WriteByte('\n')
	path := filepath.Join("..", "httpapi", "testdata", name+".json")
	if *update {
		if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run go test ./internal/middleware -update to create): %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("body mismatch with %s\n got: %s\nwant: %s", path, buf.Bytes(), want)
	}
}

// newRequest builds a request with the example ID already on it, so the
// golden files show a complete problem document.
func newRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), method, target, http.NoBody)
	r.Header.Set(HeaderRequestID, exampleRequestID)
	return r
}

// ok is a handler that answers 200 with a short body, so a test can tell "the
// request reached the router" from "something above it answered".
func ok() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

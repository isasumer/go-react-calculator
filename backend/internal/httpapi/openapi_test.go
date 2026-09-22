package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"

	"github.com/isasumer/go-react-calculator/backend/api"
)

// loadContract parses and validates backend/api/openapi.yaml once for the
// whole package: every table case below validates a response against it, and
// parsing it per case would dominate the test's runtime.
var loadContract = sync.OnceValues(func() (*openapi3.T, error) {
	ctx := context.Background()
	loader := &openapi3.Loader{Context: ctx}
	doc, err := loader.LoadFromData(api.Spec())
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	// Validate checks the document against OpenAPI 3.1 and, because example
	// validation is on by default, every example against the schema of the
	// response it illustrates.
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return doc, nil
})

// contract returns the loaded contract, or fails the test.
func contract(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := loadContract()
	if err != nil {
		t.Fatalf("backend/api/openapi.yaml: %v", err)
	}
	return doc
}

// errPathNotDescribed and errMethodNotDescribed are what [findRoute] reports
// for the two answers the router produces on its own, below any operation.
var (
	errPathNotDescribed   = errors.New("no path item for this path")
	errMethodNotDescribed = errors.New("no operation for this method")
)

// findRoute resolves req against the contract. The API has no path
// templates, so an exact lookup is the whole router: it needs no path
// parameters, keeps gorilla/mux out of the module graph, and tells a path the
// document does not describe (a 404 from the catch-all) from a method it does
// not describe on a path it does (a 405 with an Allow header).
//
// HEAD resolves to the GET operation, which is what net/http's router does
// with a "GET /path" pattern.
func findRoute(doc *openapi3.T, req *http.Request) (*routers.Route, error) {
	method := req.Method
	if method == http.MethodHead {
		method = http.MethodGet
	}
	item := doc.Paths.Value(req.URL.Path)
	if item == nil {
		return nil, errPathNotDescribed
	}
	op := item.GetOperation(method)
	if op == nil {
		return nil, errMethodNotDescribed
	}
	return &routers.Route{
		Spec:      doc,
		Path:      req.URL.Path,
		PathItem:  item,
		Method:    method,
		Operation: op,
	}, nil
}

// assertMatchesSpec fails unless resp is a response the contract describes
// for req: the status must be documented for the operation, the response's
// content type must be one it declares, declared headers must be present, and
// the body must satisfy the schema. The `code` enum is narrowed per status,
// so a problem carrying a code that does not belong to its status fails here
// even though it is a perfectly good problem document.
//
// It is called from the handler table test on every case, which is what makes
// the contract an assertion about the implementation rather than a file that
// happens to sit in the repository.
func assertMatchesSpec(t *testing.T, req *http.Request, resp *http.Response) {
	t.Helper()
	doc := contract(t)

	route, err := findRoute(doc, req)
	switch {
	case errors.Is(err, errPathNotDescribed):
		assertMatchesResponse(t, doc, "NotFound", resp)
		return
	case errors.Is(err, errMethodNotDescribed):
		assertMatchesResponse(t, doc, "MethodNotAllowed", resp)
		return
	case err != nil:
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}

	// ValidateResponse returns early for HEAD (there is no body to check),
	// so that case asserts only that the route is described.
	err = openapi3filter.ValidateResponse(req.Context(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, Route: route},
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
		Body:                   resp.Body,
		Options: &openapi3filter.Options{
			// An undocumented status is a failure, not a pass: the point is
			// that the contract covers every answer the service gives.
			IncludeResponseStatus: true,
			MultiError:            true,
		},
	})
	if err != nil {
		t.Errorf("%s %s -> %d does not match the contract: %v",
			req.Method, req.URL.Path, resp.StatusCode, err)
	}
}

// assertMatchesResponse validates resp against a response in
// components/responses, for the two answers that belong to no operation.
func assertMatchesResponse(t *testing.T, doc *openapi3.T, name string, resp *http.Response) {
	t.Helper()
	ref := doc.Components.Responses[name]
	if ref == nil || ref.Value == nil {
		t.Fatalf("the contract has no components/responses/%s", name)
	}
	contentType := resp.Header.Get("Content-Type")
	media := ref.Value.Content.Get(contentType)
	if media == nil || media.Schema == nil {
		t.Fatalf("components/responses/%s describes no %q body", name, contentType)
	}
	for header, headerRef := range ref.Value.Headers {
		if headerRef.Value.Required && resp.Header.Get(header) == "" {
			t.Errorf("%s response: header %s is missing", name, header)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("%s response is not JSON: %v\n%s", name, err, body)
	}
	if err := media.Schema.Value.VisitJSON(value,
		openapi3.MultiErrors(), openapi3.VisitAsResponse(), openapi3.EnableJSONSchema2020()); err != nil {
		t.Errorf("%d response does not match components/responses/%s: %v\n%s",
			resp.StatusCode, name, err, body)
	}
}

// TestContractIsValid is the loader-and-validator half of the acceptance
// criteria (spectral is the other half, in CI): the document parses as
// OpenAPI 3.1, every example in it satisfies the schema it illustrates, and
// the paths and codes it describes are the ones this package serves.
func TestContractIsValid(t *testing.T) {
	doc := contract(t)

	if got := doc.OpenAPI; !strings.HasPrefix(got, "3.1") {
		t.Errorf("openapi = %q, want 3.1.x", got)
	}

	// Every route the handler registers is described, including the two the
	// platform uses and the spec's own endpoint.
	for _, path := range []string{
		"/api/v1/calculate", "/api/v1/operations", "/api/v1/openapi.yaml",
		"/healthz", "/readyz", "/version", "/metrics",
	} {
		if doc.Paths.Value(path) == nil {
			t.Errorf("no path item for %s", path)
		}
	}

	// The code enum is the error catalog, exactly: a code the service can
	// send but the contract does not list would make a client's exhaustive
	// switch wrong.
	schema := doc.Components.Schemas["ProblemCode"]
	if schema == nil || schema.Value == nil {
		t.Fatal("no ProblemCode schema")
	}
	listed := make(map[string]bool, len(schema.Value.Enum))
	for _, v := range schema.Value.Enum {
		code, ok := v.(string)
		if !ok {
			t.Fatalf("ProblemCode enum holds %T, want string", v)
		}
		listed[code] = true
	}
	for _, code := range allCodes {
		if !listed[string(code)] {
			t.Errorf("ProblemCode enum is missing %s", code)
		}
		delete(listed, string(code))
	}
	for code := range listed {
		t.Errorf("ProblemCode enum has %s, which no handler sends", code)
	}
}

// TestContractExamplesMatchGoldenFiles keeps the one-truth rule: the example
// the contract shows for a code is the golden file the handler tests hold the
// implementation to, field for field and byte for byte once re-encoded
// through the Problem the server writes.
func TestContractExamplesMatchGoldenFiles(t *testing.T) {
	doc := contract(t)

	for _, code := range allCodes {
		t.Run(string(code), func(t *testing.T) {
			name := exampleName(code)
			ex := doc.Components.Examples[name]
			if ex == nil || ex.Value == nil {
				t.Fatalf("no components/examples/%s for code %s", name, code)
			}

			// Round-trip through the type the server marshals: an example
			// with a stray or misspelled member fails to decode, and one
			// that decodes is re-encoded in exactly the server's field order.
			raw, err := json.Marshal(ex.Value.Value)
			if err != nil {
				t.Fatalf("re-encode example: %v", err)
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			var p Problem
			if err = dec.Decode(&p); err != nil {
				t.Fatalf("example %s is not a problem document: %v\n%s", name, err, raw)
			}
			got, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			var indented bytes.Buffer
			if err = json.Indent(&indented, got, "", "  "); err != nil {
				t.Fatal(err)
			}
			indented.WriteByte('\n')

			golden := filepath.Join("testdata", strings.ToLower(string(code))+".json")
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(indented.Bytes(), want) {
				t.Errorf("components/examples/%s differs from %s\n got: %s\nwant: %s",
					name, golden, indented.Bytes(), want)
			}
		})
	}
}

// exampleName is the component name holding the example for a code:
// DIVISION_BY_ZERO is components/examples/DivisionByZero.
func exampleName(code Code) string {
	parts := strings.Split(strings.ToLower(string(code)), "_")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// TestServeOpenAPISpec covers the endpoint itself: it serves the embedded
// bytes, says they are YAML, and refuses to be cached or to answer anything
// but a GET.
func TestServeOpenAPISpec(t *testing.T) {
	req, rec := serve(t, http.MethodGet, "/api/v1/openapi.yaml", "", http.NoBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/yaml" {
		t.Errorf("Content-Type = %q, want application/yaml", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), api.Spec()) {
		t.Error("body is not the embedded spec")
	}
	assertMatchesSpec(t, req, rec.Result())

	_, rec = serve(t, http.MethodPost, "/api/v1/openapi.yaml", jsonCT, strings.NewReader("{}"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want \"GET, HEAD\"", got)
	}
}

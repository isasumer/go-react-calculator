package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
)

var update = flag.Bool("update", false, "rewrite testdata/*.json golden files")

// exampleRequestID stands in for the ID the request-ID middleware (B1-04)
// will set, so golden files and docs/errors.md show a complete problem.
const exampleRequestID = "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"

const jsonCT = "application/json"

// serve runs one request through Routes with an X-Request-ID already set on
// the response, as the middleware chain will do.
func serve(t *testing.T, method, path, contentType string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", exampleRequestID)
	NewHandler(calc.NewRegistry(), nil).Routes().ServeHTTP(rec, req)
	return rec
}

// assertGolden compares the indented JSON body with testdata/<name>.json.
func assertGolden(t *testing.T, name string, body []byte) {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, body)
	}
	buf.WriteByte('\n')
	path := filepath.Join("testdata", name+".json")
	if *update {
		if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run go test -update to create): %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("body mismatch with %s\n got: %s\nwant: %s", path, buf.Bytes(), want)
	}
}

func TestRoutes(t *testing.T) {
	const operationsBody = `{"operations":[` +
		`{"name":"add","symbol":"+","arity":2},` +
		`{"name":"subtract","symbol":"−","arity":2},` +
		`{"name":"multiply","symbol":"×","arity":2},` +
		`{"name":"divide","symbol":"÷","arity":2},` +
		`{"name":"power","symbol":"^","arity":2},` +
		`{"name":"sqrt","symbol":"√","arity":1},` +
		`{"name":"percent","symbol":"%","arity":2}]}`

	tests := []struct {
		name        string
		method      string // default POST
		path        string // default /api/v1/calculate
		contentType string // default application/json; "-" sends none
		body        string
		bodyReader  io.Reader // overrides body

		wantStatus int
		wantBody   string // exact body for successful responses
		wantCode   Code
		golden     string // testdata/<golden>.json for problem responses
		wantAllow  string
	}{
		// PLAN §1.4, one case per row.
		{
			name: "divide", body: `{"operation":"divide","a":10,"b":4}`,
			wantStatus: 200, wantBody: `{"operation":"divide","a":10,"b":4,"result":2.5}`,
		},
		{
			name: "sqrt", body: `{"operation":"sqrt","a":16}`,
			wantStatus: 200, wantBody: `{"operation":"sqrt","a":16,"result":4}`,
		},
		{
			name: "percent", body: `{"operation":"percent","a":200,"b":15}`,
			wantStatus: 200, wantBody: `{"operation":"percent","a":200,"b":15,"result":30}`,
		},
		{
			name: "division by zero", body: `{"operation":"divide","a":1,"b":0}`,
			wantStatus: 422, wantCode: CodeDivisionByZero, golden: "division_by_zero",
		},
		{
			name: "sqrt of negative", body: `{"operation":"sqrt","a":-1}`,
			wantStatus: 422, wantCode: CodeDomainError, golden: "domain_error",
		},
		{
			name: "overflow", body: `{"operation":"power","a":1e308,"b":2}`,
			wantStatus: 422, wantCode: CodeResultNotFinite, golden: "result_not_finite",
		},
		{
			name: "unknown operation", body: `{"operation":"modulo","a":1,"b":2}`,
			wantStatus: 422, wantCode: CodeUnsupportedOperation, golden: "unsupported_operation",
		},
		{
			name: "string-typed number", body: `{"operation":"add","a":"1","b":2}`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body",
		},
		{
			name: "missing b", body: `{"operation":"add","a":1}`,
			wantStatus: 400, wantCode: CodeValidationFailed, golden: "validation_failed",
		},
		{
			name: "list operations", method: http.MethodGet, path: "/api/v1/operations", contentType: "-",
			wantStatus: 200, wantBody: operationsBody,
		},

		// Other successful requests.
		{
			name: "content type with charset", contentType: "application/json; charset=utf-8",
			body:       `{"operation":"add","a":0.1,"b":0.2}`,
			wantStatus: 200, wantBody: `{"operation":"add","a":0.1,"b":0.2,"result":0.30000000000000004}`,
		},
		{
			name: "zero operands are present", body: `{"operation":"multiply","a":0,"b":0}`,
			wantStatus: 200, wantBody: `{"operation":"multiply","a":0,"b":0,"result":0}`,
		},
		{
			name: "null b on unary is omitted", body: `{"operation":"sqrt","a":9,"b":null}`,
			wantStatus: 200, wantBody: `{"operation":"sqrt","a":9,"result":3}`,
		},
		{
			name: "HEAD operations", method: http.MethodHead, path: "/api/v1/operations", contentType: "-",
			wantStatus: 200, wantBody: operationsBody,
		},

		// Semantic errors.
		{
			name: "b supplied to unary operation", body: `{"operation":"sqrt","a":4,"b":2}`,
			wantStatus: 422, wantCode: CodeUnexpectedOperand, golden: "unexpected_operand",
		},
		{
			name: "zero to a negative power", body: `{"operation":"power","a":0,"b":-1}`,
			wantStatus: 422, wantCode: CodeDivisionByZero, golden: "division_by_zero_power",
		},
		{
			name: "every field missing", body: `{}`,
			wantStatus: 400, wantCode: CodeValidationFailed, golden: "validation_failed_all",
		},
		{
			name: "unknown operation and missing a", body: `{"operation":"modulo"}`,
			wantStatus: 400, wantCode: CodeValidationFailed, golden: "validation_failed_unknown_op",
		},

		// Malformed bodies.
		{
			name: "unknown field", body: `{"operation":"add","a":1,"b":2,"c":3}`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_unknown_field",
		},
		{
			name: "trailing garbage", body: `{"operation":"add","a":1,"b":2} x`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_trailing",
		},
		{
			name: "two JSON values", body: `{"operation":"add","a":1,"b":2}{}`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_trailing",
		},
		{
			name: "empty body", body: ``,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_empty",
		},
		{
			name: "array body", body: `[1,2]`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_array",
		},
		{
			name: "null body", body: `null`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_null",
		},
		{
			name: "syntax error", body: `{"operation":"add","a":}`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_syntax",
		},
		{
			name: "truncated JSON", body: `{"operation":"add","a":1`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_truncated",
		},
		{
			name: "number out of float64 range", body: `{"operation":"add","a":1e400,"b":1}`,
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_out_of_range",
		},
		{
			name: "unreadable body", bodyReader: iotest.ErrReader(errors.New("connection reset")),
			wantStatus: 400, wantCode: CodeInvalidBody, golden: "invalid_body_unreadable",
		},

		// Transport errors.
		{
			name: "missing content type", contentType: "-", body: `{"operation":"add","a":1,"b":2}`,
			wantStatus: 415, wantCode: CodeUnsupportedMediaType, golden: "unsupported_media_type",
		},
		{
			name: "text/plain", contentType: "text/plain", body: `{"operation":"add","a":1,"b":2}`,
			wantStatus: 415, wantCode: CodeUnsupportedMediaType, golden: "unsupported_media_type",
		},
		{
			name: "malformed content type", contentType: "application/json; charset", body: `{}`,
			wantStatus: 415, wantCode: CodeUnsupportedMediaType, golden: "unsupported_media_type",
		},
		{
			name: "body over 4 KiB", body: `{"operation":"add",` + strings.Repeat(" ", maxBodyBytes) + `"a":1,"b":2}`,
			wantStatus: 413, wantCode: CodePayloadTooLarge, golden: "payload_too_large",
		},
		{
			name: "trailing data over 4 KiB", body: `{"operation":"add","a":1,"b":2}` + strings.Repeat(" ", maxBodyBytes),
			wantStatus: 413, wantCode: CodePayloadTooLarge, golden: "payload_too_large",
		},
		{
			name: "GET calculate", method: http.MethodGet, contentType: "-",
			wantStatus: 405, wantCode: CodeMethodNotAllowed, golden: "method_not_allowed", wantAllow: "POST",
		},
		{
			name: "DELETE operations", method: http.MethodDelete, path: "/api/v1/operations", contentType: "-",
			wantStatus: 405, wantCode: CodeMethodNotAllowed, golden: "method_not_allowed_operations",
			wantAllow: "GET, HEAD",
		},
		{
			name: "unknown route", method: http.MethodGet, path: "/api/v1/nope", contentType: "-",
			wantStatus: 404, wantCode: CodeNotFound, golden: "not_found",
		},
		{
			name: "trailing slash is a different route", path: "/api/v1/calculate/", body: `{}`,
			wantStatus: 404, wantCode: CodeNotFound, golden: "not_found_trailing_slash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, path, ct := tt.method, tt.path, tt.contentType
			if method == "" {
				method = http.MethodPost
			}
			if path == "" {
				path = "/api/v1/calculate"
			}
			switch ct {
			case "":
				ct = jsonCT
			case "-":
				ct = ""
			}
			body := tt.bodyReader
			if body == nil {
				body = strings.NewReader(tt.body)
			}

			rec := serve(t, method, path, ct, body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
			}
			if got, want := rec.Header().Get("Content-Length"), strconv.Itoa(rec.Body.Len()); got != want {
				t.Errorf("Content-Length = %s, want %s", got, want)
			}

			if tt.golden == "" {
				if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
					t.Errorf("Content-Type = %q", got)
				}
				if got := rec.Body.String(); got != tt.wantBody {
					t.Errorf("body\n got: %s\nwant: %s", got, tt.wantBody)
				}
				return
			}

			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q", got)
			}
			var p Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if p.Code != tt.wantCode || p.Status != tt.wantStatus || p.Instance != path {
				t.Errorf("code/status/instance = %s/%d/%s, want %s/%d/%s",
					p.Code, p.Status, p.Instance, tt.wantCode, tt.wantStatus, path)
			}
			if strings.Contains(p.Detail, "json:") || strings.Contains(p.Detail, "Go ") {
				t.Errorf("detail leaks Go error text: %q", p.Detail)
			}
			assertGolden(t, tt.golden, rec.Body.Bytes())
		})
	}
}

func TestValidate_NonFiniteOperands(t *testing.T) {
	// JSON cannot carry NaN or ±Inf, so exercise the guard directly.
	h := NewHandler(calc.NewRegistry(), nil)
	nan, inf := nanValue(), infValue()
	_, err := h.validate(&CalculateRequest{Operation: "add", A: &nan, B: &inf})

	p, ok := errors.AsType[*Problem](err)
	if !ok {
		t.Fatalf("err = %v, want *Problem", err)
	}
	want := []FieldError{
		{Field: "a", Message: "must be a finite number"},
		{Field: "b", Message: "must be a finite number"},
	}
	if p.Code != CodeValidationFailed || len(p.Errors) != 2 || p.Errors[0] != want[0] || p.Errors[1] != want[1] {
		t.Errorf("problem = %+v, want VALIDATION_FAILED with %+v", p, want)
	}
}

func TestWriteJSON_EncodeErrorIsProblem(t *testing.T) {
	var logs bytes.Buffer
	h := NewHandler(calc.NewRegistry(), newTestLogger(&logs))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/operations", http.NoBody)
	rec := httptest.NewRecorder()

	h.writeJSON(rec, req, http.StatusOK, nanValue())

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if strings.Contains(rec.Body.String(), "NaN") {
		t.Errorf("body leaks the encode error: %s", rec.Body)
	}
	if !strings.Contains(logs.String(), "encode response") {
		t.Errorf("encode error not logged: %s", logs.String())
	}
}

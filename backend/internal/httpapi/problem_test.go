package httpapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
)

// allCodes lists every code; each has a canonical example in
// testdata/<lower-case code>.json and in docs/errors.md.
var allCodes = []Code{
	CodeInvalidBody,
	CodeValidationFailed,
	CodeUnsupportedOperation,
	CodeUnexpectedOperand,
	CodeDivisionByZero,
	CodeDomainError,
	CodeResultNotFinite,
	CodeUnsupportedMediaType,
	CodePayloadTooLarge,
	CodeNotFound,
	CodeMethodNotAllowed,
	CodeInternal,
	CodeNotReady,
}

func TestNewProblem(t *testing.T) {
	tests := []struct {
		code       Code
		wantStatus int
		wantTitle  string
	}{
		{CodeInvalidBody, 400, "Invalid request body"},
		{CodeValidationFailed, 400, "Validation failed"},
		{CodeUnsupportedOperation, 422, "Unsupported operation"},
		{CodeUnexpectedOperand, 422, "Unexpected operand"},
		{CodeDivisionByZero, 422, "Division by zero"},
		{CodeDomainError, 422, "Result is not a real number"},
		{CodeResultNotFinite, 422, "Result is not finite"},
		{CodeUnsupportedMediaType, 415, "Unsupported media type"},
		{CodePayloadTooLarge, 413, "Payload too large"},
		{CodeNotFound, 404, "Not found"},
		{CodeMethodNotAllowed, 405, "Method not allowed"},
		{CodeInternal, 500, "Internal server error"},
		{CodeNotReady, 503, "Not ready"},
		{Code("SOMETHING_NEW"), 500, "Internal server error"},
	}
	if len(tests) != len(allCodes)+1 {
		t.Fatalf("table covers %d codes, allCodes has %d", len(tests)-1, len(allCodes))
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			p := NewProblem(tt.code, "detail", FieldError{Field: "a", Message: "m"})
			wantType := "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#" +
				strings.ToLower(string(tt.code))
			if p.Type != wantType || p.Status != tt.wantStatus || p.Title != tt.wantTitle ||
				p.Code != tt.code || p.Detail != "detail" || len(p.Errors) != 1 {
				t.Errorf("got %+v", p)
			}
			if got, want := p.Error(), string(tt.code)+": detail"; got != want {
				t.Errorf("Error() = %q, want %q", got, want)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name          string
		problem       *Problem
		requestID     string
		wantInstance  string
		wantRequestID string
		wantJSONKeys  []string
		absentKeys    []string
	}{
		{
			name:         "instance defaults to path, no request ID yet",
			problem:      NewProblem(CodeNotFound, "gone"),
			wantInstance: "/some/path",
			wantJSONKeys: []string{"type", "title", "status", "detail", "code", "instance"},
			absentKeys:   []string{"requestId", "errors"},
		},
		{
			name:          "request ID copied from response header",
			problem:       NewProblem(CodeValidationFailed, "bad", FieldError{Field: "a", Message: "is required"}),
			requestID:     "abc-123",
			wantInstance:  "/some/path",
			wantRequestID: "abc-123",
			wantJSONKeys:  []string{"requestId", "errors"},
		},
		{
			name:         "explicit instance kept",
			problem:      &Problem{Status: 400, Code: CodeInvalidBody, Instance: "/other"},
			wantInstance: "/other",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/some/path?q=1", http.NoBody)
			rec := httptest.NewRecorder()
			rec.Header().Set("Allow", "GET")
			if tt.requestID != "" {
				rec.Header().Set("X-Request-ID", tt.requestID)
			}
			before := *tt.problem

			Write(rec, req, tt.problem)

			if rec.Code != tt.problem.Status {
				t.Errorf("status = %d, want %d", rec.Code, tt.problem.Status)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q", ct)
			}
			if rec.Header().Get("Allow") != "GET" {
				t.Error("Write dropped a header set before it")
			}
			if tt.problem.Instance != before.Instance || tt.problem.RequestID != before.RequestID {
				t.Error("Write mutated the caller's problem")
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			var got Problem
			_ = json.Unmarshal(rec.Body.Bytes(), &got)
			if got.Instance != tt.wantInstance || got.RequestID != tt.wantRequestID {
				t.Errorf("instance/requestId = %q/%q, want %q/%q",
					got.Instance, got.RequestID, tt.wantInstance, tt.wantRequestID)
			}
			for _, k := range tt.wantJSONKeys {
				if _, ok := raw[k]; !ok {
					t.Errorf("key %q missing from %s", k, rec.Body)
				}
			}
			for _, k := range tt.absentKeys {
				if _, ok := raw[k]; ok {
					t.Errorf("key %q should be omitted from %s", k, rec.Body)
				}
			}
		})
	}
}

// TestInternalGolden produces the INTERNAL example, which no request to the
// real registry can trigger.
func TestInternalGolden(t *testing.T) {
	h := NewHandler(calc.NewRegistry(), nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/calculate", http.NoBody)
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", exampleRequestID)

	Write(rec, req, h.mapError(errors.New("boom")))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	assertGolden(t, "internal", rec.Body.Bytes())
}

// TestErrorCatalogueMatchesGoldenFiles keeps docs/errors.md honest: every
// code has a table row and a section whose JSON example is byte-identical to
// its golden file.
func TestErrorCatalogueMatchesGoldenFiles(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "errors.md"))
	if err != nil {
		t.Fatal(err)
	}
	examples := docExamples(t, doc)

	for _, code := range allCodes {
		t.Run(string(code), func(t *testing.T) {
			anchor := strings.ToLower(string(code))
			if !bytes.Contains(doc, []byte("](#"+anchor+")")) {
				t.Errorf("no table row linking #%s", anchor)
			}
			got, ok := examples[string(code)]
			if !ok {
				t.Fatalf("no \"### %s\" section with a json example", code)
			}
			want, err := os.ReadFile(filepath.Join("testdata", anchor+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("docs example differs from testdata/%s.json\ndocs:\n%s\ngolden:\n%s", anchor, got, want)
			}
		})
	}
}

// docExamples maps each "### <heading>" to the first ```json block under it.
func docExamples(t *testing.T, doc []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	var heading string
	var block *strings.Builder
	sc := bufio.NewScanner(bytes.NewReader(doc))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case block != nil && line == "```":
			if _, seen := out[heading]; !seen {
				out[heading] = block.String()
			}
			block = nil
		case block != nil:
			block.WriteString(line + "\n")
		case strings.HasPrefix(line, "### "):
			heading = strings.TrimPrefix(line, "### ")
		case line == "```json" && heading != "":
			block = &strings.Builder{}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

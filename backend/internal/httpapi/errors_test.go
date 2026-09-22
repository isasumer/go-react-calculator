package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strings"
	"testing"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
)

func newTestLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, nil))
}

func nanValue() float64 { return math.NaN() }
func infValue() float64 { return math.Inf(1) }

func TestMapError(t *testing.T) {
	validation := NewProblem(CodeValidationFailed, "missing or invalid fields: a",
		FieldError{Field: "a", Message: "is required"})

	tests := []struct {
		name       string
		err        error
		wantCode   Code
		wantStatus int
		wantDetail string
		wantErrors []FieldError
		wantLog    bool
	}{
		{
			name: "problem passes through", err: validation,
			wantCode: CodeValidationFailed, wantStatus: 400,
			wantDetail: validation.Detail, wantErrors: validation.Errors,
		},
		{
			name: "wrapped problem passes through", err: fmt.Errorf("decode: %w", validation),
			wantCode: CodeValidationFailed, wantStatus: 400,
			wantDetail: validation.Detail, wantErrors: validation.Errors,
		},
		{
			name:     "divide by zero",
			err:      &evalError{op: calc.Divide, err: fmt.Errorf("divide: %w", calc.ErrDivisionByZero)},
			wantCode: CodeDivisionByZero, wantStatus: 422,
			wantDetail: `b must be non-zero for operation "divide"`,
			wantErrors: []FieldError{{Field: "b", Message: "must be non-zero"}},
		},
		{
			name:     "zero to a negative power",
			err:      &evalError{op: calc.Power, err: fmt.Errorf("power: %w", calc.ErrDivisionByZero)},
			wantCode: CodeDivisionByZero, wantStatus: 422,
			wantDetail: `operation "power" divides by zero for these operands`,
		},
		{
			name:     "domain",
			err:      &evalError{op: calc.Sqrt, err: fmt.Errorf("sqrt: %w", calc.ErrDomain)},
			wantCode: CodeDomainError, wantStatus: 422,
			wantDetail: `operation "sqrt" has no real-number result for these operands`,
		},
		{
			name:     "not finite",
			err:      &evalError{op: calc.Power, err: fmt.Errorf("power: result: %w", calc.ErrNotFinite)},
			wantCode: CodeResultNotFinite, wantStatus: 422,
			wantDetail: `the result of operation "power" is outside the 64-bit floating-point range`,
		},
		{
			name:     "unknown operation",
			err:      &evalError{op: "modulo", err: fmt.Errorf(`operation "modulo": %w`, calc.ErrUnknownOperation)},
			wantCode: CodeUnsupportedOperation, wantStatus: 422,
			wantDetail: `operation "modulo" is not supported; GET /api/v1/operations lists the supported operations`,
			wantErrors: []FieldError{{
				Field:   "operation",
				Message: "must be one of: add, subtract, multiply, divide, power, sqrt, percent",
			}},
		},
		{
			// validate rejects arity mismatches first; reaching calc with one is a bug.
			name:     "arity is internal",
			err:      &evalError{op: calc.Sqrt, err: fmt.Errorf("sqrt: %w", calc.ErrArity)},
			wantCode: CodeInternal, wantStatus: 500,
			wantDetail: "an unexpected error occurred", wantLog: true,
		},
		{
			name:     "unknown error is internal",
			err:      errors.New("secret database password is hunter2"),
			wantCode: CodeInternal, wantStatus: 500,
			wantDetail: "an unexpected error occurred", wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			h := NewHandler(calc.NewRegistry(), newTestLogger(&logs))

			p := h.mapError(tt.err)

			if p.Code != tt.wantCode || p.Status != tt.wantStatus || p.Detail != tt.wantDetail {
				t.Errorf("got %s/%d/%q, want %s/%d/%q",
					p.Code, p.Status, p.Detail, tt.wantCode, tt.wantStatus, tt.wantDetail)
			}
			if fmt.Sprint(p.Errors) != fmt.Sprint(tt.wantErrors) {
				t.Errorf("errors = %+v, want %+v", p.Errors, tt.wantErrors)
			}
			logged := logs.Len() > 0
			if logged != tt.wantLog {
				t.Errorf("logged = %v, want %v: %s", logged, tt.wantLog, logs.String())
			}
			if tt.wantLog && !strings.Contains(logs.String(), tt.err.Error()) {
				t.Errorf("log line lacks the cause: %s", logs.String())
			}
			if tt.wantLog && strings.Contains(p.Detail, tt.err.Error()) {
				t.Errorf("detail leaks the cause: %q", p.Detail)
			}
		})
	}
}

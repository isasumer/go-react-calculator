package httpapi

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
)

// evalError records which operation a calc error came from, so mapError can
// name it in the detail without parsing error text.
type evalError struct {
	op  calc.Operation
	err error
}

func (e *evalError) Error() string { return string(e.op) + ": " + e.err.Error() }
func (e *evalError) Unwrap() error { return e.err }

// mapError is the single translation from errors to problems. A *Problem is
// returned as is; calc sentinels map to 422 codes; anything else is logged and
// becomes a 500 whose detail does not reveal the cause.
func (h *Handler) mapError(err error) *Problem {
	if p, ok := errors.AsType[*Problem](err); ok {
		return p
	}
	var op calc.Operation
	if ee, ok := errors.AsType[*evalError](err); ok {
		op = ee.op
	}

	switch {
	case errors.Is(err, calc.ErrDivisionByZero):
		if op == calc.Divide {
			return NewProblem(CodeDivisionByZero,
				fmt.Sprintf("b must be non-zero for operation %q", op),
				FieldError{Field: "b", Message: "must be non-zero"})
		}
		return NewProblem(CodeDivisionByZero,
			fmt.Sprintf("operation %q divides by zero for these operands", op))
	case errors.Is(err, calc.ErrDomain):
		return NewProblem(CodeDomainError,
			fmt.Sprintf("operation %q has no real-number result for these operands", op))
	case errors.Is(err, calc.ErrNotFinite):
		return NewProblem(CodeResultNotFinite,
			fmt.Sprintf("the result of operation %q is outside the 64-bit floating-point range", op))
	case errors.Is(err, calc.ErrUnknownOperation):
		return h.unsupportedOperation(op)
	}

	h.log.Error("unexpected error", slog.Any("error", err))
	return NewProblem(CodeInternal, "an unexpected error occurred")
}

package httpapi

import (
	"fmt"
	"math"
	"strings"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
)

// validate checks req against the registry and returns the operation's spec.
// Missing or non-finite fields are a 400 listing every bad field; an unknown
// operation or an operand the operation does not take is a 422.
func (h *Handler) validate(req *CalculateRequest) (calc.Spec, error) {
	var errs []FieldError
	if req.Operation == "" {
		errs = append(errs, FieldError{Field: "operation", Message: "is required"})
	}
	if req.A == nil {
		errs = append(errs, FieldError{Field: "a", Message: "is required"})
	} else if !isFinite(*req.A) {
		errs = append(errs, FieldError{Field: "a", Message: "must be a finite number"})
	}
	if req.B != nil && !isFinite(*req.B) {
		errs = append(errs, FieldError{Field: "b", Message: "must be a finite number"})
	}

	spec, known := h.calc.Lookup(calc.Operation(req.Operation))
	if known && spec.Arity == 2 && req.B == nil {
		errs = append(errs, FieldError{
			Field:   "b",
			Message: fmt.Sprintf("is required for operation %q", spec.Name),
		})
	}
	if len(errs) > 0 {
		fields := make([]string, len(errs))
		for i, e := range errs {
			fields[i] = e.Field
		}
		return calc.Spec{}, NewProblem(CodeValidationFailed,
			"missing or invalid fields: "+strings.Join(fields, ", "), errs...)
	}

	if !known {
		return calc.Spec{}, h.unsupportedOperation(calc.Operation(req.Operation))
	}
	if spec.Arity == 1 && req.B != nil {
		return calc.Spec{}, NewProblem(CodeUnexpectedOperand,
			fmt.Sprintf("operation %q takes a single operand; omit b", spec.Name),
			FieldError{Field: "b", Message: fmt.Sprintf("must be omitted for operation %q", spec.Name)})
	}
	return spec, nil
}

func (h *Handler) unsupportedOperation(op calc.Operation) *Problem {
	specs := h.calc.List()
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = string(s.Name)
	}
	return NewProblem(CodeUnsupportedOperation,
		fmt.Sprintf("operation %q is not supported; GET /api/v1/operations lists the supported operations", op),
		FieldError{Field: "operation", Message: "must be one of: " + strings.Join(names, ", ")})
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

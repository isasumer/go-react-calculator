package calc

import "errors"

// Sentinel errors. Errors returned by [Registry.Evaluate] wrap one of these
// with the operation name; match them with errors.Is.
var (
	// ErrUnknownOperation means the operation is not in the registry.
	ErrUnknownOperation = errors.New("unknown operation")
	// ErrArity means operand b was missing for a binary operation or
	// present for a unary one.
	ErrArity = errors.New("wrong number of operands")
	// ErrDivisionByZero means the operation divides by zero.
	ErrDivisionByZero = errors.New("division by zero")
	// ErrDomain means the result is not a real number, for example the
	// square root of a negative number.
	ErrDomain = errors.New("result is not a real number")
	// ErrNotFinite means an operand or the result is NaN or ±Inf.
	ErrNotFinite = errors.New("value is not finite")
)

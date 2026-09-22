package calc

import (
	"fmt"
	"math"
)

// Operation is the stable machine name of an arithmetic operation.
type Operation string

// Supported operations, in the order [Registry.List] returns them.
const (
	Add      Operation = "add"
	Subtract Operation = "subtract"
	Multiply Operation = "multiply"
	Divide   Operation = "divide"
	Power    Operation = "power"
	Sqrt     Operation = "sqrt"
	Percent  Operation = "percent"
)

// Spec describes one operation. Unary operations (Arity 1) ignore b.
type Spec struct {
	Name   Operation
	Symbol string
	Arity  int
	fn     func(a, b float64) (float64, error)
}

// Registry is the ordered, read-only set of supported operations. Build it
// with [NewRegistry]; it is safe for concurrent use.
type Registry struct {
	specs []Spec
	index map[Operation]int
}

// NewRegistry returns a registry holding every supported operation.
func NewRegistry() *Registry {
	specs := []Spec{
		{Name: Add, Symbol: "+", Arity: 2, fn: add},
		{Name: Subtract, Symbol: "−", Arity: 2, fn: subtract},
		{Name: Multiply, Symbol: "×", Arity: 2, fn: multiply},
		{Name: Divide, Symbol: "÷", Arity: 2, fn: divide},
		{Name: Power, Symbol: "^", Arity: 2, fn: power},
		{Name: Sqrt, Symbol: "√", Arity: 1, fn: sqrt},
		{Name: Percent, Symbol: "%", Arity: 2, fn: percent},
	}
	index := make(map[Operation]int, len(specs))
	for i, s := range specs {
		index[s.Name] = i
	}
	return &Registry{specs: specs, index: index}
}

// Lookup returns the spec for name and whether it exists.
func (r *Registry) Lookup(name Operation) (Spec, bool) {
	i, ok := r.index[name]
	if !ok {
		return Spec{}, false
	}
	return r.specs[i], true
}

// List returns every spec in registry order. The slice is a copy.
func (r *Registry) List() []Spec {
	return append([]Spec(nil), r.specs...)
}

// Evaluate applies op to a and, for binary operations, *b. b must be nil for
// unary operations and non-nil for binary ones. On success the result is
// finite and never negative zero; otherwise the error wraps one of the
// package's sentinel errors.
func (r *Registry) Evaluate(op Operation, a float64, b *float64) (float64, error) {
	spec, ok := r.Lookup(op)
	if !ok {
		return 0, fmt.Errorf("operation %q: %w", op, ErrUnknownOperation)
	}
	switch {
	case spec.Arity == 2 && b == nil:
		return 0, fmt.Errorf("%s: operand b is required: %w", op, ErrArity)
	case spec.Arity == 1 && b != nil:
		return 0, fmt.Errorf("%s: operand b is not allowed: %w", op, ErrArity)
	}
	if !isFinite(a) {
		return 0, fmt.Errorf("%s: operand a: %w", op, ErrNotFinite)
	}
	var bv float64
	if b != nil {
		bv = *b
		if !isFinite(bv) {
			return 0, fmt.Errorf("%s: operand b: %w", op, ErrNotFinite)
		}
	}

	res, err := spec.fn(a, bv)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	if !isFinite(res) {
		return 0, fmt.Errorf("%s: result: %w", op, ErrNotFinite)
	}
	if res == 0 {
		res = 0 // normalises -0
	}
	return res, nil
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func add(a, b float64) (float64, error)      { return a + b, nil }
func subtract(a, b float64) (float64, error) { return a - b, nil }
func multiply(a, b float64) (float64, error) { return a * b, nil }

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	return a / b, nil
}

// power follows math.Pow, so 0^0 = 1. 0 raised to a negative power is 1/0.
// A NaN from finite operands means a negative base with a non-integer
// exponent, which has no real result.
func power(a, b float64) (float64, error) {
	if a == 0 && b < 0 {
		return 0, ErrDivisionByZero
	}
	res := math.Pow(a, b)
	if math.IsNaN(res) {
		return 0, ErrDomain
	}
	return res, nil
}

func sqrt(a, _ float64) (float64, error) {
	if a < 0 {
		return 0, ErrDomain
	}
	return math.Sqrt(a), nil
}

// percent is "b percent of a". a*b/100 keeps whole-number cases exact
// (15 % of 200 is 30, not 30.000000000000004); when a*b alone overflows,
// a*(b/100) still gives the finite answer.
func percent(a, b float64) (float64, error) {
	res := a * b / 100
	if math.IsInf(res, 0) {
		res = a * (b / 100)
	}
	return res, nil
}

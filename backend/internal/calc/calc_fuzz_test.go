package calc

import (
	"errors"
	"math"
	"testing"
)

var sentinels = []error{ErrUnknownOperation, ErrArity, ErrDivisionByZero, ErrDomain, ErrNotFinite}

// FuzzEvaluate checks the package invariants for arbitrary input: Evaluate
// never panics, a nil error comes with a finite, non-negative-zero result,
// and any error wraps exactly one sentinel.
func FuzzEvaluate(f *testing.F) {
	for _, s := range NewRegistry().List() {
		f.Add(string(s.Name), 1.0, 2.0, true)
		f.Add(string(s.Name), -1.0, 0.0, true)
		f.Add(string(s.Name), 4.0, 0.0, false)
	}
	f.Add("power", -8.0, 1.0/3, true)
	f.Add("power", 0.0, -1.0, true)
	f.Add("add", math.MaxFloat64, math.MaxFloat64, true)
	f.Add("multiply", math.Copysign(0, -1), 1.0, true)
	f.Add("divide", math.NaN(), math.Inf(1), true)
	f.Add("modulo", 1.0, 2.0, true)

	reg := NewRegistry()
	f.Fuzz(func(t *testing.T, op string, a, b float64, hasB bool) {
		var bp *float64
		if hasB {
			bp = &b
		}
		got, err := reg.Evaluate(Operation(op), a, bp)
		if err == nil {
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("Evaluate(%q, %v, %v) = %v with nil error; want a finite result", op, a, b, got)
			}
			if got == 0 && math.Signbit(got) {
				t.Fatalf("Evaluate(%q, %v, %v) = -0; want -0 normalised to 0", op, a, b)
			}
			return
		}
		matched := 0
		for _, s := range sentinels {
			if errors.Is(err, s) {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("Evaluate(%q, %v, %v) error %v matches %d sentinels, want exactly 1", op, a, b, err, matched)
		}
	})
}

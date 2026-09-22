package calc

import (
	"errors"
	"math"
	"testing"
)

func ptr(f float64) *float64 { return &f }

func TestEvaluate(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	tests := []struct {
		name    string
		op      Operation
		a       float64
		b       *float64
		want    float64
		wantErr error
	}{
		// Happy paths, one per operation.
		{name: "add", op: Add, a: 2, b: ptr(3), want: 5},
		{name: "subtract", op: Subtract, a: 2, b: ptr(3), want: -1},
		{name: "multiply", op: Multiply, a: -4, b: ptr(2.5), want: -10},
		{name: "divide", op: Divide, a: 10, b: ptr(4), want: 2.5},
		{name: "power", op: Power, a: 2, b: ptr(10), want: 1024},
		{name: "power negative base integer exponent", op: Power, a: -2, b: ptr(3), want: -8},
		{name: "power fractional exponent", op: Power, a: 9, b: ptr(0.5), want: 3},
		{name: "sqrt", op: Sqrt, a: 16, want: 4},
		{name: "percent 15 of 200", op: Percent, a: 200, b: ptr(15), want: 30},

		// Boundaries.
		{name: "0.1 plus 0.2 is not 0.3", op: Add, a: 0.1, b: ptr(0.2), want: 0.30000000000000004},
		{name: "sqrt of 0", op: Sqrt, a: 0, want: 0},
		{name: "0^0 is 1", op: Power, a: 0, b: ptr(0), want: 1},
		{name: "negative percent", op: Percent, a: 200, b: ptr(-15), want: -30},
		{name: "percent of negative", op: Percent, a: -200, b: ptr(15), want: -30},
		{name: "percent where a*b overflows but result is finite", op: Percent, a: math.MaxFloat64, b: ptr(50), want: math.MaxFloat64 / 2},
		{name: "MaxFloat64 plus small value rounds to MaxFloat64", op: Add, a: math.MaxFloat64, b: ptr(1), want: math.MaxFloat64},
		{name: "underflow to 0", op: Multiply, a: 1e-308, b: ptr(1e-308), want: 0},
		{name: "divide 0 by non-zero", op: Divide, a: 0, b: ptr(5), want: 0},

		// Negative zero is normalised.
		{name: "-0 plus -0", op: Add, a: math.Copysign(0, -1), b: ptr(math.Copysign(0, -1)), want: 0},
		{name: "-1 times 0", op: Multiply, a: -1, b: ptr(0), want: 0},
		{name: "0 divided by negative", op: Divide, a: 0, b: ptr(-5), want: 0},
		{name: "sqrt of -0", op: Sqrt, a: math.Copysign(0, -1), want: 0},
		{name: "negative percent of 0", op: Percent, a: 0, b: ptr(-15), want: 0},

		// ErrUnknownOperation.
		{name: "unknown operation", op: "modulo", a: 1, b: ptr(2), wantErr: ErrUnknownOperation},
		{name: "empty operation", op: "", a: 1, b: ptr(2), wantErr: ErrUnknownOperation},
		{name: "operation names are case sensitive", op: "ADD", a: 1, b: ptr(2), wantErr: ErrUnknownOperation},

		// ErrArity.
		{name: "binary operation missing b", op: Add, a: 1, wantErr: ErrArity},
		{name: "percent missing b", op: Percent, a: 1, wantErr: ErrArity},
		{name: "unary operation given b", op: Sqrt, a: 4, b: ptr(2), wantErr: ErrArity},

		// ErrDivisionByZero.
		{name: "divide by 0", op: Divide, a: 1, b: ptr(0), wantErr: ErrDivisionByZero},
		{name: "divide by -0", op: Divide, a: 1, b: ptr(math.Copysign(0, -1)), wantErr: ErrDivisionByZero},
		{name: "0/0", op: Divide, a: 0, b: ptr(0), wantErr: ErrDivisionByZero},
		{name: "0 to a negative power", op: Power, a: 0, b: ptr(-1), wantErr: ErrDivisionByZero},

		// ErrDomain.
		{name: "sqrt of negative", op: Sqrt, a: -1, wantErr: ErrDomain},
		{name: "cube root of negative via power", op: Power, a: -8, b: ptr(1.0 / 3), wantErr: ErrDomain},
		{name: "negative base fractional exponent", op: Power, a: -2, b: ptr(0.5), wantErr: ErrDomain},

		// ErrNotFinite: results.
		{name: "MaxFloat64 overflow on add", op: Add, a: math.MaxFloat64, b: ptr(math.MaxFloat64), wantErr: ErrNotFinite},
		{name: "MaxFloat64 overflow on subtract", op: Subtract, a: -math.MaxFloat64, b: ptr(math.MaxFloat64), wantErr: ErrNotFinite},
		{name: "MaxFloat64 overflow on multiply", op: Multiply, a: math.MaxFloat64, b: ptr(2), wantErr: ErrNotFinite},
		{name: "divide overflow", op: Divide, a: math.MaxFloat64, b: ptr(0.5), wantErr: ErrNotFinite},
		{name: "power overflow", op: Power, a: 1e308, b: ptr(2), wantErr: ErrNotFinite},
		{name: "power overflow to -Inf", op: Power, a: -1e308, b: ptr(3), wantErr: ErrNotFinite},
		{name: "percent overflow", op: Percent, a: math.MaxFloat64, b: ptr(200), wantErr: ErrNotFinite},

		// ErrNotFinite: operands.
		{name: "a is NaN", op: Add, a: math.NaN(), b: ptr(1), wantErr: ErrNotFinite},
		{name: "a is +Inf", op: Sqrt, a: math.Inf(1), wantErr: ErrNotFinite},
		{name: "b is -Inf", op: Multiply, a: 1, b: ptr(math.Inf(-1)), wantErr: ErrNotFinite},
		{name: "b is NaN", op: Divide, a: 1, b: ptr(math.NaN()), wantErr: ErrNotFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := reg.Evaluate(tt.op, tt.a, tt.b)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Evaluate(%q, %v, %v) error = %v, want errors.Is %v", tt.op, tt.a, deref(tt.b), err, tt.wantErr)
				}
				if got != 0 {
					t.Errorf("Evaluate(%q, ...) = %v on error, want 0", tt.op, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Evaluate(%q, %v, %v) unexpected error: %v", tt.op, tt.a, deref(tt.b), err)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Evaluate(%q, %v, %v) = %v, want %v", tt.op, tt.a, deref(tt.b), got, tt.want)
			}
		})
	}
}

func deref(b *float64) any {
	if b == nil {
		return "<nil>"
	}
	return *b
}

func TestEvaluateErrorContext(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	tests := []struct {
		name string
		op   Operation
		a    float64
		b    *float64
		msg  string
	}{
		{name: "unknown", op: "modulo", a: 1, b: ptr(2), msg: `operation "modulo": unknown operation`},
		{name: "missing b", op: Divide, a: 1, msg: "divide: operand b is required: wrong number of operands"},
		{name: "extra b", op: Sqrt, a: 1, b: ptr(2), msg: "sqrt: operand b is not allowed: wrong number of operands"},
		{name: "a not finite", op: Add, a: math.NaN(), b: ptr(1), msg: "add: operand a: value is not finite"},
		{name: "b not finite", op: Add, a: 1, b: ptr(math.Inf(1)), msg: "add: operand b: value is not finite"},
		{name: "division by zero", op: Divide, a: 1, b: ptr(0), msg: "divide: division by zero"},
		{name: "domain", op: Sqrt, a: -1, msg: "sqrt: result is not a real number"},
		{name: "result not finite", op: Power, a: 1e308, b: ptr(2), msg: "power: result: value is not finite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := reg.Evaluate(tt.op, tt.a, tt.b)
			if err == nil || err.Error() != tt.msg {
				t.Errorf("error = %v, want %q", err, tt.msg)
			}
		})
	}
}

func TestRegistryList(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	want := []struct {
		name   Operation
		symbol string
		arity  int
	}{
		{Add, "+", 2},
		{Subtract, "−", 2},
		{Multiply, "×", 2},
		{Divide, "÷", 2},
		{Power, "^", 2},
		{Sqrt, "√", 1},
		{Percent, "%", 2},
	}
	got := reg.List()
	if len(got) != len(want) {
		t.Fatalf("List() has %d specs, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Name != w.name || got[i].Symbol != w.symbol || got[i].Arity != w.arity {
			t.Errorf("List()[%d] = {%s %s %d}, want {%s %s %d}",
				i, got[i].Name, got[i].Symbol, got[i].Arity, w.name, w.symbol, w.arity)
		}
	}
}

func TestRegistryListReturnsCopy(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	specs := reg.List()
	specs[0].Name = "hijacked"
	specs[0].Arity = 99

	if got := reg.List()[0]; got.Name != Add || got.Arity != 2 {
		t.Errorf("mutating List() result changed the registry: %+v", got)
	}
	if _, ok := reg.Lookup("hijacked"); ok {
		t.Error(`Lookup("hijacked") found a spec after mutating a List() copy`)
	}
}

func TestRegistryLookup(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	for _, s := range reg.List() {
		got, ok := reg.Lookup(s.Name)
		if !ok || got.Name != s.Name || got.Symbol != s.Symbol || got.Arity != s.Arity {
			t.Errorf("Lookup(%q) = %+v, %v; want %+v, true", s.Name, got, ok, s)
		}
	}
	if got, ok := reg.Lookup("modulo"); ok || got.Name != "" || got.fn != nil {
		t.Errorf(`Lookup("modulo") = %+v, %v; want zero Spec, false`, got, ok)
	}
}

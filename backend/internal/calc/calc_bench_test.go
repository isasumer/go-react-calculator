package calc

import "testing"

func BenchmarkEvaluate(b *testing.B) {
	reg := NewRegistry()
	benches := []struct {
		name string
		op   Operation
		a    float64
		b    *float64
	}{
		{name: "add", op: Add, a: 1.5, b: ptr(2.25)},
		{name: "divide", op: Divide, a: 10, b: ptr(4)},
		{name: "power", op: Power, a: 1.0001, b: ptr(1000)},
		{name: "sqrt", op: Sqrt, a: 2},
		{name: "percent", op: Percent, a: 200, b: ptr(15)},
		{name: "error_division_by_zero", op: Divide, a: 1, b: ptr(0)},
	}
	for _, bb := range benches {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = reg.Evaluate(bb.op, bb.a, bb.b)
			}
		})
	}
}

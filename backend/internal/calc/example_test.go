package calc_test

import (
	"errors"
	"fmt"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
)

func ExampleRegistry_Evaluate() {
	reg := calc.NewRegistry()

	b := 15.0
	res, err := reg.Evaluate(calc.Percent, 200, &b)
	fmt.Println(res, err)

	res, err = reg.Evaluate(calc.Sqrt, 16, nil)
	fmt.Println(res, err)

	zero := 0.0
	_, err = reg.Evaluate(calc.Divide, 1, &zero)
	fmt.Println(errors.Is(err, calc.ErrDivisionByZero), err)

	// Output:
	// 30 <nil>
	// 4 <nil>
	// true divide: division by zero
}

func ExampleRegistry_List() {
	for _, s := range calc.NewRegistry().List() {
		fmt.Println(s.Name, s.Symbol, s.Arity)
	}

	// Output:
	// add + 2
	// subtract − 2
	// multiply × 2
	// divide ÷ 2
	// power ^ 2
	// sqrt √ 1
	// percent % 2
}

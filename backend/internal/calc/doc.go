// Package calc is the pure calculator domain: the operations registry,
// typed errors and IEEE-754 guards. It has no dependencies outside the
// standard library and knows nothing about HTTP.
//
// # Numeric model
//
// Operands and results are float64, because the API carries JSON numbers
// and JSON numbers are decoded as IEEE-754 doubles anyway (ADR-0003,
// docs/adr/0003-numeric-model.md). The package does not hide binary
// floating point: 0.1 + 0.2 evaluates to 0.30000000000000004, and rounding
// for display is a frontend concern (ADR-0007). Decimal arithmetic is a
// documented follow-up, not part of this model.
//
// Guards make every successful result a finite number:
//
//   - Operands must be finite; NaN and ±Inf are rejected with [ErrNotFinite].
//   - A result that overflows to ±Inf is rejected with [ErrNotFinite].
//   - Division by zero (including 0/0 and 0 raised to a negative power) is
//     rejected with [ErrDivisionByZero].
//   - Results that are not real numbers, such as the square root of a
//     negative number or (-8)^(1/3) through math.Pow, are rejected with
//     [ErrDomain].
//   - Negative zero is normalised to 0, so callers never see "-0".
//
// Definitions that are conventions rather than arithmetic facts:
//
//   - percent(a, b) = a*b/100, read as "b percent of a" (15 % of 200 = 30).
//   - power(0, 0) = 1, following math.Pow and the usual convention for
//     calculators and programming languages.
//
// Every error returned by [Registry.Evaluate] wraps exactly one of the
// sentinel errors with the operation name as context, so callers classify
// errors with errors.Is.
package calc

# ADR-0003: Use float64 with explicit non-finite guards as the numeric model

- **Status:** Accepted
- **Date:** 2026-09-22
- **Ticket:** #5

## Context
The calculator has to pick one number representation for operands and results. The API speaks JSON, and in practice JSON numbers are IEEE-754 doubles: JavaScript's `JSON.parse` and `Number` are doubles, and Go's `encoding/json` decodes into `float64` unless told otherwise. A decimal type in the backend would be precise only between the decoder and the encoder. The browser would still parse the response into a double. IEEE-754 also has values that are not numbers (NaN, ±Inf) and a negative zero; left unguarded, these reach the UI as `null`, `"NaN"` or `-0`, or make JSON encoding fail.

## Decision
`internal/calc` computes in `float64` and guarantees that every successful result is a finite number:

- Operands must be finite. NaN or ±Inf is rejected with `ErrNotFinite`.
- A result that overflows to ±Inf is rejected with `ErrNotFinite` (e.g. `MaxFloat64 + MaxFloat64`, `1e308 ^ 2`).
- Division by zero, including `0/0` and `0` raised to a negative power, is rejected with `ErrDivisionByZero`.
- A result that is not a real number is rejected with `ErrDomain`. Examples are the square root of a negative number and `math.Pow` returning NaN for a negative base with a non-integer exponent (`(-8)^(1/3)`).
- Negative zero is normalised to `0`.
- Conventions: `percent(a, b) = a*b/100` ("b percent of a", so 15 % of 200 = 30). `0^0 = 1`, as in `math.Pow`.

Every error wraps exactly one sentinel with the operation name, so the transport layer maps sentinels to problem codes with `errors.Is` (ADR-0005). Binary rounding is not hidden: the API returns `0.1 + 0.2 = 0.30000000000000004` unchanged. Rounding for display is the frontend's job: 12 significant digits, with exponent notation at the edges (ADR-0007). The UI therefore shows `0.3`, and the API stays truthful.

Decimal arithmetic is deferred to follow-up #31 (FU-02, a decimal mode selectable per request).

## Consequences
- The domain package is tiny, dependency-free and fast (~10–30 ns per evaluation, zero allocations on success; see `BenchmarkEvaluate`).
- The HTTP layer can trust any `nil`-error result to be JSON-encodable; the fuzz test (`FuzzEvaluate`) enforces this invariant.
- Results carry binary-rounding artefacts (`0.1 + 0.2`), and integers above 2^53 lose precision. This is acceptable for a general-purpose calculator. It would be wrong for money. A monetary context would use decimal arithmetic, both on the wire (numbers as strings) and in the domain. That is exactly what #31 describes.
- Two conventions (`percent`, `0^0`) are product decisions, not arithmetic facts. They are documented in the package doc and must stay in step with the UI's labels.
- Where `a*b` overflows but `a*b/100` would not, `percent` falls back to `a*(b/100)`, so large finite answers are still returned.

## Alternatives considered
- `math/big.Float` or a decimal library (e.g. shopspring/decimal) now → precise only between the decoder and the encoder while JSON stays numeric. It adds a dependency or complexity that this scope does not need. Deferred to #31.
- Return NaN/±Inf as strings (`"Infinity"`) → non-standard JSON that every client must special-case. A typed error with a stable code is clearer.
- Clamp overflow to ±MaxFloat64 → silently wrong answers.
- Treat `0^0` as a domain error → contradicts `math.Pow`, most calculators and most languages, and surprises users.

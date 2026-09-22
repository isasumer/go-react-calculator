# ADR-0007: Display precision — 12 significant digits, exponent past ±1e15 / 1e-6

- **Status:** Accepted
- **Date:** 2026-09-23
- **Ticket:** #15 (absorbs #17)

## Context

The engine's internal display string is `String(value)` — the shortest decimal that round-trips to
the exact `float64` the server returned (ADR-0003). That is correct for arithmetic and wrong for a
screen: `0.1 + 0.2` comes back as `0.30000000000000004`, and a plain result in the hundreds of
trillions or the millionths reads as an unbroken run of digits nobody can parse at a glance. A
calculator has to be honest about what a `float64` is — it is not going to claim `0.1 + 0.2` is
exactly `0.3` — but "honest" and "quotes seventeen digits of binary rounding error" are different
things. `docs/PLAN.md` §1.4 keeps the wire format untouched (the backend and the JSON contract carry
full `float64` precision); this ADR is about the client-side presentation layer only.

The two questions are how many digits to show, and what to do at the extremes where fixed notation
stops being readable. Twelve significant digits is the calculator-app convention (it is comfortably
inside the ~15–17 digits a `float64` can round-trip, so trimming to 12 loses only the noise, never a
digit the user typed or a digit the server actually computed to be significant at that magnitude).
Exponent notation is the ordinary escape hatch once fixed notation would need an unreasonable number
of digits either before or after the point — beyond roughly `1e15` on the large side, or below
`1e-6` in magnitude on the small side, chosen to keep the fixed-notation string itself readable
(no more than 15 digits before the point, no more than 6 leading zeros after it) rather than to
match any particular hardware limit.

## Decision

`src/lib/format-number.ts` owns display formatting, entirely separate from the engine:

- `formatResult(n, { maxSignificant = 12 })` — for a finished value (a result, or an operand already
  shown in the expression line). Rounds to `maxSignificant` significant digits via `toPrecision` and
  reparses, which is what trims `0.30000000000000004` down to `0.3` without inventing trailing
  zeros for a value that does not have them. Switches to exponent notation (`toExponential`, trimmed
  mantissa) once `|n| >= 1e15` or `0 < |n| < 1e-6`. Normalises `-0` to `"0"`. Throws a `RangeError`
  for non-finite input — `NaN`/`±Infinity` are guarded upstream (`RESULT_NOT_FINITE`, ADR-0003) and
  reaching this function is a bug, not a value to render.
- `formatEntry(raw)` — for a value still being typed. Adds `Intl.NumberFormat("en-US")` thousands
  grouping to the integer part only; a trailing `.` and every fractional digit already typed are the
  user's own keystrokes and are left untouched.
- `parseEntry(raw)` — the inverse of both, stripping any grouping commas before parsing.

Locale is `Intl.NumberFormat("en-US")`, explicit and fixed — a locale switch (grouping character,
decimal separator, digit shapes) is real product work, deferred to follow-up **#34**.

`Display` applies `formatEntry` while a value is being typed and `formatResult` once it is not; the
untouched raw string stays available in `title` either way, and `Calculator`/`Display` never feed a
formatted string back into the engine or the wire — only `CalcState.display` (a `String(value)`) and
the numbers in a `CalculateRequest` do that.

## Consequences

- `0.1 + 0.2` shows `0.3`, with `0.30000000000000004` a hover away — truthful without punishing the
  user for a representation they did not choose.
- A value the backend can legitimately return — very large, very small, or with float noise near the
  12th significant digit — never overflows the display column; it either fits in fixed notation or
  moves to exponent notation.
- Two more places to keep in sync with the engine: `formatResult`/`formatEntry` must stay read-only
  consumers of `CalcState.display`, never a second source of truth for what the entry *is*. A future
  change to the engine's own string representation (there is none planned) would need to review this
  file too.
- The locale is hard-coded. A user whose OS locale expects `1.234,56` sees US-style grouping instead,
  until #34.

## Alternatives considered

- **`toLocaleString` end to end** → simplest code, but its rounding and exponent behaviour is
  implementation-defined across engines; a golden-file test suite needs the deterministic output
  `toPrecision`/`toExponential` give.
- **Arbitrary-precision / decimal library** → solves float noise at the root, but the wire contract
  and the numeric model (ADR-0003) are already `float64`; formatting around it is proportionate to
  the assignment's scope, a full decimal mode is not.
- **No exponent notation, just truncate** → silently drops magnitude information (`1e20` truncated
  to 12 digits reads as a wildly wrong number, not a large one); exponent notation says what it is.

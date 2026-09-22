# ADR-0008: Evaluate only through the API, with immediate-execution semantics

- **Status:** Accepted
- **Date:** 2026-09-22
- **Ticket:** #13

## Context

The assignment is to consume a calculator service, so the interesting question is not "what is 2 + 3"
but "where does arithmetic happen, and what does a key mean". Two decisions follow from that.

The first is whether the client may ever compute. A frontend that folds `2 + 3` locally and only calls
the backend for the operations it finds inconvenient has two implementations of the same rules, and the
one the reviewer reads is not the one the user exercises.

The second is what an expression means. A pocket calculator uses *immediate execution*: each operator
key closes the operation before it, so `2 + 3 × 4` is 20. An expression calculator parses the whole
line and applies precedence, giving 14. Both are defensible; they are different products. The
assignment asks for four operations and an intuitive UI, not for a parser.

`percent` is the awkward key in every calculator. The backend defines it as one binary operation,
`percent(a, b)` = "b percent of a" (`percent(200, 15)` = 30, docs/PLAN.md §1.4), and the UI has to
decide what the single `%` key sends and what happens to the answer.

## Decision

**All arithmetic goes through `POST /api/v1/calculate`.** `src/lib/calculator-engine.ts` is a pure
state machine that returns `{ state, effect? }`; the only effect it can ask for is a calculate request.
It never adds, multiplies or divides — not even to compute a percentage or to negate a number, where
the sign is flipped on the *entry string*, not by arithmetic. `src/stores/useCalculatorStore.ts`
performs the effect through an injected evaluator and feeds the answer back in.

**Immediate execution, no precedence.** An operator key pressed while a second operand is on the
display evaluates the pending operation first and keeps its result as the new left operand, so
`2 + 3 × 4 =` is 20 and needs two round trips. Related consequences of the same model:

- `=` with no second operand reuses the first (`5 + =` is 10), the classic behaviour.
- `=` pressed again repeats the last evaluation on the running total: `12 + 7 = = =` gives 19, 26, 33.
  Only `=` records a repeat; chaining, `√` and `%` do not.
- A digit typed on a result starts a fresh calculation; an operator chains from it.
- Any key pressed in the error phase clears the error first; `AC` (Escape) clears everything.

**`√` and `%` act on the value on the display.** `√` takes the entry being typed, or the left operand
when nothing is being typed, and is sent as the arity-1 request `{operation:"sqrt", a}` — with no `b`
at all, which the backend would reject as `UNEXPECTED_OPERAND`.

**`%` with a pending binary operation is `percent(a, b)`, and its answer replaces the entry.** So
`200 + 15 %` shows 30 and the pending `+` then finishes with it: `200 + 15 % =` is 230. With no
pending operation, `%` is the plain "per cent of this number" key and is sent as `percent(x, 1)` =
x/100, because even that division belongs on the server.

## Consequences

- There is exactly one implementation of the arithmetic, and it is the one under test on the backend,
  including its edge cases: division by zero, `sqrt` of a negative number and non-finite results come
  back as problem+json and are shown through `messageForError` (ADR-0005).
- Every key that produces a number costs a round trip, and `2 + 3 × 4 =` costs two. The store exposes
  `pending` so the UI can show it, ignores every key except `AC` while a request is in flight, and
  discards an answer whose calculation has since been cleared.
- The calculator is offline-useless by construction. That is the assignment's shape, and the failure is
  legible: `NETWORK` maps to "Cannot reach the calculator service."
- `2 + 3 × 4 = 20` will surprise anyone expecting algebraic precedence. It is what a physical
  calculator does, it is what the expression line shows step by step (`5 ×` after the chain), and an
  expression-mode parser is a documented follow-up, not a silent difference.
- Our `%` is consistent rather than context-sensitive: it always means "b per cent of a". For `+` and
  `−` that is exactly what people expect (`200 + 15 %` = 230). For `×` and `÷` it differs from the
  pocket-calculator convention (see below), which is a real cost, accepted in exchange for one rule
  the user can learn and one backend operation.

## Alternatives considered

- Compute simple operations locally and call the API only for the rest → two implementations of the
  same rules, and the tested one is not the one the user runs.
- Expression mode with precedence and parentheses → a parser, an AST and a different UI; out of scope
  for a four-function calculator, and a follow-up if the product ever wants it.
- Debouncing or batching round trips → hides the very integration the assignment is about.
- Spreadsheet-style `%` (the value is always x/100, so `200 + 15 %` = 200.15) → predictable but almost
  never what someone pressing `%` on a calculator wants.
- Calculator-style context-sensitive `%` (`a ± b %` → b per cent of a, but `a × b %` → b/100, so
  `200 × 15 %` = 30) → matches most pocket calculators, but it makes the meaning of a key depend on the
  operator before it and it needs a second backend operation (or client-side division) for the `×`/`÷`
  case. Rejected for this scope; it is the first candidate if the UX is ever revisited.
- Keeping the pending operator in the state while a chained request is in flight (a `pendingOperator`
  field) → unnecessary: the phase a request lands in already says where its answer goes.

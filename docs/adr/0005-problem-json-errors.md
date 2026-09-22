# ADR-0005: Report every API error as RFC 9457 problem+json with a stable machine `code`

- **Status:** Accepted
- **Date:** 2026-09-22
- **Ticket:** #6

## Context
The frontend has to tell "the user typed something mathematically impossible" (show a friendly message) apart from "the client sent a broken request" (a bug) and "the server failed" (retry). It has to do that without parsing English text. RFC 9457 (Problem Details for HTTP APIs, which obsoletes RFC 7807) standardises an error body and media type. Its `type` URI identifies the problem kind, and it allows extension members. Go's `net/http` default error responses are `text/plain` with no structure.

## Decision
- Every error response, including 404 and 405, has the `Content-Type: application/problem+json` body `{type, title, status, detail, code, instance, requestId?, errors?}`. This is built in one place: `httpapi.NewProblem` and `httpapi.Write`.
- `code` is an extension member with a stable UPPER_SNAKE value that clients branch on. `type` is the URL of that code's section in [`docs/errors.md`](../errors.md). `title` is fixed per code. `detail` explains the occurrence and never contains Go error text. `instance` is the request path. `requestId` echoes `X-Request-ID` once middleware sets it (B1-04). `errors[]` lists `{field, message}` per offending field.
- Status codes: 400 for a malformed body (`INVALID_BODY`) or missing/invalid fields (`VALIDATION_FAILED`). 422 for a well-formed request with no answer (`UNSUPPORTED_OPERATION`, `UNEXPECTED_OPERAND`, `DIVISION_BY_ZERO`, `DOMAIN_ERROR`, `RESULT_NOT_FINITE`). 404, 405 (with `Allow`), 413 and 415 for transport problems; 429 is added with rate limiting. 500 `INTERNAL` has a generic detail, and the cause is logged server-side.
- Domain errors are mapped in one function, `mapError`, using `errors.Is` / `errors.As` on the `internal/calc` sentinels (ADR-0003). `calc` knows nothing about HTTP.
- `docs/errors.md` is the catalogue. Its example bodies are the handler tests' golden files, byte for byte, and a test fails when they diverge. Codes are never renamed once released.

## Consequences
- The frontend's `apiFetch` can parse one error shape and map `code` to UX (F2-01). Unknown codes fall back to `title`/`detail`.
- Adding an error means adding a `Code` constant, a status/title in `Code.meta`, a golden file and a catalogue section. The catalogue test enforces the last two.
- Middleware (recovery, rate limiting, timeouts) reuses `httpapi.Write`, so even a panic produces the same shape.
- `detail` strings are for humans and may be reworded; clients must not match on them.
- The split between 400 and 422 has to be applied consistently. The rule: 400 when the request does not match the schema, 422 when it matches but has no answer.

## Alternatives considered
- Plain `{"error": "message"}` → no machine-readable kind, so clients would parse English.
- Only HTTP status codes → 422 alone cannot tell division by zero from overflow.
- 200 with an `error` field → breaks HTTP semantics, caches and monitoring.
- `type` URNs (`urn:problem:division-by-zero`) → valid but not dereferenceable. A docs URL gives readers the explanation.
- Numeric codes → less readable in logs and UIs than stable names.

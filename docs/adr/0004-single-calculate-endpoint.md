# ADR-0004: Expose one `POST /api/v1/calculate` endpoint with an operation enum, plus a discovery endpoint

- **Status:** Accepted
- **Date:** 2026-09-22
- **Ticket:** #6

## Context
The calculator supports seven operations: add, subtract, multiply, divide, power, sqrt and percent. More may follow. The HTTP API can offer one resource per operation (`/add`, `/divide`, …) or one resource that takes the operation as data. The frontend also needs to know which operations exist and how many operands each takes. It should not hard-code a list that can drift from the backend's registry (`internal/calc`).

## Decision
- `POST /api/v1/calculate` takes `{"operation": "<name>", "a": <number>, "b": <number>?}` and returns `{"operation", "a", "b"?, "result"}`. `operation` is a closed enum, and its values are the names in the registry. `b` is required for binary operations and must be omitted for unary ones (`sqrt`).
- `GET /api/v1/operations` returns the registry in order: `{"operations": [{"name", "symbol", "arity"}, …]}`. The UI builds its keypad and validates arity from this list.
- The API is versioned in the path (`/api/v1`). The contract in PLAN §1.4 is frozen, and changes need a new ADR.
- Routing uses the standard library `http.ServeMux` with method patterns. A wrong method on a known path returns a 405 problem with an `Allow` header, and an unknown path returns a 404 problem (ADR-0005).

## Consequences
- A new operation is one registry entry in `internal/calc`. Routes, handlers, the OpenAPI paths and the frontend fetch code stay the same, and the discovery endpoint advertises it automatically.
- The URL surface is flat and small, so CORS, rate limiting, metrics labels and the nginx proxy rule each cover one path.
- Unknown operations are a runtime validation error (422 `UNSUPPORTED_OPERATION`), not a routing 404. The error names the valid values, which is more useful to a client than a 404.
- Per-operation HTTP caching or per-operation authorisation would be harder, but a calculator needs neither.
- Metrics and logs have to label by the `operation` field rather than by route. This is bounded because the enum is closed.

## Alternatives considered
- One route per operation (`POST /api/v1/divide`) → the route table grows with every operation, and the handlers repeat each other. The frontend still needs a discovery list.
- `GET /api/v1/calculate?op=divide&a=1&b=0` → numbers in query strings lose JSON's type checking. A computation is not a cacheable resource, and error bodies on GET are unusual.
- An expression endpoint (`{"expression": "1/0"}`) → needs a parser and precedence rules, which is a different product (immediate-execution semantics, ADR-0008). It is kept as a possible follow-up.
- A hard-coded operation list in the frontend → it drifts from the backend. `GET /operations` costs one small request.

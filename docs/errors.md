# Error catalogue

Every API error is an RFC 9457 `application/problem+json` document with a stable machine-readable `code` ([ADR-0005](adr/0005-problem-json-errors.md)). The `type` field of each problem is the URL of the section for its code on this page. This table was filled by #6 (B1-02) and is extended by later tickets; codes are never renamed once released.

Clients branch on `code`, never on `title` or `detail`: those are for humans and may be reworded.

| Code | HTTP | Meaning | Client behaviour | Example |
|---|---|---|---|---|
| `INVALID_BODY` | 400 | The body is not a single JSON object matching the request schema: empty, not JSON, truncated, an array/string/null, an unknown field, trailing data after the object, or a field of the wrong JSON type. `detail` names the field or byte offset; `errors[]` points at the field when there is one. | Bug in the client: fix the request. Do not retry unchanged. | [example](#invalid_body) |
| `VALIDATION_FAILED` | 400 | The body is well-formed but a required field is missing (`operation`, `a`, or `b` for a binary operation) or an operand is not a finite number. `errors[]` lists every offending field. | Highlight each field in `errors[]`; do not retry unchanged. | [example](#validation_failed) |
| `UNSUPPORTED_OPERATION` | 422 | `operation` is not in the registry. `errors[0].message` lists the supported names. | Refresh the operation list from `GET /api/v1/operations`; do not retry unchanged. | [example](#unsupported_operation) |
| `UNEXPECTED_OPERAND` | 422 | `b` was supplied to a unary operation such as `sqrt`. (`"b": null` counts as omitted.) | Omit `b` for operations whose `arity` is 1. | [example](#unexpected_operand) |
| `DIVISION_BY_ZERO` | 422 | The operation divides by zero: `divide` with `b = 0` (including `0/0`), or `power` with `a = 0` and a negative `b`. | Show a user-facing "cannot divide by zero" message. | [example](#division_by_zero) |
| `DOMAIN_ERROR` | 422 | The result is not a real number: the square root of a negative number, or `power` with a negative base and a non-integer exponent. | Show a user-facing "invalid input" message. | [example](#domain_error) |
| `RESULT_NOT_FINITE` | 422 | The result overflows a 64-bit float (±Inf), e.g. `1e308 ^ 2`. See ADR-0003. | Show a user-facing "result too large" message. | [example](#result_not_finite) |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | `Content-Type` is missing, malformed or not `application/json` (parameters such as `charset=utf-8` are allowed). | Bug in the client: send `Content-Type: application/json`. | [example](#unsupported_media_type) |
| `PAYLOAD_TOO_LARGE` | 413 | The body exceeds 4096 bytes. | Bug in the client: a calculate request is well under 100 bytes. | [example](#payload_too_large) |
| `NOT_FOUND` | 404 | No route matches the path. | Bug in the client: check the base URL and API version. | [example](#not_found) |
| `METHOD_NOT_ALLOWED` | 405 | The path exists but not for this method. The `Allow` response header lists the methods that are allowed. | Bug in the client: use a method from `Allow`. | [example](#method_not_allowed) |
| `RATE_LIMITED` | 429 | The client went over its per-client token bucket (`RATE_LIMIT_RPS` sustained, `RATE_LIMIT_BURST` back to back). The limit is per server process and keyed on the client's address. | Back off for the whole number of seconds in the `Retry-After` header, then retry. Never retry in a tight loop. | [example](#rate_limited) |
| `INTERNAL` | 500 | An unexpected server error, including a panic caught by the recovery middleware. `detail` is deliberately generic; the cause is logged server-side with the same `requestId`. | Show a generic error and allow a retry; report the `requestId` if it persists. | [example](#internal) |
| `TIMEOUT` | 503 | The request took longer than the server's per-request budget (`REQUEST_TIMEOUT`) and was abandoned; the handler's context was canceled with it. | Retry with backoff. A calculation has no side effects, so a retry is always safe. | [example](#timeout) |
| `NOT_READY` | 503 | `GET /readyz` only: the process has started shutting down and is draining its in-flight requests. Operational, never returned by an `/api/v1` route. | Platform concern: a load balancer takes the instance out of rotation and retries elsewhere. | [example](#not_ready) |

## Shape

| Member | Always present | Meaning |
|---|---|---|
| `type` | yes | URL of the code's section on this page |
| `title` | yes | Short human-readable summary of the code; the same for every occurrence |
| `status` | yes | HTTP status code, repeated in the body |
| `detail` | yes | Human-readable explanation of this occurrence; never contains internal error text |
| `code` | yes | Stable machine-readable code from the table above |
| `instance` | yes | Request path |
| `requestId` | when set | The request's `X-Request-ID`, for correlating with server logs |
| `errors` | when field-specific | `[{"field": "...", "message": "..."}]`, one entry per offending request field |

## Examples

Each example below is byte-identical to the golden file `backend/internal/httpapi/testdata/<code>.json`
that the handler tests compare responses against (`TestErrorCatalogueMatchesGoldenFiles` enforces this).
The `requestId` shown is a placeholder.

### INVALID_BODY

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"add","a":"1","b":2}
```

Response (`400`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#invalid_body",
  "title": "Invalid request body",
  "status": 400,
  "detail": "field \"a\" must be a number (byte 26)",
  "code": "INVALID_BODY",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "errors": [
    {
      "field": "a",
      "message": "must be a number"
    }
  ]
}
```

### VALIDATION_FAILED

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"add","a":1}
```

Response (`400`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#validation_failed",
  "title": "Validation failed",
  "status": 400,
  "detail": "missing or invalid fields: b",
  "code": "VALIDATION_FAILED",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "errors": [
    {
      "field": "b",
      "message": "is required for operation \"add\""
    }
  ]
}
```

### UNSUPPORTED_OPERATION

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"modulo","a":1,"b":2}
```

Response (`422`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#unsupported_operation",
  "title": "Unsupported operation",
  "status": 422,
  "detail": "operation \"modulo\" is not supported; GET /api/v1/operations lists the supported operations",
  "code": "UNSUPPORTED_OPERATION",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "errors": [
    {
      "field": "operation",
      "message": "must be one of: add, subtract, multiply, divide, power, sqrt, percent"
    }
  ]
}
```

### UNEXPECTED_OPERAND

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"sqrt","a":4,"b":2}
```

Response (`422`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#unexpected_operand",
  "title": "Unexpected operand",
  "status": 422,
  "detail": "operation \"sqrt\" takes a single operand; omit b",
  "code": "UNEXPECTED_OPERAND",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "errors": [
    {
      "field": "b",
      "message": "must be omitted for operation \"sqrt\""
    }
  ]
}
```

### DIVISION_BY_ZERO

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"divide","a":1,"b":0}
```

Response (`422`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#division_by_zero",
  "title": "Division by zero",
  "status": 422,
  "detail": "b must be non-zero for operation \"divide\"",
  "code": "DIVISION_BY_ZERO",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "errors": [
    {
      "field": "b",
      "message": "must be non-zero"
    }
  ]
}
```

### DOMAIN_ERROR

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"sqrt","a":-1}
```

Response (`422`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#domain_error",
  "title": "Result is not a real number",
  "status": 422,
  "detail": "operation \"sqrt\" has no real-number result for these operands",
  "code": "DOMAIN_ERROR",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### RESULT_NOT_FINITE

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"power","a":1e308,"b":2}
```

Response (`422`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#result_not_finite",
  "title": "Result is not finite",
  "status": 422,
  "detail": "the result of operation \"power\" is outside the 64-bit floating-point range",
  "code": "RESULT_NOT_FINITE",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### UNSUPPORTED_MEDIA_TYPE

Request:

```http
POST /api/v1/calculate
Content-Type: text/plain

{"operation":"add","a":1,"b":2}
```

Response (`415`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#unsupported_media_type",
  "title": "Unsupported media type",
  "status": 415,
  "detail": "Content-Type must be application/json",
  "code": "UNSUPPORTED_MEDIA_TYPE",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### PAYLOAD_TOO_LARGE

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"add", …4096 spaces… "a":1,"b":2}
```

Response (`413`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#payload_too_large",
  "title": "Payload too large",
  "status": 413,
  "detail": "request body must not exceed 4096 bytes",
  "code": "PAYLOAD_TOO_LARGE",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### NOT_FOUND

Request:

```http
GET /api/v1/nope
```

Response (`404`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#not_found",
  "title": "Not found",
  "status": 404,
  "detail": "no route for GET /api/v1/nope",
  "code": "NOT_FOUND",
  "instance": "/api/v1/nope",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### METHOD_NOT_ALLOWED

Request:

```http
GET /api/v1/calculate
```

Response (`405`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#method_not_allowed",
  "title": "Method not allowed",
  "status": 405,
  "detail": "method GET is not allowed; use POST",
  "code": "METHOD_NOT_ALLOWED",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### RATE_LIMITED

Each client gets a token bucket of `RATE_LIMIT_BURST` tokens refilling at `RATE_LIMIT_RPS` per second. A
request that finds the bucket empty is refused here, before the router: the operational endpoints
(`/healthz`, `/readyz`, `/metrics`) are never rate limited, because throttling a probe would take a healthy
instance out of rotation exactly when it is busiest.

`Retry-After` is whole seconds and at least `1`.

Request (the 21st in a second, with the default limits):

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"add","a":1,"b":2}
```

Response (`429`, `application/problem+json`, `Retry-After: 1`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#rate_limited",
  "title": "Too many requests",
  "status": 429,
  "detail": "too many requests; wait for the time in the Retry-After header and try again",
  "code": "RATE_LIMITED",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### INTERNAL

Not reachable through a valid request; produced for any error the service did not anticipate, and for a
panic the recovery middleware caught. The panic value and its stack go to the log at error level under this
same `requestId`; none of it reaches the client. Response (`500`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#internal",
  "title": "Internal server error",
  "status": 500,
  "detail": "an unexpected error occurred",
  "code": "INTERNAL",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### TIMEOUT

The timeout middleware gives every request `REQUEST_TIMEOUT` (default `5s`) and cancels the handler's
context when it runs out. It is `503` rather than `504`: there is no upstream that timed out, the server
itself declined to keep working on the request. A response that the handler had already started is left
alone — there is no way to replace a status line that is on the wire — so this document is what a client
sees whenever one is sent at all.

Request:

```http
POST /api/v1/calculate
Content-Type: application/json

{"operation":"add","a":1,"b":2}
```

Response (`503`, `application/problem+json`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#timeout",
  "title": "Request timeout",
  "status": 503,
  "detail": "the server took too long to produce a response",
  "code": "TIMEOUT",
  "instance": "/api/v1/calculate",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

### NOT_READY

Only `GET /readyz`, and only while the process is draining: readiness is flipped off before the listener closes, so a load balancer stops routing to this instance while it finishes the requests it already has. Liveness (`GET /healthz`) keeps answering `200` throughout — a draining process must not be restarted.

Request:

```http
GET /readyz
```

Response (`503`, `application/problem+json`, `Cache-Control: no-store`):

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#not_ready",
  "title": "Not ready",
  "status": 503,
  "detail": "the server is shutting down and is not accepting new requests",
  "code": "NOT_READY",
  "instance": "/readyz",
  "requestId": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
}
```

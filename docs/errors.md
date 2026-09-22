# Error catalogue

Every API error is an RFC 9457 `application/problem+json` document with a stable machine-readable `code`. The `type` field of each problem points at the anchor for its code on this page. The table is filled by ticket #6 (B1-02) and extended by later tickets; codes are never renamed once released.

| Code | HTTP | Meaning | Client behaviour |
|---|---|---|---|
| _to be filled in B1-02_ | | | |

## Shape

```json
{
  "type": "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#division_by_zero",
  "title": "Division by zero",
  "status": 422,
  "detail": "b must be non-zero for operation \"divide\"",
  "code": "DIVISION_BY_ZERO",
  "instance": "/api/v1/calculate",
  "requestId": "…",
  "errors": [{ "field": "b", "message": "must be non-zero" }]
}
```

// Package httpapi is the HTTP transport for the calculator: routes, DTOs,
// strict request decoding and validation, JSON responses and RFC 9457
// application/problem+json errors.
//
// Handlers stay thin (decode → validate → calc.Evaluate → encode); the
// arithmetic and its rules live in package calc. Every error response is a
// [Problem] with a stable [Code] listed in docs/errors.md (ADR-0005).
package httpapi

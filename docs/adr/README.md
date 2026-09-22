# Architecture Decision Records

One file per decision, numbered, never edited after acceptance except to change **Status**. New decisions supersede old ones instead of rewriting them. Template: [`0000-template.md`](0000-template.md).

| ADR | Title | Status | Ticket |
|---|---|---|---|
| [0001](0001-monorepo-layout.md) | Monorepo with two independent build roots | Accepted | #1 |
| [0002](0002-frontend-stack.md) | Vite + React 19 + TypeScript strict, Tailwind v4, shadcn primitives, TanStack Query, zustand | Accepted | #1 |
| [0003](0003-numeric-model.md) | Numeric model: float64 with non-finite guards | Accepted | #5 |
| [0004](0004-single-calculate-endpoint.md) | Single `POST /api/v1/calculate` with an operation enum + discovery endpoint | Accepted | #6 |
| [0005](0005-problem-json-errors.md) | Errors as RFC 9457 problem+json with a stable machine `code` | Accepted | #6 |
| [0006](0006-runtime-stack.md) | Runtime stack: stdlib router, `log/slog`, Prometheus client | Accepted | #9 |
| 0007 | Display precision: 12 significant digits, exponent beyond ±1e15 / 1e-6 | Planned | #17 |
| [0008](0008-frontend-evaluation-semantics.md) | Evaluate only through the API, with immediate-execution semantics | Accepted | #13 |
| 0009 | Distroless non-root images, same-origin `/api` proxy, no CORS in production | Planned | #20 |

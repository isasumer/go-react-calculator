# Architecture Decision Records

One file per decision, numbered, never edited after acceptance except to change **Status**. New decisions supersede old ones instead of rewriting them. Template: [`0000-template.md`](0000-template.md).

| ADR | Title | Status | Ticket |
|---|---|---|---|
| [0001](0001-monorepo-layout.md) | Monorepo with two independent build roots | Accepted | #1 |
| [0002](0002-frontend-stack.md) | Vite + React 19 + TypeScript strict, Tailwind v4, shadcn primitives, TanStack Query, zustand | Accepted | #1 |
| [0003](0003-numeric-model.md) | Numeric model: float64 with non-finite guards | Accepted | #5 |
| 0004 | Single `POST /api/v1/calculate` with an operation enum + discovery endpoint | Planned | #6 |
| 0005 | Errors as RFC 9457 problem+json with a stable machine `code` | Planned | #6 |
| 0006 | Runtime stack: stdlib router, `log/slog`, Prometheus client | Planned | #9 |
| 0007 | Display precision: 12 significant digits, exponent beyond ±1e15 / 1e-6 | Planned | #17 |
| 0008 | Frontend evaluates only through the API; immediate-execution semantics | Planned | #13 |
| 0009 | Distroless non-root images, same-origin `/api` proxy, no CORS in production | Planned | #20 |

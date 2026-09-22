# Architecture

> Stub. Completed by ticket #27 (D4-03) once the implementation exists. The request flow below is the
> contract every ticket builds towards; see [`PLAN.md`](PLAN.md) §1 for the layout and decisions.

## Request flow

```mermaid
flowchart LR
  subgraph Browser
    K[Keypad / keyboard] --> S[useCalculatorStore<br/>pure state machine]
    S -->|effect: calculate| M[useCalculate<br/>TanStack mutation]
    M --> F[apiFetch<br/>problem+json aware, zod-validated]
    F --> D[Display + History<br/>localStorage]
  end
  F -->|same-origin /api/*| N[nginx<br/>static + reverse proxy]
  N --> G[Go service]
  subgraph G[Go service]
    direction TB
    R[recover] --> I[request id] --> L[slog logger] --> T[timeout] --> H[security headers] --> C[CORS] --> RL[rate limit] --> MT[metrics] --> RT[router] --> HD[handler] --> CALC[calc.Evaluate]
  end
```

## Sections to be written (D4-03)
- Backend request lifecycle and where each HTTP status originates
- Error taxonomy (code → status → origin → client behaviour), cross-referenced with [`errors.md`](errors.md)
- Frontend data flow and the calculator state machine (mermaid `stateDiagram`)
- Storage schema and key versioning policy
- Operational notes: configuration, probes, metrics, shutdown sequence
- Threat model and known limitations

# Changelog

## v1.0.0 — 2026-09-23

First tagged release: a production-shaped full-stack calculator (Go REST service + React/TypeScript SPA),
containerised, tested end to end, and documented with ADRs. Generated from the 22 merged pull requests,
grouped by Conventional Commit type; two PR titles predate a strict prefix and are classified by content
(noted inline).

### feat

- feat(backend): add calc domain package with registry, typed errors and fuzz tests (#59)
- feat(backend): implement calculate/operations endpoints with strict validation and problem+json errors (#61)
- feat(backend): add typed config, graceful lifecycle and health/readiness/version endpoints (#63)
- feat(frontend): add typed API layer with problem+json errors, zod validation and query hooks (#64)
- feat(frontend): add pure calculator state machine and zustand store (#68)
- feat(backend): middleware chain — recovery, request ID, structured logging, timeout, CORS, security headers, rate limiting (#69, title carried no prefix)
- feat(backend): expose Prometheus metrics for HTTP and calculator operations (#71)
- feat(frontend): build calculator UI with responsive layout, dark mode and accessible controls (#74)
- feat(backend): add OpenAPI 3.1 contract, serve it and validate responses against it in tests (#76)
- feat(frontend): add keyboard support and input-validation feedback (#77)
- feat(frontend): persist calculation history in localStorage with a history panel (#82)

### build

- build: containers — backend + frontend images, compose stack, smoke script, CI stack job; absorbs #20, #21 (#78, title carried no prefix)

### test

- test(e2e): add Playwright suite running against the compose stack in CI (#84)

### ci

- ci: add backend and frontend workflows with coverage gates and branch protection (#52)
- ci(dependabot): ignore major bumps for @types/node and eslint (#70)
- ci: add CodeQL, Trivy, gitleaks and audit steps; add SECURITY.md (#83)

### docs

- docs: allow force-with-lease on own feature branch for rebases (#66)
- docs: record ticket consolidation in the plan (#72)
- docs: finalise README with quick starts, API reference, design decisions and testing report (#85)

### chore

- chore: bootstrap repository layout, docs skeleton and templates (#47)
- chore(frontend): scaffold Vite React TS app with Tailwind, lint and test harness (#51)
- chore(deps-dev): bump jsdom from 29.1.1 to 30.1.0 in /frontend (#58)

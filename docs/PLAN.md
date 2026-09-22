# Implementation Plan — go-react-calculator

> Living document. Tickets below are mirrored 1:1 as GitHub issues; the issue is the source of truth for
> status, this file is the source of truth for the *why* and the cross-ticket contracts.

## 0. Context

**Assignment (summary).** Full-stack calculator: React (TypeScript) frontend consuming a Go REST
microservice. Operations: add, subtract, multiply, divide; optional power, square root, percentage.
Frontend: intuitive UI, validation, error handling, basic responsiveness. Backend: endpoints per operation,
input validation, edge cases (division by zero, invalid data), JSON responses. Non-functional: clean
idiomatic code, unit tests on both layers with a coverage report, README (setup, run, API examples, design
rationale), optional Dockerfile for the whole stack. Guidance: ~2–4 h, correctness/clarity/maintainability
over features, share the AI prompts used.

**What is actually being evaluated.** The calculator is a vehicle. The signals that matter, in order:
idiomatic Go, test discipline (a *coverage report*, not just tests), scope judgment, written communication
(README, design decisions), and how AI tooling is used (prompts are a graded deliverable).

**Our stance.** We deliberately exceed the 2–4 h envelope to demonstrate senior, production-grade
engineering, but we do it in a way that does not contradict the "no extra features" guidance:

1. **The product surface stays exactly the assignment.** Seven operations, one screen. No auth, no
   database, no accounts.
2. **The extra investment goes into operational maturity**, not features: config, graceful shutdown,
   structured logs, request IDs, metrics, rate limiting, problem+json errors, OpenAPI contract, CI gates,
   supply-chain checks, hardened containers, e2e tests, ADRs.
3. **Sprint 2 ends at a submittable checkpoint.** If time runs out, the repo at the end of Sprint 2 already
   satisfies every mandatory deliverable. Sprints 3–4 harden and polish.
4. **Honesty in the README.** A "time spent" section and a clear split between "what the assignment asked"
   and "what I added and why". Reviewers reward candour; they punish padding that pretends to be 4 hours.

## 1. Architecture

### 1.1 Repository layout (monorepo, two independently buildable apps)

```
go-react-calculator/
├── backend/                     # Go module: github.com/isasumer/go-react-calculator/backend
│   ├── cmd/server/main.go       # composition root: config → deps → router → http.Server; run() error pattern
│   ├── internal/calc/           # pure domain: operations, typed errors, IEEE-754 guards (zero deps)
│   ├── internal/httpapi/        # transport: routes, DTOs, decode/validate/encode, problem+json
│   ├── internal/middleware/     # request-id, logging, recovery, cors, security headers, rate limit, timeout
│   ├── internal/config/         # env → typed Config, validated, with defaults
│   ├── internal/observability/  # slog setup, Prometheus metrics, build info
│   ├── api/openapi.yaml         # OpenAPI 3.1 contract (served by the service, validated in tests)
│   ├── Dockerfile
│   └── Makefile
├── frontend/                    # Vite + React 19 + TypeScript (strict)
│   ├── src/
│   │   ├── config.ts            # single place that reads import.meta.env
│   │   ├── app/                 # App.tsx, providers wiring, global styles
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/radix primitives (kebab-case files)
│   │   │   ├── providers/       # QueryProvider, ErrorBoundary
│   │   │   ├── shared/          # cross-feature PascalCase components
│   │   │   └── calculator/      # feature components: Calculator, Display, Keypad, Key, HistoryPanel
│   │   ├── hooks/               # use-calculate.ts, use-operations.ts, use-keyboard.ts (kebab-case)
│   │   ├── lib/                 # api.ts, query-config.ts, calculator-engine.ts, format-number.ts, storage.ts, utils.ts
│   │   ├── stores/              # useCalculatorStore.ts, useHistoryStore.ts (zustand)
│   │   ├── types/               # calculator.ts (hand-written interfaces + zod schemas for API responses)
│   │   └── test/                # setup.ts, msw/handlers.ts, fixtures
│   ├── nginx/                   # nginx.conf for the production image
│   └── Dockerfile
├── e2e/                         # Playwright tests against the composed stack
├── docs/                        # PLAN.md, ARCHITECTURE.md, PROMPTS.md, adr/
├── .github/                     # workflows, templates, dependabot, CODEOWNERS
├── compose.yaml                 # full stack: frontend (nginx :8080) → backend (:8081)
├── Makefile                     # umbrella targets: dev, check, test, up, down, e2e
├── CLAUDE.md                    # session bootstrap for AI-assisted work (see §2.4)
└── README.md
```

The frontend layout mirrors the conventions of a production Next.js codebase I maintain (flat `lib/`,
kebab-case hooks, one small zustand store, hand-rolled versioned localStorage keys, `data-ui` attributes
on component roots, co-located tests). Differences are deliberate and recorded in ADR-0002: Vite instead
of Next.js (no routing/SSR need), and component tests with Testing Library + MSW (the reference codebase
only unit-tests `lib/`).

### 1.2 Request flow

```
Browser ──(same origin /api/*)──▶ nginx ──proxy──▶ Go service
   │                                                  │
   │  Keypad → useCalculatorStore (state machine)      │  middleware chain:
   │  → evaluate() → useCalculate (TanStack mutation)  │  recover → requestID → logger → timeout
   │  → apiFetch (problem+json aware) → zod-validated  │  → securityHeaders → cors → rateLimit → metrics
   │  → display + history (localStorage)               │  → router → handler → calc.Evaluate → JSON
```

### 1.3 Key decisions (each gets an ADR under `docs/adr/`)

| ADR | Decision | Short rationale |
|---|---|---|
| 0001 | Monorepo, two independent build roots | One link to share; each app still builds/tests alone; CI path filters keep runs cheap |
| 0002 | Vite + React 19 + TS strict, Tailwind v4, shadcn primitives, TanStack Query, zustand | Matches a real production frontend I run; Vite because no SSR/routing need |
| 0003 | `float64` numeric model with explicit non-finite guards; decimal mode is a documented follow-up | Assignment scope; JSON numbers are doubles anyway; document 0.1+0.2 honestly. In a money context we would use decimal arithmetic — say so |
| 0004 | Single `POST /api/v1/calculate` with an `operation` enum + `GET /api/v1/operations` discovery | Adding an operation is one registry entry; URL surface stays flat; discovery endpoint feeds the UI |
| 0005 | Errors as RFC 9457 `application/problem+json` with a stable machine `code` | Standardised, extensible, client maps `code` → UX; 400 (malformed), 422 (semantic: division by zero, sqrt of negative, non-finite result), 405/415/413/429 |
| 0006 | Go standard library router (1.22+ method patterns), `log/slog`, Prometheus client as the only non-x dependency | Idiomatic; reviewers can read everything; metrics match a Prometheus/Grafana shop |
| 0007 | Display precision: 12 significant digits, exponent notation beyond ±1e15/1e-6 | Calculator UX convention; avoids showing 0.30000000000000004 while staying truthful in the API |
| 0008 | Frontend evaluates *only* through the API; immediate-execution (no precedence) semantics | Assignment requires consuming the backend; classic 4-function UX; expression mode is a follow-up |
| 0009 | Distroless non-root images, same-origin `/api` proxy via nginx, no CORS needed in prod | Smallest attack surface; CORS allowlist exists only for local dev |

### 1.4 API contract (frozen after B1-02; changes require an ADR)

```
POST /api/v1/calculate
  {"operation":"divide","a":10,"b":4}                → 200 {"operation":"divide","a":10,"b":4,"result":2.5}
  {"operation":"sqrt","a":16}                        → 200 {"operation":"sqrt","a":16,"result":4}
  {"operation":"percent","a":200,"b":15}             → 200 {"operation":"percent","a":200,"b":15,"result":30}   # 15% of 200
  {"operation":"divide","a":1,"b":0}                 → 422 problem+json code=DIVISION_BY_ZERO
  {"operation":"sqrt","a":-1}                        → 422 code=DOMAIN_ERROR
  {"operation":"power","a":1e308,"b":2}              → 422 code=RESULT_NOT_FINITE
  {"operation":"modulo","a":1,"b":2}                 → 422 code=UNSUPPORTED_OPERATION
  {"operation":"add","a":"1","b":2}                  → 400 code=INVALID_BODY (with errors[] pointing at /a)
  {"operation":"add","a":1}                          → 400 code=VALIDATION_FAILED errors=[{field:"b",...}]
GET  /api/v1/operations                              → 200 {"operations":[{"name":"add","symbol":"+","arity":2}, ...]}
GET  /healthz | /readyz | /version | /metrics | /api/v1/openapi.yaml
```

Problem shape:
```json
{"type":"https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#division_by_zero",
 "title":"Division by zero","status":422,"detail":"b must be non-zero for operation \"divide\"",
 "code":"DIVISION_BY_ZERO","instance":"/api/v1/calculate","requestId":"…","errors":[{"field":"b","message":"must be non-zero"}]}
```

## 2. Working agreement

### 2.1 Branching, commits, PRs
- `main` is protected: PR required, CI green required, linear history (squash merge).
- Branch name: `<area>/<issue#>-<slug>` e.g. `backend/12-domain-package`, `frontend/21-api-layer`, `platform/4-ci`.
- Conventional Commits: `feat(backend): …`, `fix(frontend): …`, `chore(ci): …`, `docs: …`, `test(backend): …`.
- One issue → one PR. PR title = issue title. PR body: `Closes #N`, summary, test evidence (command + output
  excerpt), screenshots for UI, and a "Prompts used" section (copied to `docs/PROMPTS.md`).
- No drive-by changes. If you find something outside scope, open a `follow-up` issue and link it.

### 2.2 Definition of Done (every ticket)
- Acceptance criteria in the issue all checked.
- `make check` passes locally (lint + vet + typecheck + tests + coverage thresholds).
- New behaviour has tests; coverage does not drop below the gate (backend 85 %, frontend 85 % lines / 80 % branches).
- Docs updated where the ticket says so (README section, ADR, OpenAPI).
- `docs/PROMPTS.md` appended with the session's prompts (verbatim) and what was accepted/rejected.
- PR reviewed against the checklist in `.github/pull_request_template.md`, squash-merged, branch deleted.

### 2.3 Estimates
Estimates are per focused session (S = ≤1 h, M = 1–2 h, L = 2–3 h). Total ≈ 30–35 h across ~30 tickets.

### 2.4 Session protocol (each ticket is a fresh AI session)
The repo's `CLAUDE.md` bootstraps a session: read the issue, read `docs/PLAN.md` §1 and §2, read only the
directories the issue lists, implement, run `make check`, open the PR, append prompts. Sessions never
re-plan; if the plan is wrong, the session opens a follow-up issue with the proposed change and stops.

## 3. Sprints and tickets

Sprint goals:

| Sprint | Goal | Exit criterion |
|---|---|---|
| 0 Foundation | Repo, tooling, CI skeleton, docs skeleton | Empty apps build, lint and test green in CI on every PR |
| 1 Backend core | Complete, hardened Go service | All API contract cases pass; coverage ≥ 85 %; `curl` examples in README work |
| 2 Frontend core | Complete React app against the real API | Every mandatory deliverable satisfied → **submittable checkpoint** |
| 3 Hardening | Containers, compose, e2e, security, release pipeline | `docker compose up` on a clean machine + green e2e in CI |
| 4 Docs & submission | README/ADR/PROMPTS final, v1.0.0, submission | Fresh-clone review passes the checklist; recruiter email drafted |

Ticket details follow (generated from the ticket manifest, exported as `docs/tickets.json`; identical text lives in each GitHub issue).


## Sprint 0 — Foundation

### P0-01 — Repo bootstrap: layout, docs skeleton, templates, Makefile, CLAUDE.md

**Labels:** platform, docs, sprint-0 · **Estimate:** M

**Goal.** Turn the empty repository into a navigable project where every later session finds the plan, the conventions and the templates it needs.

**Scope**
- Root files: .gitignore (Go, Node, coverage, .env*, IDE), .editorconfig, LICENSE (MIT), README.md skeleton with the final section headings (Overview, Quick start, API, Design decisions, Testing, Project structure, Time log, Prompts)
- docs/: PLAN.md (this plan), ARCHITECTURE.md (stub with request-flow diagram), PROMPTS.md (log format + first entry), errors.md (error catalogue stub), adr/README.md + adr/0000-template.md + ADR-0001 (monorepo) + ADR-0002 (frontend stack)
- .github/: pull_request_template.md (checklist from DoD), ISSUE_TEMPLATE/ticket.md + follow-up.md, CODEOWNERS (@isasumer)
- Root Makefile with umbrella targets delegating to backend/ and frontend/: dev, check, test, lint, up, down, e2e (targets may be stubs that fail with a clear message until the app tickets land)
- CLAUDE.md: refine the initial version committed with the plan (session bootstrap per PLAN §2.4): add the `make check` contract, prompt-logging steps and the list of forbidden actions (no scope creep, no plan edits)

**Out of scope**
- Any application code
- CI workflows (P0-04)

**Implementation notes**
- Keep README skeleton headings final so later tickets only fill sections, never restructure.
- PROMPTS.md format: `## Session <ticket-id> — <date>` → numbered prompts verbatim → `Accepted` / `Rejected (why)` bullets.

**Acceptance criteria**
- [ ] Fresh clone: `make help` lists targets; `docs/PLAN.md`, `CLAUDE.md`, ADR-0001/0002 present and linked from README
- [ ] PR template renders the DoD checklist
- [ ] No application code added

**Tests**
- n/a (docs/tooling only)

**Files / areas**
- /
- docs/
- .github/
- Makefile
- CLAUDE.md

**PR title:** `chore: bootstrap repository layout, docs skeleton and templates`

### P0-02 — Backend scaffold: Go module, layout, lint/vet/staticcheck tooling, Makefile

**Labels:** backend, platform, sprint-0 · **Estimate:** S · **Depends on:** P0-01

**Goal.** A compiling, lint-clean Go module with the final package layout and quality gates wired, so domain work starts on rails.

**Scope**
- backend/go.mod (module github.com/isasumer/go-react-calculator/backend, go 1.27)
- cmd/server/main.go with `func main(){ if err := run(ctx, os.Args, os.Getenv, os.Stdout); err != nil { … os.Exit(1) } }` pattern and a placeholder HTTP server on :8081 returning 404 for everything
- Empty packages with doc.go: internal/calc, internal/httpapi, internal/middleware, internal/config, internal/observability
- .golangci.yml (enable: govet, staticcheck, errcheck, gosimple, revive, gocritic, gosec, misspell, errorlint, exhaustive, nilerr, bodyclose, noctx, gofumpt) with sane exclusions for tests
- backend/Makefile: fmt (gofumpt), lint (golangci-lint), vet, test (`go test -race -shuffle=on -coverprofile`), cover (func summary), coverage-check (script fails <85 %), vuln (govulncheck), build (ldflags version/commit/date), run
- tools: pin versions via `go run` tool directives or `tools.go`; document install in backend/README.md section

**Out of scope**
- Real routes/handlers (B1-02)
- Dockerfile (H3-01)

**Implementation notes**
- Go binary lives at ~/.local/go/bin on the dev machine; CI installs via actions/setup-go reading go.mod.
- Prefer `go 1.27` toolchain features: `net/http` method patterns, `slog`, `min/max` builtins, range-over-int.

**Acceptance criteria**
- [ ] `make -C backend check` runs fmt-check, lint, vet, test and passes on the empty module
- [ ] Binary starts, logs listening address, exits 0 on SIGINT
- [ ] golangci-lint config committed and green

**Tests**
- A smoke test for run() that starts on port 0 and cancels the context

**Files / areas**
- backend/

**PR title:** `chore(backend): scaffold Go module, package layout and quality tooling`

### P0-03 — Frontend scaffold: Vite + React 19 + TS strict, Tailwind v4, ESLint/Prettier, Vitest + RTL + MSW

**Labels:** frontend, platform, sprint-0 · **Estimate:** M · **Depends on:** P0-01

**Goal.** A compiling, lint-clean React app with the final folder structure, theme tokens and test harness, mirroring the reference frontend conventions.

**Scope**
- `npm create vite@latest frontend -- --template react-ts`, pin Node 22 in .nvmrc and package.json engines
- tsconfig: strict, noUncheckedIndexedAccess, exactOptionalPropertyTypes, path alias `@/* → src/*` (also in vite.config and vitest.config)
- Tailwind v4 (`@tailwindcss/vite`), `src/app/globals.css` with `@theme` tokens (surface, text, accent, danger, key colours), `@custom-variant dark` via prefers-color-scheme
- shadcn init (new-york), add `button` primitive to src/components/ui; `src/lib/utils.ts` with cn()
- ESLint flat config: typescript-eslint recommended-type-checked, react, react-hooks, jsx-a11y, eslint-config-prettier last; Prettier (semi, double quotes, width 100, trailingComma all)
- Vitest + jsdom + @testing-library/react + jest-dom + user-event + msw; `src/test/setup.ts` (jest-dom matchers, msw server lifecycle, localStorage reset); include `src/**/*.test.{ts,tsx}`
- Folder skeleton with .gitkeep/doc comments: app/, components/{ui,providers,shared,calculator}, hooks/, lib/, stores/, types/, test/msw
- `src/config.ts`: typed getConfig() reading import.meta.env (VITE_API_BASE_URL default "/api"), validated once
- scripts: dev, build, preview, lint, type-check, test, test:coverage, format, format:check; frontend/Makefile mirroring them
- Convention doc in frontend/README.md: PascalCase feature components, kebab-case ui primitives and hooks, `data-ui` attribute on component roots, co-located tests

**Out of scope**
- Calculator components, API layer, stores (Sprint 2)

**Implementation notes**
- Reference conventions come from a production Next.js app: flat lib/, one small zustand store, hand-rolled versioned storage keys, `data-ui` markers. Record the Vite-vs-Next choice in ADR-0002 (already drafted in P0-01; finalise here).

**Acceptance criteria**
- [ ] `make -C frontend check` runs lint, type-check, test and passes with a placeholder App test
- [ ] `npm run build` produces dist/ with hashed assets
- [ ] Dark/light tokens switch with the OS setting on the placeholder page

**Tests**
- Placeholder App.test.tsx renders the app shell
- config.test.ts covers default and override of VITE_API_BASE_URL

**Files / areas**
- frontend/

**PR title:** `chore(frontend): scaffold Vite React TS app with Tailwind, lint and test harness`

### P0-04 — CI v1: backend and frontend workflows, path filters, caching, branch protection, Dependabot

**Labels:** platform, sprint-0 · **Estimate:** M · **Depends on:** P0-02, P0-03

**Goal.** Every PR is validated automatically; main cannot receive unverified code.

**Scope**
- .github/workflows/backend.yml: on PR/push(main) with paths filter `backend/**`; jobs: lint (golangci-lint-action), test (`make test` with -race, upload coverage.out + HTML as artifact, write summary table to $GITHUB_STEP_SUMMARY), vuln (govulncheck)
- .github/workflows/frontend.yml: paths `frontend/**`; setup-node 22 with npm cache; `npm ci`, lint, type-check, test:coverage (upload lcov + summary), build
- Concurrency groups per ref, cancel-in-progress; timeouts; least-privilege `permissions:`; actions pinned to full SHAs with version comments
- Coverage gates enforced in CI (backend script, vitest thresholds)
- .github/dependabot.yml: gomod, npm, github-actions, docker — weekly, grouped minor/patch
- Branch protection on main via `gh api`: require PR, require status checks (backend/lint, backend/test, frontend/check), dismiss stale reviews, linear history, no force push
- Status badges added to README header

**Out of scope**
- Docker build/e2e jobs (H3-03/H3-04)
- CodeQL/Trivy (H3-05)
- Release workflow (H3-07)

**Implementation notes**
- Required checks must have stable job names; use a final `ci-ok` job that needs all others so protection references a single check per workflow.

**Acceptance criteria**
- [ ] A PR touching only backend/ runs only the backend workflow
- [ ] Coverage summary visible in the job summary
- [ ] Direct push to main is rejected
- [ ] Badges render green on README

**Tests**
- Open a throwaway PR that breaks lint; confirm it is blocked; close it

**Files / areas**
- .github/workflows/
- .github/dependabot.yml
- README.md

**PR title:** `ci: add backend and frontend workflows with coverage gates and branch protection`


## Sprint 1 — Backend core

### B1-01 — Domain package internal/calc: operations registry, typed errors, IEEE-754 guards, fuzz + benchmarks

**Labels:** backend, testing, sprint-1 · **Estimate:** M · **Depends on:** P0-02

**Goal.** A dependency-free, exhaustively tested arithmetic core that the transport layer can trust blindly.

**Scope**
- Types: `type Operation string` with constants add, subtract, multiply, divide, power, sqrt, percent; `type Spec struct{ Name Operation; Symbol string; Arity int; fn func(a, b float64) (float64, error) }`
- `Registry` (ordered, immutable after init) with `Lookup(name) (Spec, bool)` and `List() []Spec`; `Evaluate(op Operation, a float64, b *float64) (float64, error)` validating arity
- Sentinel errors: ErrUnknownOperation, ErrDivisionByZero, ErrDomain (sqrt of negative), ErrNotFinite (overflow → ±Inf or NaN), ErrArity; wrap with %w and operation context
- Guards: inputs must be finite (reject NaN/±Inf), result must be finite; percent defined as a*b/100 ("b percent of a") and documented; power via math.Pow with domain checks (e.g. (-8)^(1/3) → NaN → ErrDomain)
- Package doc comment explaining the numeric model and linking ADR-0003

**Out of scope**
- HTTP concerns
- Decimal arithmetic (follow-up FU-02)
- Expression parsing (FU-01)

**Implementation notes**
- Table-driven tests with named cases; include boundary rows: MaxFloat64 overflow, -0 handling, 0/0, sqrt(0), 0^0 (=1, document), negative percent.
- Add `FuzzEvaluate` asserting: never panics; result is finite whenever err == nil; error is one of the sentinels otherwise.
- Add `BenchmarkEvaluate` for the record (README testing section).

**Acceptance criteria**
- [ ] 100 % statement coverage on internal/calc
- [ ] `go test -fuzz=FuzzEvaluate -fuzztime=10s` runs clean
- [ ] Every sentinel error has at least one test asserting errors.Is
- [ ] ADR-0003 (numeric model) written

**Tests**
- calc_test.go table tests
- calc_fuzz_test.go
- calc_bench_test.go
- example_test.go with an Example for godoc

**Files / areas**
- backend/internal/calc/
- docs/adr/0003-numeric-model.md

**PR title:** `feat(backend): add calc domain package with registry, typed errors and fuzz tests`

### B1-02 — HTTP transport internal/httpapi: routes, strict JSON decoding, validation, problem+json errors

**Labels:** backend, sprint-1 · **Estimate:** L · **Depends on:** B1-01

**Goal.** The public API contract from PLAN §1.4 implemented end-to-end with httptest coverage of every documented status code.

**Scope**
- Router: `http.NewServeMux()` with method patterns (`POST /api/v1/calculate`, `GET /api/v1/operations`); 405 with Allow header for wrong methods; 404 problem for unknown routes
- DTOs: CalculateRequest{Operation string; A *float64; B *float64}, CalculateResponse, OperationsResponse; JSON tags; `B` optional for unary ops and rejected when supplied to unary ops (422 code=UNEXPECTED_OPERAND)
- Decoding: enforce Content-Type application/json (415), `http.MaxBytesReader` 4 KiB (413), `json.Decoder` with DisallowUnknownFields and a check that exactly one JSON value is present (400 code=INVALID_BODY with json error translated to field/offset)
- Validation layer producing `errors[]` (field, message) — missing operation/a/b, unknown operation (422 UNSUPPORTED_OPERATION), non-finite input (JSON can't carry these, but guard anyway)
- Error mapping: calc.ErrDivisionByZero→422 DIVISION_BY_ZERO, ErrDomain→422 DOMAIN_ERROR, ErrNotFinite→422 RESULT_NOT_FINITE, unknown→500 INTERNAL (no detail leak)
- `problem` package/file: Problem struct (type,title,status,detail,instance,code,requestId,errors), `Write(w, r, p)` sets `Content-Type: application/problem+json`; catalogue in docs/errors.md with anchors used as `type`
- Response encoding via a small `writeJSON` helper (sets charset, handles encode errors, no trailing newline issues)
- Handlers take dependencies via a struct (`Handler{calc *calc.Registry, log *slog.Logger}`) — no globals

**Out of scope**
- Middleware (B1-04)
- Config/lifecycle (B1-03)
- OpenAPI file (B1-06)

**Implementation notes**
- Keep handlers thin: decode → validate → calc.Evaluate → encode. Business rules live in calc.
- Use `errors.As`/`errors.Is` for mapping; a `mapError(err) Problem` function with its own table test.
- Write golden-file tests for problem bodies (testdata/*.json) so the documented examples in README are literally the test fixtures.

**Acceptance criteria**
- [ ] Every row of PLAN §1.4 has a passing httptest case
- [ ] Unknown field, trailing garbage, empty body, non-object body each produce a 400 with a helpful detail
- [ ] Coverage ≥ 90 % on internal/httpapi
- [ ] docs/errors.md lists every code with status, meaning and example

**Tests**
- handler_test.go (table over request → status/code/body)
- problem_test.go
- golden files under testdata/

**Files / areas**
- backend/internal/httpapi/
- docs/errors.md

**PR title:** `feat(backend): implement calculate/operations endpoints with strict validation and problem+json errors`

### B1-03 — Config, server lifecycle, graceful shutdown, health/readiness/version endpoints

**Labels:** backend, sprint-1 · **Estimate:** M · **Depends on:** B1-02

**Goal.** The service behaves like a well-run production process: 12-factor config, sane timeouts, clean shutdown, probes.

**Scope**
- internal/config: `Load(getenv func(string) string) (Config, error)` — PORT (8081), HOST, LOG_LEVEL, LOG_FORMAT (json|text), CORS_ALLOWED_ORIGINS (csv), READ_HEADER_TIMEOUT, READ_TIMEOUT, WRITE_TIMEOUT, IDLE_TIMEOUT, SHUTDOWN_TIMEOUT, REQUEST_TIMEOUT, RATE_LIMIT_RPS, RATE_LIMIT_BURST, MAX_BODY_BYTES; typed, validated, with defaults; `String()` redacting nothing sensitive (there is none) for startup log
- cmd/server: `run(ctx, args, getenv, stdout)` builds config, logger, registry, handler, middleware chain, `http.Server` with all timeouts, listens, and on SIGTERM/SIGINT drains with SHUTDOWN_TIMEOUT; returns error instead of calling os.Exit deep in the stack; `-version` flag prints build info and exits
- Endpoints: `GET /healthz` (liveness, always 200 once process is up), `GET /readyz` (200 after listener bound, 503 while shutting down — atomic flag flipped before drain), `GET /version` {version, commit, buildDate, goVersion}
- Build info via `-ldflags -X` variables in internal/observability/buildinfo.go with `debug.ReadBuildInfo` fallback
- Startup log line with resolved config; shutdown log with drain duration

**Out of scope**
- Metrics (B1-05)
- Middleware behaviours themselves (B1-04)

**Implementation notes**
- Test shutdown by starting run() on port 0, hitting /readyz, cancelling ctx, asserting run returns nil within the timeout and in-flight request completes.
- Readiness must flip before `Shutdown` so a load balancer stops routing first; sleep a configurable pre-stop delay (default 0 locally, documented for k8s).

**Acceptance criteria**
- [ ] `PORT=0 go run ./cmd/server` logs the bound address; Ctrl-C returns exit 0 within SHUTDOWN_TIMEOUT
- [ ] Invalid env value fails fast with a message naming the variable
- [ ] /healthz /readyz /version behave as specified with tests

**Tests**
- config_test.go (defaults, overrides, invalid values)
- main_test.go lifecycle test
- probe handler tests

**Files / areas**
- backend/cmd/server/
- backend/internal/config/
- backend/internal/observability/buildinfo.go

**PR title:** `feat(backend): add typed config, graceful lifecycle and health/readiness/version endpoints`

### B1-04 — Middleware chain: recovery, request ID, structured logging, timeout, CORS, security headers, rate limiting

**Labels:** backend, sprint-1 · **Estimate:** L · **Depends on:** B1-03

**Goal.** Cross-cutting production behaviour, composed explicitly and individually tested.

**Scope**
- `middleware.Chain(h, mws...)` helper; order (outermost first): Recover → RequestID → Logger → Timeout → SecurityHeaders → CORS → RateLimit → (metrics, B1-05) → router
- RequestID: honour incoming `X-Request-ID` if it matches `^[A-Za-z0-9-_]{1,64}$`, else generate (crypto/rand 16 bytes base32); set on response and in context; problem responses echo it
- Logger: slog with request-scoped attrs (request_id, method, path, status, bytes, duration_ms, remote_ip, user_agent); wrap ResponseWriter to capture status/bytes (implement Flush/Unwrap); log level per status (5xx error, 4xx warn, else info); skip /healthz /readyz /metrics at info level
- Recover: catch panics → log with stack → 500 problem (code=INTERNAL); never leaks panic value to client
- Timeout: `http.TimeoutHandler`-like but emitting a problem+json 503/504 (code=TIMEOUT); REQUEST_TIMEOUT from config
- SecurityHeaders: X-Content-Type-Options=nosniff, X-Frame-Options=DENY, Referrer-Policy=no-referrer, Cache-Control=no-store on API routes, Content-Security-Policy default-src 'none' (API only serves JSON)
- CORS: allowlist from config; handles preflight (204) with Access-Control-Allow-Methods/Headers/Max-Age; Vary: Origin; disabled (no headers) when list is empty — production uses same-origin proxy
- RateLimit: per-client-IP token bucket (golang.org/x/time/rate) with LRU/TTL eviction of idle buckets; 429 problem (code=RATE_LIMITED) with Retry-After; client IP from RemoteAddr only (document that X-Forwarded-For trust is a deployment decision, off by default; TRUST_PROXY_HEADERS env to enable)

**Out of scope**
- Metrics middleware (B1-05)
- Auth (not required)

**Implementation notes**
- Each middleware in its own file with its own test using httptest and a recording handler.
- Keep x/time/rate as the only new dependency here; justify in go.mod tidy commit message.

**Acceptance criteria**
- [ ] Panic in a handler yields 500 problem with request id, process keeps serving
- [ ] Preflight from allowed origin → 204 with correct headers; from other origin → no CORS headers
- [ ] Burst above limit → 429 with Retry-After; recovers after refill
- [ ] Every response carries X-Request-ID
- [ ] Logs are JSON in LOG_FORMAT=json with the documented keys

**Tests**
- One *_test.go per middleware
- chain_test.go asserting order via a probe handler

**Files / areas**
- backend/internal/middleware/

**PR title:** `feat(backend): add middleware chain (recovery, request id, logging, timeout, CORS, security headers, rate limit)`

### B1-05 — Observability: Prometheus metrics endpoint and calculation metrics

**Labels:** backend, sprint-1 · **Estimate:** S · **Depends on:** B1-04

**Goal.** Operators can see traffic, latency, errors and per-operation usage without reading logs.

**Scope**
- internal/observability/metrics.go: registry (non-global `prometheus.NewRegistry` + Go/process collectors), `http_requests_total{method,route,status}`, `http_request_duration_seconds{method,route}` histogram (buckets tuned for ms), `http_in_flight_requests`, `calc_operations_total{operation,outcome}` (outcome=ok|error_code), `build_info{version,commit}` gauge = 1
- Metrics middleware using the route pattern (`r.Pattern` from Go 1.22 mux) as the label, never the raw path (cardinality)
- `GET /metrics` via promhttp on the same listener (documented; separate admin port is a follow-up)
- Handler increments calc_operations_total with the problem `code` on error
- ADR-0006 (stdlib router + slog + Prometheus)

**Out of scope**
- Tracing/OpenTelemetry (FU-03)
- Grafana dashboards (FU-03)

**Implementation notes**
- Use a small `Metrics` struct injected into middleware/handler so tests can use a fresh registry and assert via `testutil.CollectAndCount` / `ToFloat64`.

**Acceptance criteria**
- [ ] `curl /metrics` shows the five metric families with expected labels after a few requests
- [ ] Route label is `/api/v1/calculate`, not `/api/v1/calculate?x=1` or path variants
- [ ] Tests assert counters via prometheus/testutil

**Tests**
- metrics_test.go
- middleware metrics test

**Files / areas**
- backend/internal/observability/
- backend/internal/middleware/metrics.go
- docs/adr/0006-runtime-stack.md

**PR title:** `feat(backend): expose Prometheus metrics for HTTP and calculator operations`

### B1-06 — OpenAPI 3.1 contract: authored spec, served endpoint, contract tests

**Labels:** backend, docs, sprint-1 · **Estimate:** M · **Depends on:** B1-02

**Goal.** The API is described by a machine-readable contract that the implementation is proven to honour.

**Scope**
- backend/api/openapi.yaml: info, servers, paths (/api/v1/calculate, /api/v1/operations, /healthz, /readyz, /version), components: CalculateRequest (oneOf binary/unary via `operation` enum + required rules), CalculateResponse, Operation, Problem (RFC 9457 + code enum), examples for every documented error
- Serve the file at `GET /api/v1/openapi.yaml` (embed via `embed.FS`, Content-Type application/yaml) and document viewing it with Redoc/Swagger UI via a one-liner docker command in README
- Contract tests: load spec with kin-openapi (test dependency only), validate the spec itself, then for each httptest case in B1-02 validate the response against the spec's response schema (openapi3filter)
- CI: spectral lint (`npx @stoplight/spectral-cli lint`) as a step in backend workflow (non-blocking warning level for style rules, blocking for errors)

**Out of scope**
- Code generation from the spec (deliberately not; ADR-0004 explains the hand-written approach for a 2-endpoint API)

**Implementation notes**
- Keep examples in the spec identical to README examples and golden test files — one truth.

**Acceptance criteria**
- [ ] Spec validates (spectral + kin-openapi loader)
- [ ] All existing httptest cases also pass schema validation
- [ ] Endpoint serves the spec with correct content type
- [ ] README API section links to the spec

**Tests**
- openapi_test.go (spec validity + response validation wrapper reused by handler tests)

**Files / areas**
- backend/api/openapi.yaml
- backend/internal/httpapi/openapi.go
- backend/internal/httpapi/openapi_test.go

**PR title:** `feat(backend): add OpenAPI 3.1 contract, serve it and validate responses against it in tests`

### B1-07 — Backend test hardening: integration suite, race/shuffle, golden fixtures, coverage gate script

**Labels:** backend, testing, sprint-1 · **Estimate:** M · **Depends on:** B1-05, B1-06

**Goal.** The backend test suite is the coverage report the assignment asks for: complete, reproducible and enforced.

**Scope**
- internal/httpapi/integration_test.go (build tag `integration` or plain, decide and document): starts the real `run()` on port 0 with env overrides, exercises the full middleware chain over a real TCP client: happy path, 422s, 429 burst, X-Request-ID echo, /metrics after traffic, graceful shutdown mid-request
- `scripts/coverage-check.sh` parsing `go tool cover -func` total, failing below 85 %; excludes cmd/ main wiring lines via `//go:build` or by testing run() itself
- `make cover-html` producing coverage.html; CI uploads both .out and .html as artifacts; job summary shows per-package table
- `go test -race -shuffle=on -count=1` in CI; `t.Parallel()` where safe
- README Testing section: how to run unit/fuzz/bench/integration, current numbers (filled by D4-01)

**Out of scope**
- e2e with the frontend (H3-04)
- Load testing (FU-07)

**Implementation notes**
- Integration tests should use `t.Setenv` and never bind fixed ports.
- Keep the total runtime under 10 s so nobody skips it.

**Acceptance criteria**
- [ ] `make -C backend test` prints total coverage ≥ 85 % and the gate script enforces it
- [ ] Integration test covers shutdown and rate limiting paths
- [ ] CI artifact contains coverage.html

**Tests**
- integration_test.go
- coverage-check script test (shell, trivial)

**Files / areas**
- backend/internal/httpapi/integration_test.go
- backend/scripts/
- backend/Makefile
- .github/workflows/backend.yml

**PR title:** `test(backend): add integration suite, race/shuffle runs and enforced coverage gate`


## Sprint 2 — Frontend core

### F2-01 — API layer: apiFetch + ApiError (problem+json aware), endpoints config, zod schemas, TanStack hooks, MSW handlers

**Labels:** frontend, sprint-2 · **Estimate:** M · **Depends on:** P0-03, B1-02

**Goal.** A single, typed, tested doorway to the backend that every component uses and no component bypasses.

**Scope**
- src/lib/api.ts: `apiFetch<T>(path, { method, body, signal, schema? })` over native fetch; base URL from getConfig().apiBaseUrl; JSON headers; AbortSignal.timeout(10s) default; parses `application/problem+json` into `ApiError { status, code, title, detail, requestId, errors[] }`; network failures → ApiError code=NETWORK; non-JSON 5xx → code=UNEXPECTED
- src/lib/query-config.ts: endpoint constants (CALCULATE_ENDPOINT, OPERATIONS_ENDPOINT), query keys, STALE_TIME
- src/types/calculator.ts: OperationName union, CalculateRequest/Response, OperationSpec, ProblemDetails; zod schemas (`calculateResponseSchema`, `operationsResponseSchema`, `problemSchema`) used by apiFetch to validate responses at runtime (fail closed with code=INVALID_RESPONSE)
- src/hooks/use-calculate.ts: `useCalculate()` → useMutation<CalculateResponse, ApiError, CalculateRequest>; src/hooks/use-operations.ts: useQuery with staleTime Infinity and a static fallback list so the UI never blocks on the network
- src/components/providers/QueryProvider.tsx: defaults retry: 0 for mutations, 1 for queries, refetchOnWindowFocus false
- src/test/msw/handlers.ts: handlers replicating the backend contract (incl. DIVISION_BY_ZERO, RATE_LIMITED, malformed) + `server.ts`; wired in setup.ts

**Out of scope**
- UI components
- State machine

**Implementation notes**
- Mirror the reference codebase's apiFetch shape (typed error class with status/code/requestId, envelope unwrapping) but keep only what this app needs.
- Export `isApiError(e)` type guard; map codes → user messages in `src/lib/error-messages.ts` (single table, tested).

**Acceptance criteria**
- [ ] apiFetch tests: success, problem+json 422 mapped to ApiError with code/errors, 429 with retryAfter, network error, invalid JSON body, schema mismatch
- [ ] Hooks tested with renderHook + QueryClient wrapper
- [ ] No `any`; ESLint clean

**Tests**
- api.test.ts
- error-messages.test.ts
- use-calculate.test.tsx
- use-operations.test.tsx

**Files / areas**
- frontend/src/lib/
- frontend/src/hooks/
- frontend/src/types/
- frontend/src/test/msw/
- frontend/src/components/providers/

**PR title:** `feat(frontend): add typed API layer with problem+json errors, zod validation and query hooks`

### F2-02 — Calculator engine + store: pure state machine (calculator-engine.ts) and zustand store with injected evaluate

**Labels:** frontend, sprint-2 · **Estimate:** L · **Depends on:** F2-01

**Goal.** All calculator behaviour lives in a pure, exhaustively unit-tested module; the store only wires it to the API.

**Scope**
- src/lib/calculator-engine.ts: `type CalcState = { phase: 'idle'|'enteringA'|'operatorSelected'|'enteringB'|'result'|'error'; display: string; a: number|null; b: number|null; operator: BinaryOp|null; error: string|null; pendingUnary?: ... }` and pure transition functions: inputDigit, inputDecimal, toggleSign, backspace, clearEntry, clearAll, setOperator, requestEvaluate → returns `{ state, effect?: { kind:'calculate', request: CalculateRequest } }`, applyResult, applyError, applyUnary (sqrt, percent as unary-with-context)
- Input rules: max 16 significant characters, single decimal point, no leading zeros (except '0.'), '-0' normalised, operator pressed twice replaces operator, operator right after result chains (result becomes `a`), Enter with no `b` reuses `a` (classic behaviour, documented), any digit after result starts fresh
- Percent semantics in the UI: `a op b %` → b becomes a*b/100 for +/−, and b/100 for ×/÷ (spreadsheet-calculator convention) OR simply call backend percent(a,b) — choose backend `percent` for `a % b` display path and document in ADR-0008
- src/stores/useCalculatorStore.ts: zustand store holding CalcState + actions; `evaluate` effect is executed by an injected `evaluator: (req) => Promise<CalculateResponse>` (set from the component via useCalculate) so the store is testable without React Query
- Selectors for display, phase, isBusy, error

**Out of scope**
- Rendering
- Keyboard mapping (F2-04)
- History (F2-05)

**Implementation notes**
- Model as explicit phase enum + exhaustive switch (`satisfies never`) so adding a phase fails the type-check.
- Every transition tested with a table: from-state × input → to-state; snapshot the sequence '12 + 7 = ' , '5 ÷ 0 =', '9 √', '200 + 15 %', '= = ='.

**Acceptance criteria**
- [ ] 100 % branch coverage on calculator-engine.ts
- [ ] Store test drives a full calculation with a fake evaluator (resolve + reject with ApiError) and asserts phases/display
- [ ] No React imports in engine

**Tests**
- calculator-engine.test.ts
- useCalculatorStore.test.ts

**Files / areas**
- frontend/src/lib/calculator-engine.ts
- frontend/src/stores/useCalculatorStore.ts
- docs/adr/0008-frontend-evaluation-semantics.md

**PR title:** `feat(frontend): add pure calculator state machine and zustand store`

### F2-03 — UI: Calculator, Display, Keypad, Key components; responsive layout, dark mode, accessibility

**Labels:** frontend, sprint-2 · **Estimate:** L · **Depends on:** F2-02

**Goal.** A calculator that looks deliberate, works on a phone, and is usable with a screen reader.

**Scope**
- src/components/calculator/Calculator.tsx (composition + wiring store ↔ useCalculate), Display.tsx (expression line + result line, auto-shrinking font via CSS clamp, `aria-live="polite"` region, error state styling), Keypad.tsx (CSS grid 4×5, operator column accent), Key.tsx (variant: digit|operator|action|equals, uses ui/button, `aria-label`, `aria-keyshortcuts`), OperationsBar for unary ops (√, %, x^y) driven by useOperations
- Layout: mobile-first; max-width 24rem centred; keys ≥ 44×44 px; safe-area insets; landscape phone check; no horizontal scroll at 320 px
- Theme: tokens from globals.css; dark mode via prefers-color-scheme; focus-visible rings; reduced-motion respected
- Busy state: keys disabled + subtle spinner in Display while a mutation is pending; requests are sequential (store ignores input while busy except Escape)
- `data-ui` attributes on every component root (`data-ui="calculator"`, `"calculator.display"`, …)
- src/app/App.tsx renders header (title + version from /version fetched lazily or build-time) and Calculator; ErrorBoundary provider

**Out of scope**
- Keyboard shortcuts (F2-04)
- History panel (F2-05)
- Number formatting (F2-06, use a stub formatter until then)

**Implementation notes**
- Design brief: quiet neutral surface, one accent for operators, distinct equals key, monospace tabular numerals for the display (`font-variant-numeric: tabular-nums`).
- Use Testing Library queries by role/label — this doubles as the a11y check.

**Acceptance criteria**
- [ ] Renders and performs 12 + 7 = 19 through the MSW-mocked API in a component test
- [ ] Division by zero shows the mapped message in the display error slot and clears on next input
- [ ] axe (vitest-axe) reports no violations on the initial render
- [ ] Lighthouse mobile a11y ≥ 95 locally (manual, note in PR)

**Tests**
- Calculator.test.tsx (user-event flows)
- Display.test.tsx
- Key.test.tsx
- a11y.test.tsx with vitest-axe

**Files / areas**
- frontend/src/components/calculator/
- frontend/src/app/
- frontend/src/components/ui/

**PR title:** `feat(frontend): build calculator UI with responsive layout, dark mode and accessible controls`

### F2-04 — Keyboard support and input-validation UX

**Labels:** frontend, sprint-2 · **Estimate:** S · **Depends on:** F2-03

**Goal.** Power users can drive the calculator without a mouse; invalid input is prevented rather than reported.

**Scope**
- src/hooks/use-keyboard.ts: window keydown mapping — digits, '.', '+', '-', '*', '/', '^', '%', Enter/'=', Escape (clear all), Backspace, Delete (clear entry), 'r' for √ (documented); ignores events with modifiers or when focus is in an input/textarea; prevents default only for handled keys
- Visual feedback: the matching Key gets a transient pressed state (store `lastKey` with timestamp or CSS animation via data attribute)
- Inline validation: attempting a second '.' or exceeding max digits is a no-op with a subtle shake on Display (respecting reduced-motion); operator right after operator replaces silently
- Server-side errors (422) render in Display error slot with the mapped message and the request id in a tooltip/title for support
- Network/timeout/429 errors show a non-blocking toast (sonner) with retry action; store returns to previous phase

**Out of scope**
- History
- Formatting

**Implementation notes**
- Keep the mapping table exported and tested as data; the hook test uses user-event.keyboard().

**Acceptance criteria**
- [ ] `user.keyboard('12+7{Enter}')` yields 19 in a component test
- [ ] Escape clears; Backspace edits; '%' and '^' route to the correct operations
- [ ] Typing into an unrelated input does not trigger the calculator

**Tests**
- use-keyboard.test.tsx
- keyboard flows in Calculator.test.tsx

**Files / areas**
- frontend/src/hooks/use-keyboard.ts
- frontend/src/components/calculator/

**PR title:** `feat(frontend): add keyboard support and input-validation feedback`

### F2-05 — History with localStorage persistence (versioned key, validated on read) and HistoryPanel

**Labels:** frontend, sprint-2 · **Estimate:** M · **Depends on:** F2-03

**Goal.** Recent calculations survive reloads and can be recalled, using a safe, versioned storage helper.

**Scope**
- src/lib/storage.ts: `createStorageKey('history', 1) → 'calc.history.v1'`, `readJSON(key, schema)` / `writeJSON(key, value)` / `remove(key)` wrapping localStorage in try/catch (private mode, quota), zod-validated on read (invalid → discard + console.warn once), SSR/no-window safe
- src/stores/useHistoryStore.ts: entries `{ id, operation, a, b, result, at }`, max 50 (FIFO), actions add/clear/remove; hydrated from storage on init; persisted on change (subscribe) — hand-rolled, not zustand/persist, matching the reference convention and keeping schema control explicit
- src/components/calculator/HistoryPanel.tsx: collapsible list (mobile: sheet/drawer via vaul; desktop: side column), each row shows `a op b = result`, click recalls result into Display as `a`, clear button with confirm
- Calculator wires successful mutation → history.add

**Out of scope**
- Server-side history (FU-04)

**Implementation notes**
- Storage key version bump policy in a comment: bump when the schema changes; old keys are removed by a tiny migration list.

**Acceptance criteria**
- [ ] Entries persist across a simulated reload (re-create store, read from mocked localStorage)
- [ ] Corrupted JSON in storage does not crash the app
- [ ] Recall flow tested with user-event
- [ ] Max 50 enforced

**Tests**
- storage.test.ts
- useHistoryStore.test.ts
- HistoryPanel.test.tsx

**Files / areas**
- frontend/src/lib/storage.ts
- frontend/src/stores/useHistoryStore.ts
- frontend/src/components/calculator/HistoryPanel.tsx

**PR title:** `feat(frontend): persist calculation history in localStorage with a history panel`

### F2-06 — Number formatting and parsing helpers (display precision, grouping, exponent)

**Labels:** frontend, sprint-2 · **Estimate:** S · **Depends on:** F2-02

**Goal.** Results are shown the way a calculator user expects, without lying about the underlying value.

**Scope**
- src/lib/format-number.ts: `formatResult(n, { maxSignificant: 12 })` → trims float noise (0.30000000000000004 → 0.3), switches to exponent beyond 1e15 / below 1e-6, handles -0, Infinity/NaN never reach here (guarded upstream, throw if they do), `formatEntry(raw)` adds thousands grouping to the in-progress input without touching the trailing '.', `parseEntry(raw)` → number
- Locale: use `Intl.NumberFormat('en-US')` explicitly (ADR-0007) — locale switch is a follow-up (FU-05)
- Display integration: full-precision value available on hover/title and copied on click (copy-to-clipboard with toast)

**Out of scope**
- i18n

**Implementation notes**
- Property-style tests: for random doubles, `Number(formatResult(x))` is within 1e-12 relative error of x.

**Acceptance criteria**
- [ ] Table tests for the documented examples in ADR-0007
- [ ] Display shows 0.3 for 0.1+0.2 while the API response (visible in history tooltip) shows the raw value

**Tests**
- format-number.test.ts

**Files / areas**
- frontend/src/lib/format-number.ts
- docs/adr/0007-display-precision.md

**PR title:** `feat(frontend): add number formatting helpers with documented display precision`

### F2-07 — Frontend test hardening: coverage thresholds, CI summary, a11y and MSW conventions documented

**Labels:** frontend, testing, sprint-2 · **Estimate:** S · **Depends on:** F2-04, F2-05, F2-06

**Goal.** The frontend suite is enforced at the same standard as the backend and readable as documentation.

**Scope**
- vitest.config: coverage provider v8, thresholds lines 85 / branches 80 / functions 85, `include: src/**`, exclude test/ and types-only files, reporters text + lcov + json-summary
- CI: post the json-summary as a table in $GITHUB_STEP_SUMMARY; upload lcov artifact
- frontend/README.md Testing section: layers (engine unit → hooks → components with MSW → e2e), how to run/watch, how to add MSW handlers
- Fill any gaps revealed by the threshold (target: every user-facing path has a component test)

**Out of scope**
- Playwright e2e (H3-04)

**Implementation notes**
- Do not chase 100 %; chase every branch that a reviewer might ask about (errors, busy state, storage failures).

**Acceptance criteria**
- [ ] `npm run test:coverage` passes thresholds
- [ ] CI summary shows the table
- [ ] README testing section complete

**Tests**
- as needed to reach thresholds

**Files / areas**
- frontend/vitest.config.ts
- frontend/README.md
- .github/workflows/frontend.yml

**PR title:** `test(frontend): enforce coverage thresholds and document the testing strategy`


## Sprint 3 — Production hardening

### H3-01 — Backend container: multi-stage distroless image, non-root, static binary, healthcheck subcommand

**Labels:** backend, platform, sprint-3 · **Estimate:** S · **Depends on:** B1-03

**Goal.** A minimal, reproducible, secure backend image.

**Scope**
- backend/Dockerfile: `golang:1.27-alpine` builder with module cache mounts (`--mount=type=cache`), `CGO_ENABLED=0 GOFLAGS=-trimpath`, `-ldflags='-s -w -X …version'` from build args; final `gcr.io/distroless/static-debian12:nonroot`; EXPOSE 8081; `USER nonroot`
- `server -healthcheck` subcommand that GETs /readyz and exits 0/1 (distroless has no curl) used by Docker HEALTHCHECK and compose
- backend/.dockerignore; image labels (org.opencontainers.image.*)
- Makefile: `docker-build`, `docker-run`; document expected image size (< 15 MB) in README

**Out of scope**
- Compose (H3-03)
- Registry publishing (H3-07)

**Implementation notes**
- Build must not require network at runtime; verify `docker run --network none` still serves.

**Acceptance criteria**
- [ ] `docker build` succeeds with BuildKit; `docker run -p 8081:8081 image` answers /healthz
- [ ] `docker inspect` shows non-root user and healthcheck
- [ ] Image size recorded in PR

**Tests**
- Shell smoke in CI later (H3-03); local manual verification documented in PR

**Files / areas**
- backend/Dockerfile
- backend/.dockerignore
- backend/cmd/server/healthcheck.go

**PR title:** `build(backend): add distroless multi-stage Dockerfile with healthcheck subcommand`

### H3-02 — Frontend container: Vite build served by unprivileged nginx with /api reverse proxy and hardened config

**Labels:** frontend, platform, sprint-3 · **Estimate:** M · **Depends on:** F2-03

**Goal.** Production-shaped static hosting with same-origin API access, so no CORS is needed and the API is never exposed directly.

**Scope**
- frontend/Dockerfile: `node:22-alpine` deps/build stages with npm cache mount → `nginxinc/nginx-unprivileged:alpine-slim`; listens on 8080; copies dist/ and nginx/nginx.conf + templates
- nginx config: SPA fallback (`try_files … /index.html`), immutable cache for hashed assets, `no-cache` for index.html, gzip/brotli if available, security headers (CSP allowing self + inline style hash if needed, X-Content-Type-Options, Referrer-Policy, Permissions-Policy), `/api/` → `proxy_pass http://backend:8081` with `proxy_set_header X-Request-ID $request_id` when absent, `/healthz` static 200
- Runtime configurability: `BACKEND_UPSTREAM` env consumed via nginx envsubst templates so the same image works in compose and elsewhere; frontend default `VITE_API_BASE_URL=/api` baked at build
- frontend/.dockerignore

**Out of scope**
- TLS termination (documented as ingress responsibility)

**Implementation notes**
- Verify CSP does not break Vite output (no inline scripts by default; style handling via Tailwind is fine).

**Acceptance criteria**
- [ ] `docker run` serves the app; `/api/v1/operations` proxied when backend reachable
- [ ] Security headers present (curl -I)
- [ ] Image runs as non-root (id 101)

**Tests**
- Manual + compose smoke (H3-03)

**Files / areas**
- frontend/Dockerfile
- frontend/nginx/
- frontend/.dockerignore

**PR title:** `build(frontend): add nginx-based production image with API reverse proxy`

### H3-03 — Compose stack, smoke script, Make targets, and CI job that builds and smoke-tests the images

**Labels:** platform, testing, sprint-3 · **Estimate:** M · **Depends on:** H3-01, H3-02

**Goal.** `docker compose up` gives a reviewer the whole application in one command, and CI proves it every PR.

**Scope**
- compose.yaml: services backend (build ./backend, healthcheck via `-healthcheck`, env LOG_FORMAT=json, read_only rootfs, `cap_drop: [ALL]`, `security_opt: no-new-privileges`, mem/cpu limits) and frontend (build ./frontend, depends_on backend condition service_healthy, ports 8080:8080); named network; no host port for backend by default (override file exposes 8081 for local API poking)
- compose.override.example.yaml → documented copy step for dev-time overrides
- scripts/smoke.sh: waits for http://localhost:8080/healthz, asserts operations list, 12+7=19 via the proxy, division-by-zero 422 with code, X-Request-ID present, frontend index served; exits non-zero on failure
- Root Makefile: `up`, `down`, `logs`, `smoke`; README Quick start path #1 = compose
- .github/workflows/stack.yml: builds both images (buildx, GHA cache), `docker compose up -d --wait`, runs smoke.sh, dumps logs on failure

**Out of scope**
- Playwright (H3-04)
- Publishing (H3-07)

**Implementation notes**
- `--wait` relies on healthchecks; ensure both services define them.
- Keep smoke.sh POSIX sh + curl + jq only.

**Acceptance criteria**
- [ ] Clean machine: `git clone && docker compose up` → app usable at :8080
- [ ] `make smoke` green locally and in CI
- [ ] Backend not reachable from host unless override applied

**Tests**
- scripts/smoke.sh in CI

**Files / areas**
- compose.yaml
- compose.override.example.yaml
- scripts/smoke.sh
- Makefile
- .github/workflows/stack.yml

**PR title:** `build: add compose stack with hardened services, smoke script and CI stack job`

### H3-04 — End-to-end tests with Playwright against the composed stack

**Labels:** testing, frontend, sprint-3 · **Estimate:** M · **Depends on:** H3-03, F2-07

**Goal.** Prove the real browser + real nginx + real Go service path works, including persistence and error UX.

**Scope**
- e2e/ as its own npm package (playwright/test), config with baseURL http://localhost:8080, chromium + mobile Safari emulation project, trace on first retry, HTML report
- Specs: happy path (click + keyboard) 12 + 7 = 19; chained operations; unary sqrt/percent; division by zero shows error and recovers; history persists across reload and recall works; responsive: no horizontal overflow at 320×568; a11y smoke with @axe-core/playwright
- Test-only hook: backend env RATE_LIMIT_RPS raised in compose override for e2e to avoid 429 flakiness; a dedicated spec asserts the 429 toast with a low limit override (optional, mark @slow)
- CI: extend stack.yml (or new e2e.yml) to run Playwright after smoke; upload report + traces as artifacts on failure
- Root Makefile `e2e` target

**Out of scope**
- Visual regression (follow-up)

**Implementation notes**
- Select by role/label; never by CSS class. Use `data-ui` only as a last resort.

**Acceptance criteria**
- [ ] All specs green locally and in CI
- [ ] Report artifact uploaded on failure
- [ ] Total e2e runtime < 3 min in CI

**Tests**
- e2e/tests/*.spec.ts

**Files / areas**
- e2e/
- .github/workflows/
- Makefile

**PR title:** `test(e2e): add Playwright suite running against the compose stack in CI`

### H3-05 — Security and supply chain: CodeQL, Trivy image scan, gitleaks, npm audit, SECURITY.md, pinned actions audit

**Labels:** platform, sprint-3 · **Estimate:** S · **Depends on:** H3-03

**Goal.** Demonstrate the routine security posture of a production service without turning the repo into a compliance exercise.

**Scope**
- .github/workflows/security.yml: CodeQL (go, javascript-typescript) on PR + weekly; Trivy scan of both built images (fail on CRITICAL, ignore unfixed), `npm audit --audit-level=high` (frontend + e2e), govulncheck already in backend.yml (leave)
- gitleaks action on PRs; `.gitleaks.toml` allowlist for fixtures if needed
- SECURITY.md (reporting, supported versions), threat-model paragraph in docs/ARCHITECTURE.md (no auth by design; API is a stateless pure function; abuse controls = rate limit + body size + timeouts)
- Audit all workflows: every `uses:` pinned to a SHA with a version comment; `permissions:` least privilege at workflow level

**Out of scope**
- SAST beyond CodeQL
- Signing images (cosign) — mention as follow-up

**Implementation notes**
- Keep scans non-flaky: cache Trivy DB; set timeouts.

**Acceptance criteria**
- [ ] Security workflow green on main
- [ ] SECURITY.md present and linked from README
- [ ] Dependabot + CodeQL visible under repository Security tab

**Tests**
- n/a

**Files / areas**
- .github/workflows/security.yml
- SECURITY.md
- docs/ARCHITECTURE.md

**PR title:** `ci: add CodeQL, Trivy, gitleaks and audit steps; add SECURITY.md`

### H3-06 — Release pipeline: tag-driven multi-arch images to GHCR, SBOM, release notes

**Labels:** platform, sprint-3 · **Estimate:** M · **Depends on:** H3-03, H3-05

**Goal.** A reviewer can `docker compose pull` prebuilt images, and the project has a real release story.

**Scope**
- .github/workflows/release.yml on `v*` tags: buildx multi-arch (linux/amd64, linux/arm64) for backend and frontend → ghcr.io/isasumer/go-react-calculator/{backend,frontend}:{version,latest}; OCI labels; provenance/SBOM via `docker/build-push-action` attestations or syft; attach SBOMs to the GitHub Release
- Release notes generated from Conventional Commits (git-cliff config) into CHANGELOG.md; GitHub Release created with the changelog section
- compose.yaml supports `IMAGE_TAG` env to use published images instead of building (documented: `IMAGE_TAG=v1.0.0 docker compose up --no-build`)
- Version injected into backend `/version` and frontend header from the tag

**Out of scope**
- Deploy to any cloud (out of scope; FU-08 covers k8s manifests)

**Implementation notes**
- Test with a pre-release tag `v0.9.0-rc.1` before Sprint 4 cuts v1.0.0.

**Acceptance criteria**
- [ ] Pushing a tag produces two multi-arch images on GHCR and a Release with notes and SBOMs
- [ ] `IMAGE_TAG=… docker compose up --no-build` works on a clean machine

**Tests**
- Dry run with rc tag

**Files / areas**
- .github/workflows/release.yml
- cliff.toml
- compose.yaml
- CHANGELOG.md

**PR title:** `ci: add tag-driven release pipeline publishing multi-arch images and SBOMs to GHCR`


## Sprint 4 — Docs, release & submission

### D4-01 — README final: quick starts, API reference with curl/httpie examples, design decisions, testing numbers, time log

**Labels:** docs, sprint-4 · **Estimate:** M · **Depends on:** H3-06, B1-07, F2-07

**Goal.** The README is the deliverable most reviewers read first and possibly only; it must answer every question in the assignment brief within two screens and link deeper.

**Scope**
- Sections (from the skeleton): badges; one-paragraph overview + screenshot/GIF; architecture diagram (mermaid); Quick start ×3 (compose prebuilt, compose build, local dev with Go + Node); API reference: every endpoint with request/response, `curl` and `httpie` examples copy-pasted from golden fixtures, error catalogue table linking docs/errors.md, link to OpenAPI + Redoc one-liner; Design decisions: 9 ADR one-liners with links, plus 'What the assignment asked vs what I added and why'; Testing: how to run each layer, current coverage numbers (backend per-package table, frontend summary), fuzz/bench/e2e; Project structure tree; Configuration table (env vars); Time log (honest, per sprint); Prompts link; License
- Screenshots: light + dark, mobile + desktop (store under docs/images, optimised)
- Verify every command in README by running it on a fresh clone in a temp dir (record in PR)

**Out of scope**
- Marketing tone; keep it technical and short

**Implementation notes**
- Coverage numbers must be copied from CI artifacts of the merge commit, not typed from memory.
- Keep the 'assignment vs added' table brutally honest; it is the single most important paragraph for a senior reviewer.

**Acceptance criteria**
- [ ] Every README command executes successfully on a fresh clone
- [ ] All four mandatory README items from the brief are present and findable in the TOC
- [ ] Screenshots current

**Tests**
- Fresh-clone script run (documented in PR body)

**Files / areas**
- README.md
- docs/images/

**PR title:** `docs: finalise README with quick starts, API reference, design decisions and testing report`

### D4-02 — PROMPTS.md consolidation and AI-usage reflection

**Labels:** docs, sprint-4 · **Estimate:** S · **Depends on:** D4-01

**Goal.** Turn the per-session prompt log into the deliverable the brief asks for, showing judgment rather than just usage.

**Scope**
- Normalise all session entries in docs/PROMPTS.md: heading per ticket, prompts verbatim, tool used (Claude Code), what was accepted, what was rejected and why, what was written by hand
- Add a short front section: how AI was used (planning, scaffolding, test generation, review), guardrails (every generated line read and run; no generated ADR text accepted without editing), and 2–3 concrete examples where the AI suggestion was wrong and how it was caught
- Link from README 'Prompts' section

**Out of scope**
- Publishing full transcripts (prompts only, as asked)

**Implementation notes**
- Nothing fabricated; if a session forgot to log, say so.

**Acceptance criteria**
- [ ] Every merged PR has a corresponding PROMPTS.md section
- [ ] Reflection section present and honest

**Tests**
- n/a

**Files / areas**
- docs/PROMPTS.md
- README.md

**PR title:** `docs: consolidate prompt log and add AI-usage reflection`

### D4-03 — ARCHITECTURE.md and ADR index: request lifecycle, middleware order, error taxonomy, frontend state machine diagram

**Labels:** docs, sprint-4 · **Estimate:** S · **Depends on:** B1-07, F2-07

**Goal.** A reviewer preparing an interview can understand the system in ten minutes without opening code.

**Scope**
- docs/ARCHITECTURE.md: component diagram, backend request lifecycle with middleware order and where each error status originates, error taxonomy table (code → status → origin → client behaviour), frontend data flow (store → effect → mutation → apiFetch), state-machine diagram (mermaid stateDiagram) of calculator phases, storage schema and versioning policy, operational notes (config, probes, metrics, shutdown sequence), threat model paragraph, known limitations
- docs/adr/README.md index with status per ADR; ensure ADR-0001…0009 exist and match the code (fix drift)
- Cross-link from README

**Out of scope**
- Duplicating README content

**Implementation notes**
- Diagrams as mermaid so they render on GitHub.

**Acceptance criteria**
- [ ] All 9 ADRs present, status Accepted, consistent with implementation
- [ ] ARCHITECTURE.md renders with diagrams on GitHub

**Tests**
- n/a

**Files / areas**
- docs/ARCHITECTURE.md
- docs/adr/

**PR title:** `docs: add architecture document and complete the ADR index`

### D4-04 — Release v1.0.0 and clean-machine verification

**Labels:** platform, sprint-4 · **Estimate:** S · **Depends on:** D4-01, D4-02, D4-03

**Goal.** Cut the version the reviewers will see and prove it works from nothing.

**Scope**
- Tag v1.0.0 from main; verify release workflow output (images, SBOMs, notes)
- Clean-machine check (fresh WSL distro or `docker run -v` sandbox / temporary directory with cleared caches): follow README quick start #1 and #3 literally; fix anything that breaks (separate PRs, then re-tag as v1.0.1 if needed)
- Run the full checklist from docs/SUBMISSION_CHECKLIST.md (created here): gofmt/vet/lint clean, coverage artifacts present, no secrets, no TODOs, LICENSE, commit history tidy, repo description/topics set (go, react, typescript, calculator, rest-api, docker)

**Out of scope**
- Feature changes

**Implementation notes**
- Repository topics and About link (to the deployed demo if any) are set via `gh repo edit`.

**Acceptance criteria**
- [ ] v1.0.x release exists with images
- [ ] Checklist fully ticked in the PR/issue
- [ ] Fresh-clone run recorded

**Tests**
- Checklist

**Files / areas**
- docs/SUBMISSION_CHECKLIST.md

**PR title:** `chore(release): v1.0.0`

### D4-05 — Submission package: recruiter email draft, repository link, prompts summary, follow-up issues triaged

**Labels:** docs, sprint-4 · **Estimate:** S · **Depends on:** D4-04

**Goal.** Hand over cleanly. The email is drafted here and sent by Isa manually.

**Scope**
- Draft email (English) with: repo link, one-paragraph summary, how to run in one command, where to find tests/coverage/prompts, honest time note, availability for walkthrough
- Ensure all open issues are either closed or labelled follow-up in the Backlog milestone with a one-line rationale so the open-issue list reads as intentional
- Pin the epic issue and the README in the repo; set the repo's About text
- Record submission in the job-hunt tracker (outside this repo)

**Out of scope**
- Sending the email (manual)

**Implementation notes**
- No company name inside the repository; keep it in the email only.

**Acceptance criteria**
- [ ] Email draft reviewed
- [ ] Open issues = follow-ups only
- [ ] Tracker updated

**Tests**
- n/a

**Files / areas**
- (outside repo) job-hunt tracker; docs/SUBMISSION_CHECKLIST.md tick

**PR title:** `n/a (no code change)`


## Backlog — Post-submission follow-ups

Intentionally not in v1. Each is filed as an issue so the open-issue list reads as a roadmap, not as unfinished work.

| ID | Title | Why deferred |
|---|---|---|
| FU-01 | Expression evaluation endpoint and UI mode (`POST /api/v1/evaluate` with precedence, shunting-yard) | Extends the API with an `expression` string, parsed and evaluated server-side; frontend gets an expression-entry mode. Deliberately excluded from v1 (ADR-0008) because the brief asks for basic operations and warns against extra features. |
| FU-02 | Decimal arithmetic mode (shopspring/decimal) selectable per request | In a money context float64 is unacceptable; add `precision: "decimal"` request option backed by decimal arithmetic with scale/rounding rules, and document exactness guarantees. Excluded from v1 per ADR-0003. |
| FU-03 | OpenTelemetry tracing + compose `observability` profile (Prometheus, Grafana, Tempo) with a starter dashboard | Traces across nginx → Go with W3C context; dashboard JSON committed; `docker compose --profile observability up`. |
| FU-04 | Server-side calculation history with Postgres and Idempotency-Key | Persist calculations per anonymous device id; idempotent POST via `Idempotency-Key` header; migrations; history sync in the frontend. Adds state to a deliberately stateless service — only if a product need appears. |
| FU-05 | Locale-aware number formatting and language switch | Intl-based formatting per user locale, decimal comma input, RTL check. |
| FU-06 | Offline fallback: local evaluation when the API is unreachable, clearly labelled | Service worker + local engine; results tagged 'computed locally' — must not silently diverge from the API contract. |
| FU-07 | Load test and performance notes (k6 script, p50/p99 at N rps, GOMAXPROCS/limits) | Document baseline numbers and rate-limit tuning. |
| FU-08 | Kubernetes manifests (Kustomize) with probes, HPA, PDB, NetworkPolicy and resource limits | Turns the compose stack into a deployable k8s app; matches how the service would actually run in production. |
| FU-09 | Separate admin listener for /metrics and probes | Move operational endpoints off the public port to reduce exposure. |
| FU-10 | Component style guide page (like a `/dev/ui` route) and visual regression tests | Living style guide of ui primitives and calculator components; Playwright screenshot comparisons. |
| FU-11 | Sign container images with cosign and verify in compose/k8s | Supply-chain hardening beyond SBOM. |


## 4. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Reviewer reads "2–4 h" and sees 30 h of work as ignoring instructions | Scope-judgment score | Product surface unchanged; README "assignment vs added" table + honest time log; Sprint-2 checkpoint is a legitimate 4-hour-shaped submission by itself |
| Go idiom mistakes (I have not shipped Go in production) | Core signal | Zero-magic stdlib design; golangci-lint with strict linters; every generated line read and run; fuzz + race; ask the AI *why* on every non-obvious construct and log it in PROMPTS.md |
| Over-engineering perception (metrics, rate limiting on a calculator) | Judgment | Each extra is an ADR with a one-line "why here"; all extras are infra, none are features; nothing added that a real deployment would not need |
| Float display surprises (0.1+0.2) | Correctness perception | ADR-0003/0007; API returns raw double, UI formats; tests document it |
| Sessions drift from the plan (fresh context each time) | Consistency | CLAUDE.md bootstrap, frozen API contract, issue bodies self-contained, `make check` as the single gate, follow-up issues instead of scope creep |
| CI flakiness (e2e, rate limit) | Trust in the badge | Health-gated compose `--wait`, raised rate limit in e2e overrides, retries with traces |
| Timeline overrun | Submission delay | Sprint 2 checkpoint is submittable; Sprint 3/4 tickets can be cut individually without breaking anything |

## 5. Submission checklist (executed in D4-04)

- gofmt/gofumpt, vet, golangci-lint clean; ESLint/Prettier/type-check clean
- Coverage artifacts from the release commit copied into README (backend per package, frontend summary)
- README commands verified on a fresh clone; `docker compose up` verified from a clean image cache
- No secrets, no `TODO`, no committed build output, `.env*` ignored
- Every merged PR has a PROMPTS.md section; PROMPTS.md reflection written
- All 9 ADRs Accepted and consistent with code; ARCHITECTURE.md diagrams render
- Open issues are only `follow-up` in the Backlog milestone
- Repo: description, topics, About link, pinned epic; LICENSE present; commit author matches the GitHub account
- v1.0.x release with images + SBOMs; `IMAGE_TAG=v1.0.x docker compose up --no-build` works
- Recruiter email drafted (sent manually); job-hunt tracker updated

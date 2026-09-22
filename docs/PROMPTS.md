# Prompts log

The assignment asks for the prompts used. This file is appended by every working session, verbatim, with
what was accepted, what was rejected and why, and what was written by hand. Tool: Claude Code (Anthropic).
Prompts were written in Turkish; an English gloss follows each one in brackets. The company name is
redacted as `[company]` because this repository is public.

Format for each session:

```
## Session <ticket-id> — <YYYY-MM-DD>
1. <prompt, verbatim>
   [English gloss]
**Accepted** — …
**Rejected (why)** — …
**Written by hand** — …
```

---

## Session PLANNING — 2026-09-22

1. `bana [company] dan böyle bir metin geldi analiz et` + the assignment text pasted verbatim.
   [I received this text from [company], analyse it.]
2. `go yu kur ama ben önce end to end bir epic oluşturmanı istiyorum. yani önce bir tane plan ve sprint oluşturmamız lazım. bir github reposu oluşturmamız lazım. Daha sonra hem fe hem de backend için ticketlar sub ticket lar follw up lar açmamız lazım . yani önce planlama gerekiyor. ayrıca front-end için bununla aynı dosyada olan lived projesi baz alınabilir dosyalama UI, helpers, api callings, state management, local storage etc. backend için production grade solution lazım. yani sadece bu task i implement etmeyeceğiz senior olduğumuzu göstermemiz gerekiyor. Benim aklıma gelmeyen ve veya yapmamız gereken implementasyon planlama vs varsa ilerletelim. Her feature ı section ı veya ticket ı yeni pr da ve yeni session da implement edeceğim. dolayısıyla her session taze scope ile başlamış olacak. Senden detaylı implementasyon planı istiyorum ingilizce olsun`
   [Install Go, but first create an end-to-end epic: a plan and sprints, a GitHub repository, tickets / sub-tickets / follow-ups for both frontend and backend. Base the frontend structure (files, UI, helpers, API calls, state management, local storage) on an existing production project of mine. The backend must be a production-grade solution; we are not just implementing the task, we are demonstrating seniority. Add anything I have not thought of. Every ticket will be implemented in a new PR and a new session with fresh scope. Give me a detailed implementation plan in English.]

**Accepted** — The AI's assessment of what the assignment actually evaluates; the sprint structure with a
"submittable checkpoint" at the end of Sprint 2; the frozen API contract; the decision to use Go despite no
production Go experience; the ADR list; the per-issue session protocol captured in `CLAUDE.md`; the 40
tickets and 6 epics generated from a single manifest so issues and `docs/PLAN.md` cannot drift.

**Rejected (why)** — Nothing was rejected outright. The plan was shaped by the second prompt: the AI's
first analysis proposed a 4-hour build; the direction was changed to production-grade with the extra
investment confined to operational maturity rather than features, and the frontend conventions were
pinned to the author's existing codebase instead of the AI's defaults.

**Written by hand** — The direction and constraints above. The plan text was generated and reviewed
before commit; later sessions refine it only through follow-up issues.

## Session P0-01 — 2026-09-22

1. `issue #1'i implement et`
   [Implement issue #1.]

**Accepted** — Everything in the issue scope as generated: `.editorconfig`, MIT `LICENSE`, `CODEOWNERS`,
PR and issue templates carrying the Definition of Done, root `Makefile` whose targets fail fast naming the
ticket that will provide them, ADR template + ADR-0001 (monorepo) + ADR-0002 (frontend stack), ADR index,
`errors.md` and `ARCHITECTURE.md` stubs, this prompts log, README links, `CLAUDE.md` refinements.

**Rejected (why)** — n/a. The session was checked against the issue's acceptance criteria and the
out-of-scope list (no application code, no CI workflows).

**Written by hand** — None in this session; review happens on the PR before merge.

## Session P0-02 — 2026-09-22

1. ```
   Implement GitHub issue #2 (P0-02: Backend scaffold) in this repository.

   Start by reading CLAUDE.md, then `gh issue view 2`, then docs/PLAN.md §1 and §2 only. Do not explore the rest of the repo beyond the paths the issue lists.

   Constraints for this ticket:
   - Go 1.27 toolchain is at ~/.local/go/bin (on PATH via ~/.bashrc). Module path: github.com/isasumer/go-react-calculator/backend.
   - Standard library only for the placeholder server. Use the run(ctx, args, getenv, stdout) error pattern from the issue; main() only calls run() and exits non-zero on error.
   - Create the empty packages with doc.go files exactly as listed (internal/calc, internal/httpapi, internal/middleware, internal/config, internal/observability). No routes, no handlers, no config parsing yet — those are B1-02 and B1-03.
   - .golangci.yml with the linters listed in the issue; backend/Makefile with fmt, fmt-check, lint, vet, test (-race -shuffle=on -coverprofile), cover, coverage-check (fails below 85 %), vuln (govulncheck), build (ldflags for version/commit/date), run, check, tools. Pin tool versions.
   - Write the smoke test for run() (port 0, cancel context, returns nil).
   - Root `make check` must now succeed for the backend part and still fail fast for the frontend part with the P0-03 message.

   Finish by: running `make -C backend check` and pasting the output into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session P0-02 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #2; opening the PR with `gh pr create` using the PR title from the issue and `Closes #2`. Do not merge. Report the PR URL.
   ```

**Accepted** — `go.mod` (`go 1.27`) and the five `doc.go` packages. `cmd/server` with `run(ctx, args, getenv,
stdout) error`: `signal.NotifyContext` for SIGINT/SIGTERM, a single `-addr` flag (default `:8081`) so tests
can bind port 0, `net.ListenConfig.Listen` so the listener respects ctx (the `noctx` linter asks for this),
`http.NotFoundHandler`, `ReadHeaderTimeout` (gosec G112), JSON slog, a `listening` log line with the bound
address and build info, graceful `Shutdown` with a timeout derived from `context.WithoutCancel`. Smoke test
reads the first log line through an `io.Pipe`, sends a GET that returns 404, cancels, and expects `nil`. A
table test covers the flag and listen error paths. Tool versions are pinned in a separate `tools/` module
using `tool` directives (golangci-lint v2.13.2, gofumpt v0.12.0, govulncheck v1.8.0), so none of their
dependencies reach the service's `go.mod`. `make tools` builds them into `bin/` with `go install tool`.
`scripts/coverage-check.sh` parses `go tool cover -func`.

**Rejected (why)** — Adding the tools to the service's own `go.mod` with `tool` directives: that pulls
hundreds of lint dependencies into the service's module graph and govulncheck scope. `gosimple` as a
separate linter: in golangci-lint v2 it is merged into `staticcheck`, and `gofumpt` is configured under
`formatters`. The config says so. gofumpt `extra-rules`: deprecated in the pinned version, and it would
make `make fmt` and lint disagree. `govulncheck` in `make check`: it needs network access to the vuln DB,
so it stays a separate `make vuln` target (CI hardening in H3-05). Reading `getenv` for the listen address:
config parsing belongs to B1-03, so the parameter is `_` for now.

**Written by hand** — None. Every file was generated, read, and run locally (`make check`, `make build`,
the binary run by hand with SIGINT, `make vuln`, and fmt-check and coverage-check run against deliberately
failing inputs).

## Session P0-03 — 2026-09-22

1. Prompt (verbatim):

   ```text
   Implement GitHub issue #3 (P0-03: Frontend scaffold) in this repository.

   Start by reading CLAUDE.md, then `gh issue view 3`, then docs/PLAN.md §1 and §2 and docs/adr/0002-frontend-stack.md. Do not explore beyond the paths the issue lists.

   Constraints for this ticket:
   - Node 22 via nvm. Scaffold with `npm create vite@latest frontend -- --template react-ts`, then pin engines and add .nvmrc.
   - tsconfig: strict, noUncheckedIndexedAccess, exactOptionalPropertyTypes, alias `@/* → src/*` mirrored in vite.config and vitest.config.
   - Tailwind v4 via @tailwindcss/vite; src/app/globals.css with @theme tokens and a prefers-color-scheme dark variant. shadcn init (new-york) and add only the `button` primitive; src/lib/utils.ts with cn().
   - ESLint flat config (typescript-eslint recommended-type-checked, react, react-hooks, jsx-a11y, eslint-config-prettier last) and Prettier (semi, double quotes, width 100, trailingComma all).
   - Vitest + jsdom + @testing-library/react + jest-dom + user-event + msw; src/test/setup.ts wires jest-dom, MSW server lifecycle and localStorage reset. Include src/**/*.test.{ts,tsx}.
   - Folder skeleton with .gitkeep or a one-line doc comment: app/, components/{ui,providers,shared,calculator}, hooks/, lib/, stores/, types/, test/msw. No calculator code, no API layer, no stores — those are Sprint 2.
   - src/config.ts: typed getConfig() reading import.meta.env with VITE_API_BASE_URL defaulting to "/api", validated once, with a test for default and override.
   - package.json scripts: dev, build, preview, lint, type-check, test, test:coverage, format, format:check. frontend/Makefile with dev, lint, type-check, test, check mirroring them so the root `make check` works.
   - frontend/README.md conventions section: PascalCase feature components, kebab-case ui primitives and hooks, data-ui root attribute, co-located tests.
   - Placeholder App.test.tsx renders the shell. `npm run build` must produce dist/ with hashed assets.

   Finish by: running `make -C frontend check` and `npm run build` and pasting the output into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session P0-03 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #3; opening the PR with `gh pr create` using the PR title from the issue and `Closes #3`. Do not merge. Report the PR URL.
   ```

**Accepted** — Vite 8 / React 19 / TypeScript 6 scaffold with the template demo assets removed; tsconfig
strictness flags and the `@/*` alias (tsconfig, `vite.config.ts`, `vitest.config.ts`); Tailwind v4 tokens
in `src/app/globals.css` (surface, text, accent, danger, key/operator/function keys) with shadcn names
aliased onto them and dark mode driven by `prefers-color-scheme` for both tokens and `dark:` utilities;
shadcn `button` (new-york); `cn()`; ESLint flat config + Prettier; Vitest/jsdom/RTL/jest-dom/user-event/MSW
with `onUnhandledRequest: "error"` and storage reset; `getConfig()` split into a pure `parseConfig()` plus a
cached getter, rejecting protocol-relative and non-http(s) base URLs; `make check` = format-check → lint →
type-check → tests with coverage thresholds at the Definition-of-Done values (85 % lines / 80 % branches).
Dark/light switching was verified by screenshotting `vite preview` in headless Chromium with each colour
scheme forced.

**Rejected (why)**
- The template's `oxlint` setup — the ticket specifies ESLint with type-checked rules; removed.
- ESLint 10 (what `npm install eslint` resolves) — `eslint-plugin-jsx-a11y` and `eslint-plugin-react`
  only declare peer support up to ESLint 9; pinned `eslint@^9` instead of `--legacy-peer-deps`. Follow-up #50.
- `shadcn init` as a command — the current CLI is preset-based and no longer takes a `new-york` style
  option, so `components.json` was written with the fields `init` produces for new-york and then
  `shadcn add button` was run.
- The CLI's output for `button`: it rewrote the utils import to `from "cn"` and **installed an unrelated npm
  package called `cn`**. Uninstalled it and pointed the import at `@/lib/utils`. Lesson: read the
  `package.json` diff after every generator run.
- Keeping the generated button file byte-identical — it was reformatted by Prettier and given
  `import type * as React` to satisfy `consistent-type-imports`.

**Written by hand** — None; every generated file was read and `make -C frontend check` / `npm run build`
were executed locally before commit.

## Session P0-04 — 2026-09-22

1. Prompt (verbatim):

   ```text
   Implement GitHub issue #4 (P0-04: CI v1) in this repository. Prerequisites #2 and #3 are merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 4`, then docs/PLAN.md §2, backend/Makefile and frontend/Makefile (to reuse their targets rather than duplicating commands in YAML).

   Constraints for this ticket:
   - .github/workflows/backend.yml: triggers on pull_request and push to main with paths filter backend/** plus the workflow file itself. Jobs: lint (golangci-lint-action, version pinned to the one in backend/Makefile), test (make -C backend test; upload coverage.out and coverage.html as artifacts; write a per-package coverage table to $GITHUB_STEP_SUMMARY; run the coverage gate script), vuln (govulncheck). Use actions/setup-go reading go.mod with module cache.
   - .github/workflows/frontend.yml: paths filter frontend/**; setup-node 22 with npm cache; npm ci, lint, type-check, test:coverage (upload lcov, write the json-summary as a table to the step summary), build.
   - Every workflow: concurrency group per ref with cancel-in-progress, job timeouts, least-privilege top-level `permissions: contents: read`, every `uses:` pinned to a full commit SHA with a `# vX.Y.Z` comment.
   - Add a final `ci-ok` job per workflow that `needs` all others and fails if any failed or was cancelled, so branch protection references one check per workflow.
   - .github/dependabot.yml: gomod (backend/), npm (frontend/), github-actions, docker; weekly; group minor and patch updates.
   - Branch protection on main via `gh api` (PUT repos/isasumer/go-react-calculator/branches/main/protection): require PR, required status checks = the two ci-ok jobs (strict), dismiss stale reviews, required linear history, no force pushes, no deletions, enforce for admins. Print the resulting protection JSON in the PR body.
   - Add status badges for both workflows at the top of README.md (keep the section headings untouched).
   - Verify path filtering: this PR touches only .github/ and README, so note in the PR which workflows ran and why. After the PR is open, open a throwaway PR from a branch that introduces a deliberate lint error in backend/, confirm the ci-ok check blocks it, then close that PR and delete its branch; record the PR number as evidence.
   - Do not add Docker build, e2e, CodeQL, Trivy or release jobs — those are H3-03, H3-04, H3-05, H3-06.

   Finish by: pasting the workflow run URLs and the protection JSON into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session P0-04 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #4; opening the PR with `gh pr create` using the PR title from the issue and `Closes #4`. Do not merge. Report the PR URL.
   ```

2. Answer to the assistant's question about the conflict between workflow-level path filters and required
   status checks (verbatim option chosen):

   ```text
   Filter at job level (Recommended)
   ```

**Accepted** — `backend.yml` with lint (golangci-lint-action pinned to v2.13.2, the version in
`backend/tools/go.mod`, plus a step that fails if the two drift apart; `make fmt-check` for gofumpt),
test (`make -C backend test`, `coverage.out` + `coverage.html` artifacts, a per-package table from an awk
pass over the profile, `make coverage-check`) and vuln (`make -C backend vuln`, i.e. the pinned govulncheck);
`frontend.yml` running the `frontend/Makefile` targets, with vitest called directly only to add the
`json-summary` reporter, which a node snippet turns into the step-summary table; `ci-ok (backend)` /
`ci-ok (frontend)` gate jobs; all actions pinned to release SHAs; Dependabot for gomod (`/backend` and
`/backend/tools`), npm, github-actions and docker with grouped minor/patch; README badges; branch
protection with the two ci-ok checks (strict), 0 required approvals (solo repository; with enforce_admins a
required approval could never be given), stale-review dismissal, linear history, no force pushes/deletions,
admins included. Throwaway PR #53 (deliberate gosec/errcheck violation) showed `ci-ok (backend)` failing and
merge state `BLOCKED`; closed and branch deleted.

**Rejected (why)**
- Workflow-level `paths:` filters on `pull_request` as written in the prompt: GitHub leaves a required check
  from a skipped workflow in "Pending", so with both ci-ok checks required (and enforce_admins on)
  every backend-only, frontend-only or docs-only PR could never merge. The user chose job-level filtering:
  a `changes` job lists the PR's files via the API and the heavy jobs are skipped when nothing relevant
  changed; ci-ok treats "skipped" as pass. Pushes to main keep a workflow-level paths filter.
- `cancel-in-progress: true` for every event — only PR runs are cancelled, so a push to main never cancels
  the run for the previous main commit.
- A third-party paths-filter action — the GitHub API plus `grep` does the same with no extra dependency.
- A live `git push` to main to prove it is rejected — if protection were misconfigured, undoing it would
  need a force-push, which is forbidden. The protection JSON (PR required, enforce_admins) is the evidence.
- actionlint as a CI job — outside this ticket's Files list; filed as follow-up #54. It was run locally.

**Written by hand** — None; the workflow summary steps were extracted from the YAML and executed locally
against real coverage output, actionlint was run on both workflows, and `make check` passed before commit.

## Session B1-01 — 2026-09-22

1. Task prompt (verbatim; written in English):

   ```text
   Implement GitHub issue #5 (B1-01: Domain package internal/calc) in this repository. Start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 5`, then docs/PLAN.md §1.3 and §1.4 (numeric model and the operation list), and backend/Makefile. Do not explore beyond backend/internal/calc and docs/adr.

   Constraints for this ticket:
   - Zero third-party dependencies. Package doc comment explains the numeric model and links ADR-0003.
   - Types exactly as the issue specifies: Operation string constants (add, subtract, multiply, divide, power, sqrt, percent), Spec{Name, Symbol, Arity, fn}, an immutable ordered Registry with Lookup and List, and Evaluate(op, a, b *float64) (float64, error) that validates arity.
   - Sentinel errors ErrUnknownOperation, ErrDivisionByZero, ErrDomain, ErrNotFinite, ErrArity; wrap with %w and operation context so errors.Is works.
   - Guards: reject NaN/±Inf inputs; result must be finite; percent = a*b/100 ("b percent of a"); 0^0 = 1 documented; sqrt(-x) and non-real Pow results → ErrDomain; -0 normalised to 0.
   - Tests: table-driven with named cases including MaxFloat64 overflow, 0/0, sqrt(0), 0^0, negative percent, unary op given b, binary op missing b; FuzzEvaluate asserting no panic, finite result when err == nil, sentinel error otherwise; BenchmarkEvaluate; an Example in example_test.go.
   - Write docs/adr/0003-numeric-model.md using the template (float64 with non-finite guards; decimal mode deferred to follow-up #31; why JSON numbers make float64 the honest choice; the 0.1+0.2 caveat and where it is handled in the UI per ADR-0007). Flip its row in docs/adr/README.md to Accepted.
   - 100 % statement coverage on internal/calc; `go test -fuzz=FuzzEvaluate -fuzztime=10s ./internal/calc` runs clean.

   Finish by: running `make -C backend check` and the fuzz command and pasting the output into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session B1-01 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #5; opening the PR with `gh pr create` using the PR title from the issue and `Closes #5`. Do not merge. Report the PR URL.
   ```

**Accepted** — `backend/internal/calc`: `Operation` constants; `Spec{Name, Symbol, Arity, fn}`; `Registry`
(built by `NewRegistry`, ordered, read-only, `Lookup` plus `List` returning a copy); `Registry.Evaluate(op, a
float64, b *float64)` with arity checks, finite-operand and finite-result guards, `-0` → `0`; five sentinels,
each wrapped with `%w` and the operation name. Two small decisions beyond the prompt, both documented in the
package doc and in ADR-0003: `0` raised to a negative power returns `ErrDivisionByZero` (it is `1/0`, and
`math.Pow` would give +Inf); when `a*b` overflows but the percent itself is finite, `percent` falls back to
`a*(b/100)`. Tests: a 47-row table (every sentinel checked with `errors.Is`, `-0` checked via `math.Signbit`),
exact error-message context, registry order/copy/lookup, `FuzzEvaluate` (also asserts no `-0` and exactly one
sentinel per error), `BenchmarkEvaluate`, and two godoc Examples. ADR-0003 written; its README row flipped to
Accepted.

**Rejected (why)**
- `Evaluate(op, a, b *float64)` as written in the prompt → the issue and PLAN specify `a float64, b *float64`.
  Only `b` is optional, and the prompt said "types exactly as the issue specifies", so I followed the issue.
- `Evaluate` as a package-level function over a package-level registry → CLAUDE.md says "no globals,
  dependencies injected via structs". `Evaluate` is a method on `*Registry`, which B1-02 will inject into the
  handler.
- Filling the README Testing section with the benchmark numbers → that section belongs to B1-07/D4-01.
  The numbers are in the PR body instead.

**Written by hand** — None; every file was generated, read, and run locally (`make -C backend check`, the
10 s fuzz run, the benchmark) before commit.

## Session B1-02 — 2026-09-22

1. Task prompt (verbatim; written in English):

   ```text
   Implement GitHub issue #6 (B1-02: HTTP transport internal/httpapi) in this repository. Prerequisite #5 is merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 6`, then docs/PLAN.md §1.4 (the frozen API contract — every row there is a test case), docs/errors.md, docs/adr/README.md, and the public API of backend/internal/calc (go doc ./internal/calc). Do not explore beyond backend/internal/httpapi, backend/internal/calc (read-only), backend/cmd/server (wiring only) and docs.

   Constraints for this ticket:
   - Standard library only. Router is http.NewServeMux with Go 1.22+ method patterns: `POST /api/v1/calculate`, `GET /api/v1/operations`. Wrong method → 405 problem with an Allow header; unknown route → 404 problem. Do not add /healthz, /readyz, /version or middleware — those are #7 and #8.
   - Handler is a struct with injected dependencies (`Handler{calc *calc.Registry, log *slog.Logger}`), exposing `Routes() http.Handler`. No package-level globals. Handlers stay thin: decode → validate → calc.Evaluate → encode; business rules live in calc.
   - DTOs: CalculateRequest{Operation string; A *float64; B *float64} with pointers so "missing" and "zero" are distinguishable; CalculateResponse{Operation, A, B (omitted for unary), Result}; OperationsResponse listing name, symbol, arity from the registry.
   - Decoding: require Content-Type application/json (415 UNSUPPORTED_MEDIA_TYPE); http.MaxBytesReader at 4 KiB (413 PAYLOAD_TOO_LARGE); json.Decoder with DisallowUnknownFields; reject trailing data after the first JSON value; empty body, non-object body, wrong field types → 400 INVALID_BODY with a detail derived from the json error (field name / offset), never the raw Go error text.
   - Validation producing errors[] {field, message}: missing operation/a/b → 400 VALIDATION_FAILED; unknown operation → 422 UNSUPPORTED_OPERATION; b supplied to a unary operation → 422 UNEXPECTED_OPERAND; non-finite inputs guarded even though JSON cannot carry them.
   - Error mapping via errors.Is/As in a single mapError(err) Problem function with its own table test: calc.ErrDivisionByZero → 422 DIVISION_BY_ZERO, ErrDomain → 422 DOMAIN_ERROR, ErrNotFinite → 422 RESULT_NOT_FINITE, anything else → 500 INTERNAL with a generic detail (no leak) and an error log line.
   - Problem type: struct with type, title, status, detail, code, instance, requestId (read from an `X-Request-ID` response header if already set, empty otherwise — #8 will populate it), errors[]; Write(w, r, p) sets Content-Type application/problem+json. The `type` URL is the docs/errors.md anchor for the code.
   - writeJSON helper sets `application/json; charset=utf-8`, encodes to a buffer first so encode errors become a 500 rather than a half-written body.
   - Tests: handler_test.go table over request → status/code/body covering every row of PLAN §1.4 plus 405, 404, 415, 413, unknown field, trailing garbage, empty body, array body, string-typed number; problem_test.go; golden files under testdata/*.json for every problem body, and the golden files must be byte-identical to the examples you put in docs/errors.md. Coverage ≥ 90 % on internal/httpapi.
   - Wire Routes() into cmd/server run() replacing the placeholder 404 handler; keep the existing run() smoke test green.
   - Docs: fill the table in docs/errors.md (code, HTTP status, meaning, client behaviour, example) for every code above; write docs/adr/0004-single-calculate-endpoint.md and docs/adr/0005-problem-json-errors.md from the template and flip both rows in docs/adr/README.md to Accepted.

   Finish by: running `make -C backend check` and pasting the output (including the per-package coverage line) into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session B1-02 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #6; opening the PR with `gh pr create` using the PR title from the issue and `Closes #6`. Do not merge. Report the PR URL.
   ```

**Accepted** — `backend/internal/httpapi`: `Handler{calc, log}` built by `NewHandler`, with `Routes()` on a
`http.ServeMux` using method patterns. Wrong methods are caught by method-less fallback patterns for the same
paths, which return a 405 problem with `Allow`; `/` catches unknown paths and returns a 404 problem. The
mux's own 405/404 responses are plain text, so they are never used. DTOs use pointer operands. `decodeJSON`
enforces the media type, the 4 KiB `MaxBytesReader`, `DisallowUnknownFields`, a single value, and an object
rather than `null`, and maps each decoder error to a detail naming the field or byte offset. `validate`
lists every missing or non-finite field in one 400, then returns 422 for an unknown operation or an unexpected
`b`. `mapError` is the single error-to-problem mapping (`errors.Is` on the calc sentinels, `errors.AsType`
for `*Problem` and the operation context). `writeJSON` marshals to memory first and sets `Content-Length`.
Tests: a 37-case route table with golden files for every problem body (26 files), a `mapError` table (with no-leak and log
assertions), `Write`/`NewProblem` tables, and `TestErrorCatalogueMatchesGoldenFiles`, which fails when
docs/errors.md drifts from `testdata/`. Coverage is 96.4 %. `cmd/server` now serves `Routes()`, and its smoke
test also asserts the 404 is `application/problem+json`. docs/errors.md is filled (table, shape, one example
per code), ADR-0004 and ADR-0005 are written, and both README rows are Accepted. The table caught one real
bug before commit: for `{…}{}`, the trailing-data check passed a `nil` error to `decodeProblem` and panicked.

**Rejected (why)**
- `mapError(err) Problem` as a free function returning a value → implemented as `(h *Handler) mapError(err)
  *Problem`. The 500 branch must log through the injected logger (no globals). `*Problem` implements `error`,
  so decode and validation failures pass through the same single mapping point.
- `errors[]` "pointing at /a" (PLAN §1.4 wording) as a JSON Pointer → `field: "a"`, matching the contract's
  own problem example (`field: "b"`) and the VALIDATION_FAILED errors.
- `requestId` as an empty string when no header is set → omitted (`omitempty`), so a problem never claims an
  empty ID. The golden files set `X-Request-ID` the way B1-04's middleware will, so the docs show the field.
- An unexported `Write`/codes → exported (`Write`, `NewProblem`, `Code*`) because B1-04's recovery, timeout
  and rate-limit middleware must emit the same shape.
- Working around `encoding/json`'s case-insensitive key matching (`{"A":1}` is accepted) → outside the
  contract and needs a design choice (json/v2 or a key pre-scan); filed as follow-up #60.
- Field-specific detail for every DIVISION_BY_ZERO → only `divide` gets "b must be non-zero" (the contract's
  example). `power` with `0^-n` gets a generic detail, so the transport does not reimplement calc's rules.

**Written by hand** — None. Every file was generated, read, and run locally (`make -C backend check`, plus
the built binary exercised with curl) before commit.

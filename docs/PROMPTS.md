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

## Session B1-03 — 2026-09-22

1. Session mechanics (verbatim; the wrapper each ticket session is started with):

   ```text
   ORCHESTRATION RULES (from the orchestrator, override nothing in CLAUDE.md, add to it):
   - Work ONLY inside the git worktree /home/sumer/gorc-wt/issue-7. It is already on branch `backend/7-config-lifecycle`, cut from fresh origin/main. Do not switch branches, do not touch /home/sumer/go-react-calculator or any other worktree.
   - Go toolchain: /home/sumer/.local/go/bin (export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" in every shell you run). Node 22 via nvm is not needed for this ticket.
   - Never merge. Never push to main. Open the PR and stop.
   - Before opening the PR run `git fetch origin && git rebase origin/main`; if docs/PROMPTS.md or docs/adr/README.md conflict, keep both sides (all sessions' sections) and continue.
   - Your final message must contain: the PR URL, the `make -C backend check` coverage lines, anything you deliberately left out, and any follow-up issues you filed.
   ```

2. Task prompt (verbatim; written in English):

   ```text
   Implement GitHub issue #7 (B1-03: Config, server lifecycle, graceful shutdown, health/readiness/version endpoints) in this repository. Prerequisite #6 is merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 7`, then docs/PLAN.md §1.1 and §1.4 (the operational endpoints line), backend/cmd/server/main.go and its test, and the exported API of backend/internal/httpapi (go doc). Do not explore beyond backend/cmd/server, backend/internal/config, backend/internal/observability, backend/internal/httpapi (only to register the probe routes) and docs.

   Constraints for this ticket:
   - Standard library only. No middleware behaviour and no metrics — those are #8 and #9.
   - internal/config: `Load(getenv func(string) string) (Config, error)`. Variables with defaults: PORT=8081, HOST=0.0.0.0, LOG_LEVEL=info (debug|info|warn|error), LOG_FORMAT=json (json|text), CORS_ALLOWED_ORIGINS="" (csv, parsed to []string), READ_HEADER_TIMEOUT=5s, READ_TIMEOUT=10s, WRITE_TIMEOUT=10s, IDLE_TIMEOUT=60s, SHUTDOWN_TIMEOUT=15s, REQUEST_TIMEOUT=5s, RATE_LIMIT_RPS=20, RATE_LIMIT_BURST=40, MAX_BODY_BYTES=4096, TRUST_PROXY_HEADERS=false, PRE_STOP_DELAY=0s. Durations via time.ParseDuration, ints/bools strictly parsed. Invalid value → error naming the variable and the received value. Config.String() renders the resolved values for the startup log (nothing here is secret; say so in a comment).
   - internal/observability/buildinfo.go: Version, Commit, BuildDate set via -ldflags -X (already in backend/Makefile build target — check and align names), with debug.ReadBuildInfo fallback for commit/vcs.time and "dev" defaults. `-version` flag on the binary prints them and exits 0.
   - Logger: slog handler chosen by LOG_FORMAT and LOG_LEVEL, constructed in run() and injected into httpapi.Handler. Startup line logs the resolved config and bound address; shutdown line logs the drain duration.
   - run(ctx, args, getenv, stdout) error: parse flags, Load config, build logger, registry, handler, http.Server with all five timeouts set from config, net.Listen first (so PORT=0 works and the bound address is known), serve in a goroutine, wait for ctx.Done() (signal.NotifyContext for SIGINT/SIGTERM in main only, not in run), then: flip readiness to false, sleep PRE_STOP_DELAY, Shutdown with SHUTDOWN_TIMEOUT, return the first non-nil error. main() stays a thin wrapper that exits 1 on error.
   - Probes in httpapi (or a small probes.go there): GET /healthz → 200 {"status":"ok"} always once the process serves; GET /readyz → 200 {"status":"ready"} while ready, 503 problem code=NOT_READY once shutdown began (atomic.Bool owned by the server lifecycle, passed in as a func() bool or *atomic.Bool); GET /version → 200 {version, commit, buildDate, goVersion}. All three set Cache-Control: no-store. Wrong method → 405 as elsewhere.
   - Tests: config_test.go (all defaults, every override, each invalid value with the variable name in the error, CSV parsing with spaces/empties); buildinfo_test.go; probe handler tests including readiness flip; main_test.go lifecycle test: start run() with PORT=0 using t.Setenv-style getenv, wait until /readyz is 200, start a slow in-flight request, cancel ctx, assert /readyz turns 503 before the server closes, the in-flight request completes, and run() returns nil within SHUTDOWN_TIMEOUT. Also a test that an invalid PORT makes run() return an error without listening. No fixed ports anywhere.
   - Update backend/README.md (or the Configuration section it owns) with the env-var table: name, default, meaning. Root README "Configuration" section gets one line pointing there; do not restructure README headings.

   Finish by: running `make -C backend check` and `PORT=0 go run ./cmd/server` (then Ctrl-C, or send SIGINT via timeout -s INT 2 ...) and pasting the outputs — including the startup and shutdown log lines — into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session B1-03 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #7; opening the PR with `gh pr create` using the PR title from the issue and `Closes #7`. Do not merge. Report the PR URL.
   ```

**Accepted** — `internal/config`: `Load(getenv)` with a small `loader` that reads every variable, collects
*all* parse failures and returns them joined, so one restart shows every mistake instead of one per attempt.
Each failure is a `*ParseError{Name, Value, Err}` rendering as `config: PORT="eight thousand": must be an
integer`. Unset and blank are the same thing (a deployment that passes `PORT=""` gets the default). Types are
real types, not strings: `slog.Level`, a `LogFormat` enum, `time.Duration`, `int64` for `MAX_BODY_BYTES`, and
`[]string` for the CSV allowlist (spaces trimmed, empty items dropped). `Config.String()` renders every
resolved value for the startup log with a comment saying nothing here is secret and where to redact if that
ever changes. `internal/observability.Build()` layers the three sources — `-ldflags -X`, the toolchain's
embedded VCS stamps (short revision plus `-dirty`), then `dev` — behind `resolve()`, which takes the
`*debug.BuildInfo` as an argument so the fallback is testable without building a binary; `backend/Makefile`
now stamps `…/internal/observability.Version|Commit|BuildDate` instead of the `main.version|commit|date`
variables that no longer exist. `cmd/server`: `main` owns only signals and the exit code; `run` listens
before it serves (so `PORT=0` resolves to a real port and a bind failure is an error, not a log line), then
`drain()` flips readiness off, waits `PRE_STOP_DELAY`, and calls `Shutdown` with `SHUTDOWN_TIMEOUT`.
`httpapi` gained `probes.go` (`/healthz`, `/readyz`, `/version`, all `Cache-Control: no-store`, 405 with
`Allow: GET, HEAD` for anything else) and functional options `WithReadiness`/`WithBuildInfo`, so the handler
stays free of globals and every existing `NewHandler(reg, log)` call still compiles. Coverage: config and
observability 100 %, httpapi 97.2 %, cmd/server 90.9 %, total 97.0 %.

**Rejected (why)**
- The `-addr` flag that `run()` had since P0-02 → deleted. With `HOST`/`PORT` in the environment, a second
  way to set the address is a second source of truth; `-version` is the only flag left. `make run` and the
  backend README now document `PORT=9000 make run`.
- `*atomic.Bool` passed into `httpapi` (the prompt offered either) → `WithReadiness(func() bool)`. The flag
  belongs to the process lifecycle; the transport only needs to ask. `WithReadiness(nil)` keeps the default
  (always ready) rather than panicking on the first probe.
- `slog.Level.UnmarshalText` for `LOG_LEVEL` → a strict four-value lookup. slog's parser also accepts
  `info+2`, which is not a contract worth supporting; the error message lists the four names instead.
- A `main.version`-style set of variables kept for compatibility → removed. Two copies of the build identity
  is how a `/version` endpoint starts lying.
- Building the logger in `internal/observability` (where PLAN §1.1 files "slog setup") → the prompt is
  explicit that `run()` constructs it, and a 12-line `newLogger` in the composition root beats a package
  that exists to hide a `switch`. `observability` keeps build information only.
- `t.Setenv` in the lifecycle test → a map-backed `getenv`. `t.Setenv` mutates the process environment and
  forbids `t.Parallel`, and `run` takes `getenv` precisely so no test has to.
- `time.Sleep` to get a request in flight across the shutdown → `Expect: 100-continue` with an
  `httptrace.Got1xxResponse` hook. The server answers `100` exactly when the handler first reads the body,
  which is the moment the connection counts as active, so the test synchronises on the server's own
  behaviour instead of guessing. The body is then sent in two halves around the cancellation. No sleeps and
  no fixed ports anywhere (`PORT=0`, plus one test that binds a port first and hands it to `run` to prove a
  listen failure is returned).
- Adding `NOT_READY` as a code without touching the docs → not possible, and rightly: ADR-0005 makes
  `docs/errors.md` the source of the `type` anchors and `TestErrorCatalogueMatchesGoldenFiles` enforces it.
  The catalogue got a row, a section and the golden file `testdata/not_ready.json`.
- Wiring `MAX_BODY_BYTES` into `httpapi`'s 4 KiB constant here → out of scope (this ticket only reads
  config; B1-04's Files/areas is `internal/middleware/`). Filed as follow-up #62 so the variable does not
  stay documented-but-ignored.

**Written by hand** — None. Every file was generated, read and run locally (`make -C backend check`, the
built binary exercised with curl against all three probes, and `PORT=0 go run ./cmd/server` interrupted with
SIGINT) before commit.

## Session F2-01 — 2026-09-22

Two prompts: the session was run by a small orchestrator that prepares a worktree per issue, so the
first prompt is that harness's rules and the second is the ticket itself.

1. ```
   ORCHESTRATION RULES (from the orchestrator, override nothing in CLAUDE.md, add to it):
   - Work ONLY inside the git worktree /home/sumer/gorc-wt/issue-12. It is already on branch `frontend/12-api-layer`, cut from fresh origin/main. Do not switch branches, do not touch /home/sumer/go-react-calculator or any other worktree.
   - Node 22 via nvm: run `source ~/.nvm/nvm.sh && nvm use 22` (or whatever frontend/.nvmrc says) in every shell you run. Go is not needed for this ticket; the backend is only a reference for the contract.
   - A backend session (#7) runs in parallel and will also append docs/PROMPTS.md. Never merge. Never push to main. Before opening the PR run `git fetch origin && git rebase origin/main`; if docs/PROMPTS.md conflicts, keep both sides and continue.
   - Your final message must contain: the PR URL, the coverage summary from `npm run test:coverage`, anything you deliberately left out, and any follow-up issues you filed.
   ```

2. ```
   Implement GitHub issue #12 (F2-01: Frontend API layer) in this repository. Prerequisites #3 and #6 are merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 12`, then docs/PLAN.md §1.1, §1.2 and §1.4 (the frozen API contract), docs/errors.md (every problem code the backend emits — this is your MSW fixture list), docs/adr/0002-frontend-stack.md and docs/adr/0005-problem-json-errors.md, frontend/README.md conventions, frontend/src/config.ts and frontend/src/test/setup.ts. For the exact wire shapes, read backend/internal/httpapi/dto.go and backend/internal/httpapi/testdata/*.json read-only. Do not explore beyond frontend/src/{lib,hooks,types,test,components/providers}, frontend/package.json and docs.

   Constraints for this ticket:
   - src/lib/api.ts: `apiFetch<T>(path, { method?, body?, signal?, schema?, timeoutMs? })` over native fetch. Base URL from getConfig().apiBaseUrl; JSON headers; default timeout 10 s via AbortSignal.timeout merged with the caller's signal (AbortSignal.any if available, else manual). Parse `application/problem+json` (and any JSON error body with a `code`) into `class ApiError extends Error { status, code, title, detail, requestId, errors: {field,message}[], retryAfterSeconds? }`; network/abort failures → ApiError code=NETWORK (status 0) or TIMEOUT; non-JSON or unexpected bodies → code=UNEXPECTED; when `schema` (zod) is given, validate the parsed body and fail closed with code=INVALID_RESPONSE. Export `isApiError(e): e is ApiError`. Also read X-Request-ID from the response headers into ApiError.requestId when the body has none.
   - src/lib/query-config.ts: CALCULATE_ENDPOINT, OPERATIONS_ENDPOINT, query keys (`calculatorKeys.operations()`), STALE_TIME constants.
   - src/types/calculator.ts: OperationName union matching the backend registry (add, subtract, multiply, divide, power, sqrt, percent), CalculateRequest (a required, b optional), CalculateResponse, OperationSpec {name, symbol, arity}, ProblemDetails; zod schemas `calculateResponseSchema`, `operationsResponseSchema`, `problemSchema` (lenient on unknown fields) with inferred types re-exported. Keep hand-written interfaces where zod inference would hurt readability.
   - src/lib/error-messages.ts: single table mapping every backend code in docs/errors.md plus NETWORK/TIMEOUT/UNEXPECTED/INVALID_RESPONSE to a short user-facing message; `messageForError(e: unknown): string` with a generic fallback. Tested exhaustively (every code has a message; unknown code falls back).
   - src/hooks/use-calculate.ts: `useCalculate()` → useMutation<CalculateResponse, ApiError, CalculateRequest> using apiFetch with calculateResponseSchema. src/hooks/use-operations.ts: `useOperations()` → useQuery with staleTime Infinity, retry 1, and `placeholderData` = a static fallback list (same seven operations) so the UI never blocks on the network; expose `isFallback`.
   - src/components/providers/QueryProvider.tsx: QueryClient defaults (queries: retry 1, refetchOnWindowFocus false, staleTime 60 s; mutations: retry 0); exported `createQueryClient()` for tests and a `renderWithProviders`/`createWrapper` helper in src/test/utils.tsx.
   - src/test/msw/handlers.ts + server.ts: handlers replicating the backend contract for POST /api/v1/calculate (every operation, DIVISION_BY_ZERO, DOMAIN_ERROR, RESULT_NOT_FINITE, UNSUPPORTED_OPERATION, VALIDATION_FAILED, INVALID_BODY) and GET /api/v1/operations, plus helpers to override a handler per test (429 with Retry-After, 500 text body, network error, malformed JSON). Wire server lifecycle in src/test/setup.ts (listen with onUnhandledRequest: "error", resetHandlers after each, close after all). Problem bodies in fixtures must match docs/errors.md examples.
   - Tests: api.test.ts (success with schema, 422 problem → ApiError fields incl. errors[], 429 → retryAfterSeconds, network error → NETWORK, timeout → TIMEOUT, malformed JSON → UNEXPECTED, schema mismatch → INVALID_RESPONSE, request id from header), error-messages.test.ts, use-calculate.test.tsx and use-operations.test.tsx with renderHook + wrapper (success, error, fallback path when the network fails). No `any`; ESLint and type-check clean. Coverage for the new files ≥ 90 % lines.
   - Do not build UI components, stores or the calculator engine — those are #13 and #14. Do not change vitest thresholds (that is #18).

   Finish by: running `make -C frontend check` and `npm run test:coverage` and pasting the outputs into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session F2-01 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #12; opening the PR with `gh pr create` using the PR title from the issue and `Closes #12`. Do not merge. Report the PR URL.
   ```

**Accepted** — `src/lib/api.ts` is the whole HTTP surface: `apiFetch<T>(path, options)` over native `fetch`,
with `fetch` and `response.text()` inside one `try` so a connection dropped mid-body fails like any other
transport error. `anySignal` merges `AbortSignal.timeout(timeoutMs)` with the caller's signal, preferring
`AbortSignal.any` and otherwise wiring listeners with `{ once: true, signal: controller.signal }` so nothing
leaks; both paths are tested (the test deletes `AbortSignal.any` and restores its property descriptor).
Nothing escapes as a `TypeError`, `DOMException` or `ZodError`: a failed request is `NETWORK` (status 0) or
`TIMEOUT`, any JSON body carrying a `code` keeps that code, anything else becomes `UNEXPECTED` with a
200-character excerpt of the body, and a schema mismatch fails closed as `INVALID_RESPONSE`. `requestId`
falls back from the body to the `X-Request-ID` header, and `Retry-After` is read as delay-seconds or an
HTTP-date. `src/types/calculator.ts` hand-writes what we send (`CalculateRequest`) and the documented
problem shape (`ProblemDetails`, `FieldError`) and infers what the server sends from the zod schemas, so a
schema and its type cannot drift. `src/test/msw/handlers.ts` is a reimplementation of the service's
decode → validate → evaluate pipeline, including the 400-before-422 ordering, the per-operation details and
`httpapi.NewProblem`'s `type`/`title`/`status` derivation; one test asserts the DIVISION_BY_ZERO body field
for field against the example in `docs/errors.md`. `FALLBACK_OPERATIONS` lives in the hook (app code never
imports from `src/test`) and a test asserts it equals the MSW registry, so the two cannot drift either.
119 tests, `make -C frontend check` green, 100 % statements/branches/functions/lines on every measured file.
Dependencies added: `@tanstack/react-query` and `zod`, both named in ADR-0002.

**Rejected (why)**
- `apiFetch` rethrowing a caller-initiated abort unchanged → every rejection is an `ApiError`, so one
  `isApiError` check covers the layer. A cancellation maps to `NETWORK`; TanStack Query discards it anyway.
- `calculateResponseSchema.operation` as a free string → `z.enum`: the server can only echo an operation we
  just sent, so an unknown one is a real defect. The discovery endpoint keeps `name: z.string()` for the
  opposite reason — it exists precisely so a backend that gains an operation does not break this client.
- Adding a `RATE_LIMITED` row to `docs/errors.md` → the catalogue is B1-04's to extend. The message table
  and the 429 handler use the code ADR-0005 already fixes, and `error-messages.test.ts` says so.
- `error-messages.test.ts` reading `docs/errors.md` to enumerate the codes → the list is hard-coded.
  PLAN §1.1 wants each app to test standalone; a unit test that reads a sibling directory breaks that.
- Wiring `QueryProvider` into `main.tsx` → left to #13, the first ticket with a component that needs it.
  The provider has its own test, so it is not untested dead code.
- `@tanstack/react-query-devtools` → installed while wiring the provider, then removed: not in ADR-0002's
  stack and not needed by this ticket.
- Hoisting `AbortSignal.any` into a variable (`const any = ctor.any`) → `typeof ctor.any === "function"`
  followed by `ctor.any(...)`. Detaching a static method loses its receiver; `@typescript-eslint/unbound-method`
  flagged it and was right to.
- Touching `vitest.config.ts` thresholds, `frontend/README.md` or `main.tsx` → outside this issue's
  **Files / areas**; coverage thresholds are #18.

**Written by hand** — None. Every file was generated, read and run locally (`make -C frontend check`,
`npm run test:coverage`) before commit. Two generated tests were wrong and were corrected after reading the
failures: `useOperations` reports `isSuccess` from the very first render because `placeholderData` counts as
data, so those tests now wait on `isFallback`; and `statusText` is filled from the standard reason phrase, so
the "no status text, fall back to the code as the title" case needs a non-standard status (599).

## Session F2-02 — 2026-09-22

Three prompts: the orchestrator's harness rules, the ticket itself, and a resume instruction after the
session was interrupted mid-way.

1. ```
   ORCHESTRATION RULES (from the orchestrator, override nothing in CLAUDE.md, add to it):
   - Work ONLY inside the git worktree /home/sumer/gorc-wt/issue-13. It is already on branch `frontend/13-calculator-engine`, cut from fresh origin/main (which includes #12: the API layer with useCalculate, ApiError, types and MSW handlers). Do not switch branches, do not touch /home/sumer/go-react-calculator or any other worktree.
   - Node 22 via nvm: run `source ~/.nvm/nvm.sh && nvm use` (frontend/.nvmrc) in every shell you run.
   - A backend session (#8) runs in parallel and will also append docs/PROMPTS.md. Never merge. Never push to main. Before opening the PR run `git fetch origin && git rebase origin/main`; if docs/PROMPTS.md conflicts, keep both sides (blank line between sessions) and continue; `git push --force-with-lease` on your own feature branch is allowed for that rebase only.
   - Your final message must contain: the PR URL, the coverage summary from `npm run test:coverage`, anything you deliberately left out, and any follow-up issues you filed.
   ```

2. ```
   Implement GitHub issue #13 (F2-02: Calculator engine + store) in this repository. Prerequisite #12 is merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 13`, then docs/PLAN.md §1.1, §1.2 and §1.3 (ADR-0008 row), docs/adr/0002-frontend-stack.md, frontend/README.md conventions, and the exported surface of frontend/src/lib/api.ts, frontend/src/types/calculator.ts, frontend/src/hooks/use-calculate.ts and frontend/src/lib/error-messages.ts. Do not explore beyond frontend/src/{lib,stores,types,hooks,test} and docs/adr.

   Constraints for this ticket:
   - src/lib/calculator-engine.ts is pure TypeScript: no React, no zustand, no fetch. Model: `type Phase = 'idle' | 'enteringA' | 'operatorSelected' | 'enteringB' | 'result' | 'error'`; `interface CalcState { phase; display: string; a: number | null; b: number | null; operator: BinaryOperation | null; error: string | null; lastRequest: CalculateRequest | null }`; `initialState`. Transition functions return `{ state: CalcState; effect?: { kind: 'calculate'; request: CalculateRequest } }`: inputDigit(d), inputDecimal(), toggleSign(), backspace(), clearEntry(), clearAll(), setOperator(op), requestEvaluate(), applyUnary(op: 'sqrt' | 'percent'), applyResult(response), applyError(message). Every function has an exhaustive `switch (state.phase)` with a `satisfies never` default so adding a phase fails type-check.
   - Input rules: max 16 significant characters in the entry (ignore further digits), single decimal point, no leading zeros except "0.", "-0" normalised to "0", toggleSign on "0" is a no-op, backspace on a single char yields "0", operator pressed twice replaces the operator, operator pressed in `result` phase chains (result becomes `a`), digit pressed in `result` phase starts fresh, Enter with no `b` reuses `a` as `b` (classic behaviour, documented), any input in `error` phase first clears the error (Escape clears fully), inputs while a request is pending are the caller's responsibility (store guards them, engine does not know about pending).
   - Evaluation semantics (write docs/adr/0008-frontend-evaluation-semantics.md from the template and flip its row in docs/adr/README.md to Accepted): immediate execution, no precedence — `2 + 3 × 4` yields 20. All arithmetic goes through the API; the engine never computes. Unary ops: `sqrt` applies to the current entry (or `a` in result/operatorSelected phase) and emits a calculate effect with arity-1 request; `percent` in `enteringB` phase emits `percent(a, b)` per the backend definition (b percent of a) and, on result, replaces the entry so the pending binary operation can continue — document this choice and its alternatives (spreadsheet-style vs calculator-style) in the ADR.
   - src/stores/useCalculatorStore.ts: zustand store holding CalcState plus `pending: boolean`; actions mirror the engine functions; when a transition returns an effect the store sets pending, calls an injected `evaluator: (req: CalculateRequest) => Promise<CalculateResponse>` (set via `setEvaluator` from the component layer in #14, so the store is testable without React Query), then applies `applyResult` or `applyError(messageForError(e))`. While pending, all inputs except clearAll are ignored. Handle the race where clearAll happens mid-request: the stale response must be discarded (sequence number or request identity check, tested). Export selectors: selectDisplay, selectPhase, selectIsBusy, selectError, selectExpression (a human string like "12 +" for the Display's expression line).
   - Tests: calculator-engine.test.ts as a table from-state × input → to-state covering every rule above, plus scripted sequences with expected display after each step: "12 + 7 =" → 19, "5 ÷ 0 =" → error phase with the mapped message, "9 √" → 3, "200 + 15 % =" → 230, "= = =" repeats the last operation (document), "2 + 3 × 4 =" → 20, backspace/CE/AC interactions, 16-char cap, chaining after result. Aim for 100 % branch coverage on calculator-engine.ts. useCalculatorStore.test.ts drives full calculations with a fake evaluator resolving and rejecting with ApiError (DIVISION_BY_ZERO), asserts pending flag, ignores input while pending, discards a stale response after clearAll, and verifies the expression selector.
   - Do not render anything, do not touch main.tsx or components — that is #14 (which must also wire QueryProvider into main.tsx; mention this in your PR body so the orchestrator carries it forward).

   Finish by: running `make -C frontend check` and `npm run test:coverage` and pasting the outputs into the PR body; appending this prompt verbatim to docs/PROMPTS.md under "## Session F2-02 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #13; opening the PR with `gh pr create` using the PR title from the issue and `Closes #13`. Do not merge. Report the PR URL.
   ```

3. ```
   Resume issue #13 exactly where you stopped: you were interrupted by an API rate limit while writing the engine tests. The limit has reset. First run `git -C /home/sumer/gorc-wt/issue-13 status --short` and `git log --oneline -3` to see what is committed vs uncommitted, do not redo finished work, then continue the ticket to completion: calculator-engine tests, useCalculatorStore + its tests, ADR-0008 and its index row, `make -C frontend check` and `npm run test:coverage`, PROMPTS.md section, rebase on origin/main (force-with-lease on your own branch is allowed for that), tick issue #13 criteria, open the PR with `Closes #13`. Do not merge. Report the PR URL, coverage summary, anything left out and any follow-ups filed.
   ```

**Accepted** — `src/lib/calculator-engine.ts` is the whole calculator as `(state, input) => { state, effect? }`
with no import of React, zustand or `fetch`, and no arithmetic of its own: even `x %` alone is sent as
`percent(x, 1)`, and `±` flips the sign of the *entry string* rather than negating a number. Every
transition is an exhaustive `switch (state.phase)` whose `default` is
`exhausted(state.phase satisfies never, stay(state))` — the `satisfies never` and the helper's `never`
parameter both break the build if a phase is added, and at runtime an impossible phase ignores the key
instead of throwing, which a test asserts for all eleven transitions (that is also what keeps the
defensive branch inside the coverage numbers). The insight that removed the extra state the ticket
hinted at (`pendingUnary`): **while a request is in flight the phase is already the one its answer lands
in**, so `applyResult` needs nothing else to know where the number goes — `√`/`%` on an entry land in
`enteringA`/`enteringB` and replace the entry, a chained operator lands in `operatorSelected` and fills
`a`, `=` lands in `result`. `lastRequest` is recorded only by `=`, which is what makes `= = =` repeat
(19, 26, 33) without repeating a chain or a `√`. `src/stores/useCalculatorStore.ts` keeps the engine
state under one key (`calc`) so a transition is applied atomically, guards every key but `AC` behind
`pending`, and settles a response only if its ticket is still the current one — `AC` bumps the ticket,
so the answer to a cleared calculation is dropped, resolved *and* rejected, both tested. 283 tests,
`make -C frontend check` green, 100 % statements/branches/functions/lines overall and on both new files.
Dependency added: `zustand` 5.0.15, named in ADR-0002.

**Rejected (why)**
- Computing anything locally, including `x / 100` for a bare `%` or `-x` for `±` → two implementations
  of the same rules, and the tested one would not be the one the user runs (ADR-0008).
- A `pendingOperator` / `pendingUnary` field on `CalcState` → the landing phase already carries that
  information; the extra field would have been a second source of truth for the same fact.
- `default: throw new Error("unreachable")` in the exhaustive switches → an unreachable branch is an
  uncoverable branch, and a calculator that crashes on a key is worse than one that ignores it.
- Context-sensitive `%` (`a × b %` → b/100) and spreadsheet `%` (always x/100) → one rule the user can
  learn, one backend operation; both alternatives are written up in ADR-0008 with what they cost.
- Flattening `CalcState` into the store's own state → actions and engine fields would live in one
  object and an engine call could be handed the actions by accident.
- Number formatting beyond `String(value)` → ADR-0007's 12-significant-digit display and
  `format-number.ts` are F2-06's; the engine says so where it formats.
- Keeping `lastRequest` when a digit starts a fresh calculation → `= ` then repeats an operation the
  user has visibly left behind.
- Wiring `QueryProvider` into `main.tsx`, rendering anything, keyboard mapping, history → #14, F2-04
  and F2-05. Nothing outside the issue's **Files / areas** was touched (plus this file and ADR-0008,
  which the ticket names).

**Written by hand** — None. Every file was generated, read and run locally (`make -C frontend check`,
`npm run test:coverage`) before commit. Two generated tests were wrong and were corrected after reading
the failure and the coverage report: the 16-digit cap test forgot that the placeholder `0` in `-0.` is
itself one of the sixteen digits, and the `state.a ?? 0` fallback in `applyUnary` needed its own row in
the table (a `√` pressed in `operatorSelected` on a state whose `a` has not arrived yet) before the
engine reached 100 % branch coverage.

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

## Session B1-04 — 2026-09-22

1. `Implement GitHub issue #8 (B1-04: Middleware chain) in this repository. Prerequisite #7 is merged; start from a fresh `main`.`

   ```
   Start by reading CLAUDE.md, then `gh issue view 8`, then docs/PLAN.md §1.2 (middleware order) and §1.4, docs/errors.md, the exported API of backend/internal/httpapi and backend/internal/config (go doc), and backend/cmd/server/main.go to see where the chain is assembled. Do not explore beyond backend/internal/middleware, backend/cmd/server (wiring only), backend/internal/httpapi (only to reuse the Problem writer and populate requestId), docs/errors.md and docs/adr.

   Constraints for this ticket:
   - One new dependency only: golang.org/x/time/rate. Justify it in the go.mod commit message. Everything else is standard library. No metrics middleware — that is #9; leave a clearly named slot in the chain.
   - middleware.Chain(h http.Handler, mws ...Middleware) http.Handler where Middleware is func(http.Handler) http.Handler; the first middleware listed is the outermost. Order assembled in run(): Recover → RequestID → Logger → Timeout → SecurityHeaders → CORS → RateLimit → (metrics slot) → router. Write chain_test.go proving the order with a probe handler that records entry/exit.
   - RequestID: accept an incoming X-Request-ID only if it matches ^[A-Za-z0-9_-]{1,64}$, otherwise generate 16 random bytes from crypto/rand encoded base32 without padding. Set the header on the response before the handler runs, store it in the context with an unexported key and an exported FromContext(ctx) accessor. httpapi.Problem.requestId must now be populated from the context (small change in the problem writer).
   - Logger: slog, one line per request with request_id, method, path, route pattern (r.Pattern), status, bytes, duration_ms, remote_ip, user_agent. Wrap ResponseWriter to capture status and bytes; implement Flush and Unwrap so http.ResponseController keeps working. Level by status: 5xx error, 4xx warn, else info; /healthz and /readyz logged at debug.
   - Recover: recover panics (including http.ErrAbortHandler re-panicked as net/http expects), log at error with the stack, write a 500 problem code=INTERNAL if nothing was written yet. Never expose the panic value.
   - Timeout: enforce REQUEST_TIMEOUT via context and a problem+json 503 code=TIMEOUT (document why not http.TimeoutHandler: its body is plain text and not problem+json). Make sure the handler cannot write after the timeout response (guard with a mutex like TimeoutHandler does).
   - SecurityHeaders: X-Content-Type-Options nosniff, X-Frame-Options DENY, Referrer-Policy no-referrer, Content-Security-Policy "default-src 'none'; frame-ancestors 'none'", Cache-Control no-store on /api/ routes only.
   - CORS: allowlist from config CORS_ALLOWED_ORIGINS; when empty, add no CORS headers at all. Preflight OPTIONS from an allowed origin → 204 with Allow-Origin (echo, never *), Allow-Methods "POST, GET, OPTIONS", Allow-Headers "Content-Type, X-Request-ID", Max-Age 600, and Vary: Origin on every response that consults the allowlist. Disallowed origin → no CORS headers, request otherwise proceeds.
   - RateLimit: per-client token bucket (rate.Limiter) with RATE_LIMIT_RPS and RATE_LIMIT_BURST, buckets in a map guarded by a mutex with TTL eviction of idle entries (evict on a ticker or lazily every N inserts; test it). Client key = host part of RemoteAddr; only when TRUST_PROXY_HEADERS=true use the first X-Forwarded-For entry. Over limit → 429 problem code=RATE_LIMITED with Retry-After (integer seconds, minimum 1). Skip rate limiting for /healthz, /readyz and /metrics.
   - Tests: one *_test.go per middleware with httptest and a recording handler, plus chain_test.go. Cover: invalid incoming request id replaced; log line fields with a slog handler writing to a buffer; panic → 500 with the request id and the server still serving; timeout → 503 and no double write; preflight allowed/disallowed/empty config; burst then 429 then recovery after refill (use a small rps and time.Sleep sparingly, or inject a clock); eviction; X-Forwarded-For honoured only with TRUST_PROXY_HEADERS.
   - Docs: add INTERNAL, TIMEOUT, RATE_LIMITED rows to docs/errors.md if missing (NOT_READY already exists), with examples that match golden files. Add a "Middleware" subsection to backend/README.md listing the order and what each layer does in one line.

   Finish by: running `make -C backend check` and pasting the output into the PR body, plus a short curl transcript showing X-Request-ID on a response, a preflight from an allowed origin, and a 429 with Retry-After; appending this prompt verbatim to docs/PROMPTS.md under "## Session B1-04 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #8; opening the PR with `gh pr create` using the PR title from the issue and `Closes #8`. Do not merge. Report the PR URL.
   ```
   [The prompt was given in English this time; there is no Turkish original to gloss.]

**Accepted** — Seven middleware, seven files, seven test files, plus `chain.go` (the `Middleware` type and
`Chain`, which skips `nil` entries so the metrics slot can be declared before B1-05 fills it) and
`responsewriter.go` (the `recorder` both `Recover` and `Logger` wrap responses in, with `Flush` and `Unwrap`
so `http.ResponseController` still reaches the connection through four layers — there is a test that sets a
write deadline through the whole chain against a real server, because that path only exists via `Unwrap`).
`Timeout` is `http.TimeoutHandler`'s mechanics with a problem+json body: handler on its own goroutine, a
deadline context, a mutex between it and the response, and its own header map so a late handler cannot
mutate headers of a response already on the wire. `RateLimit` keys `rate.Limiter`s in a mutex-guarded map
and sweeps idle buckets every 64th insert — no ticker, so no goroutine to own and stop — and takes a
`Now func() time.Time` so the burst/refill test asserts the refill instead of sleeping through it. The
clock, the `IdleTTL` and the eviction sweep are all exercised directly. `docs/errors.md` gained
`RATE_LIMITED` and `TIMEOUT` rows and sections; their golden files are written by the *middleware* tests
(`../httpapi/testdata/`), so the documented example is literally the response the chain sends, and
`httpapi.TestErrorCatalogueMatchesGoldenFiles` then checks the docs against it. Coverage: middleware 99.6 %,
httpapi 97.2 %, cmd/server 91.3 %, total 97.9 %.

**Rejected (why)**
- The unexported context key in `internal/middleware`, as the prompt specified → the key and the two
  accessors live in `internal/httpapi` (`context.go`). `middleware` imports `httpapi` for the problem
  writer, so `httpapi` reading a key owned by `middleware` would be an import cycle. `middleware.FromContext`
  is still the accessor the prompt asked for; it is a three-line forward, documented with the reason.
- `httpapi.Write` reading the ID *only* from the context → context first, the `X-Request-ID` response header
  second. `Recover` wraps `RequestID`, so it never sees the context that `RequestID` derived; the header it
  shares is the only thing that ties its 500 to the log line. Same helper (`requestIDOf`) in `Logger`.
- Buffering the response body like `http.TimeoutHandler` → pass-through writes with a `closed` flag. The
  handlers here marshal into memory and write once, so buffering would only add a copy; what actually
  matters is that after `stop()` returns no write is in flight, which the mutex gives directly. The late
  handler gets `http.ErrHandlerTimeout`, as it would from `TimeoutHandler`.
- Logging `remote_ip` from `X-Forwarded-For` when `TRUST_PROXY_HEADERS=true` → the log always records the
  peer address. A log line records what happened; a header anyone can set is not that. The flag changes
  behaviour only where it has to, in the limiter's key.
- `crypto/rand.Text()` (26 base32 characters, one call) → an explicit 16-byte `rand.Read`, because the
  ticket says 16 bytes and the length is then something the test can assert rather than inherit.
- Treating an `OPTIONS` without `Access-Control-Request-Method` as a non-preflight → any `OPTIONS` from an
  allowed origin is answered 204. The API defines no other `OPTIONS` semantics, so the alternative outcome
  is the 405 a disallowed origin already gets.
- Probe paths logged at `debug` unconditionally → status wins first, so a draining `/readyz` (503) stays
  visible at `error` for the few seconds it lasts. Only a *healthy* probe drops to `debug`.
- Closing follow-up #62 (`MAX_BODY_BYTES`) here → it is already closed on GitHub, and the change it
  describes is not the one-line wiring the orchestrator allowed: it needs a `WithMaxBodyBytes` option and a
  new field on `httpapi.Handler`, which is request decoding, not the middleware chain. The backend README
  now says plainly that the variable is parsed but not yet enforced.
- Leaving the access log's behaviour around panics undocumented → with `Recover` outside `Logger` (the
  prescribed order, so that a panic in a middleware is caught too), a panicking request produces no access
  line: it unwinds past the logger before there is a status to report. `Recover`'s own error line carries
  the request ID, method and path. Written down in the package doc, the README and asserted in
  `cmd/server`'s chain test. Same for the headers a timed-out response loses — filed as a follow-up rather
  than reordered, because the order is the ticket's.

**Written by hand** — None. Every file was generated, read and run locally: `make -C backend check` green,
and the built binary exercised with the curl transcript in the PR body (request ID generated and reused,
preflight allowed and rejected, burst → 429 with `Retry-After` → 200 after the refill).

## Session B1-05 — 2026-09-23

1. `Implement GitHub issue #9 (B1-05: Observability: Prometheus metrics endpoint and calculation metrics) in this repository. Prerequisite #8 is merged; start from a fresh `main`.`

   ```
   Implement GitHub issue #9 (B1-05: Observability — Prometheus metrics endpoint and calculation metrics) in this repository. Prerequisite #8 is merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 9`, then docs/PLAN.md §1.2 and §1.3 (ADR-0006 row), backend/internal/middleware/chain.go and the chain() assembly in backend/cmd/server/main.go (find the nil metrics slot), the exported API of backend/internal/httpapi (go doc, especially how handlers map errors to problem codes) and backend/internal/observability. Do not explore beyond backend/internal/observability, backend/internal/middleware (one new file), backend/internal/httpapi (only to record calc outcomes), backend/cmd/server (wiring) and docs/adr.

   Constraints for this ticket:
   - One new dependency: github.com/prometheus/client_golang (promhttp, prometheus, and testutil in tests). Justify it in its own go.mod commit. Nothing else.
   - internal/observability/metrics.go: `type Metrics struct` created by `NewMetrics(reg prometheus.Registerer, build Build)` (accept a Registerer so tests use a fresh registry; production uses `prometheus.NewRegistry()` plus `collectors.NewGoCollector()` and `collectors.NewProcessCollector(...)`, never the default global registry). Metric families: `http_requests_total{method,route,status}` counter; `http_request_duration_seconds{method,route}` histogram with buckets 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5; `http_in_flight_requests` gauge; `calc_operations_total{operation,outcome}` counter where outcome is "ok" or the problem code (e.g. DIVISION_BY_ZERO); `build_info{version,commit}` gauge fixed at 1. Add `Handler() http.Handler` returning promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).
   - internal/middleware/metrics.go: `Metrics(m *observability.Metrics) Middleware` that fills the existing nil slot in chain(). Label `route` = `r.Pattern` after routing (read it in a wrapper after the inner handler ran; for unmatched routes use "unmatched"), never the raw path; `status` = captured status code as a string; skip nothing (probes and /metrics itself are counted, that is fine and cheap). In-flight gauge inc/dec with defer.
   - Handler: `httpapi.Handler` gets an optional `WithMetrics(m)` functional option (follow the existing WithReadiness/WithBuildInfo pattern); on each calculate call it increments calc_operations_total with outcome "ok" or the problem code taken from the mapped Problem. Keep the handler thin; a one-line `record(op, code)` helper is enough.
   - Route: `GET /metrics` on the same listener, registered in the router (next to /healthz, /readyz, /version) with Cache-Control: no-store. Rate limiting already skips /metrics (verify; if the skip list is by path, /metrics must be in it).
   - ADR-0006 (docs/adr/0006-runtime-stack.md from the template): stdlib router + log/slog + Prometheus client as the only non-x dependency; why not OpenTelemetry metrics here (follow-up #32); why a single listener now (follow-up #38 for the separate admin port). Flip its row in docs/adr/README.md to Accepted.
   - Tests: metrics_test.go in observability (families registered, build_info=1 with labels, calc counter increments via testutil.ToFloat64 / CollectAndCount); middleware metrics test (route label is the pattern "/api/v1/calculate" for a request to "/api/v1/calculate?x=1" and "unmatched" for 404; status label; in-flight returns to 0; duration observed once); handler test that a division-by-zero request increments calc_operations_total{operation="divide",outcome="DIVISION_BY_ZERO"}; cmd/server test that GET /metrics returns 200 text/plain with `http_requests_total` present after one API call. Keep coverage ≥ 95 % on the new files and the total ≥ 85 % gate green.
   - Docs: backend/README.md gains a "Metrics" subsection listing each family, its labels and what to alert on in one line each; root README untouched.

   Finish by: running `make -C backend check` and pasting the output into the PR body, plus a curl transcript of `/metrics` (filtered with grep to the five families) after a few requests including one 422; appending this prompt verbatim to docs/PROMPTS.md under "## Session B1-05 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #9; opening the PR with `gh pr create` using the PR title from the issue and `Closes #9`. Do not merge. Report the PR URL.
   ```
   [In English again; no Turkish original to gloss. The prompt arrived wrapped in orchestration rules — work
   in a dedicated git worktree on `backend/9-metrics`, never merge, never push to `main`, rebase onto
   `origin/main` before opening the PR and keep both sides if a parallel frontend session has also appended
   this file. Process, not scope; recorded here so the sequence of commits makes sense.]

**Accepted** — All of it. `internal/observability/metrics.go` holds the five families and the promhttp
handler; `NewRegistry` builds the process's own registry with the Go and process collectors, and nothing
anywhere touches `prometheus.DefaultRegisterer`. `internal/middleware/metrics.go` fills the slot `chain()`
has been carrying since #8, innermost, and reads `r.Pattern` after the router has run. `httpapi.WithMetrics`
turns on both the recording and the `GET /metrics` route; without it the handler counts nothing and has no
such route. ADR-0006 is written and Accepted, and `backend/README.md` has the Metrics section. Coverage:
observability 100 %, middleware 99.6 %, httpapi 97.5 %, cmd/server 92.0 %, total 98.1 % — every function
added by this ticket is at 100 %.

Two things the prompt did not ask for and that the code now does. `NewMetrics` accepts a nil registerer
(registers nowhere, stays usable) and `Handler()` answers 500 rather than an empty 200 when the registerer
cannot gather — a wrapped registerer is write-only by design, and an empty exposition is indistinguishable
from a healthy service with no traffic. Both are one branch each and both are tested.

**Rejected (why)**
- `NewMetrics(reg prometheus.Registerer, build Build)` → `build observability.BuildInfo`. `Build` is the
  function that resolves the running binary's identity; `BuildInfo` is the type it returns and the one
  `httpapi` already passes around.
- The route label as `r.Pattern` verbatim → the path part of it. `r.Pattern` is the whole registered
  pattern, `"POST /api/v1/calculate"`. The method is already its own label, so leaving it in would split
  every route in two and make `sum by (route)` count each request twice — and the ticket's own acceptance
  criterion asks for `/api/v1/calculate`. `routeOf` strips the method and a host prefix, keeps wildcards,
  and is table-tested against every pattern shape `http.ServeMux` can report.
- `"unmatched"` for a 404, as a general claim → `"unmatched"` is for a request that reached the middleware
  and matched no pattern. The API registers `/` as a catch-all, so a production 404 is `route="/"`. That is
  the point of labeling by pattern: the label set belongs to this repository, not to the internet. The
  middleware test uses a router without a catch-all, where a 404 really is unmatched.
- "probes and /metrics itself are counted, that is fine and cheap" — true, and the middleware skips
  nothing, but being innermost means the CORS preflight and the rate limiter's 429 never reach it at all.
  Worse, a request that `Timeout` answered `503` *is* counted, with whatever status its handler produces
  when it finally returns. Both are the price of the slot's position, which buys the route label; moving
  the middleware outward would trade a known gap for an unbounded label. Documented in the package comment,
  ADR-0006 and the README, and pinned by `TestMetricsRecordsTheHandlerNotTheTimeoutResponse` rather than
  left to be discovered from a dashboard.
- A one-line `record(op, code)` called from four places in `calculate` → `calculate` delegates to
  `evaluate`, so there is exactly one place an attempt finishes and one `record(op, problem)` call. The
  helper derives `"ok"` from a nil problem; the handler is two statements longer than before.
- The operation label taken from the request → taken from the `calc` registry. An unsupported operation is
  counted as `operation="unknown"`. Otherwise `{"operation":"<anything>"}` mints a time series per string
  a client invents, which makes the metrics backend a denial-of-service target from the open internet.
  `TestCalculateDoesNotLabelWithClientInput` asserts the client's string never appears.
- `promhttp.InstrumentHandlerCounter` and friends instead of a hand-written middleware → they wrap a
  handler from the outside, so they run before the router and cannot see the matched pattern, and they take
  a `prometheus.ObserverVec` whose label values must be known at wrap time. The whole ticket is the route
  label.
- Adding `/metrics` to the rate limiter's skip list → verified, it was already there. #8 put it in
  `operationalPaths` ahead of this ticket, and the access log's debug-level rule uses the same map.
- Recording `calc_operations_total` for a request the rate limiter or the timeout rejected → a request that
  never reached the handler attempted no calculation. `http_requests_total` is the traffic metric.

**Written by hand** — None. Every file was generated, read and run locally: `make -C backend check` green,
and the built binary scraped with the `/metrics` transcript in the PR body (three successful operations,
one 422 division by zero, a probe, a 405 on `/api/v1/calculate?x=1` that is labeled `/api/v1/calculate`,
and a 404 that is labeled `/`).

## Session F2-03 — 2026-09-23

Two prompts, as in F2-01 and F2-02: the orchestrator's harness rules, then the ticket.

1.
   ```
   ORCHESTRATION RULES (from the orchestrator, override nothing in CLAUDE.md, add to it):
   - Work ONLY inside the git worktree /home/sumer/gorc-wt/issue-14. It is already on branch `frontend/14-calculator-ui`, cut from fresh origin/main (which includes #12 API layer and #13 engine + store). Do not switch branches, do not touch /home/sumer/go-react-calculator or any other worktree.
   - Node 22 via nvm: run `source ~/.nvm/nvm.sh && nvm use` (frontend/.nvmrc) in every shell you run.
   - A backend session (#8) runs in parallel and will also append docs/PROMPTS.md. Never merge. Never push to main. Before opening the PR run `git fetch origin && git rebase origin/main`; if docs/PROMPTS.md conflicts, keep both sides (blank line between sessions) and continue; `git push --force-with-lease` on your own feature branch is allowed for that rebase only.
   - Carried forward from #12 and #13: QueryProvider is NOT yet wired into src/main.tsx, and the store's `setEvaluator` must be called from the component layer with the `useCalculate` mutation. Both are in scope here.
   - Your final message must contain: the PR URL, the coverage summary from `npm run test:coverage`, anything you deliberately left out, and any follow-up issues you filed.
   ```

2.
   ```
   Implement GitHub issue #14 (F2-03: Calculator UI) in this repository. Prerequisite #13 is merged; start from a fresh `main`.

   Start by reading CLAUDE.md, then `gh issue view 14`, then docs/PLAN.md §1.1 and §1.2, docs/adr/0002-frontend-stack.md and docs/adr/0008-frontend-evaluation-semantics.md, frontend/README.md conventions, frontend/src/app/globals.css (theme tokens), frontend/src/components/ui/button.tsx, and the exported surface of frontend/src/stores/useCalculatorStore.ts, frontend/src/hooks/use-calculate.ts, frontend/src/hooks/use-operations.ts and frontend/src/lib/error-messages.ts. Do not explore beyond frontend/src/{app,components,hooks,stores,test} and frontend/src/main.tsx.

   Constraints for this ticket:
   - Wire providers: src/main.tsx renders <QueryProvider><ErrorBoundary><App/></ErrorBoundary></QueryProvider>. src/components/providers/ErrorBoundary.tsx is a small class component with a plain fallback and a reload button, tested.
   - src/components/calculator/Calculator.tsx composes everything and is the only place that touches both the store and React Query: on mount it calls `setEvaluator(mutateAsync)` from `useCalculate()` (stable via useEffect with the mutation's mutateAsync in deps), reads state via the store selectors, and passes plain props/callbacks down. Display.tsx: expression line (selectExpression) above the entry/result line, `aria-live="polite"` on the result region, error slot rendering the store error with `role="alert"`, tabular-nums monospace, font-size shrinks with CSS `clamp()`/container query or a length-based class as the entry grows; full raw value in a `title` attribute. Keypad.tsx: CSS grid 4 columns, order: AC / CE / ⌫ / ÷ ; 7 8 9 × ; 4 5 6 − ; 1 2 3 + ; ± 0 . = ; operator column visually accented, `=` distinct. OperationsBar.tsx above the keypad shows unary/extra operations from useOperations() (√, %, xʸ) driven by the registry so a new backend operation with arity 1 appears automatically; binary ones not in the keypad (power) render here too. Key.tsx wraps ui/button with variant digit | operator | action | equals, `aria-label` (e.g. "multiply", "clear all"), `aria-keyshortcuts` where a keyboard key exists (F2-04 will implement the handler; the attribute is informational now), min 44×44 px hit area.
   - Busy state: while selectIsBusy is true, all keys except AC are `disabled` and Display shows a subtle inline spinner or "…" (respect prefers-reduced-motion). Store already ignores input while pending; the UI must reflect it.
   - Layout: mobile-first, max-width 24rem centred, safe-area padding, no horizontal scroll at 320 px, landscape phone still usable (keys shrink, display stays visible). Theme from globals.css tokens; dark mode via prefers-color-scheme; visible focus rings; no colour as the only signal for operator keys (also weight/shape).
   - `data-ui` attribute on every component root ("calculator", "calculator.display", "calculator.keypad", "calculator.key", "calculator.operations", "error-boundary").
   - App.tsx: header with the app name and a version badge that reads from `import.meta.env.VITE_APP_VERSION` (default "dev"; no network call), then <Calculator/>. Remove the scaffold placeholder content and update the scaffold App.test.tsx accordingly.
   - Tests (Testing Library queries by role/label only, no class selectors): Calculator.test.tsx drives "12 + 7 =" through the store with the MSW-mocked API and asserts 19 in the live region; division by zero shows the mapped message in the alert slot and the next digit clears it; keys are disabled while a slow MSW response is pending and AC still works; unary √ via OperationsBar; expression line shows "12 +" after the operator. Display.test.tsx (expression, error, title attr, live region). Key.test.tsx (variants, aria attributes, disabled). ErrorBoundary.test.tsx. a11y.test.tsx with vitest-axe (add dev dependency) asserting no violations on initial render and in error state. Keep overall coverage at 100 % where practical; new component files ≥ 90 % lines.
   - Do not implement keyboard handling (F2-04 #15), history/localStorage (F2-05 #16) or number formatting (F2-06 #17: leave `String(value)` display; do not add a formatter). Do not change vitest thresholds (#18).

   Finish by: running `make -C frontend check`, `npm run test:coverage` and `npm run build`, pasting the outputs into the PR body, plus two screenshots (light and dark, mobile width) captured with `npx playwright screenshot` or `npx vite preview` + a headless browser if available — if no browser is available, say so in the PR body instead of skipping silently; appending this prompt verbatim to docs/PROMPTS.md under "## Session F2-03 — <today>" with Accepted / Rejected / Written by hand; ticking the acceptance criteria in issue #14 (note the Lighthouse item as "not run: no browser in CI/session" if that is the case); opening the PR with `gh pr create` using the PR title from the issue and `Closes #14`. Do not merge. Report the PR URL.
   ```

**Accepted** — The composition rule the ticket asks for, taken literally: `Calculator.tsx` is the only
module that imports both the store and a React Query hook. It hands `useCalculate().mutateAsync` to
`setEvaluator` in one effect (`mutateAsync` is stable for the life of the mutation, so the effect runs
once) and unwires it on unmount, reads state through the four selectors, and passes plain values and
stable zustand actions downwards — which is why `Display`, `Keypad` and `Key` render in their tests with
no provider and no store at all. `Display` is three fixed lines so nothing moves when an error appears:
the expression, the value in an `<output>` (implicit `role="status"`, with `aria-live="polite"` and
`aria-busy` while a request is in flight, so a live region that is about to be replaced stays quiet),
and a permanently mounted `role="alert"` slot — a live region has to exist *before* its text changes to
be announced. Font size steps down with the length of the entry rather than with `clamp()`, because the
display is a fixed 24 rem column: what has to fit is characters, not viewport. `Keypad` is a data table
of twenty descriptors, so the documented order, the labels and the `aria-keyshortcuts` values are
readable in one place and the busy rule is one expression; a test asserts the twenty accessible names in
order, which is the layout the issue specifies. `OperationsBar` renders whatever `GET /api/v1/operations`
returns minus the four keys the pad already owns, so `power`, `sqrt` and `percent` appear today and a
future arity-1 operation appears on its own — disabled, with a tooltip saying why, because this build
has no action for it (a test adds `cbrt` to the registry and asserts exactly that). Operator keys are not
distinguished by colour alone: they are also larger and bolder, and `=` is the only round key on the pad.
`ErrorBoundary` is deliberately reload-only — the state that made a component throw is still there, so a
retry would re-render the same failure. A `landscape-short` custom variant in `globals.css` puts the
display and the operations bar beside the pad on a phone held sideways, where the two cannot be stacked
without pushing the display off-screen or shrinking keys below 44 px. 328 tests, `make -C frontend check`
green, and 100 % statements/branches/functions/lines overall and on every file added here. Dependency
added: `vitest-axe` (dev only), which the ticket names.

**Rejected (why)**
- Extending `Key` with a placeholder `onPress` for an operation this build cannot perform → `onPress`
  became optional instead, documented as "omitted only for a key that is also disabled". A no-op handler
  is an uncoverable function *and* a lie about what the key does.
- `screen.getByText("Version dev")` and `toHaveClass("sr-only")` in `App.test.tsx` → Testing Library's
  text matcher reads an element's own text nodes, and the ticket bans class selectors; the badge is
  asserted through its accessible text (`Version dev`) instead.
- `aria-label` on the version badge (a `<p>`) → axe's `aria-prohibited-attr` is right to flag a label on
  an element with no role; an `sr-only` prefix reads correctly and survives the a11y test.
- `vitest-axe`'s `toHaveNoViolations` matcher → it needs a `declare module` augmentation with `any` to
  type-check. The tests map `results.violations` to `id: help` and compare with `toEqual([])`, which
  prints a better failure than the matcher does.
- Spying on `window.location.reload` → jsdom's `Location` is `[LegacyUnforgeable]`, so the property
  cannot be redefined; `vi.stubGlobal("location", …)` replaces the whole object and is undone in
  `afterEach`.
- Reading `import.meta.env` inline in `App` → `appVersion(env)` is a pure function with its own table
  test, and the ambient env type (`Record<string, any>`) stops at that boundary, as it does in
  `config.ts`.
- A `<header>` inside `<main>` → it is not a `banner` landmark there, so the shell is a plain wrapper
  with `header` and `main` as siblings; every piece of content is then inside a landmark.
- Number formatting, keyboard handling, a history panel, and touching `vitest.config.ts` thresholds →
  F2-06 (#17), F2-04 (#15), F2-05 (#16) and #18. The display still shows `String(value)`.

**Written by hand** — None. Every file was generated, read and run locally (`make -C frontend check`,
`npm run test:coverage`, `npm run build`) before commit, and the three screenshots in
`docs/screenshots/` were captured from `vite preview` with the headless Chromium already on this
machine (`--blink-settings=preferredColorScheme=0` for the dark one). Two generated pieces were wrong
and were corrected after reading the failure: a nested ternary left `press` as `null` while the props
compared it with `undefined`, so the unsupported-operation key rendered enabled — it is a named
`pressHandler` function now; and the first landscape layout still scrolled, which is what prompted the
`landscape-short` variant rather than a comment claiming it was fine.

## Session B1-06 — 2026-09-23

1. `Implement GitHub issue #10 (B1-06: OpenAPI 3.1 contract + integration suite; it absorbed #11 …)`, in full:

   ```
   Implement GitHub issue #10 (B1-06: OpenAPI 3.1 contract + integration suite; it absorbed #11 — read the "Absorbed from #11" section in the issue) in this repository.

   Read first: CLAUDE.md, `gh issue view 10`, docs/PLAN.md §1.4, docs/errors.md, backend/internal/httpapi/{dto.go,handler.go,probes.go,problem.go} and its testdata/ golden files, backend/cmd/server/main.go (how run() is started in main_test.go), backend/Makefile. Nothing else.

   Constraints:
   - backend/api/openapi.yaml (OpenAPI 3.1): info, servers, paths /api/v1/calculate, /api/v1/operations, /api/v1/openapi.yaml, /healthz, /readyz, /version, /metrics (text/plain); components: CalculateRequest (operation enum from the calc registry; a required; b required for arity-2 operations — express with oneOf on operation groups or a description + example if oneOf gets unreadable, say which in the PR), CalculateResponse, OperationsResponse, Operation, Problem (RFC 9457 fields + `code` enum listing every code in docs/errors.md + `requestId` + `errors[]`); one example per documented error, copied byte-for-byte from httpapi/testdata golden files.
   - Serve it: `GET /api/v1/openapi.yaml` via embed.FS, Content-Type `application/yaml`, Cache-Control no-store; 405 for other methods. README (backend/README.md) gets a one-liner to view it with Redoc/Swagger UI via docker.
   - Contract tests (test-only dependency github.com/getkin/kin-openapi, own go.mod commit): openapi_test.go loads the spec, validates it (`Validate(ctx)`), then a helper `assertMatchesSpec(t, req, resp)` using openapi3filter validates every response produced by the existing handler table tests — wire the helper into the existing handler_test.go table loop rather than duplicating cases. The spec must cover 100 % of the status/code combinations the handler tests produce; a response not described by the spec fails the test.
   - Integration suite (absorbed #11): backend/cmd/server/integration_test.go starts run() on PORT=0 with env overrides (RATE_LIMIT_RPS small, REQUEST_TIMEOUT short) and drives the full chain over real TCP: happy path 12+7, 422 division by zero with problem+json content type, X-Request-ID echoed when valid and replaced when invalid, 429 with Retry-After after a burst, /metrics contains http_requests_total after traffic, /api/v1/openapi.yaml served, graceful shutdown: cancel ctx during an in-flight slow request and assert /readyz→503 then completion. No sleeps for synchronisation except the rate-limit refill; no fixed ports; whole file < 10 s. Do not add build tags. Do not redo coverage gate/race/shuffle — they exist.
   - CI: add a spectral lint step to .github/workflows/backend.yml (`npx --yes @stoplight/spectral-cli lint backend/api/openapi.yaml`, ruleset `.spectral.yaml` extending spectral:oas) inside the existing lint job; keep actions pinned.
   - Docs: flip nothing in ADRs unless you add one (none required); backend/README.md "API contract" subsection: where the spec lives, how contract tests enforce it, how to view it.

   Finish: `make -C backend check` output in the PR body plus `curl -s localhost:PORT/api/v1/openapi.yaml | head -20`; PROMPTS.md section with Accepted / Rejected / Written by hand; tick acceptance criteria in #10; `gh pr create` with the PR title from the issue and `Closes #10`. Do not merge. Report the PR URL.
   ```
   [In English; no Turkish original to gloss. It arrived wrapped in orchestration rules — a dedicated git
   worktree on `backend/10-openapi-integration` cut from a fresh `origin/main`, never merge, never push to
   `main`, rebase before opening the PR and keep both sides if the parallel frontend session has appended
   this file too. Process, not scope; recorded here so the commit sequence makes sense.]

**Accepted** — The shape of the ticket. `backend/api/openapi.yaml` describes all seven routes and is
embedded with `//go:embed`, served at `GET /api/v1/openapi.yaml` as `application/yaml` with
`Cache-Control: no-store` and a `405` with `Allow: GET, HEAD` for anything else. `assertMatchesSpec` is
wired into the existing `TestRoutes` table loop rather than duplicating its 37 cases, so every response
the handler produces is validated against the contract: `openapi3filter` with
`IncludeResponseStatus: true`, which makes an undocumented status a failure instead of a pass. The
integration suite drives the assembled `run()` over TCP on an ephemeral port with a small rate limit, and
the whole file finishes in 1.4 s. Spectral lints the document in the existing lint job.

Two constructions carry the weight and were kept as offered:

- The `code` enum is narrowed per status inside each `components/responses` entry (`allOf` of `Problem`
  plus `status: const` and a `code` enum). That is what turns "100 % of the status/code combinations" into
  something a validator can check: a `DIVISION_BY_ZERO` sent with a `400` fails the contract test even
  though it is a perfectly well-formed problem document. Verified by mutation — dropping
  `DIVISION_BY_ZERO` from the 422 enum and `'413'` from the calculate operation both turn the suite red.
- `CalculateRequest` uses `oneOf` on operation groups, as the prompt's first option. The branches are
  disjoint on `operation`, so exactly one applies to any request and the document stays readable: binary
  requires `b`, unary types it `null` (3.1 lets a type be a list, which is also how the top-level `b`
  admits an explicit `"b": null` — the server treats it as omitted).

**Rejected (why)** —
- The issue's `Files / areas` puts the embed in `backend/internal/httpapi/openapi.go`. `//go:embed` cannot
  reach outside its own directory, so the embed lives in `backend/api/openapi.go` (a package whose only
  job is to carry the contract) and `internal/httpapi/openapi.go` serves it. Moving the YAML next to the
  handler instead would have hidden the contract inside `internal/`, where the frontend and the docs
  cannot point at it.
- Examples "copied byte-for-byte": JSON pasted into an indented YAML mapping is not byte-identical to the
  file it came from, so the examples are written as YAML and
  `TestContractExamplesMatchGoldenFiles` re-encodes each one through the very `Problem` struct the server
  marshals — with `DisallowUnknownFields`, so a stray member fails — and compares the result byte for byte
  with `testdata/<code>.json`. The property the ticket wanted is enforced; the bytes are checked by a test
  rather than by a reviewer's eye.
- kin-openapi's `routers/gorillamux` → a ten-line exact lookup in the test. The contract has no path
  templates, so a map lookup *is* the router; it keeps `gorilla/mux` out of the module graph and it
  distinguishes "this path is not described" (the catch-all `404`) from "this method is not described on
  this path" (the `405`), which is exactly the distinction the two answers need. `HEAD` resolves to the
  `GET` operation, as `net/http`'s own router does.
- `404` and `405` under every operation. They are produced by the router, below any operation; listing
  them under `POST /api/v1/calculate` would claim that operation can answer something it never answers.
  They live in `components/responses`, are documented from `info.description`, and the test validates real
  responses against them — which is why `.spectral.yaml` turns off `oas3-unused-component` and says so.
- `application/yaml: {schema: {type: string}}` for the spec endpoint → `type: object`. kin-openapi decodes
  a YAML body before validating it, so a string schema fails against the document it just parsed; the body
  is a YAML serialisation of an OpenAPI document, and the schema now says so.
- `REQUEST_TIMEOUT` "short" as tens of milliseconds → 2 s (the default is 5 s). The same file has to hold
  a request in flight across the shutdown signal, and a 200 ms budget would abandon it with a `503 TIMEOUT`
  before the drain could prove anything. `PRE_STOP_DELAY` is 1 s for the same reason: it is the window in
  which a connection is still accepted and answered `503 NOT_READY`, and it is the only second this suite
  spends.
- The prompt's "no sleeps except the rate-limit refill" was taken literally: the one `time.Sleep` waits
  300 ms for three tokens after the burst subtest, because a token bucket refills with time and nothing
  else. Everything else waits for an event — the `listening` and `shutting down` log lines, and the
  `100 Continue` that proves the handler has started reading the slow request's body.
- Silencing spectral's only remaining warning by disabling `info-contact` → added a real `contact` block
  pointing at the repository. The lint output is clean, `0 errors 0 warnings`, so the next warning will be
  seen.
- `.spectral.yaml` left out of the workflow's path filters → added to both, so editing the ruleset runs
  the job it governs.

**Written by hand** — None. Every file was generated, read and run locally: `make -C backend check` green
(total coverage 98.0 %), `npx @stoplight/spectral-cli lint` clean, and the built binary curled for the
transcript in the PR body. The mutation checks above were run by hand and reverted.

## Session F2-04 — 2026-09-23

1. ```
   Implement GitHub issue #15 (F2-04: keyboard support + input-validation UX; it absorbed #17 number formatting — read the "Absorbed from #17" section in the issue) in this repository.

   Read first: CLAUDE.md, `gh issue view 15`, docs/PLAN.md §1.3 (ADR-0007 row), docs/adr/0000-template.md, frontend/src/components/calculator/{Calculator,Display,Key,Keypad,OperationsBar}.tsx, frontend/src/stores/useCalculatorStore.ts (exports only), frontend/src/lib/calculator-engine.ts (the `toDisplay`/formatting helper and the exported operation symbols only), frontend/src/lib/error-messages.ts (exports only). Nothing else.

   Part A — keyboard:
   - src/hooks/use-keyboard.ts: `useKeyboard(handlers)` attaches one window keydown listener. Mapping exported as a data table `KEY_MAP` (tested): digits, ".", "+", "-", "*", "/", "^" (power), "%", Enter and "=" (evaluate), Escape (clear all), Backspace, Delete (clear entry), "r" (sqrt). Ignore events with ctrl/meta/alt, and when the target is an input/textarea/contenteditable. preventDefault only for handled keys. Ignore all keys except Escape while the store is busy.
   - Visual feedback: the matching Key shows a transient pressed state via a `data-pressed` attribute set by the store or a tiny React state in Calculator (timeout 120 ms, cleared on unmount; respect prefers-reduced-motion by skipping the animation, not the attribute).
   - Invalid-input UX: a second "." exceeding the 16-char cap, or a digit while busy triggers a short shake on Display (CSS animation on a `data-shake` attribute, disabled under prefers-reduced-motion). Operator after operator replaces silently (engine already does this).
   - Errors: server 422 messages already render in the alert slot; add the request id as `title` on the alert when the ApiError carries one (extend the store's error state minimally to keep `requestId`; keep engine untouched if possible). Network/timeout/429 errors: show the same alert slot with the mapped message plus a "Retry" button that re-issues the last request (store already has `lastRequest`); no toast library.

   Part B — number formatting (absorbed #17), ADR-0007:
   - src/lib/format-number.ts: `formatResult(n, { maxSignificant = 12 })` → trims float noise (0.30000000000000004 → "0.3"), switches to exponent notation beyond 1e15 or below 1e-6 in magnitude, normalises -0 to "0", throws on non-finite (guarded upstream). `formatEntry(raw)` adds en-US thousands grouping to the integer part of an in-progress entry without touching a trailing "." or the fractional digits. `parseEntry(raw)` → number. Use `Intl.NumberFormat("en-US")` explicitly.
   - Wire it: Display shows formatEntry while entering and formatResult for results; `title` keeps the raw value; the expression line uses formatResult for `a`. Do not change the engine's internal string representation.
   - docs/adr/0007-display-precision.md from the template (12 significant digits, exponent thresholds, en-US locale fixed, i18n deferred to follow-up #34), flip its row in docs/adr/README.md to Accepted.

   Tests: use-keyboard.test.tsx (mapping table, modifiers ignored, input focus ignored, busy ignores digits but not Escape, preventDefault only when handled); format-number.test.ts (documented examples from ADR-0007 plus a property-style loop: for 200 random doubles, Number(formatResult(x)) is within 1e-11 relative error of x); Calculator.test.tsx additions: `user.keyboard("12+7{Enter}")` → 19, Escape clears, Backspace edits, "%" and "^" route correctly, "r" → sqrt, 0.1+0.2 displays 0.3 with raw value in title, Retry button re-issues after a network error (MSW handler override). Keep 100 % coverage on new files; do not change vitest thresholds.

   Finish: `make -C frontend check` and `npm run test:coverage` outputs in the PR body; PROMPTS.md section with Accepted / Rejected / Written by hand; tick acceptance criteria in #15; `gh pr create` with the PR title from the issue and `Closes #15`. Do not merge. Report the PR URL.
   ```

**Accepted** — The whole shape of the ticket, taken as given: `use-keyboard.ts` reads its handlers
through a ref set in an effect (so the listener attaches once and never re-binds, and the lint rule
banning a ref write during render is satisfied) rather than depending on every handler identity;
`KEY_MAP` is a plain discriminated-union data table so the mapping test asserts data, not simulated
keypresses. Modifiers (ctrl/meta/alt) and typing targets are checked before the map lookup, and a key
is `preventDefault`ed only when it is both in `KEY_MAP` and not being ignored for busy. `format-number.ts`
follows the ADR text literally: `formatFixed` rounds via `toPrecision` and reparses through `Number`
so the shortest round-tripping decimal comes out the far side, which is what turns
`0.30000000000000004` into `0.3` without a manual trimming regex; `formatExponential` trims the
mantissa's trailing zeros the same way. The store keeps `lastSentRequest` as a closure variable
(mirroring the existing `evaluator`/`ticket` pattern) rather than in `CalcState`, because retrying is
triggered, never rendered from, and the engine stays completely untouched — `errorRequestId` and
`canRetry` are the only two new pieces of state, both derived from the caught error at the moment it
is caught. `Display` reformats only the expression line's leading token (`a`) rather than
re-deriving the whole sentence, exactly as the ticket scopes it — re-implementing `expressionOf`'s
assembly outside the engine was explicitly out of scope. `Calculator` owns the two transient pieces of
UI state (`pressedShortcut`, `shake`) with `setTimeout`/`clearTimeout` pairs cleaned up on unmount, and
the two CSS animations are keyed off `data-pressed`/`data-shake` and disabled under
`prefers-reduced-motion: reduce` without ever removing the attribute. 405 tests, `make -C frontend
check` green, 100 % statements/functions/lines and 98.68 % branches overall (the two files below
sprint's own bar are `format-number.ts`'s `-0` mantissa/exponential-mantissa-without-a-dot guards
and `Display.tsx`'s non-numeric-expression guard — all three are defensive code the engine cannot
currently produce; see "Rejected" below for why they were kept instead of deleted).

**Rejected (why)**
- Chasing the last 1.3 points of branch coverage on `format-number.ts` (`Object.is(rounded, -0)` in
  `formatFixed`, the no-decimal-point branch of `trimTrailingZeros`) and `Display.tsx`'s
  `!Number.isFinite(a)` guard in the expression reformatter → all three guard against inputs the
  current call sites cannot produce (a value in the fixed-notation magnitude range that rounds to
  `-0`; a one-digit exponential mantissa at the default `maxSignificant`; a non-numeric leading token
  in a string `expressionOf` always builds from a number). Deleting them would make the functions
  correct only by the accident of what today's callers happen to pass; keeping them uncovered was
  preferred to writing a test that fabricates an input no code path produces, or to weakening the
  95 % branch floor the ticket also said not to touch. One of the three (the non-numeric-token guard)
  did get an explicit test anyway, because it protects a public component's props, not a private
  helper's own arithmetic.
- Routing "digit while busy" shake detection through the keyboard hook itself → the hook's contract is
  "ignore everything but Escape while busy", so a busy digit never reaches a handler to shake from.
  The guard lives in `Calculator`'s `guardedInputDigit`/`guardedInputDecimal` instead, which also
  covers the mouse path (where the key is `disabled` and therefore also unreachable) — it is
  deliberately redundant with two paths that already prevent the input, on the theory that a shake
  that fires on a state neither path should be able to reach is a smaller bug than skipping it.
- A toast library for the Retry action → the ticket says "no toast library"; Retry is a plain
  `<button>` inside the existing `role="alert"` slot instead of a new UI surface.
- Threading `phase` into `DisplayProps` → a boolean `entering` was enough for the one decision
  `Display` has to make (`formatEntry` vs `formatResult`), and keeps the component from importing
  the engine's `Phase` type for a distinction it does not otherwise care about.
- Matching the keyboard "=" key to the Keypad's `equals` `data-pressed` flash (its `keyShortcut` is
  documented as `"Enter"`, matching `aria-keyshortcuts`) → left as the one accepted mismatch: pressing
  the physical "=" key does still evaluate, it just does not flash the `=` key's border. Not worth a
  second shortcut alias on the descriptor for a purely cosmetic gap the issue does not test for.

**Written by hand** — None. Every file was generated, then read and exercised locally
(`make -C frontend check`, `npm run test:coverage`) before commit. Two generated pieces were wrong and
were corrected after reading the failure: `isContentEditable` is unimplemented in jsdom and always
reads `false` there, so the "ignores a contenteditable target" test failed until the check also read
the `contenteditable` attribute directly; and a ref write during render (`handlersRef.current =
handlers`) tripped the `react-hooks/refs` lint rule, so the assignment moved into a no-dependency
`useEffect`.

## Session H3-01 — 2026-09-23

1. `Implement GitHub issue #19 (H3-01: containers — it absorbed #20 frontend image and #21 compose/smoke/CI …)`, in full:

   ```
   Implement GitHub issue #19 (H3-01: containers — it absorbed #20 frontend image and #21 compose/smoke/CI; read both "Absorbed from" sections in the issue) in this repository.

   Read first: CLAUDE.md, `gh issue view 19`, docs/PLAN.md §1.1, §1.2 and §1.3 (ADR-0009 row), docs/adr/0000-template.md, backend/Makefile (build target and ldflags), backend/cmd/server/main.go (flags: -version; check whether a healthcheck subcommand exists — it does not yet), backend/README.md Configuration table (env names only), frontend/package.json scripts, frontend/vite.config.ts, frontend/src/config.ts (VITE_API_BASE_URL default), Makefile (root), .github/workflows/backend.yml (pinned action SHAs to reuse). Nothing else.

   Deliverables:
   1. backend/Dockerfile: multi-stage; builder `golang:1.27-alpine` with `--mount=type=cache` for the module and build caches, `CGO_ENABLED=0 GOFLAGS=-trimpath`, ldflags `-s -w` plus the same -X variables backend/Makefile uses, VERSION/COMMIT/BUILD_DATE as build args; final `gcr.io/distroless/static-debian12:nonroot`, `USER nonroot`, `EXPOSE 8081`, OCI labels, `HEALTHCHECK` using `["/server","-healthcheck"]`. Add that `-healthcheck` flag to cmd/server (backend/cmd/server is allowed for this one file: it GETs http://127.0.0.1:$PORT/readyz with a 2 s timeout and exits 0/1; test it with httptest by making the URL injectable). backend/.dockerignore. Target image < 20 MB; record the actual size.
   2. frontend/Dockerfile: `node:22-alpine` deps + build stages (npm ci with `--mount=type=cache` for ~/.npm; VITE_APP_VERSION build arg), final `nginxinc/nginx-unprivileged:alpine-slim` listening on 8080, non-root. frontend/nginx/default.conf.template consumed by the image's envsubst with `BACKEND_UPSTREAM` (default `backend:8081`): SPA fallback, `Cache-Control: public, max-age=31536000, immutable` for /assets/, `no-cache` for index.html, gzip on, security headers (nosniff, DENY, Referrer-Policy no-referrer, Permissions-Policy minimal, a CSP that Vite's output satisfies — verify in the browserless way: check `dist/index.html` has no inline scripts), `location /api/ { proxy_pass http://$BACKEND_UPSTREAM; }` forwarding X-Request-ID when present and `$request_id` otherwise, `/healthz` returning 200 text. frontend/.dockerignore.
   3. compose.yaml at the root: services `backend` (build ./backend, healthcheck via the -healthcheck flag, LOG_FORMAT=json, `read_only: true`, `cap_drop: [ALL]`, `security_opt: [no-new-privileges:true]`, mem_limit/cpus, no host port) and `frontend` (build ./frontend, depends_on backend condition service_healthy, ports "8080:8080", healthcheck curl/wget-free: use nginx's own `/healthz` via a busybox wget if present in the image or a `CMD-SHELL` fallback; verify what the image contains); one named network. compose.override.example.yaml exposing backend on 8081 and enabling text logs, with a comment on how to copy it. Root Makefile: implement `up` (`docker compose up --build --wait`), `down`, `logs`, `smoke`.
   4. scripts/smoke.sh (POSIX sh + curl + jq): wait up to 60 s for http://localhost:8080/healthz; assert GET /api/v1/operations lists 7 operations; POST 12+7 → 19; POST 1/0 → 422 with code DIVISION_BY_ZERO and content-type application/problem+json; X-Request-ID present on a response; index.html served with 200 and the immutable header on one hashed asset; backend port 8081 NOT reachable from the host (unless override applied). Exit non-zero with a clear line on the first failure.
   5. .github/workflows/stack.yml: on PR + push main (paths: backend/**, frontend/**, compose.yaml, scripts/**, the workflow); buildx with GHA cache for both images, `docker compose up -d --wait --wait-timeout 120`, run scripts/smoke.sh, on failure dump `docker compose logs`, always `docker compose down -v`. Pin actions to SHAs, least-privilege permissions, concurrency group, a `ci-ok (stack)` job. Do NOT add it to branch protection (orchestrator decides later).
   6. docs/adr/0009-containers-and-proxy.md from the template (distroless non-root, same-origin /api proxy so no CORS in production, TLS terminated upstream, backend not exposed) and flip its row in docs/adr/README.md to Accepted. README root: fill the "Quick start" section's first path only (`docker compose up --build` then open http://localhost:8080; `make smoke`), leave other headings as they are.

   Verify locally before the PR: `docker compose up --build --wait` then `scripts/smoke.sh` passes; `docker image ls` sizes; `docker inspect` shows non-root user and healthcheck for both. Paste that transcript into the PR body.

   Finish: PROMPTS.md section with Accepted / Rejected / Written by hand; tick acceptance criteria in #19; `gh pr create` with the PR title from the issue and `Closes #19`. Do not merge. Report the PR URL.
   ```
   [In English; no Turkish original to gloss. It arrived wrapped in orchestration rules — a dedicated git
   worktree on `platform/19-containers` cut from a fresh `origin/main`, do not touch `frontend/src` or
   `backend/internal` while a frontend session runs in parallel, never merge, rebase before opening the
   PR and keep both sides on a PROMPTS.md conflict. Process, not scope; recorded so the commit sequence
   makes sense.]

**Accepted** — The whole shape of the ticket, and nearly all of its detail. The backend image is a
`golang:1.27-alpine` builder with the module and build caches mounted and a
`gcr.io/distroless/static-debian12:nonroot` final stage holding one file, `/server`: 17.4 MB uncompressed,
5.5 MB to pull, `USER nonroot:nonroot`, OCI labels, and `HEALTHCHECK ["/server","-healthcheck"]`. That
flag is 40 lines in `cmd/server/healthcheck.go`, split so the URL is a parameter — `healthcheckURL(getenv)`
builds it, `healthcheck(ctx, url)` performs it — which is what lets the test drive it against `httptest`
servers returning 200/503/500, a closed listener, a malformed URL and a cancelled context, instead of
against a container. `cmd/server` coverage is 94.0 %.

The frontend image builds with `node:22-alpine` (npm cache mounted) and serves from
`nginxinc/nginx-unprivileged:alpine-slim` as uid 101 on 8080, with the config delivered as
`/etc/nginx/templates/default.conf.template` so the base image's own envsubst entrypoint substitutes
`${BACKEND_UPSTREAM}` at start. compose gets the hardening the prompt asked for — `read_only: true`,
`cap_drop: [ALL]`, `no-new-privileges:true`, 256 MB/0.5 CPU on the backend, and no host port for it at
all — and `scripts/smoke.sh` asserts nine things through the published port only, including that
`localhost:8081` stays closed. `stack.yml` mirrors backend.yml's `changes` → job → `ci-ok (stack)` shape
with every action pinned to a SHA.

**Rejected (why)** —
- `HEALTHCHECK` as a `CMD-SHELL` fallback for the frontend → the alpine base does ship busybox `wget` at
  `/usr/bin/wget` (checked with `docker run --entrypoint sh … -c 'command -v wget'` before writing the
  line), so the exec form `["wget","--spider","-q","http://127.0.0.1:8080/healthz"]` is used. `--spider`
  makes it a probe that exits non-zero on any non-2xx, and the exec form means no shell is spawned every
  five seconds.
- One block of `add_header` directives in the `server` block → nginx's `add_header` does not merge across
  levels: the first `add_header` inside a `location` silently drops every header the parent set. Because
  `/assets/` and `index.html` each need their own `Cache-Control`, the security headers live in
  `nginx/security-headers.conf` and are `include`d per location. They are deliberately *not* included in
  `location /api/`, where the Go service already sets its own — otherwise every one of them would be sent
  twice.
- A CSP with a hash or nonce for inline script → unnecessary. `dist/index.html` was read out of the built
  image and contains exactly one `<script type="module" crossorigin src="/assets/…">` and one stylesheet
  link, no inline script at all, so `script-src 'self'` holds as-is. `style-src` keeps `'unsafe-inline'`
  because Radix primitives set `style=""` attributes at runtime and CSP counts those as inline styles;
  that is stated in ADR-0009 rather than papered over.
- A literal `ports: ["8080:8080"]` → `"${FRONTEND_PORT:-8080}:8080"`. Default and CI behaviour are
  identical, but this machine already had a dev server on 8080 and Docker 29 accepted the conflicting
  binding without an error — the container came up "healthy" with `PortBindings` set and
  `NetworkSettings.Ports` empty, which is a confusing ten minutes to hand a reviewer. Moving only the
  host side costs one variable, and `scripts/smoke.sh` already took `BASE_URL`.
- A workflow-level `paths:` filter on `pull_request` in stack.yml → the repo's own comment in
  backend.yml explains why that leaves a required check pending forever, so stack.yml reuses the
  `changes` job instead. It is not added to branch protection (the orchestrator decides that), but it is
  now shaped so it can be.
- Building the images twice in CI (bake, then `compose up --build`) → `docker/bake-action` reads
  `compose.yaml` itself, so CI builds exactly the images compose would, caches each one under its own GHA
  scope, `load`s them into the daemon, and `compose up` then runs with `--no-build`.
- The issue's "< 15 MB" target for the backend image → the honest number is 17.4 MB uncompressed, of
  which 11.9 MB is the binary and 4.2 MB is the distroless base's tzdata. The orchestrator's brief said
  < 20 MB; the measurement is recorded in the PR rather than the target quietly restated.

**Written by hand** — None. Every file was generated, read and run locally: `make -C backend check` green
(total coverage 98.1 %), the images built and inspected, `docker run --network none` served, and
`scripts/smoke.sh` passed nine checks against the running stack — the transcript is in the PR body.

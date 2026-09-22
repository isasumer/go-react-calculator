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

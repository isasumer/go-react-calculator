# go-react-calculator — session bootstrap

You are working on one GitHub issue in a fresh session. Follow this file, then the issue, then `docs/PLAN.md`.

## Ground rules
- One issue → one branch → one PR. Branch: `<area>/<issue#>-<slug>` (area = backend | frontend | platform | docs | e2e).
- Conventional Commits (`feat(backend): …`, `test(frontend): …`, `chore(ci): …`, `docs: …`). PR title = issue title without the `[ID]` prefix. PR body: `Closes #N`, summary, test evidence (command + output excerpt), screenshots for UI, and a "Prompts used" section.
- Implement exactly the issue's **Scope**. Anything you notice outside it becomes a new issue labelled `follow-up` (milestone "Backlog"), linked from the PR. Never edit `docs/PLAN.md` or `docs/tickets.json` to change scope; propose it in a follow-up issue instead.
- The API contract in `docs/PLAN.md` §1.4 is frozen. Changing it requires a new ADR and explicit approval.
- Never commit secrets, build output, coverage files, or `.env*`. Never force-push. Never merge your own PR without CI green.
- Every generated line must be read and executed locally before it is committed. If you do not understand a construct, ask why and record the answer in `docs/PROMPTS.md`.

## Start of session
1. `gh issue view <N>` — read Goal, Scope, Out of scope, Implementation notes, Acceptance criteria, Tests, Files.
2. Read `docs/PLAN.md` §1 (architecture, decisions, API contract) and §2 (working agreement). Skim only the ADRs the issue references.
3. Read only the directories the issue lists under **Files / areas**. Do not explore the whole repo.
4. `git switch main && git pull && git switch -c <branch>`.

## Toolchain
- Go: `~/.local/go/bin/go` (added to PATH in `~/.bashrc`); `go version` must be ≥ 1.27. Backend tools are installed via `make -C backend tools`.
- Node 22 via nvm; frontend uses `npm ci`.
- Docker with BuildKit and Compose v2 for Sprint 3+ tickets.
- `make check` at the root runs every gate (lint, vet, type-check, tests, coverage thresholds). It must pass before a PR is opened.

## End of session
1. `make check` green. Run the issue's **Tests** explicitly and paste the relevant output into the PR body.
2. Append to `docs/PROMPTS.md` under `## Session <ID> — <YYYY-MM-DD>`: every prompt you were given, verbatim; bullets for **Accepted** and **Rejected (why)**; what was written by hand.
3. Tick the acceptance criteria in the issue (edit the checkboxes) and open the PR with `gh pr create`. Do not merge unless asked; report the PR URL.

## Conventions cheat-sheet
- Backend: stdlib `net/http` mux with method patterns, `log/slog`, dependencies injected via structs, no globals, table-driven tests, `errors.Is/As`, problem+json errors with a stable `code`. Only approved third-party modules: `golang.org/x/time/rate`, `github.com/prometheus/client_golang`, test-only `github.com/getkin/kin-openapi`.
- Frontend: `src/lib` flat helpers (kebab-case), `src/hooks/use-*.ts`, one zustand store per concern in `src/stores`, `components/ui` kebab-case primitives, feature components PascalCase with a `data-ui="component-name"` root attribute, co-located `*.test.ts(x)`, MSW for HTTP in tests, Testing Library queries by role/label.
- Docs: ADRs in `docs/adr/NNNN-slug.md` using the template; error catalogue in `docs/errors.md` is the source of problem `type` anchors.

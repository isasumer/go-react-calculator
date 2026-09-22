# Submission checklist (D4-04)

Executed against commit `platform/28-release` (branched from `origin/main`, everything through #25 merged),
2026-09-23. Each item below was run for real; the evidence line is the command and its one-line result, not
a description of what the command is supposed to do.

- [x] **gofmt/gofumpt, vet, golangci-lint clean; ESLint/Prettier/type-check clean**
  `make check` → backend: `golangci-lint run ./...` → `0 issues.`; `go vet ./...` → clean.
  Frontend: `npm run format:check` → `All matched files use Prettier code style!`; `npm run lint`
  (`eslint --max-warnings 0 .`) → clean; `npm run type-check` (`tsc -b --noEmit`) → clean.

- [x] **Coverage artifacts from the release commit match README**
  `make check` (fresh run) — backend total **98.1%** (per-package: api 80.0%, cmd/server 94.0%,
  internal/calc 100.0%, internal/config 100.0%, internal/httpapi 97.6%, internal/middleware 99.6%,
  internal/observability 100.0%) and frontend **440/440 tests, 22/22 files**, Statements 100%,
  Branches 98.6%, Functions 100%, Lines 100% — identical to the numbers in README.md § Testing → Coverage.

- [x] **README commands verified on a fresh clone / clean image cache**
  `VERSION=v1.0.0 FRONTEND_PORT=18080 docker compose up --build --wait` → both services reported
  `Healthy`. `BASE_URL=http://localhost:18080 make smoke` → `8 checks passed against
  http://localhost:18080`. `make check` → green (see above). Ran in this worktree, which is a clean
  checkout of `platform/28-release` cut from fresh `origin/main`; the compose build used no pre-warmed
  layer cache for the changed args (`--build`).

- [x] **No secrets**
  `git log -p | grep -iE 'api[_-]?key|secret|token'` → only prose/code hits (rate-limit "token bucket",
  `GH_TOKEN: ${{ github.token }}` workflow expressions, "gitleaks secret scanning" documentation) — no
  literal credentials. gitleaks already runs in `.github/workflows/security.yml` (`gitleaks` job) on every
  PR and push.

- [x] **No `TODO`/`FIXME` in tracked files**
  `git grep -nE 'TODO|FIXME' -- . ':!*.md'` → the only hit is `docs/tickets.json` quoting this checklist's
  own ticket text ("no TODOs"), not a marker in code. Nothing to fix, nothing to file.

- [x] **No committed build output; `.env*` ignored**
  `git ls-files | grep -E '(^|/)(dist|coverage|bin)/'` → empty. `.gitignore` covers `.env` / `.env.*`
  (with `!.env.example` kept), `backend/bin/`, `backend/coverage.out`, `backend/coverage.html`,
  `node_modules/`, `frontend/dist/`, `frontend/coverage/`.

- [x] **Every merged PR has a PROMPTS.md section**
  `gh pr list --state merged` → 22 PRs. `docs/PROMPTS.md` has 20 session sections (PLANNING, P0-01…04,
  B1-01…06, F2-01…05, H3-01, H3-04, H3-05, D4-01) plus the reconciliation note under "One section per
  merged pull request": PR #51 carries two sessions (P0-02 + P0-03, since the P0-02 branch's original PR
  #49 was closed unmerged and refolded into #51); #58 and #70 are Dependabot (bot PR + its config
  follow-up), #66 is a one-line `CLAUDE.md` amendment, and #72 records ticket consolidation — none of
  these four was an implementation session, so they correctly have no section. No gaps found; this PR
  (#28) adds the 21st section (D4-04) below.

- [x] **All ADRs Accepted**
  `grep -irn "Status" docs/adr/*.md` → all 9 numbered ADRs (0001–0009) read `**Status:** Accepted`; only
  the template (0000) shows the placeholder `Proposed | Accepted | Superseded`.

- [x] **Open issues are only `follow-up`-labelled in the Backlog milestone**
  `gh issue list --state open` → 6 epics (#41–46, label `epic`, no milestone — by design, epics track
  sprints rather than living in a milestone) and this ticket, #28 (`Sprint 4` milestone, the one still
  being worked). Every remaining open issue (#24, #30–40, #50, #54, #60, #62, #65, #67, #73, #75, #79–81)
  carries `follow-up` and sits in milestone "Backlog — Post-submission follow-ups". No sprint issue other
  than #28 is open, so no label/milestone edits were needed.

- [x] **Repo: description, topics, About link, pinned epic; LICENSE present; commit author matches account**
  `gh repo edit` set description to "Full-stack calculator: Go REST microservice + React/TypeScript
  frontend, built to production standards (tests, CI, containers, ADRs)" and topics
  `go, react, typescript, calculator, rest-api, docker, openapi, prometheus` (verified via
  `gh api repos/.../topics`). No live demo exists, so no About link is set — nothing to point it at.
  GraphQL `pinnedIssues` already shows only #46 (the v1.0.0 epic) pinned; nothing else was pinned.
  `LICENSE` exists at repo root (MIT). `git log --format='%an <%ae>' | sort -u` → `Isa SUMER
  <sumerisa5@gmail.com>`, `İsa Sümer <81411169+isasumer@users.noreply.github.com>` (both the same person,
  the GitHub account `isasumer` per `gh api user`) and `dependabot[bot]` for its own automated commits —
  no mismatched author.

- [x] **v1.0.x images work with the `VERSION` build arg; `GET /version` and the frontend badge reflect it**
  `VERSION=v1.0.0 FRONTEND_PORT=18080 docker compose up --build --wait`, then
  `docker exec go-react-calculator-backend-1 /server -version` → `server version v1.0.0 (commit none,
  built unknown, go1.27.1)`. The backend's `/version` is an operational endpoint and is deliberately not
  proxied by nginx (only `/healthz` and `/api/` are — see `frontend/nginx/default.conf.template`), which
  is why it is checked with `-version` / from inside the network rather than through the published port;
  this matches the documented API surface (`docs/adr/0009`). The frontend bundle
  (`/assets/index-*.js` served from `http://localhost:18080`) contains `v1.0.0` — the build-time
  `VITE_APP_VERSION` arg reached `App.tsx`'s version badge. No plumbing changes were needed:
  `compose.yaml` already threads `VERSION` into both the backend `-ldflags` build (via
  `backend/Makefile`'s `docker-build` target and the compose `args:`) and the frontend's
  `VITE_APP_VERSION`/`VERSION`/`COMMIT`/`BUILD_DATE` build args, and this is already documented in
  README § Configuration → "Compose level".

- [x] **Recruiter email drafted; job-hunt tracker updated**
  Draft at `docs/SUBMISSION_EMAIL.md` (English, placeholders for recruiter name/company, no company name
  anywhere in the repo). The job-hunt tracker lives outside this repository and is updated by the
  orchestrator, not from here (see `CLAUDE.md` ground rules — this session touches only this repo).

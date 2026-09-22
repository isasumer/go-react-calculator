# ADR-0001: Monorepo with two independent build roots

- **Status:** Accepted
- **Date:** 2026-09-22
- **Ticket:** #1

## Context
The deliverable is one Git repository containing a Go service and a React application. Reviewers will clone once and expect to run everything from the root, but the two applications have different toolchains, dependency managers and release cadences. CI should not rebuild the frontend when only Go files change.

## Decision
One repository, two self-contained build roots: `backend/` (a Go module) and `frontend/` (an npm project). Each has its own `Makefile`, lint configuration, tests and Dockerfile and can be built without the other. A root `Makefile` provides umbrella targets (`check`, `test`, `lint`, `dev`, `up`, `down`, `e2e`) that delegate downwards. End-to-end tests live in `e2e/` as a third small npm project. Cross-cutting concerns (compose file, CI workflows, docs, ADRs) live at the root.

## Consequences
- Reviewers get one link and one `make check`; contributors of either app never need the other toolchain for unit work.
- CI workflows use path filters per app, keeping PR feedback fast.
- No shared code between the apps by design; the API contract (OpenAPI) is the only coupling. Type sharing via codegen is deliberately not done for a two-endpoint API (see ADR-0004).
- Go module path is `github.com/isasumer/go-react-calculator/backend`; tooling must be invoked from `backend/` or via the root Makefile.

## Alternatives considered
- Two repositories → two links to review, harder to keep the contract and docs in one place.
- Workspace tooling (Turborepo/Nx) spanning Go and Node → adds a layer the reviewer must learn; brings nothing for two projects.
- Go embedding the built frontend into one binary → couples releases and hides the reverse-proxy/static-hosting story that production would actually have (ADR-0009).

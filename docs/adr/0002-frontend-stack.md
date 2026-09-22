# ADR-0002: Vite + React 19 + TypeScript strict, Tailwind v4, shadcn primitives, TanStack Query, zustand

- **Status:** Accepted
- **Date:** 2026-09-22
- **Ticket:** #1

## Context
The assignment asks for a React frontend, TypeScript preferred, with an intuitive UI, validation, error handling and basic responsiveness. The app is one screen with no routing, no SSR and no SEO need. The conventions should mirror a production codebase the author maintains so the structure is habitual rather than invented for the exercise: flat `src/lib` helpers, a single typed `apiFetch` with a typed `ApiError`, TanStack Query for server state, one small zustand store per concern for UI state, hand-rolled versioned `localStorage` keys, kebab-case `components/ui` primitives, PascalCase feature components with a `data-ui` root attribute, co-located tests.

## Decision
- **Build:** Vite with the React + TypeScript template; `strict`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`; path alias `@/* → src/*`.
- **UI:** Tailwind v4 (CSS-first `@theme` tokens, `prefers-color-scheme` dark mode) with shadcn/radix primitives in `components/ui`.
- **Data:** TanStack Query wraps a single `apiFetch` in `src/lib/api.ts`; responses are validated with zod at the boundary.
- **State:** the calculator is a pure state machine in `src/lib/calculator-engine.ts`; a zustand store wires it to the API; history is a second store persisted through a versioned storage helper.
- **Tests:** Vitest + jsdom + Testing Library + user-event + MSW; Playwright for e2e (separate project).
- **Quality:** ESLint flat config (typescript-eslint type-checked, react, react-hooks, jsx-a11y) + Prettier.

## Consequences
- No SSR/routing framework to explain; the reviewer reads plain React.
- The API layer and stores are testable without React; components are tested by role/label, which doubles as the accessibility check.
- Two deliberate departures from the reference codebase: Vite instead of Next.js (no need for its features here) and component-level tests with MSW (the reference only unit-tests `lib/`). Both are improvements for this scope, not style drift.
- zod is used at the HTTP boundary here (the reference uses it for forms only) because the backend is a separate deployable and its responses are untrusted input.

## Alternatives considered
- Next.js → matches the reference stack but adds a server runtime and routing concepts for a single static screen.
- Create React App → deprecated.
- Redux Toolkit → more ceremony than a pure reducer plus one small store warrants.
- `zustand/persist` middleware → hides the storage schema and versioning; a 30-line helper keeps them explicit and testable.
- No UI primitives → fine for one screen, but shadcn buttons give correct focus/disabled semantics for free.

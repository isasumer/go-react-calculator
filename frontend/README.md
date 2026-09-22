# Frontend

Vite + React 19 + TypeScript (strict) single-page calculator UI. Stack and rationale: [ADR-0002](../docs/adr/0002-frontend-stack.md).

## Quick start

```bash
nvm use            # Node 22, from .nvmrc
npm ci
npm run dev        # http://localhost:5173
```

## Scripts

| npm script              | make target         | What it does                                         |
| ----------------------- | ------------------- | ---------------------------------------------------- |
| `npm run dev`           | `make dev`          | Vite dev server with HMR                             |
| `npm run build`         | `make build`        | `tsc -b` then production build to `dist/` (hashed)   |
| `npm run preview`       | —                   | Serve `dist/` locally                                |
| `npm run lint`          | `make lint`         | ESLint, zero warnings allowed                        |
| `npm run type-check`    | `make type-check`   | TypeScript project build without emit                |
| `npm run test`          | —                   | Vitest, single run                                   |
| `npm run test:coverage` | `make test`         | Vitest with v8 coverage and the threshold gate       |
| `npm run format`        | `make format`       | Prettier write                                       |
| `npm run format:check`  | `make format-check` | Prettier check                                       |
| —                       | `make check`        | format-check → lint → type-check → test (used by CI) |

## Configuration

`src/config.ts` is the only module that reads `import.meta.env`; call `getConfig()` everywhere else.

| Variable            | Default | Notes                                                             |
| ------------------- | ------- | ----------------------------------------------------------------- |
| `VITE_API_BASE_URL` | `/api`  | Absolute path or http(s) URL; trailing slash stripped; validated. |

## Layout

```
src/
├── main.tsx            # entry: mounts <App />, imports global styles
├── config.ts           # typed, validated env access
├── app/                # App.tsx (shell), globals.css (Tailwind v4 @theme tokens)
├── components/
│   ├── ui/             # shadcn primitives (new-york), kebab-case files
│   ├── providers/      # context providers (query client, error boundary)
│   ├── shared/         # cross-feature components
│   └── calculator/     # calculator feature components
├── hooks/              # use-*.ts
├── lib/                # flat helpers (utils.ts: cn())
├── stores/             # zustand stores, one per concern
├── types/              # shared types and zod schemas
└── test/               # setup.ts, msw/ (server + handlers)
```

## Conventions

- **Feature components** are PascalCase (`Calculator.tsx`, `HistoryPanel.tsx`) and live under
  `components/<feature>/` or `components/shared/`.
- **UI primitives** in `components/ui/` are kebab-case (`button.tsx`, `dropdown-menu.tsx`), as generated
  by shadcn. Treat them as vendored: restyle through tokens, not by forking.
- **Hooks** are kebab-case files named `use-*.ts` (`use-keyboard.ts`) exporting `useKeyboard`.
- **Helpers** in `lib/` are flat kebab-case modules (`format-number.ts`); no nested folders.
- **`data-ui` attribute**: every feature component's root element carries `data-ui="component-name"`
  (kebab-case of the component, e.g. `<div data-ui="history-panel">`). It is a stable hook for e2e tests
  and DOM inspection, independent of styling classes.
- **Tests are co-located**: `Foo.tsx` → `Foo.test.tsx`, `format-number.ts` → `format-number.test.ts`.
  Query by role/label with Testing Library; mock HTTP with MSW (`server.use(...)`), never `fetch` spies.
  Unhandled requests fail the test.
- **Imports** use the `@/` alias (`@/lib/utils`) rather than relative `../../` paths.
- **Styling** uses the tokens in `app/globals.css` (`bg-surface`, `text-danger`, `bg-key-operator`, …);
  dark mode follows `prefers-color-scheme` automatically, so avoid hard-coded colours.

## Testing

Four layers, from the bottom up:

1. **Engine unit tests** (`lib/calculator-engine.test.ts`, `lib/format-number.test.ts`, …) — pure
   functions in, pure data out, no DOM and no network.
2. **Hook tests** (`hooks/*.test.ts`) — `renderHook` from Testing Library, exercising a hook's
   contract in isolation from any component that happens to use it.
3. **Component tests** (`components/**/*.test.tsx`) — rendered with Testing Library and
   `renderWithProviders` (`src/test/utils.tsx`), querying by role/label rather than by class or
   test id. Anything that talks to the API is intercepted with MSW (`src/test/msw`); a request with
   no matching handler fails the test (`onUnhandledRequest: "error"` in `src/test/setup.ts`) rather
   than silently hitting the network or hanging.
4. **End-to-end** — Playwright against the composed stack, in [`e2e/`](../e2e) rather than in this
   suite: a separate npm project, so nothing here depends on Docker. Run it with `make e2e` from the
   repository root once the stack is up — see the root
   [README § Run the tests](../README.md#3-run-the-tests).

### Running

```bash
npm run test              # single run
npm run test -- --watch   # watch mode
npm run test:coverage     # single run with the v8 coverage gate (make -C frontend test)
```

`make -C frontend check` runs format-check → lint → type-check → test, the same sequence CI runs on
every PR.

### Adding an MSW handler override

The default handlers live in `src/test/msw/handlers.ts` and are installed for the whole suite by
`src/test/msw/server.ts`. A single test that needs different behaviour (an error response, a
delayed response, a specific payload) overrides just that request with `server.use(...)` — the
default handler for everything else stays in effect, and `server.resetHandlers()` (in
`src/test/setup.ts`'s `afterEach`) puts the defaults back before the next test:

```ts
import { http, HttpResponse } from "msw";

import { apiUrl } from "@/lib/api";
import { CALCULATE_ENDPOINT } from "@/lib/query-config";
import { server } from "@/test/msw/server";

server.use(
  http.post(apiUrl(CALCULATE_ENDPOINT), () =>
    HttpResponse.json({ code: "DIVISION_BY_ZERO" }, { status: 422 }),
  ),
);
```

### Coverage

`vitest.config.ts` sets v8 coverage thresholds — lines 85%, statements 85%, functions 85%, branches
80% — over `src/**/*.{ts,tsx}`, excluding tests, `src/test/**`, `src/main.tsx` and the vendored
`components/ui/**` primitives. `npm run test:coverage` fails the run if any threshold is missed;
CI runs the same command and additionally posts a per-file table to the job's step summary from
`coverage/coverage-summary.json` (the `json-summary` reporter), and uploads `coverage/lcov.info` as
a build artifact.

### Storage keys and versioning

Anything persisted to `localStorage` goes through `src/lib/storage.ts` (`storageKey`, `readJSON`,
`writeJSON`, `migrateKeys`), never `localStorage` directly. A key is `calc.<name>.v<version>`
(e.g. `calc.history.v1`); every read is validated against a zod schema, and anything missing,
unparsable, or schema-invalid is treated as absent rather than crashing the app. When a persisted
shape changes, bump `version` for that key and call `migrateKeys(prefix, currentKey)` once at
hydration so the old version's key is removed instead of lingering unread forever.

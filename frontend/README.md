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

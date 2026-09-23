/**
 * The app shell: a title, the build it is running, a link to the source, and the calculator.
 *
 * Providers are not here — `main.tsx` wraps this in `QueryProvider` and `ErrorBoundary`, so a test
 * can render `<App />` with only the providers it cares about.
 */
import { Calculator } from "@/components/calculator/Calculator";

export const REPOSITORY_URL = "https://github.com/isasumer/go-react-calculator";

/** The GitHub mark (Octicons `mark-github`, MIT), inlined so the CSP needs no external image source. */
function GitHubMark() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" focusable="false" className="size-5 fill-current">
      <path d="M8 0c4.42 0 8 3.58 8 8a8.013 8.013 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27-.68 0-1.36.09-2 .27-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8Z" />
    </svg>
  );
}

/**
 * The version badge, from `VITE_APP_VERSION` at build time. It is deliberately not the backend's
 * `/version` endpoint: this says which frontend you are looking at, and it must not depend on the
 * service being reachable. Unset (a dev server, a plain `vite build`) reads `dev`.
 *
 * It takes the env as an argument rather than reading `import.meta.env` inline so that it is a pure
 * function with a test of its own, and it treats the value as `unknown` because the ambient env type
 * types every unknown key as `any` — the same reason `config.ts` parses rather than trusts it.
 */
export function appVersion(env: Readonly<Record<string, unknown>>): string {
  const raw = env["VITE_APP_VERSION"];
  if (typeof raw !== "string" || raw.trim() === "") {
    return "dev";
  }
  return raw.trim();
}

export function App() {
  const version = appVersion(import.meta.env);

  return (
    <div
      data-ui="app-shell"
      className={
        // Safe-area insets keep the pad clear of a phone's rounded corners and home indicator.
        // The shell only centres itself once the viewport is tall enough to centre in; on a
        // landscape phone it starts at the top and scrolls, so the display is never cut off.
        "flex min-h-dvh flex-col items-center justify-start gap-3 bg-surface landscape-short:gap-1 " +
        "pt-[max(0.75rem,env(safe-area-inset-top))] pr-[max(0.75rem,env(safe-area-inset-right))] " +
        "pb-[max(0.75rem,env(safe-area-inset-bottom))] pl-[max(0.75rem,env(safe-area-inset-left))] " +
        "[@media(min-height:44rem)]:justify-center"
      }
    >
      <header className="flex w-full max-w-md items-center justify-between gap-3 px-1 sm:max-w-[47rem] lg:max-w-[53rem] landscape-short:max-w-2xl">
        <h1 className="text-xl font-semibold tracking-tight">Calculator</h1>
        <div className="flex items-center gap-2">
          <p className="rounded-full border border-border px-2 py-0.5 font-mono text-xs text-text-muted">
            <span className="sr-only">Version </span>
            {version}
          </p>
          <a
            href={REPOSITORY_URL}
            target="_blank"
            rel="noopener noreferrer"
            aria-label="Source code on GitHub (opens in a new tab)"
            title="Source code on GitHub"
            className="inline-flex size-9 items-center justify-center rounded-full text-text-muted hover:bg-surface-raised hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            <GitHubMark />
          </a>
        </div>
      </header>

      {/* From `sm:` `HistoryPanel` (F2-05, #16) sits beside the calculator card as a side column
          instead of stacking below it. The widths are card + gap + panel (28 + 1 + 18 rem, then
          32 + 1 + 20 rem), and the header uses the same ones so it lines up with the content. */}
      <main className="w-full max-w-md sm:max-w-[47rem] lg:max-w-[53rem] landscape-short:max-w-2xl">
        <Calculator />
      </main>
    </div>
  );
}

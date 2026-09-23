/**
 * The app shell: a title, the build it is running, and the calculator.
 *
 * Providers are not here — `main.tsx` wraps this in `QueryProvider` and `ErrorBoundary`, so a test
 * can render `<App />` with only the providers it cares about.
 */
import { Calculator } from "@/components/calculator/Calculator";

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
      <header className="flex w-full max-w-md items-baseline justify-between gap-3 px-1 sm:max-w-[47rem] lg:max-w-[53rem] landscape-short:max-w-2xl">
        <h1 className="text-xl font-semibold tracking-tight">Calculator</h1>
        <p className="rounded-full border border-border px-2 py-0.5 font-mono text-xs text-text-muted">
          <span className="sr-only">Version </span>
          {version}
        </p>
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

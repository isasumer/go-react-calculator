import { Button } from "@/components/ui/button";

/** App shell. The calculator feature replaces the placeholder content in Sprint 2 (F2-03). */
export function App() {
  return (
    <main data-ui="app-shell" className="grid min-h-dvh place-items-center p-6">
      <section
        aria-labelledby="app-title"
        className="w-full max-w-sm space-y-4 rounded-xl border bg-surface-raised p-6 shadow-sm"
      >
        <h1 id="app-title" className="text-2xl font-semibold tracking-tight">
          Calculator
        </h1>
        <p className="text-text-muted">
          Scaffold ready. Colours follow your operating system&apos;s light or dark setting.
        </p>
        <div className="grid grid-cols-3 gap-2" aria-hidden="true">
          <span className="rounded-md bg-key p-3 text-center text-key-foreground">7</span>
          <span className="rounded-md bg-key-function p-3 text-center text-key-function-foreground">
            AC
          </span>
          <span className="rounded-md bg-key-operator p-3 text-center text-key-operator-foreground">
            ÷
          </span>
        </div>
        <p className="text-sm text-danger">Errors render in the danger token.</p>
        <Button type="button" className="w-full" disabled>
          Coming in Sprint 2
        </Button>
      </section>
    </main>
  );
}

/**
 * Collapsible list of recent calculations (F2-05, #16), backed by `useHistoryStore`/`localStorage`.
 *
 * A native `<details>`/`<summary>` rather than a drawer library: it is keyboard- and
 * screen-reader-accessible for free, and its open state is one boolean, which is exactly what
 * `calc.history-open.v1` needs to remember across a reload.
 *
 * Layout: below the keypad on a phone (the calculator card and this panel just stack), and from
 * `sm:` (40rem) up it becomes a side column next to the card — see `Calculator.tsx`, which lays
 * the two out in a row at that breakpoint.
 */
import { useEffect, useRef, useState } from "react";
import { z } from "zod";

import { OPERATION_SYMBOLS } from "@/lib/calculator-engine";
import { formatResult } from "@/lib/format-number";
import { readJSON, storageKey, writeJSON } from "@/lib/storage";
import { cn } from "@/lib/utils";
import { type HistoryEntry, useHistoryStore } from "@/stores/useHistoryStore";

const OPEN_STATE_KEY = storageKey("history-open", 1);
const openStateSchema = z.boolean();

/** Open by default: routed through `storage.ts` so the no-window/throwing-localStorage/malformed
 *  cases are handled in the one place that already covers them, not duplicated here. */
function readOpenState(): boolean {
  return readJSON(OPEN_STATE_KEY, openStateSchema) ?? true;
}

function writeOpenState(open: boolean): void {
  writeJSON(OPEN_STATE_KEY, open);
}

/** `a op b = result`, or `op a = result` for a unary entry (`b` is `null`). */
function rowText(entry: HistoryEntry): string {
  const symbol = OPERATION_SYMBOLS[entry.operation];
  const a = formatResult(entry.a);
  const result = formatResult(entry.result);
  if (entry.b === null) {
    return `${symbol}${a} = ${result}`;
  }
  return `${a} ${symbol} ${formatResult(entry.b)} = ${result}`;
}

/** The raw (unformatted) values, for the row button's `title`. */
function rowTitle(entry: HistoryEntry): string {
  if (entry.b === null) {
    return `${entry.operation}(${entry.a}) = ${entry.result}`;
  }
  return `${entry.a} ${entry.operation} ${entry.b} = ${entry.result}`;
}

export interface HistoryPanelProps {
  /** Recalls an entry's result into the display, as if it had just been typed as `a`. */
  readonly onRecall: (value: number) => void;
}

export function HistoryPanel({ onRecall }: HistoryPanelProps) {
  const entries = useHistoryStore((state) => state.entries);
  const remove = useHistoryStore((state) => state.remove);
  const clear = useHistoryStore((state) => state.clear);

  const [open, setOpen] = useState(readOpenState);

  return (
    <details
      data-ui="calculator.history"
      open={open}
      onToggle={(event) => {
        const next = event.currentTarget.open;
        setOpen(next);
        writeOpenState(next);
      }}
      className="w-full max-w-sm rounded-2xl border border-border bg-surface-raised p-3 shadow-sm sm:w-64 sm:max-w-none"
    >
      <summary className="cursor-pointer select-none text-sm font-semibold text-text">
        History ({entries.length})
      </summary>

      <div className="mt-2 flex flex-col gap-2">
        {entries.length === 0 ? (
          <p className="py-1 text-sm text-text-muted">No calculations yet.</p>
        ) : (
          <ul className="flex max-h-72 flex-col gap-1 overflow-y-auto">
            {entries.map((entry) => (
              <li key={entry.id} className="flex items-center gap-1">
                <button
                  type="button"
                  title={rowTitle(entry)}
                  onClick={() => {
                    onRecall(entry.result);
                  }}
                  className="flex-1 truncate rounded-lg px-2 py-1 text-left text-sm hover:bg-surface"
                >
                  {rowText(entry)}
                </button>
                <button
                  type="button"
                  aria-label={`Remove ${rowText(entry)} from history`}
                  onClick={() => {
                    remove(entry.id);
                  }}
                  className="shrink-0 rounded-lg px-2 py-1 text-xs text-text-muted hover:bg-danger hover:text-danger-foreground"
                >
                  ✕
                </button>
              </li>
            ))}
          </ul>
        )}

        {entries.length > 0 && <ClearHistoryButton onClear={clear} />}
      </div>
    </details>
  );
}

interface ClearHistoryButtonProps {
  readonly onClear: () => void;
}

/**
 * A two-step confirm instead of `window.confirm`: the first click turns the button itself into
 * "Confirm clear", the second click (within the same focus) actually clears. Losing focus or
 * pressing Escape cancels back to the first state without clearing anything.
 */
function ClearHistoryButton({ onClear }: ClearHistoryButtonProps) {
  const [confirming, setConfirming] = useState(false);
  const resetTimeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => () => clearTimeout(resetTimeout.current), []);

  function cancel(): void {
    clearTimeout(resetTimeout.current);
    setConfirming(false);
  }

  function handleClick(): void {
    if (confirming) {
      cancel();
      onClear();
      return;
    }
    setConfirming(true);
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      onBlur={cancel}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          cancel();
        }
      }}
      className={cn(
        "self-start rounded-lg px-2 py-1 text-xs font-medium",
        confirming
          ? "bg-danger text-danger-foreground"
          : "text-text-muted hover:bg-surface hover:text-danger",
      )}
    >
      {confirming ? "Confirm clear" : "Clear history"}
    </button>
  );
}

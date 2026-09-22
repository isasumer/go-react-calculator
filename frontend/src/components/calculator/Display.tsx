/**
 * The calculator's screen: what has been entered so far, the number on show, and the error slot.
 *
 * Three lines, always in the same place, so nothing below moves when an error appears or clears:
 * the expression (`12 +`), the entry or result, and a reserved line for the failure sentence.
 * The result line is a live region so a screen reader hears the answer without hunting for it;
 * the error line is an `alert`, because a failure should interrupt.
 *
 * Numbers can be up to {@link MAX_ENTRY_DIGITS} digits long, so the entry shrinks instead of
 * wrapping or overflowing, and the untruncated value is always in `title`.
 */
import { cn } from "@/lib/utils";

export interface DisplayProps {
  /** The secondary line, from `selectExpression`. Empty renders as a blank reserved line. */
  readonly expression: string;
  /** The main line, from `selectDisplay`. Raw for now — formatting is F2-06 (#17). */
  readonly value: string;
  /** The mapped sentence from `selectError`, or `null` when the last calculation was fine. */
  readonly error: string | null;
  /** True while a calculation is in flight (`selectIsBusy`). */
  readonly busy: boolean;
}

/**
 * Font size as a function of length. A `clamp()` would size the text against the *viewport*, which
 * is the wrong variable: the display is a fixed 24 rem column, so what matters is how many
 * characters have to fit in it. Four steps cover 1 to 18 characters (16 digits plus sign and point).
 */
export function entrySizeClass(value: string): string {
  const { length } = value;
  if (length > 15) {
    return "text-xl";
  }
  if (length > 11) {
    return "text-2xl";
  }
  if (length > 8) {
    return "text-3xl";
  }
  return "text-4xl";
}

export function Display({ expression, value, error, busy }: DisplayProps) {
  return (
    <div
      data-ui="calculator.display"
      className="rounded-xl border border-border bg-surface-raised px-4 py-3"
    >
      <p
        title={expression}
        className="min-h-5 truncate text-right font-mono text-sm text-text-muted tabular-nums"
      >
        {expression}
      </p>

      <div className="flex items-baseline justify-end gap-2">
        {busy ? (
          // Decorative: the answer itself is announced by the live region below, and a second
          // announcement for "working on it" would talk over it.
          <span
            aria-hidden="true"
            className="font-mono text-lg text-text-muted motion-safe:animate-pulse"
          >
            …
          </span>
        ) : null}
        {/* A busy live region stays quiet: the value on screen is about to be replaced, and
            announcing it now would talk over the answer. */}
        <output
          aria-live="polite"
          aria-busy={busy}
          title={value}
          className={cn(
            "block min-w-0 truncate text-right font-mono font-semibold tabular-nums",
            entrySizeClass(value),
          )}
        >
          {value}
        </output>
      </div>

      {/* Always rendered: a live region has to exist before its text changes to be announced. */}
      <p role="alert" className="min-h-5 text-right text-sm font-medium text-danger">
        {error ?? ""}
      </p>
    </div>
  );
}

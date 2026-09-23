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
 *
 * Formatting (ADR-0007, absorbed #17): the raw string from `selectDisplay` is never shown as-is.
 * While a value is still being typed it gets thousands grouping (`formatEntry`); once it is a
 * finished operand or result it gets full display formatting (`formatResult`) — trimmed float
 * noise, exponent notation past the thresholds. Either way `title` keeps the untouched raw value.
 */
import { formatEntry, formatResult, parseEntry } from "@/lib/format-number";
import { cn } from "@/lib/utils";

export interface DisplayProps {
  /** The secondary line, from `selectExpression`. Empty renders as a blank reserved line. */
  readonly expression: string;
  /** The main line, from `selectDisplay`. Formatted for display; `title` keeps the raw value. */
  readonly value: string;
  /** The mapped sentence from `selectError`, or `null` when the last calculation was fine. */
  readonly error: string | null;
  /** True while a calculation is in flight (`selectIsBusy`). */
  readonly busy: boolean;
  /** True while `value` is still being typed (`enteringA`/`enteringB`) — switches `value`'s
   *  formatting from `formatResult` to `formatEntry`. */
  readonly entering?: boolean | undefined;
  /** The failing request's server request id (`selectErrorRequestId`), shown as the alert's
   *  `title` for support. `null`/omitted when there is no error or it carried none. */
  readonly errorRequestId?: string | null | undefined;
  /** Whether the alert should offer a Retry button (`selectCanRetry`). */
  readonly canRetry?: boolean | undefined;
  /** Re-issues the request that failed. Rendered only when `canRetry` is also true. */
  readonly onRetry?: (() => void) | undefined;
  /** True for a brief moment after a rejected keystroke (a second ".", the digit cap, a digit
   *  while busy); drives a CSS shake off `data-shake`, skipped under `prefers-reduced-motion`. */
  readonly shake?: boolean | undefined;
}

/** `value` is always a plain numeric string (the engine's `String(value)`), so this never throws. */
function formatMainValue(value: string, entering: boolean): string {
  return entering ? formatEntry(value) : formatResult(parseEntry(value));
}

/**
 * The expression line is a complete sentence built by `expressionOf` (`"12 + 7 ="`); only its
 * leading operand, `a`, is reformatted here — re-deriving the whole sentence would mean duplicating
 * the engine's assembly logic, which is not this ticket's job.
 */
function formatExpression(expression: string): string {
  if (expression === "") {
    return expression;
  }
  const [first, ...rest] = expression.split(" ");
  const a = parseEntry(first ?? "");
  if (!Number.isFinite(a)) {
    return expression;
  }
  return [formatResult(a), ...rest].join(" ");
}

/**
 * Font size as a function of length. A `clamp()` would size the text against the *viewport*, which
 * is the wrong variable: the display is a column of at most 32 rem, so what matters is how many
 * characters have to fit in it. Four steps cover 1 to 18 characters (16 digits plus sign and point);
 * each still fits a 320 px phone, where the display is about 230 px wide. The 9–11 step grows only
 * from `lg:`, because a phone on its side is already past `sm:`; there the display shares the card
 * with the keys, so the two short steps drop back down.
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
    return "text-3xl lg:text-4xl landscape-short:text-2xl";
  }
  return "text-5xl landscape-short:text-2xl";
}

export function Display({
  expression,
  value,
  error,
  busy,
  entering = false,
  errorRequestId = null,
  canRetry = false,
  onRetry,
  shake = false,
}: DisplayProps) {
  const displayValue = formatMainValue(value, entering);
  const displayExpression = formatExpression(expression);

  return (
    <div
      data-ui="calculator.display"
      data-shake={shake ? "true" : undefined}
      className="rounded-xl border border-border bg-surface-raised px-4 py-3 tall:py-5"
    >
      <p
        title={expression}
        className="min-h-6 truncate text-right font-mono text-base text-text-muted tabular-nums"
      >
        {displayExpression}
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
            entrySizeClass(displayValue),
          )}
        >
          {displayValue}
        </output>
      </div>

      {/* Always rendered: a live region has to exist before its text changes to be announced. */}
      <p
        role="alert"
        title={errorRequestId ?? undefined}
        className="min-h-5 text-right text-sm font-medium text-danger"
      >
        {error ?? ""}
        {canRetry && onRetry ? (
          <button
            type="button"
            onClick={onRetry}
            className="ml-2 underline decoration-dotted underline-offset-2 hover:no-underline"
          >
            Retry
          </button>
        ) : null}
      </p>
    </div>
  );
}

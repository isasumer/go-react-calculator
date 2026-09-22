/**
 * One calculator key: a `ui/button` with a hit area, a role-based look and the labels a screen
 * reader needs.
 *
 * The visible label is a symbol (`÷`, `⌫`), which is why `ariaLabel` is required rather than
 * optional — every key says what it does in words. `keyShortcut` is informational for now: it
 * renders `aria-keyshortcuts` so assistive technology can announce the key, and F2-04 (#15) adds
 * the handler that makes the keyboard actually work.
 */
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type KeyVariant = "digit" | "operator" | "action" | "equals";

/**
 * Colour is never the only signal: operators are also heavier and larger than digits, and `=` is
 * the only round key on the pad. That keeps the accent readable for someone who cannot see it.
 */
const VARIANT_CLASSES: Readonly<Record<KeyVariant, string>> = {
  digit: "bg-key text-key-foreground font-medium hover:bg-key/70",
  action:
    "bg-key-function text-key-function-foreground text-base font-semibold hover:bg-key-function/70",
  operator:
    "bg-key-operator text-key-operator-foreground text-2xl font-bold hover:bg-key-operator/85",
  equals:
    "bg-key-operator text-key-operator-foreground rounded-full text-2xl font-extrabold ring-2 ring-key-operator/40 hover:bg-key-operator/85",
};

export interface KeyProps {
  /** What the key shows: a digit, a symbol or a short word. */
  readonly label: ReactNode;
  /** What the key does, in words — `"multiply"`, `"clear all"`. Required: symbols do not read. */
  readonly ariaLabel: string;
  /**
   * What the key does. Omitted only for a key this build cannot perform — an operation the backend
   * advertises and we have no action for — which is then also `disabled`: a live key that does
   * nothing would be worse than no key at all.
   */
  readonly onPress?: (() => void) | undefined;
  readonly variant?: KeyVariant | undefined;
  /** Keyboard equivalent in `aria-keyshortcuts` syntax, e.g. `"Escape"`, `"+"`. */
  readonly keyShortcut?: string | undefined;
  readonly disabled?: boolean | undefined;
  readonly title?: string | undefined;
  readonly className?: string | undefined;
}

export function Key({
  label,
  ariaLabel,
  onPress,
  variant = "digit",
  keyShortcut,
  disabled = false,
  title,
  className,
}: KeyProps) {
  return (
    <Button
      type="button"
      variant="ghost"
      data-ui="calculator.key"
      data-key-variant={variant}
      aria-label={ariaLabel}
      aria-keyshortcuts={keyShortcut}
      title={title}
      disabled={disabled}
      onClick={onPress}
      className={cn(
        // 44 px is the smallest comfortable touch target; the keys grow with the grid above that
        // and shrink no further, which is what keeps the pad usable in landscape.
        "h-full min-h-11 w-full min-w-11 rounded-xl px-0 text-xl tabular-nums select-none",
        VARIANT_CLASSES[variant],
        className,
      )}
    >
      {label}
    </Button>
  );
}

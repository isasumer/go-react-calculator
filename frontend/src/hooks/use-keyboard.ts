/**
 * Window-level keydown handling for the calculator (F2-04, #15).
 *
 * One listener for the life of the component, not one per render: handlers are read through a ref
 * so the effect that attaches the listener never has to re-run, which is also what keeps a fresh
 * `busy` value visible to a listener that was attached long before the current render.
 *
 * {@link KEY_MAP} is the whole policy, kept as plain data so it can be asserted on directly instead
 * of only through simulated keypresses.
 */
import { useEffect, useRef } from "react";

import type { BinaryOperation, Digit, UnaryKey } from "@/lib/calculator-engine";

export interface KeyboardHandlers {
  readonly onDigit: (digit: Digit) => void;
  readonly onDecimal: () => void;
  readonly onOperator: (operator: BinaryOperation) => void;
  readonly onUnary: (op: UnaryKey) => void;
  readonly onEquals: () => void;
  readonly onClearAll: () => void;
  readonly onClearEntry: () => void;
  readonly onBackspace: () => void;
  /** True while a calculation is in flight: every key but Escape is ignored (ADR-0008). */
  readonly busy: boolean;
  /** Called with the raw `event.key` for every key the hook actually acted on — the pressed-state
   *  visual feedback hangs off this rather than off its own copy of the mapping. */
  readonly onHandled?: ((key: string) => void) | undefined;
}

type KeyAction =
  | { readonly type: "digit"; readonly digit: Digit }
  | { readonly type: "decimal" }
  | { readonly type: "operator"; readonly operator: BinaryOperation }
  | { readonly type: "unary"; readonly op: UnaryKey }
  | { readonly type: "equals" }
  | { readonly type: "clearAll" }
  | { readonly type: "clearEntry" }
  | { readonly type: "backspace" };

/**
 * `event.key` → what it does. Digits and the four arithmetic symbols read the way a physical
 * keypad's do; `^` and `%` are the operations bar's `xʸ` and `%` keys; `r` is documented in the
 * issue as the mnemonic for `√` (there is no ASCII symbol for it). `Enter` and `=` both evaluate —
 * `=` because it is the character on the key, `Enter` because it is the one everybody actually
 * presses.
 */
export const KEY_MAP: Readonly<Record<string, KeyAction>> = {
  "0": { type: "digit", digit: "0" },
  "1": { type: "digit", digit: "1" },
  "2": { type: "digit", digit: "2" },
  "3": { type: "digit", digit: "3" },
  "4": { type: "digit", digit: "4" },
  "5": { type: "digit", digit: "5" },
  "6": { type: "digit", digit: "6" },
  "7": { type: "digit", digit: "7" },
  "8": { type: "digit", digit: "8" },
  "9": { type: "digit", digit: "9" },
  ".": { type: "decimal" },
  "+": { type: "operator", operator: "add" },
  "-": { type: "operator", operator: "subtract" },
  "*": { type: "operator", operator: "multiply" },
  "/": { type: "operator", operator: "divide" },
  "^": { type: "operator", operator: "power" },
  "%": { type: "unary", op: "percent" },
  r: { type: "unary", op: "sqrt" },
  Enter: { type: "equals" },
  "=": { type: "equals" },
  Escape: { type: "clearAll" },
  Backspace: { type: "backspace" },
  Delete: { type: "clearEntry" },
};

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") {
    return true;
  }
  // `isContentEditable` is unimplemented in jsdom (always `false`); the attribute check is the
  // part the test environment can actually see, and it is correct in a real browser too.
  return target.isContentEditable || target.getAttribute("contenteditable") === "true";
}

function dispatch(action: KeyAction, handlers: KeyboardHandlers): void {
  switch (action.type) {
    case "digit":
      handlers.onDigit(action.digit);
      return;
    case "decimal":
      handlers.onDecimal();
      return;
    case "operator":
      handlers.onOperator(action.operator);
      return;
    case "unary":
      handlers.onUnary(action.op);
      return;
    case "equals":
      handlers.onEquals();
      return;
    case "clearAll":
      handlers.onClearAll();
      return;
    case "clearEntry":
      handlers.onClearEntry();
      return;
    case "backspace":
      handlers.onBackspace();
  }
}

/**
 * Attaches one `window` "keydown" listener for the lifetime of the component and routes it through
 * {@link KEY_MAP}. Modifier combinations (ctrl/meta/alt — shift is left alone so `+` and `%` still
 * work on layouts that need it) and typing into an unrelated field are left completely alone: no
 * dispatch, no `preventDefault`. While `busy` is true every key except Escape is likewise left
 * alone, because the store would ignore it anyway and a swallowed `Ctrl+R`-adjacent keystroke for
 * nothing is worse than a key that briefly does nothing.
 */
export function useKeyboard(handlers: KeyboardHandlers): void {
  const handlersRef = useRef(handlers);
  useEffect(() => {
    handlersRef.current = handlers;
  });

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent): void {
      if (event.ctrlKey || event.metaKey || event.altKey) {
        return;
      }
      if (isTypingTarget(event.target)) {
        return;
      }
      const action = KEY_MAP[event.key];
      if (action === undefined) {
        return;
      }
      const current = handlersRef.current;
      if (current.busy && action.type !== "clearAll") {
        return;
      }
      event.preventDefault();
      dispatch(action, current);
      current.onHandled?.(event.key);
    }

    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
    };
  }, []);
}

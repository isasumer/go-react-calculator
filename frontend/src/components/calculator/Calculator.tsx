/**
 * The calculator, assembled.
 *
 * This is the only component that knows both halves of the app: it reads the store's state through
 * its selectors and hands the store's evaluator to React Query. Everything below it is given plain
 * props and callbacks, which is what makes `Display`, `Keypad` and `Key` renderable — and
 * testable — without a provider or a store in sight.
 *
 * The wiring is a single effect: `useCalculate().mutateAsync` is the function the store calls when
 * the engine asks for a calculation (ADR-0008). It is stable for the life of the mutation, so the
 * effect runs once, and its cleanup unwires the store so an unmounted tree cannot be called back.
 *
 * F2-04 (#15) adds two more things this component owns because nowhere else has both the store and
 * the DOM: the keyboard listener (`useKeyboard`), and the transient visual feedback — a pressed key
 * and a shaking display — that no piece of state should have to remember past its own timeout.
 */
import { useCallback, useEffect, useRef, useState } from "react";

import { useCalculate } from "@/hooks/use-calculate";
import { useKeyboard } from "@/hooks/use-keyboard";
import { MAX_ENTRY_DIGITS, type Digit } from "@/lib/calculator-engine";
import {
  selectCanRetry,
  selectDisplay,
  selectError,
  selectErrorRequestId,
  selectExpression,
  selectIsBusy,
  selectPhase,
  useCalculatorStore,
} from "@/stores/useCalculatorStore";

import { Display } from "./Display";
import { Keypad } from "./Keypad";
import { OperationsBar } from "./OperationsBar";

/** How long a key's pressed state (keyboard) or the display's shake (invalid input) stays on. */
const PRESSED_MS = 120;
const SHAKE_MS = 300;

export function Calculator() {
  const { mutateAsync } = useCalculate();
  const setEvaluator = useCalculatorStore((state) => state.setEvaluator);

  useEffect(() => {
    setEvaluator(mutateAsync);
    return () => {
      setEvaluator(null);
    };
  }, [setEvaluator, mutateAsync]);

  const expression = useCalculatorStore(selectExpression);
  const value = useCalculatorStore(selectDisplay);
  const error = useCalculatorStore(selectError);
  const errorRequestId = useCalculatorStore(selectErrorRequestId);
  const canRetry = useCalculatorStore(selectCanRetry);
  const busy = useCalculatorStore(selectIsBusy);
  const phase = useCalculatorStore(selectPhase);
  const entering = phase === "enteringA" || phase === "enteringB";

  // Actions are created once by zustand, so each of these is a stable reference and can be passed
  // straight down as the callback prop.
  const inputDigit = useCalculatorStore((state) => state.inputDigit);
  const inputDecimal = useCalculatorStore((state) => state.inputDecimal);
  const toggleSign = useCalculatorStore((state) => state.toggleSign);
  const backspace = useCalculatorStore((state) => state.backspace);
  const clearEntry = useCalculatorStore((state) => state.clearEntry);
  const clearAll = useCalculatorStore((state) => state.clearAll);
  const setOperator = useCalculatorStore((state) => state.setOperator);
  const evaluate = useCalculatorStore((state) => state.evaluate);
  const applyUnary = useCalculatorStore((state) => state.applyUnary);
  const retry = useCalculatorStore((state) => state.retry);

  // --- transient visual feedback: pressed key (keyboard) and shake (rejected keystroke) --------

  const [pressedShortcut, setPressedShortcut] = useState<string | null>(null);
  const pressedTimeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [shake, setShake] = useState(false);
  const shakeTimeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(
    () => () => {
      clearTimeout(pressedTimeout.current);
      clearTimeout(shakeTimeout.current);
    },
    [],
  );

  const flashPressed = useCallback((key: string) => {
    setPressedShortcut(key);
    clearTimeout(pressedTimeout.current);
    pressedTimeout.current = setTimeout(() => {
      setPressedShortcut(null);
    }, PRESSED_MS);
  }, []);

  const triggerShake = useCallback(() => {
    setShake(true);
    clearTimeout(shakeTimeout.current);
    shakeTimeout.current = setTimeout(() => {
      setShake(false);
    }, SHAKE_MS);
  }, []);

  /**
   * Digits are rejected silently by the store (busy, or the entry is already at
   * {@link MAX_ENTRY_DIGITS}); the shake is how the UI admits that the key press did nothing.
   */
  const guardedInputDigit = useCallback(
    (digit: Digit) => {
      const { calc, pending } = useCalculatorStore.getState();
      const atCap =
        (calc.phase === "enteringA" || calc.phase === "enteringB") &&
        calc.display.replace(/[^0-9]/g, "").length >= MAX_ENTRY_DIGITS;
      if (pending || atCap) {
        triggerShake();
      }
      inputDigit(digit);
    },
    [inputDigit, triggerShake],
  );

  /** A second "." on the current entry is also a silent no-op in the engine. */
  const guardedInputDecimal = useCallback(() => {
    const { calc, pending } = useCalculatorStore.getState();
    const duplicate =
      (calc.phase === "enteringA" || calc.phase === "enteringB") && calc.display.includes(".");
    if (pending || duplicate) {
      triggerShake();
    }
    inputDecimal();
  }, [inputDecimal, triggerShake]);

  useKeyboard({
    busy,
    onDigit: guardedInputDigit,
    onDecimal: guardedInputDecimal,
    onOperator: setOperator,
    onUnary: applyUnary,
    onEquals: evaluate,
    onClearAll: clearAll,
    onClearEntry: clearEntry,
    onBackspace: backspace,
    onHandled: flashPressed,
  });

  return (
    <section
      data-ui="calculator"
      aria-label="Calculator"
      className={
        // Portrait: one column, display on top. On a phone held sideways there is no room for a
        // display *and* five rows of keys, so the two sit next to each other instead and the
        // display stays on screen rather than scrolling away above the pad.
        "flex w-full max-w-sm flex-col gap-3 rounded-2xl border border-border " +
        "bg-surface-raised p-3 shadow-sm " +
        "landscape-short:grid landscape-short:max-w-2xl landscape-short:grid-cols-2 " +
        "landscape-short:items-start"
      }
    >
      <div className="flex flex-col gap-3">
        <Display
          expression={expression}
          value={value}
          error={error}
          busy={busy}
          entering={entering}
          errorRequestId={errorRequestId}
          canRetry={canRetry}
          onRetry={retry}
          shake={shake}
        />
        <OperationsBar
          busy={busy}
          onUnary={applyUnary}
          onOperator={setOperator}
          pressedShortcut={pressedShortcut}
        />
      </div>
      <Keypad
        busy={busy}
        onDigit={guardedInputDigit}
        onDecimal={guardedInputDecimal}
        onToggleSign={toggleSign}
        onBackspace={backspace}
        onClearEntry={clearEntry}
        onClearAll={clearAll}
        onOperator={setOperator}
        onEquals={evaluate}
        pressedShortcut={pressedShortcut}
      />
    </section>
  );
}

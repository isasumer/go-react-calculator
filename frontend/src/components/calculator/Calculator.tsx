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
 */
import { useEffect } from "react";

import { useCalculate } from "@/hooks/use-calculate";
import {
  selectDisplay,
  selectError,
  selectExpression,
  selectIsBusy,
  useCalculatorStore,
} from "@/stores/useCalculatorStore";

import { Display } from "./Display";
import { Keypad } from "./Keypad";
import { OperationsBar } from "./OperationsBar";

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
  const busy = useCalculatorStore(selectIsBusy);

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
        <Display expression={expression} value={value} error={error} busy={busy} />
        <OperationsBar busy={busy} onUnary={applyUnary} onOperator={setOperator} />
      </div>
      <Keypad
        busy={busy}
        onDigit={inputDigit}
        onDecimal={inputDecimal}
        onToggleSign={toggleSign}
        onBackspace={backspace}
        onClearEntry={clearEntry}
        onClearAll={clearAll}
        onOperator={setOperator}
        onEquals={evaluate}
      />
    </section>
  );
}

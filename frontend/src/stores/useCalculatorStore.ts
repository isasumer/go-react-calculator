/**
 * The calculator store: the engine plus the one thing the engine cannot have — a request in flight.
 *
 * Every action is `state = engine.transition(state, input)`, and when a transition asks for a
 * calculation the store performs it through an injected `evaluator`. The evaluator is injected
 * rather than imported so the store can be driven in a plain test with a fake promise; the
 * component layer wires the real one (`useCalculate`) in with {@link CalculatorStore.setEvaluator}.
 *
 * While a request is in flight every input except `AC` is ignored, and an answer that belongs to a
 * calculation the user has since cleared is thrown away instead of overwriting the display.
 */
import { create } from "zustand";

import * as engine from "@/lib/calculator-engine";
import { FALLBACK_ERROR_MESSAGE, messageForError } from "@/lib/error-messages";
import type { CalculateRequest, CalculateResponse } from "@/types/calculator";

/** Performs one calculation. Rejections are expected to be `ApiError`s; anything else is mapped too. */
export type Evaluator = (request: CalculateRequest) => Promise<CalculateResponse>;

export interface CalculatorStore {
  /** The engine's state, replaced as a whole so a transition can never be applied by halves. */
  readonly calc: engine.CalcState;
  /** True from the moment a calculation is sent until its answer (or failure) is applied. */
  readonly pending: boolean;

  /** Inject the function that performs a calculation. `null` unwires it (component unmount). */
  setEvaluator: (evaluator: Evaluator | null) => void;

  inputDigit: (digit: engine.Digit) => void;
  inputDecimal: () => void;
  toggleSign: () => void;
  backspace: () => void;
  clearEntry: () => void;
  /** `AC`. The only action that works while a calculation is pending. */
  clearAll: () => void;
  setOperator: (operator: engine.BinaryOperation) => void;
  /** `=` / Enter. */
  evaluate: () => void;
  applyUnary: (op: engine.UnaryKey) => void;
}

export const useCalculatorStore = create<CalculatorStore>()((set, get) => {
  let evaluator: Evaluator | null = null;
  /**
   * Identity of the request the store is waiting for. Every new request and every `AC` bumps it, so
   * an answer whose ticket is no longer current belongs to a calculation nobody is waiting for.
   */
  let ticket = 0;

  function settle(sent: number, resolve: (calc: engine.CalcState) => engine.Transition): void {
    if (sent !== ticket) {
      return;
    }
    set({ calc: resolve(get().calc).state, pending: false });
  }

  function apply({ state, effect }: engine.Transition): void {
    if (effect === undefined) {
      set({ calc: state });
      return;
    }
    const sent = (ticket += 1);
    set({ calc: state, pending: true });

    const evaluate = evaluator;
    if (evaluate === null) {
      // Nothing is wired up to answer (the store is used outside the app). Fail the press rather
      // than leave the UI pending for ever.
      settle(sent, (calc) => engine.applyError(calc, FALLBACK_ERROR_MESSAGE));
      return;
    }
    void evaluate(effect.request).then(
      (response) => {
        settle(sent, (calc) => engine.applyResult(calc, response));
      },
      (error: unknown) => {
        settle(sent, (calc) => engine.applyError(calc, messageForError(error)));
      },
    );
  }

  /** Keys are inert while the server is answering; the result would land on top of them. */
  function input(transition: (calc: engine.CalcState) => engine.Transition): void {
    if (get().pending) {
      return;
    }
    apply(transition(get().calc));
  }

  return {
    calc: engine.initialState,
    pending: false,

    setEvaluator: (next) => {
      evaluator = next;
    },

    inputDigit: (digit) => {
      input((calc) => engine.inputDigit(calc, digit));
    },
    inputDecimal: () => {
      input(engine.inputDecimal);
    },
    toggleSign: () => {
      input(engine.toggleSign);
    },
    backspace: () => {
      input(engine.backspace);
    },
    clearEntry: () => {
      input(engine.clearEntry);
    },
    clearAll: () => {
      // Never guarded by `pending`: AC is how the user gets out of a slow request, and bumping the
      // ticket is what makes that request's answer stale.
      ticket += 1;
      set({ calc: engine.clearAll(get().calc).state, pending: false });
    },
    setOperator: (operator) => {
      input((calc) => engine.setOperator(calc, operator));
    },
    evaluate: () => {
      input(engine.requestEvaluate);
    },
    applyUnary: (op) => {
      input((calc) => engine.applyUnary(calc, op));
    },
  };
});

// --- selectors --------------------------------------------------------------------------------

/** The main display line. */
export const selectDisplay = (state: CalculatorStore): string => state.calc.display;

export const selectPhase = (state: CalculatorStore): engine.Phase => state.calc.phase;

/** True while a calculation is in flight: keys are inert and the display should say so. */
export const selectIsBusy = (state: CalculatorStore): boolean => state.pending;

/** The sentence to show, or `null` when the last calculation succeeded. */
export const selectError = (state: CalculatorStore): string | null => state.calc.error;

/** The secondary display line, e.g. `12 +` or `12 + 7 =`. */
export const selectExpression = (state: CalculatorStore): string => engine.expressionOf(state.calc);

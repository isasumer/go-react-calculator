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

import { isApiError } from "@/lib/api";
import * as engine from "@/lib/calculator-engine";
import { messageForError } from "@/lib/error-messages";
import type { CalculateRequest, CalculateResponse } from "@/types/calculator";

/** Performs one calculation. Rejections are expected to be `ApiError`s; anything else is mapped too. */
export type Evaluator = (request: CalculateRequest) => Promise<CalculateResponse>;

/**
 * `ApiError` codes {@link messageForError} maps to a transport-level sentence rather than a
 * semantic one: the request never reached (or never came back from) the server, so sending the same
 * body again is the right recovery, not a data-entry mistake to correct first.
 */
const RETRYABLE_ERROR_CODES: ReadonlySet<string> = new Set(["NETWORK", "TIMEOUT", "RATE_LIMITED"]);

export interface CalculatorStore {
  /** The engine's state, replaced as a whole so a transition can never be applied by halves. */
  readonly calc: engine.CalcState;
  /** True from the moment a calculation is sent until its answer (or failure) is applied. */
  readonly pending: boolean;
  /** The failed request's server request id, for support — `null` when there is no current error
   *  or the error carried none (a plain `TypeError`, for instance, never does). */
  readonly errorRequestId: string | null;
  /** True when the current error is transport-level and the same request can just be resent. */
  readonly canRetry: boolean;

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
  /** Re-sends the request that produced the current error. A no-op when `canRetry` is false or a
   *  request is already in flight. */
  retry: () => void;
}

export const useCalculatorStore = create<CalculatorStore>()((set, get) => {
  let evaluator: Evaluator | null = null;
  /**
   * Identity of the request the store is waiting for. Every new request and every `AC` bumps it, so
   * an answer whose ticket is no longer current belongs to a calculation nobody is waiting for.
   */
  let ticket = 0;
  /** The request the last effect sent, for {@link CalculatorStore.retry}. Not state: retrying is
   *  never rendered from, only triggered. */
  let lastSentRequest: CalculateRequest | null = null;

  function settle(sent: number, resolve: (calc: engine.CalcState) => engine.Transition): void {
    if (sent !== ticket) {
      return;
    }
    set({ calc: resolve(get().calc).state, pending: false, errorRequestId: null, canRetry: false });
  }

  function settleError(sent: number, error: unknown): void {
    if (sent !== ticket) {
      return;
    }
    const requestId = isApiError(error) ? (error.requestId ?? null) : null;
    const canRetry = isApiError(error) && RETRYABLE_ERROR_CODES.has(error.code);
    set({
      calc: engine.applyError(get().calc, messageForError(error)).state,
      pending: false,
      errorRequestId: requestId,
      canRetry,
    });
  }

  function send(state: engine.CalcState, request: CalculateRequest): void {
    const sent = (ticket += 1);
    lastSentRequest = request;
    set({ calc: state, pending: true, errorRequestId: null, canRetry: false });

    const evaluate = evaluator;
    if (evaluate === null) {
      // Nothing is wired up to answer (the store is used outside the app). Fail the press rather
      // than leave the UI pending for ever.
      settleError(sent, undefined);
      return;
    }
    void evaluate(request).then(
      (response) => {
        settle(sent, (calc) => engine.applyResult(calc, response));
      },
      (error: unknown) => {
        settleError(sent, error);
      },
    );
  }

  function apply({ state, effect }: engine.Transition): void {
    if (effect === undefined) {
      set({ calc: state });
      return;
    }
    send(state, effect.request);
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
    errorRequestId: null,
    canRetry: false,

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
      set({
        calc: engine.clearAll(get().calc).state,
        pending: false,
        errorRequestId: null,
        canRetry: false,
      });
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
    retry: () => {
      if (get().pending || !get().canRetry || lastSentRequest === null) {
        return;
      }
      send(get().calc, lastSentRequest);
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

/** The failing request's server request id, for the alert's `title`, or `null`. */
export const selectErrorRequestId = (state: CalculatorStore): string | null => state.errorRequestId;

/** Whether the alert should offer a Retry button. */
export const selectCanRetry = (state: CalculatorStore): boolean => state.canRetry;

import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api";
import { initialState } from "@/lib/calculator-engine";
import { FALLBACK_ERROR_MESSAGE } from "@/lib/error-messages";
import {
  selectDisplay,
  selectError,
  selectExpression,
  selectIsBusy,
  selectPhase,
  useCalculatorStore,
  type Evaluator,
} from "@/stores/useCalculatorStore";
import type { CalculateRequest, CalculateResponse } from "@/types/calculator";

/** The store is a module singleton; `clearAll` is the reset button, pending request included. */
function store(): ReturnType<typeof useCalculatorStore.getState> {
  return useCalculatorStore.getState();
}

function answer(request: CalculateRequest, result: number): CalculateResponse {
  return { ...request, result };
}

/** A promise the test resolves by hand, to hold a request in flight. */
function deferred<T>(): {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (reason: unknown) => void;
} {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

/** Let every already-scheduled microtask run, so "nothing happened" can be asserted. */
async function flush(): Promise<void> {
  for (let i = 0; i < 3; i += 1) {
    await Promise.resolve();
  }
}

const divisionByZero = new ApiError({
  status: 422,
  code: "DIVISION_BY_ZERO",
  title: "Division by zero",
  detail: `b must be non-zero for operation "divide"`,
  errors: [{ field: "b", message: "must be non-zero" }],
});

/** Answers every request with the arithmetic the backend would do. */
const arithmetic: Evaluator = (request) => {
  const b = request.b ?? 0;
  switch (request.operation) {
    case "add":
      return Promise.resolve(answer(request, request.a + b));
    case "multiply":
      return Promise.resolve(answer(request, request.a * b));
    case "divide":
      return b === 0
        ? Promise.reject(divisionByZero)
        : Promise.resolve(answer(request, request.a / b));
    case "sqrt":
      return Promise.resolve(answer(request, Math.sqrt(request.a)));
    case "percent":
      return Promise.resolve(answer(request, (request.a * b) / 100));
    default:
      return Promise.reject(new Error(`unexpected operation: ${request.operation}`));
  }
};

beforeEach(() => {
  store().clearAll();
  store().setEvaluator(arithmetic);
});

describe("useCalculatorStore", () => {
  it("starts on the engine's initial state, with nothing in flight", () => {
    expect(store().calc).toEqual(initialState);
    expect(store().pending).toBe(false);
  });

  it("drives a full calculation through the injected evaluator", async () => {
    const evaluator = vi.fn(arithmetic);
    store().setEvaluator(evaluator);

    store().inputDigit("1");
    store().inputDigit("2");
    store().setOperator("add");
    store().inputDigit("7");

    expect(selectDisplay(store())).toBe("7");
    expect(selectExpression(store())).toBe("12 +");

    store().evaluate();

    expect(selectIsBusy(store())).toBe(true);
    await vi.waitFor(() => {
      expect(selectIsBusy(store())).toBe(false);
    });

    expect(evaluator).toHaveBeenCalledExactlyOnceWith({ operation: "add", a: 12, b: 7 });
    expect(selectDisplay(store())).toBe("19");
    expect(selectPhase(store())).toBe("result");
    expect(selectError(store())).toBeNull();
    expect(selectExpression(store())).toBe("12 + 7 =");
  });

  it("maps a rejected ApiError to its user-facing sentence", async () => {
    store().inputDigit("5");
    store().setOperator("divide");
    store().inputDigit("0");
    store().evaluate();

    await vi.waitFor(() => {
      expect(selectPhase(store())).toBe("error");
    });

    expect(selectError(store())).toBe("Cannot divide by zero.");
    expect(selectIsBusy(store())).toBe(false);
    expect(selectDisplay(store())).toBe("0");
  });

  it("falls back to the generic sentence for a rejection that is not an ApiError", async () => {
    store().setEvaluator(() => Promise.reject(new TypeError("boom")));

    store().inputDigit("1");
    store().setOperator("add");
    store().inputDigit("1");
    store().evaluate();

    await vi.waitFor(() => {
      expect(selectPhase(store())).toBe("error");
    });

    expect(selectError(store())).toBe(FALLBACK_ERROR_MESSAGE);
  });

  it("fails the press instead of hanging when no evaluator is wired", () => {
    store().setEvaluator(null);

    store().inputDigit("9");
    store().applyUnary("sqrt");

    expect(selectPhase(store())).toBe("error");
    expect(selectError(store())).toBe(FALLBACK_ERROR_MESSAGE);
    expect(selectIsBusy(store())).toBe(false);
  });

  it("ignores every key except AC while a request is in flight", async () => {
    const pending = deferred<CalculateResponse>();
    const evaluator = vi.fn(() => pending.promise);
    store().setEvaluator(evaluator);

    store().inputDigit("8");
    store().setOperator("add");
    store().inputDigit("2");
    store().evaluate();

    expect(selectIsBusy(store())).toBe(true);

    store().inputDigit("5");
    store().inputDecimal();
    store().toggleSign();
    store().backspace();
    store().clearEntry();
    store().setOperator("multiply");
    store().applyUnary("percent");
    store().evaluate();

    expect(evaluator).toHaveBeenCalledTimes(1);
    expect(selectDisplay(store())).toBe("2");

    pending.resolve({ operation: "add", a: 8, b: 2, result: 10 });
    await vi.waitFor(() => {
      expect(selectIsBusy(store())).toBe(false);
    });

    expect(selectDisplay(store())).toBe("10");
  });

  it("discards a response that arrives after AC", async () => {
    const pending = deferred<CalculateResponse>();
    store().setEvaluator(() => pending.promise);

    store().inputDigit("8");
    store().setOperator("add");
    store().inputDigit("2");
    store().evaluate();

    store().clearAll();

    expect(store().calc).toEqual(initialState);
    expect(selectIsBusy(store())).toBe(false);

    pending.resolve({ operation: "add", a: 8, b: 2, result: 10 });
    await flush();

    expect(store().calc).toEqual(initialState);
    expect(selectIsBusy(store())).toBe(false);
  });

  it("discards a failure that arrives after AC", async () => {
    const pending = deferred<CalculateResponse>();
    store().setEvaluator(() => pending.promise);

    store().inputDigit("5");
    store().setOperator("divide");
    store().inputDigit("0");
    store().evaluate();

    store().clearAll();
    pending.reject(divisionByZero);
    await flush();

    expect(store().calc).toEqual(initialState);
    expect(selectPhase(store())).toBe("idle");
    expect(selectError(store())).toBeNull();
  });

  it("chains an operator pressed mid-entry through the server", async () => {
    store().inputDigit("2");
    store().setOperator("add");
    store().inputDigit("3");
    store().setOperator("multiply");

    await vi.waitFor(() => {
      expect(selectDisplay(store())).toBe("5");
    });

    store().inputDigit("4");
    store().evaluate();

    await vi.waitFor(() => {
      expect(selectDisplay(store())).toBe("20");
    });

    expect(selectExpression(store())).toBe("5 × 4 =");
  });

  it("mirrors the engine's entry keys", () => {
    store().inputDigit("1");
    store().inputDecimal();
    store().inputDigit("5");
    store().toggleSign();

    expect(selectDisplay(store())).toBe("-1.5");

    store().backspace();

    expect(selectDisplay(store())).toBe("-1.");

    store().clearEntry();

    expect(store().calc).toEqual(initialState);
  });

  it("applies a percentage to the pending operation", async () => {
    store().inputDigit("2");
    store().inputDigit("0");
    store().inputDigit("0");
    store().setOperator("add");
    store().inputDigit("1");
    store().inputDigit("5");
    store().applyUnary("percent");

    await vi.waitFor(() => {
      expect(selectDisplay(store())).toBe("30");
    });

    store().evaluate();

    await vi.waitFor(() => {
      expect(selectDisplay(store())).toBe("230");
    });
  });

  describe("selectors", () => {
    it("report an idle store", () => {
      expect(selectDisplay(store())).toBe("0");
      expect(selectPhase(store())).toBe("idle");
      expect(selectIsBusy(store())).toBe(false);
      expect(selectError(store())).toBeNull();
      expect(selectExpression(store())).toBe("");
    });

    it("report the expression while the second operand is being typed", () => {
      store().inputDigit("1");
      store().inputDigit("2");
      store().setOperator("add");

      expect(selectExpression(store())).toBe("12 +");

      store().inputDigit("7");

      expect(selectExpression(store())).toBe("12 +");
      expect(selectDisplay(store())).toBe("7");
    });
  });
});

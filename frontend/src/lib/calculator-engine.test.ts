import { describe, expect, it } from "vitest";

import { ApiError } from "@/lib/api";
import {
  applyError,
  applyResult,
  applyUnary,
  backspace,
  BINARY_OPERATIONS,
  clearAll,
  clearEntry,
  expressionOf,
  initialState,
  inputDecimal,
  inputDigit,
  MAX_ENTRY_DIGITS,
  OPERATION_SYMBOLS,
  requestEvaluate,
  setOperator,
  toggleSign,
  UNARY_KEYS,
  type BinaryOperation,
  type CalcState,
  type Digit,
  type Phase,
  type Transition,
} from "@/lib/calculator-engine";
import { messageForError } from "@/lib/error-messages";
import { FALLBACK_OPERATIONS } from "@/hooks/use-operations";
import type { CalculateRequest, CalculateResponse } from "@/types/calculator";

// --- fixtures ---------------------------------------------------------------------------------

/** A state to transition from or to. Everything not named keeps its initial value. */
function s(overrides: Partial<CalcState> = {}): CalcState {
  return { ...initialState, ...overrides };
}

/** A server answer. Only `result` matters to the engine; the echo is filled in for realism. */
function response(result: number): CalculateResponse {
  return { operation: "add", a: 0, b: 0, result };
}

const ALL_DIGITS = [
  "0",
  "1",
  "2",
  "3",
  "4",
  "5",
  "6",
  "7",
  "8",
  "9",
] as const satisfies readonly Digit[];

interface TransitionCase {
  readonly name: string;
  readonly from: CalcState;
  readonly act: (state: CalcState) => Transition;
  readonly to: CalcState;
  /** The request the transition must ask for, if any. */
  readonly effect?: CalculateRequest;
}

/** Assert one row of the table: the next state, the effect, and that the input was not mutated. */
function check({ from, act, to, effect }: TransitionCase): void {
  const before = structuredClone(from);

  const transition = act(from);

  expect(transition.state).toEqual(to);
  expect(transition.effect).toEqual(
    effect === undefined ? undefined : { kind: "calculate", request: effect },
  );
  expect(from).toEqual(before);
}

// --- transition table: from-state × input → to-state -------------------------------------------

describe("inputDigit", () => {
  const cases: readonly TransitionCase[] = [
    {
      name: "idle: starts the first operand",
      from: s(),
      act: (state) => inputDigit(state, "7"),
      to: s({ phase: "enteringA", display: "7" }),
    },
    {
      name: "enteringA: appends",
      from: s({ phase: "enteringA", display: "1" }),
      act: (state) => inputDigit(state, "2"),
      to: s({ phase: "enteringA", display: "12" }),
    },
    {
      name: "enteringA: the first digit replaces the placeholder zero",
      from: s({ phase: "enteringA", display: "0" }),
      act: (state) => inputDigit(state, "5"),
      to: s({ phase: "enteringA", display: "5" }),
    },
    {
      name: "enteringA: a zero after the point is kept",
      from: s({ phase: "enteringA", display: "0." }),
      act: (state) => inputDigit(state, "0"),
      to: s({ phase: "enteringA", display: "0.0" }),
    },
    {
      name: `enteringA: the ${String(MAX_ENTRY_DIGITS)}-digit cap ignores further digits`,
      from: s({ phase: "enteringA", display: "1234567890123456" }),
      act: (state) => inputDigit(state, "7"),
      to: s({ phase: "enteringA", display: "1234567890123456" }),
    },
    {
      name: "operatorSelected: starts the second operand",
      from: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      act: (state) => inputDigit(state, "7"),
      to: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
    },
    {
      name: "enteringB: appends",
      from: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
      act: (state) => inputDigit(state, "0"),
      to: s({ phase: "enteringB", display: "70", a: 12, operator: "add" }),
    },
    {
      name: "result: starts a fresh calculation",
      from: s({
        phase: "result",
        display: "19",
        a: 19,
        b: 7,
        operator: "add",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: (state) => inputDigit(state, "3"),
      to: s({ phase: "enteringA", display: "3" }),
    },
    {
      name: "error: clears the error first",
      from: s({ phase: "error", display: "0", error: "Cannot divide by zero." }),
      act: (state) => inputDigit(state, "5"),
      to: s({ phase: "enteringA", display: "5" }),
    },
  ];

  it.each(cases)("$name", check);

  it.each(ALL_DIGITS)("accepts the digit %s", (digit) => {
    expect(inputDigit(initialState, digit).state.display).toBe(digit);
  });

  it("caps the entry at 16 digits; the sign and the point are not digits", () => {
    let state = s({ phase: "enteringA", display: "-0." });
    for (let i = 0; i < 20; i += 1) {
      state = inputDigit(state, "9").state;
    }

    // The leading zero is one of the sixteen, so fifteen nines fit after the point.
    expect(state.display).toBe(`-0.${"9".repeat(MAX_ENTRY_DIGITS - 1)}`);
    expect(state.display.replace(/[^0-9]/g, "")).toHaveLength(MAX_ENTRY_DIGITS);
  });
});

describe("inputDecimal", () => {
  const cases: readonly TransitionCase[] = [
    {
      name: "idle: starts `0.`",
      from: s(),
      act: inputDecimal,
      to: s({ phase: "enteringA", display: "0." }),
    },
    {
      name: "enteringA: appends the point",
      from: s({ phase: "enteringA", display: "12" }),
      act: inputDecimal,
      to: s({ phase: "enteringA", display: "12." }),
    },
    {
      name: "enteringA: a second point is ignored",
      from: s({ phase: "enteringA", display: "1.2" }),
      act: inputDecimal,
      to: s({ phase: "enteringA", display: "1.2" }),
    },
    {
      name: "operatorSelected: starts the second operand at `0.`",
      from: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      act: inputDecimal,
      to: s({ phase: "enteringB", display: "0.", a: 12, operator: "add" }),
    },
    {
      name: "enteringB: appends the point",
      from: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
      act: inputDecimal,
      to: s({ phase: "enteringB", display: "7.", a: 12, operator: "add" }),
    },
    {
      name: "result: starts a fresh entry",
      from: s({ phase: "result", display: "19", a: 19 }),
      act: inputDecimal,
      to: s({ phase: "enteringA", display: "0." }),
    },
    {
      name: "error: clears the error first",
      from: s({ phase: "error", display: "19", error: "Cannot divide by zero." }),
      act: inputDecimal,
      to: s({ phase: "enteringA", display: "0." }),
    },
  ];

  it.each(cases)("$name", check);
});

describe("toggleSign", () => {
  const cases: readonly TransitionCase[] = [
    { name: "idle: nothing to negate", from: s(), act: toggleSign, to: s() },
    {
      name: "error: clears the error",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: toggleSign,
      to: s(),
    },
    {
      name: "enteringA: negates the entry",
      from: s({ phase: "enteringA", display: "5" }),
      act: toggleSign,
      to: s({ phase: "enteringA", display: "-5" }),
    },
    {
      name: "enteringA: negates back",
      from: s({ phase: "enteringA", display: "-5" }),
      act: toggleSign,
      to: s({ phase: "enteringA", display: "5" }),
    },
    {
      name: "enteringA: `-0` is normalised to `0`",
      from: s({ phase: "enteringA", display: "0" }),
      act: toggleSign,
      to: s({ phase: "enteringA", display: "0" }),
    },
    {
      name: "enteringB: negates the entry",
      from: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
      act: toggleSign,
      to: s({ phase: "enteringB", display: "-7", a: 12, operator: "add" }),
    },
    {
      name: "operatorSelected: negates `a`, which is what the display shows",
      from: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      act: toggleSign,
      to: s({ phase: "operatorSelected", display: "-12", a: -12, operator: "add" }),
    },
    {
      name: "operatorSelected: a missing `a` negates to zero, not to `-0`",
      from: s({ phase: "operatorSelected", display: "12", a: null, operator: "add" }),
      act: toggleSign,
      to: s({ phase: "operatorSelected", display: "0", a: 0, operator: "add" }),
    },
    {
      name: "result: negates the result",
      from: s({
        phase: "result",
        display: "19",
        a: 19,
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: toggleSign,
      to: s({
        phase: "result",
        display: "-19",
        a: -19,
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
    },
  ];

  it.each(cases)("$name", check);
});

describe("backspace", () => {
  const cases: readonly TransitionCase[] = [
    { name: "idle: nothing to delete", from: s(), act: backspace, to: s() },
    {
      name: "error: clears the error",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: backspace,
      to: s(),
    },
    {
      name: "enteringA: drops the last character",
      from: s({ phase: "enteringA", display: "12" }),
      act: backspace,
      to: s({ phase: "enteringA", display: "1" }),
    },
    {
      name: "enteringA: the last character leaves a zero",
      from: s({ phase: "enteringA", display: "7" }),
      act: backspace,
      to: s({ phase: "enteringA", display: "0" }),
    },
    {
      name: "enteringA: a lone sign leaves a zero",
      from: s({ phase: "enteringA", display: "-7" }),
      act: backspace,
      to: s({ phase: "enteringA", display: "0" }),
    },
    {
      name: "enteringB: drops the last character",
      from: s({ phase: "enteringB", display: "70", a: 12, operator: "add" }),
      act: backspace,
      to: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
    },
    {
      name: "operatorSelected: there is no entry to edit",
      from: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      act: backspace,
      to: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
    },
    {
      name: "result: a result is not typed text",
      from: s({ phase: "result", display: "19", a: 19 }),
      act: backspace,
      to: s({ phase: "result", display: "19", a: 19 }),
    },
  ];

  it.each(cases)("$name", check);
});

describe("clearEntry", () => {
  const cases: readonly TransitionCase[] = [
    { name: "idle: already clear", from: s(), act: clearEntry, to: s() },
    {
      name: "enteringA: clears the entry",
      from: s({ phase: "enteringA", display: "12" }),
      act: clearEntry,
      to: s(),
    },
    {
      name: "result: clears the result",
      from: s({
        phase: "result",
        display: "19",
        a: 19,
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: clearEntry,
      to: s(),
    },
    {
      name: "error: clears the error",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: clearEntry,
      to: s(),
    },
    {
      name: "operatorSelected: there is no entry yet",
      from: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      act: clearEntry,
      to: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
    },
    {
      name: "enteringB: keeps the pending operation and shows `a` again",
      from: s({ phase: "enteringB", display: "7", a: 12, b: 3, operator: "add" }),
      act: clearEntry,
      to: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
    },
    {
      name: "enteringB: a missing `a` shows zero",
      from: s({ phase: "enteringB", display: "7", a: null, operator: "add" }),
      act: clearEntry,
      to: s({ phase: "operatorSelected", display: "0", a: null, operator: "add" }),
    },
  ];

  it.each(cases)("$name", check);
});

describe("clearAll", () => {
  const phases: readonly Phase[] = [
    "idle",
    "enteringA",
    "operatorSelected",
    "enteringB",
    "result",
    "error",
  ];

  it.each(phases)("clears everything from %s", (phase) => {
    const from = s({
      phase,
      display: "-12.5",
      a: 12,
      b: 7,
      operator: "multiply",
      error: "Cannot divide by zero.",
      lastRequest: { operation: "add", a: 12, b: 7 },
    });

    const transition = clearAll(from);

    expect(transition.state).toEqual(initialState);
    expect(transition.effect).toBeUndefined();
  });
});

describe("setOperator", () => {
  const cases: readonly TransitionCase[] = [
    {
      name: "idle: the operator applies to zero",
      from: s(),
      act: (state) => setOperator(state, "add"),
      to: s({ phase: "operatorSelected", display: "0", a: 0, operator: "add" }),
    },
    {
      name: "error: clears the error first",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: (state) => setOperator(state, "add"),
      to: s({ phase: "operatorSelected", display: "0", a: 0, operator: "add" }),
    },
    {
      name: "enteringA: the entry becomes `a`",
      from: s({ phase: "enteringA", display: "12" }),
      act: (state) => setOperator(state, "add"),
      to: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
    },
    {
      name: "enteringA: `-0.` is parsed as a positive zero",
      from: s({ phase: "enteringA", display: "-0." }),
      act: (state) => setOperator(state, "divide"),
      to: s({ phase: "operatorSelected", display: "0", a: 0, operator: "divide" }),
    },
    {
      name: "operatorSelected: pressed twice, the second operator replaces the first",
      from: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      act: (state) => setOperator(state, "multiply"),
      to: s({ phase: "operatorSelected", display: "12", a: 12, operator: "multiply" }),
    },
    {
      name: "result: chains, the result becomes `a`",
      from: s({
        phase: "result",
        display: "19",
        a: 19,
        b: 7,
        operator: "add",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: (state) => setOperator(state, "multiply"),
      to: s({
        phase: "operatorSelected",
        display: "19",
        a: 19,
        operator: "multiply",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
    },
    {
      name: "enteringB: evaluates the pending operation first (immediate execution)",
      from: s({ phase: "enteringB", display: "3", a: 2, operator: "add" }),
      act: (state) => setOperator(state, "multiply"),
      to: s({ phase: "operatorSelected", display: "3", a: null, operator: "multiply" }),
      effect: { operation: "add", a: 2, b: 3 },
    },
    {
      name: "enteringB: a malformed state still produces a sendable request",
      from: s({ phase: "enteringB", display: "3", a: null, operator: null }),
      act: (state) => setOperator(state, "subtract"),
      to: s({ phase: "operatorSelected", display: "3", a: null, operator: "subtract" }),
      effect: { operation: "add", a: 0, b: 3 },
    },
  ];

  it.each(cases)("$name", check);

  it.each(BINARY_OPERATIONS)("accepts the %s key", (operation) => {
    expect(setOperator(s({ phase: "enteringA", display: "2" }), operation).state.operator).toBe(
      operation,
    );
  });
});

describe("requestEvaluate", () => {
  const cases: readonly TransitionCase[] = [
    { name: "idle: nothing to evaluate", from: s(), act: requestEvaluate, to: s() },
    {
      name: "error: clears the error",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: requestEvaluate,
      to: s(),
    },
    {
      name: "enteringA: with no operator the entry simply stands",
      from: s({ phase: "enteringA", display: "12" }),
      act: requestEvaluate,
      to: s({ phase: "result", display: "12", a: 12 }),
    },
    {
      name: "operatorSelected: with no second operand `a` is reused",
      from: s({ phase: "operatorSelected", display: "5", a: 5, operator: "add" }),
      act: requestEvaluate,
      to: s({
        phase: "result",
        display: "5",
        a: 5,
        b: 5,
        operator: "add",
        lastRequest: { operation: "add", a: 5, b: 5 },
      }),
      effect: { operation: "add", a: 5, b: 5 },
    },
    {
      name: "operatorSelected: a malformed state still produces a sendable request",
      from: s({ phase: "operatorSelected", display: "0", a: null, operator: null }),
      act: requestEvaluate,
      to: s({
        phase: "result",
        display: "0",
        a: null,
        b: 0,
        lastRequest: { operation: "add", a: 0, b: 0 },
      }),
      effect: { operation: "add", a: 0, b: 0 },
    },
    {
      name: "enteringB: evaluates a op b",
      from: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
      act: requestEvaluate,
      to: s({
        phase: "result",
        display: "7",
        a: 12,
        b: 7,
        operator: "add",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      effect: { operation: "add", a: 12, b: 7 },
    },
    {
      name: "result: repeats the last evaluation on the running total",
      from: s({
        phase: "result",
        display: "19",
        a: 19,
        b: 7,
        operator: "add",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: requestEvaluate,
      to: s({
        phase: "result",
        display: "19",
        a: 19,
        b: 7,
        operator: "add",
        lastRequest: { operation: "add", a: 19, b: 7 },
      }),
      effect: { operation: "add", a: 19, b: 7 },
    },
    {
      name: "result: a missing `a` repeats from zero",
      from: s({
        phase: "result",
        display: "19",
        a: null,
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: requestEvaluate,
      to: s({
        phase: "result",
        display: "19",
        a: null,
        lastRequest: { operation: "add", a: 0, b: 7 },
      }),
      effect: { operation: "add", a: 0, b: 7 },
    },
    {
      name: "result: with nothing to repeat it does nothing",
      from: s({ phase: "result", display: "12", a: 12 }),
      act: requestEvaluate,
      to: s({ phase: "result", display: "12", a: 12 }),
    },
  ];

  it.each(cases)("$name", check);
});

describe("applyUnary", () => {
  const cases: readonly TransitionCase[] = [
    {
      name: "sqrt in idle: the root of zero",
      from: s(),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "result" }),
      effect: { operation: "sqrt", a: 0 },
    },
    {
      name: "sqrt in error: clears the error first",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "result" }),
      effect: { operation: "sqrt", a: 0 },
    },
    {
      name: "sqrt in enteringA: applies to the entry",
      from: s({ phase: "enteringA", display: "9" }),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "result", display: "9" }),
      effect: { operation: "sqrt", a: 9 },
    },
    {
      name: "sqrt in operatorSelected: applies to `a` and stays pending",
      from: s({ phase: "operatorSelected", display: "9", a: 9, operator: "add" }),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "operatorSelected", display: "9", a: 9, operator: "add" }),
      effect: { operation: "sqrt", a: 9 },
    },
    {
      name: "sqrt in operatorSelected: a missing `a` roots zero",
      from: s({ phase: "operatorSelected", display: "9", a: null, operator: "add" }),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "operatorSelected", display: "9", a: null, operator: "add" }),
      effect: { operation: "sqrt", a: 0 },
    },
    {
      name: "sqrt in result: applies to the result",
      from: s({ phase: "result", display: "9", a: 9 }),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "result", display: "9", a: 9 }),
      effect: { operation: "sqrt", a: 9 },
    },
    {
      name: "sqrt in enteringB: applies to the entry, not to `a`",
      from: s({ phase: "enteringB", display: "9", a: 200, operator: "add" }),
      act: (state) => applyUnary(state, "sqrt"),
      to: s({ phase: "enteringB", display: "9", a: 200, operator: "add" }),
      effect: { operation: "sqrt", a: 9 },
    },
    {
      name: "percent in idle: one per cent of zero",
      from: s(),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "result" }),
      effect: { operation: "percent", a: 0, b: 1 },
    },
    {
      name: "percent in error: clears the error first",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "result" }),
      effect: { operation: "percent", a: 0, b: 1 },
    },
    {
      name: "percent in enteringA: the entry over one hundred",
      from: s({ phase: "enteringA", display: "50" }),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "result", display: "50" }),
      effect: { operation: "percent", a: 50, b: 1 },
    },
    {
      name: "percent in operatorSelected: `a` over one hundred",
      from: s({ phase: "operatorSelected", display: "50", a: 50, operator: "add" }),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "operatorSelected", display: "50", a: 50, operator: "add" }),
      effect: { operation: "percent", a: 50, b: 1 },
    },
    {
      name: "percent in result: the result over one hundred",
      from: s({ phase: "result", display: "50", a: 50 }),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "result", display: "50", a: 50 }),
      effect: { operation: "percent", a: 50, b: 1 },
    },
    {
      name: "percent in enteringB: b per cent of a",
      from: s({ phase: "enteringB", display: "15", a: 200, operator: "add" }),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "enteringB", display: "15", a: 200, operator: "add" }),
      effect: { operation: "percent", a: 200, b: 15 },
    },
    {
      name: "percent in enteringB: a malformed state still produces a sendable request",
      from: s({ phase: "enteringB", display: "15", a: null, operator: "add" }),
      act: (state) => applyUnary(state, "percent"),
      to: s({ phase: "enteringB", display: "15", a: null, operator: "add" }),
      effect: { operation: "percent", a: 0, b: 15 },
    },
  ];

  it.each(cases)("$name", check);

  it.each(UNARY_KEYS)("%s always asks the server rather than computing", (op) => {
    const transition = applyUnary(s({ phase: "enteringA", display: "9" }), op);

    expect(transition.effect?.kind).toBe("calculate");
    expect(transition.effect?.request.operation).toBe(op);
  });

  it("never sends `b` with sqrt: it is an arity-1 operation", () => {
    const request = applyUnary(s({ phase: "enteringA", display: "9" }), "sqrt").effect?.request;

    expect(request).toEqual({ operation: "sqrt", a: 9 });
    expect(request).not.toHaveProperty("b");
  });
});

describe("applyResult", () => {
  const cases: readonly TransitionCase[] = [
    {
      name: "enteringA: replaces the entry in place",
      from: s({ phase: "enteringA", display: "9" }),
      act: (state) => applyResult(state, response(3)),
      to: s({ phase: "enteringA", display: "3" }),
    },
    {
      name: "enteringB: replaces the entry so the pending operation can continue",
      from: s({ phase: "enteringB", display: "15", a: 200, operator: "add" }),
      act: (state) => applyResult(state, response(30)),
      to: s({ phase: "enteringB", display: "30", a: 200, operator: "add" }),
    },
    {
      name: "operatorSelected: becomes `a`",
      from: s({ phase: "operatorSelected", display: "3", a: null, operator: "multiply" }),
      act: (state) => applyResult(state, response(5)),
      to: s({ phase: "operatorSelected", display: "5", a: 5, operator: "multiply" }),
    },
    {
      name: "result: becomes the result",
      from: s({
        phase: "result",
        display: "7",
        a: 12,
        b: 7,
        operator: "add",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      act: (state) => applyResult(state, response(19)),
      to: s({
        phase: "result",
        display: "19",
        a: 19,
        operator: "add",
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
    },
    {
      name: "idle: an unexpected answer still shows its number",
      from: s(),
      act: (state) => applyResult(state, response(42)),
      to: s({ phase: "result", display: "42", a: 42 }),
    },
    {
      name: "error: an answer that arrives anyway clears the error",
      from: s({ phase: "error", display: "5", error: "Cannot divide by zero." }),
      act: (state) => applyResult(state, response(42)),
      to: s({ phase: "result", display: "42", a: 42 }),
    },
    {
      name: "a negative zero is displayed as zero",
      from: s({ phase: "result", display: "5", a: 5 }),
      act: (state) => applyResult(state, response(-0)),
      to: s({ phase: "result", display: "0", a: -0 }),
    },
  ];

  it.each(cases)("$name", check);
});

describe("applyError", () => {
  const phases: readonly Phase[] = [
    "idle",
    "enteringA",
    "operatorSelected",
    "enteringB",
    "result",
    "error",
  ];

  it.each(phases)("records the message and keeps the display, from %s", (phase) => {
    const from = s({ phase, display: "0", a: 5, operator: "divide" });

    const transition = applyError(from, "Cannot divide by zero.");

    expect(transition.state).toEqual(
      s({
        phase: "error",
        display: "0",
        a: 5,
        operator: "divide",
        error: "Cannot divide by zero.",
      }),
    );
    expect(transition.effect).toBeUndefined();
  });
});

describe("expressionOf", () => {
  it.each([
    { name: "idle", state: s(), expected: "" },
    { name: "enteringA", state: s({ phase: "enteringA", display: "12" }), expected: "" },
    {
      name: "error",
      state: s({ phase: "error", error: "Cannot divide by zero." }),
      expected: "",
    },
    {
      name: "operatorSelected",
      state: s({ phase: "operatorSelected", display: "12", a: 12, operator: "add" }),
      expected: "12 +",
    },
    {
      name: "enteringB",
      state: s({ phase: "enteringB", display: "7", a: 12, operator: "add" }),
      expected: "12 +",
    },
    {
      name: "operatorSelected without an operator",
      state: s({ phase: "operatorSelected", display: "12", a: 12 }),
      expected: "",
    },
    {
      name: "operatorSelected without an `a`",
      state: s({ phase: "operatorSelected", display: "12", a: null, operator: "divide" }),
      expected: "0 ÷",
    },
    {
      name: "result",
      state: s({
        phase: "result",
        display: "19",
        a: 19,
        lastRequest: { operation: "add", a: 12, b: 7 },
      }),
      expected: "12 + 7 =",
    },
    {
      name: "result of a unary request",
      state: s({
        phase: "result",
        display: "3",
        a: 3,
        lastRequest: { operation: "sqrt", a: 9 },
      }),
      expected: "9 √ =",
    },
    {
      name: "result with nothing to echo",
      state: s({ phase: "result", display: "12", a: 12 }),
      expected: "",
    },
  ])("renders $name as `$expected`", ({ state, expected }) => {
    expect(expressionOf(state)).toBe(expected);
  });

  it("uses the same symbols as the backend registry", () => {
    const backend = Object.fromEntries(FALLBACK_OPERATIONS.map((op) => [op.name, op.symbol]));

    expect(OPERATION_SYMBOLS).toEqual(backend);
  });
});

describe("an impossible phase", () => {
  // The `default` clause of every switch. It cannot happen while `Phase` is the union it is, but a
  // calculator that throws is worse than one that ignores a key, so each transition is a no-op.
  const impossible = s({ phase: "quantum" as unknown as Phase, display: "12" });

  it.each([
    { name: "inputDigit", act: (state: CalcState) => inputDigit(state, "7") },
    { name: "inputDecimal", act: inputDecimal },
    { name: "toggleSign", act: toggleSign },
    { name: "backspace", act: backspace },
    { name: "clearEntry", act: clearEntry },
    { name: "clearAll", act: clearAll },
    { name: "setOperator", act: (state: CalcState) => setOperator(state, "add") },
    { name: "requestEvaluate", act: requestEvaluate },
    { name: "applyUnary", act: (state: CalcState) => applyUnary(state, "sqrt") },
    { name: "applyResult", act: (state: CalcState) => applyResult(state, response(1)) },
    { name: "applyError", act: (state: CalcState) => applyError(state, "nope") },
  ])("is ignored by $name", ({ act }) => {
    const transition = act(impossible);

    expect(transition.state).toEqual(impossible);
    expect(transition.effect).toBeUndefined();
  });

  it("is ignored by expressionOf", () => {
    expect(expressionOf(impossible)).toBe("");
  });
});

// --- scripted sequences -----------------------------------------------------------------------

type OperatorKey = "+" | "−" | "×" | "÷" | "^";
type Key = Digit | "." | "±" | "⌫" | "CE" | "AC" | "=" | "√" | "%" | OperatorKey;

const BINARY_BY_KEY: Readonly<Record<OperatorKey, BinaryOperation>> = {
  "+": "add",
  "−": "subtract",
  "×": "multiply",
  "÷": "divide",
  "^": "power",
};

const DIGIT_KEYS: readonly string[] = ALL_DIGITS;

function isDigit(key: Key): key is Digit {
  return DIGIT_KEYS.includes(key);
}

function press(state: CalcState, key: Key): Transition {
  if (isDigit(key)) {
    return inputDigit(state, key);
  }
  switch (key) {
    case ".":
      return inputDecimal(state);
    case "±":
      return toggleSign(state);
    case "⌫":
      return backspace(state);
    case "CE":
      return clearEntry(state);
    case "AC":
      return clearAll(state);
    case "=":
      return requestEvaluate(state);
    case "√":
      return applyUnary(state, "sqrt");
    case "%":
      return applyUnary(state, "percent");
    default:
      return setOperator(state, BINARY_BY_KEY[key]);
  }
}

/** The backend, reimplemented from `backend/internal/calc` for the sequences below. */
function fakeServer(request: CalculateRequest): CalculateResponse {
  const { operation, a } = request;
  const b = request.b ?? 0;
  switch (operation) {
    case "add":
      return { ...request, result: a + b };
    case "subtract":
      return { ...request, result: a - b };
    case "multiply":
      return { ...request, result: a * b };
    case "divide":
      if (b === 0) {
        throw new ApiError({
          status: 422,
          code: "DIVISION_BY_ZERO",
          title: "Division by zero",
          detail: `b must be non-zero for operation "divide"`,
        });
      }
      return { ...request, result: a / b };
    case "power":
      return { ...request, result: a ** b };
    case "sqrt":
      if (a < 0) {
        throw new ApiError({
          status: 422,
          code: "DOMAIN_ERROR",
          title: "Result is not a real number",
          detail: `operation "sqrt" has no real-number result for these operands`,
        });
      }
      return { ...request, result: Math.sqrt(a) };
    case "percent":
      return { ...request, result: (a * b) / 100 };
  }
}

interface Run {
  /** The display after every key, in order. */
  readonly displays: readonly string[];
  /** Every request the engine asked for, in order. */
  readonly requests: readonly CalculateRequest[];
  readonly state: CalcState;
}

/** Press the keys, performing each effect immediately, as the store does asynchronously. */
function run(keys: readonly Key[]): Run {
  const displays: string[] = [];
  const requests: CalculateRequest[] = [];
  let state = initialState;

  for (const key of keys) {
    const transition = press(state, key);
    state = transition.state;
    const effect = transition.effect;
    if (effect !== undefined) {
      requests.push(effect.request);
      try {
        state = applyResult(state, fakeServer(effect.request)).state;
      } catch (error) {
        state = applyError(state, messageForError(error)).state;
      }
    }
    displays.push(state.display);
  }

  return { displays, requests, state };
}

describe("sequences", () => {
  it("`12 + 7 =` is 19", () => {
    const { displays, requests, state } = run(["1", "2", "+", "7", "="]);

    expect(displays).toEqual(["1", "12", "12", "7", "19"]);
    expect(requests).toEqual([{ operation: "add", a: 12, b: 7 }]);
    expect(state.phase).toBe("result");
    expect(expressionOf(state)).toBe("12 + 7 =");
  });

  it("`5 ÷ 0 =` fails with the mapped message and no arithmetic of our own", () => {
    const { displays, requests, state } = run(["5", "÷", "0", "="]);

    expect(displays).toEqual(["5", "5", "0", "0"]);
    expect(requests).toEqual([{ operation: "divide", a: 5, b: 0 }]);
    expect(state.phase).toBe("error");
    expect(state.error).toBe("Cannot divide by zero.");
  });

  it("recovers from an error: the next digit starts a fresh calculation", () => {
    const { state } = run(["5", "÷", "0", "=", "8", "+", "1", "="]);

    expect(state.phase).toBe("result");
    expect(state.display).toBe("9");
    expect(state.error).toBeNull();
  });

  it("`AC` clears an error completely", () => {
    const { state } = run(["5", "÷", "0", "=", "AC"]);

    expect(state).toEqual(initialState);
  });

  it("`9 √` is 3 and sends an arity-1 request", () => {
    const { displays, requests } = run(["9", "√"]);

    expect(displays).toEqual(["9", "3"]);
    expect(requests).toEqual([{ operation: "sqrt", a: 9 }]);
    expect(requests[0]).not.toHaveProperty("b");
  });

  it("the square root of a negative number is the server's refusal", () => {
    const { state } = run(["9", "±", "√"]);

    expect(state.phase).toBe("error");
    expect(state.error).toBe("That calculation has no real-number result.");
  });

  it("`200 + 15 % =` is 230: the percentage replaces the entry", () => {
    const { displays, requests, state } = run(["2", "0", "0", "+", "1", "5", "%", "="]);

    expect(displays).toEqual(["2", "20", "200", "200", "1", "15", "30", "230"]);
    expect(requests).toEqual([
      { operation: "percent", a: 200, b: 15 },
      { operation: "add", a: 200, b: 30 },
    ]);
    expect(state.display).toBe("230");
  });

  it("`50 %` alone is one per cent of fifty, asked for on the server", () => {
    const { displays, requests } = run(["5", "0", "%"]);

    expect(displays).toEqual(["5", "50", "0.5"]);
    expect(requests).toEqual([{ operation: "percent", a: 50, b: 1 }]);
  });

  it("`= = =` repeats the last operation on the running total", () => {
    const { displays, requests } = run(["1", "2", "+", "7", "=", "=", "="]);

    expect(displays).toEqual(["1", "12", "12", "7", "19", "26", "33"]);
    expect(requests).toEqual([
      { operation: "add", a: 12, b: 7 },
      { operation: "add", a: 19, b: 7 },
      { operation: "add", a: 26, b: 7 },
    ]);
  });

  it("`5 + =` reuses `a` as the second operand", () => {
    const { displays, requests } = run(["5", "+", "="]);

    expect(displays).toEqual(["5", "5", "10"]);
    expect(requests).toEqual([{ operation: "add", a: 5, b: 5 }]);
  });

  it("`2 + 3 × 4 =` is 20: immediate execution, no precedence", () => {
    const { displays, requests, state } = run(["2", "+", "3", "×", "4", "="]);

    expect(displays).toEqual(["2", "2", "3", "5", "4", "20"]);
    expect(requests).toEqual([
      { operation: "add", a: 2, b: 3 },
      { operation: "multiply", a: 5, b: 4 },
    ]);
    expect(state.display).toBe("20");
  });

  it("chains a new operation onto a result", () => {
    const { displays, state } = run(["1", "2", "+", "7", "=", "×", "2", "="]);

    expect(displays).toEqual(["1", "12", "12", "7", "19", "19", "2", "38"]);
    expect(state.display).toBe("38");
  });

  it("an operator pressed twice replaces the first one", () => {
    const { displays, requests } = run(["8", "+", "−", "2", "="]);

    expect(displays).toEqual(["8", "8", "8", "2", "6"]);
    expect(requests).toEqual([{ operation: "subtract", a: 8, b: 2 }]);
  });

  it("a digit after a result starts fresh", () => {
    const { displays, state } = run(["1", "2", "+", "7", "=", "3"]);

    expect(displays).toEqual(["1", "12", "12", "7", "19", "3"]);
    expect(state).toEqual(s({ phase: "enteringA", display: "3" }));
  });

  it("`⌫` edits the operand being typed", () => {
    const { displays, requests } = run(["1", "2", "3", "⌫", "+", "4", "5", "⌫", "="]);

    expect(displays).toEqual(["1", "12", "123", "12", "12", "4", "45", "4", "16"]);
    expect(requests).toEqual([{ operation: "add", a: 12, b: 4 }]);
  });

  it("`CE` clears the second operand but keeps the pending operation", () => {
    const { displays, requests } = run(["1", "2", "+", "9", "9", "CE", "7", "="]);

    expect(displays).toEqual(["1", "12", "12", "9", "99", "12", "7", "19"]);
    expect(requests).toEqual([{ operation: "add", a: 12, b: 7 }]);
  });

  it("`AC` in the middle of an entry starts over", () => {
    const { displays, state } = run(["1", "2", "+", "9", "AC", "4", "="]);

    expect(displays).toEqual(["1", "12", "12", "9", "0", "4", "4"]);
    expect(state).toEqual(s({ phase: "result", display: "4", a: 4 }));
  });

  it("types a decimal and a sign, then stops at the digit cap", () => {
    const nines: Key[] = Array.from({ length: 20 }, () => "9");

    const { state } = run(["1", ".", ".", "5", "±", ...nines]);

    expect(state.display).toBe(`-1.5${"9".repeat(MAX_ENTRY_DIGITS - 2)}`);
    expect(state.display.replace(/[^0-9]/g, "")).toHaveLength(MAX_ENTRY_DIGITS);
  });

  it("`2 ^ 1 0 =` is 1024", () => {
    const { requests, state } = run(["2", "^", "1", "0", "="]);

    expect(requests).toEqual([{ operation: "power", a: 2, b: 10 }]);
    expect(state.display).toBe("1024");
  });
});

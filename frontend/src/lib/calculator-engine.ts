/**
 * The calculator, as a pure state machine.
 *
 * Nothing here imports React, zustand or `fetch`: a key press is `(state, input) => { state, effect? }`,
 * and the single effect this module can ask for is "evaluate this request on the server". The engine
 * never does arithmetic itself — every number the user sees comes back from `POST /api/v1/calculate`
 * (ADR-0008) — so it can be read, reasoned about and tested as a table.
 *
 * Semantics are immediate execution with no operator precedence: pressing a binary operator while a
 * second operand is on the display finishes the pending operation first, so `2 + 3 × 4 =` is 20, not
 * 14. See docs/adr/0008-frontend-evaluation-semantics.md for the choice and its alternatives.
 *
 * The engine knows nothing about requests being in flight. `src/stores/useCalculatorStore.ts` owns
 * that: it performs the effect, ignores input while it is pending, and feeds the answer back through
 * {@link applyResult} or {@link applyError}.
 */
import type { CalculateRequest, CalculateResponse, OperationName } from "@/types/calculator";

/** Operations that take two operands, i.e. the ones a keypad operator key can select. */
export const BINARY_OPERATIONS = [
  "add",
  "subtract",
  "multiply",
  "divide",
  "power",
] as const satisfies readonly OperationName[];

export type BinaryOperation = (typeof BINARY_OPERATIONS)[number];

/**
 * Keys that act on the value already on the display. `percent` is binary on the wire — the backend
 * computes "b percent of a" — but it is a one-key press for the user, so it lives here.
 */
export const UNARY_KEYS = ["sqrt", "percent"] as const;

export type UnaryKey = (typeof UNARY_KEYS)[number];

export type Digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9";

/**
 * Where the machine is, and therefore what the next key means.
 *
 * - `idle` — nothing entered; the display shows a bare `0`.
 * - `enteringA` — the user is typing the first operand; the display *is* the entry.
 * - `operatorSelected` — an operator is pending and the display shows `a`.
 * - `enteringB` — the user is typing the second operand; the display *is* the entry.
 * - `result` — the display shows a value the server computed.
 * - `error` — the last request failed; {@link CalcState.error} holds the sentence to show.
 *
 * While a request is in flight the phase is already the one the answer will land in, so
 * {@link applyResult} needs no extra state to know where the number goes.
 */
export type Phase = "idle" | "enteringA" | "operatorSelected" | "enteringB" | "result" | "error";

export interface CalcState {
  readonly phase: Phase;
  /** What the main display line shows. In the entering phases this is the raw entry, not a format. */
  readonly display: string;
  /** First operand. Set when an operator is chosen, and replaced by every result. */
  readonly a: number | null;
  /** Second operand, as sent with the last evaluation. */
  readonly b: number | null;
  readonly operator: BinaryOperation | null;
  /** User-facing sentence for the failure, from `messageForError`. Non-null only in `error`. */
  readonly error: string | null;
  /**
   * The evaluation `=` last asked for, so pressing `=` again can repeat it on the running result
   * ("= = =" keeps adding 7). Only `=` records one: chaining, `√` and `%` do not.
   */
  readonly lastRequest: CalculateRequest | null;
}

export const initialState: CalcState = {
  phase: "idle",
  display: "0",
  a: null,
  b: null,
  operator: null,
  error: null,
  lastRequest: null,
};

/** The only thing the engine can ask the outside world to do. */
export interface CalculateEffect {
  readonly kind: "calculate";
  readonly request: CalculateRequest;
}

/** Result of a transition: the next state, plus the request the caller must perform, if any. */
export interface Transition {
  readonly state: CalcState;
  readonly effect?: CalculateEffect;
}

/**
 * Longest entry the user can type, counted in digits: the sign and the decimal point do not count,
 * and further digits are ignored rather than truncating what is already there. 16 is past the 15–17
 * significant digits a float64 can round-trip (ADR-0003), so nothing typable is silently lost.
 */
export const MAX_ENTRY_DIGITS = 16;

/** Display symbols, same list and same characters as the backend registry (`GET /operations`). */
export const OPERATION_SYMBOLS: Readonly<Record<OperationName, string>> = {
  add: "+",
  subtract: "−",
  multiply: "×",
  divide: "÷",
  power: "^",
  sqrt: "√",
  percent: "%",
};

// --- transitions ------------------------------------------------------------------------------

/** A digit key: starts, extends or replaces the entry depending on where we are. */
export function inputDigit(state: CalcState, digit: Digit): Transition {
  switch (state.phase) {
    // A digit after a result (or an error) starts a fresh calculation rather than editing it.
    case "idle":
    case "result":
    case "error":
      return stay({ ...initialState, phase: "enteringA", display: appendDigit("0", digit) });
    case "enteringA":
    case "enteringB":
      return stay({ ...state, display: appendDigit(state.display, digit) });
    case "operatorSelected":
      return stay({ ...state, phase: "enteringB", display: appendDigit("0", digit), b: null });
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/** The decimal point. A second one is ignored; on an empty entry it starts `0.`. */
export function inputDecimal(state: CalcState): Transition {
  switch (state.phase) {
    case "idle":
    case "result":
    case "error":
      return stay({ ...initialState, phase: "enteringA", display: "0." });
    case "enteringA":
    case "enteringB":
      return stay({ ...state, display: appendDecimal(state.display) });
    case "operatorSelected":
      return stay({ ...state, phase: "enteringB", display: "0.", b: null });
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/** `±`. On a bare `0` there is nothing to negate, so it does nothing. */
export function toggleSign(state: CalcState): Transition {
  switch (state.phase) {
    case "idle":
    case "error":
      return stay(initialState);
    case "enteringA":
    case "enteringB":
      return stay({ ...state, display: toggleEntrySign(state.display) });
    // Nothing is being typed, so `±` negates the value on the display, which is `a`.
    case "operatorSelected":
    case "result": {
      const value = negate(state.a);
      return stay({ ...state, a: value, display: toDisplay(value) });
    }
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/** `⌫`. Edits the entry only; the last character of a one-character entry leaves `0` behind. */
export function backspace(state: CalcState): Transition {
  switch (state.phase) {
    case "idle":
    case "error":
      return stay(initialState);
    case "enteringA":
    case "enteringB":
      return stay({ ...state, display: dropLastChar(state.display) });
    // There is no entry to edit: a pending operator and a finished result are not typed text.
    case "operatorSelected":
    case "result":
      return stay(state);
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/** `CE`. Clears what is being typed and keeps the pending operation; `AC` is {@link clearAll}. */
export function clearEntry(state: CalcState): Transition {
  switch (state.phase) {
    case "idle":
    case "enteringA":
    case "result":
    case "error":
      return stay(initialState);
    case "operatorSelected":
      return stay(state);
    case "enteringB":
      return stay({
        ...state,
        phase: "operatorSelected",
        display: toDisplay(state.a ?? 0),
        b: null,
      });
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/** `AC` / Escape. Always the same: back to a bare zero, whatever went before. */
export function clearAll(state: CalcState): Transition {
  switch (state.phase) {
    case "idle":
    case "enteringA":
    case "operatorSelected":
    case "enteringB":
    case "result":
    case "error":
      return stay(initialState);
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/**
 * A binary operator key. Pressed twice it replaces the operator; pressed on a result it chains
 * (the result becomes `a`); pressed while the second operand is on the display it evaluates the
 * pending operation first — that is what makes `2 + 3 × 4 =` 20.
 */
export function setOperator(state: CalcState, operator: BinaryOperation): Transition {
  switch (state.phase) {
    case "idle":
    case "error":
      return stay({ ...initialState, phase: "operatorSelected", a: 0, operator });
    case "enteringA": {
      const a = parseEntry(state.display);
      return stay({ ...state, phase: "operatorSelected", a, display: toDisplay(a), operator });
    }
    case "operatorSelected":
      return stay({ ...state, operator });
    case "result":
      return stay({ ...state, phase: "operatorSelected", operator, b: null });
    case "enteringB": {
      const b = parseEntry(state.display);
      // `a` is unknown until the pending operation comes back; the landing phase is already the
      // one that puts the answer there.
      return emit(
        { ...state, phase: "operatorSelected", a: null, b: null, operator },
        binaryRequest(state.operator, state.a, b),
      );
    }
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/**
 * `=` / Enter. With no second operand it reuses `a` (classic behaviour: `5 + =` is 10), and on a
 * result it repeats the last evaluation on the running total (`12 + 7 = = =` → 19, 26, 33).
 */
export function requestEvaluate(state: CalcState): Transition {
  switch (state.phase) {
    case "idle":
    case "error":
      return stay(initialState);
    // Nothing to compute: `=` just freezes the number that was typed.
    case "enteringA": {
      const a = parseEntry(state.display);
      return stay({ ...initialState, phase: "result", a, display: toDisplay(a) });
    }
    case "operatorSelected": {
      const a = state.a ?? 0;
      const request = binaryRequest(state.operator, a, a);
      return emit({ ...state, phase: "result", b: a, lastRequest: request }, request);
    }
    case "enteringB": {
      const b = parseEntry(state.display);
      const request = binaryRequest(state.operator, state.a, b);
      return emit({ ...state, phase: "result", b, lastRequest: request }, request);
    }
    case "result": {
      const last = state.lastRequest;
      if (last === null) {
        return stay(state);
      }
      const request: CalculateRequest = { ...last, a: state.a ?? 0 };
      return emit({ ...state, lastRequest: request }, request);
    }
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/**
 * `√` or `%`: keys that act on the value already on the display.
 *
 * `√` takes the entry being typed, or `a` when nothing is being typed. `%` with a pending binary
 * operation is `percent(a, b)` — "b percent of a" — and its answer replaces the entry so the pending
 * operation can finish with it (`200 + 15 % =` → 230). Anywhere else `%` is the plain "per cent of
 * this number" key, asked for as `percent(x, 1)` = x/100 so that the server still does the
 * arithmetic. Both are spelled out in ADR-0008.
 */
export function applyUnary(state: CalcState, op: UnaryKey): Transition {
  switch (state.phase) {
    case "idle":
    case "error":
      return emit({ ...initialState, phase: "result" }, unaryRequest(op, 0));
    case "enteringA":
      return emit({ ...state, phase: "result" }, unaryRequest(op, parseEntry(state.display)));
    case "operatorSelected":
    case "result":
      return emit(state, unaryRequest(op, state.a ?? 0));
    case "enteringB": {
      const entry = parseEntry(state.display);
      return emit(state, op === "sqrt" ? sqrtRequest(entry) : percentRequest(state.a ?? 0, entry));
    }
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/**
 * Feed a successful response back in. The phase says where the number belongs: it replaces the
 * entry while one is being typed (`√`, `%`) and becomes `a` everywhere else.
 */
export function applyResult(state: CalcState, response: CalculateResponse): Transition {
  const value = response.result;
  switch (state.phase) {
    case "enteringA":
    case "enteringB":
      return stay({ ...state, display: toDisplay(value), error: null });
    case "operatorSelected":
    case "result":
      return stay({ ...state, a: value, b: null, display: toDisplay(value), error: null });
    // Defensive: an answer for a calculation nothing is waiting for still shows its number.
    case "idle":
    case "error":
      return stay({ ...initialState, phase: "result", a: value, display: toDisplay(value) });
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/**
 * Feed a failure back in. `message` is already the sentence to show (`messageForError`), never a
 * code or a server `detail`. The display keeps the last good value; the next key clears both.
 */
export function applyError(state: CalcState, message: string): Transition {
  switch (state.phase) {
    case "idle":
    case "enteringA":
    case "operatorSelected":
    case "enteringB":
    case "result":
    case "error":
      return stay({ ...state, phase: "error", error: message });
    default:
      return exhausted(state.phase satisfies never, stay(state));
  }
}

/**
 * The secondary display line: what has been entered so far, for example `12 +` while the second
 * operand is being typed and `12 + 7 =` once the answer is showing. Empty when there is nothing
 * worth echoing.
 */
export function expressionOf(state: CalcState): string {
  switch (state.phase) {
    case "idle":
    case "enteringA":
    case "error":
      return "";
    case "operatorSelected":
    case "enteringB":
      return state.operator === null
        ? ""
        : `${toDisplay(state.a ?? 0)} ${OPERATION_SYMBOLS[state.operator]}`;
    case "result": {
      const last = state.lastRequest;
      if (last === null) {
        return "";
      }
      const operand = last.b === undefined ? "" : ` ${toDisplay(last.b)}`;
      return `${toDisplay(last.a)} ${OPERATION_SYMBOLS[last.operation]}${operand} =`;
    }
    default:
      return exhausted(state.phase satisfies never, "");
  }
}

// --- helpers ----------------------------------------------------------------------------------

function stay(state: CalcState): Transition {
  return { state };
}

function emit(state: CalcState, request: CalculateRequest): Transition {
  return { state, effect: { kind: "calculate", request } };
}

/**
 * The `default` clause of every switch above. The `never` parameter is what makes the switches
 * exhaustive: adding a member to {@link Phase} without handling it stops compiling here. At runtime
 * an impossible phase must not throw — a calculator that crashes is worse than one that ignores a
 * key — so the caller's fallback is returned unchanged.
 */
function exhausted<T>(_phase: never, fallback: T): T {
  return fallback;
}

/** Digits in the entry, ignoring the sign and the decimal point. */
function countDigits(entry: string): number {
  return entry.replace(/[^0-9]/g, "").length;
}

function appendDigit(entry: string, digit: Digit): string {
  if (countDigits(entry) >= MAX_ENTRY_DIGITS) {
    return entry;
  }
  // No leading zeros: the first digit replaces the placeholder instead of extending it.
  return entry === "0" ? digit : entry + digit;
}

function appendDecimal(entry: string): string {
  return entry.includes(".") ? entry : `${entry}.`;
}

function dropLastChar(entry: string): string {
  return normaliseEntry(entry.slice(0, -1));
}

function toggleEntrySign(entry: string): string {
  return normaliseEntry(entry.startsWith("-") ? entry.slice(1) : `-${entry}`);
}

/** An entry is never empty, never a lone sign and never a negative zero. */
function normaliseEntry(entry: string): string {
  return entry === "" || entry === "-" || entry === "-0" ? "0" : entry;
}

function parseEntry(entry: string): number {
  const value = Number(entry);
  // `Number("-0.")` is -0; neither the display nor the wire should ever carry a negative zero.
  return value === 0 ? 0 : value;
}

function negate(value: number | null): number {
  const negated = -(value ?? 0);
  return negated === 0 ? 0 : negated;
}

/**
 * How a computed value is written on the display. Deliberately minimal: significant-digit and
 * exponent formatting is ADR-0007's, and arrives with `format-number.ts` in F2-06.
 */
function toDisplay(value: number): string {
  return String(value);
}

/**
 * `operator` and `a` are typed nullable because the early phases have no operand yet. Every phase
 * that builds a request has both; the fallbacks keep a malformed state from producing a request
 * that cannot be sent.
 */
function binaryRequest(
  operator: BinaryOperation | null,
  a: number | null,
  b: number,
): CalculateRequest {
  return { operation: operator ?? "add", a: a ?? 0, b };
}

/** `sqrt` is arity 1: sending `b` at all is a 422 `UNEXPECTED_OPERAND`. */
function sqrtRequest(a: number): CalculateRequest {
  return { operation: "sqrt", a };
}

/** `percent(a, b)` is "b percent of a" (docs/PLAN.md §1.4): `percent(200, 15)` is 30. */
function percentRequest(a: number, b: number): CalculateRequest {
  return { operation: "percent", a, b };
}

/** A unary key with no operand to work against: `√x`, and `x %` as one per cent of x, times one. */
function unaryRequest(op: UnaryKey, value: number): CalculateRequest {
  return op === "sqrt" ? sqrtRequest(value) : percentRequest(value, 1);
}

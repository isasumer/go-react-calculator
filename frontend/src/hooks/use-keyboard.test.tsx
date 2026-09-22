import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { KEY_MAP, useKeyboard, type KeyboardHandlers } from "@/hooks/use-keyboard";
import type { BinaryOperation, Digit, UnaryKey } from "@/lib/calculator-engine";

/** The mapping is data, tested as data: every entry the issue lists, and nothing more. */
describe("KEY_MAP", () => {
  it.each([
    ["0", { type: "digit", digit: "0" }],
    ["1", { type: "digit", digit: "1" }],
    ["9", { type: "digit", digit: "9" }],
    [".", { type: "decimal" }],
    ["+", { type: "operator", operator: "add" }],
    ["-", { type: "operator", operator: "subtract" }],
    ["*", { type: "operator", operator: "multiply" }],
    ["/", { type: "operator", operator: "divide" }],
    ["^", { type: "operator", operator: "power" }],
    ["%", { type: "unary", op: "percent" }],
    ["r", { type: "unary", op: "sqrt" }],
    ["Enter", { type: "equals" }],
    ["=", { type: "equals" }],
    ["Escape", { type: "clearAll" }],
    ["Backspace", { type: "backspace" }],
    ["Delete", { type: "clearEntry" }],
  ] as const)("maps %s", (key, action) => {
    expect(KEY_MAP[key]).toEqual(action);
  });

  it("has no entries beyond the documented keys", () => {
    expect(Object.keys(KEY_MAP).sort()).toEqual(
      [
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
        ".",
        "+",
        "-",
        "*",
        "/",
        "^",
        "%",
        "r",
        "Enter",
        "=",
        "Escape",
        "Backspace",
        "Delete",
      ].sort(),
    );
  });
});

function makeHandlers(overrides: Partial<KeyboardHandlers> = {}) {
  const base = {
    onDigit: vi.fn<(digit: Digit) => void>(),
    onDecimal: vi.fn<() => void>(),
    onOperator: vi.fn<(operator: BinaryOperation) => void>(),
    onUnary: vi.fn<(op: UnaryKey) => void>(),
    onEquals: vi.fn<() => void>(),
    onClearAll: vi.fn<() => void>(),
    onClearEntry: vi.fn<() => void>(),
    onBackspace: vi.fn<() => void>(),
    onHandled: vi.fn<(key: string) => void>(),
    busy: false,
  } satisfies KeyboardHandlers;
  return { ...base, ...overrides };
}

/** A tree with an unrelated text input next to the calculator, exactly like the app's real DOM. */
function Harness({ handlers }: { handlers: KeyboardHandlers }) {
  useKeyboard(handlers);
  return (
    <div>
      <input aria-label="unrelated field" />
      <textarea aria-label="unrelated textarea" />
      <div aria-label="editable" contentEditable suppressContentEditableWarning />
      <button type="button">a button</button>
    </div>
  );
}

describe("useKeyboard", () => {
  it("routes a digit, an operator and Enter to the matching handlers", async () => {
    const user = userEvent.setup();
    const handlers = makeHandlers();
    render(<Harness handlers={handlers} />);

    await user.keyboard("1+7{Enter}");

    expect(handlers.onDigit).toHaveBeenNthCalledWith(1, "1");
    expect(handlers.onOperator).toHaveBeenNthCalledWith(1, "add");
    expect(handlers.onDigit).toHaveBeenNthCalledWith(2, "7");
    expect(handlers.onEquals).toHaveBeenCalledTimes(1);
    expect(handlers.onHandled).toHaveBeenCalledWith("1");
    expect(handlers.onHandled).toHaveBeenCalledWith("Enter");
  });

  it("routes Escape, Backspace, Delete and the unary keys", async () => {
    const user = userEvent.setup();
    const handlers = makeHandlers();
    render(<Harness handlers={handlers} />);

    await user.keyboard("{Escape}{Backspace}{Delete}r%^");

    expect(handlers.onClearAll).toHaveBeenCalledTimes(1);
    expect(handlers.onBackspace).toHaveBeenCalledTimes(1);
    expect(handlers.onClearEntry).toHaveBeenCalledTimes(1);
    expect(handlers.onUnary).toHaveBeenNthCalledWith(1, "sqrt");
    expect(handlers.onUnary).toHaveBeenNthCalledWith(2, "percent");
    expect(handlers.onOperator).toHaveBeenCalledWith("power");
  });

  it("ignores a key pressed with ctrl, meta or alt", async () => {
    const user = userEvent.setup();
    const handlers = makeHandlers();
    render(<Harness handlers={handlers} />);

    await user.keyboard("{Control>}1{/Control}");
    await user.keyboard("{Meta>}2{/Meta}");
    await user.keyboard("{Alt>}3{/Alt}");

    expect(handlers.onDigit).not.toHaveBeenCalled();
  });

  it("ignores keys typed into an input, a textarea or a contenteditable element", async () => {
    const user = userEvent.setup();
    const handlers = makeHandlers();
    render(<Harness handlers={handlers} />);

    await user.click(screen.getByLabelText("unrelated field"));
    await user.keyboard("5");
    await user.click(screen.getByLabelText("unrelated textarea"));
    await user.keyboard("6");
    await user.click(screen.getByLabelText("editable"));
    await user.keyboard("7");

    expect(handlers.onDigit).not.toHaveBeenCalled();
  });

  it("ignores every key but Escape while busy, and does not flash pressed for the ignored ones", async () => {
    const user = userEvent.setup();
    const handlers = makeHandlers({ busy: true });
    render(<Harness handlers={handlers} />);

    await user.keyboard("5+");
    expect(handlers.onDigit).not.toHaveBeenCalled();
    expect(handlers.onOperator).not.toHaveBeenCalled();
    expect(handlers.onHandled).not.toHaveBeenCalled();

    await user.keyboard("{Escape}");
    expect(handlers.onClearAll).toHaveBeenCalledTimes(1);
    expect(handlers.onHandled).toHaveBeenCalledWith("Escape");
  });

  it("calls preventDefault only for a key it actually handles", () => {
    const handlers = makeHandlers();
    render(<Harness handlers={handlers} />);

    const handled = new KeyboardEvent("keydown", { key: "1", bubbles: true, cancelable: true });
    const unhandled = new KeyboardEvent("keydown", { key: "F5", bubbles: true, cancelable: true });

    window.dispatchEvent(handled);
    window.dispatchEvent(unhandled);

    expect(handled.defaultPrevented).toBe(true);
    expect(unhandled.defaultPrevented).toBe(false);
    expect(handlers.onDigit).toHaveBeenCalledWith("1");
  });

  it("does not preventDefault a key that busy is ignoring", () => {
    const handlers = makeHandlers({ busy: true });
    render(<Harness handlers={handlers} />);

    const event = new KeyboardEvent("keydown", { key: "5", bubbles: true, cancelable: true });
    window.dispatchEvent(event);

    expect(event.defaultPrevented).toBe(false);
    expect(handlers.onDigit).not.toHaveBeenCalled();
  });

  it("removes its listener on unmount", async () => {
    const user = userEvent.setup();
    const handlers = makeHandlers();
    const { unmount } = render(<Harness handlers={handlers} />);

    unmount();
    await user.keyboard("1");

    expect(handlers.onDigit).not.toHaveBeenCalled();
  });
});

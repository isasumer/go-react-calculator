import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Keypad, type KeypadProps } from "@/components/calculator/Keypad";

function renderKeypad(busy = false) {
  const handlers = {
    onDigit: vi.fn(),
    onDecimal: vi.fn(),
    onToggleSign: vi.fn(),
    onBackspace: vi.fn(),
    onClearEntry: vi.fn(),
    onClearAll: vi.fn(),
    onOperator: vi.fn(),
    onEquals: vi.fn(),
  } satisfies Omit<KeypadProps, "busy">;
  render(<Keypad busy={busy} {...handlers} />);
  return handlers;
}

function keyNames(): (string | null)[] {
  return within(screen.getByRole("group", { name: "Keypad" }))
    .getAllByRole("button")
    .map((button) => button.getAttribute("aria-label"));
}

describe("Keypad", () => {
  it("lays the keys out in the order a hand expects", () => {
    renderKeypad();

    expect(screen.getByRole("group", { name: "Keypad" })).toHaveAttribute(
      "data-ui",
      "calculator.keypad",
    );
    expect(keyNames()).toEqual([
      "clear all",
      "clear entry",
      "backspace",
      "divide",
      "7",
      "8",
      "9",
      "multiply",
      "4",
      "5",
      "6",
      "subtract",
      "1",
      "2",
      "3",
      "add",
      "toggle sign",
      "0",
      "decimal point",
      "equals",
    ]);
  });

  it("routes every key to its action", async () => {
    const user = userEvent.setup();
    const handlers = renderKeypad();

    await user.click(screen.getByRole("button", { name: "7" }));
    await user.click(screen.getByRole("button", { name: "decimal point" }));
    await user.click(screen.getByRole("button", { name: "toggle sign" }));
    await user.click(screen.getByRole("button", { name: "backspace" }));
    await user.click(screen.getByRole("button", { name: "clear entry" }));
    await user.click(screen.getByRole("button", { name: "clear all" }));
    await user.click(screen.getByRole("button", { name: "subtract" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    expect(handlers.onDigit).toHaveBeenCalledExactlyOnceWith("7");
    expect(handlers.onDecimal).toHaveBeenCalledOnce();
    expect(handlers.onToggleSign).toHaveBeenCalledOnce();
    expect(handlers.onBackspace).toHaveBeenCalledOnce();
    expect(handlers.onClearEntry).toHaveBeenCalledOnce();
    expect(handlers.onClearAll).toHaveBeenCalledOnce();
    expect(handlers.onOperator).toHaveBeenCalledExactlyOnceWith("subtract");
    expect(handlers.onEquals).toHaveBeenCalledOnce();
  });

  it("disables everything but AC while a calculation is in flight", () => {
    renderKeypad(true);

    const buttons = within(screen.getByRole("group", { name: "Keypad" })).getAllByRole("button");
    for (const button of buttons) {
      if (button.getAttribute("aria-label") === "clear all") {
        expect(button).toBeEnabled();
      } else {
        expect(button).toBeDisabled();
      }
    }
  });

  it("announces the keyboard equivalent of the keys that have one", () => {
    renderKeypad();

    expect(screen.getByRole("button", { name: "clear all" })).toHaveAttribute(
      "aria-keyshortcuts",
      "Escape",
    );
    expect(screen.getByRole("button", { name: "equals" })).toHaveAttribute(
      "aria-keyshortcuts",
      "Enter",
    );
    expect(screen.getByRole("button", { name: "multiply" })).toHaveAttribute(
      "aria-keyshortcuts",
      "*",
    );
    // `±` has no key on a keyboard, so it claims none.
    expect(screen.getByRole("button", { name: "toggle sign" })).not.toHaveAttribute(
      "aria-keyshortcuts",
    );
  });
});

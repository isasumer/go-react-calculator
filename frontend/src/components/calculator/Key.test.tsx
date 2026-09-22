import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Key, type KeyVariant } from "@/components/calculator/Key";

describe("Key", () => {
  it("renders a button labelled in words and reports presses", async () => {
    const user = userEvent.setup();
    const onPress = vi.fn();
    render(<Key label="×" ariaLabel="multiply" onPress={onPress} />);

    const key = screen.getByRole("button", { name: "multiply" });
    expect(key).toHaveTextContent("×");
    expect(key).toHaveAttribute("data-ui", "calculator.key");
    expect(key).toHaveAttribute("type", "button");

    await user.click(key);

    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("defaults to the digit variant", () => {
    render(<Key label="7" ariaLabel="7" onPress={vi.fn()} />);

    expect(screen.getByRole("button", { name: "7" })).toHaveAttribute("data-key-variant", "digit");
  });

  it.each(["digit", "operator", "action", "equals"] as const satisfies readonly KeyVariant[])(
    "marks the %s variant on the button",
    (variant) => {
      render(<Key label="k" ariaLabel={variant} variant={variant} onPress={vi.fn()} />);

      expect(screen.getByRole("button", { name: variant })).toHaveAttribute(
        "data-key-variant",
        variant,
      );
    },
  );

  it("announces its keyboard equivalent and its tooltip", () => {
    render(
      <Key
        label="AC"
        ariaLabel="clear all"
        variant="action"
        keyShortcut="Escape"
        title="clears everything"
        onPress={vi.fn()}
      />,
    );

    const key = screen.getByRole("button", { name: "clear all" });
    expect(key).toHaveAttribute("aria-keyshortcuts", "Escape");
    expect(key).toHaveAttribute("title", "clears everything");
  });

  it("omits aria-keyshortcuts when the key has no keyboard equivalent", () => {
    render(<Key label="±" ariaLabel="toggle sign" onPress={vi.fn()} />);

    expect(screen.getByRole("button", { name: "toggle sign" })).not.toHaveAttribute(
      "aria-keyshortcuts",
    );
  });

  it("does not fire while disabled", async () => {
    const user = userEvent.setup();
    const onPress = vi.fn();
    render(<Key label="7" ariaLabel="7" onPress={onPress} disabled />);

    const key = screen.getByRole("button", { name: "7" });
    expect(key).toBeDisabled();

    await user.click(key);

    expect(onPress).not.toHaveBeenCalled();
  });
});

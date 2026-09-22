import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { OperationsBar } from "@/components/calculator/OperationsBar";
import { jsonBody, TEST_OPERATIONS } from "@/test/msw/handlers";
import { OPERATIONS_ENDPOINT } from "@/lib/query-config";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/utils";

function renderBar(busy = false) {
  const onUnary = vi.fn();
  const onOperator = vi.fn();
  renderWithProviders(<OperationsBar busy={busy} onUnary={onUnary} onOperator={onOperator} />);
  return { onUnary, onOperator };
}

describe("OperationsBar", () => {
  it("shows only the operations the keypad has no key for", () => {
    renderBar();

    const group = screen.getByRole("group", { name: "More operations" });
    expect(group).toHaveAttribute("data-ui", "calculator.operations");
    expect(screen.getByRole("button", { name: "square root" })).toHaveTextContent("√");
    expect(screen.getByRole("button", { name: "percent" })).toHaveTextContent("%");
    // `^` alone reads as a caret, so the power key shows what it means.
    expect(screen.getByRole("button", { name: "power" })).toHaveTextContent("xʸ");
    for (const name of ["add", "subtract", "multiply", "divide"]) {
      expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
    }
  });

  it("sends the unary keys to the store's unary action and the binary ones to the operator action", async () => {
    const user = userEvent.setup();
    const { onUnary, onOperator } = renderBar();

    await user.click(screen.getByRole("button", { name: "square root" }));
    await user.click(screen.getByRole("button", { name: "percent" }));
    await user.click(screen.getByRole("button", { name: "power" }));

    expect(onUnary).toHaveBeenNthCalledWith(1, "sqrt");
    expect(onUnary).toHaveBeenNthCalledWith(2, "percent");
    expect(onOperator).toHaveBeenCalledExactlyOnceWith("power");
  });

  it("disables every operation while a calculation is in flight", () => {
    renderBar(true);

    for (const name of ["square root", "percent", "power"]) {
      expect(screen.getByRole("button", { name })).toBeDisabled();
    }
  });

  it("still shows an operation this build has no action for, disabled and explained", async () => {
    // Discovery exists so a newer backend does not break this client: the key appears on its own,
    // but nothing here knows how to press it.
    server.use(
      jsonBody(
        { operations: [...TEST_OPERATIONS, { name: "cbrt", symbol: "∛", arity: 1 }] },
        OPERATIONS_ENDPOINT,
      ),
    );
    const { onUnary, onOperator } = renderBar();

    const key = await waitFor(() => screen.getByRole("button", { name: "cbrt" }));
    expect(key).toBeDisabled();
    expect(key).toHaveAttribute("title", "cbrt needs a newer version of this app");
    expect(onUnary).not.toHaveBeenCalled();
    expect(onOperator).not.toHaveBeenCalled();
  });
});

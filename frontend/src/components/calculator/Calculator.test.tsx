import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { Calculator } from "@/components/calculator/Calculator";
import { apiUrl } from "@/lib/api";
import { FALLBACK_ERROR_MESSAGE } from "@/lib/error-messages";
import { CALCULATE_ENDPOINT } from "@/lib/query-config";
import { useCalculatorStore } from "@/stores/useCalculatorStore";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/utils";

/** The store is a module singleton; every test starts from a bare zero with nothing wired in. */
beforeEach(() => {
  useCalculatorStore.getState().setEvaluator(null);
  useCalculatorStore.getState().clearAll();
});

/** The main display line, which is the `aria-live` region. */
function result(): HTMLElement {
  return screen.getByRole("status");
}

function key(name: string): HTMLElement {
  return screen.getByRole("button", { name });
}

describe("Calculator", () => {
  it("computes 12 + 7 = 19 through the API and announces the answer", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Calculator />);

    expect(screen.getByRole("region", { name: "Calculator" })).toHaveAttribute(
      "data-ui",
      "calculator",
    );

    await user.click(key("1"));
    await user.click(key("2"));
    expect(result()).toHaveTextContent("12");

    await user.click(key("add"));
    // The expression line keeps the part of the sum that is no longer on the main line.
    expect(screen.getByText("12 +")).toBeInTheDocument();

    await user.click(key("7"));
    await user.click(key("equals"));

    await waitFor(() => {
      expect(result()).toHaveTextContent("19");
    });
    expect(screen.getByText("12 + 7 =")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toBeEmptyDOMElement();
  });

  it("shows the mapped message for a division by zero and clears it on the next digit", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Calculator />);

    await user.click(key("1"));
    await user.click(key("2"));
    await user.click(key("divide"));
    await user.click(key("0"));
    await user.click(key("equals"));

    await waitFor(() => {
      // The sentence from error-messages.ts, never the server's `detail` (ADR-0005).
      expect(screen.getByRole("alert")).toHaveTextContent("Cannot divide by zero.");
    });

    await user.click(key("5"));

    expect(screen.getByRole("alert")).toBeEmptyDOMElement();
    expect(result()).toHaveTextContent("5");
  });

  it("locks every key but AC while a calculation is in flight, and AC gets out of it", async () => {
    const user = userEvent.setup();
    let release = (): void => undefined;
    const answered = new Promise<void>((resolve) => {
      release = resolve;
    });
    server.use(
      http.post(apiUrl(CALCULATE_ENDPOINT), async () => {
        await answered;
        return HttpResponse.json({ operation: "add", a: 1, b: 1, result: 2 });
      }),
    );
    renderWithProviders(<Calculator />);

    await user.click(key("1"));
    await user.click(key("add"));
    await user.click(key("1"));
    await user.click(key("equals"));

    await waitFor(() => {
      expect(key("7")).toBeDisabled();
    });
    expect(key("equals")).toBeDisabled();
    expect(key("square root")).toBeDisabled();
    expect(key("clear all")).toBeEnabled();
    expect(screen.getByText("…")).toBeInTheDocument();
    expect(result()).toHaveAttribute("aria-busy", "true");

    await user.click(key("clear all"));

    expect(result()).toHaveTextContent("0");
    expect(key("7")).toBeEnabled();
    expect(screen.queryByText("…")).not.toBeInTheDocument();
    release();
  });

  it("applies a unary operation from the operations bar", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Calculator />);

    await user.click(key("9"));
    await user.click(key("square root"));

    await waitFor(() => {
      expect(result()).toHaveTextContent("3");
    });
  });

  it("unwires the store when it unmounts", async () => {
    const user = userEvent.setup();
    const { unmount } = renderWithProviders(<Calculator />);
    await user.click(key("4"));
    await user.click(key("add"));

    unmount();
    useCalculatorStore.getState().evaluate();

    // With no evaluator left, the store fails the press instead of waiting for an answer for ever.
    await waitFor(() => {
      expect(useCalculatorStore.getState().calc.error).toBe(FALLBACK_ERROR_MESSAGE);
    });
    expect(useCalculatorStore.getState().pending).toBe(false);
  });
});

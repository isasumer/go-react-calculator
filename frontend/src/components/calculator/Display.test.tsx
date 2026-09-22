import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Display, entrySizeClass } from "@/components/calculator/Display";

describe("Display", () => {
  it("shows the expression above the value and keeps both in full in a tooltip", () => {
    render(<Display expression="12 +" value="1234567" error={null} busy={false} entering />);

    expect(screen.getByText("12 +")).toHaveAttribute("title", "12 +");

    const result = screen.getByRole("status");
    // While entering, the value is grouped (`formatEntry`), not display-formatted.
    expect(result).toHaveTextContent("1,234,567");
    // The line truncates visually, so the untruncated raw value has to live somewhere readable.
    expect(result).toHaveAttribute("title", "1234567");
  });

  it("formats a finished value with formatResult, keeping the raw value in title", () => {
    render(<Display expression="" value="0.30000000000000004" error={null} busy={false} />);

    const result = screen.getByRole("status");
    expect(result).toHaveTextContent("0.3");
    expect(result).toHaveAttribute("title", "0.30000000000000004");
  });

  it("reformats the leading operand of the expression line with formatResult", () => {
    render(
      <Display
        expression="0.30000000000000004 +"
        value="7"
        error={null}
        busy={false}
        entering={false}
      />,
    );

    expect(screen.getByText("0.3 +")).toBeInTheDocument();
  });

  it("leaves the expression alone if its leading token is not a number", () => {
    // Defensive: `expressionOf` never actually produces this, but a formatter that throws or
    // mangles unexpected input is worse than one that echoes it back untouched.
    render(<Display expression="not-a-number +" value="0" error={null} busy={false} />);

    expect(screen.getByText("not-a-number +")).toBeInTheDocument();
  });

  it("announces the value politely", () => {
    render(<Display expression="" value="19" error={null} busy={false} />);

    expect(screen.getByRole("status")).toHaveAttribute("aria-live", "polite");
  });

  it("renders the error slot as an alert and leaves it empty when there is no error", () => {
    const { rerender } = render(
      <Display expression="" value="12" error="Cannot divide by zero." busy={false} />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("Cannot divide by zero.");

    rerender(<Display expression="" value="5" error={null} busy={false} />);

    // The slot stays in the DOM: a live region has to exist before its text changes.
    expect(screen.getByRole("alert")).toBeEmptyDOMElement();
  });

  it("shows the request id as the alert's title when one is given", () => {
    render(
      <Display
        expression=""
        value="0"
        error="The calculator service had a problem. Please try again."
        busy={false}
        errorRequestId="4bf92f35-77b3-4da6-a3ce-929d0e0e4736"
      />,
    );

    expect(screen.getByRole("alert")).toHaveAttribute(
      "title",
      "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
    );
  });

  it("offers a Retry button only when canRetry is true, and fires onRetry when clicked", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    const { rerender } = render(
      <Display
        expression=""
        value="0"
        error="Cannot reach the calculator service. Check your connection."
        busy={false}
        canRetry
        onRetry={onRetry}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledTimes(1);

    rerender(
      <Display
        expression=""
        value="0"
        error="Cannot divide by zero."
        busy={false}
        canRetry={false}
        onRetry={onRetry}
      />,
    );
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("marks itself busy and shows a quiet placeholder while a calculation is in flight", () => {
    const { rerender } = render(<Display expression="12 +" value="7" error={null} busy />);

    // A busy live region stays quiet, so the value about to be replaced is not announced.
    expect(screen.getByRole("status")).toHaveAttribute("aria-busy", "true");
    const marker = screen.getByText("…");
    expect(marker).toHaveAttribute("aria-hidden", "true");

    rerender(<Display expression="12 +" value="7" error={null} busy={false} />);

    expect(screen.queryByText("…")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveAttribute("aria-busy", "false");
  });

  it("marks the shake attribute for a brief, rejected keystroke", () => {
    const { rerender } = render(<Display expression="" value="0" error={null} busy={false} />);

    expect(document.querySelector("[data-ui='calculator.display']")).not.toHaveAttribute(
      "data-shake",
    );

    rerender(<Display expression="" value="0" error={null} busy={false} shake />);

    expect(document.querySelector("[data-ui='calculator.display']")).toHaveAttribute(
      "data-shake",
      "true",
    );
  });
});

describe("entrySizeClass", () => {
  it.each([
    ["0", "text-4xl"],
    ["12345678", "text-4xl"],
    ["123456789", "text-3xl"],
    ["12345678901", "text-3xl"],
    ["123456789012", "text-2xl"],
    ["123456789012345", "text-2xl"],
    ["-1234567890123456", "text-xl"],
  ])("sizes %s as %s", (value, expected) => {
    expect(entrySizeClass(value)).toBe(expected);
  });
});

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Display, entrySizeClass } from "@/components/calculator/Display";

const LONG_VALUE = "1234567890123456";

describe("Display", () => {
  it("shows the expression above the value and keeps both in full in a tooltip", () => {
    render(<Display expression="12 +" value={LONG_VALUE} error={null} busy={false} />);

    expect(screen.getByText("12 +")).toHaveAttribute("title", "12 +");

    const result = screen.getByRole("status");
    expect(result).toHaveTextContent(LONG_VALUE);
    // The line truncates visually, so the untruncated value has to live somewhere readable.
    expect(result).toHaveAttribute("title", LONG_VALUE);
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

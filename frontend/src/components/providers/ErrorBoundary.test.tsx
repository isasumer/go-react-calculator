import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ErrorBoundary } from "@/components/providers/ErrorBoundary";

function Boom(): never {
  throw new Error("render failed");
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    // React logs every caught error itself; the boundary adds one more. Neither is a test failure.
    vi.spyOn(console, "error").mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders its children while nothing throws", () => {
    render(
      <ErrorBoundary>
        <p>calculator</p>
      </ErrorBoundary>,
    );

    expect(screen.getByText("calculator")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("replaces the tree with a fallback when a child throws", () => {
    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    );

    const fallback = screen.getByRole("alert");
    expect(fallback).toHaveAttribute("data-ui", "error-boundary");
    expect(screen.getByRole("heading", { level: 1, name: "Something went wrong" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Reload the page" })).toBeEnabled();
    expect(console.error).toHaveBeenCalled();
  });

  it("reloads the page from the fallback", async () => {
    const user = userEvent.setup();
    const reload = vi.fn();
    // jsdom's Location is unforgeable, so the whole object is stubbed rather than the method.
    vi.stubGlobal("location", { ...window.location, reload });

    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    );
    await user.click(screen.getByRole("button", { name: "Reload the page" }));

    expect(reload).toHaveBeenCalledTimes(1);
  });
});

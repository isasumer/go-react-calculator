import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { HistoryPanel } from "@/components/calculator/HistoryPanel";
import { useHistoryStore } from "@/stores/useHistoryStore";

beforeEach(() => {
  useHistoryStore.getState().clear();
});

describe("HistoryPanel", () => {
  it("carries data-ui and shows the count, empty by default", () => {
    render(<HistoryPanel onRecall={() => undefined} />);

    expect(screen.getByText("History (0)")).toBeInTheDocument();
    const panel = document.querySelector('[data-ui="calculator.history"]');
    expect(panel).toBeInTheDocument();
    expect(screen.getByText("No calculations yet.")).toBeInTheDocument();
  });

  it("lists entries newest first", () => {
    useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
    useHistoryStore.getState().add({ operation: "multiply", a: 4, b: 5, result: 20 });

    render(<HistoryPanel onRecall={() => undefined} />);

    const rows = screen
      .getAllByRole("button")
      .filter((button) => button.textContent?.includes("="));
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveTextContent("4 × 5 = 20");
    expect(rows[1]).toHaveTextContent("1 + 2 = 3");
  });

  it("renders a unary entry as `op a = result`", () => {
    useHistoryStore.getState().add({ operation: "sqrt", a: 9, b: null, result: 3 });

    render(<HistoryPanel onRecall={() => undefined} />);

    expect(screen.getByRole("button", { name: "√9 = 3" })).toBeInTheDocument();
  });

  it("recalls a row's result via user-event", async () => {
    const user = userEvent.setup();
    useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
    const onRecall = vi.fn();
    render(<HistoryPanel onRecall={onRecall} />);

    await user.click(screen.getByRole("button", { name: "1 + 2 = 3" }));

    expect(onRecall).toHaveBeenCalledExactlyOnceWith(3);
  });

  it("removes a row via its remove button", async () => {
    const user = userEvent.setup();
    useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
    render(<HistoryPanel onRecall={() => undefined} />);

    await user.click(screen.getByRole("button", { name: "Remove 1 + 2 = 3 from history" }));

    expect(screen.queryByRole("button", { name: "1 + 2 = 3" })).not.toBeInTheDocument();
    expect(useHistoryStore.getState().entries).toHaveLength(0);
  });

  describe("clear history (two-step confirm)", () => {
    it("requires a second click to actually clear", async () => {
      const user = userEvent.setup();
      useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
      render(<HistoryPanel onRecall={() => undefined} />);

      await user.click(screen.getByRole("button", { name: "Clear history" }));
      expect(screen.getByRole("button", { name: "Confirm clear" })).toBeInTheDocument();
      expect(useHistoryStore.getState().entries).toHaveLength(1);

      await user.click(screen.getByRole("button", { name: "Confirm clear" }));
      expect(useHistoryStore.getState().entries).toHaveLength(0);
    });

    it("Escape cancels back to the first state without clearing", async () => {
      const user = userEvent.setup();
      useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
      render(<HistoryPanel onRecall={() => undefined} />);

      await user.click(screen.getByRole("button", { name: "Clear history" }));
      await user.keyboard("{Escape}");

      expect(screen.getByRole("button", { name: "Clear history" })).toBeInTheDocument();
      expect(useHistoryStore.getState().entries).toHaveLength(1);
    });

    it("losing focus cancels back to the first state without clearing", async () => {
      const user = userEvent.setup();
      useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
      render(
        <div>
          <HistoryPanel onRecall={() => undefined} />
          <button type="button">elsewhere</button>
        </div>,
      );

      await user.click(screen.getByRole("button", { name: "Clear history" }));
      expect(screen.getByRole("button", { name: "Confirm clear" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "elsewhere" }));

      expect(screen.getByRole("button", { name: "Clear history" })).toBeInTheDocument();
      expect(useHistoryStore.getState().entries).toHaveLength(1);
    });
  });

  it("persists the open state to localStorage and restores it on remount", async () => {
    const user = userEvent.setup();
    useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
    const { unmount } = render(<HistoryPanel onRecall={() => undefined} />);

    await user.click(screen.getByText("History (1)"));

    await waitFor(() => {
      expect(localStorage.getItem("calc.history-open.v1")).toBe("false");
    });

    unmount();
    render(<HistoryPanel onRecall={() => undefined} />);

    const panel = document.querySelector('[data-ui="calculator.history"]') as HTMLDetailsElement;
    expect(panel.open).toBe(false);
  });
});

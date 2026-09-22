/**
 * Automated accessibility checks. axe catches the mechanical mistakes — an unlabelled control, a
 * broken ARIA attribute, a live region with no accessible text — which is exactly the class of
 * defect a component test written with roles and labels cannot see, because such a test passes by
 * asserting the same names the markup got wrong.
 *
 * It is not a substitute for the manual pass noted in the PR: axe finds roughly a third of real
 * barriers, and never the ones about whether the wording makes sense.
 */
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxeResults } from "axe-core";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { App } from "@/app/App";
import { renderWithProviders } from "@/test/utils";

/** Violation ids plus their help text: a bare `expect(violations).toEqual([])` prints nothing useful. */
function summarise(results: AxeResults): readonly string[] {
  return results.violations.map(
    (violation) => `${violation.id}: ${violation.help} (${String(violation.nodes.length)} nodes)`,
  );
}

describe("accessibility", () => {
  it("has no violations on the initial render", async () => {
    const { container } = renderWithProviders(<App />);

    expect(summarise(await axe(container))).toEqual([]);
  }, 20_000);

  it("has no violations while an error is on the display", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<App />);

    await user.click(screen.getByRole("button", { name: "1" }));
    await user.click(screen.getByRole("button", { name: "divide" }));
    await user.click(screen.getByRole("button", { name: "0" }));
    await user.click(screen.getByRole("button", { name: "equals" }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("Cannot divide by zero.");
    });

    expect(summarise(await axe(container))).toEqual([]);
  }, 20_000);
});

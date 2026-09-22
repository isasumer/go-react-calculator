import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { App } from "@/app/App";

describe("App", () => {
  it("renders the app shell", () => {
    render(<App />);

    expect(screen.getByRole("main")).toHaveAttribute("data-ui", "app-shell");
    expect(screen.getByRole("heading", { level: 1, name: "Calculator" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Coming in Sprint 2" })).toBeDisabled();
  });
});

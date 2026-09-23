import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { App, appVersion, REPOSITORY_URL } from "@/app/App";
import { renderWithProviders } from "@/test/utils";

describe("App", () => {
  it("renders the shell, the version badge and the calculator", () => {
    renderWithProviders(<App />);

    const banner = screen.getByRole("banner");
    expect(screen.getByRole("heading", { level: 1, name: "Calculator" })).toBeInTheDocument();
    // The badge shows "dev" and reads "Version dev": a bare version number is not self-describing.
    expect(within(banner).getByText("dev")).toHaveTextContent("Version dev");
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Calculator" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "equals" })).toBeEnabled();
  });

  it("links to the source code in a new tab", () => {
    renderWithProviders(<App />);

    const link = within(screen.getByRole("banner")).getByRole("link", {
      name: "Source code on GitHub (opens in a new tab)",
    });
    expect(link).toHaveAttribute("href", REPOSITORY_URL);
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("shows the build the bundle was made with", () => {
    vi.stubEnv("VITE_APP_VERSION", "1.4.2");

    renderWithProviders(<App />);

    expect(within(screen.getByRole("banner")).getByText("1.4.2")).toBeInTheDocument();
  });
});

describe("appVersion", () => {
  it.each([
    [{}, "dev"],
    [{ VITE_APP_VERSION: "" }, "dev"],
    [{ VITE_APP_VERSION: "   " }, "dev"],
    [{ VITE_APP_VERSION: 42 }, "dev"],
    [{ VITE_APP_VERSION: " 1.4.2 " }, "1.4.2"],
    [{ VITE_APP_VERSION: "2026.09.22-abc1234" }, "2026.09.22-abc1234"],
  ])("reads %o as %s", (env: Readonly<Record<string, unknown>>, expected: string) => {
    expect(appVersion(env)).toBe(expected);
  });
});

import { useQueryClient } from "@tanstack/react-query";
import { render, renderHook, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createQueryClient, QueryProvider } from "@/components/providers/QueryProvider";
import { MUTATION_RETRY, QUERY_RETRY, STALE_TIME } from "@/lib/query-config";

describe("createQueryClient", () => {
  it("applies the app's defaults", () => {
    const defaults = createQueryClient().getDefaultOptions();

    expect(defaults.queries?.retry).toBe(QUERY_RETRY);
    expect(defaults.queries?.refetchOnWindowFocus).toBe(false);
    expect(defaults.queries?.staleTime).toBe(STALE_TIME.default);
    expect(defaults.mutations?.retry).toBe(MUTATION_RETRY);
  });
});

describe("QueryProvider", () => {
  it("renders its children", () => {
    render(
      <QueryProvider>
        <p>child</p>
      </QueryProvider>,
    );

    expect(screen.getByText("child")).toBeInTheDocument();
  });

  it("provides one client that survives re-renders", () => {
    const { result, rerender } = renderHook(() => useQueryClient(), { wrapper: QueryProvider });
    const first = result.current;

    rerender();

    expect(result.current).toBe(first);
    expect(first.getDefaultOptions().queries?.retry).toBe(QUERY_RETRY);
  });
});

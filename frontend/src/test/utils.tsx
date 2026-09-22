import { QueryClientProvider, type QueryClient } from "@tanstack/react-query";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";

import { createQueryClient } from "@/components/providers/QueryProvider";

/**
 * The app's QueryClient, with retries off by default and no backoff for the hooks that ask for a
 * retry explicitly. Tests assert the mapped error, not the retry policy, and a one-second backoff
 * would be a second of waiting in every error test.
 */
export function createTestQueryClient(): QueryClient {
  const client = createQueryClient();
  const defaults = client.getDefaultOptions();
  client.setDefaultOptions({
    queries: { ...defaults.queries, retry: false, retryDelay: 0 },
    mutations: { ...defaults.mutations, retry: false, retryDelay: 0 },
  });
  return client;
}

/** Wrapper for `renderHook`. Pass a client to assert on the cache afterwards. */
export function createWrapper(client: QueryClient = createTestQueryClient()) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

export interface RenderWithProvidersResult extends RenderResult {
  readonly queryClient: QueryClient;
}

/** `render` with the app's providers around it. */
export function renderWithProviders(
  ui: ReactElement,
  options: Omit<RenderOptions, "wrapper"> & { queryClient?: QueryClient } = {},
): RenderWithProvidersResult {
  const { queryClient = createTestQueryClient(), ...renderOptions } = options;
  return {
    ...render(ui, { wrapper: createWrapper(queryClient), ...renderOptions }),
    queryClient,
  };
}

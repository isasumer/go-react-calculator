import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useState } from "react";

import { MUTATION_RETRY, QUERY_RETRY, STALE_TIME } from "@/lib/query-config";

/**
 * The app's QueryClient defaults, in one place so tests and the app cannot drift.
 *
 * - queries retry once: a calculator is a foreground tool, so one silent retry is helpful and a
 *   second is just latency the user waits through.
 * - no refetch on window focus: results never change server-side, and a refetch on every tab
 *   switch would be pure noise.
 * - mutations never retry: the user pressed `=`; if it failed they press it again, and retrying a
 *   request the server may already have processed is the caller's decision, not the library's.
 */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: QUERY_RETRY,
        refetchOnWindowFocus: false,
        staleTime: STALE_TIME.default,
      },
      mutations: {
        retry: MUTATION_RETRY,
      },
    },
  });
}

export interface QueryProviderProps {
  readonly children: ReactNode;
}

/** Provides the app's single QueryClient. Created once per mount, never on re-render. */
export function QueryProvider({ children }: QueryProviderProps) {
  const [client] = useState(createQueryClient);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

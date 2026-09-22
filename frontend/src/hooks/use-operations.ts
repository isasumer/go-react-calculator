import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { apiFetch, type ApiError } from "@/lib/api";
import { calculatorKeys, OPERATIONS_ENDPOINT, QUERY_RETRY, STALE_TIME } from "@/lib/query-config";
import {
  operationsResponseSchema,
  type OperationsResponse,
  type OperationSpec,
} from "@/types/calculator";

/**
 * The registry as it is compiled into the backend today. The keypad is labelled from this until the
 * real list arrives, and stays on it if the request fails, so the UI never blocks on the network for
 * something it already knows. It is the same seven operations `backend/internal/calc` registers.
 */
export const FALLBACK_OPERATIONS: readonly OperationSpec[] = Object.freeze([
  { name: "add", symbol: "+", arity: 2 },
  { name: "subtract", symbol: "−", arity: 2 },
  { name: "multiply", symbol: "×", arity: 2 },
  { name: "divide", symbol: "÷", arity: 2 },
  { name: "power", symbol: "^", arity: 2 },
  { name: "sqrt", symbol: "√", arity: 1 },
  { name: "percent", symbol: "%", arity: 2 },
] as const satisfies readonly OperationSpec[]);

const FALLBACK_RESPONSE: OperationsResponse = { operations: [...FALLBACK_OPERATIONS] };

export type UseOperationsResult = UseQueryResult<OperationsResponse, ApiError> & {
  /** The operations to render: the server's list, or {@link FALLBACK_OPERATIONS}. */
  readonly operations: readonly OperationSpec[];
  /** True while `operations` is the built-in list rather than the server's answer. */
  readonly isFallback: boolean;
};

/**
 * The operation registry. Discovery only ever changes on a backend deploy, so it is fetched once per
 * session (`staleTime: Infinity`) and never refetched on focus.
 */
export function useOperations(): UseOperationsResult {
  const query = useQuery<OperationsResponse, ApiError>({
    queryKey: calculatorKeys.operations(),
    queryFn: ({ signal }) =>
      apiFetch(OPERATIONS_ENDPOINT, { signal, schema: operationsResponseSchema }),
    staleTime: STALE_TIME.operations,
    retry: QUERY_RETRY,
    placeholderData: FALLBACK_RESPONSE,
  });

  return {
    ...query,
    operations: query.data?.operations ?? FALLBACK_OPERATIONS,
    isFallback: query.data === undefined || query.isPlaceholderData,
  };
}

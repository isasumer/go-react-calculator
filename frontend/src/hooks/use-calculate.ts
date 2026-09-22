import { useMutation, type UseMutationResult } from "@tanstack/react-query";

import { apiFetch, type ApiError } from "@/lib/api";
import { CALCULATE_ENDPOINT } from "@/lib/query-config";
import {
  calculateResponseSchema,
  type CalculateRequest,
  type CalculateResponse,
} from "@/types/calculator";

export type UseCalculateResult = UseMutationResult<CalculateResponse, ApiError, CalculateRequest>;

/**
 * Evaluate one operation on the server. A mutation rather than a query: pressing `=` is an action
 * with a result the user asked for now, not cacheable state, and it must never be retried or
 * replayed on its own (see `createQueryClient`).
 *
 * `error` is always an {@link ApiError}; map it to a sentence with `messageForError`.
 */
export function useCalculate(): UseCalculateResult {
  return useMutation<CalculateResponse, ApiError, CalculateRequest>({
    mutationFn: (request) =>
      apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: request,
        schema: calculateResponseSchema,
      }),
  });
}

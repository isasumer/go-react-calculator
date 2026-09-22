import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useCalculate } from "@/hooks/use-calculate";
import { isApiError } from "@/lib/api";
import { messageForError } from "@/lib/error-messages";
import { jsonBody, networkError, TEST_REQUEST_ID } from "@/test/msw/handlers";
import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/utils";

describe("useCalculate", () => {
  it("returns the result of a binary operation", async () => {
    const { result } = renderHook(() => useCalculate(), { wrapper: createWrapper() });

    result.current.mutate({ operation: "divide", a: 10, b: 4 });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
    expect(result.current.data).toEqual({ operation: "divide", a: 10, b: 4, result: 2.5 });
    expect(result.current.error).toBeNull();
  });

  it("omits b for a unary operation", async () => {
    const { result } = renderHook(() => useCalculate(), { wrapper: createWrapper() });

    result.current.mutate({ operation: "sqrt", a: 16 });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
    expect(result.current.data).toEqual({ operation: "sqrt", a: 16, result: 4 });
  });

  it("surfaces a domain failure as a typed ApiError", async () => {
    const { result } = renderHook(() => useCalculate(), { wrapper: createWrapper() });

    result.current.mutate({ operation: "divide", a: 1, b: 0 });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    const { error } = result.current;
    expect(isApiError(error)).toBe(true);
    expect(error?.code).toBe("DIVISION_BY_ZERO");
    expect(error?.status).toBe(422);
    expect(error?.requestId).toBe(TEST_REQUEST_ID);
    expect(error?.errors).toEqual([{ field: "b", message: "must be non-zero" }]);
    expect(messageForError(error)).toBe("Cannot divide by zero.");
  });

  it("surfaces a transport failure as NETWORK", async () => {
    server.use(networkError());
    const { result } = renderHook(() => useCalculate(), { wrapper: createWrapper() });

    result.current.mutate({ operation: "add", a: 1, b: 2 });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    expect(result.current.error?.code).toBe("NETWORK");
    expect(messageForError(result.current.error)).toBe(
      "Cannot reach the calculator service. Check your connection.",
    );
  });

  it("fails closed when the server answers with the wrong shape", async () => {
    server.use(jsonBody({ operation: "add", a: 1, b: 2, result: null }));
    const { result } = renderHook(() => useCalculate(), { wrapper: createWrapper() });

    result.current.mutate({ operation: "add", a: 1, b: 2 });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    expect(result.current.error?.code).toBe("INVALID_RESPONSE");
    expect(result.current.data).toBeUndefined();
  });

  it("does not retry a failed calculation", async () => {
    let attempts = 0;
    server.use(networkError());
    const wrapper = createWrapper();
    const { result } = renderHook(() => useCalculate(), { wrapper });

    server.events.on("request:start", () => {
      attempts += 1;
    });
    result.current.mutate({ operation: "add", a: 1, b: 2 });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    server.events.removeAllListeners("request:start");

    expect(attempts).toBe(1);
  });
});

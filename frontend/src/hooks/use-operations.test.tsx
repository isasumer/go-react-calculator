import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FALLBACK_OPERATIONS, useOperations } from "@/hooks/use-operations";
import { OPERATION_NAMES } from "@/types/calculator";
import { jsonBody, networkError, TEST_OPERATIONS } from "@/test/msw/handlers";
import { server } from "@/test/msw/server";
import { createTestQueryClient, createWrapper } from "@/test/utils";

describe("FALLBACK_OPERATIONS", () => {
  it("is the backend registry, in order", () => {
    expect(FALLBACK_OPERATIONS.map((op) => op.name)).toEqual([...OPERATION_NAMES]);
    expect(FALLBACK_OPERATIONS).toEqual(TEST_OPERATIONS);
  });

  it("marks sqrt as the only unary operation", () => {
    expect(FALLBACK_OPERATIONS.filter((op) => op.arity === 1).map((op) => op.name)).toEqual([
      "sqrt",
    ]);
  });
});

describe("useOperations", () => {
  it("renders the fallback list before the network answers", () => {
    const { result } = renderHook(() => useOperations(), { wrapper: createWrapper() });

    expect(result.current.operations).toEqual(FALLBACK_OPERATIONS);
    expect(result.current.isFallback).toBe(true);
  });

  it("replaces the fallback with the server's registry", async () => {
    const { result } = renderHook(() => useOperations(), { wrapper: createWrapper() });

    // placeholderData reports `success` from the first render, so the fetch is what we wait for.
    await waitFor(() => {
      expect(result.current.isFallback).toBe(false);
    });
    expect(result.current.isSuccess).toBe(true);
    expect(result.current.operations).toEqual(TEST_OPERATIONS);
  });

  it("stays on the fallback when the network fails", async () => {
    server.use(networkError(), networkError("/v1/operations"));
    const { result } = renderHook(() => useOperations(), { wrapper: createWrapper() });

    await waitFor(
      () => {
        expect(result.current.isError).toBe(true);
      },
      { timeout: 3000 },
    );
    expect(result.current.error?.code).toBe("NETWORK");
    expect(result.current.operations).toEqual(FALLBACK_OPERATIONS);
    expect(result.current.isFallback).toBe(true);
  });

  it("stays on the fallback when the registry does not match the schema", async () => {
    server.use(
      jsonBody({ operations: [{ name: "add", symbol: "+", arity: 3 }] }, "/v1/operations"),
    );
    const { result } = renderHook(() => useOperations(), { wrapper: createWrapper() });

    await waitFor(
      () => {
        expect(result.current.isError).toBe(true);
      },
      { timeout: 3000 },
    );
    expect(result.current.error?.code).toBe("INVALID_RESPONSE");
    expect(result.current.operations).toEqual(FALLBACK_OPERATIONS);
    expect(result.current.isFallback).toBe(true);
  });

  it("accepts an operation the client does not know yet", async () => {
    server.use(
      jsonBody(
        { operations: [...TEST_OPERATIONS, { name: "modulo", symbol: "mod", arity: 2 }] },
        "/v1/operations",
      ),
    );
    const { result } = renderHook(() => useOperations(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isFallback).toBe(false);
    });
    expect(result.current.operations).toHaveLength(8);
  });

  it("fetches once per session and serves the cache afterwards", async () => {
    let requests = 0;
    server.events.on("request:start", () => {
      requests += 1;
    });
    const client = createTestQueryClient();
    const wrapper = createWrapper(client);

    const first = renderHook(() => useOperations(), { wrapper });
    await waitFor(() => {
      expect(first.result.current.isFallback).toBe(false);
    });
    first.unmount();

    const second = renderHook(() => useOperations(), { wrapper });
    await waitFor(() => {
      expect(second.result.current.isFallback).toBe(false);
    });
    server.events.removeAllListeners("request:start");

    expect(requests).toBe(1);
    expect(second.result.current.isFallback).toBe(false);
  });
});

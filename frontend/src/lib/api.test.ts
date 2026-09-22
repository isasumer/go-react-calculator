import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { apiFetch, ApiError, apiUrl, DEFAULT_TIMEOUT_MS, isApiError, joinApiPath } from "@/lib/api";
import { CALCULATE_ENDPOINT, OPERATIONS_ENDPOINT } from "@/lib/query-config";
import {
  jsonBody,
  malformedJson,
  networkError,
  problem,
  rateLimited,
  serverErrorText,
  slowResponse,
  TEST_OPERATIONS,
  TEST_REQUEST_ID,
} from "@/test/msw/handlers";
import { server } from "@/test/msw/server";
import {
  calculateResponseSchema,
  operationsResponseSchema,
  type CalculateRequest,
} from "@/types/calculator";

/** Reject if the call resolves; return the ApiError otherwise. */
async function expectApiError(promise: Promise<unknown>): Promise<ApiError> {
  try {
    await promise;
  } catch (e) {
    expect(isApiError(e)).toBe(true);
    return e as ApiError;
  }
  throw new Error("expected apiFetch to reject");
}

const divide: CalculateRequest = { operation: "divide", a: 10, b: 4 };

describe("joinApiPath", () => {
  it.each([
    ["/api", "/v1/calculate", "/api/v1/calculate"],
    ["/api", "v1/calculate", "/api/v1/calculate"],
    ["/", "/v1/calculate", "/v1/calculate"],
    ["https://api.example.com", "/v1/calculate", "https://api.example.com/v1/calculate"],
  ])("joins %s with %s", (base, path, expected) => {
    expect(joinApiPath(base, path)).toBe(expected);
  });
});

describe("apiUrl", () => {
  it("resolves the configured base against the document origin", () => {
    expect(apiUrl(CALCULATE_ENDPOINT)).toBe(`${globalThis.location.origin}/api/v1/calculate`);
  });
});

describe("apiFetch", () => {
  it("returns the validated body of a successful POST", async () => {
    const result = await apiFetch(CALCULATE_ENDPOINT, {
      method: "POST",
      body: divide,
      schema: calculateResponseSchema,
    });

    expect(result).toEqual({ operation: "divide", a: 10, b: 4, result: 2.5 });
  });

  it("defaults to GET with no body and returns the operation registry", async () => {
    const result = await apiFetch(OPERATIONS_ENDPOINT, { schema: operationsResponseSchema });

    expect(result.operations).toEqual(TEST_OPERATIONS);
  });

  it("sends JSON headers and serialises the body", async () => {
    let seen: Request | undefined;
    server.use(
      http.post(apiUrl(CALCULATE_ENDPOINT), ({ request }) => {
        seen = request.clone();
        return HttpResponse.json({ operation: "sqrt", a: 16, result: 4 });
      }),
    );

    await apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: { operation: "sqrt", a: 16 } });

    expect(seen?.method).toBe("POST");
    expect(seen?.headers.get("Content-Type")).toBe("application/json");
    expect(seen?.headers.get("Accept")).toContain("application/problem+json");
    await expect(seen?.text()).resolves.toBe('{"operation":"sqrt","a":16}');
  });

  it("defaults to POST when a body is given", async () => {
    let seen: Request | undefined;
    server.use(
      http.post(apiUrl(CALCULATE_ENDPOINT), ({ request }) => {
        seen = request.clone();
        return HttpResponse.json({ operation: "add", a: 1, b: 2, result: 3 });
      }),
    );

    await apiFetch(CALCULATE_ENDPOINT, { body: { operation: "add", a: 1, b: 2 } });

    expect(seen?.method).toBe("POST");
  });

  it("returns the raw parsed body when no schema is given", async () => {
    const result = await apiFetch<unknown>(CALCULATE_ENDPOINT, { method: "POST", body: divide });

    expect(result).toEqual({ operation: "divide", a: 10, b: 4, result: 2.5 });
  });

  it("maps a 422 problem document onto ApiError, errors[] included", async () => {
    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: { operation: "modulo", a: 1, b: 2 },
        schema: calculateResponseSchema,
      }),
    );

    expect(error).toBeInstanceOf(ApiError);
    expect(error.name).toBe("ApiError");
    expect(error.status).toBe(422);
    expect(error.code).toBe("UNSUPPORTED_OPERATION");
    expect(error.title).toBe("Unsupported operation");
    expect(error.detail).toContain('operation "modulo" is not supported');
    expect(error.requestId).toBe(TEST_REQUEST_ID);
    expect(error.errors).toEqual([
      {
        field: "operation",
        message: "must be one of: add, subtract, multiply, divide, power, sqrt, percent",
      },
    ]);
    expect(error.message).toBe(error.detail);
    expect(error.retryAfterSeconds).toBeUndefined();
  });

  it("maps a 400 VALIDATION_FAILED problem, one entry per offending field", async () => {
    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: { operation: "add", a: 1 } }),
    );

    expect(error.status).toBe(400);
    expect(error.code).toBe("VALIDATION_FAILED");
    expect(error.errors).toEqual([{ field: "b", message: 'is required for operation "add"' }]);
  });

  it("maps a 400 INVALID_BODY problem when a field has the wrong JSON type", async () => {
    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: { operation: "add", a: "1", b: 2 } }),
    );

    expect(error.code).toBe("INVALID_BODY");
    expect(error.detail).toBe('field "a" must be a number (byte 26)');
    expect(error.errors).toEqual([{ field: "a", message: "must be a number" }]);
  });

  it("reads Retry-After seconds from a 429", async () => {
    server.use(rateLimited(30));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.status).toBe(429);
    expect(error.code).toBe("RATE_LIMITED");
    expect(error.retryAfterSeconds).toBe(30);
  });

  it("reads Retry-After given as an HTTP-date", async () => {
    server.use(
      http.post(apiUrl(CALCULATE_ENDPOINT), () =>
        HttpResponse.json(problem("RATE_LIMITED", "slow down"), {
          status: 429,
          headers: {
            "Content-Type": "application/problem+json",
            "Retry-After": new Date(Date.now() + 45_000).toUTCString(),
          },
        }),
      ),
    );

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.retryAfterSeconds).toBeGreaterThan(40);
    expect(error.retryAfterSeconds).toBeLessThanOrEqual(45);
  });

  it("ignores an unparsable Retry-After", async () => {
    server.use(
      http.post(apiUrl(CALCULATE_ENDPOINT), () =>
        HttpResponse.json(problem("RATE_LIMITED", "slow down"), {
          status: 429,
          headers: { "Content-Type": "application/problem+json", "Retry-After": "soonish" },
        }),
      ),
    );

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.retryAfterSeconds).toBeUndefined();
  });

  it("maps a failed request onto NETWORK with status 0", async () => {
    server.use(networkError());

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.code).toBe("NETWORK");
    expect(error.status).toBe(0);
    expect(error.cause).toBeDefined();
  });

  it("maps a caller-cancelled request onto NETWORK", async () => {
    const controller = new AbortController();
    controller.abort();

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide, signal: controller.signal }),
    );

    expect(error.code).toBe("NETWORK");
  });

  it("maps an unanswered request onto TIMEOUT", async () => {
    server.use(slowResponse(300));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide, timeoutMs: 20 }),
    );

    expect(error.code).toBe("TIMEOUT");
    expect(error.status).toBe(0);
    expect(error.detail).toContain("20 ms");
  });

  it("merges the caller's signal without AbortSignal.any where the runtime lacks it", async () => {
    const ctor: { any?: (signals: AbortSignal[]) => AbortSignal } = AbortSignal;
    const original = Object.getOwnPropertyDescriptor(AbortSignal, "any");
    delete ctor.any;
    try {
      const controller = new AbortController();
      const result = await apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: divide,
        signal: controller.signal,
        schema: calculateResponseSchema,
      });
      expect(result.result).toBe(2.5);

      server.use(slowResponse(300));
      const error = await expectApiError(
        apiFetch(CALCULATE_ENDPOINT, {
          method: "POST",
          body: divide,
          signal: controller.signal,
          timeoutMs: 20,
        }),
      );
      expect(error.code).toBe("TIMEOUT");

      controller.abort();
      const aborted = await expectApiError(
        apiFetch(CALCULATE_ENDPOINT, {
          method: "POST",
          body: divide,
          signal: controller.signal,
        }),
      );
      expect(aborted.code).toBe("NETWORK");
    } finally {
      if (original !== undefined) {
        Object.defineProperty(AbortSignal, "any", original);
      }
    }
  });

  it("maps a body that is not JSON onto UNEXPECTED", async () => {
    server.use(malformedJson());

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.code).toBe("UNEXPECTED");
    expect(error.status).toBe(200);
    expect(error.detail).toContain('{"result":');
  });

  it("maps a plain-text 5xx onto UNEXPECTED, quoting the body", async () => {
    server.use(serverErrorText("502 Bad Gateway"));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.code).toBe("UNEXPECTED");
    expect(error.status).toBe(502);
    expect(error.detail).toBe("the server returned 502: 502 Bad Gateway");
  });

  it("truncates a long unexpected body", async () => {
    server.use(serverErrorText("x".repeat(500)));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.detail).toHaveLength("the server returned 502: ".length + 201);
    expect(error.detail.endsWith("…")).toBe(true);
  });

  it("reports an empty error body", async () => {
    server.use(serverErrorText("", 503));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.detail).toBe("the server returned 503 with an empty body");
  });

  it("fills the gaps in a problem body that carries only a code", async () => {
    server.use(
      jsonBody({ code: "GATEWAY_REFUSED" }, CALCULATE_ENDPOINT, {
        status: 503,
        statusText: "Service Unavailable",
      }),
    );

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.code).toBe("GATEWAY_REFUSED");
    expect(error.status).toBe(503);
    expect(error.title).toBe("Service Unavailable");
    expect(error.detail).toBe("");
    expect(error.errors).toEqual([]);
  });

  it("uses the code as the title when there is no status text either", async () => {
    // A non-standard status has no reason phrase, so `statusText` really is empty.
    server.use(jsonBody({ code: "GATEWAY_REFUSED" }, CALCULATE_ENDPOINT, { status: 599 }));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.title).toBe("GATEWAY_REFUSED");
    expect(error.message).toBe("GATEWAY_REFUSED");
  });

  it("falls back to UNEXPECTED for a JSON error body with no code", async () => {
    server.use(jsonBody({ message: "nope" }, CALCULATE_ENDPOINT, { status: 500 }));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: divide }),
    );

    expect(error.code).toBe("UNEXPECTED");
    expect(error.status).toBe(500);
  });

  it("fails closed with INVALID_RESPONSE when the body does not match the schema", async () => {
    server.use(jsonBody({ operation: "divide", a: 10, b: 4, result: "2.5" }));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: divide,
        schema: calculateResponseSchema,
      }),
    );

    expect(error.code).toBe("INVALID_RESPONSE");
    expect(error.status).toBe(200);
    expect(error.detail).toContain("result");
    expect(error.cause).toBeDefined();
  });

  it("rejects an operation outside the registry echoed back by the server", async () => {
    server.use(jsonBody({ operation: "modulo", a: 1, b: 2, result: 1 }));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: divide,
        schema: calculateResponseSchema,
      }),
    );

    expect(error.code).toBe("INVALID_RESPONSE");
    expect(error.detail).toContain("operation");
  });

  it("names the root when the whole body is the wrong type", async () => {
    server.use(jsonBody(["not", "an", "object"]));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: divide,
        schema: calculateResponseSchema,
      }),
    );

    expect(error.code).toBe("INVALID_RESPONSE");
    expect(error.detail).toContain("(root)");
  });

  it("takes the request id from the response header when the body has none", async () => {
    const { requestId, ...withoutRequestId } = problem("DOMAIN_ERROR", "no real result");
    expect(requestId).toBe(TEST_REQUEST_ID);
    server.use(
      http.post(apiUrl(CALCULATE_ENDPOINT), () =>
        HttpResponse.json(withoutRequestId, {
          status: 422,
          headers: {
            "Content-Type": "application/problem+json",
            "X-Request-ID": "header-only-id",
          },
        }),
      ),
    );

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, { method: "POST", body: { operation: "sqrt", a: -1 } }),
    );

    expect(error.requestId).toBe("header-only-id");
  });

  it("carries the request id onto an INVALID_RESPONSE", async () => {
    server.use(jsonBody({ nope: true }));

    const error = await expectApiError(
      apiFetch(CALCULATE_ENDPOINT, {
        method: "POST",
        body: divide,
        schema: calculateResponseSchema,
      }),
    );

    expect(error.code).toBe("INVALID_RESPONSE");
    expect(error.requestId).toBeUndefined();
  });

  it("times out after 10 seconds by default", () => {
    expect(DEFAULT_TIMEOUT_MS).toBe(10_000);
  });
});

describe("the MSW double and the frozen contract", () => {
  it.each([
    [{ operation: "add", a: 1, b: 2 } as const, 3],
    [{ operation: "subtract", a: 1, b: 2 } as const, -1],
    [{ operation: "multiply", a: 3, b: 4 } as const, 12],
    [{ operation: "divide", a: 10, b: 4 } as const, 2.5],
    [{ operation: "power", a: 2, b: 10 } as const, 1024],
    [{ operation: "sqrt", a: 16 } as const, 4],
    [{ operation: "percent", a: 200, b: 15 } as const, 30],
  ])("evaluates %j", async (body, expected) => {
    const result = await apiFetch(CALCULATE_ENDPOINT, {
      method: "POST",
      body,
      schema: calculateResponseSchema,
    });

    expect(result.result).toBe(expected);
  });

  it.each([
    [{ operation: "divide", a: 1, b: 0 }, "DIVISION_BY_ZERO", 422],
    [{ operation: "power", a: 0, b: -1 }, "DIVISION_BY_ZERO", 422],
    [{ operation: "sqrt", a: -1 }, "DOMAIN_ERROR", 422],
    [{ operation: "power", a: -8, b: 0.5 }, "DOMAIN_ERROR", 422],
    [{ operation: "power", a: 1e308, b: 2 }, "RESULT_NOT_FINITE", 422],
    [{ operation: "modulo", a: 1, b: 2 }, "UNSUPPORTED_OPERATION", 422],
    [{ operation: "sqrt", a: 4, b: 2 }, "UNEXPECTED_OPERAND", 422],
    [{ operation: "add", a: 1 }, "VALIDATION_FAILED", 400],
    [{ a: 1, b: 2 }, "VALIDATION_FAILED", 400],
    [{ operation: "add", a: 1, b: 2, c: 3 }, "INVALID_BODY", 400],
  ])("rejects %j with %s", async (body, code, status) => {
    const error = await expectApiError(apiFetch(CALCULATE_ENDPOINT, { method: "POST", body }));

    expect(error.code).toBe(code);
    expect(error.status).toBe(status);
  });

  it("serves the division-by-zero problem exactly as docs/errors.md documents it", async () => {
    const response = await fetch(apiUrl(CALCULATE_ENDPOINT), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ operation: "divide", a: 1, b: 0 }),
    });

    expect(response.status).toBe(422);
    expect(response.headers.get("Content-Type")).toContain("application/problem+json");
    await expect(response.json()).resolves.toEqual({
      type: "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#division_by_zero",
      title: "Division by zero",
      status: 422,
      detail: 'b must be non-zero for operation "divide"',
      code: "DIVISION_BY_ZERO",
      instance: "/api/v1/calculate",
      requestId: TEST_REQUEST_ID,
      errors: [{ field: "b", message: "must be non-zero" }],
    });
  });
});

describe("isApiError", () => {
  it.each([
    [new ApiError({ status: 0, code: "NETWORK", title: "t", detail: "d" }), true],
    [new Error("boom"), false],
    ["NETWORK", false],
    [null, false],
    [undefined, false],
  ])("narrows %s", (value, expected) => {
    expect(isApiError(value)).toBe(expected);
  });

  it("falls back to the title and then the code for its message", () => {
    expect(new ApiError({ status: 0, code: "NETWORK", title: "Offline", detail: "" }).message).toBe(
      "Offline",
    );
    expect(new ApiError({ status: 0, code: "NETWORK", title: "", detail: "" }).message).toBe(
      "NETWORK",
    );
  });
});

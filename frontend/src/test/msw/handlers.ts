/**
 * A faithful stand-in for the Go service: same URLs, same status codes, same problem+json bodies as
 * `backend/internal/httpapi` (docs/PLAN.md §1.4, docs/errors.md). Tests exercise the real
 * `apiFetch`, so anything that differs here is a bug this suite would not catch.
 */
import { delay, http, HttpResponse, type RequestHandler } from "msw";

import { apiUrl } from "@/lib/api";
import { CALCULATE_ENDPOINT, OPERATIONS_ENDPOINT } from "@/lib/query-config";
import type { FieldError, OperationSpec, ProblemDetails } from "@/types/calculator";

/** Placeholder ID, the same one the backend's golden files use. */
export const TEST_REQUEST_ID = "4bf92f35-77b3-4da6-a3ce-929d0e0e4736";

const TYPE_BASE = "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#";

const PROBLEM_META = {
  INVALID_BODY: { status: 400, title: "Invalid request body" },
  VALIDATION_FAILED: { status: 400, title: "Validation failed" },
  UNSUPPORTED_OPERATION: { status: 422, title: "Unsupported operation" },
  UNEXPECTED_OPERAND: { status: 422, title: "Unexpected operand" },
  DIVISION_BY_ZERO: { status: 422, title: "Division by zero" },
  DOMAIN_ERROR: { status: 422, title: "Result is not a real number" },
  RESULT_NOT_FINITE: { status: 422, title: "Result is not finite" },
  UNSUPPORTED_MEDIA_TYPE: { status: 415, title: "Unsupported media type" },
  PAYLOAD_TOO_LARGE: { status: 413, title: "Payload too large" },
  RATE_LIMITED: { status: 429, title: "Too many requests" },
  NOT_FOUND: { status: 404, title: "Not found" },
  METHOD_NOT_ALLOWED: { status: 405, title: "Method not allowed" },
  INTERNAL: { status: 500, title: "Internal server error" },
} as const satisfies Record<string, { status: number; title: string }>;

export type ProblemCode = keyof typeof PROBLEM_META;

/** The backend's operation registry, in `GET /api/v1/operations` order. */
export const TEST_OPERATIONS: readonly OperationSpec[] = [
  { name: "add", symbol: "+", arity: 2 },
  { name: "subtract", symbol: "−", arity: 2 },
  { name: "multiply", symbol: "×", arity: 2 },
  { name: "divide", symbol: "÷", arity: 2 },
  { name: "power", symbol: "^", arity: 2 },
  { name: "sqrt", symbol: "√", arity: 1 },
  { name: "percent", symbol: "%", arity: 2 },
];

const OPERATION_NAMES = TEST_OPERATIONS.map((op) => op.name).join(", ");

/** Build a problem document exactly as `httpapi.NewProblem` + `httpapi.Write` do. */
export function problem(
  code: ProblemCode,
  detail: string,
  options: { readonly errors?: readonly FieldError[]; readonly instance?: string } = {},
): ProblemDetails {
  const meta = PROBLEM_META[code];
  const body: ProblemDetails = {
    type: `${TYPE_BASE}${code.toLowerCase()}`,
    title: meta.title,
    status: meta.status,
    detail,
    code,
    instance: options.instance ?? "/api/v1/calculate",
    requestId: TEST_REQUEST_ID,
    ...(options.errors === undefined ? {} : { errors: options.errors }),
  };
  return body;
}

function problemResponse(body: ProblemDetails, headers: Record<string, string> = {}): Response {
  return HttpResponse.json(body, {
    status: body.status,
    headers: {
      "Content-Type": "application/problem+json",
      "X-Request-ID": TEST_REQUEST_ID,
      ...headers,
    },
  });
}

/** Ready-made failing response, for handlers and for asserting on the exact wire body. */
export function problemFor(
  code: ProblemCode,
  detail: string,
  options: { readonly errors?: readonly FieldError[]; readonly instance?: string } = {},
): Response {
  return problemResponse(problem(code, detail, options));
}

// --- The calculator, reimplemented from backend/internal/calc ---------------------------------

type Evaluated =
  | { readonly ok: true; readonly result: number }
  | { readonly ok: false; readonly response: Response };

function divisionByZero(operation: string): Response {
  return operation === "divide"
    ? problemFor("DIVISION_BY_ZERO", `b must be non-zero for operation "divide"`, {
        errors: [{ field: "b", message: "must be non-zero" }],
      })
    : problemFor("DIVISION_BY_ZERO", `operation "${operation}" divides by zero for these operands`);
}

function domainError(operation: string): Response {
  return problemFor(
    "DOMAIN_ERROR",
    `operation "${operation}" has no real-number result for these operands`,
  );
}

function evaluate(operation: string, a: number, b: number | undefined): Evaluated {
  let result: number;
  switch (operation) {
    case "add":
      result = a + (b ?? 0);
      break;
    case "subtract":
      result = a - (b ?? 0);
      break;
    case "multiply":
      result = a * (b ?? 0);
      break;
    case "divide":
      if (b === 0) return { ok: false, response: divisionByZero(operation) };
      result = a / (b ?? 1);
      break;
    case "power":
      if (a === 0 && (b ?? 0) < 0) return { ok: false, response: divisionByZero(operation) };
      result = Math.pow(a, b ?? 0);
      if (Number.isNaN(result)) return { ok: false, response: domainError(operation) };
      break;
    case "sqrt":
      if (a < 0) return { ok: false, response: domainError(operation) };
      result = Math.sqrt(a);
      break;
    default:
      // percent: "b percent of a"; a*(b/100) rescues the cases where a*b alone overflows.
      result = (a * (b ?? 0)) / 100;
      if (!Number.isFinite(result)) result = a * ((b ?? 0) / 100);
      break;
  }
  if (!Number.isFinite(result)) {
    return {
      ok: false,
      response: problemFor(
        "RESULT_NOT_FINITE",
        `the result of operation "${operation}" is outside the 64-bit floating-point range`,
      ),
    };
  }
  return { ok: true, result: result === 0 ? 0 : result };
}

// --- Decoding and validation, reimplemented from httpapi.decodeJSON / httpapi.validate ---------

const KNOWN_FIELDS = new Set(["operation", "a", "b"]);

/** `json.UnmarshalTypeError.Offset`: bytes read when the bad value ended. */
function byteOffset(raw: string, field: string, value: unknown): number {
  const key = `"${field}":`;
  const start = raw.indexOf(key);
  return start < 0 ? raw.length : start + key.length + JSON.stringify(value).length;
}

function invalidBody(detail: string, errors?: readonly FieldError[]): Response {
  return problemFor("INVALID_BODY", detail, errors === undefined ? {} : { errors });
}

type Decoded =
  | { readonly ok: true; readonly body: Record<string, unknown> }
  | { readonly ok: false; readonly response: Response };

function decode(raw: string): Decoded {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return { ok: false, response: invalidBody("body is not valid JSON") };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { ok: false, response: invalidBody("body must be a JSON object") };
  }
  const body = parsed as Record<string, unknown>;
  for (const key of Object.keys(body)) {
    if (!KNOWN_FIELDS.has(key)) {
      return { ok: false, response: invalidBody(`unknown field "${key}"`) };
    }
  }
  for (const field of ["a", "b"] as const) {
    const value = body[field];
    if (value !== undefined && value !== null && typeof value !== "number") {
      return {
        ok: false,
        response: invalidBody(
          `field "${field}" must be a number (byte ${String(byteOffset(raw, field, value))})`,
          [{ field, message: "must be a number" }],
        ),
      };
    }
  }
  if (body["operation"] !== undefined && typeof body["operation"] !== "string") {
    return {
      ok: false,
      response: invalidBody(
        `field "operation" must be a string (byte ${String(byteOffset(raw, "operation", body["operation"]))})`,
        [{ field: "operation", message: "must be a string" }],
      ),
    };
  }
  return { ok: true, body };
}

const calculate = http.post(apiUrl(CALCULATE_ENDPOINT), async ({ request }) => {
  const decoded = decode(await request.text());
  if (!decoded.ok) return decoded.response;

  const operation = typeof decoded.body["operation"] === "string" ? decoded.body["operation"] : "";
  const a = typeof decoded.body["a"] === "number" ? decoded.body["a"] : undefined;
  const b = typeof decoded.body["b"] === "number" ? decoded.body["b"] : undefined;

  // 400 first, listing every missing or non-finite field, exactly like httpapi.validate.
  const errors: FieldError[] = [];
  if (operation === "") errors.push({ field: "operation", message: "is required" });
  if (a === undefined) errors.push({ field: "a", message: "is required" });
  const spec = TEST_OPERATIONS.find((op) => op.name === operation);
  if (spec?.arity === 2 && b === undefined) {
    errors.push({ field: "b", message: `is required for operation "${operation}"` });
  }
  if (errors.length > 0) {
    return problemFor(
      "VALIDATION_FAILED",
      `missing or invalid fields: ${errors.map((e) => e.field).join(", ")}`,
      { errors },
    );
  }
  if (spec === undefined) {
    return problemFor(
      "UNSUPPORTED_OPERATION",
      `operation "${operation}" is not supported; GET /api/v1/operations lists the supported operations`,
      { errors: [{ field: "operation", message: `must be one of: ${OPERATION_NAMES}` }] },
    );
  }
  if (spec.arity === 1 && b !== undefined) {
    return problemFor(
      "UNEXPECTED_OPERAND",
      `operation "${operation}" takes a single operand; omit b`,
      {
        errors: [{ field: "b", message: `must be omitted for operation "${operation}"` }],
      },
    );
  }

  const evaluated = evaluate(operation, a ?? 0, b);
  if (!evaluated.ok) return evaluated.response;
  return HttpResponse.json(
    { operation, a, ...(b === undefined ? {} : { b }), result: evaluated.result },
    { headers: { "X-Request-ID": TEST_REQUEST_ID } },
  );
});

const operations = http.get(apiUrl(OPERATIONS_ENDPOINT), () =>
  HttpResponse.json(
    { operations: TEST_OPERATIONS },
    { headers: { "X-Request-ID": TEST_REQUEST_ID } },
  ),
);

/** Default handlers: the happy backend. Per-test failures go through the helpers below. */
export const handlers: RequestHandler[] = [calculate, operations];

// --- Per-test overrides: `server.use(rateLimited())` -------------------------------------------

const METHOD_FOR: Readonly<Record<string, "get" | "post">> = {
  [CALCULATE_ENDPOINT]: "post",
  [OPERATIONS_ENDPOINT]: "get",
};

function on(endpoint: string, resolver: Parameters<typeof http.get>[1]): RequestHandler {
  return http[METHOD_FOR[endpoint] ?? "get"](apiUrl(endpoint), resolver);
}

/** 429 with a `Retry-After` header, as the rate limiter will send (ADR-0005). */
export function rateLimited(retryAfterSeconds = 30, endpoint = CALCULATE_ENDPOINT): RequestHandler {
  return on(endpoint, () =>
    problemResponse(
      problem("RATE_LIMITED", "too many requests; retry later", { instance: `/api${endpoint}` }),
      { "Retry-After": String(retryAfterSeconds) },
    ),
  );
}

/** A 5xx that is not problem+json at all — an upstream proxy, not our service. */
export function serverErrorText(
  body = "502 Bad Gateway",
  status = 502,
  endpoint = CALCULATE_ENDPOINT,
): RequestHandler {
  return on(
    endpoint,
    () => new HttpResponse(body, { status, headers: { "Content-Type": "text/plain" } }),
  );
}

/** The request never completes: offline, DNS failure, connection reset. */
export function networkError(endpoint = CALCULATE_ENDPOINT): RequestHandler {
  return on(endpoint, () => HttpResponse.error());
}

/** 200 with a body that claims to be JSON and is not. */
export function malformedJson(endpoint = CALCULATE_ENDPOINT): RequestHandler {
  return on(
    endpoint,
    () =>
      new HttpResponse('{"result":', {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
  );
}

/** 200 with well-formed JSON of the wrong shape, for the schema-validation path. */
export function jsonBody(
  body: unknown,
  endpoint = CALCULATE_ENDPOINT,
  init: ResponseInit = {},
): RequestHandler {
  return on(
    endpoint,
    () =>
      new HttpResponse(JSON.stringify(body), {
        status: 200,
        ...init,
        headers: { "Content-Type": "application/json", ...init.headers },
      }),
  );
}

/** Answers after `ms`, for the timeout path. */
export function slowResponse(ms: number, endpoint = CALCULATE_ENDPOINT): RequestHandler {
  return on(endpoint, async () => {
    await delay(ms);
    return HttpResponse.json({});
  });
}

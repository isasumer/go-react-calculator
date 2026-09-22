/**
 * The single doorway to the backend. Everything that talks HTTP goes through `apiFetch`, so error
 * handling, timeouts and response validation exist in exactly one place.
 *
 * Every rejection is an {@link ApiError} with a stable `code`: either one the backend emitted
 * (docs/errors.md) or one of the four this module adds for failures that never reach the server.
 */
import type { ZodError, ZodType } from "zod";

import { getConfig } from "@/config";
import { type FieldError, problemSchema } from "@/types/calculator";

/** A request that gets no answer in this long is treated as failed. */
export const DEFAULT_TIMEOUT_MS = 10_000;

/** Longest excerpt of an unexpected response body to quote in `ApiError.detail`. */
const MAX_DETAIL_LENGTH = 200;

/**
 * Codes produced by the client rather than the server:
 * - `NETWORK` — the request never completed (offline, DNS, CORS, caller aborted).
 * - `TIMEOUT` — no answer within `timeoutMs`.
 * - `UNEXPECTED` — an answer arrived that is not the JSON we can make sense of.
 * - `INVALID_RESPONSE` — valid JSON that does not match the schema; we fail closed rather than
 *   hand a half-typed object to the UI.
 */
export type ClientErrorCode = "NETWORK" | "TIMEOUT" | "UNEXPECTED" | "INVALID_RESPONSE";

export interface ApiErrorInit {
  readonly status: number;
  readonly code: string;
  readonly title: string;
  readonly detail: string;
  readonly requestId?: string | undefined;
  readonly errors?: readonly FieldError[] | undefined;
  readonly retryAfterSeconds?: number | undefined;
  readonly cause?: unknown;
}

/**
 * Every failure `apiFetch` can produce. Branch on `code`, never on `title` or `detail`: those are
 * for humans and may be reworded (ADR-0005). `src/lib/error-messages.ts` turns a code into the
 * sentence a user sees.
 */
export class ApiError extends Error {
  override name = "ApiError";

  /** HTTP status, or 0 when the request never completed. */
  readonly status: number;
  /** Stable machine code: a backend code from docs/errors.md or a {@link ClientErrorCode}. */
  readonly code: string;
  /** Fixed summary for the code. */
  readonly title: string;
  /** Explains this occurrence. */
  readonly detail: string;
  /** `X-Request-ID`, from the body or the response header. Quote it in bug reports. */
  readonly requestId?: string | undefined;
  /** One entry per offending request field; empty when the failure is not field-specific. */
  readonly errors: readonly FieldError[];
  /** Seconds to wait before retrying, from a `Retry-After` header. */
  readonly retryAfterSeconds?: number | undefined;

  constructor(init: ApiErrorInit) {
    super(init.detail || init.title || init.code, { cause: init.cause });
    this.status = init.status;
    this.code = init.code;
    this.title = init.title;
    this.detail = init.detail;
    this.requestId = init.requestId;
    this.errors = init.errors ?? [];
    this.retryAfterSeconds = init.retryAfterSeconds;
  }
}

/** Narrows an unknown rejection to an {@link ApiError}. */
export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError;
}

/**
 * Join the configured base with an endpoint path. Exported for tests and for MSW handlers, which
 * must match exactly the URL this module builds.
 */
export function joinApiPath(base: string, path: string): string {
  const suffix = path.startsWith("/") ? path : `/${path}`;
  return base === "/" ? suffix : `${base}${suffix}`;
}

/** Absolute URL for an endpoint. A path-only base (`/api`) resolves against the document origin. */
export function apiUrl(path: string): string {
  return new URL(joinApiPath(getConfig().apiBaseUrl, path), globalThis.location.origin).toString();
}

export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export interface ApiFetchOptions<T> {
  /** Defaults to `POST` when a body is given, `GET` otherwise. */
  readonly method?: HttpMethod;
  /** Serialised as JSON; `undefined` sends no body. */
  readonly body?: unknown;
  /** Caller's cancellation signal, merged with the timeout. */
  readonly signal?: AbortSignal | undefined;
  /** Validates the parsed body. Without it the caller's `T` is an unchecked assertion. */
  readonly schema?: ZodType<T>;
  /** Overrides {@link DEFAULT_TIMEOUT_MS}. */
  readonly timeoutMs?: number;
}

/**
 * Send a JSON request and return the parsed (and, with `schema`, validated) body.
 *
 * @throws {ApiError} always — never a `TypeError`, a `DOMException` or a `ZodError`.
 */
export async function apiFetch<T>(path: string, options: ApiFetchOptions<T> = {}): Promise<T> {
  const { method, body, signal, schema, timeoutMs = DEFAULT_TIMEOUT_MS } = options;
  const timeout = AbortSignal.timeout(timeoutMs);

  const headers: Record<string, string> = {
    Accept: "application/json, application/problem+json",
  };
  const init: RequestInit = {
    method: method ?? (body === undefined ? "GET" : "POST"),
    headers,
    signal: anySignal(signal === undefined ? [timeout] : [timeout, signal]),
  };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(body);
  }

  let response: Response;
  let text: string;
  try {
    response = await fetch(apiUrl(path), init);
    text = await response.text();
  } catch (cause) {
    throw timeout.aborted
      ? new ApiError({
          status: 0,
          code: "TIMEOUT",
          title: "Request timed out",
          detail: `the server did not respond within ${String(timeoutMs)} ms`,
          cause,
        })
      : new ApiError({
          status: 0,
          code: "NETWORK",
          title: "Network error",
          detail: "the request could not be sent",
          cause,
        });
  }

  const payload = parseJson(text);
  if (!response.ok) {
    throw errorFromResponse(response, payload, text);
  }
  if (!payload.ok) {
    throw new ApiError({
      status: response.status,
      code: "UNEXPECTED",
      title: "Unexpected response",
      detail: describeBody(text, response.status),
      requestId: requestIdOf(response),
    });
  }
  if (schema === undefined) {
    // No schema: the caller's T is an assertion, exactly as with a plain `fetch`.
    return payload.value as T;
  }
  const parsed = schema.safeParse(payload.value);
  if (!parsed.success) {
    throw new ApiError({
      status: response.status,
      code: "INVALID_RESPONSE",
      title: "Unexpected response",
      detail: describeIssues(parsed.error),
      requestId: requestIdOf(response),
      cause: parsed.error,
    });
  }
  return parsed.data;
}

type JsonResult = { readonly ok: true; readonly value: unknown } | { readonly ok: false };

function parseJson(text: string): JsonResult {
  try {
    return { ok: true, value: JSON.parse(text) as unknown };
  } catch {
    return { ok: false };
  }
}

function requestIdOf(response: Response): string | undefined {
  return response.headers.get("X-Request-ID") ?? undefined;
}

/**
 * Turn a failed response into an ApiError. A problem+json document (or any JSON body carrying a
 * `code`) keeps its code; anything else — a proxy's HTML, a plain-text 500, an empty body — becomes
 * `UNEXPECTED` so the UI still has something stable to branch on.
 */
function errorFromResponse(response: Response, payload: JsonResult, text: string): ApiError {
  const requestId = requestIdOf(response);
  const retryAfterSeconds = parseRetryAfter(response.headers.get("Retry-After"));
  const problem = payload.ok ? problemSchema.safeParse(payload.value) : undefined;

  if (problem?.success) {
    const { code, status, title, detail, requestId: bodyRequestId, errors } = problem.data;
    return new ApiError({
      status: status ?? response.status,
      code,
      title: title ?? (response.statusText || code),
      detail: detail ?? "",
      requestId: bodyRequestId ?? requestId,
      errors,
      retryAfterSeconds,
    });
  }

  return new ApiError({
    status: response.status,
    code: "UNEXPECTED",
    title: "Unexpected response",
    detail: describeBody(text, response.status),
    requestId,
    retryAfterSeconds,
  });
}

/** `Retry-After` is either delay-seconds or an HTTP-date (RFC 9110 §10.2.3). */
function parseRetryAfter(value: string | null): number | undefined {
  if (value === null || value.trim() === "") {
    return undefined;
  }
  const seconds = Number(value);
  if (Number.isFinite(seconds)) {
    return Math.max(0, Math.ceil(seconds));
  }
  const date = Date.parse(value);
  if (Number.isNaN(date)) {
    return undefined;
  }
  return Math.max(0, Math.ceil((date - Date.now()) / 1000));
}

function describeBody(text: string, status: number): string {
  const trimmed = text.trim();
  if (trimmed === "") {
    return `the server returned ${String(status)} with an empty body`;
  }
  const excerpt =
    trimmed.length > MAX_DETAIL_LENGTH ? `${trimmed.slice(0, MAX_DETAIL_LENGTH)}…` : trimmed;
  return `the server returned ${String(status)}: ${excerpt}`;
}

function describeIssues(error: ZodError): string {
  const summary = error.issues
    .slice(0, 3)
    .map((issue) => `${issue.path.join(".") || "(root)"}: ${issue.message}`)
    .join("; ");
  return `the response does not match the expected shape (${summary})`;
}

/** `AbortSignal.any` is missing before Safari 17.4, so the lib's required member is optional here. */
type MaybeAbortSignalAny = { any?: (signals: AbortSignal[]) => AbortSignal };

/**
 * Abort as soon as any input signal aborts. Uses `AbortSignal.any` where the runtime has it and
 * falls back to wiring the listeners by hand where it does not.
 */
function anySignal(signals: AbortSignal[]): AbortSignal {
  const ctor: MaybeAbortSignalAny = AbortSignal;
  if (typeof ctor.any === "function") {
    return ctor.any(signals);
  }
  const controller = new AbortController();
  for (const signal of signals) {
    if (signal.aborted) {
      controller.abort(signal.reason);
      break;
    }
    signal.addEventListener(
      "abort",
      () => {
        controller.abort(signal.reason);
      },
      // Listeners are dropped as soon as the merged signal aborts, so nothing leaks.
      { once: true, signal: controller.signal },
    );
  }
  return controller.signal;
}

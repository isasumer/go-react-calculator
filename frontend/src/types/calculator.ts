/**
 * Wire types for the calculator API (docs/PLAN.md §1.4, frozen after B1-02) and the zod schemas
 * `apiFetch` validates responses with.
 *
 * Hand-written interfaces describe what *we* send and the documented shape of a problem document.
 * Inferred types describe what the backend sends, so the schema and the type can never drift.
 */
import { z } from "zod";

/** Operations in the backend registry, in `GET /api/v1/operations` order. */
export const OPERATION_NAMES = [
  "add",
  "subtract",
  "multiply",
  "divide",
  "power",
  "sqrt",
  "percent",
] as const;

/** Machine name of a supported operation. */
export type OperationName = (typeof OPERATION_NAMES)[number];

export const operationNameSchema = z.enum(OPERATION_NAMES);

/**
 * Body of `POST /api/v1/calculate`. `b` is required for binary operations and must be omitted for
 * unary ones (`sqrt`); sending it anyway is a 422 `UNEXPECTED_OPERAND`.
 */
export interface CalculateRequest {
  readonly operation: OperationName;
  readonly a: number;
  readonly b?: number | undefined;
}

/**
 * 200 body of `POST /api/v1/calculate`: the request echoed back with its result. `operation` is the
 * enum rather than a free string because the backend can only echo an operation we just sent.
 */
export const calculateResponseSchema = z.object({
  operation: operationNameSchema,
  a: z.number(),
  b: z.number().optional(),
  result: z.number(),
});

export type CalculateResponse = z.infer<typeof calculateResponseSchema>;

/**
 * One entry of `GET /api/v1/operations`. `name` is a plain string, not the enum: discovery exists so
 * a backend that gains an operation does not break this client, and an unknown name is simply one we
 * have no key for. `arity` stays closed — a calculator operation takes one or two operands.
 */
export const operationSpecSchema = z.object({
  name: z.string().min(1),
  symbol: z.string().min(1),
  arity: z.union([z.literal(1), z.literal(2)]),
});

export type OperationSpec = z.infer<typeof operationSpecSchema>;

export const operationsResponseSchema = z.object({
  operations: z.array(operationSpecSchema),
});

export type OperationsResponse = z.infer<typeof operationsResponseSchema>;

/** One offending request field in a problem document's `errors[]`. */
export interface FieldError {
  readonly field: string;
  readonly message: string;
}

/**
 * The documented RFC 9457 error body (docs/errors.md → "Shape"). Every member except `requestId`
 * and `errors` is always present on a response the backend produced.
 */
export interface ProblemDetails {
  /** URL of the code's section in docs/errors.md. */
  readonly type: string;
  /** Fixed human-readable summary for the code. */
  readonly title: string;
  readonly status: number;
  /** Explains this occurrence. For humans: never branch on it. */
  readonly detail: string;
  /** Stable machine-readable code. Branch on this. */
  readonly code: string;
  /** Request path. */
  readonly instance: string;
  readonly requestId?: string;
  readonly errors?: readonly FieldError[];
}

export const fieldErrorSchema = z.object({
  field: z.string(),
  message: z.string(),
});

/**
 * Lenient parser for an error body. Only `code` is required and unknown members are dropped, because
 * `apiFetch` must still build a useful `ApiError` from a truncated body, a future problem document or
 * a proxy's own JSON error. `ProblemDetails` above is the contract; this is what we dare to assume.
 */
export const problemSchema = z.object({
  type: z.string().optional(),
  title: z.string().optional(),
  status: z.number().optional(),
  detail: z.string().optional(),
  code: z.string().min(1),
  instance: z.string().optional(),
  requestId: z.string().optional(),
  errors: z.array(fieldErrorSchema).optional(),
});

export type ParsedProblem = z.infer<typeof problemSchema>;

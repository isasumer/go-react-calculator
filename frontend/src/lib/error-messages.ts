/**
 * The one place a machine code becomes a sentence a person reads.
 *
 * Keys are the stable codes from docs/errors.md plus the four {@link ClientErrorCode}s `apiFetch`
 * adds. Messages are short, say what happened in the user's terms and never quote server text —
 * `detail` is written for developers and may change wording at any time (ADR-0005).
 */
import { isApiError } from "@/lib/api";

/** Shown for an error we have no entry for: an unknown code, or something that is not an ApiError. */
export const FALLBACK_ERROR_MESSAGE = "Something went wrong. Please try again.";

export const ERROR_MESSAGES: Readonly<Record<string, string>> = {
  // --- Backend codes (docs/errors.md) -------------------------------------------------------
  // The client sent something wrong: a bug here, not a user mistake, so the wording stays neutral.
  INVALID_BODY: "The calculator sent an invalid request. Please try again.",
  VALIDATION_FAILED: "Please enter a number for every field.",
  UNSUPPORTED_OPERATION: "That operation is not available.",
  UNEXPECTED_OPERAND: "That operation takes a single number.",
  // The user asked for something arithmetic cannot answer.
  DIVISION_BY_ZERO: "Cannot divide by zero.",
  DOMAIN_ERROR: "That calculation has no real-number result.",
  RESULT_NOT_FINITE: "The result is too large to display.",
  // Transport-level mistakes; all of them are client bugs.
  UNSUPPORTED_MEDIA_TYPE: "The calculator sent an invalid request. Please try again.",
  PAYLOAD_TOO_LARGE: "That number is too long.",
  NOT_FOUND: "The calculator service could not be reached.",
  METHOD_NOT_ALLOWED: "The calculator service could not be reached.",
  // Rate limiting lands with B1-04; the code is fixed by ADR-0005 so the message can exist now.
  RATE_LIMITED: "Too many calculations. Please wait a moment and try again.",
  INTERNAL: "The calculator service had a problem. Please try again.",

  // --- Client-side codes (src/lib/api.ts) ---------------------------------------------------
  NETWORK: "Cannot reach the calculator service. Check your connection.",
  TIMEOUT: "The calculator service took too long to respond. Please try again.",
  UNEXPECTED: "The calculator service returned something unexpected.",
  INVALID_RESPONSE: "The calculator service returned something unexpected.",
};

/**
 * The sentence to show for a caught error. Anything that is not an `ApiError`, and any code we do
 * not know, falls back to {@link FALLBACK_ERROR_MESSAGE} rather than leaking an exception message.
 */
export function messageForError(e: unknown): string {
  if (!isApiError(e)) {
    return FALLBACK_ERROR_MESSAGE;
  }
  return ERROR_MESSAGES[e.code] ?? FALLBACK_ERROR_MESSAGE;
}

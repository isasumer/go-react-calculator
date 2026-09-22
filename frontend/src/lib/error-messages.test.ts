import { describe, expect, it } from "vitest";

import { ApiError } from "@/lib/api";
import { ERROR_MESSAGES, FALLBACK_ERROR_MESSAGE, messageForError } from "@/lib/error-messages";

/**
 * Every code the client can ever see. Backend codes are the rows of docs/errors.md (plus
 * RATE_LIMITED, fixed by ADR-0005 and landing with B1-04); the last four are produced by apiFetch.
 * A code added to the catalogue without a message here fails this test.
 */
const ALL_CODES = [
  "INVALID_BODY",
  "VALIDATION_FAILED",
  "UNSUPPORTED_OPERATION",
  "UNEXPECTED_OPERAND",
  "DIVISION_BY_ZERO",
  "DOMAIN_ERROR",
  "RESULT_NOT_FINITE",
  "UNSUPPORTED_MEDIA_TYPE",
  "PAYLOAD_TOO_LARGE",
  "RATE_LIMITED",
  "NOT_FOUND",
  "METHOD_NOT_ALLOWED",
  "INTERNAL",
  "NETWORK",
  "TIMEOUT",
  "UNEXPECTED",
  "INVALID_RESPONSE",
] as const;

function apiError(code: string): ApiError {
  return new ApiError({ status: 500, code, title: "Title", detail: "internal detail" });
}

describe("ERROR_MESSAGES", () => {
  it("covers every code, and nothing else", () => {
    expect(Object.keys(ERROR_MESSAGES).sort()).toEqual([...ALL_CODES].sort());
  });

  it.each(ALL_CODES)("gives %s a short, user-facing message", (code) => {
    const message = messageForError(apiError(code));

    expect(message).not.toBe(FALLBACK_ERROR_MESSAGE);
    expect(message.length).toBeGreaterThan(10);
    expect(message.length).toBeLessThanOrEqual(80);
    expect(message).toMatch(/[.!]$/);
  });

  it("never leaks the server's detail text", () => {
    for (const code of ALL_CODES) {
      expect(messageForError(apiError(code))).not.toContain("internal detail");
    }
  });
});

describe("messageForError", () => {
  it("maps a known code", () => {
    expect(messageForError(apiError("DIVISION_BY_ZERO"))).toBe("Cannot divide by zero.");
  });

  it("falls back for a code released after this build", () => {
    expect(messageForError(apiError("TEAPOT"))).toBe(FALLBACK_ERROR_MESSAGE);
  });

  it.each([
    ["a plain Error", new Error("kaboom")],
    ["a string", "DIVISION_BY_ZERO"],
    ["null", null],
    ["undefined", undefined],
  ])("falls back for %s", (_label, value) => {
    expect(messageForError(value)).toBe(FALLBACK_ERROR_MESSAGE);
  });
});

import { describe, expect, it } from "vitest";

import { formatEntry, formatResult, parseEntry } from "@/lib/format-number";

describe("formatResult", () => {
  // Documented examples from ADR-0007.
  it.each([
    [0.30000000000000004, "0.3"],
    [-0, "0"],
    [0, "0"],
    [19, "19"],
    [2.5, "2.5"],
    [1 / 3, "0.333333333333"],
    [123456789012, "123456789012"],
    [-42.5, "-42.5"],
  ])("formats %p as %s", (input, expected) => {
    expect(formatResult(input)).toBe(expected);
  });

  it("switches to exponent notation at or beyond 1e15", () => {
    expect(formatResult(1e15)).toBe("1e+15");
    expect(formatResult(1234567890123456)).toBe("1.23456789012e+15");
    // Just under the threshold takes the fixed path, but rounding to 12 significant digits still
    // carries it to the next power of ten — an honest artefact of rounding, not a bug.
    expect(formatResult(999999999999999)).toBe("1000000000000000");
  });

  it("switches to exponent notation below 1e-6 in magnitude", () => {
    expect(formatResult(0.0000009)).toBe("9e-7");
    expect(formatResult(-0.0000009)).toBe("-9e-7");
    // At the threshold itself, still fixed.
    expect(formatResult(0.000001)).toBe("0.000001");
  });

  it("trims trailing zeros out of the exponential mantissa", () => {
    expect(formatResult(2e20)).toBe("2e+20");
    expect(formatResult(1.5e20)).toBe("1.5e+20");
  });

  it("respects a custom maxSignificant", () => {
    expect(formatResult(1 / 3, { maxSignificant: 4 })).toBe("0.3333");
  });

  it("leaves a single-digit exponential mantissa (no decimal point to trim) alone", () => {
    expect(formatResult(2e20, { maxSignificant: 1 })).toBe("2e+20");
  });

  it("throws for non-finite input", () => {
    expect(() => formatResult(Number.NaN)).toThrow(RangeError);
    expect(() => formatResult(Number.POSITIVE_INFINITY)).toThrow(RangeError);
    expect(() => formatResult(Number.NEGATIVE_INFINITY)).toThrow(RangeError);
  });

  it("keeps every formatted result round-tripping within 1e-11 relative error", () => {
    // A fixed seed (mulberry32) so a failure is reproducible without storing 200 numbers by hand.
    let state = 0x2f6e2b1;
    function next(): number {
      state |= 0;
      state = (state + 0x6d2b79f5) | 0;
      let t = Math.imul(state ^ (state >>> 15), 1 | state);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    }

    for (let i = 0; i < 200; i += 1) {
      const exponent = next() * 40 - 20; // roughly 1e-20 .. 1e20
      const sign = next() < 0.5 ? -1 : 1;
      const value = sign * (1 + next() * 8) * 10 ** exponent;
      if (!Number.isFinite(value) || value === 0) {
        continue;
      }

      const formatted = formatResult(value);
      const roundTripped = Number(formatted);
      const relativeError = Math.abs((roundTripped - value) / value);
      expect(relativeError).toBeLessThan(1e-11);
    }
  });
});

describe("formatEntry", () => {
  it.each([
    ["0", "0"],
    ["7", "7"],
    ["1234", "1,234"],
    ["1234567", "1,234,567"],
    ["1234.5", "1,234.5"],
    ["1234.", "1,234."],
    ["0.5", "0.5"],
    ["-1234567", "-1,234,567"],
    ["-1234.56", "-1,234.56"],
  ])("groups %s as %s", (raw, expected) => {
    expect(formatEntry(raw)).toBe(expected);
  });

  it("never touches digits after the decimal point", () => {
    expect(formatEntry("1000000.000100")).toBe("1,000,000.000100");
  });

  it("leaves an empty integer part alone (a bare leading '.')", () => {
    expect(formatEntry(".5")).toBe(".5");
  });
});

describe("parseEntry", () => {
  it.each([
    ["0", 0],
    ["-0", 0],
    ["-0.", 0],
    ["1,234.5", 1234.5],
    ["-1,234", -1234],
    ["7", 7],
  ])("parses %s as %d", (raw, expected) => {
    expect(parseEntry(raw)).toBe(expected);
  });
});

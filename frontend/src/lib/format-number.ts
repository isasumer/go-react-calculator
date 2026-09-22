/**
 * Number formatting and parsing for the display (ADR-0007).
 *
 * The engine (`calculator-engine.ts`) keeps its internal `display` string as `String(value)` — the
 * shortest round-tripping decimal, which is exactly right for arithmetic but wrong for a screen: it
 * shows float noise (`0.1 + 0.2` → `0.30000000000000004`) and never groups digits. This module is
 * the presentation layer on top of that string; it never feeds back into the engine.
 *
 * `formatResult` is for a value the engine has finished with (a result, or an operand already
 * shown in the expression line). `formatEntry` is for a value still being typed, where the only
 * thing to add is thousands grouping — the trailing decimal point and the digits after it are the
 * user's keystrokes and must survive untouched. `parseEntry` is the inverse of both: text back to
 * a `number`.
 */

/** `Intl.NumberFormat("en-US")` explicitly (ADR-0007) — a locale switch is follow-up #34. */
const GROUPING_FORMAT = new Intl.NumberFormat("en-US");

export interface FormatResultOptions {
  /** Significant digits kept before trimming float noise or switching to exponent notation. */
  readonly maxSignificant?: number;
}

const DEFAULT_MAX_SIGNIFICANT = 12;

/** Magnitudes at or beyond this switch to exponent notation. */
const EXPONENT_UPPER_BOUND = 1e15;

/** Magnitudes strictly below this (and non-zero) switch to exponent notation. */
const EXPONENT_LOWER_BOUND = 1e-6;

/**
 * A finished value, formatted for a person: `12` significant digits by default, trimmed of the
 * float noise a `float64` calculation leaves behind, switching to exponent notation once the
 * magnitude would otherwise need more digits than a display column can show.
 *
 * `NaN` and `±Infinity` never reach here — the engine and the backend both guard against them
 * (ADR-0003, `RESULT_NOT_FINITE`) — so a non-finite input is a bug upstream, not a value to render.
 */
export function formatResult(n: number, options: FormatResultOptions = {}): string {
  const maxSignificant = options.maxSignificant ?? DEFAULT_MAX_SIGNIFICANT;
  if (!Number.isFinite(n)) {
    throw new RangeError(`formatResult expects a finite number, got ${String(n)}`);
  }

  // Normalises -0 to 0: the sign of a zero is an implementation detail no user asked to see.
  const value = n === 0 ? 0 : n;
  const magnitude = Math.abs(value);

  if (magnitude !== 0 && (magnitude >= EXPONENT_UPPER_BOUND || magnitude < EXPONENT_LOWER_BOUND)) {
    return formatExponential(value, maxSignificant);
  }
  return formatFixed(value, maxSignificant);
}

/**
 * `toPrecision` rounds to `maxSignificant` significant digits; feeding that back through `Number`
 * and then `String` gives the shortest decimal that round-trips to the rounded value, which is what
 * trims `0.30000000000000004` down to `0.3` without also trimming a genuine `0.30`-shaped answer
 * down to the wrong digit count (there is no "wrong" digit count here — trailing zeros are noise).
 */
function formatFixed(value: number, maxSignificant: number): string {
  const rounded = Number(value.toPrecision(maxSignificant));
  return Object.is(rounded, -0) ? "0" : String(rounded);
}

/** `toExponential` gives a fixed mantissa width; trimming trailing zeros keeps it honest. */
function formatExponential(value: number, maxSignificant: number): string {
  const [mantissa, exponent] = value.toExponential(maxSignificant - 1).split("e");
  const trimmedMantissa = trimTrailingZeros(mantissa ?? "0");
  const exponentDigits = (exponent ?? "+0").replace(/^\+/, "+");
  return `${trimmedMantissa}e${exponentDigits}`;
}

function trimTrailingZeros(mantissa: string): string {
  if (!mantissa.includes(".")) {
    return mantissa;
  }
  return mantissa.replace(/0+$/, "").replace(/\.$/, "");
}

/**
 * An in-progress entry (the raw string the engine is building as the user types), with en-US
 * thousands grouping added to the integer part. The trailing decimal point — present the instant
 * `.` is pressed, before any fractional digit exists — and every fractional digit already typed are
 * left exactly as they are: grouping only ever touches digits to the left of the point.
 */
export function formatEntry(raw: string): string {
  const negative = raw.startsWith("-");
  const unsigned = negative ? raw.slice(1) : raw;
  const dotIndex = unsigned.indexOf(".");
  const integerPart = dotIndex === -1 ? unsigned : unsigned.slice(0, dotIndex);
  const fractionalPart = dotIndex === -1 ? "" : unsigned.slice(dotIndex);
  return `${negative ? "-" : ""}${groupIntegerPart(integerPart)}${fractionalPart}`;
}

function groupIntegerPart(integerPart: string): string {
  if (integerPart === "") {
    return integerPart;
  }
  const numeric = Number(integerPart);
  return Number.isFinite(numeric) ? GROUPING_FORMAT.format(numeric) : integerPart;
}

/**
 * The inverse of both {@link formatResult} and {@link formatEntry}: strips any grouping commas the
 * display may have added and parses what is left. `Number("-0.")` is `-0`, and neither the display
 * nor the wire should ever carry a negative zero.
 */
export function parseEntry(raw: string): number {
  const value = Number(raw.replace(/,/g, ""));
  return Object.is(value, -0) ? 0 : value;
}

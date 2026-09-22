/**
 * Small, versioned localStorage helper.
 *
 * Every persisted shape gets a key with a version suffix (`calc.<name>.v<version>`), so a schema
 * change is a version bump plus {@link migrateKeys} removing the old key, never a runtime shape
 * mismatch silently corrupting state. Every access is wrapped in try/catch and safe when
 * `window`/`localStorage` is unavailable: private browsing can make `localStorage` throw just by
 * being touched, an SSR/pre-hydration environment has no `window` at all, and quota errors on
 * write are a fact of life — none of these are bugs a caller should have to guard against.
 *
 * A read that finds nothing, malformed JSON, or JSON that fails its zod schema all return `null`
 * (never throw); the schema-mismatch and malformed-JSON cases also log one `console.warn` per key,
 * so a real corruption is visible without spamming the console on every render.
 */
import type { z } from "zod";

const warnedKeys = new Set<string>();

function warnOnce(key: string, message: string): void {
  if (warnedKeys.has(key)) {
    return;
  }
  warnedKeys.add(key);
  console.warn(message);
}

/** `null` for SSR/no-`window`, and for a `localStorage` access that itself throws (private mode). */
function getStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

/** `storageKey("history", 1) → "calc.history.v1"`. Bump `version` whenever the stored shape changes. */
export function storageKey(name: string, version: number): string {
  return `calc.${name}.v${version}`;
}

/** The zod-parsed value at `key`, or `null` when it is missing, malformed JSON, or fails `schema`. */
export function readJSON<T>(key: string, schema: z.ZodType<T>): T | null {
  const storage = getStorage();
  if (storage === null) {
    return null;
  }

  let raw: string | null;
  try {
    raw = storage.getItem(key);
  } catch {
    return null;
  }
  if (raw === null) {
    return null;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    warnOnce(key, `storage: malformed JSON at "${key}"; discarding.`);
    return null;
  }

  const result = schema.safeParse(parsed);
  if (!result.success) {
    warnOnce(key, `storage: value at "${key}" failed schema validation; discarding.`);
    return null;
  }
  return result.data;
}

/** Best-effort write. Silently drops the write on quota errors or a blocked `localStorage`. */
export function writeJSON<T>(key: string, value: T): void {
  const storage = getStorage();
  if (storage === null) {
    return;
  }
  try {
    storage.setItem(key, JSON.stringify(value));
  } catch {
    // Quota exceeded, private-mode block, or a value that fails to serialise: persistence is
    // best-effort and must never crash the app that asked for it.
  }
}

export function removeKey(key: string): void {
  const storage = getStorage();
  if (storage === null) {
    return;
  }
  try {
    storage.removeItem(key);
  } catch {
    // ignore
  }
}

/**
 * Removes every key that starts with `prefix` other than `currentKey` — the migration a version
 * bump needs so a stale `calc.history.v1` does not linger once `v2` is the live key.
 */
export function migrateKeys(prefix: string, currentKey: string): void {
  const storage = getStorage();
  if (storage === null) {
    return;
  }

  const stale: string[] = [];
  try {
    for (let i = 0; i < storage.length; i += 1) {
      const key = storage.key(i);
      if (key !== null && key.startsWith(prefix) && key !== currentKey) {
        stale.push(key);
      }
    }
  } catch {
    return;
  }

  for (const key of stale) {
    removeKey(key);
  }
}

import { z } from "zod";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { migrateKeys, readJSON, removeKey, storageKey, writeJSON } from "@/lib/storage";

const schema = z.object({ n: z.number() });

describe("storageKey", () => {
  it("builds a versioned key", () => {
    expect(storageKey("history", 1)).toBe("calc.history.v1");
    expect(storageKey("history-open", 1)).toBe("calc.history-open.v1");
  });
});

describe("readJSON / writeJSON round-trip", () => {
  it("writes and reads back a value that matches its schema", () => {
    writeJSON("calc.thing.v1", { n: 42 });
    expect(readJSON("calc.thing.v1", schema)).toEqual({ n: 42 });
  });

  it("returns null for a key that was never written", () => {
    expect(readJSON("calc.missing.v1", schema)).toBeNull();
  });

  it("removeKey removes a previously written value", () => {
    writeJSON("calc.thing.v1", { n: 1 });
    removeKey("calc.thing.v1");
    expect(readJSON("calc.thing.v1", schema)).toBeNull();
  });
});

describe("readJSON with malformed JSON", () => {
  it("returns null and warns once", () => {
    localStorage.setItem("calc.bad.v1", "{not json");
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);

    expect(readJSON("calc.bad.v1", schema)).toBeNull();
    expect(readJSON("calc.bad.v1", schema)).toBeNull();

    expect(warn).toHaveBeenCalledTimes(1);
  });
});

describe("readJSON with a schema mismatch", () => {
  it("returns null and warns once", () => {
    localStorage.setItem("calc.mismatch.v1", JSON.stringify({ wrong: "shape" }));
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);

    expect(readJSON("calc.mismatch.v1", schema)).toBeNull();
    expect(readJSON("calc.mismatch.v1", schema)).toBeNull();

    expect(warn).toHaveBeenCalledTimes(1);
  });
});

describe("a throwing localStorage (private mode / quota)", () => {
  let original: Storage;

  beforeEach(() => {
    original = window.localStorage;
  });

  afterEach(() => {
    Object.defineProperty(window, "localStorage", { value: original, configurable: true });
  });

  it("readJSON returns null instead of throwing when getItem throws", () => {
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        getItem: () => {
          throw new DOMException("blocked");
        },
      },
    });
    expect(readJSON("calc.thing.v1", schema)).toBeNull();
  });

  it("writeJSON swallows a quota error instead of throwing", () => {
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        setItem: () => {
          throw new DOMException("QuotaExceededError");
        },
      },
    });
    expect(() => writeJSON("calc.thing.v1", { n: 1 })).not.toThrow();
  });

  it("removeKey swallows an error instead of throwing", () => {
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        removeItem: () => {
          throw new DOMException("blocked");
        },
      },
    });
    expect(() => removeKey("calc.thing.v1")).not.toThrow();
  });

  it("migrateKeys returns without throwing when accessing localStorage throws", () => {
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      get() {
        throw new DOMException("blocked");
      },
    });
    expect(() => migrateKeys("calc.history.v", "calc.history.v2")).not.toThrow();
  });

  it("migrateKeys returns without throwing when iterating localStorage itself throws", () => {
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        length: 3,
        key: () => {
          throw new DOMException("blocked");
        },
      },
    });
    expect(() => migrateKeys("calc.history.v", "calc.history.v2")).not.toThrow();
  });
});

describe("no window (SSR-like environment)", () => {
  it("every function is a safe no-op / returns null", () => {
    const originalWindow = globalThis.window;
    // @ts-expect-error simulating an environment without `window`
    delete globalThis.window;

    try {
      expect(readJSON("calc.thing.v1", schema)).toBeNull();
      expect(() => writeJSON("calc.thing.v1", { n: 1 })).not.toThrow();
      expect(() => removeKey("calc.thing.v1")).not.toThrow();
      expect(() => migrateKeys("calc.history.v", "calc.history.v1")).not.toThrow();
    } finally {
      globalThis.window = originalWindow;
    }
  });
});

describe("migrateKeys", () => {
  it("removes stale keys under the prefix, keeping the current key and unrelated keys", () => {
    localStorage.setItem("calc.history.v1", "[]");
    localStorage.setItem("calc.history.v2", "[]");
    localStorage.setItem("calc.other.v1", "[]");

    migrateKeys("calc.history.v", "calc.history.v2");

    expect(localStorage.getItem("calc.history.v1")).toBeNull();
    expect(localStorage.getItem("calc.history.v2")).toBe("[]");
    expect(localStorage.getItem("calc.other.v1")).toBe("[]");
  });

  it("does nothing when no key matches the prefix", () => {
    localStorage.setItem("calc.other.v1", "[]");
    expect(() => migrateKeys("calc.history.v", "calc.history.v1")).not.toThrow();
    expect(localStorage.getItem("calc.other.v1")).toBe("[]");
  });
});

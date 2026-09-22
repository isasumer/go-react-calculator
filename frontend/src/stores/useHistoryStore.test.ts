import { describe, expect, it, vi } from "vitest";

import type * as HistoryModule from "@/stores/useHistoryStore";

/**
 * The store hydrates from `localStorage` once, at module load — so every test that cares about a
 * particular starting state seeds `localStorage` first and then imports a fresh module instance
 * with `vi.resetModules()`, rather than relying on the already-loaded singleton.
 */
async function freshStore(): Promise<typeof HistoryModule> {
  vi.resetModules();
  return import("@/stores/useHistoryStore");
}

describe("useHistoryStore", () => {
  it("hydrates from a pre-seeded localStorage entry", async () => {
    const seeded = [
      { id: "seed-1", operation: "add", a: 1, b: 2, result: 3, at: "2024-01-01T00:00:00.000Z" },
    ];
    localStorage.setItem("calc.history.v1", JSON.stringify(seeded));

    const { useHistoryStore } = await freshStore();

    expect(useHistoryStore.getState().entries).toEqual(seeded);
  });

  it("hydrates to an empty list when nothing is stored", async () => {
    const { useHistoryStore } = await freshStore();

    expect(useHistoryStore.getState().entries).toEqual([]);
  });

  it("does not throw on corrupted storage and hydrates empty", async () => {
    localStorage.setItem("calc.history.v1", "{not json");

    const { useHistoryStore } = await freshStore();

    expect(useHistoryStore.getState().entries).toEqual([]);
  });

  it("discards a stored value that fails the schema and hydrates empty", async () => {
    localStorage.setItem("calc.history.v1", JSON.stringify([{ not: "an entry" }]));

    const { useHistoryStore } = await freshStore();

    expect(useHistoryStore.getState().entries).toEqual([]);
  });

  it("persists to localStorage on add, remove and clear", async () => {
    const { useHistoryStore } = await freshStore();

    useHistoryStore.getState().add({ operation: "add", a: 1, b: 2, result: 3 });
    expect(JSON.parse(localStorage.getItem("calc.history.v1") ?? "[]")).toHaveLength(1);

    const id = useHistoryStore.getState().entries[0]?.id;
    expect(id).toBeTruthy();

    useHistoryStore.getState().remove(id ?? "");
    expect(JSON.parse(localStorage.getItem("calc.history.v1") ?? "[]")).toHaveLength(0);

    useHistoryStore.getState().add({ operation: "multiply", a: 2, b: 3, result: 6 });
    useHistoryStore.getState().clear();
    expect(JSON.parse(localStorage.getItem("calc.history.v1") ?? "[]")).toEqual([]);
  });

  it("assigns a unique id and an ISO timestamp to a new entry", async () => {
    const { useHistoryStore } = await freshStore();

    useHistoryStore.getState().add({ operation: "sqrt", a: 9, b: null, result: 3 });

    const [entry] = useHistoryStore.getState().entries;
    expect(entry?.id).toBeTruthy();
    expect(entry?.at).toMatch(/^\d{4}-\d{2}-\d{2}T/);
    expect(new Date(entry?.at ?? "").toString()).not.toBe("Invalid Date");
  });

  it("keeps at most 50 entries, dropping the oldest first (newest-first FIFO)", async () => {
    const { useHistoryStore } = await freshStore();

    for (let i = 0; i < 55; i += 1) {
      useHistoryStore.getState().add({ operation: "add", a: i, b: 1, result: i + 1 });
    }

    const { entries } = useHistoryStore.getState();
    expect(entries).toHaveLength(50);
    // Newest first: the last add (a=54) leads; the oldest survivor is a=5 (a=0..4 were dropped).
    expect(entries[0]?.a).toBe(54);
    expect(entries[49]?.a).toBe(5);
  });

  it("falls back to a non-crypto id when crypto.randomUUID is unavailable", async () => {
    const originalCrypto = globalThis.crypto;
    Object.defineProperty(globalThis, "crypto", { value: undefined, configurable: true });

    try {
      const { useHistoryStore } = await freshStore();
      useHistoryStore.getState().add({ operation: "add", a: 1, b: 1, result: 2 });
      const [entry] = useHistoryStore.getState().entries;
      expect(entry?.id).toBeTruthy();
    } finally {
      Object.defineProperty(globalThis, "crypto", { value: originalCrypto, configurable: true });
    }
  });

  it("remove is a no-op for an id that is not present", async () => {
    const { useHistoryStore } = await freshStore();

    useHistoryStore.getState().add({ operation: "add", a: 1, b: 1, result: 2 });
    useHistoryStore.getState().remove("does-not-exist");

    expect(useHistoryStore.getState().entries).toHaveLength(1);
  });
});

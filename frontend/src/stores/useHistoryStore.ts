/**
 * Recent calculations, persisted to `localStorage` (F2-05, #16).
 *
 * Hand-rolled persistence rather than `zustand/persist`: `persist` would own the storage key and
 * (de)serialisation, but the reference codebase keeps that explicit — {@link storage} is the one
 * place every persisted shape goes through, versioned and zod-validated on the way back in. This
 * store hydrates once, from `readJSON` at module load, and writes back through a manual `subscribe`
 * after every change.
 *
 * `useCalculatorStore` knows nothing about history: `Calculator` is the only place that reads a
 * successful `CalculateResponse` and calls {@link HistoryStore.add}, which keeps the calculator
 * engine's store free of a concern that is purely about what the UI remembers.
 */
import { z } from "zod";
import { create } from "zustand";

import { migrateKeys, readJSON, storageKey, writeJSON } from "@/lib/storage";
import { operationNameSchema } from "@/types/calculator";

/** Bump this and the key it produces whenever `historyEntrySchema` changes shape. */
const HISTORY_VERSION = 1;
const HISTORY_KEY = storageKey("history", HISTORY_VERSION);
const HISTORY_KEY_PREFIX = "calc.history.v";

/** Oldest entries are dropped first once the list would exceed this. */
export const MAX_HISTORY_ENTRIES = 50;

export const historyEntrySchema = z.object({
  id: z.string().min(1),
  operation: operationNameSchema,
  a: z.number(),
  /** `null` for a unary operation (`sqrt`), which never sends a second operand. */
  b: z.number().nullable(),
  result: z.number(),
  /** ISO 8601 timestamp of when the entry was recorded. */
  at: z.string().min(1),
});

export type HistoryEntry = z.infer<typeof historyEntrySchema>;

const historyListSchema = z.array(historyEntrySchema);

/** What a caller supplies; `id` and `at` are assigned by {@link HistoryStore.add}. */
export interface NewHistoryEntry {
  readonly operation: HistoryEntry["operation"];
  readonly a: number;
  readonly b: number | null;
  readonly result: number;
}

export interface HistoryStore {
  /** Newest first. */
  readonly entries: readonly HistoryEntry[];
  add: (entry: NewHistoryEntry) => void;
  remove: (id: string) => void;
  clear: () => void;
}

/** `crypto.randomUUID` is unavailable in some older/sandboxed environments; this never throws. */
function makeId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

function loadInitialEntries(): HistoryEntry[] {
  // Discard any key from a previous schema version before reading the current one, so a stale
  // `v0`/`v1` blob never leaks into a later version's store.
  migrateKeys(HISTORY_KEY_PREFIX, HISTORY_KEY);
  return readJSON(HISTORY_KEY, historyListSchema) ?? [];
}

export const useHistoryStore = create<HistoryStore>()((set, get) => ({
  entries: loadInitialEntries(),

  add: (entry) => {
    const withMeta: HistoryEntry = { ...entry, id: makeId(), at: new Date().toISOString() };
    // Newest first; slicing from the front after unshifting keeps the newest `MAX_HISTORY_ENTRIES`
    // and drops the oldest ones, which sit at the end of the list.
    const entries = [withMeta, ...get().entries].slice(0, MAX_HISTORY_ENTRIES);
    set({ entries });
  },

  remove: (id) => {
    set({ entries: get().entries.filter((entry) => entry.id !== id) });
  },

  clear: () => {
    set({ entries: [] });
  },
}));

// Persist after every change. A hand-rolled `subscribe` rather than `zustand/persist` middleware,
// so the storage key and schema stay explicit in `storage.ts` instead of hidden in a wrapper.
useHistoryStore.subscribe((state) => {
  writeJSON(HISTORY_KEY, state.entries);
});

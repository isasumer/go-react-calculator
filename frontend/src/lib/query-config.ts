/**
 * Endpoints, TanStack Query keys and cache lifetimes. Everything that names a route or a cache entry
 * lives here so a path is never spelled twice.
 */

/** `POST` — evaluate one operation. Relative to `getConfig().apiBaseUrl` (default `/api`). */
export const CALCULATE_ENDPOINT = "/v1/calculate";

/** `GET` — the operation registry the UI labels its keys from. */
export const OPERATIONS_ENDPOINT = "/v1/operations";

/**
 * Query keys. Always build them through this object so an invalidation cannot miss a cache entry
 * because of a typo.
 */
export const calculatorKeys = {
  all: ["calculator"] as const,
  operations: () => [...calculatorKeys.all, "operations"] as const,
} as const;

/** How long cached data stays fresh, in milliseconds. */
export const STALE_TIME = {
  /** Default for queries that have no better answer. */
  default: 60_000,
  /**
   * The registry only changes when the backend is redeployed, and the page is reloaded then anyway,
   * so it never goes stale within a session.
   */
  operations: Number.POSITIVE_INFINITY,
} as const;

/** Attempts per failed query. Mutations do not retry: the user presses the key again. */
export const QUERY_RETRY = 1;
export const MUTATION_RETRY = 0;

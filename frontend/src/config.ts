/**
 * The only module that reads `import.meta.env`. Everything else asks `getConfig()` for typed values.
 */

export interface Config {
  /** Base URL for API requests, without a trailing slash: "/api" or "https://host/api". */
  readonly apiBaseUrl: string;
}

export const DEFAULT_API_BASE_URL = "/api";

export class ConfigError extends Error {
  override name = "ConfigError";
}

type RawEnv = Readonly<Record<string, string | boolean | undefined>>;

function parseApiBaseUrl(raw: RawEnv[string]): string {
  if (raw === undefined || raw === "") {
    return DEFAULT_API_BASE_URL;
  }
  if (typeof raw !== "string") {
    throw new ConfigError("VITE_API_BASE_URL must be a string");
  }
  const value = raw.trim();
  const isPath = value.startsWith("/") && !value.startsWith("//");
  if (!isPath && !URL.canParse(value)) {
    throw new ConfigError(
      `VITE_API_BASE_URL must be an absolute path ("/api") or an http(s) URL, got ${JSON.stringify(raw)}`,
    );
  }
  if (!isPath && !/^https?:$/.test(new URL(value).protocol)) {
    throw new ConfigError(`VITE_API_BASE_URL must use http or https, got ${JSON.stringify(raw)}`);
  }
  // Strip trailing slashes so callers can always write `${apiBaseUrl}/v1/...`; keep "/" as-is.
  return value.replace(/\/+$/, "") || "/";
}

/** Validate a raw env object into a Config. Pure; exported for tests. */
export function parseConfig(env: RawEnv): Config {
  return Object.freeze({
    apiBaseUrl: parseApiBaseUrl(env["VITE_API_BASE_URL"]),
  });
}

let cached: Config | undefined;

/** Typed, validated runtime config. Parsed on first call and cached; throws ConfigError if invalid. */
export function getConfig(): Config {
  cached ??= parseConfig(import.meta.env);
  return cached;
}

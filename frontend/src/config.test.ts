import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConfigError, DEFAULT_API_BASE_URL, parseConfig } from "@/config";

describe("getConfig", () => {
  beforeEach(() => {
    // getConfig caches per module instance; a fresh import sees the stubbed env.
    vi.resetModules();
  });

  it('defaults VITE_API_BASE_URL to "/api"', async () => {
    vi.stubEnv("VITE_API_BASE_URL", undefined);
    const { getConfig } = await import("@/config");

    expect(getConfig().apiBaseUrl).toBe("/api");
  });

  it("uses VITE_API_BASE_URL when set", async () => {
    vi.stubEnv("VITE_API_BASE_URL", "http://localhost:8081/api");
    const { getConfig } = await import("@/config");

    expect(getConfig().apiBaseUrl).toBe("http://localhost:8081/api");
  });

  it("validates once and returns the same frozen object", async () => {
    vi.stubEnv("VITE_API_BASE_URL", "/api");
    const { getConfig } = await import("@/config");

    const first = getConfig();
    vi.stubEnv("VITE_API_BASE_URL", "/changed");

    expect(getConfig()).toBe(first);
    expect(Object.isFrozen(first)).toBe(true);
  });
});

describe("parseConfig", () => {
  it.each([
    { name: "missing", env: {}, want: DEFAULT_API_BASE_URL },
    { name: "empty string", env: { VITE_API_BASE_URL: "" }, want: DEFAULT_API_BASE_URL },
    { name: "relative path", env: { VITE_API_BASE_URL: "/backend" }, want: "/backend" },
    { name: "trailing slash", env: { VITE_API_BASE_URL: "/api/" }, want: "/api" },
    { name: "root path", env: { VITE_API_BASE_URL: "/" }, want: "/" },
    { name: "surrounding space", env: { VITE_API_BASE_URL: " /api " }, want: "/api" },
    {
      name: "https URL",
      env: { VITE_API_BASE_URL: "https://calc.example.com/api/" },
      want: "https://calc.example.com/api",
    },
  ])("accepts $name", ({ env, want }) => {
    expect(parseConfig(env).apiBaseUrl).toBe(want);
  });

  it.each([
    { name: "bare word", value: "api" },
    { name: "protocol-relative URL", value: "//evil.example.com" },
    { name: "non-http scheme", value: "ftp://example.com/api" },
    { name: "javascript scheme", value: "javascript:alert(1)" },
    { name: "boolean", value: true },
  ])("rejects $name", ({ value }) => {
    expect(() => parseConfig({ VITE_API_BASE_URL: value })).toThrow(ConfigError);
  });
});

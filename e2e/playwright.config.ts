import { defineConfig, devices } from "@playwright/test";

/**
 * Runs against the composed stack (nginx + Go service), not a dev server — see
 * `../compose.yaml` and `../compose.e2e.yaml`. `E2E_BASE_URL` lets a local run point at
 * `FRONTEND_PORT=18080` when 8080 is already taken; CI leaves it at the default.
 */
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:8080";

export default defineConfig({
  testDir: "./tests",
  // The backend's rate limiter (RATE_LIMIT_RPS/BURST) is shared across the whole compose stack,
  // so specs running in parallel workers would trip 429s meant to catch real abuse. One worker,
  // one spec file at a time.
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [["html", { outputFolder: "playwright-report", open: "never" }], ["list"]],
  outputDir: "test-results",
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  expect: {
    timeout: 5_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      // responsive.spec.ts asserts mobile-viewport behaviour and only needs to run once, on the
      // mobile-safari project below.
      testIgnore: /responsive\.spec\.ts/,
    },
    {
      name: "mobile-safari",
      // WebKit under Playwright's "iPhone 13" device profile. If WebKit is not installed
      // locally, run `npx playwright install webkit` (see e2e/Makefile's `install` target).
      use: { ...devices["iPhone 13"] },
      testMatch: /responsive\.spec\.ts/,
    },
  ],
});

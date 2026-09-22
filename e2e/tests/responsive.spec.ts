/**
 * Mobile-viewport assertions only — this file is matched to the `mobile-safari` project alone
 * (see `../playwright.config.ts`), which already runs at the "iPhone 13" viewport. We additionally
 * pin the classic small-phone size the ticket calls out (320×568, iPhone SE 1st gen) so a
 * regression there is caught even as device presets change.
 */
import { expect, test } from "@playwright/test";

test.use({ viewport: { width: 320, height: 568 } });

test("no horizontal overflow at 320x568", async ({ page }) => {
  await page.goto("/");

  const overflow = await page.evaluate(() => {
    const root = document.documentElement;
    return { scrollWidth: root.scrollWidth, clientWidth: root.clientWidth };
  });
  expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.clientWidth);
});

test("every key is at least 44px tall", async ({ page }) => {
  await page.goto("/");

  // "Keys" means the keypad and the operations bar (√, %, xʸ) — the two `role="group"`s the
  // touch-target requirement is actually about — not the small recall/remove rows in the
  // history panel, which is a list, not a set of keys.
  const keys = page
    .getByRole("group", { name: "Keypad" })
    .getByRole("button")
    .or(page.getByRole("group", { name: "More operations" }).getByRole("button"));

  const heights = await keys.evaluateAll((buttons) =>
    buttons.map((button) => button.getBoundingClientRect().height),
  );
  expect(heights.length).toBeGreaterThan(0);
  for (const height of heights) {
    expect(height).toBeGreaterThanOrEqual(44);
  }
});

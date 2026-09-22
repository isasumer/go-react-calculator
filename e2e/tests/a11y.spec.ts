import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

import { clickDigits, equals, errorAlert, operator } from "./helpers";

/** Only serious/critical violations fail the build; the ticket asks for an a11y smoke test, not a
 *  full audit gate (that belongs to a dedicated a11y ticket). */
function seriousOrCritical(results: Awaited<ReturnType<AxeBuilder["analyze"]>>) {
  return results.violations.filter((v) => v.impact === "serious" || v.impact === "critical");
}

test("initial screen has no serious/critical accessibility violations", async ({ page }) => {
  await page.goto("/");
  const results = await new AxeBuilder({ page }).analyze();
  expect(seriousOrCritical(results)).toEqual([]);
});

test("screen with a visible error has no serious/critical accessibility violations", async ({
  page,
}) => {
  await page.goto("/");
  await clickDigits(page, "5");
  await operator(page, "divide").click();
  await clickDigits(page, "0");
  await equals(page).click();
  await expect(errorAlert(page)).toContainText("Cannot divide by zero.");

  const results = await new AxeBuilder({ page }).analyze();
  expect(seriousOrCritical(results)).toEqual([]);
});

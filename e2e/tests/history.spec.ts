import { expect, test, type Page } from "@playwright/test";

import { clickDigits, equals, historyRecallButton, historyRows, operator } from "./helpers";

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

async function calculate(page: Page, a: string, b: string) {
  await clickDigits(page, a);
  await operator(page, "add").click();
  await clickDigits(page, b);
  await equals(page).click();
}

test("history lists calculations newest first, survives reload, recalls, and clears in two steps", async ({
  page,
}) => {
  await calculate(page, "1", "1"); // 1 + 1 = 2
  await calculate(page, "3", "3"); // 3 + 3 = 6

  const rows = historyRows(page);
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText("3 + 3 = 6");
  await expect(rows.nth(1)).toContainText("1 + 1 = 2");

  await page.reload();
  await expect(historyRows(page)).toHaveCount(2);
  await expect(historyRows(page).nth(0)).toContainText("3 + 3 = 6");

  await historyRecallButton(page, "1 + 1 = 2").click();
  await expect(page.getByRole("status")).toHaveText("2");

  // Two-step clear: first click arms it, second click (same button, new label) confirms.
  await page.getByRole("button", { name: "Clear history", exact: true }).click();
  await page.getByRole("button", { name: "Confirm clear", exact: true }).click();
  await expect(historyRows(page)).toHaveCount(0);
});

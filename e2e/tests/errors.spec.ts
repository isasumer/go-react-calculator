import { expect, test } from "@playwright/test";

import { clickDigits, digit, equals, errorAlert, expectDisplay, operator } from "./helpers";

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test("5 / 0 = shows the division-by-zero message, and the next digit clears it", async ({
  page,
}) => {
  await clickDigits(page, "5");
  await operator(page, "divide").click();
  await clickDigits(page, "0");
  await equals(page).click();

  await expect(errorAlert(page)).toContainText("Cannot divide by zero.");

  // A fresh entry is how the user gets past the error; the reserved alert line goes quiet again.
  await digit(page, "1").click();
  await expect(errorAlert(page)).not.toContainText("Cannot divide by zero.");
  await expectDisplay(page, "1");
});

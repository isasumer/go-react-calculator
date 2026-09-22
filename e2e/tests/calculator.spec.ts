import { test } from "@playwright/test";

import { clickDigits, display, equals, expectDisplay, operator, unary } from "./helpers";

test.beforeEach(async ({ page }) => {
  await page.goto("/");
  // Synchronise on the app actually being interactive before driving it: the keyboard tests in
  // particular dispatch real key events, which are lost if they land before React has mounted
  // and `useKeyboard` has attached its listener.
  await display(page).waitFor({ state: "visible" });
});

test("click flow: 12 + 7 = 19", async ({ page }) => {
  await clickDigits(page, "12");
  await operator(page, "add").click();
  await clickDigits(page, "7");
  await equals(page).click();
  await expectDisplay(page, "19");
});

test("keyboard flow: 12*3<Enter> = 36", async ({ page }) => {
  // `page.keyboard.press` one key at a time, not `.type()`: the app's key handler listens on
  // `keydown`, and Chromium's `insertText` fast path for `.type()` does not always fire it for
  // punctuation like `*`.
  for (const key of ["1", "2", "*", "3", "Enter"]) {
    await page.keyboard.press(key);
  }
  await expectDisplay(page, "36");
});

test("chained operators execute immediately: 2 + 3 x 4 = 20", async ({ page }) => {
  // This calculator has no operator precedence: pressing a second operator evaluates the pending
  // one against what has been entered so far before starting the next, exactly like a basic
  // pocket calculator. So `2 + 3 × 4 =` is `(2 + 3) × 4`, not `2 + (3 × 4)`.
  await clickDigits(page, "2");
  await operator(page, "add").click();
  await clickDigits(page, "3");
  await operator(page, "multiply").click();
  await clickDigits(page, "4");
  await equals(page).click();
  await expectDisplay(page, "20");
});

test("square root of 16 is 4, via the operations bar", async ({ page }) => {
  await clickDigits(page, "16");
  await unary(page, "square root").click();
  await expectDisplay(page, "4");
});

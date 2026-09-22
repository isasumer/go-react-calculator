/**
 * Shared page-object-ish helpers for the calculator specs. Everything here queries by role,
 * accessible name or (for the display's live/alert regions) implicit role — never by CSS class.
 * `data-ui` is not used at all: every element this suite needs already has a role and a name.
 */
import { expect, type Locator, type Page } from "@playwright/test";

/** The result/entry line: an `<output>`, whose implicit ARIA role is "status". */
export function display(page: Page): Locator {
  return page.getByRole("status");
}

/** The reserved error line: always present, `role="alert"`, empty text when there is no error. */
export function errorAlert(page: Page): Locator {
  return page.getByRole("alert");
}

export function digit(page: Page, d: string): Locator {
  return page.getByRole("button", { name: d, exact: true });
}

export function operator(
  page: Page,
  name: "add" | "subtract" | "multiply" | "divide",
): Locator {
  return page.getByRole("button", { name, exact: true });
}

export function equals(page: Page): Locator {
  return page.getByRole("button", { name: "equals", exact: true });
}

export function clearAll(page: Page): Locator {
  return page.getByRole("button", { name: "clear all", exact: true });
}

export function clearEntry(page: Page): Locator {
  return page.getByRole("button", { name: "clear entry", exact: true });
}

export function unary(page: Page, name: "square root" | "percent"): Locator {
  return page.getByRole("button", { name, exact: true });
}

/** Clicks one digit key per character; only digits (no sign/decimal) are needed by this suite. */
export async function clickDigits(page: Page, value: string): Promise<void> {
  for (const ch of value) {
    await digit(page, ch).click();
  }
}

export async function expectDisplay(page: Page, value: string): Promise<void> {
  await expect(display(page)).toHaveText(value);
}

/** The history list (an unordered list of recent calculations, newest first). */
export function historyList(page: Page): Locator {
  return page.getByRole("list");
}

export function historyRows(page: Page): Locator {
  return historyList(page).getByRole("listitem");
}

export function historyRecallButton(page: Page, text: string | RegExp): Locator {
  return historyRows(page).getByRole("button", { name: text, exact: typeof text === "string" });
}

import { describe, expect, it } from "vitest";

import { cn } from "@/lib/utils";

describe("cn", () => {
  it.each([
    { name: "joins plain classes", input: ["a", "b"], want: "a b" },
    { name: "drops falsy values", input: ["a", false, null, undefined, "", "b"], want: "a b" },
    { name: "later tailwind utility wins", input: ["px-2", "px-4"], want: "px-4" },
    { name: "supports object syntax", input: [{ a: true, b: false }], want: "a" },
  ])("$name", ({ input, want }) => {
    expect(cn(...input)).toBe(want);
  });
});

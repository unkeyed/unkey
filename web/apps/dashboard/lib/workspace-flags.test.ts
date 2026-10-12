import { expect, it } from "vitest";
import { useFlag } from "./workspace-flags";

it("preserves an explicit false instead of using a true fallback", () => {
  expect(
    useFlag({
      workspace: { flags: { preview: false } },
      flag: { slug: "preview", defaultValue: true },
    }),
  ).toBe(false);
});

it.each([
  { workspace: { flags: { preview: true } }, name: "preview", default: false, expected: true },
  { workspace: { flags: { other: false } }, name: "preview", default: true, expected: true },
  { workspace: null, name: "preview", default: false, expected: false },
  { workspace: undefined, name: "preview", default: true, expected: true },
])("resolves $name with workspace $workspace and default $default", ({ expected, ...options }) => {
  expect(
    useFlag({
      workspace: options.workspace,
      flag: { slug: options.name, defaultValue: options.default },
    }),
  ).toBe(expected);
});

it("uses the fallback for an absent slug that matches an object prototype property", () => {
  expect(
    useFlag({ workspace: { flags: {} }, flag: { slug: "constructor", defaultValue: false } }),
  ).toBe(false);
});

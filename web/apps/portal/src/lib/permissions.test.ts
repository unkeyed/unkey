import { describe, expect, it } from "vitest";
import { canReadKeys, getDefaultTabHref } from "./permissions";

describe("canReadKeys", () => {
  it("is true when keys:read is present", () => {
    expect(canReadKeys(["keys:read"])).toBe(true);
  });

  it("is true alongside other capabilities", () => {
    expect(canReadKeys(["keys:read", "keys:reroll", "analytics:read"])).toBe(true);
  });

  it("is false when only other keys capabilities are present", () => {
    expect(canReadKeys(["keys:reroll", "keys:create"])).toBe(false);
  });

  it("is false for unrelated capabilities", () => {
    expect(canReadKeys(["analytics:read"])).toBe(false);
  });

  it("is false for empty permissions", () => {
    expect(canReadKeys([])).toBe(false);
  });
});

describe("getDefaultTabHref", () => {
  it("lands on the keys page when the session can read keys", () => {
    expect(getDefaultTabHref(["keys:read"])).toBe("/keys");
  });

  it("ignores deferred analytics capability when keys is absent", () => {
    // Analytics is deferred to v2, so analytics:read no longer grants a landing
    // destination even though the session carries it.
    expect(getDefaultTabHref(["analytics:read"])).toBeNull();
  });

  it("is null when the session can't read keys", () => {
    expect(getDefaultTabHref(["keys:reroll"])).toBeNull();
  });

  it("is null for empty permissions", () => {
    expect(getDefaultTabHref([])).toBeNull();
  });
});

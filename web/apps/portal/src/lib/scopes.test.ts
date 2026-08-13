import { describe, expect, it } from "vitest";
import { canReadKeys, getDefaultTabHref } from "./scopes";

describe("canReadKeys", () => {
  it("is true when keys:read is present", () => {
    expect(canReadKeys(["keys:read"])).toBe(true);
  });

  it("is true alongside other scopes", () => {
    expect(canReadKeys(["keys:read", "keys:reroll", "analytics:read"])).toBe(true);
  });

  it("is false when only other keys scopes are present", () => {
    expect(canReadKeys(["keys:reroll", "keys:create"])).toBe(false);
  });

  it("is false for unrelated scopes", () => {
    expect(canReadKeys(["analytics:read"])).toBe(false);
  });

  it("is false for empty scopes", () => {
    expect(canReadKeys([])).toBe(false);
  });
});

describe("getDefaultTabHref", () => {
  it("lands on the keys page when the session can read keys", () => {
    expect(getDefaultTabHref(["keys:read"])).toBe("/keys");
  });

  it("ignores deferred analytics scope when keys is absent", () => {
    // Analytics is deferred to v2, so analytics:read no longer grants a landing
    // destination even though the session carries it.
    expect(getDefaultTabHref(["analytics:read"])).toBeNull();
  });

  it("is null when the session can't read keys", () => {
    expect(getDefaultTabHref(["keys:reroll"])).toBeNull();
  });

  it("is null for empty scopes", () => {
    expect(getDefaultTabHref([])).toBeNull();
  });
});

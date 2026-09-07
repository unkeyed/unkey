import { describe, expect, it } from "vitest";
import {
  canReadAnalytics,
  canReadKeys,
  canRerollKeys,
  deriveVisibleTabs,
  getDefaultTabHref,
} from "./scopes";

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

describe("canRerollKeys", () => {
  it("is true when keys:reroll is present", () => {
    expect(canRerollKeys(["keys:read", "keys:reroll"])).toBe(true);
  });

  // portal.rerollKey rejects this session, so the table must not offer it.
  it("is false for a read-only session", () => {
    expect(canRerollKeys(["keys:read"])).toBe(false);
  });

  it("is false for empty scopes", () => {
    expect(canRerollKeys([])).toBe(false);
  });
});

describe("canReadAnalytics", () => {
  it("is true when analytics:read is present", () => {
    expect(canReadAnalytics(["keys:read", "analytics:read"])).toBe(true);
  });

  it("is false for a keys-only session", () => {
    expect(canReadAnalytics(["keys:read", "keys:reroll"])).toBe(false);
  });

  it("is false for empty scopes", () => {
    expect(canReadAnalytics([])).toBe(false);
  });
});

describe("deriveVisibleTabs", () => {
  it("lists both tabs in navigation order", () => {
    expect(deriveVisibleTabs(["analytics:read", "keys:read"])).toEqual([
      { id: "keys", label: "API keys", href: "/keys" },
      { id: "analytics", label: "Analytics", href: "/analytics" },
    ]);
  });

  it("omits the analytics tab without analytics:read", () => {
    expect(deriveVisibleTabs(["keys:read"])).toEqual([
      { id: "keys", label: "API keys", href: "/keys" },
    ]);
  });

  it("omits the keys tab without keys:read", () => {
    expect(deriveVisibleTabs(["analytics:read"])).toEqual([
      { id: "analytics", label: "Analytics", href: "/analytics" },
    ]);
  });

  it("is empty for empty scopes", () => {
    expect(deriveVisibleTabs([])).toEqual([]);
  });
});

describe("getDefaultTabHref", () => {
  it("lands on the keys page when the session can read keys", () => {
    expect(getDefaultTabHref(["keys:read"])).toBe("/keys");
  });

  it("prefers keys over analytics", () => {
    expect(getDefaultTabHref(["analytics:read", "keys:read"])).toBe("/keys");
  });

  it("falls back to analytics when keys is absent", () => {
    expect(getDefaultTabHref(["analytics:read"])).toBe("/analytics");
  });

  it("is null when the session can reach no page", () => {
    expect(getDefaultTabHref(["keys:reroll"])).toBeNull();
  });

  it("is null for empty scopes", () => {
    expect(getDefaultTabHref([])).toBeNull();
  });
});

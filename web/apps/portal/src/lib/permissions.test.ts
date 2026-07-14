import { describe, expect, it } from "vitest";
import { canReadKeys, deriveVisibleTabs } from "./permissions";

describe("deriveVisibleTabs", () => {
  it("shows Keys and Docs tabs for a keys capability", () => {
    const tabs = deriveVisibleTabs(["keys:read"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("analytics");
  });

  it("hides Keys tab for a keys capability that can't read (e.g. reroll only)", () => {
    // The keys page's access guard requires keys:read, so the tab must not show
    // for a reroll-only session or clicking it would redirect to a dead end.
    const tabs = deriveVisibleTabs(["keys:reroll"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).not.toContain("keys");
    expect(ids).not.toContain("analytics");
    expect(ids).toContain("docs");
  });

  it("shows Analytics and Docs tabs for analytics capability", () => {
    const tabs = deriveVisibleTabs(["analytics:read"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("analytics");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("keys");
  });

  it("shows Keys, Analytics, and Docs tabs for all capabilities", () => {
    const tabs = deriveVisibleTabs(["keys:read", "keys:reroll", "analytics:read"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("analytics");
    expect(ids).toContain("docs");
  });

  it("shows only Docs tab for a capability that matches no tab", () => {
    const tabs = deriveVisibleTabs(["something:else"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toEqual(["docs"]);
  });

  it("returns no tabs for empty permissions array", () => {
    const tabs = deriveVisibleTabs([]);

    expect(tabs).toHaveLength(0);
  });
});

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

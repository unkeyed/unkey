import { describe, expect, it } from "vitest";
import { portalSessionGrantSchema } from "./capabilities";
import { deriveVisibleTabs } from "./permissions";

describe("deriveVisibleTabs", () => {
  it("shows Keys and Docs tabs for keys:read", () => {
    const tabs = deriveVisibleTabs(["keys:read"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("analytics");
  });

  it("shows Analytics and Docs tabs for analytics:read", () => {
    const tabs = deriveVisibleTabs(["analytics:read"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("analytics");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("keys");
  });

  it("shows Keys and Docs tabs for non-read key capabilities", () => {
    const tabs = deriveVisibleTabs(["keys:create", "keys:reroll"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("analytics");
  });

  it("returns no tabs for empty permissions array", () => {
    const tabs = deriveVisibleTabs([]);

    expect(tabs).toHaveLength(0);
  });
});

describe("portalSessionGrantSchema", () => {
  it("accepts the persisted portal grant", () => {
    expect(
      portalSessionGrantSchema.safeParse({
        keyspaceIds: ["ks_123"],
        permissions: ["keys:read", "analytics:read"],
      }).success,
    ).toBe(true);
  });

  it("rejects malformed and unknown grants", () => {
    expect(portalSessionGrantSchema.safeParse(["keys:read"]).success).toBe(false);
    expect(
      portalSessionGrantSchema.safeParse({
        keyspaceIds: ["ks_123"],
        permissions: ["unknown:capability"],
      }).success,
    ).toBe(false);
  });
});

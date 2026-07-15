import { describe, expect, it } from "vitest";
import { deriveVisibleTabs } from "./permissions";

describe("deriveVisibleTabs", () => {
  it("shows Keys and Docs tabs for key read permission", () => {
    const tabs = deriveVisibleTabs(["api.*.read_key"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("analytics");
  });

  it("shows Analytics and Docs tabs for analytics permission", () => {
    const tabs = deriveVisibleTabs(["api.*.read_analytics"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("analytics");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("keys");
  });

  it("shows Keys, Analytics, and Docs tabs for all permissions", () => {
    const tabs = deriveVisibleTabs(["api.*.read_key", "api.*.create_key", "api.*.read_analytics"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("analytics");
    expect(ids).toContain("docs");
  });

  it("shows Keys and Docs tabs for specific resource ID", () => {
    const tabs = deriveVisibleTabs(["api.api_123.read_key"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toContain("keys");
    expect(ids).toContain("docs");
    expect(ids).not.toContain("analytics");
  });

  it("shows only Docs tab for non-matching action", () => {
    const tabs = deriveVisibleTabs(["ratelimit.*.limit"]);
    const ids = tabs.map((t) => t.id);

    expect(ids).toEqual(["docs"]);
  });

  it("shows only Docs tab for malformed permission with fewer than 3 segments", () => {
    const tabs = deriveVisibleTabs(["keys:read"]);
    const ids = tabs.map((t) => t.id);

    // Present in array (length > 0) so Docs is visible, but no action segment matches
    expect(ids).toEqual(["docs"]);
  });

  it("returns no tabs for empty permissions array", () => {
    const tabs = deriveVisibleTabs([]);

    expect(tabs).toHaveLength(0);
  });
});

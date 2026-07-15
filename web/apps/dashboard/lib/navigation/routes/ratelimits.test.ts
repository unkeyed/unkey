import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const namespaceId = "ns_123";

describe("ratelimit-scoped paths", () => {
  it("builds the list path", () => {
    expect(routes.ratelimits.list({ workspaceSlug: ws })).toBe("/acme/ratelimits");
  });

  it("builds the namespace base path", () => {
    expect(routes.ratelimits.detail({ workspaceSlug: ws, namespaceId })).toBe(
      "/acme/ratelimits/ns_123",
    );
  });

  it("builds namespace leaf paths", () => {
    const scope = { workspaceSlug: ws, namespaceId };
    expect(routes.ratelimits.logs(scope)).toBe("/acme/ratelimits/ns_123/logs");
    expect(routes.ratelimits.settings(scope)).toBe("/acme/ratelimits/ns_123/settings");
    expect(routes.ratelimits.overrides(scope)).toBe("/acme/ratelimits/ns_123/overrides");
  });
});

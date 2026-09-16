import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const namespaceId = "ns_123";
const projectId = "proj_123";

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

describe("project-scoped ratelimit paths", () => {
  it("builds the list and namespace base paths inside the project", () => {
    expect(routes.ratelimits.list({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/ratelimits",
    );
    expect(routes.ratelimits.detail({ workspaceSlug: ws, projectId, namespaceId })).toBe(
      "/acme/projects/proj_123/ratelimits/ns_123",
    );
  });

  it("builds namespace leaf paths inside the project", () => {
    const scope = { workspaceSlug: ws, projectId, namespaceId };
    expect(routes.ratelimits.logs(scope)).toBe("/acme/projects/proj_123/ratelimits/ns_123/logs");
    expect(routes.ratelimits.settings(scope)).toBe(
      "/acme/projects/proj_123/ratelimits/ns_123/settings",
    );
    expect(routes.ratelimits.overrides(scope)).toBe(
      "/acme/projects/proj_123/ratelimits/ns_123/overrides",
    );
  });

  it("keeps workspace paths when projectId is absent", () => {
    expect(routes.ratelimits.detail({ workspaceSlug: ws, projectId: undefined, namespaceId })).toBe(
      "/acme/ratelimits/ns_123",
    );
  });
});

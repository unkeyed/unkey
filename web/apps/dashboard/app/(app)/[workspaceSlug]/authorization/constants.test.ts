import { describe, expect, it } from "vitest";
import { navigation } from "./constants";

describe("authorization rail", () => {
  it("links to the workspace pages when no project is in scope", () => {
    expect(navigation({ workspaceSlug: "acme" })).toEqual([
      { segment: "roles", label: "Roles", href: "/acme/authorization/roles" },
      { segment: "permissions", label: "Permissions", href: "/acme/authorization/permissions" },
    ]);
  });

  it("stays inside the project when one is in scope", () => {
    expect(navigation({ workspaceSlug: "acme", projectId: "proj_123" })).toEqual([
      {
        segment: "roles",
        label: "Roles",
        href: "/acme/projects/proj_123/authorization/roles",
      },
      {
        segment: "permissions",
        label: "Permissions",
        href: "/acme/projects/proj_123/authorization/permissions",
      },
    ]);
  });
});

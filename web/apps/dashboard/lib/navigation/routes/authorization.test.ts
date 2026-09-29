import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";

describe("authorization-scoped paths", () => {
  it("builds the leaf paths", () => {
    const scope = { workspaceSlug: ws };
    expect(routes.authorization.roles(scope)).toBe("/acme/authorization/roles");
    expect(routes.authorization.permissions(scope)).toBe("/acme/authorization/permissions");
  });

  it("stays inside a project when one is in scope", () => {
    const scope = { workspaceSlug: ws, projectId: "proj_123" };
    expect(routes.authorization.roles(scope)).toBe("/acme/projects/proj_123/authorization/roles");
    expect(routes.authorization.permissions(scope)).toBe(
      "/acme/projects/proj_123/authorization/permissions",
    );
  });
});

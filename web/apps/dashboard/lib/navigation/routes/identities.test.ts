import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const identityId = "identity_123";
const projectId = "proj_123";

describe("identity-scoped paths", () => {
  it("builds the list path", () => {
    expect(routes.identities.list({ workspaceSlug: ws })).toBe("/acme/identities");
  });

  it("builds the detail path", () => {
    expect(routes.identities.detail({ workspaceSlug: ws, identityId })).toBe(
      "/acme/identities/identity_123",
    );
  });
});

describe("project-scoped identity paths", () => {
  it("builds the list path inside the project", () => {
    expect(routes.identities.list({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/identities",
    );
  });

  it("builds the detail path inside the project", () => {
    expect(routes.identities.detail({ workspaceSlug: ws, projectId, identityId })).toBe(
      "/acme/projects/proj_123/identities/identity_123",
    );
  });

  it("keeps workspace paths when projectId is absent", () => {
    expect(routes.identities.detail({ workspaceSlug: ws, projectId: undefined, identityId })).toBe(
      "/acme/identities/identity_123",
    );
  });
});

import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const identityId = "identity_123";

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

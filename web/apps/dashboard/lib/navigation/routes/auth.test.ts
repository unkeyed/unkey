import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("auth paths", () => {
  it("drops the optional catch-all to yield the base sign-in path", () => {
    expect(routes.auth.signIn()).toBe("/auth/sign-in");
  });

  it("builds a safe organization-switch document route", () => {
    expect(
      routes.auth.switchOrganization({
        organizationId: "org_123",
        returnTo: routes.apis.list({ workspaceSlug: "acme" }),
      }),
    ).toBe("/auth/switch-organization?organization_id=org_123&return_to=/acme/apis");
  });
});

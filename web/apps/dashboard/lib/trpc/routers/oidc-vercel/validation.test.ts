import { describe, expect, it } from "vitest";
import { oidcVercelInputSchema } from "./validation";

describe("Vercel OIDC integration validation", () => {
  it("accepts verb-only URNs produced by the policy builder", () => {
    expect(
      oidcVercelInputSchema.safeParse({
        name: "Production API",
        vercelTeamSlug: "acme",
        vercelProjectName: "storefront",
        vercelEnvironment: "production",
        permissions: ["unkey:v1:ws_test:projects/*/keyspaces/*/keys/*#verify"],
      }).success,
    ).toBe(true);
  });

  it("normalizes the exact Vercel subject fields used for authentication", () => {
    expect(
      oidcVercelInputSchema.parse({
        name: " Production API ",
        vercelTeamSlug: " acme ",
        vercelProjectName: " storefront ",
        vercelEnvironment: " production ",
        permissions: ["api.*.read_api", "api.api_0OIl0OIl.create_key"],
      }),
    ).toEqual({
      name: "Production API",
      vercelTeamSlug: "acme",
      vercelProjectName: "storefront",
      vercelEnvironment: "production",
      permissions: ["api.*.read_api", "api.api_0OIl0OIl.create_key"],
    });
  });

  it("rejects duplicate permission assignments", () => {
    const result = oidcVercelInputSchema.safeParse({
      name: "Production API",
      vercelTeamSlug: "acme",
      vercelProjectName: "storefront",
      vercelEnvironment: "production",
      permissions: ["api.*.read_api", "api.*.read_api"],
    });

    expect(result.success).toBe(false);
  });

  it("rejects values outside the root-key permission grammar", () => {
    const result = oidcVercelInputSchema.safeParse({
      name: "Production API",
      vercelTeamSlug: "acme",
      vercelProjectName: "storefront",
      vercelEnvironment: "production",
      permissions: ["perm_read"],
    });

    expect(result.success).toBe(false);
  });
});

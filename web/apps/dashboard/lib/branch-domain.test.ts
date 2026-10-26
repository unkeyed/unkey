import { describe, expect, it } from "vitest";
import { branchDomainPattern } from "./branch-domain";

describe("branchDomainPattern", () => {
  it("swaps the environment slug for the branch placeholder and keeps the code", () => {
    expect(branchDomainPattern("shop-preview-acme-a1b2c3.unkey.app", "preview", "acme")).toBe(
      "shop-git-<branch>-acme-a1b2c3.unkey.app",
    );
  });

  it("still parses a route created before the code existed", () => {
    expect(branchDomainPattern("shop-preview-acme.unkey.app", "preview", "acme")).toBe(
      "shop-git-<branch>-acme.unkey.app",
    );
  });

  it("parses a route with no code whose workspace slug ends in six hex characters", () => {
    expect(branchDomainPattern("shop-preview-deadbe.unkey.app", "preview", "deadbe")).toBe(
      "shop-git-<branch>-deadbe.unkey.app",
    );
  });

  it("returns null when the label does not end with the environment and workspace", () => {
    expect(branchDomainPattern("shop-a1b2c3d4-acme.unkey.app", "preview", "acme")).toBeNull();
    expect(branchDomainPattern("preview-acme-a1b2c3.unkey.app", "preview", "acme")).toBeNull();
    expect(branchDomainPattern("localhost", "preview", "acme")).toBeNull();
  });
});

import { describe, expect, it } from "vitest";
import { branchDomainPattern } from "./model";

describe("branchDomainPattern", () => {
  it("swaps the environment slug for the branch placeholder", () => {
    expect(branchDomainPattern("shop-preview-acme.unkey.app", "preview", "acme")).toBe(
      "shop-git-<branch>-acme.unkey.app",
    );
  });

  it("only replaces the slug next to the workspace", () => {
    expect(branchDomainPattern("my-preview-shop-preview-acme.unkey.app", "preview", "acme")).toBe(
      "my-preview-shop-git-<branch>-acme.unkey.app",
    );
  });

  it("returns null for hashed or unexpected hostnames", () => {
    expect(branchDomainPattern("shop-a1b2c3d4-acme.unkey.app", "preview", "acme")).toBeNull();
    expect(branchDomainPattern("preview-acme.unkey.app", "preview", "acme")).toBeNull();
    expect(branchDomainPattern("localhost", "preview", "acme")).toBeNull();
  });
});

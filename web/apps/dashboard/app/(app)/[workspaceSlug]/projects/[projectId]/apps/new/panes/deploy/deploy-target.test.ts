import { describe, expect, it } from "vitest";
import { firstDeployEnvironment } from "./deploy-target";

const environments = [
  { slug: "production", kind: "production" as const },
  { slug: "preview", kind: "preview" as const },
];

describe("firstDeployEnvironment", () => {
  it("deploys a repository to production", () => {
    expect(firstDeployEnvironment("git", environments)).toBe("production");
  });

  it("deploys an image to preview", () => {
    expect(firstDeployEnvironment("oci", environments)).toBe("preview");
  });

  it("falls back to the first environment for an image without preview", () => {
    expect(firstDeployEnvironment("oci", [environments[0]])).toBe("production");
  });

  it("has no target for a repository without production", () => {
    expect(firstDeployEnvironment("git", [environments[1]])).toBeNull();
  });
});

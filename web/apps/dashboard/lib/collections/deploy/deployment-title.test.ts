import { shortenId } from "@/lib/shorten-id";
import { describe, expect, test } from "vitest";
import { deploymentTitle } from "./deployment-title";

const base = {
  id: "d_1234567890abcdef",
  source: "unknown" as const,
  gitCommitMessage: null,
  requestedImage: null,
  resolvedImage: null,
};

describe("deploymentTitle", () => {
  test("prefers the commit message", () => {
    expect(
      deploymentTitle({ ...base, source: "git", gitCommitMessage: "fix(api): guard ids" }),
    ).toBe("fix(api): guard ids");
  });

  test("falls back to the image for container deployments", () => {
    expect(
      deploymentTitle({ ...base, source: "oci", requestedImage: "docker.io/library/nginx:1.27" }),
    ).toContain("nginx:1.27");
  });

  test("falls back to the resolved image when none was requested", () => {
    expect(
      deploymentTitle({ ...base, source: "oci", resolvedImage: "docker.io/library/nginx:1.26" }),
    ).toContain("nginx:1.26");
  });

  test("falls back to the short deployment id", () => {
    expect(deploymentTitle(base)).toBe(shortenId(base.id));
  });
});

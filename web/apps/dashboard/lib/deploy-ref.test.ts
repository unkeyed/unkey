import { describe, expect, it } from "vitest";
import { parseDeployRef, UnsupportedDeployRefError } from "./deploy-ref";

describe("parseDeployRef", () => {
  it("returns a plain branch", () => {
    expect(parseDeployRef("main")).toEqual({ branch: "main" });
  });

  it("returns a branch containing slashes", () => {
    expect(parseDeployRef("feat/oz/deploy")).toEqual({ branch: "feat/oz/deploy" });
  });

  it("trims whitespace", () => {
    expect(parseDeployRef("  main  ")).toEqual({ branch: "main" });
  });

  it("recognizes a raw 40-char hex string as a commit", () => {
    const sha = "9f2c1a7b3d4e5f60718293a4b5c6d7e8f9012345";
    expect(parseDeployRef(sha)).toEqual({ commitSha: sha });
  });

  it("treats a shorter hex string as a branch, since only full SHAs are unambiguous", () => {
    expect(parseDeployRef("9f2c1a7")).toEqual({ branch: "9f2c1a7" });
  });

  it("takes the commit and repository out of a commit URL", () => {
    const sha = "9f2c1a7b3d4e5f60718293a4b5c6d7e8f9012345";
    expect(parseDeployRef(`https://github.com/acme/api/commit/${sha}`)).toEqual({
      commitSha: sha,
      repository: "acme/api",
    });
  });

  it("passes a bare repo URL through as a branch, matching the previous router", () => {
    expect(parseDeployRef("https://github.com/acme/api")).toEqual({
      branch: "https://github.com/acme/api",
    });
  });

  // The request carries no pull request field, and resolving one to its head
  // commit needs a GitHub call the dashboard cannot make.
  it("rejects a pull request URL", () => {
    expect(() => parseDeployRef("https://github.com/acme/api/pull/42")).toThrow(
      UnsupportedDeployRefError,
    );
  });

  it("rejects a pull request URL with a trailing slash", () => {
    expect(() => parseDeployRef("https://github.com/acme/api/pull/42/")).toThrow(
      UnsupportedDeployRefError,
    );
  });

  // repository is only accepted alongside commitSha, so a branch on another
  // repository has no representation in the request.
  it("rejects a tree URL", () => {
    expect(() => parseDeployRef("https://github.com/acme/api/tree/main")).toThrow(
      UnsupportedDeployRefError,
    );
  });

  it("rejects a fork reference written as owner:branch", () => {
    expect(() => parseDeployRef("contributor:fix/typo")).toThrow(UnsupportedDeployRefError);
  });

  it("keeps a branch containing a colon after a slash, which is not a fork reference", () => {
    expect(parseDeployRef("feat/a:b")).toEqual({ branch: "feat/a:b" });
  });
});

import { describe, expect, it } from "vitest";
import { parseDeployRef, resolveSourceRepo, validateSourceRepo } from "./resolve-deploy-ref";

const repoConn = { installationId: 1, repositoryFullName: "ogzhanolguncu/demo_api" };

describe("parseDeployRef", () => {
  it("parses a PR URL with sourceRepo", () => {
    expect(parseDeployRef("https://github.com/owner/repo/pull/123")).toEqual({
      kind: "pr",
      prNumber: 123,
      sourceRepo: "owner/repo",
    });
  });

  it("parses a PR URL with trailing slash", () => {
    expect(parseDeployRef("https://github.com/owner/repo/pull/456/")).toEqual({
      kind: "pr",
      prNumber: 456,
      sourceRepo: "owner/repo",
    });
  });

  it("parses a tree URL as branch with sourceRepo", () => {
    expect(parseDeployRef("https://github.com/owner/repo/tree/main")).toEqual({
      kind: "branch",
      branch: "main",
      sourceRepo: "owner/repo",
    });
  });

  it("parses a tree URL with slashes in branch name", () => {
    expect(parseDeployRef("https://github.com/owner/repo/tree/feat/something")).toEqual({
      kind: "branch",
      branch: "feat/something",
      sourceRepo: "owner/repo",
    });
  });

  it("parses a commit URL as sha with sourceRepo", () => {
    const sha = "a".repeat(40);
    expect(parseDeployRef(`https://github.com/fork-owner/repo/commit/${sha}`)).toEqual({
      kind: "sha",
      sha,
      sourceRepo: "fork-owner/repo",
    });
  });

  it("parses fork reference (owner:branch)", () => {
    expect(parseDeployRef("contributor:feature-branch")).toEqual({
      kind: "branch",
      branch: "feature-branch",
      sourceRepo: "contributor",
    });
  });

  it("parses fork reference with slashes in branch", () => {
    expect(parseDeployRef("alice-dev:feat/redesign-navbar")).toEqual({
      kind: "branch",
      branch: "feat/redesign-navbar",
      sourceRepo: "alice-dev",
    });
  });

  it("parses a raw 40-char hex string as sha", () => {
    const sha = "abcdef1234567890abcdef1234567890abcdef12";
    expect(parseDeployRef(sha)).toEqual({
      kind: "sha",
      sha,
    });
  });

  it("returns a plain branch", () => {
    expect(parseDeployRef("main")).toEqual({
      kind: "branch",
      branch: "main",
    });
  });

  it("passes through a bare repo URL as branch", () => {
    expect(parseDeployRef("https://github.com/owner/repo")).toEqual({
      kind: "branch",
      branch: "https://github.com/owner/repo",
    });
  });

  it("trims whitespace", () => {
    expect(parseDeployRef("  main  ")).toEqual({
      kind: "branch",
      branch: "main",
    });
  });
});

describe("resolveSourceRepo", () => {
  it("returns fork repo when repo name matches", () => {
    expect(resolveSourceRepo("fork-owner/demo_api", "ogzhanolguncu/demo_api")).toBe(
      "fork-owner/demo_api",
    );
  });

  it("returns undefined when repo names differ", () => {
    expect(resolveSourceRepo("ogzhanolguncu/http-echo", "ogzhanolguncu/demo_api")).toBeUndefined();
  });

  it("returns undefined when source is the same as base", () => {
    expect(resolveSourceRepo("ogzhanolguncu/demo_api", "ogzhanolguncu/demo_api")).toBeUndefined();
  });

  it("constructs full repo from owner-only sourceRepo", () => {
    expect(resolveSourceRepo("fork-owner", "ogzhanolguncu/demo_api")).toBe("fork-owner/demo_api");
  });

  it("rejects names with characters that could smuggle a fragment or query into the git URL", () => {
    // Bad char lives in the owner segment so it survives the repo-name match
    // and is caught by the charset guard rather than the name comparison.
    expect(resolveSourceRepo("evil#frag/demo_api", "ogzhanolguncu/demo_api")).toBeUndefined();
    expect(resolveSourceRepo("evil?q=1/demo_api", "ogzhanolguncu/demo_api")).toBeUndefined();
    expect(resolveSourceRepo("evil owner/demo_api", "ogzhanolguncu/demo_api")).toBeUndefined();
  });
});

describe("validateSourceRepo", () => {
  it("returns the resolved fork for a valid fork ref", () => {
    expect(validateSourceRepo("fork-owner", repoConn)).toBe("fork-owner/demo_api");
    expect(validateSourceRepo("fork-owner/demo_api", repoConn)).toBe("fork-owner/demo_api");
  });

  it("returns empty string when the ref points at the base repo", () => {
    expect(validateSourceRepo("ogzhanolguncu", repoConn)).toBe("");
    expect(validateSourceRepo("ogzhanolguncu/demo_api", repoConn)).toBe("");
  });

  it("throws for a slashed ref that is not a fork of the base", () => {
    expect(() => validateSourceRepo("ogzhanolguncu/http-echo", repoConn)).toThrow();
  });

  it("throws for a malformed owner-only ref instead of silently dropping it", () => {
    // Regression: an owner-only ref that fails the charset guard must not be
    // dropped and reinterpreted as a trusted build of the base branch.
    expect(() => validateSourceRepo("evil owner", repoConn)).toThrow();
    expect(() => validateSourceRepo("evil#frag", repoConn)).toThrow();
  });
});

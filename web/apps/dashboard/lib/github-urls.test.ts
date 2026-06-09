import { describe, expect, it } from "vitest";
import {
  githubBranchUrl,
  githubCommitUrl,
  githubDeploymentUrl,
  githubPullUrl,
  githubRepoUrl,
} from "./github-urls";

describe("single-part builders", () => {
  it("builds urls when all parts are present", () => {
    expect(githubRepoUrl("acme/api")).toBe("https://github.com/acme/api");
    expect(githubCommitUrl("acme/api", "abc123")).toBe("https://github.com/acme/api/commit/abc123");
    expect(githubBranchUrl("acme/api", "main")).toBe("https://github.com/acme/api/tree/main");
    expect(githubPullUrl("acme/api", 42)).toBe("https://github.com/acme/api/pull/42");
  });

  it("returns undefined when any required part is missing", () => {
    expect(githubRepoUrl(null)).toBeUndefined();
    expect(githubRepoUrl(undefined)).toBeUndefined();

    expect(githubCommitUrl(null, "abc123")).toBeUndefined();
    expect(githubCommitUrl("acme/api", null)).toBeUndefined();

    expect(githubBranchUrl("acme/api", undefined)).toBeUndefined();
    expect(githubBranchUrl(null, "main")).toBeUndefined();

    expect(githubPullUrl("acme/api", null)).toBeUndefined();
    expect(githubPullUrl(null, 42)).toBeUndefined();
  });

  it("treats PR number 0 as present, not missing", () => {
    // prNumber is gated on `!= null`, so a falsy-but-valid 0 must still build.
    expect(githubPullUrl("acme/api", 0)).toBe("https://github.com/acme/api/pull/0");
  });
});

describe("githubDeploymentUrl", () => {
  it("prefers the PR link over the commit link", () => {
    expect(
      githubDeploymentUrl({
        repoFullName: "acme/api",
        forkRepoFullName: null,
        prNumber: 42,
        sha: "abc123",
      }),
    ).toBe("https://github.com/acme/api/pull/42");
  });

  it("points the PR at the base repo even for a fork deployment", () => {
    // The PR lives on the repo it targets, never on the fork.
    expect(
      githubDeploymentUrl({
        repoFullName: "acme/api",
        forkRepoFullName: "contributor/api",
        prNumber: 42,
        sha: "abc123",
      }),
    ).toBe("https://github.com/acme/api/pull/42");
  });

  it("falls back to the commit on the fork repo when there is no PR", () => {
    // A fork push without a PR: the commit only exists on the fork.
    expect(
      githubDeploymentUrl({
        repoFullName: "acme/api",
        forkRepoFullName: "contributor/api",
        prNumber: null,
        sha: "abc123",
      }),
    ).toBe("https://github.com/contributor/api/commit/abc123");
  });

  it("falls back to the commit on the base repo when there is no fork", () => {
    expect(
      githubDeploymentUrl({
        repoFullName: "acme/api",
        forkRepoFullName: null,
        prNumber: null,
        sha: "abc123",
      }),
    ).toBe("https://github.com/acme/api/commit/abc123");
  });

  it("returns undefined when neither a PR nor a commit can be built", () => {
    expect(
      githubDeploymentUrl({
        repoFullName: null,
        forkRepoFullName: null,
        prNumber: null,
        sha: null,
      }),
    ).toBeUndefined();

    // Image-based deployment: a repo exists but no git metadata.
    expect(
      githubDeploymentUrl({
        repoFullName: "acme/api",
        forkRepoFullName: null,
        prNumber: null,
        sha: null,
      }),
    ).toBeUndefined();
  });
});

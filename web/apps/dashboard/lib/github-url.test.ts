import { describe, expect, it } from "vitest";
import { githubUrl } from "./github-url";

describe("single-part builders", () => {
  it("builds urls when all parts are present", () => {
    expect(githubUrl.repo("acme/api")).toBe("https://github.com/acme/api");
    expect(githubUrl.commit("acme/api", "abc123")).toBe(
      "https://github.com/acme/api/commit/abc123",
    );
    expect(githubUrl.branch("acme/api", "main")).toBe("https://github.com/acme/api/tree/main");
    expect(githubUrl.pull("acme/api", 42)).toBe("https://github.com/acme/api/pull/42");
  });

  it("returns undefined when any required part is missing", () => {
    expect(githubUrl.repo(null)).toBeUndefined();
    expect(githubUrl.repo(undefined)).toBeUndefined();

    expect(githubUrl.commit(null, "abc123")).toBeUndefined();
    expect(githubUrl.commit("acme/api", null)).toBeUndefined();

    expect(githubUrl.branch("acme/api", undefined)).toBeUndefined();
    expect(githubUrl.branch(null, "main")).toBeUndefined();

    expect(githubUrl.pull("acme/api", null)).toBeUndefined();
    expect(githubUrl.pull(null, 42)).toBeUndefined();
  });

  it("treats PR number 0 as present, not missing", () => {
    // prNumber is gated on `!= null`, so a falsy-but-valid 0 must still build.
    expect(githubUrl.pull("acme/api", 0)).toBe("https://github.com/acme/api/pull/0");
  });
});

describe("githubUrl.deployment", () => {
  it("prefers the PR link over the commit link", () => {
    expect(
      githubUrl.deployment({
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
      githubUrl.deployment({
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
      githubUrl.deployment({
        repoFullName: "acme/api",
        forkRepoFullName: "contributor/api",
        prNumber: null,
        sha: "abc123",
      }),
    ).toBe("https://github.com/contributor/api/commit/abc123");
  });

  it("falls back to the commit on the base repo when there is no fork", () => {
    expect(
      githubUrl.deployment({
        repoFullName: "acme/api",
        forkRepoFullName: null,
        prNumber: null,
        sha: "abc123",
      }),
    ).toBe("https://github.com/acme/api/commit/abc123");
  });

  it("returns undefined when neither a PR nor a commit can be built", () => {
    expect(
      githubUrl.deployment({
        repoFullName: null,
        forkRepoFullName: null,
        prNumber: null,
        sha: null,
      }),
    ).toBeUndefined();

    // Image-based deployment: a repo exists but no git metadata.
    expect(
      githubUrl.deployment({
        repoFullName: "acme/api",
        forkRepoFullName: null,
        prNumber: null,
        sha: null,
      }),
    ).toBeUndefined();
  });
});

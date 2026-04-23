import { describe, expect, it } from "vitest";
import { parseGitRef } from "./parse-git-ref";

describe("parseGitRef", () => {
  it("parses a PR URL", () => {
    expect(parseGitRef("https://github.com/owner/repo/pull/123")).toEqual({
      kind: "pr",
      value: 123,
    });
  });

  it("parses a PR URL with trailing slash", () => {
    expect(parseGitRef("https://github.com/owner/repo/pull/456/")).toEqual({
      kind: "pr",
      value: 456,
    });
  });

  it("parses a tree URL", () => {
    expect(parseGitRef("https://github.com/owner/repo/tree/main")).toEqual({
      kind: "ref",
      value: "main",
    });
  });

  it("parses a tree URL with slashes in branch name", () => {
    expect(parseGitRef("https://github.com/owner/repo/tree/feat/something")).toEqual({
      kind: "ref",
      value: "feat/something",
    });
  });

  it("parses a commit URL with a 40-char hex SHA", () => {
    const sha = "a".repeat(40);
    expect(parseGitRef(`https://github.com/owner/repo/commit/${sha}`)).toEqual({
      kind: "ref",
      value: sha,
    });
  });

  it("returns a plain branch as a ref", () => {
    expect(parseGitRef("main")).toEqual({
      kind: "ref",
      value: "main",
    });
  });

  it("passes through a fork reference as-is", () => {
    expect(parseGitRef("contributor:feature-branch")).toEqual({
      kind: "ref",
      value: "contributor:feature-branch",
    });
  });

  it("passes through a bare repo URL as-is", () => {
    expect(parseGitRef("https://github.com/owner/repo")).toEqual({
      kind: "ref",
      value: "https://github.com/owner/repo",
    });
  });

  it("trims whitespace", () => {
    expect(parseGitRef("  main  ")).toEqual({
      kind: "ref",
      value: "main",
    });
  });
});

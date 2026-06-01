import { describe, expect, it } from "vitest";
import { parseForkRef } from "./parse-fork-ref";

describe("parseForkRef", () => {
  it("parses a fork reference", () => {
    expect(parseForkRef("contributor:feature-branch")).toEqual({
      forkOwner: "contributor",
      branch: "feature-branch",
    });
  });

  it("parses a fork reference with slash in branch", () => {
    expect(parseForkRef("alice-dev:feat/redesign-navbar")).toEqual({
      forkOwner: "alice-dev",
      branch: "feat/redesign-navbar",
    });
  });

  it("rejects an https URL", () => {
    expect(parseForkRef("https://github.com/owner/repo")).toBeNull();
  });

  it("rejects an http URL with colon-port", () => {
    expect(parseForkRef("http://example.com:8080")).toBeNull();
  });

  it("rejects a plain branch name", () => {
    expect(parseForkRef("main")).toBeNull();
  });

  it("rejects a branch with slash but no colon", () => {
    expect(parseForkRef("feature/something")).toBeNull();
  });

  it("rejects an empty string", () => {
    expect(parseForkRef("")).toBeNull();
  });

  it("rejects an empty owner", () => {
    expect(parseForkRef(":branch")).toBeNull();
  });

  it("rejects an empty branch", () => {
    expect(parseForkRef("owner:")).toBeNull();
  });

  it("rejects a slash before colon", () => {
    expect(parseForkRef("org/owner:branch")).toBeNull();
  });
});

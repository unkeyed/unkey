import { describe, expect, it } from "vitest";
import { sourceAction } from "./source-choice-state";

describe("sourceAction", () => {
  it("lets a new app pick either source", () => {
    expect(sourceAction("git", { lockedTo: null, needsGithub: false })).toBe("pick");
    expect(sourceAction("oci", { lockedTo: null, needsGithub: false })).toBe("pick");
  });

  it("asks to connect GitHub before importing a repository", () => {
    expect(sourceAction("git", { lockedTo: null, needsGithub: true })).toBe("connect-github");
    expect(sourceAction("oci", { lockedTo: null, needsGithub: true })).toBe("pick");
  });

  it("locks the other source once the app has one", () => {
    expect(sourceAction("git", { lockedTo: "git", needsGithub: true })).toBe("connect-github");
    expect(sourceAction("oci", { lockedTo: "git", needsGithub: true })).toBe("locked");
    expect(sourceAction("git", { lockedTo: "oci", needsGithub: false })).toBe("locked");
  });
});

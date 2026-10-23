import { describe, expect, it } from "vitest";
import { sourceAction } from "./source-choice-state";

describe("sourceAction", () => {
  it("lets a new app pick either source", () => {
    expect(sourceAction("git", false)).toBe("pick");
    expect(sourceAction("oci", false)).toBe("pick");
  });

  it("asks to connect GitHub before importing a repository", () => {
    expect(sourceAction("git", true)).toBe("connect-github");
    expect(sourceAction("oci", true)).toBe("pick");
  });
});

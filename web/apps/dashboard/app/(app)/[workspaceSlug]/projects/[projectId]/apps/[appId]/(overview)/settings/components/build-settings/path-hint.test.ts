import { describe, expect, it } from "vitest";
import { pathHint, pathInputVariant } from "./path-hint";

const noMatch = () => null;

describe("pathHint", () => {
  it("shows nothing for a valid path", () => {
    expect(pathHint("valid", () => "Dockerfile", "main")).toEqual({ type: "none" });
  });

  it("shows nothing while the tree is unknown", () => {
    expect(pathHint("unknown", noMatch, "main")).toEqual({ type: "none" });
  });

  it("suggests the case-insensitive match first", () => {
    expect(pathHint("invalid", () => "Dockerfile", "main")).toEqual({
      type: "case-match",
      path: "Dockerfile",
    });
  });

  it("reports not found with the branch", () => {
    expect(pathHint("invalid", noMatch, "main")).toEqual({ type: "not-found", branch: "main" });
  });

  it("reports not found without a branch", () => {
    expect(pathHint("invalid", noMatch, null)).toEqual({ type: "not-found", branch: null });
  });
});

describe("pathInputVariant", () => {
  it("prefers the schema error", () => {
    expect(pathInputVariant(true, { type: "not-found", branch: null })).toBe("error");
  });

  it("warns on any hint", () => {
    expect(pathInputVariant(false, { type: "case-match", path: "api" })).toBe("warning");
    expect(pathInputVariant(false, { type: "not-found", branch: "main" })).toBe("warning");
  });

  it("is default with no hint", () => {
    expect(pathInputVariant(false, { type: "none" })).toBe("default");
  });
});

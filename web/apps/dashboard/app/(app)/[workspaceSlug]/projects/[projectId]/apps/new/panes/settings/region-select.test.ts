import { describe, expect, it } from "vitest";
import { toggleRegion } from "./region-select";

describe("toggleRegion", () => {
  it("adds and removes a region", () => {
    expect(toggleRegion(["us-east-1"], "eu-central-1")).toEqual(["us-east-1", "eu-central-1"]);
    expect(toggleRegion(["us-east-1", "eu-central-1"], "us-east-1")).toEqual(["eu-central-1"]);
  });

  it("keeps the last region", () => {
    expect(toggleRegion(["us-east-1"], "us-east-1")).toEqual(["us-east-1"]);
  });
});

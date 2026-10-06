import { describe, expect, it } from "vitest";
import { mapRegionToFlag } from "./utils";

describe("mapRegionToFlag", () => {
  it.each([
    ["us-east-1", "us"],
    ["eu-central-1", "de"],
    ["ap-southeast-1", "sg"],
    ["ap-southeast-2", "au"],
    ["ap-northeast-1", "jp"],
    ["sa-east-1", "br"],
    ["mars-1", "us"],
  ])("maps %s to %s", (region, flag) => {
    expect(mapRegionToFlag(region)).toBe(flag);
  });
});

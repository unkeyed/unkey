import { describe, expect, test } from "vitest";
import { regionInfo } from "./regions";

describe("regionInfo", () => {
  test("ap-southeast-1 is Singapore and ap-southeast-2 is Sydney", () => {
    expect(regionInfo("ap-southeast-1")).toMatchObject({ city: "Singapore", flag: "sg" });
    expect(regionInfo("ap-southeast-2")).toMatchObject({ city: "Sydney", flag: "au" });
  });

  test("unknown regions fall back to the flag of their prefix", () => {
    expect(regionInfo("us-east-2")).toEqual({
      name: "us-east-2",
      city: "us-east-2",
      flag: "us",
      pin: null,
    });
    expect(regionInfo("ap-south-2").flag).toBe("in");
  });

  test("does not read ap-southeast as ap-south", () => {
    expect(regionInfo("ap-southeast-3").flag).toBeNull();
  });

  test("regions with no known prefix have no flag", () => {
    expect(regionInfo("me-central-1")).toMatchObject({ city: "me-central-1", flag: null });
  });
});

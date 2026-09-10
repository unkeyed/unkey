import { describe, expect, it } from "vitest";
import { expectedBandForPoint } from "./anomaly-chart";

describe("expectedBandForPoint", () => {
  it("renders a request spike band before request-drop eligibility", () => {
    expect(
      expectedBandForPoint("requests", {
        expectedMean: 100,
        upperBound: 180,
        lowerBound: null,
      }),
    ).toEqual([100, 180]);
  });

  it("uses the request-drop bound when both bounds are eligible", () => {
    expect(
      expectedBandForPoint("requests", {
        expectedMean: 100,
        upperBound: 180,
        lowerBound: 25,
      }),
    ).toEqual([25, 180]);
  });

  it("does not invent a band without a valid upper bound", () => {
    expect(
      expectedBandForPoint("requests", {
        expectedMean: 100,
        upperBound: null,
        lowerBound: 25,
      }),
    ).toBeNull();
  });
});

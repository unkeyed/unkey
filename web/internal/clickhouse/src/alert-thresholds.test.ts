import { describe, expect, it } from "vitest";
import { calculateAlertExpectedBand } from "./alert-thresholds";

describe("calculateAlertExpectedBand", () => {
  it("pads a flat request baseline by ten percent of its mean", () => {
    expect(calculateAlertExpectedBand("requests", 1_000, 0, 288)).toEqual({
      lowerBound: 600,
      upperBound: 1_400,
    });
  });

  it("omits the band until twelve lifetime buckets exist", () => {
    expect(calculateAlertExpectedBand("requests", 1_000, 0, 11)).toBeNull();
  });
});

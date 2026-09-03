import { describe, expect, it } from "vitest";
import { calculateAlertExpectedBand } from "./alert-thresholds";

describe("calculateAlertExpectedBand", () => {
  it("uses the recent median for traffic drops and sigma padding for traffic spikes", () => {
    expect(calculateAlertExpectedBand("requests", 1_000, 0, 288)).toEqual({
      lowerBound: 250,
      upperBound: 1_400,
    });
  });

  it("uses a one-percentage-point floor for error ratios", () => {
    expect(calculateAlertExpectedBand("error_5xx", 0.004, 0, 288)).toEqual({
      lowerBound: null,
      upperBound: 0.044,
    });
  });

  it("omits the band until twelve lifetime buckets exist", () => {
    expect(calculateAlertExpectedBand("requests", 1_000, 0, 11)).toBeNull();
  });
});

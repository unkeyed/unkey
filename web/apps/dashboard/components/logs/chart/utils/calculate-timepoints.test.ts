import { describe, expect, it } from "vitest";
import { calculateTimePoints } from "./calculate-timepoints";

describe("calculateTimePoints", () => {
  it("spreads six ticks from start to end", () => {
    expect(calculateTimePoints(0, 50).map((d) => d.getTime())).toEqual([0, 10, 20, 30, 40, 50]);
  });

  it("returns unique ticks for any range, one when the range is too short to split", () => {
    for (const range of [0, 1, 4, 5, 59_999, 3_600_000]) {
      const ticks = calculateTimePoints(1_000, 1_000 + range).map((d) => d.getTime());
      expect(new Set(ticks).size).toBe(ticks.length);
      expect(ticks[0]).toBe(1_000);
    }
    expect(calculateTimePoints(1_000, 1_000)).toHaveLength(1);
  });
});

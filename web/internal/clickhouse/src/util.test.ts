import { describe, expect, it } from "vitest";
import { normalizeTimeRange } from "./util";

describe("normalizeTimeRange", () => {
  it("returns same range when start <= end", () => {
    expect(normalizeTimeRange(1, 2)).toEqual({ startTime: 1, endTime: 2 });
  });

  it("swaps when start > end", () => {
    expect(normalizeTimeRange(10, 3)).toEqual({ startTime: 3, endTime: 10 });
  });

  it("keeps equal bounds", () => {
    expect(normalizeTimeRange(5, 5)).toEqual({ startTime: 5, endTime: 5 });
  });
});

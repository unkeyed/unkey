import { describe, expect, it } from "vitest";
import { assertOrderedTimeRange } from "./util";

describe("assertOrderedTimeRange", () => {
  it("accepts an ordered range", () => {
    expect(() => assertOrderedTimeRange(1_000, 2_000)).not.toThrow();
  });

  it("accepts equal bounds", () => {
    expect(() => assertOrderedTimeRange(1_000, 1_000)).not.toThrow();
  });

  it("rejects a reversed range and names both bounds", () => {
    expect(() => assertOrderedTimeRange(2_000, 1_000)).toThrow(
      "startTime (2000) is after endTime (1000)",
    );
  });
});

import { describe, expect, it } from "vitest";
import { alertSeriesRange } from "./series-range";

describe("alertSeriesRange", () => {
  it("excludes the current partial five-minute bucket", () => {
    expect(
      alertSeriesRange({
        startMs: Date.UTC(2026, 8, 3, 6, 37),
        endMs: Date.UTC(2026, 8, 3, 12, 7),
        resolution: "5m",
        now: Date.UTC(2026, 8, 3, 12, 7),
      }),
    ).toEqual({
      startMs: Date.UTC(2026, 8, 3, 6, 35),
      endMs: Date.UTC(2026, 8, 3, 12, 5),
    });
  });

  it("uses only closed hours for the overview", () => {
    expect(
      alertSeriesRange({
        startMs: Date.UTC(2026, 8, 1, 6, 37),
        endMs: Date.UTC(2026, 8, 3, 12, 7),
        resolution: "1h",
        now: Date.UTC(2026, 8, 3, 12, 7),
      }),
    ).toEqual({
      startMs: Date.UTC(2026, 8, 1, 6),
      endMs: Date.UTC(2026, 8, 3, 12),
    });
  });
});

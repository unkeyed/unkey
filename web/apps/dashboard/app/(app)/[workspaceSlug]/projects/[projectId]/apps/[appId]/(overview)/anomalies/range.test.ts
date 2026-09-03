import { describe, expect, it } from "vitest";
import { dayMs, focusedAlertRange, lastClosedBucketEnd, presetRange } from "./range";

describe("anomaly ranges", () => {
  it("uses the last closed bucket as the preset end", () => {
    const now = Date.UTC(2026, 8, 3, 12, 7);
    expect(lastClosedBucketEnd(now)).toBe(Date.UTC(2026, 8, 3, 12, 5));
    expect(presetRange("24h", now)).toEqual({
      startMs: Date.UTC(2026, 8, 2, 12, 5),
      endMs: Date.UTC(2026, 8, 3, 12, 5),
    });
    expect(presetRange("7d", now).endMs - presetRange("7d", now).startMs).toBe(7 * dayMs);
  });

  it("centers focus mode on the alert and clamps future data", () => {
    const now = Date.UTC(2026, 8, 3, 12, 7);
    expect(focusedAlertRange(Date.UTC(2026, 8, 3, 9), Date.UTC(2026, 8, 3, 9, 5), now)).toEqual({
      startMs: Date.UTC(2026, 8, 3, 6),
      endMs: Date.UTC(2026, 8, 3, 12, 5),
    });
  });
});

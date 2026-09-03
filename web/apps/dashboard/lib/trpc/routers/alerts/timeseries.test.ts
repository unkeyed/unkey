import { describe, expect, it } from "vitest";
import { alertTimeseriesRange } from "./timeseries-range";

const minuteMs = 60 * 1000;
const dayMs = 24 * 60 * minuteMs;

describe("alertTimeseriesRange", () => {
  it("ends at the exclusive boundary of the last closed bucket", () => {
    const now = Date.UTC(2026, 8, 3, 12, 7);
    const firedAt = Date.UTC(2026, 8, 3, 6, 36);

    expect(alertTimeseriesRange({ firedAt, resolvedAt: null, now })).toEqual({
      startMs: firedAt - dayMs + 4 * minuteMs,
      endMs: Date.UTC(2026, 8, 3, 12, 5),
    });
  });

  it("caps resolved alerts one hour after resolution on a closed bucket boundary", () => {
    const now = Date.UTC(2026, 8, 3, 12, 7);
    const firedAt = Date.UTC(2026, 8, 2, 6, 36);
    const resolvedAt = Date.UTC(2026, 8, 3, 10, 3);

    expect(alertTimeseriesRange({ firedAt, resolvedAt, now }).endMs).toBe(
      Date.UTC(2026, 8, 3, 11, 0),
    );
  });
});

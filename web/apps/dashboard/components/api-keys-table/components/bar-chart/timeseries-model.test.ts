import { describe, expect, it } from "vitest";
import { buildTimeseriesByKey, toProcessedTimeseriesDataPoint } from "./timeseries-model";

const HOUR = 60 * 60 * 1000;

const point = (x: number, y: Partial<Record<string, number>>) => ({
  x,
  y: {
    total: 0,
    valid: 0,
    valid_count: 0,
    rate_limited_count: 0,
    insufficient_permissions_count: 0,
    forbidden_count: 0,
    disabled_count: 0,
    expired_count: 0,
    usage_exceeded_count: 0,
    ...y,
  },
});

describe("toProcessedTimeseriesDataPoint", () => {
  it("maps totals and outcome counts", () => {
    const result = toProcessedTimeseriesDataPoint(
      point(0, { total: 10, valid: 7, rate_limited_count: 2, forbidden_count: 1 }),
    );

    expect(result).toEqual({
      valid: 7,
      total: 10,
      success: 7,
      error: 3,
      rate_limited: 2,
      insufficient_permissions: 0,
      forbidden: 1,
      disabled: 0,
      expired: 0,
      usage_exceeded: 0,
    });
  });
});

describe("buildTimeseriesByKey", () => {
  it("fills an hourly point for every bucket and keeps keys separate", () => {
    const byKey = buildTimeseriesByKey(
      {
        key_1: [point(0, { total: 5, valid: 4 }), point(2 * HOUR, { total: 3, valid: 3 })],
      },
      ["key_1", "key_2"],
      0,
      2 * HOUR,
    );

    const key1 = byKey.get("key_1");
    expect(key1).toHaveLength(3);
    expect(key1?.map((p) => p.total)).toEqual([5, 0, 3]);
    expect(key1?.[0].error).toBe(1);

    const key2 = byKey.get("key_2");
    expect(key2).toHaveLength(3);
    expect(key2?.every((p) => p.total === 0 && p.error === 0)).toBe(true);
  });

  it("returns zero points when the key has no rows", () => {
    const byKey = buildTimeseriesByKey(undefined, ["key_1"], 0, HOUR);

    expect(byKey.get("key_1")).toHaveLength(2);
    expect(byKey.get("key_1")?.every((p) => p.total === 0)).toBe(true);
  });
});

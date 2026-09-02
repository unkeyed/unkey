import { describe, expect, it } from "vitest";
import {
  computeMetrics,
  mapVerificationsResponse,
  type RawVerificationDataPoint,
} from "./analytics-transform";

describe("mapVerificationsResponse", () => {
  it("maps raw data points, deriving error as total minus valid", () => {
    const raw: RawVerificationDataPoint[] = [
      { time: 1000, total: 10, valid: 8 },
      { time: 2000, total: 4, valid: 4 },
    ];

    expect(mapVerificationsResponse(raw)).toEqual([
      { time: 1000, total: 10, valid: 8, error: 2 },
      { time: 2000, total: 4, valid: 4, error: 0 },
    ]);
  });

  it("clamps error at zero when valid exceeds total", () => {
    const raw: RawVerificationDataPoint[] = [{ time: 1000, total: 5, valid: 7 }];

    expect(mapVerificationsResponse(raw)[0].error).toBe(0);
  });

  it("returns an empty array for no data points", () => {
    expect(mapVerificationsResponse([])).toEqual([]);
  });
});

describe("computeMetrics", () => {
  it("aggregates totals and derives success/error rates", () => {
    const metrics = computeMetrics([
      { time: 1000, total: 10, valid: 8, error: 2 },
      { time: 2000, total: 10, valid: 6, error: 4 },
    ]);

    expect(metrics.totalRequests).toBe(20);
    expect(metrics.validRequests).toBe(14);
    expect(metrics.errorRequests).toBe(6);
    expect(metrics.successRate).toBeCloseTo(0.7);
    expect(metrics.errorRate).toBeCloseTo(0.3);
  });

  it("reports zero rates (not NaN) when there are no requests", () => {
    const metrics = computeMetrics([
      { time: 1000, total: 0, valid: 0, error: 0 },
      { time: 2000, total: 0, valid: 0, error: 0 },
    ]);

    expect(metrics.totalRequests).toBe(0);
    expect(metrics.successRate).toBe(0);
    expect(metrics.errorRate).toBe(0);
  });

  it("returns zeroed metrics for an empty series", () => {
    expect(computeMetrics([])).toEqual({
      totalRequests: 0,
      validRequests: 0,
      errorRequests: 0,
      successRate: 0,
      errorRate: 0,
    });
  });
});

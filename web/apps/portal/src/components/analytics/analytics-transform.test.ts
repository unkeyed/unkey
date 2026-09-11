import { describe, expect, it } from "vitest";
import {
  type RawVerificationDataPoint,
  computeMetrics,
  emptyOutcomes,
  mapVerificationsResponse,
  sumBuckets,
} from "./analytics-transform";
import type { VerificationBucket } from "./schema/analytics.schema";

function point(
  time: number,
  total: number,
  valid: number,
  outcomes: Partial<RawVerificationDataPoint> = {},
): RawVerificationDataPoint {
  return { time, total, valid, ...emptyOutcomes(), ...outcomes };
}

function bucket(
  time: number,
  total: number,
  valid: number,
  outcomes: Partial<VerificationBucket> = {},
): VerificationBucket {
  return { time, total, valid, error: total - valid, other: 0, ...emptyOutcomes(), ...outcomes };
}

describe("mapVerificationsResponse", () => {
  it("keeps every outcome count and derives error as their sum", () => {
    const raw = [point(1000, 10, 8, { rateLimited: 1, forbidden: 1 }), point(2000, 4, 4)];

    expect(mapVerificationsResponse(raw)).toEqual([
      { ...raw[0], error: 2, other: 0 },
      { ...raw[1], error: 0, other: 0 },
    ]);
  });

  it("puts the outcomes the API does not break out in the other bucket", () => {
    const [mapped] = mapVerificationsResponse([point(1000, 10, 6, { rateLimited: 1 })]);

    expect(mapped.other).toBe(3);
    expect(mapped.error).toBe(4);
    expect(mapped.valid + mapped.error).toBe(mapped.total);
  });

  it("ignores a valid count that exceeds total when deriving error", () => {
    const raw = [point(1000, 5, 7)];

    expect(mapVerificationsResponse(raw)[0].error).toBe(0);
    expect(mapVerificationsResponse(raw)[0].other).toBe(0);
  });

  it("returns an empty array for no data points", () => {
    expect(mapVerificationsResponse([])).toEqual([]);
  });
});

describe("sumBuckets", () => {
  it("adds totals, valid, error and each outcome across the series", () => {
    const totals = sumBuckets([
      bucket(1000, 10, 8, { rateLimited: 2 }),
      bucket(2000, 10, 6, { rateLimited: 1, expired: 3 }),
    ]);

    expect(totals).toEqual({
      ...emptyOutcomes(),
      total: 20,
      valid: 14,
      error: 6,
      other: 0,
      rateLimited: 3,
      expired: 3,
    });
  });
});

describe("computeMetrics", () => {
  it("aggregates totals and derives success/error rates", () => {
    const metrics = computeMetrics([
      bucket(1000, 10, 8, { rateLimited: 2 }),
      bucket(2000, 10, 6, { forbidden: 4 }),
    ]);

    expect(metrics.totalRequests).toBe(20);
    expect(metrics.validRequests).toBe(14);
    expect(metrics.errorRequests).toBe(6);
    expect(metrics.successRate).toBeCloseTo(0.7);
    expect(metrics.errorRate).toBeCloseTo(0.3);
  });

  it("reports zero rates (not NaN) when there are no requests", () => {
    const metrics = computeMetrics([bucket(1000, 0, 0), bucket(2000, 0, 0)]);

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

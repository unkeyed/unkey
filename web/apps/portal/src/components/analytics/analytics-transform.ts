import {
  OUTCOME_KINDS,
  type OutcomeCounts,
  type VerificationBucket,
} from "./schema/analytics.schema";

/**
 * The subset of a `v2/portal.getVerifications` data point this page consumes.
 * Declared structurally (rather than importing the SDK type) so the transform
 * stays dependency-free and directly unit-testable.
 */
export type RawVerificationDataPoint = OutcomeCounts & {
  time: number;
  total: number;
  valid: number;
};

export function emptyOutcomes(): OutcomeCounts {
  return {
    rateLimited: 0,
    expired: 0,
    disabled: 0,
    usageExceeded: 0,
    insufficientPermissions: 0,
    forbidden: 0,
  };
}

export function emptyBucket(time: number): VerificationBucket {
  return { ...emptyOutcomes(), time, total: 0, valid: 0, error: 0, other: 0 };
}

/**
 * Map the raw `portal.getVerifications` timeseries onto the buckets the page
 * renders. `total` counts outcomes the API does not break out, so the
 * remainder lands in `other` and the stacked bars always add up to `total`.
 * Both derived counts clamp at zero, so a malformed point can never yield a
 * negative bar.
 */
export function mapVerificationsResponse(
  data: ReadonlyArray<RawVerificationDataPoint>,
): VerificationBucket[] {
  return data.map((point) => {
    let broken = 0;
    for (const kind of OUTCOME_KINDS) {
      broken += point[kind];
    }
    const other = Math.max(0, point.total - point.valid - broken);
    return { ...point, other, error: broken + other };
  });
}

/** Window totals for a series: the same shape as a bucket, minus its time. */
export type VerificationTotals = Omit<VerificationBucket, "time">;

export function sumBuckets(buckets: ReadonlyArray<VerificationBucket>): VerificationTotals {
  const totals: VerificationTotals = {
    total: 0,
    valid: 0,
    error: 0,
    other: 0,
    ...emptyOutcomes(),
  };
  for (const bucket of buckets) {
    totals.total += bucket.total;
    totals.valid += bucket.valid;
    totals.error += bucket.error;
    totals.other += bucket.other;
    for (const kind of OUTCOME_KINDS) {
      totals[kind] += bucket[kind];
    }
  }
  return totals;
}

/** Aggregate metrics rendered as the page's summary cards. */
export type VerificationMetrics = {
  totalRequests: number;
  validRequests: number;
  errorRequests: number;
  /** Fraction in [0, 1]; `0` when there are no requests. */
  successRate: number;
  /** Fraction in [0, 1]; `0` when there are no requests. */
  errorRate: number;
};

/**
 * Reduce the timeseries to window totals and success/error rates. With no
 * requests both rates are `0` (rather than NaN), which the UI renders as an
 * empty dash.
 */
export function computeMetrics(buckets: ReadonlyArray<VerificationBucket>): VerificationMetrics {
  const { total, valid, error } = sumBuckets(buckets);
  return {
    totalRequests: total,
    validRequests: valid,
    errorRequests: error,
    successRate: total === 0 ? 0 : valid / total,
    errorRate: total === 0 ? 0 : error / total,
  };
}

import type { VerificationBucket } from "./schema/analytics.schema";

/**
 * The subset of a `v2/portal.getVerifications` data point this page consumes.
 * Declared structurally (rather than importing the SDK type) so the transform
 * stays dependency-free and directly unit-testable.
 */
export type RawVerificationDataPoint = {
  time: number;
  total: number;
  valid: number;
};

/**
 * Map the raw `portal.getVerifications` timeseries onto the buckets the page
 * renders. Every non-valid outcome is collapsed into a single `error` count,
 * clamped at zero so a malformed point (valid > total) can never yield a
 * negative bar.
 */
export function mapVerificationsResponse(
  data: ReadonlyArray<RawVerificationDataPoint>,
): VerificationBucket[] {
  return data.map((point) => ({
    time: point.time,
    total: point.total,
    valid: point.valid,
    error: Math.max(point.total - point.valid, 0),
  }));
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
  let totalRequests = 0;
  let validRequests = 0;
  for (const bucket of buckets) {
    totalRequests += bucket.total;
    validRequests += bucket.valid;
  }
  const errorRequests = totalRequests - validRequests;
  const successRate = totalRequests === 0 ? 0 : validRequests / totalRequests;
  const errorRate = totalRequests === 0 ? 0 : errorRequests / totalRequests;
  return { totalRequests, validRequests, errorRequests, successRate, errorRate };
}

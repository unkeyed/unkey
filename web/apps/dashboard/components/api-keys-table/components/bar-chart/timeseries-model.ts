import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";

export type ProcessedTimeseriesDataPoint = {
  valid: number;
  total: number;
  success: number;
  error: number;
  rate_limited?: number;
  insufficient_permissions?: number;
  forbidden?: number;
  disabled?: number;
  expired?: number;
  usage_exceeded?: number;
};

export const EMPTY_TIMESERIES: ProcessedTimeseriesDataPoint[] = [];

const EMPTY_POINT: ProcessedTimeseriesDataPoint = {
  valid: 0,
  total: 0,
  success: 0,
  error: 0,
  rate_limited: 0,
  insufficient_permissions: 0,
  forbidden: 0,
  disabled: 0,
  expired: 0,
  usage_exceeded: 0,
};

const HOUR_MS = 60 * 60 * 1000;

export const toProcessedTimeseriesDataPoint = (
  ts: VerificationTimeseriesDataPoint,
): ProcessedTimeseriesDataPoint => {
  const result: ProcessedTimeseriesDataPoint = {
    valid: ts.y.valid,
    total: ts.y.total,
    success: ts.y.valid,
    error: ts.y.total - ts.y.valid,
  };

  if (ts.y.rate_limited_count !== undefined) {
    result.rate_limited = ts.y.rate_limited_count;
  }
  if (ts.y.insufficient_permissions_count !== undefined) {
    result.insufficient_permissions = ts.y.insufficient_permissions_count;
  }
  if (ts.y.forbidden_count !== undefined) {
    result.forbidden = ts.y.forbidden_count;
  }
  if (ts.y.disabled_count !== undefined) {
    result.disabled = ts.y.disabled_count;
  }
  if (ts.y.expired_count !== undefined) {
    result.expired = ts.y.expired_count;
  }
  if (ts.y.usage_exceeded_count !== undefined) {
    result.usage_exceeded = ts.y.usage_exceeded_count;
  }

  return result;
};

// The API returns one row per key only for the hourly buckets that saw traffic.
// The chart expects a point for every hour in the window, so the gaps become
// zero-count points here.
export const buildTimeseriesByKey = (
  series: Record<string, VerificationTimeseriesDataPoint[]> | undefined,
  keyIds: string[],
  startTime: number,
  endTime: number,
): Map<string, ProcessedTimeseriesDataPoint[]> => {
  const byKeyId = new Map<string, ProcessedTimeseriesDataPoint[]>();

  const firstBucket = Math.floor(startTime / HOUR_MS) * HOUR_MS;
  const lastBucket = Math.floor(endTime / HOUR_MS) * HOUR_MS;

  for (const keyId of keyIds) {
    const byBucket = new Map<number, VerificationTimeseriesDataPoint>();
    for (const point of series?.[keyId] ?? []) {
      byBucket.set(point.x, point);
    }

    const filled: ProcessedTimeseriesDataPoint[] = [];
    for (let bucket = firstBucket; bucket <= lastBucket; bucket += HOUR_MS) {
      const point = byBucket.get(bucket);
      filled.push(point ? toProcessedTimeseriesDataPoint(point) : EMPTY_POINT);
    }

    byKeyId.set(keyId, filled);
  }

  return byKeyId;
};

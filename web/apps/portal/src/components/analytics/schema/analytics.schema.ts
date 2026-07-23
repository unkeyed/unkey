import { z } from "zod";

/**
 * A selectable analytics window, identified by its length in days (`1` renders
 * as "Last 24 hours"). Periods are day-counts rather than a fixed enum because
 * the available set is derived from the workspace's data retention, which is a
 * per-plan number (7, 30, 90, or an arbitrary value) — see
 * {@link availableAnalyticsPeriods}.
 */
export type AnalyticsPeriod = { days: number; label: string };

/** Upper bound on any window, in days. Matches the daily ClickHouse rollup's
 * 365-day TTL — the widest range for which verification data still exists. */
export const MAX_PERIOD_DAYS = 365;

/** Fixed ladder rungs, in ascending days. `1` = 24 hours. */
const PERIOD_LADDER_DAYS = [1, 7, 30, 90] as const;

function labelForDays(days: number): string {
  return days === 1 ? "Last 24 hours" : `Last ${days} days`;
}

/**
 * The periods a workspace can query, given its log-retention quota.
 * `portal.getVerifications` rejects any window wider than `logsRetentionDays`,
 * so we only offer windows within it. The set is the fixed ladder (24h / 7d /
 * 30d / 90d) capped at retention, plus a final rung at the exact retention when
 * it doesn't already land on a ladder value (e.g. a 45-day plan gets
 * 24h/7d/30d/45d). A non-positive `retentionDays` means "uncapped" (the API
 * skips the check), so the full ladder is offered.
 */
export function availableAnalyticsPeriods(retentionDays: number): AnalyticsPeriod[] {
  const uncapped = retentionDays <= 0;
  const cap = uncapped
    ? PERIOD_LADDER_DAYS[PERIOD_LADDER_DAYS.length - 1]
    : Math.min(retentionDays, MAX_PERIOD_DAYS);

  const days: number[] = PERIOD_LADDER_DAYS.filter((rung) => rung <= cap);
  if (!uncapped && cap > 1 && !days.includes(cap)) {
    days.push(cap);
  }
  if (days.length === 0) {
    days.push(1); // Always offer at least the 24h view.
  }

  return days.sort((a, b) => a - b).map((d) => ({ days: d, label: labelForDays(d) }));
}

/**
 * The default period to land on: the 7-day view when retention allows it,
 * otherwise the widest period that fits.
 */
export function defaultAnalyticsPeriodDays(retentionDays: number): number {
  const available = availableAnalyticsPeriods(retentionDays);
  return available.find((p) => p.days === 7)?.days ?? available[available.length - 1].days;
}

/** Query params for `v2/portal.getVerifications`: the window length in days. */
export const getVerificationsQuerySchema = z.object({
  days: z.number().int().positive().max(MAX_PERIOD_DAYS),
});

export type GetVerificationsQuery = z.infer<typeof getVerificationsQuerySchema>;

/**
 * One bucket of the verification timeseries, mapped from a
 * `v2/portal.getVerifications` data point. Kept intentionally small: the page
 * renders totals and a valid/error split, so the per-outcome rejection
 * breakdown from the API is collapsed into a single `error` count.
 */
export type VerificationBucket = {
  /** Bucket start as a unix timestamp in milliseconds. */
  time: number;
  /** Total verifications in this bucket, across all outcomes. */
  total: number;
  /** Verifications with a VALID outcome. */
  valid: number;
  /** Non-valid verifications (rate limited, forbidden, expired, etc.). */
  error: number;
};

/** The zero-filled verification timeseries for a query window. */
export type VerificationsTimeseries = {
  days: number;
  buckets: VerificationBucket[];
};

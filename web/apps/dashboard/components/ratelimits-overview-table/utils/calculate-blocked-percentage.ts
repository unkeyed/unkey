import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";

// Above this share of blocked traffic the identifier is flagged visually
// (warning icon + tinted row) so high-block clients are scannable at a glance.
const BLOCKED_WARNING_THRESHOLD_PERCENT = 60;

// getBlockedPercentage returns the share of blocked requests for an identifier
// as a 0-100 percentage. Returns 0 when there was no traffic in the window so
// callers do not have to guard against division by zero.
export const getBlockedPercentage = (log: RatelimitOverviewLog): number => {
  const totalRequests = log.blocked_count + log.passed_count;
  if (totalRequests === 0) {
    return 0;
  }
  return (log.blocked_count / totalRequests) * 100;
};

// isMostlyBlocked reports whether an identifier's blocked share crosses the
// warning threshold that drives the high-block visual treatment.
export const isMostlyBlocked = (log: RatelimitOverviewLog): boolean => {
  return getBlockedPercentage(log) > BLOCKED_WARNING_THRESHOLD_PERCENT;
};

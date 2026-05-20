import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";

// Above this share of blocked traffic the identifier is flagged visually
// (warning icon + tinted row) so high-block clients are scannable at a glance.
const BLOCKED_WARNING_THRESHOLD = 0.6;

export const isMostlyBlocked = (log: RatelimitOverviewLog): boolean => {
  const totalRequests = log.blocked_count + log.passed_count;
  if (totalRequests === 0) {
    return false;
  }
  return log.blocked_count / totalRequests > BLOCKED_WARNING_THRESHOLD;
};

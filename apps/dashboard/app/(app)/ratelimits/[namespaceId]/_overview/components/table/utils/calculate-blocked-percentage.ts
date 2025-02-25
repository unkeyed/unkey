import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";

export const calculateBlockedPercentage = (log: RatelimitOverviewLog) => {
  const totalRequests = log.blocked_count + log.passed_count;
  const blockRate = totalRequests > 0 ? (log.blocked_count / totalRequests) * 100 : 0;
  const hasMoreBlocked = blockRate > 60;

  return hasMoreBlocked;
};

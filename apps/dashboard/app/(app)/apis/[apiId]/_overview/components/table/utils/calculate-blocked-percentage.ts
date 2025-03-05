import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";

/**
 * Calculate whether a key has a high error percentage
 * Returns true if more than 30% of requests are errors
 */
export const calculateErrorPercentage = (log: KeysOverviewLog): boolean => {
  const totalRequests = log.valid_count + log.error_count;

  // Avoid division by zero
  if (totalRequests === 0) {
    return false;
  }

  return log.error_count / totalRequests > 0.3;
};

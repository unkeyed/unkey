import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";

/**
 * Severity levels for error percentages as string literal union
 */
export type ErrorSeverity = "none" | "low" | "moderate" | "high";

/**
 * Calculate the error percentage for a key
 * @param log The keys overview log
 * @returns The percentage of errors (0-100)
 */
export const getErrorPercentage = (log: KeysOverviewLog): number => {
  const totalRequests = log.valid_count + log.error_count;

  // Avoid division by zero
  if (totalRequests === 0) {
    return 0;
  }

  return (log.error_count / totalRequests) * 100;
};

/**
 * Calculate the success percentage for a key
 * @param log The keys overview log
 * @returns The percentage of success (0-100)
 */
export const getSuccessPercentage = (log: KeysOverviewLog): number => {
  const totalRequests = log.valid_count + log.error_count;

  if (totalRequests === 0) {
    return 0;
  }

  return (log.valid_count / totalRequests) * 100;
};

/**
 * Determine the error severity based on the error percentage
 * @param log The keys overview log
 * @returns The severity level as a string literal
 */
export const getErrorSeverity = (log: KeysOverviewLog): ErrorSeverity => {
  const errorPercentage = getErrorPercentage(log);

  if (errorPercentage >= 50) {
    return "high";
  }
  if (errorPercentage >= 20) {
    return "moderate";
  }
  if (errorPercentage > 0) {
    return "low";
  }
  return "none";
};
